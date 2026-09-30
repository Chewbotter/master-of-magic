package game

// The economy of mana and units as the original game has it, the game's side. Rules in words:
// docs/mod/economy.md; the player's side in player/economy.go.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/INITGAME.c:
// Init_Magic_Personalities_Objectives, Init_Summoning_Circle_And_Spell_Of_Mastery; NEXTTURN.c:
// Players_Apply_Upkeeps__WIP, WIZ_MatchFoodUpkeep__WIP, WIZ_MatchGoldUpkeep,
// WIZ_MatchManaUpkeep__WIP, WIZ_DisbandSummons, All_Players_Apply_Spell_Casting,
// Player_Hero_Casting_Skill, Next_Turn_Calc). The code is ours.

import (
    "fmt"
    "log"
    "math/rand/v2"
    "slices"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    herolib "github.com/kazzmir/master-of-magic/game/magic/hero"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// Init_Magic_Personalities_Objectives: at Hard and above a computer wizard gets a summoning spell
// to research when the guard spell is unknown to it, and loses it from what it knows if it had
// it (kept, the original's)
const quirkHardGrantUnlearns = true

// Next_Turn_Calc: Time Stop takes 200 mana a turn and ends when the reserve goes below 0
const timeStopDrain = 200

// the summoning spell a computer wizard knows from the start, by the realm of most books
var mainRealmSummon = map[data.MagicType]string{
    data.NatureMagic: "Sprites",
    data.SorceryMagic: "Nagas",
    data.ChaosMagic: "Hell Hounds",
    data.LifeMagic: "Guardian Spirit",
    data.DeathMagic: "Ghouls",
}

// at Hard and above: the guard spell, the spell given to research
var hardGrants = map[data.MagicType][2]string{
    data.NatureMagic: {"Basilisk", "Basilisk"},
    data.SorceryMagic: {"Basilisk", "Phantom Beast"},
    data.ChaosMagic: {"Basilisk", "Chimeras"},
    data.LifeMagic: {"Path Finding", "Unicorns"},
    data.DeathMagic: {"Basilisk", "Shadow Demons"},
}

// the realm of most books; a tie goes to the first in the original's order
func mainRealm(wizard interface{ MagicLevel(data.MagicType) int }) data.MagicType {
    best := data.MagicNone
    most := 0
    for _, magic := range []data.MagicType{data.NatureMagic, data.SorceryMagic, data.ChaosMagic, data.LifeMagic, data.DeathMagic} {
        if wizard.MagicLevel(magic) > most {
            most = wizard.MagicLevel(magic)
            best = magic
        }
    }
    return best
}

// the spells a new player gets beyond its starting spells, and the research cost of the Spell of
// Mastery (called in AddPlayer after the spells that can be researched are known)
func (model *GameModel) classicStartingMagic(player *playerlib.Player) {
    mastery := model.allSpells.FindByName("Spell of Mastery")
    if mastery.Valid() {
        player.SpellOfMasteryCost = mastery.ResearchCost
        allEleven := true
        for _, magic := range []data.MagicType{data.NatureMagic, data.SorceryMagic, data.ChaosMagic, data.LifeMagic, data.DeathMagic} {
            if player.Wizard.MagicLevel(magic) != 11 {
                allEleven = false
            }
        }
        if allEleven {
            player.SpellOfMasteryCost = max(0, player.SpellOfMasteryCost - 3000)
        }
    }

    if player.IsHuman() || player.IsNeutral() {
        return
    }

    realm := mainRealm(&player.Wizard)
    if name, ok := mainRealmSummon[realm]; ok {
        spell := model.allSpells.FindByName(name)
        if spell.Valid() {
            player.KnownSpells.AddSpell(spell)
            player.ResearchPoolSpells.AddSpell(spell)
        }
    }

    difficulty := model.Settings.Difficulty
    if difficulty == data.DifficultyHard || difficulty == data.DifficultyExtreme || difficulty == data.DifficultyImpossible {
        grant, ok := hardGrants[realm]
        guard := model.allSpells.FindByName(grant[0])
        given := model.allSpells.FindByName(grant[1])
        if ok && guard.Valid() && given.Valid() && !player.KnownSpells.Contains(guard) && !player.ResearchPoolSpells.Contains(guard) {
            player.ResearchPoolSpells.AddSpell(given)
            if quirkHardGrantUnlearns {
                player.KnownSpells.RemoveSpell(given)
            }
        }
    }
}

// Player_Hero_Casting_Skill: half the spell points of each hero on the fortress's square
func heroFortressSkill(player *playerlib.Player) int {
    fortress := player.FindFortressCity()
    if fortress == nil {
        return 0
    }
    skill := 0
    for _, unit := range player.GetUnits(fortress.X, fortress.Y, fortress.Plane) {
        if unit.IsHero() {
            skill += int(unit.GetCastingSkill()) / 2
        }
    }
    return skill
}

// All_Players_Apply_Spell_Casting: the skill left of the last turn and the heroes at the fortress
// go into the spell; when next turn's skill finishes it, it is borrowed. The skill of the turn is
// what is left of it after the borrowing
func (game *Game) classicCasting(player *playerlib.Player) {
    nominal := player.ComputeCastingSkill()
    borrowed := 0

    if !player.CastingSpell.Invalid() {
        cost := player.ComputeEffectiveSpellCost(player.CastingSpell, true)
        skillLeft := player.RemainingCastingSkill + heroFortressSkill(player)

        spent := max(0, min(skillLeft, player.Mana, cost - player.CastingSpellProgress))
        player.IncreaseCastingSkillProgress(spent)

        remaining := cost - player.CastingSpellProgress
        if remaining > 0 {
            more := max(0, min(nominal, player.Mana, remaining))
            if more == remaining {
                player.IncreaseCastingSkillProgress(more)
                borrowed = more
            }
        }

        if cost <= player.CastingSpellProgress {
            game.doCastSpell(player, player.CastingSpell)
            player.CastingSpell = spellbook.Spell{}
            player.CastingSpellProgress = 0
        }
    }

    player.RemainingCastingSkill = nominal - borrowed
}

// Time Stop: 200 a turn in place of all upkeep; below 0 the reserve is 0 and the spell ends.
// true when it goes on
func classicTimeStop(player *playerlib.Player) bool {
    player.Mana -= timeStopDrain
    if player.Mana < 0 {
        player.Mana = 0
        player.RemoveEnchantment(data.EnchantmentTimeStop)
        return false
    }
    return true
}

// units a player loses to pay: the human only (Players_Apply_Upkeeps__WIP); a computer player's
// reserves just stop at 0. food, gold, mana in that order, each from the newest unit back
func (game *Game) classicUpkeep(player *playerlib.Player, power int) []string {
    if !player.IsHuman() || (DebugNoDisband && player.IsHuman()) {
        return nil
    }

    var messages []string
    lose := func(unit units.StackUnit, why string) {
        log.Printf("Disband %v: %v", unit.GetName(), why)
        messages = append(messages, fmt.Sprintf("%v - %v", unit.GetName(), why))
        if hero, ok := unit.(*herolib.Hero); ok {
            for i := range hero.Equipment {
                hero.Equipment[i] = nil
            }
        }
        player.RemoveUnit(unit)
    }

    newestFirst := func() []units.StackUnit {
        all := slices.Collect(player.Units())
        slices.Reverse(all)
        return all
    }

    // a normal unit: not a creature, not a hero, not a ship
    normal := func(unit units.StackUnit) bool {
        return unit.GetRace() != data.RaceFantastic && !unit.IsHero() && !unit.IsTransport()
    }

    // food: what the cities with a surplus have over their people
    excess := 0
    for _, city := range player.Cities {
        excess += max(0, city.SurplusFood())
    }
    foodUpkeep := player.TotalUnitUpkeepFood()
    if excess < foodUpkeep {
        for _, unit := range newestFirst() {
            if foodUpkeep <= excess {
                break
            }
            if normal(unit) && unit.GetUpkeepFood() > 0 {
                foodUpkeep -= unit.GetUpkeepFood()
                lose(unit, "deserted: lack of food.")
            }
        }
    }

    // gold: the reserve with the income of the turn against the upkeep of the units
    goldUpkeep := player.TotalUnitUpkeepGold()
    goldReserve := player.Gold + player.GoldPerTurn() + goldUpkeep
    if goldUpkeep > goldReserve {
        for _, unit := range newestFirst() {
            if goldReserve >= goldUpkeep {
                break
            }
            if normal(unit) && unit.GetUpkeepGold() > 0 {
                goldUpkeep -= unit.GetUpkeepGold()
                lose(unit, "deserted: lack of gold.")
            }
        }
    }

    // mana: groups in an order by chance until the upkeep is covered
    manaUpkeep := player.ManaUpkeep(game.Model)
    manaReserve := player.Mana + player.ManaPerTurn(power, game.Model) + manaUpkeep
    divisor := 1
    if player.Wizard.RetortEnabled(data.RetortChanneler) {
        divisor = 2
    }
    group := rand.N(4)
    for range 4 {
        if manaReserve >= manaUpkeep {
            break
        }
        switch group {
            case 0:
                // unit enchantments (not reconstructed: MY CALL, the newest unit first)
                for _, unit := range newestFirst() {
                    for _, enchantment := range slices.Clone(unit.GetUpkeepEnchantments()) {
                        if manaReserve >= manaUpkeep {
                            break
                        }
                        manaUpkeep -= enchantment.UpkeepMana() / divisor
                        unit.RemoveEnchantment(enchantment)
                    }
                }
            case 1:
                // city enchantments the player cast (not reconstructed: MY CALL, in the order kept)
                for _, enchantment := range game.Model.GetCityEnchantmentsByBanner(player.GetBanner()) {
                    if manaReserve >= manaUpkeep {
                        break
                    }
                    manaUpkeep -= enchantment.Enchantment.Enchantment.UpkeepMana() / divisor
                    enchantment.City.CancelEnchantment(enchantment.Enchantment.Enchantment, enchantment.Enchantment.Owner)
                }
            case 2:
                // global enchantments (not reconstructed: MY CALL, in the order kept)
                for _, enchantment := range player.GlobalEnchantments.Values() {
                    if manaReserve >= manaUpkeep {
                        break
                    }
                    manaUpkeep -= enchantment.UpkeepMana() / divisor
                    player.RemoveEnchantment(enchantment)
                }
            case 3:
                // WIZ_DisbandSummons: creatures only
                for _, unit := range newestFirst() {
                    if manaReserve >= manaUpkeep {
                        break
                    }
                    if unit.GetRace() == data.RaceFantastic {
                        manaUpkeep -= player.UnitManaUpkeep(unit) / divisor
                        lose(unit, "dispelled: lack of mana.")
                    }
                }
        }
        group = (group + 1) % 4
    }

    return messages
}
