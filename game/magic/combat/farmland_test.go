package combat

import (
    "testing"

    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    "github.com/kazzmir/master-of-magic/game/magic/mod"
)

// where a battle has farmland, and where the farmland has crops
func TestFarmGround(test *testing.T) {
    zone := ZoneType{}
    zone.Ground.SetBiome(mod.BiomeForest)
    zone.Ground.Farmland = FarmlandNear

    // the biome stays, and has no crops
    ground := farmGround(CombatLandscapeGrass, zone)
    if ground.Farmland != FarmlandNear || ground.Biome != mod.BiomeForest || !ground.Forest {
        test.Fatalf("farmland in a forest: %+v", ground)
    }
    if farmCrops(CombatLandscapeGrass, zone) {
        test.Fatalf("crops in a forest")
    }

    plain := ZoneType{}
    plain.Ground.Farmland = FarmlandFar
    if !farmCrops(CombatLandscapeGrass, plain) {
        test.Fatalf("no crops on grass land")
    }
    for _, landscape := range []CombatLandscape{CombatLandscapeDesert, CombatLandscapeMountain, CombatLandscapeTundra} {
        if ground := farmGround(landscape, plain); ground.Farmland != FarmlandFar || farmCrops(landscape, plain) {
            test.Fatalf("landscape %v: farmland %v, crops %v", landscape, ground.Farmland, farmCrops(landscape, plain))
        }
        made := makeBattleGround(BattlefieldWidth, BattlefieldHeight, landscape, 0, plain)
        if made.Farmland != FarmlandFar || made.Crops != nil || len(makeFarmHouses(BattlefieldWidth, BattlefieldHeight, plain, made)) == 0 {
            test.Fatalf("landscape %v: the ground has farmland %v, crops %v", landscape, made.Farmland, made.Crops != nil)
        }
    }
    if ground := farmGround(CombatLandscapeWater, plain); ground.Farmland != FarmlandNone {
        test.Fatalf("farmland on the sea")
    }

    town := zone
    town.City = &citylib.City{}
    if ground := farmGround(CombatLandscapeGrass, town); ground.Farmland != FarmlandNone || ground.Biome != mod.BiomeForest {
        test.Fatalf("farmland in a town: %+v", ground)
    }

    lair := zone
    lair.Encounter = ZoneLair
    if ground := farmGround(CombatLandscapeGrass, lair); ground.Farmland != FarmlandNone {
        test.Fatalf("farmland at a lair: %+v", ground)
    }

    // about half of the ground of plain grass land is crops, and of the rest more is dirt than grass
    crops, dirt, grass := 0, 0, 0
    for range 10 {
        made := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, plain)
        for _, group := range made.Group {
            switch group {
                case TerrainCrop: crops += 1
                case TerrainDirt: dirt += 1
                case TerrainGrass: grass += 1
            }
        }
    }
    if crops * 5 < (crops + dirt + grass) * 2 || crops * 5 > (crops + dirt + grass) * 3 || dirt <= grass {
        test.Fatalf("%v cells of crops, %v of dirt, %v of grass", crops, dirt, grass)
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
                        if around && group != TerrainGrass && group != TerrainDirt && group != TerrainSand && group != TerrainWater {
                            test.Fatalf("%v beside the plot at %v, %v", group, x, y)
                        }
                        if !around && cropKind(ground.Picture[ground.index(x, y)]) != cropKind(picture) {
                            test.Fatalf("a plot of more than one kind of crops at %v, %v", x, y)
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
                    if ground.GroupAt(cgx, cgy) == TerrainCrop || armyStarts(cgx, cgy) || ground.sceneryAllowed(cgx, cgy) {
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

// fences stand on sides of cells, once on a side, not where the armies start and not across a
// road, and most of them along a plot, a road or a house
func TestFences(test *testing.T) {
    if cellSide(3, 4, 2) != cellSide(3, 5, 0) || cellSide(3, 4, 1) != cellSide(4, 4, 3) {
        test.Fatalf("the sides of cells side by side")
    }
    // the NE side runs down to the right on the screen, the NW side up to the right
    x, y := cellCenterScreen(3, 4)
    if neX, neY := (fencePlace{Cgx: 3, Cgy: 4, Side: fenceNE}).screen(); neX != x + 8 || neY != y - 4 {
        test.Fatalf("the NE side is at %v, %v", neX, neY)
    }
    if nwX, nwY := (fencePlace{Cgx: 3, Cgy: 4, Side: fenceNW}).screen(); nwX != x - 8 || nwY != y - 4 {
        test.Fatalf("the NW side is at %v, %v", nwX, nwY)
    }

    total := 0
    for range 10 {
        zone := ZoneType{}
        zone.Ground.Farmland = FarmlandNear
        zone.Ground.Roads = [9]bool{false, true, false, false, true, false, false, true, false}
        ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, zone)
        makeFarmHouses(BattlefieldWidth, BattlefieldHeight, zone, ground)
        if len(ground.Plots) == 0 {
            test.Fatalf("no plots")
        }

        seen := make(map[fencePlace]bool)
        for _, place := range ground.fencePlaces(zone) {
            if seen[place] {
                test.Fatalf("two fences at %+v", place)
            }
            seen[place] = true
            total += 1

            x1, y1, x2, y2 := place.cells()
            if armyStartsOf(x1, y1, false) || armyStartsOf(x2, y2, false) {
                test.Fatalf("a fence where an army starts, at %+v", place)
            }
            if ground.RoadAt(x1, y1) != 0 && ground.RoadAt(x2, y2) != 0 {
                test.Fatalf("a fence across a road at %+v", place)
            }

            crop := ground.GroupAt(x1, y1) == TerrainCrop || ground.GroupAt(x2, y2) == TerrainCrop
            road := ground.RoadAt(x1, y1) != 0 || ground.RoadAt(x2, y2) != 0
            house := ground.builtAt(x1, y1) || ground.builtAt(x2, y2)
            if !crop && !road && !house {
                test.Fatalf("a fence at %+v with no plot, road or house beside it", place)
            }
        }
    }
    if total < 100 {
        test.Fatalf("%v fences", total)
    }

    // no farmland, no fences
    ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, ZoneType{})
    if len(makeFences(ZoneType{}, ground)) != 0 {
        test.Fatalf("fences without farmland")
    }
}
