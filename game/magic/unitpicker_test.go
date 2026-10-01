package main

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
)

// the ground that is picked in the list of the test battle is the one of its battle, and is found
// again by the name it is kept under; and so are the roads
func TestTestBattleGround(test *testing.T) {
    ground := testBattleGround
    defer func() {
        testBattleGround = ground
    }()

    for index, picked := range testGrounds {
        testBattleGround = index
        landscape, zone := testBattleZone()
        if landscape != picked.Landscape || zone.Ground.Forest != picked.Forest || zone.Ground.Hills != picked.Hills {
            test.Fatalf("%v: landscape %v, ground %+v", picked.Name, landscape, zone.Ground)
        }
        if testGroundByName(picked.Name) != index {
            test.Fatalf("%v is not found by its name", picked.Name)
        }
    }

    // the roads
    road := testBattleRoad
    defer func() {
        testBattleRoad = road
    }()
    for index, picked := range testRoads {
        testBattleRoad = index
        _, zone := testBattleZone()
        if testRoadByName(picked.Key) != index || picked.Name == "" {
            test.Fatalf("the roads %v are not found by their name", picked.Key)
        }
        if picked.Any {
            continue
        }
        for side, has := range zone.Ground.Roads {
            want := len(picked.Sides) > 0 && side == 4
            for _, of := range picked.Sides {
                want = want || of == side
            }
            if has != want {
                test.Fatalf("the roads %v: side %v is %v", picked.Key, side, has)
            }
        }
        if zone.Ground.EnchantedRoads != picked.Enchanted {
            test.Fatalf("the roads %v: enchanted %v", picked.Key, zone.Ground.EnchantedRoads)
        }
    }
}

func TestAllOfRace(test *testing.T) {
    all := allUnitOf(data.RaceHighMen)
    if !isAllOfRace(all) || all.Name != "All High Men" || findUnit(unitFullName(all)) != all {
        test.Fatalf("all of the High Men: %+v", all)
    }
    if isAllOfRace(findUnit("High Men Swordsmen")) {
        test.Fatalf("one unit counts as all")
    }

    kinds := 0
    for _, unit := range unitsOfRace(data.RaceHighMen) {
        if !unit.IsSettlers() {
            kinds += 1
        }
    }
    army := testArmyUnits(all, 2)
    if kinds == 0 || len(army) != kinds * 2 {
        test.Fatalf("%v units in the army, %v kinds", len(army), kinds)
    }
    for _, unit := range army {
        if unit.IsSettlers() || unit.Race != data.RaceHighMen {
            test.Fatalf("%v in the army", unit.Name)
        }
    }

    // only the races of towns
    for _, race := range []data.Race{data.RaceFantastic, data.RaceHero, data.RaceAll} {
        if hasAllOfRace(race) || findUnit(unitFullName(allUnitOf(race))) != nil {
            test.Fatalf("all of %v", race)
        }
    }

    if one := testArmyUnits(findUnit("High Men Swordsmen"), 1); len(one) != TestBattleUnits {
        test.Fatalf("one kind: %v units", len(one))
    }
}
