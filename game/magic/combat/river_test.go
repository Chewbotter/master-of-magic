package combat

import (
    "image"
    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    "testing"
)

// a river of both courses: every column across it has a bank, 2 to 3 cells of water and a bank,
// the banks of columns side by side touch, the places of the armies are no water, and the
// battlefield has no roads
func TestRiver(test *testing.T) {
    for _, course := range []RiverCourse{RiverAcross, RiverBeside} {
        for _, large := range []bool{false, true} {
            for range 20 {
                zone := ZoneType{}
                zone.Ground.River = course
                zone.Ground.LargeArmy = large
                zone.Ground.Roads = [9]bool{true, true, true, true, true, true, true, true, true}
                ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeMountain, 0, zone)

                if ground.River != course || ground.Stream == nil {
                    test.Fatalf("river %v: the ground has %v", course, ground.River)
                }

                units := deployPlaces
                if large {
                    units = deployPlaces * 3
                }
                for _, team := range []Team{TeamDefender, TeamAttacker} {
                    for _, cell := range deployCells(team, units, false) {
                        if ground.riverAt(cell.X, cell.Y) == riverWater {
                            test.Fatalf("river %v, large %v: a place of an army at %v is water", course, large, cell)
                        }
                    }
                }

                stream := ground.Stream
                for along := range stream.Starts {
                    width := stream.Waters[along] - stream.Starts[along] - 1
                    if width < riverWidthMin || width > riverWidthMax {
                        test.Fatalf("river %v: column %v has %v cells of water", course, along, width)
                    }
                    if along == 0 {
                        continue
                    }
                    stepStart := stream.Starts[along] - stream.Starts[along - 1]
                    stepWater := stream.Waters[along] - stream.Waters[along - 1]
                    if stepStart < -1 || stepStart > 1 || stepWater < -1 || stepWater > 1 {
                        test.Fatalf("river %v: column %v moves by %v and %v", course, along, stepStart, stepWater)
                    }
                }

                water, banks := 0, 0
                for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
                    for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
                        if ground.RoadAt(cgx, cgy) != 0 {
                            test.Fatalf("river %v: a road at %v, %v", course, cgx, cgy)
                        }

                        part := ground.riverAt(cgx, cgy)
                        if part == riverLand {
                            continue
                        }
                        if part == riverWater {
                            water += 1
                        } else {
                            banks += 1
                        }

                        // the ground under the water is the one of the landscape
                        group := ground.GroupAt(cgx, cgy)
                        if group != TerrainGrass && group != TerrainDirt {
                            test.Fatalf("river %v: %v under the river at %v, %v", course, group, cgx, cgy)
                        }
                        if ground.HeightAt(cgx, cgy) != 0 {
                            test.Fatalf("river %v: a plateau in the river at %v, %v", course, cgx, cgy)
                        }
                        if ground.sceneryAllowed(cgx, cgy) {
                            test.Fatalf("river %v: trees can stand in the river at %v, %v", course, cgx, cgy)
                        }
                    }
                }
                if water == 0 || banks == 0 {
                    test.Fatalf("river %v: %v cells of water, %v banks", course, water, banks)
                }
            }
        }
    }

    // any: one of the two. none in a town, which keeps its roads
    zone := ZoneType{}
    zone.Ground.River = RiverAny
    seen := make(map[RiverCourse]bool)
    for range 40 {
        seen[makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, zone).River] = true
    }
    if len(seen) != 2 || seen[RiverNone] || seen[RiverAny] {
        test.Fatalf("any river: %v", seen)
    }

    zone.City = &citylib.City{}
    if ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, zone); ground.River != RiverNone || ground.Banks != nil {
        test.Fatalf("a river in a town")
    }

    // with a coast the river ends at the sea
    zone = ZoneType{}
    zone.Ground.River = RiverAcross
    zone.Ground.Coast = CoastNorth
    ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, zone)
    for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
        for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
            if ground.GroupAt(cgx, cgy) == TerrainWater && ground.riverAt(cgx, cgy) != riverLand {
                test.Fatalf("the river runs on in the sea at %v, %v", cgx, cgy)
            }
        }
    }
}

