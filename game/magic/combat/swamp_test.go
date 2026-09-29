package combat

import (
    "image"
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/mod"
)

// a swamp has small pools all over its ground, not where the armies start, and they are drawn in
// the cells they are in
func TestSwampPools(test *testing.T) {
    SwampFlooded = false
    defer func() {
        SwampFlooded = true
    }()

    zone := ZoneType{}
    zone.Ground.SetBiome(mod.BiomeSwamp)
    zone.Ground.Roads = [9]bool{true, true, true, true, true, true, true, true, true}

    for range 5 {
        ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, zone)
        if ground.Pools == nil || len(ground.Pools.Rounds) < 60 {
            rounds := 0
            if ground.Pools != nil {
                rounds = len(ground.Pools.Rounds)
            }
            test.Fatalf("the pools of the swamp have %v rounds", rounds)
        }

        water, onField := 0, 0
        for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
            for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
                part := ground.poolAt(cgx, cgy)
                if part == poolNone {
                    continue
                }
                if armyStartsOf(cgx, cgy, false) || ground.RoadAt(cgx, cgy) != 0 || ground.HeightAt(cgx, cgy) != 0 || ground.sceneryAllowed(cgx, cgy) {
                    test.Fatalf("a pool at %v, %v: army %v, road %v, height %v, trees %v", cgx, cgy, armyStarts(cgx, cgy), ground.RoadAt(cgx, cgy), ground.HeightAt(cgx, cgy), ground.sceneryAllowed(cgx, cgy))
                }
                if part == poolWater {
                    water += 1
                    x, y := CellToTile(cgx, cgy)
                    if x >= 0 && y >= 0 && x < BattlefieldWidth && y < BattlefieldHeight {
                        onField += 1
                    }
                }
            }
        }
        if water < 150 || water > ground.Width * ground.Height / 3 || onField < 20 {
            test.Fatalf("%v cells of water, %v of them on the field", water, onField)
        }

        picture, left, top := makePoolWater(ground)
        if picture == nil {
            test.Fatalf("no water")
        }
        bounds := picture.Bounds()
        drawn := make(map[image.Point]bool)
        for y := range bounds.Dy() {
            for x := range bounds.Dx() {
                pixel := picture.NRGBAAt(x, y)
                if pixel.A == 0 {
                    continue
                }
                cellX, cellY := screenToCell(float64(left + x) + 0.5, float64(top + y) + 0.5)
                cell := image.Pt(int(cellX + 1000) - 1000, int(cellY + 1000) - 1000)
                if ground.poolAt(cell.X, cell.Y) == poolNone {
                    test.Fatalf("water on %v, which has no pool", cell)
                }
                if pixel.G > 0 {
                    drawn[cell] = true
                }
            }
        }
        for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
            for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
                tileX, tileY := CellToTile(cgx, cgy)
                if ground.poolAt(cgx, cgy) == poolWater && borderBrightness(borderDepth(float64(tileX), float64(tileY))) > 0.2 && !drawn[image.Pt(cgx, cgy)] {
                    test.Fatalf("no water is drawn at %v, %v", cgx, cgy)
                }
            }
        }

        // the water costs as rough ground, what a pool only reaches into as the ground
        for index := range ground.Trees {
            ground.Trees[index] = 0
        }
        tiles := makeTiles(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, zone)
        ground.applyTo(tiles)
        model := &CombatModel{Tiles: tiles, Ground: ground}
        for y := range BattlefieldHeight {
            for x := range BattlefieldWidth {
                cgx, cgy := TileToCell(x, y)
                want := moveHalvesGround
                switch {
                    case ground.poolAt(cgx, cgy) == poolWater: want = moveHalvesRough
                    case ground.RoadAt(cgx, cgy) != 0: want = moveHalvesRoad
                }
                if halves := model.cellMoveHalves(x, y, false); halves != want {
                    test.Fatalf("the tile %v, %v costs %v halves, pool %v", x, y, halves, ground.poolAt(cgx, cgy))
                }
            }
        }
    }

    // grass land has none
    ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, ZoneType{})
    if water, _, _ := makePoolWater(ground); ground.Pools != nil || water != nil {
        test.Fatalf("pools on grass land")
    }
}

