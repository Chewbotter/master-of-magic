package combat

import (
    "image"
    "slices"
    "testing"

    "github.com/kazzmir/master-of-magic/lib/fraction"

    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
)

// a hill of one cell has a slope on all four edges and four outer corners; a cell inside a plateau
// has none, a notch in a plateau shows an inner corner, the end of a ridge two outer corners
func TestSlopeParts(test *testing.T) {
    ground := testGround(7, 7)
    ground.Heights = make([]int, 49)

    ground.setHeight(3, 3, 1)
    edges, outer, inner := ground.slopeParts(3, 3)
    if !slices.Equal(edges, []int{0, 1, 2, 3}) || !slices.Equal(outer, []int{0, 1, 2, 3}) || len(inner) != 0 {
        test.Fatalf("single hill: %v %v %v", edges, outer, inner)
    }

    for cgy := 2; cgy <= 4; cgy++ {
        for cgx := 2; cgx <= 4; cgx++ {
            ground.setHeight(cgx, cgy, 1)
        }
    }
    edges, outer, inner = ground.slopeParts(3, 3)
    if len(edges) != 0 || len(outer) != 0 || len(inner) != 0 {
        test.Fatalf("inside a plateau: %v %v %v", edges, outer, inner)
    }

    // the cell beyond the S corner of the middle (+x, +y of the grid) is low: an inner corner S
    ground.setHeight(4, 4, 0)
    edges, outer, inner = ground.slopeParts(3, 3)
    if len(edges) != 0 || len(outer) != 0 || !slices.Equal(inner, []int{2}) {
        test.Fatalf("notch at the S corner: %v %v %v", edges, outer, inner)
    }

    // the lower right cell of the plateau: edges SE and SW drop, a cut can go on its S corner
    edges, outer, _ = ground.slopeParts(4, 3)
    if !slices.Equal(edges, []int{1, 2}) || !slices.Equal(outer, []int{2}) {
        test.Fatalf("corner cell: %v %v", edges, outer)
    }
}

// the faces toward the viewer hang below the tile; the far edges stay inside it
func TestSlopeFacesHangBelow(test *testing.T) {
    below := func(picture image.Image) bool {
        bounds := picture.Bounds()
        for y := slopeCanvasTop + slopeTileHeight; y < bounds.Dy(); y++ {
            for x := range bounds.Dx() {
                _, _, _, alpha := picture.At(x, y).RGBA()
                if alpha > 0 {
                    return true
                }
            }
        }
        return false
    }

    if !below(slopeEdgePicture(1)) || !below(slopeEdgePicture(2)) {
        test.Fatalf("no face below the edges SE and SW")
    }
    if below(slopeEdgePicture(0)) || below(slopeEdgePicture(3)) {
        test.Fatalf("a face below the far edges")
    }
    if len(SlopePictures()) != 8 + 4 * len(slopeCuts) {
        test.Fatalf("%v pictures", len(SlopePictures()))
    }
}

// plateaus replace the rough ground, stay out of the town, and a step up or down costs a move more
func TestPlateaus(test *testing.T) {
    zone := ZoneType{City: &citylib.City{}}
    for range 20 {
        ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeMountain, zone)
        raised := 0
        for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
            for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
                if ground.HeightAt(cgx, cgy) > 0 {
                    raised += 1
                    if plateauKeepsOut(zone, cgx, cgy) {
                        test.Fatalf("a plateau in the town at %v, %v", cgx, cgy)
                    }
                }
            }
        }
        if raised == 0 {
            test.Fatalf("no plateaus on mountains")
        }
    }

    ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, ZoneType{})
    for index := range ground.Heights {
        ground.Heights[index] = 0
        ground.Trees[index] = 0
    }
    model := &CombatModel{
        Tiles: makeTiles(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, ZoneType{}),
        Ground: ground,
    }
    for y := range model.Tiles {
        for x := range model.Tiles[y] {
            model.Tiles[y][x].Road = false
            model.Tiles[y][x].Ground = TerrainGrass
        }
    }

    cgx, cgy := TileToCell(10, 11)
    ground.setHeight(cgx, cgy, 1)
    up := model.StepCost(image.Pt(10, 10), image.Pt(10, 11), false)
    down := model.StepCost(image.Pt(10, 11), image.Pt(10, 10), false)
    flying := model.StepCost(image.Pt(10, 10), image.Pt(10, 11), true)
    if !up.Equals(fraction.FromInt(2)) || !down.Equals(fraction.FromInt(2)) || !flying.Equals(fraction.FromInt(1)) {
        test.Fatalf("up %v, down %v, flying %v", up, down, flying)
    }

    // on the top it is ground as any other
    other := image.Pt(10, 12)
    cgx, cgy = TileToCell(other.X, other.Y)
    ground.setHeight(cgx, cgy, 1)
    if across := model.StepCost(image.Pt(10, 11), other, false); !across.Equals(fraction.FromInt(1)) {
        test.Fatalf("across the top: %v", across)
    }
}
