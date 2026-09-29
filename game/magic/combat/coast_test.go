package combat

import (
    "testing"

    "github.com/kazzmir/master-of-magic/lib/fraction"
)

// a coast on every side: the places of the armies are land, the beach lies between the land and the
// water and is 2 to 3 cells wide, the water goes on to the end, and both are flat and without what
// does not belong there
func TestCoast(test *testing.T) {
    for _, side := range []CoastSide{CoastEast, CoastNorth, CoastWest, CoastSouth} {
        for _, large := range []bool{false, true} {
            zone := ZoneType{}
            zone.Ground.Coast = side
            zone.Ground.LargeArmy = large
            zone.Ground.Roads = [9]bool{true, true, true, true, true, true, true, true, true}
            ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeMountain, 0, zone)

            units := deployPlaces
            if large {
                units = deployPlaces * 3
            }
            for _, team := range []Team{TeamDefender, TeamAttacker} {
                for _, cell := range deployCells(team, units, false) {
                    if ground.GroupAt(cell.X, cell.Y) == TerrainSand || ground.GroupAt(cell.X, cell.Y) == TerrainWater {
                        test.Fatalf("coast %v, large %v: a place of an army at %v is %v", side, large, cell, ground.GroupAt(cell.X, cell.Y))
                    }
                }
            }

            sand, water := 0, 0
            for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
                for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
                    group := ground.GroupAt(cgx, cgy)
                    depth, _ := side.depth(cgx, cgy, large)

                    switch group {
                        case TerrainSand: sand += 1
                        case TerrainWater: water += 1
                    }

                    if group == TerrainSand || group == TerrainWater {
                        if depth < 0 {
                            test.Fatalf("coast %v: %v inland at %v, %v", side, group, cgx, cgy)
                        }
                        if ground.HeightAt(cgx, cgy) != 0 {
                            test.Fatalf("coast %v: a plateau on the coast at %v, %v", side, cgx, cgy)
                        }
                        if ground.sceneryAllowed(cgx, cgy) {
                            test.Fatalf("coast %v: trees can stand on the coast at %v, %v", side, cgx, cgy)
                        }
                    }
                    if group == TerrainWater && ground.RoadAt(cgx, cgy) != 0 {
                        test.Fatalf("coast %v: a road on the water at %v, %v", side, cgx, cgy)
                    }

                    // from the farthest the beach can start at, all is beach or water, and from the
                    // farthest the water can start at, water
                    if depth >= coastWander && group != TerrainSand && group != TerrainWater {
                        test.Fatalf("coast %v: %v at depth %v", side, group, depth)
                    }
                    if depth >= coastWander + coastBeachMax && group != TerrainWater {
                        test.Fatalf("coast %v: %v at depth %v", side, group, depth)
                    }
                    if depth < coastBeachMin && group == TerrainWater {
                        test.Fatalf("coast %v: water at depth %v", side, depth)
                    }
                }
            }
            if sand == 0 || water == 0 {
                test.Fatalf("coast %v: %v cells of beach, %v of water", side, sand, water)
            }

            // the pictures are the ones of the beach and of the water
            for index, group := range ground.Group {
                picture := ground.Picture[index]
                switch group {
                    case TerrainSand:
                        if groundVariant(picture, groundSandFirst, groundSandCount) < 0 {
                            test.Fatalf("coast %v: picture %v on the beach", side, picture)
                        }
                    case TerrainWater:
                        if groundVariant(picture, groundWaterFirst, groundWaterCount) < 0 {
                            test.Fatalf("coast %v: picture %v on the water", side, picture)
                        }
                    default:
                        if picture >= groundSandFirst && picture < groundExtraFirst {
                            test.Fatalf("coast %v: picture %v on %v", side, picture, group)
                        }
                }
            }
        }
    }

    // no coast, no beach
    ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, ZoneType{})
    for _, group := range ground.Group {
        if group == TerrainSand || group == TerrainWater {
            test.Fatalf("%v without a coast", group)
        }
    }
}

// the beach costs twice what grass costs, and the water is for what can enter it
func TestCoastRules(test *testing.T) {
    zone := ZoneType{}
    zone.Ground.Coast = CoastEast
    ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, zone)
    tiles := makeTiles(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, zone)
    ground.applyTo(tiles)
    model := &CombatModel{Tiles: tiles, Ground: ground}

    // along a row of the grid toward the east: land, beach, water
    _, lastX, firstY, _ := armyPlaces(false)
    var land, beach, sea [2]int
    found := 0
    for cgx := lastX; cgx < ground.MinX + ground.Width && found < 2; cgx++ {
        x, y := CellToTile(cgx, firstY)
        if y >= BattlefieldHeight {
            break
        }
        switch {
            case found == 0 && ground.GroupAt(cgx, firstY) == TerrainSand:
                landX, landY := CellToTile(cgx - 1, firstY)
                land = [2]int{landX, landY}
                beach = [2]int{x, y}
                found = 1
            case found == 1 && ground.GroupAt(cgx, firstY) == TerrainWater:
                sea = [2]int{x, y}
                found = 2
        }
    }
    if found != 2 {
        test.Fatalf("no beach and water on the field toward the east")
    }

    if !model.IsWaterTile(sea[0], sea[1]) || model.IsWaterTile(beach[0], beach[1]) || model.IsWaterTile(land[0], land[1]) {
        test.Fatalf("water: sea %v, beach %v, land %v", model.IsWaterTile(sea[0], sea[1]), model.IsWaterTile(beach[0], beach[1]), model.IsWaterTile(land[0], land[1]))
    }

    if halves := model.cellMoveHalves(beach[0], beach[1], false); halves != moveHalvesRough {
        test.Fatalf("the beach costs %v halves", halves)
    }
    if halves := model.cellMoveHalves(beach[0], beach[1], true); halves != moveHalvesGround {
        test.Fatalf("the beach costs a flying unit %v halves", halves)
    }
    if cost := model.cellMoveHalves(sea[0], sea[1], false); fraction.Make(cost, 2).LessThan(fraction.FromInt(1)) {
        test.Fatalf("the water costs %v halves", cost)
    }

    // a path into the water: not for a unit of the land
    if _, ok := model.computePath(land[0], land[1], sea[0], sea[1], false, false, false); ok {
        test.Fatalf("a unit of the land finds a path into the water")
    }
    if _, ok := model.computePath(land[0], land[1], sea[0], sea[1], false, false, true); !ok {
        test.Fatalf("a unit that swims finds no path into the water")
    }
    if _, ok := model.computePath(land[0], land[1], beach[0], beach[1], false, false, false); !ok {
        test.Fatalf("a unit of the land finds no path onto the beach")
    }
}
