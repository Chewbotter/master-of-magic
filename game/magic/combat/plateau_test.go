package combat

import (
    "image"
    "testing"

    "github.com/kazzmir/master-of-magic/lib/fraction"

    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
)

func outlineGround(size int, raised func(cgx int, cgy int) bool) *BattleGround {
    ground := testGround(size, size)
    ground.Heights = make([]int, size * size)
    for cgy := range size {
        for cgx := range size {
            if raised(cgx, cgy) {
                ground.setHeight(cgx, cgy, 1)
            }
        }
    }
    return ground
}

// the corners of the outline: a hill of one cell is rounded on all sides, the corner of a plateau is
// cut along an arc, a staircase in half, an inward corner is filled in
func TestOutlineCorners(test *testing.T) {
    // a hill of one cell
    ground := outlineGround(9, func(cgx int, cgy int) bool { return cgx == 4 && cgy == 4 })
    outline := makePlateauOutline(ground)
    for corner, cut := range outline.corners[ground.index(4, 4)] {
        if cut <= 0 || cut > 0.5 {
            test.Fatalf("single hill, corner %v: %v", corner, cut)
        }
    }
    if !outline.topAt(4.5, 4.5) || outline.topAt(4.02, 4.02) || outline.topAt(4.98, 4.98) {
        test.Fatalf("single hill: middle %v, corners %v %v", outline.topAt(4.5, 4.5), outline.topAt(4.02, 4.02), outline.topAt(4.98, 4.98))
    }

    // a plateau of 3 by 3: its corners are arcs, its sides and its middle are whole
    ground = outlineGround(9, func(cgx int, cgy int) bool { return cgx >= 3 && cgx <= 5 && cgy >= 3 && cgy <= 5 })
    outline = makePlateauOutline(ground)
    south := outline.corners[ground.index(5, 5)]
    if south[2] < 0.75 || south[0] != 0 || south[1] != 0 || south[3] != 0 {
        test.Fatalf("corner of a plateau: %v", south)
    }
    if outline.corners[ground.index(4, 5)] != [4]float64{} || outline.corners[ground.index(4, 4)] != [4]float64{} {
        test.Fatalf("side %v, middle %v", outline.corners[ground.index(4, 5)], outline.corners[ground.index(4, 4)])
    }
    if outline.topAt(5.97, 5.97) || !outline.topAt(5.3, 5.3) {
        test.Fatalf("corner of a plateau: tip %v, inside %v", outline.topAt(5.97, 5.97), outline.topAt(5.3, 5.3))
    }

    // a staircase: every step is cut in half
    ground = outlineGround(12, func(cgx int, cgy int) bool { return cgx + cgy <= 10 && cgx >= 1 && cgy >= 1 })
    outline = makePlateauOutline(ground)
    for _, cgx := range []int{4, 5, 6} {
        cuts := outline.corners[ground.index(cgx, 10 - cgx)]
        if cuts[2] != slopeCutDiagonal {
            test.Fatalf("step %v: %v", cgx, cuts)
        }
    }
    if outline.topAt(5.8, 5.8) || !outline.topAt(5.2, 5.2) {
        test.Fatalf("step: lower half %v, upper half %v", outline.topAt(5.8, 5.8), outline.topAt(5.2, 5.2))
    }

    // an inward corner: the low cell is filled in at the corner toward the plateau
    ground = outlineGround(12, func(cgx int, cgy int) bool { return cgx >= 1 && cgy >= 1 && cgx <= 9 && cgy <= 9 && (cgx <= 4 || cgy <= 4) })
    outline = makePlateauOutline(ground)
    fills := outline.corners[ground.index(5, 5)]
    if fills[0] != slopeFillRadius || fills[1] != 0 || fills[2] != 0 || fills[3] != 0 {
        test.Fatalf("inward corner: %v", fills)
    }
    if !outline.topAt(5.03, 5.03) || outline.topAt(5.5, 5.5) {
        test.Fatalf("inward corner: at the corner %v, middle %v", outline.topAt(5.03, 5.03), outline.topAt(5.5, 5.5))
    }
}

// the shading: light on top, the slope toward the viewer below the hill, the shorter one above it,
// nothing far from it
func TestSlopeShading(test *testing.T) {
    ground := outlineGround(9, func(cgx int, cgy int) bool { return cgx == 4 && cgy == 4 })
    shading, left, top := makeSlopeShading(ground)
    if shading == nil {
        test.Fatalf("no shading")
    }

    centerX, centerY := cellCenterScreen(4, 4)
    at := func(dx int, dy int) (uint32, uint32) {
        red, _, _, alpha := shading.At(centerX - left + dx, centerY - top + dy).RGBA()
        return red, alpha
    }

    if red, alpha := at(0, 0); alpha == 0 || red == 0 {
        test.Fatalf("the top is not lighter: %v %v", red, alpha)
    }
    if red, alpha := at(0, CellStepY + 2); alpha == 0 || red != 0 {
        test.Fatalf("no slope below the hill: %v %v", red, alpha)
    }
    if red, alpha := at(0, -CellStepY - 1); alpha == 0 || red != 0 {
        test.Fatalf("no slope above the hill: %v %v", red, alpha)
    }
    if _, alpha := at(0, CellStepY + slopeFaceDepth + 2); alpha != 0 {
        test.Fatalf("shading far below the hill: %v", alpha)
    }

    // no plateaus, no picture
    if shading, _, _ := makeSlopeShading(outlineGround(9, func(cgx int, cgy int) bool { return false })); shading != nil {
        test.Fatalf("shading without plateaus")
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
