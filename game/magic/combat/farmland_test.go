package combat

import (
    "testing"

    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    "github.com/kazzmir/master-of-magic/game/magic/mod"
)

// where a battle is fought on the fields of a town, and that the fields take the place of the biome
func TestFarmGround(test *testing.T) {
    zone := ZoneType{}
    zone.Ground.SetBiome(mod.BiomeForest)
    zone.Ground.Farmland = FarmlandNear

    ground := farmGround(CombatLandscapeGrass, zone)
    if ground.Farmland != FarmlandNear || ground.Biome != "" || ground.Forest {
        test.Fatalf("fields in a forest: %+v", ground)
    }

    for _, landscape := range []CombatLandscape{CombatLandscapeDesert, CombatLandscapeMountain, CombatLandscapeTundra} {
        plain := ZoneType{}
        plain.Ground.Farmland = FarmlandNear
        if ground := farmGround(landscape, plain); ground.Farmland != FarmlandNone {
            test.Fatalf("fields on landscape %v", landscape)
        }
    }

    volcano := zone
    volcano.Ground.SetBiome(mod.BiomeVolcano)
    if ground := farmGround(CombatLandscapeGrass, volcano); ground.Farmland != FarmlandNone || ground.Biome != mod.BiomeVolcano {
        test.Fatalf("fields on a volcano: %+v", ground)
    }

    town := zone
    town.City = &citylib.City{}
    if ground := farmGround(CombatLandscapeGrass, town); ground.Farmland != FarmlandNone || ground.Biome != mod.BiomeForest {
        test.Fatalf("fields in a town: %+v", ground)
    }

    lair := zone
    lair.Encounter = ZoneLair
    if ground := farmGround(CombatLandscapeGrass, lair); ground.Farmland != FarmlandNone {
        test.Fatalf("fields at a lair: %+v", ground)
    }
}

// the plots are squares of one size with a row of grass between them, flat, without trees, and
// not on roads, the coast or the river
func TestFarmland(test *testing.T) {
    for range 20 {
        zone := ZoneType{}
        zone.Ground.Farmland = FarmlandFar
        zone.Ground.Coast = CoastNorth
        zone.Ground.Roads = [9]bool{true, true, true, true, true, true, true, true, true}
        ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, zone)

        if ground.Farmland != FarmlandFar {
            test.Fatalf("the ground has farmland %v", ground.Farmland)
        }

        crops := 0
        for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
            for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
                if ground.HeightAt(cgx, cgy) != 0 {
                    test.Fatalf("a plateau in the fields at %v, %v", cgx, cgy)
                }
                if ground.GroupAt(cgx, cgy) != TerrainCrop {
                    continue
                }
                crops += 1

                picture := ground.Picture[ground.index(cgx, cgy)]
                if groundVariant(picture, groundCropFirst, groundCropCount) < 0 {
                    test.Fatalf("picture %v on crops", picture)
                }
                if ground.RoadAt(cgx, cgy) != 0 || ground.coastAt(cgx, cgy) != coastLand || ground.sceneryAllowed(cgx, cgy) {
                    test.Fatalf("crops at %v, %v: road %v, coast %v, trees %v", cgx, cgy, ground.RoadAt(cgx, cgy), ground.coastAt(cgx, cgy), ground.sceneryAllowed(cgx, cgy))
                }

                // the plot the cell is of: as wide as it is high, of one picture, with no crops
                // around it
                left, right, top, bottom := cgx, cgx, cgy, cgy
                for ground.contains(left - 1, cgy) && ground.GroupAt(left - 1, cgy) == TerrainCrop {
                    left -= 1
                }
                for ground.contains(right + 1, cgy) && ground.GroupAt(right + 1, cgy) == TerrainCrop {
                    right += 1
                }
                for ground.contains(cgx, top - 1) && ground.GroupAt(cgx, top - 1) == TerrainCrop {
                    top -= 1
                }
                for ground.contains(cgx, bottom + 1) && ground.GroupAt(cgx, bottom + 1) == TerrainCrop {
                    bottom += 1
                }

                width := right - left + 1
                height := bottom - top + 1
                if width > farmPlotMax || height > farmPlotMax {
                    test.Fatalf("a plot of %v by %v at %v, %v", width, height, cgx, cgy)
                }
                // cut off by the edge of the ground it can be less than a square
                inside := left > ground.MinX && top > ground.MinY && right < ground.MinX + ground.Width - 1 && bottom < ground.MinY + ground.Height - 1
                if inside && (width != height || width < farmPlotSmall) {
                    test.Fatalf("a plot of %v by %v at %v, %v", width, height, cgx, cgy)
                }
                for y := top - 1; y <= bottom + 1; y++ {
                    for x := left - 1; x <= right + 1; x++ {
                        if !ground.contains(x, y) {
                            continue
                        }
                        around := x < left || x > right || y < top || y > bottom
                        group := ground.GroupAt(x, y)
                        if around && group != TerrainGrass && group != TerrainSand && group != TerrainWater {
                            test.Fatalf("%v beside the plot at %v, %v", group, x, y)
                        }
                        if !around && ground.Picture[ground.index(x, y)] != picture {
                            test.Fatalf("a plot of more than one picture at %v, %v", x, y)
                        }
                    }
                }
            }
        }
        if crops == 0 {
            test.Fatalf("no crops")
        }
    }

    // no town, no fields
    ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, ZoneType{})
    for _, group := range ground.Group {
        if group == TerrainCrop || ground.Crops != nil {
            test.Fatalf("crops without a town")
        }
    }
}

// houses stand on the grass, one to a cell, not where the armies start, and beside the town
// there are more of them
func TestFarmHouses(test *testing.T) {
    counts := make(map[int]int)
    for _, away := range []int{FarmlandNear, FarmlandFar} {
        for range 10 {
            zone := ZoneType{}
            zone.Ground.Farmland = away
            ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, zone)
            houses := makeFarmHouses(BattlefieldWidth, BattlefieldHeight, zone, ground)
            counts[away] += len(houses)

            built := 0
            for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
                for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
                    if !ground.builtAt(cgx, cgy) {
                        continue
                    }
                    built += 1
                    if ground.GroupAt(cgx, cgy) != TerrainGrass || armyStarts(cgx, cgy) || ground.sceneryAllowed(cgx, cgy) {
                        test.Fatalf("a house on %v at %v, %v, where an army starts: %v", ground.GroupAt(cgx, cgy), cgx, cgy, armyStarts(cgx, cgy))
                    }
                }
            }
            if built != len(houses) || built == 0 {
                test.Fatalf("%v houses in %v cells", len(houses), built)
            }
            for _, house := range houses {
                if house.Kind != SceneryHouse {
                    test.Fatalf("a house of kind %v", house.Kind)
                }
            }
        }
    }

    if counts[FarmlandNear] < counts[FarmlandFar] * 2 {
        test.Fatalf("%v houses beside the town, %v two tiles away", counts[FarmlandNear], counts[FarmlandFar])
    }

    ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, ZoneType{})
    if houses := makeFarmHouses(BattlefieldWidth, BattlefieldHeight, ZoneType{}, ground); len(houses) != 0 {
        test.Fatalf("houses without a town")
    }
}
