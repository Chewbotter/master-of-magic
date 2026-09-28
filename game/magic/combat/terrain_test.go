package combat

import (
    "image"
    "image/color"
    "testing"

    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
)

func testGround(width int, height int) *BattleGround {
    return &BattleGround{
        Width: width,
        Height: height,
        Group: make([]TerrainGroup, width * height),
        Picture: make([]int, width * height),
        Roads: make([]int, width * height),
        Trees: make([]int, width * height),
    }
}

// grass shows the edge of the dirt next to it, as Set_Terrain_Tile_Types picks it
func TestGrassShowsTheEdgeOfDirt(test *testing.T) {
    cases := []struct {
        Name string
        DirtX []int
        DirtY []int
        First int
    }{
        {Name: "none", First: 0},
        {Name: "right", DirtX: []int{3}, DirtY: []int{2}, First: 30},
        {Name: "right and up", DirtX: []int{3, 2}, DirtY: []int{2, 1}, First: 28},
        {Name: "right and down", DirtX: []int{3, 2}, DirtY: []int{2, 3}, First: 12},
        {Name: "up", DirtX: []int{2}, DirtY: []int{1}, First: 24},
        {Name: "up and left", DirtX: []int{2, 1}, DirtY: []int{1, 2}, First: 22},
        {Name: "left", DirtX: []int{1}, DirtY: []int{2}, First: 18},
        {Name: "left and down", DirtX: []int{1, 2}, DirtY: []int{2, 3}, First: 16},
        {Name: "down", DirtX: []int{2}, DirtY: []int{3}, First: 10},
        {Name: "down right corner", DirtX: []int{3}, DirtY: []int{3}, First: 8},
        {Name: "up right corner", DirtX: []int{3}, DirtY: []int{1}, First: 26},
        {Name: "up left corner", DirtX: []int{1}, DirtY: []int{1}, First: 20},
        {Name: "down left corner", DirtX: []int{1}, DirtY: []int{3}, First: 14},
    }

    for _, testCase := range cases {
        ground := testGround(5, 5)
        for index := range testCase.DirtX {
            ground.setGroup(testCase.DirtX[index], testCase.DirtY[index], TerrainDirt)
        }

        last := testCase.First + 1
        if testCase.Name == "none" {
            last = 3
        }
        for range 20 {
            picture := ground.pictureOf(2, 2)
            if picture < testCase.First || picture > last {
                test.Fatalf("%v: picture %v, want %v to %v", testCase.Name, picture, testCase.First, last)
            }
        }
    }
}

// rough ground shows which of its sides go on as rough: the pictures 32 to 47
func TestRoughFollowsItsNeighbors(test *testing.T) {
    ground := testGround(5, 5)
    ground.setGroup(2, 2, TerrainRough)
    if picture := ground.pictureOf(2, 2); picture != 47 {
        test.Fatalf("single rough: %v, want 47", picture)
    }

    ground.setGroup(2, 3, TerrainRough)
    if picture := ground.pictureOf(2, 2); picture != 32 {
        test.Fatalf("rough below: %v, want 32", picture)
    }

    ground.setGroup(2, 1, TerrainRough)
    ground.setGroup(1, 2, TerrainRough)
    ground.setGroup(3, 2, TerrainRough)
    if picture := ground.pictureOf(2, 2); picture != 36 {
        test.Fatalf("rough on all sides: %v, want 36", picture)
    }
}

// no rough next to dirt, on a road or in a town, and a road runs from the town to the edge
func TestGroundKeepsTheOriginalsRules(test *testing.T) {
    zone := ZoneType{City: &citylib.City{}}
    zone.Ground.Roads[4] = true
    zone.Ground.Roads[5] = true

    for range 20 {
        ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeMountain, 0, zone)

        reachesEdge := false
        for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
            for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
                picture := ground.Picture[ground.index(cgx, cgy)]
                if picture < 0 || picture > 47 {
                    test.Fatalf("picture %v at %v, %v", picture, cgx, cgy)
                }

                if ground.GroupAt(cgx, cgy) != TerrainRough {
                    continue
                }
                if ground.RoadAt(cgx, cgy) != 0 {
                    test.Fatalf("rough on a road at %v, %v", cgx, cgy)
                }
                if cgx >= townCellMinX && cgx <= townCellMaxX && cgy >= townCellMinY && cgy <= townCellMaxY {
                    test.Fatalf("rough in the town at %v, %v", cgx, cgy)
                }
                for dy := -1; dy <= 1; dy++ {
                    for dx := -1; dx <= 1; dx++ {
                        if ground.GroupAt(cgx + dx, cgy + dy) == TerrainDirt {
                            test.Fatalf("rough next to dirt at %v, %v", cgx, cgy)
                        }
                    }
                }
            }
            if ground.RoadAt(ground.MinX + ground.Width - 1, cgy) != 0 {
                reachesEdge = true
            }
        }

        // the cell a road starts on can lose its road as in the original: with no pair of neighbors
        // it only has bit 0, which the last pass clears for half of the roads
        if !reachesEdge {
            test.Fatalf("the road east does not reach the edge")
        }
    }
}

