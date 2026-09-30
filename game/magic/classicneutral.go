package main

// What a neutral city of the original's worlds starts with: buildings by its size and a garrison
// (ReMoM MoM/src/MAPGEN.c: Generate_Neutral_Cities). The code is ours.
//
//   buildings, the city's people less 2 from the smallest up: Barracks (0), Smithy (1), Builders'
//     Hall (2), Armory (3), Granary (5), Stables (6), City Walls (8), Fighters' Guild (10), Armorers'
//     Guild (11), Shrine (12); without the ones its race can not build, then (one pass in the order
//     of the buildings) without the ones whose buildings before them it lacks
//   garrison: people / 4 (at most 9) of its best close combat unit, then as many of its best unit
//     with bows or magic, else of the close combat one again; "best" is the last of the race's units
//     in the game's order it could build, no settlers, engineers or ships (Dark Elves have no close
//     combat unit, so they get half: the original's slip, kept)

import (
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    buildinglib "github.com/kazzmir/master-of-magic/game/magic/building"
    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

var classicNeutralBuildings = []struct {
    From int
    Building buildinglib.Building
}{
    {0, buildinglib.BuildingBarracks},
    {1, buildinglib.BuildingSmithy},
    {2, buildinglib.BuildingBuildersHall},
    {3, buildinglib.BuildingArmory},
    {5, buildinglib.BuildingGranary},
    {6, buildinglib.BuildingStables},
    {8, buildinglib.BuildingCityWalls},
    {10, buildinglib.BuildingFightersGuild},
    {11, buildinglib.BuildingArmorersGuild},
    {12, buildinglib.BuildingShrine},
}

const classicStackMost = 9

// ranged of bows or magic (the original's Ranged_Type below srat_Thrown)
func classicShoots(unit *units.Unit) bool {
    if unit.RangedAttackPower <= 0 {
        return false
    }
    switch unit.RangedAttackDamageType {
        case units.DamageRangedMagical, units.DamageRangedPhysical: return true
    }
    return false
}

func classicGarrisonKind(unit *units.Unit, race data.Race) bool {
    return unit.Race == race && !unit.HasAbility(data.AbilityCreateOutpost) && !unit.HasAbility(data.AbilityConstruction) && !unit.HasAbility(data.AbilityTransport)
}

func classicNeutralSetup(player *playerlib.Player, city *citylib.City) {
    people := city.Citizens()
    allowed := city.GetBuildableBuildings()
    for _, entry := range classicNeutralBuildings {
        if people - 2 >= entry.From && allowed.Contains(entry.Building) {
            city.Buildings.Insert(entry.Building)
        }
    }
    for _, building := range buildinglib.Buildings() {
        if !city.Buildings.Contains(building) {
            continue
        }
        for _, dependency := range city.BuildingInfo.Dependencies(building) {
            if !city.Buildings.Contains(dependency) {
                city.Buildings.Remove(building)
                break
            }
        }
    }

    var melee, ranged units.Unit
    melee = units.UnitNone
    ranged = units.UnitNone
    for _, unit := range city.ComputePossibleUnits() {
        if !classicGarrisonKind(&unit, city.Race) {
            continue
        }
        if classicShoots(&unit) {
            ranged = unit
        } else {
            melee = unit
        }
    }

    count := min(people / 4, classicStackMost)
    second := ranged
    if second.IsNone() {
        second = melee
    }
    for _, unit := range []units.Unit{melee, second} {
        if unit.IsNone() {
            continue
        }
        for range count {
            player.AddUnit(units.MakeOverworldUnitFromUnit(unit, city.X, city.Y, city.Plane, player.Wizard.Banner, player.MakeExperienceInfo(), player.MakeUnitEnchantmentProvider()))
        }
    }
    city.ResetCitizens()
}
