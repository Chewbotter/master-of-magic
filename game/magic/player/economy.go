package player

// The economy of mana and units as the original game has it. Rules in words: docs/mod/economy.md.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/CITYCALC.c:
// Player_Armies_And_Enchantments_Mana_Upkeep, Unit_Mana_Upkeep, Player_Armies_Food_Upkeep,
// Player_Magic_Power_Distribution, Player_Resource_Income_Total; NEXTTURN.c:
// Update_Players_Gold_Reserve, Player_Gets_Spell; Spells128.c: Player_Research_Spells,
// Build_Research_List, Sort_Research_List). The code is ours.

import (
    rand "github.com/kazzmir/master-of-magic/lib/chance"
    "slices"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// false: upstream's economy of mana and units
var ClassicEconomy = true

// Sort_Research_List judges the research bonus of a realm on its first spell, whatever the spell
// on the list is (kept, the original's)
const quirkResearchBonusOfFirstSpell = true

const researchCandidates = 8

// Unit_Mana_Upkeep: a creature's upkeep (half again for undead), with Conjurer three quarters
// rounded up, then its enchantments; heroes and normal units have none of their own
func (player *Player) unitManaUpkeep(unit units.StackUnit) int {
    base := 0
    if unit.GetRace() == data.RaceFantastic {
        base = unit.GetUpkeepMana()
        if player.Wizard.RetortEnabled(data.RetortConjurer) {
            base = (base * 3 + 3) / 4
        }
    }
    for _, enchantment := range unit.GetUpkeepEnchantments() {
        base += enchantment.UpkeepMana()
    }
    return base
}

// Player_Armies_And_Enchantments_Mana_Upkeep: units, city enchantments the player cast (curses
// too) and global enchantments; Channeler halves the sum, once
func (player *Player) classicManaUpkeep(cityEnchantmentsProvider CityEnchantmentsProvider) int {
    return player.classicManaUpkeepWith(cityEnchantmentsProvider, 0)
}

// the same with the upkeep of one more unit
func (player *Player) classicManaUpkeepWith(cityEnchantmentsProvider CityEnchantmentsProvider, extraUnitMana int) int {
    upkeep := player.TotalUnitUpkeepMana() + extraUnitMana + player.TotalEnchantmentUpkeep(cityEnchantmentsProvider)
    if player.Wizard.RetortEnabled(data.RetortChanneler) {
        upkeep /= 2
    }
    return upkeep
}

// the units that eat: not creatures, not heroes, not undead (Player_Armies_Food_Upkeep)
func (player *Player) NormalUnits() int {
    count := 0
    for unit := range player.Units() {
        if unit.GetUpkeepFood() > 0 {
            count += 1
        }
    }
    return count
}

// the gold of food. Update_Players_Gold_Reserve: half the surplus of every city on its own, less
// half the units that eat, never below 0. the summary (planning): half the food total when above 0
func (player *Player) foodSale(planning bool) int {
    if planning {
        return max(0, player.foodPerTurn(true) / 2)
    }
    sale := 0
    for _, city := range player.CitiesInOrder() {
        sale += max(0, city.SurplusFood()) / 2
    }
    return max(0, sale - player.NormalUnits() / 2)
}

// Player_Magic_Power_Distribution: mana and skill rounded to the nearest, research the rest; with
// no research the rest goes to skill (to mana with no skill either); Mana Focusing a quarter more
// mana, rounded down
func (player *Player) classicPowerSplit(power int) (int, int, int) {
    percent := func(share float64) int {
        return int(share * 100 + 0.5)
    }
    manaShare := percent(player.PowerDistribution.Mana)
    skillShare := percent(player.PowerDistribution.Skill)
    researchShare := percent(player.PowerDistribution.Research)

    mana := (power * manaShare + 50) / 100
    skill := (power * skillShare + 50) / 100
    research := power - mana - skill
    if researchShare == 0 && research > 0 {
        if skillShare == 0 {
            mana += research
        } else {
            skill += research
        }
        research = 0
    }
    if player.Wizard.RetortEnabled(data.RetortManaFocusing) {
        mana += mana / 4
    }
    return mana, research, skill
}

// the research income the list of candidates is sorted by, set every turn (StartPlayerTurn)
func (player *Player) SetResearchIncome(income float64) {
    player.researchIncome = income
}

// Build_Research_List: every spell that can be researched of the lowest tier that still has one,
// in every realm; never the Spell of Mastery
func (player *Player) researchPool() []spellbook.Spell {
    available := player.ResearchPoolSpells.Copy()
    available.RemoveSpells(player.KnownSpells)
    available.RemoveSpells(player.ResearchCandidateSpells)

    realms := []data.MagicType{data.NatureMagic, data.SorceryMagic, data.ChaosMagic, data.LifeMagic, data.DeathMagic, data.ArcaneMagic}
    rarities := []spellbook.SpellRarity{spellbook.SpellRarityCommon, spellbook.SpellRarityUncommon, spellbook.SpellRarityRare, spellbook.SpellRarityVeryRare}

    var pool []spellbook.Spell
    for _, realm := range realms {
        realmSpells := available.GetSpellsByMagic(realm)
        for _, rarity := range rarities {
            var tier []spellbook.Spell
            for _, spell := range realmSpells.GetSpellsByRarity(rarity).Spells {
                if !spell.IsSpellOfMastery() && spell.Name != "Spell of Return" {
                    tier = append(tier, spell)
                }
            }
            if len(tier) > 0 {
                slices.SortFunc(tier, func(a spellbook.Spell, b spellbook.Spell) int {
                    return a.Index - b.Index
                })
                pool = append(pool, tier...)
                break
            }
        }
    }
    return pool
}

// Player_Research_Spells: the empty places of the list filled by chance from the pool, the pool
// made anew after each; the Spell of Mastery only when nothing else is left, at its own cost
func (player *Player) classicResearchCandidates() {
    for len(player.ResearchCandidateSpells.Spells) < researchCandidates {
        pool := player.researchPool()
        if len(pool) == 0 {
            break
        }
        player.ResearchCandidateSpells.AddSpell(pool[rand.N(len(pool))])
    }

    mastery := player.ResearchPoolSpells.FindByName(spellbook.SpellOfMastery.Name)
    if len(player.ResearchCandidateSpells.Spells) < researchCandidates && len(player.researchPool()) == 0 &&
       mastery.Valid() && !player.KnownSpells.Contains(mastery) && !player.ResearchCandidateSpells.Contains(mastery) {
        player.ResearchCandidateSpells.AddSpell(mastery)
    }

    // the Spell of Mastery on the list costs what it costs now (Player_Gets_Spell lowers it)
    for i := range player.ResearchCandidateSpells.Spells {
        if player.ResearchCandidateSpells.Spells[i].IsSpellOfMastery() {
            player.ResearchCandidateSpells.Spells[i].ResearchCost = player.SpellOfMasteryCost
        }
    }

    player.sortResearchCandidates()
}

// Sort_Research_List: by the turns each takes at the research income, with the bonus of its realm
func (player *Player) sortResearchCandidates() {
    firstOfRealm := make(map[data.MagicType]spellbook.Spell)
    for _, list := range []spellbook.Spells{player.ResearchPoolSpells, player.KnownSpells} {
        for _, spell := range list.Spells {
            first, ok := firstOfRealm[spell.Magic]
            if !ok || spell.Index < first.Index {
                firstOfRealm[spell.Magic] = spell
            }
        }
    }

    turns := func(spell spellbook.Spell) int {
        judged := spell
        if quirkResearchBonusOfFirstSpell {
            if first, ok := firstOfRealm[spell.Magic]; ok {
                judged = first
            }
        }
        income := player.ComputeEffectiveResearchPerTurn(player.researchIncome, judged)
        if income <= 0 {
            return spell.ResearchCost
        }
        return spell.ResearchCost / income
    }

    slices.SortStableFunc(player.ResearchCandidateSpells.Spells, func(a spellbook.Spell, b spellbook.Spell) int {
        return turns(a) - turns(b)
    })
}

// the mana a unit costs a turn (Unit_Mana_Upkeep)
func (player *Player) UnitManaUpkeep(unit units.StackUnit) int {
    if ClassicEconomy {
        return player.unitManaUpkeep(unit)
    }
    total := unit.GetUpkeepMana()
    for _, enchantment := range unit.GetUpkeepEnchantments() {
        total += enchantment.UpkeepMana()
    }
    return total
}

// all the mana upkeep of a turn, as it is paid
func (player *Player) ManaUpkeep(cityEnchantmentsProvider CityEnchantmentsProvider) int {
    if ClassicEconomy {
        return player.classicManaUpkeep(cityEnchantmentsProvider)
    }
    return player.TotalUnitUpkeepMana() + player.TotalEnchantmentUpkeep(cityEnchantmentsProvider)
}
