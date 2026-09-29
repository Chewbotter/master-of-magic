package main

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/combat"
    "github.com/kazzmir/master-of-magic/game/magic/mod"
)

// the ground and the coast that are picked in the list of the test battle are the ones of its
// battle, and are found again by the names they are kept under
func TestTestBattleGround(test *testing.T) {
    ground, coast := testBattleGround, testBattleCoast
    defer func() {
        testBattleGround, testBattleCoast = ground, coast
    }()

    for index, picked := range testGrounds {
        for _, side := range testCoasts {
            testBattleGround = index
            testBattleCoast = side

            landscape, zone := testBattleZone()
            if landscape != picked.Landscape || zone.Ground.Biome != picked.Biome || zone.Ground.Coast != side {
                test.Fatalf("%v with coast %v: landscape %v, ground %+v", picked.Name, side, landscape, zone.Ground)
            }
            // a biome is a kind of the landscape it is listed with
            if combat.BiomeLandscape(picked.Biome, picked.Landscape) != picked.Landscape {
                test.Fatalf("%v is no kind of landscape %v", picked.Name, picked.Landscape)
            }
            if zone.Ground.Forest != (picked.Biome == mod.BiomeForest) || zone.Ground.Hills != (picked.Biome == mod.BiomeHills) {
                test.Fatalf("%v: %+v", picked.Name, zone.Ground)
            }

            if testGroundByName(picked.Name) != index || testCoastByName(side.String()) != side {
                test.Fatalf("%v, %v are not found by their names", picked.Name, side)
            }
            if testCoastNames[side] == "" {
                test.Fatalf("coast %v has no name in the list", side)
            }
        }
    }

    // every biome the game shows can be picked
    for _, biome := range mod.Biomes {
        if !biome.Planned && testGrounds[testGroundByName(biome.Name)].Biome != biome.Name {
            test.Fatalf("biome %v is not in the list", biome.Name)
        }
    }

    // the weather
    weather := testBattleWeather
    defer func() {
        testBattleWeather = weather
    }()
    for _, picked := range combat.Weathers {
        testBattleWeather = picked
        _, zone := testBattleZone()
        if zone.Ground.Weather != picked || testWeatherNames[picked] == "" {
            test.Fatalf("weather %v: %+v", picked, zone.Ground)
        }
    }
    testBattleWeather = weather

    // the fields of a town
    farmland := testBattleFarmland
    defer func() {
        testBattleFarmland = farmland
    }()
    for away := range testFarmlandNames {
        testBattleFarmland = away
        _, zone := testBattleZone()
        if zone.Ground.Farmland != away || away > combat.FarmlandFar {
            test.Fatalf("farmland %v: %+v", away, zone.Ground)
        }
    }
    testBattleFarmland = farmland

    // and the river
    river := testBattleRiver
    defer func() {
        testBattleRiver = river
    }()
    for _, course := range testRivers {
        testBattleRiver = course
        _, zone := testBattleZone()
        if zone.Ground.River != course || testRiverByName(course.String()) != course || testRiverNames[course] == "" {
            test.Fatalf("river %v: %+v", course, zone.Ground)
        }
    }

    if testGroundByName("what is not there") != 0 || testCoastByName("nowhere") != combat.CoastNone {
        test.Fatalf("names that are not there")
    }
}
