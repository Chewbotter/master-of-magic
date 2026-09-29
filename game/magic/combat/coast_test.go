package combat

import (
    "image"
    "image/color"
    "testing"
)

// a coast on every side: the places of the armies are land, and every column along the coast has
// land, one transition, 1 to 5 cells of beach, one transition and water, with the transitions of
// columns side by side touching
func TestCoast(test *testing.T) {
    for _, side := range []CoastSide{CoastEast, CoastNorth, CoastWest, CoastSouth} {
        for _, large := range []bool{false, true} {
            zone := ZoneType{}
            zone.Ground.Coast = side
            zone.Ground.LargeArmy = large
            zone.Ground.Roads = [9]bool{true, true, true, true, true, true, true, true, true}
            ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeMountain, 0, zone)

            if ground.Coast != side || ground.Lines == nil {
                test.Fatalf("coast %v: the ground has %v", side, ground.Coast)
            }

            units := deployPlaces
            if large {
                units = deployPlaces * 3
            }
            for _, team := range []Team{TeamDefender, TeamAttacker} {
                for _, cell := range deployCells(team, units, false) {
                    group := ground.GroupAt(cell.X, cell.Y)
                    if group == TerrainSand || group == TerrainWater || ground.shoreAt(cell.X, cell.Y) {
                        test.Fatalf("coast %v, large %v: a place of an army at %v is %v, shore %v", side, large, cell, group, ground.shoreAt(cell.X, cell.Y))
                    }
                }
            }

            coast := ground.Lines
            for along := range coast.Starts {
                beach := coast.Waters[along] - coast.Starts[along] - 1
                if coast.Starts[along] < 0 || coast.Starts[along] > coastWander || beach < coastNarrowMin || beach > coastBeachMax {
                    test.Fatalf("coast %v: column %v starts at %v with %v cells of beach", side, along, coast.Starts[along], beach)
                }
                if along == 0 {
                    continue
                }
                stepStart := coast.Starts[along] - coast.Starts[along - 1]
                stepWater := coast.Waters[along] - coast.Waters[along - 1]
                if stepStart < -1 || stepStart > 1 || stepWater < -1 || stepWater > 1 {
                    test.Fatalf("coast %v: column %v moves by %v and %v", side, along, stepStart, stepWater)
                }
            }

            sand, water, edges := 0, 0, 0
            for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
                for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
                    group := ground.GroupAt(cgx, cgy)
                    part := ground.coastAt(cgx, cgy)

                    // what a cell counts as
                    switch part {
                        case coastLand, coastLandEdge:
                            if group == TerrainSand || group == TerrainWater {
                                test.Fatalf("coast %v: %v counts as %v at %v, %v", side, part, group, cgx, cgy)
                            }
                        case coastBeach:
                            sand += 1
                            if group != TerrainSand {
                                test.Fatalf("coast %v: the beach counts as %v at %v, %v", side, group, cgx, cgy)
                            }
                        case coastWaterEdge, coastWater:
                            water += 1
                            if group != TerrainWater {
                                test.Fatalf("coast %v: %v counts as %v at %v, %v", side, part, group, cgx, cgy)
                            }
                    }
                    if ground.shoreAt(cgx, cgy) {
                        edges += 1
                    }

                    // the edge of the sand stays inside of a transition
                    if ground.shoreAt(cgx, cgy) {
                        _, along := coast.cellDepth(cgx, cgy)
                        values := coast.Starts
                        if part == coastWaterEdge {
                            values = coast.Waters
                        }
                        for step := 0; step <= 10; step++ {
                            edge := coast.edgeIn(values, float64(along) + float64(step) / 10 * 0.999)
                            if edge < 0 || edge > 1 {
                                test.Fatalf("coast %v: the edge of the sand leaves the cell %v, %v: %v", side, cgx, cgy, edge)
                            }
                        }
                    }

                    if part != coastLand {
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
                }
            }
            if sand == 0 || water == 0 || edges == 0 {
                test.Fatalf("coast %v: %v cells of beach, %v of water, %v transitions", side, sand, water, edges)
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

    // any: one of the four
    zone := ZoneType{}
    zone.Ground.Coast = CoastAny
    seen := make(map[CoastSide]bool)
    for range 60 {
        seen[makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, zone).Coast] = true
    }
    if len(seen) != 4 || seen[CoastNone] || seen[CoastAny] {
        test.Fatalf("any coast: %v", seen)
    }

    // no coast, no beach
    ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, ZoneType{})
    for _, group := range ground.Group {
        if group == TerrainSand || group == TerrainWater || ground.Shore != nil {
            test.Fatalf("%v without a coast", group)
        }
    }
}

