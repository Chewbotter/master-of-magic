package combat

import (
    "image"
    "slices"
    "testing"

    "github.com/kazzmir/master-of-magic/lib/fraction"

    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
)

// a hill of one cell has a slope on all four edges; a cell inside a plateau has none, a notch in a
// plateau shows an inner corner
func TestSlopePieces(test *testing.T) {
    ground := testGround(7, 7)
    ground.Heights = make([]int, 49)

    ground.setHeight(3, 3, 1)
    if pieces := ground.slopePieces(3, 3); !slices.Equal(pieces, []int{0, 1, 2, 3}) {
        test.Fatalf("single hill: %v", pieces)
    }

    for cgy := 2; cgy <= 4; cgy++ {
        for cgx := 2; cgx <= 4; cgx++ {
            ground.setHeight(cgx, cgy, 1)
        }
    }
    if pieces := ground.slopePieces(3, 3); len(pieces) != 0 {
        test.Fatalf("inside a plateau: %v", pieces)
    }

    // the cell beyond the S corner of the middle (+x, +y of the grid) is low: an inner corner S
    ground.setHeight(4, 4, 0)
    if pieces := ground.slopePieces(3, 3); !slices.Equal(pieces, []int{6}) {
        test.Fatalf("notch at the S corner: %v", pieces)
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
                if ground.GroupAt(cgx, cgy) == TerrainRough {
                    test.Fatalf("rough ground at %v, %v", cgx, cgy)
                }
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