// without a road on the world map at the battle, the battlefield has none
func TestNoRoadsWithoutOne(test *testing.T) {
    zone := ZoneType{}
    zone.Ground.Roads[5] = true
    ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, zone)
    for _, bits := range ground.Roads {
        if bits != 0 {
            test.Fatalf("a road without one at the battle")
        }
    }
}

// with pictures added to a use, the battle picks among the game's and the added ones
func TestAddedPicturesArePicked(test *testing.T) {
    ground := testGround(5, 5)
    ground.Extras = map[int]int{groundGrassFirst: 2, groundRoughFirst + 15: 1}

    seen := make(map[int]bool)
    for range 400 {
        seen[ground.pictureOf(2, 2)] = true
    }
    for _, picture := range []int{0, 1, 2, 3, groundExtraFirst, groundExtraFirst + 1} {
        if !seen[picture] {
            test.Fatalf("grass %v was never picked: %v", picture, seen)
        }
    }
    if len(seen) != 6 {
        test.Fatalf("grass: %v", seen)
    }

    role, number, added := groundExtra(groundExtraFirst + 1)
    if !added || role.Name != "grass" || number != 6 {
        test.Fatalf("the second added grass: %+v %v %v", role, number, added)
    }
    if _, _, added := groundExtra(47); added {
        test.Fatalf("a picture of the game counts as added")
    }

    // a piece of raised ground on its own has one picture of the game, 47
    ground.setGroup(2, 2, TerrainRough)
    seen = make(map[int]bool)
    for range 200 {
        seen[ground.pictureOf(2, 2)] = true
    }
    extra := groundExtraFirst + (groundRoughFirst + 15) * groundExtraStep
    if len(seen) != 2 || !seen[47] || !seen[extra] {
        test.Fatalf("rough: %v", seen)
    }
    role, number, _ = groundExtra(extra)
    if role.Name != "rough single" || number != 2 {
        test.Fatalf("the added rough: %+v %v", role, number)
    }
}

// trees, rocks and houses: with pictures added, all are picked, and the anchors of the pictures of
// the game are the original's
func TestSceneryPool(test *testing.T) {
    pool := sceneryPool{Set: "Grass", Name: "tree", Lbx: "cmbgrass.lbx", First: sceneryTreeIndex, Count: sceneryPictures, Extras: 2}

    game := make(map[int]bool)
    added := make(map[int]bool)
    for range 400 {
        piece := pool.piece(SceneryTree, 10, 20)
        if piece.Number > 0 {
            added[piece.Number] = true
            if piece.Set != "Grass" || piece.Name != "tree" {
                test.Fatalf("an added tree of %v/%v", piece.Set, piece.Name)
            }
        } else {
            game[piece.Index] = true
        }
    }
    if len(game) != 5 || !game[48] || !game[52] || len(added) != 2 || !added[6] || !added[7] {
        test.Fatalf("the game's %v, added %v", game, added)
    }

    tree := SceneryPiece{Kind: SceneryTree}
    rock := SceneryPiece{Kind: SceneryRock}
    if x, y := tree.anchor(16, 18); x != 8 || y != 13 {
        test.Fatalf("tree: %v, %v", x, y)
    }
    if x, y := rock.anchor(12, 13); x != 6 || y != 12 {
        test.Fatalf("rock: %v, %v", x, y)
    }
}

