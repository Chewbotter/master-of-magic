package combat

import (
    "image"
    "testing"
)

// the original's numbers of Get_Effective_Melee_Strength and Get_Effective_Hits
func TestChewbotStrengthFormulas(test *testing.T) {
    threats := []struct {
        strength, figures, perFigure, want int
    }{
        {0, 6, 40, 0},
        {1, 6, 40, 240 / 7},
        {2, 6, 40, 240 / 3},
        {3, 6, 40, 240 * 2 / 3},
        {4, 6, 40, 6 * 20 * 2},
        {10, 1, 40, 20 * 8},
        {5, 4, 60, 4 * 30 * 3},
    }
    for _, use := range threats {
        got := chewThreat(use.strength, use.figures, use.perFigure)
        if got != use.want {
            test.Errorf("threat of strength %v, %v figures, %v: %v, want %v", use.strength, use.figures, use.perFigure, got, use.want)
        }
    }

    hits := []struct {
        hits, defense, want int
    }{
        {10, 0, 15},
        {10, 1, 18},
        {10, 2, 21},
        {10, 3, 24},
        // (4 - 2)^2 / 2 = 2
        {10, 4, 60},
        // (5 - 2)^2 / 2 = 4, whole numbers first
        {10, 5, 120},
    }
    for _, use := range hits {
        got := chewEffectiveHits(use.hits, use.defense)
        if got != use.want {
            test.Errorf("hits %v defense %v: %v, want %v", use.hits, use.defense, got, use.want)
        }
    }

    // a gaze counts 300, three gazes 1200 more than the attack alone
    plain := chewEffectiveMelee(4, 0, 1, 0, chewGazeNone)
    if chewEffectiveMelee(4, 0, 1, 0, chewGazeOne) != plain + 300 || chewEffectiveMelee(4, 0, 1, 0, chewGazeMulti) != plain + 1200 {
        test.Errorf("gazes do not count as the original counts them")
    }
    // no melee attack is nothing, whatever else the unit has
    if chewEffectiveMelee(0, 5, 4, chewPoison, chewGazeOne) != 0 {
        test.Errorf("a unit without a melee attack has a melee strength")
    }
}

func TestChewbotCellsMapBack(test *testing.T) {
    for x := 0; x < 30; x++ {
        for y := 0; y < 30; y++ {
            tile := image.Pt(x, y)
            if fromCell(toCell(tile)) != tile {
                test.Fatalf("tile %v does not map back", tile)
            }
        }
    }

    // the gate of the original is the gate of the fork's walled city
    if fromCell(chewGateCell) != image.Pt(10, 11) {
        test.Errorf("gate cell %v is tile %v", chewGateCell, fromCell(chewGateCell))
    }
}

// Auto_Move_Unit: a unit does not go past the stage point in the direction it moves
func TestChewbotBoxStopsAtStagePoint(test *testing.T) {
    tileOf := func(cgx int, cgy int) image.Point {
        return fromCell(image.Pt(cgx, cgy))
    }

    // no stage point: anywhere
    free := makeChewBox(tileOf(2, 5), tileOf(2, 15), image.Point{}, false)
    if !free.contains(tileOf(2, 14)) {
        test.Errorf("a box without a stage point is not free")
    }

    // moving along cgy toward larger values, stage point at cgy 9: cells up to 9 only
    box := makeChewBox(tileOf(2, 5), tileOf(2, 15), tileOf(3, 9), true)
    if !box.contains(tileOf(2, 9)) || box.contains(tileOf(2, 10)) {
        test.Errorf("a unit moving toward larger cgy is not held at the stage point: %+v", box)
    }
    // across the direction of the move it is free
    if !box.contains(tileOf(12, 7)) {
        test.Errorf("the box limits a unit across its direction: %+v", box)
    }

    // moving toward smaller cgy: from the stage point on
    back := makeChewBox(tileOf(2, 15), tileOf(2, 5), tileOf(3, 9), true)
    if !back.contains(tileOf(2, 9)) || back.contains(tileOf(2, 8)) {
        test.Errorf("a unit moving toward smaller cgy is not held at the stage point: %+v", back)
    }

    // the stage point on the destination: anywhere
    same := makeChewBox(tileOf(2, 5), tileOf(2, 15), tileOf(2, 15), true)
    if !same.contains(tileOf(2, 20)) {
        test.Errorf("a stage point on the destination limits the unit")
    }
}

// a hero that waits for its second turn waits until its turn begins again, and then only once
func TestChewbotHeroWaitsOnce(test *testing.T) {
    model := &CombatModel{CurrentTurn: 3}
    unit := &ArmyUnit{Team: TeamAttacker}

    if model.chewbotWaits(unit) {
        test.Fatalf("a unit waits before any plan")
    }

    chew := model.chewbotState()
    plan := &chewbotPlan{Turn: 3, Pass: map[*ArmyUnit]int{unit: 1}}
    chew.Plans[TeamAttacker] = plan

    if !model.chewbotWaits(unit) {
        test.Fatalf("a hero after its first turn does not wait")
    }
    model.chewbotBeginUnit(unit)
    if model.chewbotWaits(unit) || plan.Pass[unit] != 2 {
        test.Fatalf("the second turn of a hero did not begin")
    }

    // a plan of another turn is no reason to wait
    plan.Pass[unit] = 1
    model.CurrentTurn = 4
    if model.chewbotWaits(unit) {
        test.Fatalf("a hero waits in the turn after")
    }
}