// the beach costs twice what grass costs, a transition what is underneath, and the water is for
// what can enter it
func TestCoastRules(test *testing.T) {
    zone := ZoneType{}
    zone.Ground.Coast = CoastEast
    ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, zone)
    // without trees, which cost as well
    for index := range ground.Trees {
        ground.Trees[index] = 0
    }
    tiles := makeTiles(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, zone)
    ground.applyTo(tiles)
    model := &CombatModel{Tiles: tiles, Ground: ground}

    // along a row of the grid toward the east: land, the transition, beach, water
    _, lastX, firstY, _ := armyPlaces(false)
    var land, edge, beach, sea image.Point
    found := 0
    for cgx := lastX; cgx < ground.MinX + ground.Width && found < 3; cgx++ {
        x, y := CellToTile(cgx, firstY)
        if y >= BattlefieldHeight {
            break
        }
        group := ground.GroupAt(cgx, firstY)
        switch {
            case found == 0 && ground.shoreAt(cgx, firstY) && group != TerrainWater:
                landX, landY := CellToTile(cgx - 1, firstY)
                land = image.Pt(landX, landY)
                edge = image.Pt(x, y)
                found = 1
            case found == 1 && group == TerrainSand:
                beach = image.Pt(x, y)
                found = 2
            case found == 2 && group == TerrainWater:
                sea = image.Pt(x, y)
                found = 3
        }
    }
    if found != 3 {
        test.Fatalf("no transition, beach and water on the field toward the east: %v", found)
    }

    if !model.IsWaterTile(sea.X, sea.Y) || model.IsWaterTile(beach.X, beach.Y) || model.IsWaterTile(edge.X, edge.Y) {
        test.Fatalf("water: sea %v, beach %v, transition %v", model.IsWaterTile(sea.X, sea.Y), model.IsWaterTile(beach.X, beach.Y), model.IsWaterTile(edge.X, edge.Y))
    }

    if halves := model.cellMoveHalves(beach.X, beach.Y, false); halves != moveHalvesRough {
        test.Fatalf("the beach costs %v halves", halves)
    }
    if halves := model.cellMoveHalves(edge.X, edge.Y, false); halves != moveHalvesGround {
        test.Fatalf("the transition costs %v halves", halves)
    }
    if halves := model.cellMoveHalves(beach.X, beach.Y, true); halves != moveHalvesGround {
        test.Fatalf("the beach costs a flying unit %v halves", halves)
    }

    // a path into the water: not for a unit of the land
    if _, ok := model.computePath(land.X, land.Y, sea.X, sea.Y, false, false, false); ok {
        test.Fatalf("a unit of the land finds a path into the water")
    }
    if _, ok := model.computePath(land.X, land.Y, sea.X, sea.Y, false, false, true); !ok {
        test.Fatalf("a unit that swims finds no path into the water")
    }
    if _, ok := model.computePath(land.X, land.Y, beach.X, beach.Y, false, false, false); !ok {
        test.Fatalf("a unit of the land finds no path onto the beach")
    }
}

// the sand over the transitions: on a part of every one of them, in the color of the tiles of sand,
// and nowhere else
func TestShoreSand(test *testing.T) {
    zone := ZoneType{}
    zone.Ground.Coast = CoastNorth
    ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, zone)

    // a tile of sand of one color, see-through around its diamond
    sandColor := color.NRGBA{R: 200, G: 180, B: 100, A: 255}
    tile := image.NewNRGBA(image.Rect(0, 0, shoreTileWidth, shoreTileHeight))
    for y := range shoreTileHeight {
        for x := range shoreTileWidth {
            a := (float64(x) + 0.5 - shoreTileMiddleX) / CellStepX
            b := (float64(y) + 0.5 - shoreTileMiddleY) / CellStepY
            if max(a, -a) + max(b, -b) <= 1 {
                tile.SetNRGBA(x, y, sandColor)
            }
        }
    }

    sand, left, top := makeShoreSand(ground, []image.Image{tile})
    if sand == nil {
        test.Fatalf("no sand")
    }

    covered := make(map[image.Point]int)
    bounds := sand.Bounds()
    for y := range bounds.Dy() {
        for x := range bounds.Dx() {
            pixel := sand.NRGBAAt(x, y)
            if pixel.A == 0 {
                continue
            }
            if pixel.A != 255 {
                test.Fatalf("a pixel of sand that is half there: %v", pixel)
            }

            cellX, cellY := screenToCell(float64(left + x) + 0.5, float64(top + y) + 0.5)
            cell := image.Pt(int(cellX + 1000) - 1000, int(cellY + 1000) - 1000)
            if !ground.shoreAt(cell.X, cell.Y) {
                test.Fatalf("sand on %v, which is no transition", cell)
            }
            covered[cell] += 1

            // on the field it has the color of the tile, around it it is darker
            tileX, tileY := CellToTile(cell.X, cell.Y)
            if borderDepth(float64(tileX), float64(tileY)) == 0 && pixel != sandColor {
                test.Fatalf("sand of color %v", pixel)
            }
        }
    }

    // a tile has 256 pixels. a transition has sand on a part of them, not on none and not on all
    part, whole := 0, 0
    for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
        for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
            tileX, tileY := CellToTile(cgx, cgy)
            if !ground.shoreAt(cgx, cgy) || borderBrightness(borderDepth(float64(tileX), float64(tileY))) <= 0 {
                continue
            }
            pixels := covered[image.Pt(cgx, cgy)]
            if pixels > 0 && pixels < 250 {
                part += 1
            } else {
                whole += 1
            }
        }
    }
    if part == 0 || whole > 0 {
        test.Fatalf("%v transitions have sand on a part of them, %v on none or on all", part, whole)
    }

    // no coast, no sand
    plain := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, ZoneType{})
    if sand, _, _ := makeShoreSand(plain, []image.Image{tile}); sand != nil {
        test.Fatalf("sand without a coast")
    }
}