// large pieces lie on open grass, never two on a cell, and trees keep off them
func TestLargePieces(test *testing.T) {
    zone := ZoneType{City: &citylib.City{}}
    zone.Ground.Roads = [9]bool{true, true, true, true, true, true, true, true, true}

    for range 10 {
        ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, zone)
        if len(ground.Large) == 0 {
            test.Fatalf("no large pieces on grass")
        }

        seen := make(map[image.Point]bool)
        for _, piece := range ground.Large {
            if piece.Number < 1 || piece.Number > largeGameCount {
                test.Fatalf("picture %v of %v", piece.Number, largeGameCount)
            }
            for _, cell := range largeCells(piece.Cgx, piece.Cgy) {
                if seen[cell] {
                    test.Fatalf("two large pieces on %v", cell)
                }
                seen[cell] = true

                if ground.GroupAt(cell.X, cell.Y) != TerrainGrass || ground.RoadAt(cell.X, cell.Y) != 0 || ground.HeightAt(cell.X, cell.Y) != 0 {
                    test.Fatalf("a large piece on %v: group %v, road %v, height %v", cell, ground.GroupAt(cell.X, cell.Y), ground.RoadAt(cell.X, cell.Y), ground.HeightAt(cell.X, cell.Y))
                }
                if plateauKeepsOut(zone, cell.X, cell.Y) {
                    test.Fatalf("a large piece in the town at %v", cell)
                }
                if picture := ground.Picture[ground.index(cell.X, cell.Y)]; picture > 3 {
                    test.Fatalf("a large piece on picture %v at %v", picture, cell)
                }
                if ground.sceneryAllowed(cell.X, cell.Y) {
                    test.Fatalf("trees can stand on the large piece at %v", cell)
                }
            }
        }
    }
}

// the patch of dirt the game makes: dirt in the middle, see-through at the corners and beyond the
// four tiles
func TestDirtPatch(test *testing.T) {
    palette := make(color.Palette, 256)
    for index := range palette {
        palette[index] = color.RGBA{R: uint8(index), A: 255}
    }
    palette[0] = color.RGBA{}

    // a tile filled whole, to see what the patch takes of it
    asked := make(map[int]bool)
    patch := MakeDirtPatch(func(index int) *image.Paletted {
        asked[index] = true
        tile := image.NewPaletted(image.Rect(0, 0, largeTileWidth, largeTileHeight), palette)
        for y := range largeTileHeight {
            for x := range largeTileWidth {
                tile.SetColorIndex(x, y, uint8(index))
            }
        }
        return tile
    })

    if patch == nil || patch.Bounds().Dx() != largeWidth || patch.Bounds().Dy() != largeHeight {
        test.Fatalf("the patch: %v", patch)
    }
    for index := groundDirtFirst; index < groundDirtFirst + groundDirtCount; index++ {
        if !asked[index] {
            test.Fatalf("dirt %v was not used", index)
        }
    }
    if len(asked) != groundDirtCount {
        test.Fatalf("pictures used: %v", asked)
    }
    if patch.ColorIndexAt(largeWidth / 2, largeHeight / 2) == 0 {
        test.Fatalf("no dirt in the middle")
    }
    for _, corner := range []image.Point{{0, 0}, {largeWidth - 1, 0}, {0, largeHeight - 1}, {largeWidth - 1, largeHeight - 1}, {largeWidth / 2, 1}, {2, largeHeight / 2}} {
        if patch.ColorIndexAt(corner.X, corner.Y) != 0 {
            test.Fatalf("dirt at %v", corner)
        }
    }

    if MakeDirtPatch(func(index int) *image.Paletted { return nil }) != nil {
        test.Fatalf("a patch without tiles")
    }
}

// the cluster of rocks the game makes: all five rocks, inside of the picture, taller than the four
// tiles for the rocks at the top
func TestRockCluster(test *testing.T) {
    palette := make(color.Palette, 256)
    for index := range palette {
        palette[index] = color.RGBA{R: uint8(index), A: 255}
    }
    palette[0] = color.RGBA{}

    asked := make(map[int]bool)
    cluster := MakeRockCluster(func(index int) *image.Paletted {
        asked[index] = true
        rock := image.NewPaletted(image.Rect(0, 0, 12, 13), palette)
        for y := 6; y < 13; y++ {
            for x := 1; x < 11; x++ {
                rock.SetColorIndex(x, y, uint8(index))
            }
        }
        return rock
    })

    if cluster == nil || cluster.Bounds().Dx() != largeWidth || cluster.Bounds().Dy() != largeHeight + rockClusterAbove {
        test.Fatalf("the cluster: %v", cluster)
    }
    if len(asked) != sceneryPictures || !asked[sceneryRockIndex] || !asked[sceneryRockIndex + sceneryPictures - 1] {
        test.Fatalf("rocks used: %v", asked)
    }

    seen := make(map[uint8]bool)
    for _, value := range cluster.Pix {
        seen[value] = true
    }
    if len(seen) != sceneryPictures + 1 || !seen[0] {
        test.Fatalf("rocks in the picture: %v", seen)
    }
    if len(MakeLargePieces(func(index int) *image.Paletted { return nil })) != largeGameCount {
        test.Fatalf("the game makes %v large pieces", largeGameCount)
    }
}