// the water of a river costs twice what grass costs and every unit can enter it, a bank costs
// what the ground costs
func TestRiverRules(test *testing.T) {
    zone := ZoneType{}
    zone.Ground.River = RiverAcross
    ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, zone)
    for index := range ground.Trees {
        ground.Trees[index] = 0
    }
    tiles := makeTiles(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, zone)
    ground.applyTo(tiles)
    model := &CombatModel{Tiles: tiles, Ground: ground}

    // from the front row of the defenders to the one of the attackers
    cgy := deployColumns[0]
    var bank, water image.Point
    found := 0
    for cgx := deployDefenderRows[0]; cgx <= deployAttackerRows[0]; cgx++ {
        x, y := CellToTile(cgx, cgy)
        switch ground.riverAt(cgx, cgy) {
            case riverNearBank:
                bank = image.Pt(x, y)
                found |= 1
            case riverWater:
                water = image.Pt(x, y)
                found |= 2
        }
    }
    if found != 3 {
        test.Fatalf("no bank and water between the armies: %v", found)
    }

    if halves := model.cellMoveHalves(water.X, water.Y, false); halves != moveHalvesRough {
        test.Fatalf("the river costs %v halves", halves)
    }
    if halves := model.cellMoveHalves(bank.X, bank.Y, false); halves != moveHalvesGround {
        test.Fatalf("the bank costs %v halves", halves)
    }
    if halves := model.cellMoveHalves(water.X, water.Y, true); halves != moveHalvesGround {
        test.Fatalf("the river costs a flying unit %v halves", halves)
    }

    fromX, fromY := CellToTile(deployDefenderRows[0], cgy)
    toX, toY := CellToTile(deployAttackerRows[0], cgy)
    if _, ok := model.computePath(fromX, fromY, toX, toY, false, false, false); !ok {
        test.Fatalf("a unit of the land finds no path across the river")
    }
}

// the water that is drawn: on all of a cell of the river, on a part of every bank, nowhere else
func TestRiverWater(test *testing.T) {
    zone := ZoneType{}
    zone.Ground.River = RiverAcross
    ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, zone)

    water, left, top := makeRiverWater(ground)
    if water == nil {
        test.Fatalf("no water")
    }

    // pixels of water and of wet ground of every cell
    wet := make(map[image.Point]int)
    dry := make(map[image.Point]int)
    bounds := water.Bounds()
    for y := range bounds.Dy() {
        for x := range bounds.Dx() {
            cellX, cellY := screenToCell(float64(left + x) + 0.5, float64(top + y) + 0.5)
            cell := image.Pt(int(cellX + 1000) - 1000, int(cellY + 1000) - 1000)
            pixel := water.NRGBAAt(x, y)
            switch {
                case pixel.A == 0: dry[cell] += 1
                case ground.riverAt(cell.X, cell.Y) == riverLand: test.Fatalf("water on %v, which is land", cell)
                case pixel.B > 0: wet[cell] += 1
                default: dry[cell] += 1
            }
        }
    }

    for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
        for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
            tileX, tileY := CellToTile(cgx, cgy)
            if borderBrightness(borderDepth(float64(tileX), float64(tileY))) <= 0.2 {
                continue
            }
            cell := image.Pt(cgx, cgy)
            switch ground.riverAt(cgx, cgy) {
                case riverWater:
                    if dry[cell] > 0 || wet[cell] == 0 {
                        test.Fatalf("the river at %v has %v pixels of water and %v without", cell, wet[cell], dry[cell])
                    }
                case riverNearBank, riverFarBank:
                    if dry[cell] == 0 || wet[cell] == 0 {
                        test.Fatalf("the bank at %v has %v pixels of water and %v without", cell, wet[cell], dry[cell])
                    }
            }
        }
    }

    plain := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, ZoneType{})
    if water, _, _ := makeRiverWater(plain); water != nil {
        test.Fatalf("water without a river")
    }
}