// mountains and hills have more rocks than the other grounds
func TestRockCounts(test *testing.T) {
    hills := ZoneGround{}
    hills.SetBiome(mod.BiomeHills)

    for range 50 {
        if count := rockCount(CombatLandscapeMountain, ZoneGround{}); count < 13 || count > 24 {
            test.Fatalf("%v rocks on a mountain", count)
        }
        if count := rockCount(CombatLandscapeGrass, hills); count < 9 || count > 16 {
            test.Fatalf("%v rocks on hills", count)
        }
        if count := rockCount(CombatLandscapeGrass, ZoneGround{}); count > 4 {
            test.Fatalf("%v rocks on grass land", count)
        }
    }
}

// a flooded swamp: much raised ground, and all that is low is water, but for the beach of a coast
func TestSwampFlooded(test *testing.T) {
    zone := ZoneType{}
    zone.Ground.SetBiome(mod.BiomeSwamp)
    zone.Ground.Coast = CoastNorth
    zone.Ground.Roads = [9]bool{true, true, true, true, true, true, true, true, true}

    raised, low := 0, 0
    for range 5 {
        ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, zone)
        if ground.Pools == nil || !ground.Pools.Flooded || len(ground.Pools.Rounds) != 0 {
            test.Fatalf("the swamp is not flooded")
        }

        for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
            for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
                water := ground.poolAt(cgx, cgy) == poolWater
                dry := ground.coastAt(cgx, cgy) != coastLand
                switch {
                    case dry:
                        if water {
                            test.Fatalf("the swamp floods the coast at %v, %v", cgx, cgy)
                        }
                    case ground.HeightAt(cgx, cgy) > 0:
                        raised += 1
                        if water {
                            test.Fatalf("water on a plateau at %v, %v", cgx, cgy)
                        }
                    default:
                        low += 1
                        if !water || ground.sceneryAllowed(cgx, cgy) {
                            test.Fatalf("low ground at %v, %v: water %v, trees %v", cgx, cgy, water, ground.sceneryAllowed(cgx, cgy))
                        }
                }
            }
        }

        picture, left, top := makePoolWater(ground)
        if picture == nil {
            test.Fatalf("no water")
        }
        // the middle of a cell of water has water, the middle of a cell inside of a plateau none
        for cgy := ground.MinY + 1; cgy < ground.MinY + ground.Height - 1; cgy++ {
            for cgx := ground.MinX + 1; cgx < ground.MinX + ground.Width - 1; cgx++ {
                tileX, tileY := CellToTile(cgx, cgy)
                if ground.coastAt(cgx, cgy) != coastLand || borderBrightness(borderDepth(float64(tileX), float64(tileY))) <= 0.2 {
                    continue
                }
                x, y := cellCenterScreen(cgx, cgy)
                drawn := picture.NRGBAAt(x - left, y - top).A > 0
                neighbors := ground.raisedNeighbors(cgx, cgy)
                if ground.HeightAt(cgx, cgy) == 0 && neighbors == 0 && !drawn {
                    test.Fatalf("no water is drawn at %v, %v", cgx, cgy)
                }
                if ground.HeightAt(cgx, cgy) > 0 && neighbors == 8 && drawn {
                    test.Fatalf("water is drawn on the plateau at %v, %v", cgx, cgy)
                }
            }
        }

        // a road through the water costs what a road costs
        tiles := makeTiles(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, zone)
        ground.applyTo(tiles)
        model := &CombatModel{Tiles: tiles, Ground: ground}
        for y := range BattlefieldHeight {
            for x := range BattlefieldWidth {
                cgx, cgy := TileToCell(x, y)
                if ground.poolAt(cgx, cgy) != poolWater {
                    continue
                }
                want := moveHalvesRough
                if ground.RoadAt(cgx, cgy) != 0 {
                    want = moveHalvesRoad
                }
                if halves := model.cellMoveHalves(x, y, false); halves != want {
                    test.Fatalf("the water at %v, %v costs %v halves", x, y, halves)
                }
            }
        }
    }

    // between a third and two thirds of the swamp are islands
    if raised * 3 < raised + low || raised * 3 > (raised + low) * 2 {
        test.Fatalf("%v cells of raised ground, %v of low ground", raised, low)
    }
}
