package game

// Overland magic as the original game has it. Rules in words: docs/mod/magic.md.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/Spells129.c:
// Apply_Automatic_Spell_Counters, Cast_Stasis, Apply_Call_The_Void, Cast_Warp_Node,
// Cast_Move_Fortress, Apply_Cruel_Unminding; Spells130.c: Apply_Drain_Power, Cast_Spell_Binding,
// Cast_Change_Terrain, Cast_Corruption, Apply_Subversion; Spells132.c: WIZ_MeteorStorm,
// WIZ_GreatWasting, WIZ_Armageddon, Cast_Plane_Shift, Cast_Floating_Island, Cast_Summon_Hero;
// OverSpel.c: Cast_Disenchant (read by its checked combat twin), Calculate_Dispel_Difficulty;
// Terrain.c: All_City_Nightshade_Count; SBookScr.c: Cast_Spell_Overland_Do; NEXTTURN.c:
// Next_Turn_Calc, Pick_Random_Hero, Do_All_Units_XP_Check). The code is ours.

import (
    "math/rand/v2"

    buildinglib "github.com/kazzmir/master-of-magic/game/magic/building"
    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    "github.com/kazzmir/master-of-magic/game/magic/combat"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    herolib "github.com/kazzmir/master-of-magic/game/magic/hero"
    "github.com/kazzmir/master-of-magic/game/magic/maplib"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    "github.com/kazzmir/master-of-magic/game/magic/terrain"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// false: upstream's overland magic
var ClassicMagic = true

// WIZ_MeteorStorm asks the counters of a city with the human as the caster, whoever cast the storm
const quirkMeteorStormHumanCounter = true

// nobody's retorts: the dispel difficulty of magic without an owner
type noRetorts struct{}

func (noRetorts) RetortEnabled(retort data.Retort) bool {
    return false
}

// Cast_Spell_Overland_Do: the strength of a spell as cast, the table cost or the wizard's cost when
// that is higher, with the extra of a slider
func classicCastStrength(player *playerlib.Player, spell spellbook.Spell) int {
    if spell.OverrideCost > 0 {
        return spell.OverrideCost
    }
    return max(spell.CastCost, player.ComputeEffectiveSpellCost(spell, true))
}

// All_City_Nightshade_Count: the Nightshade of the squares a city works, when it has a building of
// religion or learning
func classicNightshadeCount(city *citylib.City) int {
    any := false
    for _, building := range []buildinglib.Building{buildinglib.BuildingShrine, buildinglib.BuildingTemple,
        buildinglib.BuildingParthenon, buildinglib.BuildingCathedral, buildinglib.BuildingSagesGuild,
        buildinglib.BuildingOracle, buildinglib.BuildingAlchemistsGuild, buildinglib.BuildingWizardsGuild} {
        if city.Buildings.Contains(building) {
            any = true
        }
    }
    if !any {
        return 0
    }
    count := 0
    for _, tile := range city.GetCatchmentArea() {
        if tile.GetBonus() == data.BonusNightshade {
            count += 1
        }
    }
    return count
}

// the ward of a realm
func classicWardOf(magic data.MagicType) data.CityEnchantment {
    switch magic {
        case data.DeathMagic: return data.CityEnchantmentDeathWard
        case data.LifeMagic: return data.CityEnchantmentLifeWard
        case data.NatureMagic: return data.CityEnchantmentNatureWard
        case data.ChaosMagic: return data.CityEnchantmentChaosWard
        case data.SorceryMagic: return data.CityEnchantmentSorceryWard
    }
    return data.CityEnchantmentNone
}

// Apply_Automatic_Spell_Counters: Consecration counters death and chaos, a ward its realm, and the
// Nightshade of the city rolls against the strength of the spell and the caster's retorts
func classicCountered(city *citylib.City, spell spellbook.Spell, caster *playerlib.Player, strength int) bool {
    if city == nil {
        return false
    }
    countered := false
    if city.HasEnchantment(data.CityEnchantmentConsecration) && (spell.Magic == data.DeathMagic || spell.Magic == data.ChaosMagic) {
        countered = true
    }
    if ward := classicWardOf(spell.Magic); ward != data.CityEnchantmentNone && city.HasEnchantment(ward) {
        countered = true
    }
    if count := classicNightshadeCount(city); count > 0 {
        var owner spellbook.RetortOwner = noRetorts{}
        if caster != nil {
            owner = &caster.Wizard
        }
        if spellbook.RollDispelChance(spellbook.ComputeDispelChance(100 * count, strength, spell.Magic, owner)) {
            countered = true
        }
    }
    return countered
}

// the counters of a city against a spell cast at it: the original's, or upstream's
func (game *Game) cityCounters(city *citylib.City, spell spellbook.Spell, caster *playerlib.Player) bool {
    if city == nil {
        return false
    }
    if ClassicMagic {
        return classicCountered(city, spell, caster, classicCastStrength(caster, spell))
    }
    return city.CheckDispel(spell)
}

// upstream refuses a spell at a stack in a city warded against it; the original lets the counters
// decide
func (game *Game) cityRefuses(city *citylib.City, spell spellbook.Spell) bool {
    if city == nil || ClassicMagic {
        return false
    }
    return !city.CanTarget(spell)
}

// Apply_Drain_Power: 10 rolls of 1 to 20
func classicDrainAmount() int {
    amount := 0
    for range 10 {
        amount += rand.N(20) + 1
    }
    return amount
}

// Apply_Cruel_Unminding: 1 to 10 percent of the skill, at least 1
func classicUnminding(skill int) int {
    return max(1, skill * (rand.N(10) + 1) / 100)
}

// units that went into Stasis this turn: they roll from the next turn on (Do_All_Units_XP_Check)
var classicStasisNew = make(map[units.StackUnit]bool)

// the turn of Stasis: a unit put in it this turn waits, the others are freed on 1d10 at most their
// sorcery resistance - 5
func classicStasisTurn(stack *playerlib.UnitStack) {
    for _, unit := range stack.Units() {
        if unit.GetBusy() != units.BusyStatusStasis {
            continue
        }
        if classicStasisNew[unit] {
            delete(classicStasisNew, unit)
            continue
        }
        if rand.N(10) + 1 <= combat.GetResistanceFor(unit, data.SorceryMagic) - 5 {
            unit.SetBusy(units.BusyStatusNone)
        }
    }
}

// Cast_Disenchant: every enchantment on the square that is not the caster's, units with Spell Lock
// only that, the caster's own units out of Stasis, a warped node back; table costs, the owner's
// retorts, out of 250
func (game *Game) classicDisenchant(player *playerlib.Player, spell spellbook.Spell, disenchantTrue bool, tileX int, tileY int) {
    strength := spell.Cost(true)
    if disenchantTrue {
        strength *= 3
    }
    if player.Wizard.RetortEnabled(data.RetortRunemaster) {
        strength *= 2
    }
    allSpells := game.AllSpells()
    roll := func(cost int, magic data.MagicType, owner spellbook.RetortOwner) bool {
        return spellbook.RollDispelChance(spellbook.ComputeDispelChance(strength, cost, magic, owner))
    }
    ownerOf := func(banner data.BannerType) spellbook.RetortOwner {
        owner := game.GetPlayerByBanner(banner)
        if owner == nil {
            return noRetorts{}
        }
        return &owner.Wizard
    }

    city, _ := game.Model.FindCity(tileX, tileY, game.Model.Plane)
    if city != nil {
        for _, enchantment := range city.Enchantments.Values() {
            if enchantment.Owner == player.GetBanner() {
                continue
            }
            target := allSpells.FindByName(enchantment.Enchantment.SpellName())
            if roll(target.CastCost, target.Magic, ownerOf(enchantment.Owner)) {
                city.RemoveEnchantments(enchantment.Enchantment)
            }
        }
    }

    stack, owner := game.Model.FindStack(tileX, tileY, game.Model.Plane)
    if stack != nil && owner == player {
        stasis := allSpells.FindByName("Stasis")
        for _, unit := range stack.Units() {
            if unit.GetBusy() == units.BusyStatusStasis && roll(stasis.CastCost, stasis.Magic, &player.Wizard) {
                unit.SetBusy(units.BusyStatusNone)
                delete(classicStasisNew, unit)
            }
        }
    } else if stack != nil {
        for _, unit := range stack.Units() {
            if unit.HasEnchantment(data.UnitEnchantmentSpellLock) {
                // only the Spell Lock is rolled, at 150
                if roll(150, data.SorceryMagic, &owner.Wizard) {
                    unit.RemoveEnchantment(data.UnitEnchantmentSpellLock)
                }
                continue
            }
            var toRemove []data.UnitEnchantment
            for _, enchantment := range unit.GetEnchantments() {
                target := allSpells.FindByName(enchantment.SpellName())
                if roll(target.CastCost, target.Magic, &owner.Wizard) {
                    toRemove = append(toRemove, enchantment)
                }
            }
            for _, enchantment := range toRemove {
                unit.RemoveEnchantment(enchantment)
            }
        }
    }

    magicNode := game.GetMap(game.Model.Plane).GetMagicNode(tileX, tileY)
    if magicNode != nil && magicNode.Warped {
        warpNode := allSpells.FindByName("Warp Node")
        if roll(warpNode.CastCost, warpNode.Magic, noRetorts{}) {
            magicNode.Warped = false
        }
    }
}

// within 2 squares of a city of the player on the plane (the larger of the distances, x wraps)
func classicNearOwnCity(player *playerlib.Player, mapUse *maplib.Map, x int, y int) bool {
    for _, city := range player.Cities {
        if city.Plane != mapUse.Plane {
            continue
        }
        dx := mapUse.XDistance(city.X, x)
        if dx < 0 {
            dx = -dx
        }
        dy := city.Y - y
        if dy < 0 {
            dy = -dy
        }
        if max(dx, dy) <= 2 {
            return true
        }
    }
    return false
}

// WIZ_Armageddon and WIZ_GreatWasting: 4 to 6 tries a caster, each up to 50 squares by chance;
// a try that finds none still counts
func (game *Game) classicWorldTries(enchantment data.Enchantment, fits func(player *playerlib.Player, mapUse *maplib.Map, x int, y int) bool, apply func(player *playerlib.Player, mapUse *maplib.Map, x int, y int)) {
    maps := []*maplib.Map{game.Model.ArcanusMap, game.Model.MyrrorMap}
    for _, player := range game.Model.Players {
        if player.Defeated || !player.GlobalEnchantments.Contains(enchantment) {
            continue
        }
        for range rand.N(3) + 3 {
            for range 50 {
                mapUse := maps[rand.N(2)]
                if mapUse == nil {
                    continue
                }
                x := rand.N(mapUse.Width())
                y := rand.N(mapUse.Height())
                if fits(player, mapUse, x, y) {
                    apply(player, mapUse, x, y)
                    break
                }
            }
        }
    }
}

func (game *Game) classicArmageddon() {
    game.classicWorldTries(data.EnchantmentArmageddon, func(player *playerlib.Player, mapUse *maplib.Map, x int, y int) bool {
        tile := mapUse.GetTile(x, y).Tile
        if tile.IsWater() || tile.IsRiver() || tile.TerrainType() == terrain.Volcano || mapUse.HasVolcano(x, y) || mapUse.HasMagicNode(x, y) {
            return false
        }
        return !classicNearOwnCity(player, mapUse, x, y)
    }, func(player *playerlib.Player, mapUse *maplib.Map, x int, y int) {
        mapUse.SetVolcano(x, y, player)
    })
}

func (game *Game) classicGreatWasting() {
    game.classicWorldTries(data.EnchantmentGreatWasting, func(player *playerlib.Player, mapUse *maplib.Map, x int, y int) bool {
        if mapUse.GetTile(x, y).Tile.IsWater() || mapUse.HasCorruption(x, y) {
            return false
        }
        return !classicNearOwnCity(player, mapUse, x, y)
    }, func(player *playerlib.Player, mapUse *maplib.Map, x int, y int) {
        mapUse.SetCorruption(x, y)
    })
}

// WIZ_MeteorStorm, once a turn for every wizard who has it: every city not countered loses each
// building at 1% when it is not the caster's; every unit outside a city takes the storm, the
// caster's own too
func (game *Game) classicMeteorStorm() {
    allSpells := game.AllSpells()
    storm := allSpells.FindByName("Meteor Storm")
    human := game.Model.GetHumanPlayer()
    for _, caster := range game.Model.Players {
        if caster.Defeated || !caster.HasEnchantment(data.EnchantmentMeteorStorm) {
            continue
        }
        info := game.Model.ComputeCityStackInfo()
        for _, owner := range game.Model.Players {
            for _, city := range owner.Cities {
                counterCaster := caster
                if quirkMeteorStormHumanCounter {
                    // the human as the caster, with no strength of a spell of his
                    counterCaster = human
                }
                if classicCountered(city, storm, counterCaster, 0) || owner == caster {
                    continue
                }
                for _, building := range city.Buildings.Values() {
                    if rand.N(100) == 0 {
                        city.Buildings.Remove(building)
                    }
                }
            }
        }
        for _, owner := range game.Model.Players {
            var dead []units.StackUnit
            for unit := range owner.Units() {
                if info.FindCity(unit.GetX(), unit.GetY(), unit.GetPlane()) != nil {
                    continue
                }
                combat.ApplyAreaDamage(&UnitDamageWrapper{StackUnit: unit}, 4, units.DamageImmolation, 0)
                if unit.GetHealth() <= 0 {
                    dead = append(dead, unit)
                }
            }
            for _, unit := range dead {
                owner.RemoveUnit(unit)
            }
        }
    }
}

// the city summons appear in: the summoning circle, else the fortress, which takes the circle
// (the original keeps a summoning point that falls back to the fortress)
func classicSummonCity(player *playerlib.Player) *citylib.City {
    city := player.FindSummoningCity()
    if city != nil || !ClassicMagic {
        return city
    }
    fortress := player.FindFortressCity()
    if fortress != nil {
        fortress.Buildings.Insert(buildinglib.BuildingSummoningCircle)
    }
    return fortress
}

// Pick_Random_Hero for Summon Hero and Summon Champion: up to 100 draws of the class, no fame,
// the books of the Priestess, Paladin, Black Knight and Necromancer; a champion that was not found
// gets one more draw among all
func classicSummonPick(player *playerlib.Player, champion bool) *herolib.Hero {
    var class, all []*herolib.Hero
    for _, hero := range player.HeroPool {
        if hero.HeroType == herolib.HeroTorin {
            continue
        }
        all = append(all, hero)
        if hero.IsChampion() == champion {
            class = append(class, hero)
        }
    }
    fits := func(hero *herolib.Hero) bool {
        if hero.Status != herolib.StatusAvailable {
            return false
        }
        switch hero.HeroType {
            case herolib.HeroElana, herolib.HeroRoland: return player.HasLifeBooks()
            case herolib.HeroMortu, herolib.HeroRavashack: return player.HasDeathBooks()
        }
        return true
    }
    if len(class) > 0 {
        for range 100 {
            if hero := class[rand.N(len(class))]; fits(hero) {
                return hero
            }
        }
    }
    if champion && len(all) > 0 {
        if hero := all[rand.N(len(all))]; fits(hero) {
            return hero
        }
    }
    return nil
}

// Cast_Plane_Shift: the whole stack to the other plane, refused by a Planar Seal, any unit there,
// or water it can not enter
func (game *Game) classicPlaneShift(stack *playerlib.UnitStack, player *playerlib.Player) bool {
    if game.IsGlobalEnchantmentActive(data.EnchantmentPlanarSeal) {
        return false
    }
    other := stack.Plane().Opposite()
    if found, _ := game.Model.FindStack(stack.X(), stack.Y(), other); found != nil {
        return false
    }
    mapUse := game.GetMap(other)
    if mapUse.GetTile(stack.X(), stack.Y()).Tile.IsWater() {
        water := false
        for _, unit := range stack.Units() {
            if unit.IsFlying() || unit.IsSwimmer() || unit.IsSailing() || unit.HasAbility(data.AbilityWindWalking) {
                water = true
            }
        }
        if !water {
            return false
        }
    }
    // the fork keeps refusing a lair there (MY CALL: a lair holds no units in the original)
    if mapUse.GetEncounter(stack.X(), stack.Y()) != nil {
        return false
    }
    // nobody stands there, so nothing to merge with
    stack.SetPlane(other)
    player.UpdateFogVisibility()
    game.discoverWizards(nil)
    return true
}
