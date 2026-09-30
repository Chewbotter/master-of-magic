package ai

import (
    "image"
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// Delta_XY_With_Wrap and Range of the original
func TestChewbotDistances(test *testing.T) {
    if got := chewDistance(1, 5, 58, 7, 60); got != 3 {
        test.Errorf("distance across the edge of the map: %v, want 3", got)
    }
    if got := chewDistance(10, 5, 14, 12, 60); got != 7 {
        test.Errorf("distance along: %v, want 7", got)
    }
    if got := chewRange(10, 10, 14, 12, 60); got != 5 {
        test.Errorf("range: %v, want 4 and half of 2", got)
    }
}

// Effective_Unit_Type_Strength of 8 spearmen of melee 1, defense 2, 1 hit each: hits 8 * 3 * 7/10
// and melee 8 * 40 / 7
func TestChewbotUnitStrength(test *testing.T) {
    if got := chewKindStrength(units.HighMenSpearmen); got != 16 + 45 {
        test.Errorf("strength of spearmen: %v, want 61", got)
    }
}

// a square of the evaluation map keeps its flags when strength is added, and the strength stops at
// its mask
func TestChewbotEvaluationSquare(test *testing.T) {
    world := &chewWorld{Width: 4, Height: 4}
    world.Eval[0] = make([]int, 16)
    world.flagEval(1, 1, 0, chewEvalSite)
    world.addEval(1, 1, 0, 30)
    if got := world.EvalAt(1, 1, 0); got != chewEvalSite | 30 {
        test.Errorf("site with 30: %x", got)
    }
    world.addEval(1, 1, 0, 50000)
    if got := world.EvalAt(1, 1, 0); got != chewEvalSite | chewEvalStrength {
        test.Errorf("strength past the mask: %x", got)
    }
    if got := world.EvalAt(5, 1, 0); got != chewEvalSite | chewEvalStrength {
        test.Errorf("the map wraps across: %x", got)
    }
}

// the nearest free square, and the middle of a coast
func TestChewbotNearestFree(test *testing.T) {
    world := &chewWorld{Width: 10, Height: 10}
    world.Eval[0] = make([]int, 100)
    world.Eval[0][2 * 10 + 2] = chewEvalSite
    squares := []image.Point{image.Pt(2, 2), image.Pt(4, 4), image.Pt(8, 8)}
    square, ok := world.nearestFree(squares, 0, 1, 1, 0)
    if !ok || square != image.Pt(4, 4) {
        test.Errorf("nearest free square: %v %v, want 4,4 (2,2 is taken)", square, ok)
    }
    world.DockSquares[0] = [][]image.Point{nil, {image.Pt(2, 2), image.Pt(4, 6)}}
    x, y, ok := world.dockCenter(0, 1)
    if !ok || x != 3 || y != 4 {
        test.Errorf("middle of the coast: %v,%v %v", x, y, ok)
    }
    if _, _, ok := world.dockCenter(0, 0); ok {
        test.Errorf("the sea has no coast of its own")
    }
}

// a landmass with a city of the wizard is its own when the others' units cost less than a tenth
func TestChewbotLandmassType(test *testing.T) {
    world := &chewWorld{Width: 10, Height: 10}
    if got := world.classify(0, 1, 1, 100, 9); got != chewLandOwn {
        test.Errorf("100 against 9: %v, want own", got)
    }
    if got := world.classify(0, 1, 1, 100, 10); got != chewLandContested {
        test.Errorf("100 against 10: %v, want contested", got)
    }
    if got := world.classify(0, 1, 0, 100, 0); got != chewLandNoOwnCity {
        test.Errorf("no city: %v, want no own city", got)
    }
}
