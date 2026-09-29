package combat

import (
    "image"
    "image/color"
    "testing"

    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    "github.com/kazzmir/master-of-magic/game/magic/mod"
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

        // every picture once at most
        if len(ground.Large) > largeGameCount {
            test.Fatalf("%v large pieces of %v pictures", len(ground.Large), largeGameCount)
        }
        numbers := make(map[int]bool)

        seen := make(map[image.Point]bool)
        for _, piece := range ground.Large {
            if piece.Number < 1 || piece.Number > largeGameCount {
                test.Fatalf("picture %v of %v", piece.Number, largeGameCount)
            }
            if numbers[piece.Number] {
                test.Fatalf("picture %v twice", piece.Number)
            }
            numbers[piece.Number] = true
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
                if x, y := CellToTile(cell.X, cell.Y); x < 0 || y < 0 || x >= BattlefieldWidth || y >= BattlefieldHeight {
                    test.Fatalf("a large piece beyond the field at tile %v, %v", x, y)
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

// props: none without pictures, and with pictures the ones that lie and the ones that stand, on
// grass without a road and off the large pieces
func TestProps(test *testing.T) {
    ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, ZoneType{})

    // the replacement folder has none
    if props := makeProps(BattlefieldWidth, BattlefieldHeight, "Grass", ZoneType{}, ground); len(props) != 0 {
        test.Fatalf("%v props without pictures", len(props))
    }

    pool := sceneryPool{Set: "Grass", Name: propFlatName, Extras: 3}
    props := scatterProps(propsFlat, SceneryFlat, pool, ZoneType{}, originalArea(), ground)
    if len(props) != propsFlat {
        test.Fatalf("%v props of %v", len(props), propsFlat)
    }

    numbers := make(map[int]bool)
    for range 20 {
        for _, prop := range scatterProps(propsFlat, SceneryFlat, pool, ZoneType{}, originalArea(), ground) {
            numbers[prop.Number] = true
            if prop.Kind != SceneryFlat || prop.Set != "Grass" || prop.Name != propFlatName {
                test.Fatalf("a prop of %v %v/%v", prop.Kind, prop.Set, prop.Name)
            }
        }
    }
    if len(numbers) != 3 || !numbers[1] || !numbers[3] {
        test.Fatalf("pictures of props: %v", numbers)
    }

    // one that lies has its middle on its place, one that stands its foot
    flat := SceneryPiece{Kind: SceneryFlat}
    standing := SceneryPiece{Kind: SceneryProp}
    if x, y := flat.anchor(20, 10); x != 10 || y != 5 {
        test.Fatalf("flat: %v, %v", x, y)
    }
    if x, y := standing.anchor(20, 10); x != 10 || y != 10 - propAnchorBelow {
        test.Fatalf("standing: %v, %v", x, y)
    }
}

// a bag gives every picture once before any comes again
func TestPictureBag(test *testing.T) {
    var bag pictureBag
    for range 5 {
        seen := make(map[int]bool)
        for range 16 {
            seen[bag.next(16)] = true
        }
        if len(seen) != 16 {
            test.Fatalf("%v of 16 pictures in a round", len(seen))
        }
    }
}

// added pictures of the ground are sprinkled in: AddedGroundShare of the tiles, however many
// pictures there are, the same one never within addedRepeatReach of itself, and the game's by
// chance
func TestAddedGroundIsSprinkled(test *testing.T) {
    share := AddedGroundShare
    defer func() { AddedGroundShare = share }()

    make := func(extras int) *BattleGround {
        ground := testGround(40, 40)
        ground.Extras = map[int]int{groundGrassFirst: extras}
        ground.choosePictures()
        return ground
    }
    added := func(ground *BattleGround) float64 {
        count := 0
        for _, picture := range ground.Picture {
            if picture >= groundExtraFirst {
                count += 1
            }
        }
        return float64(count) / float64(len(ground.Picture))
    }

    AddedGroundShare = 0.15
    for _, extras := range []int{4, 12, 40} {
        ground := make(extras)

        // about the share, whatever the number of pictures. a little less with few, which give
        // way where all of them are near
        got := added(ground)
        if got < 0.09 || got > 0.19 {
            test.Fatalf("%v added pictures: %v of the ground, want about %v", extras, got, AddedGroundShare)
        }
        test.Logf("%v added pictures: %.3f of the ground shows one", extras, got)

        // the same added picture is never within reach of itself
        for cgy := 0; cgy < 40; cgy++ {
            for cgx := 0; cgx < 40; cgx++ {
                picture := ground.Picture[ground.index(cgx, cgy)]
                if picture < groundExtraFirst {
                    continue
                }
                for dy := -addedRepeatReach; dy <= addedRepeatReach; dy++ {
                    for dx := -addedRepeatReach; dx <= addedRepeatReach; dx++ {
                        if (dx != 0 || dy != 0) && ground.contains(cgx + dx, cgy + dy) && ground.Picture[ground.index(cgx + dx, cgy + dy)] == picture {
                            test.Fatalf("added picture %v at %v, %v and again %v, %v from it", picture, cgx, cgy, dx, dy)
                        }
                    }
                }
            }
        }
    }

    // different added pictures do lie side by side: they are not set out evenly
    ground := make(12)
    beside := 0
    for cgy := 0; cgy < 40; cgy++ {
        for cgx := 0; cgx < 39; cgx++ {
            if ground.Picture[ground.index(cgx, cgy)] >= groundExtraFirst && ground.Picture[ground.index(cgx + 1, cgy)] >= groundExtraFirst {
                beside += 1
            }
        }
    }
    if beside == 0 {
        test.Fatalf("no two added pictures side by side")
    }

    // the game's pictures by chance: all four, and the same ones side by side
    counts := map[int]int{}
    repeats := 0
    for cgy := 0; cgy < 40; cgy++ {
        for cgx := 0; cgx < 39; cgx++ {
            picture := ground.Picture[ground.index(cgx, cgy)]
            if picture < groundExtraFirst {
                counts[picture] += 1
                if ground.Picture[ground.index(cgx + 1, cgy)] == picture {
                    repeats += 1
                }
            }
        }
    }
    if len(counts) != groundGrassCount || repeats < 100 {
        test.Fatalf("the game's pictures: %v, %v side by side", counts, repeats)
    }

    AddedGroundShare = 0
    if got := added(make(4)); got != 0 {
        test.Fatalf("at 0: %v", got)
    }
    AddedGroundShare = 1
    if got := added(make(40)); got < 0.95 {
        test.Fatalf("at 1 with 40 pictures: %v", got)
    }
}

// with many pictures of large pieces no more than the share of the field, each a different one
func TestLargePiecesOnceEach(test *testing.T) {
    ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, ZoneType{})
    ground.Large = nil
    ground.placeLarge(BattlefieldWidth, BattlefieldHeight, ZoneType{}, 40)

    most := int(float64(BattlefieldWidth * BattlefieldHeight) * largeShare / 4)
    if len(ground.Large) == 0 || len(ground.Large) > most {
        test.Fatalf("%v large pieces, at most %v", len(ground.Large), most)
    }
    numbers := make(map[int]bool)
    for _, piece := range ground.Large {
        if numbers[piece.Number] || piece.Number < 1 || piece.Number > 40 {
            test.Fatalf("picture %v", piece.Number)
        }
        numbers[piece.Number] = true
    }
}

// a biome is a kind of one landscape and keeps the rules that were there: a forest its trees, hills
// their raised ground. its added pictures are the ones of its own folder
func TestBiomes(test *testing.T) {
    var ground ZoneGround
    ground.SetBiome(mod.BiomeForest)
    if !ground.Forest || ground.Hills || ground.Biome != mod.BiomeForest {
        test.Fatalf("forest: %+v", ground)
    }
    ground.SetBiome(mod.BiomeHills)
    if ground.Forest || !ground.Hills {
        test.Fatalf("hills: %+v", ground)
    }
    ground.SetBiome(mod.BiomeSwamp)
    if ground.Forest || ground.Hills {
        test.Fatalf("swamp: %+v", ground)
    }

    if BiomeLandscape(mod.BiomeVolcano, CombatLandscapeGrass) != CombatLandscapeMountain || BiomeLandscape(mod.BiomeSwamp, CombatLandscapeDesert) != CombatLandscapeGrass || BiomeLandscape("", CombatLandscapeTundra) != CombatLandscapeTundra {
        test.Fatalf("landscapes of biomes")
    }

    zone := ZoneType{}
    zone.Ground.SetBiome(mod.BiomeVolcano)
    made := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeMountain, 0, zone)
    if made.Set != "Volcano" || made.BaseSet != "Mountain" {
        test.Fatalf("folders: %v of %v", made.Set, made.BaseSet)
    }
    plain := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeMountain, 0, ZoneType{})
    if plain.Set != "Mountain" || plain.BaseSet != "Mountain" {
        test.Fatalf("folders without a biome: %v of %v", plain.Set, plain.BaseSet)
    }
}
