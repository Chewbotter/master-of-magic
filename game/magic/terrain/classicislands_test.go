package terrain

import (
    "image"
    "testing"
)

// a node alone in the sea grows land to the east and south, not to the west (Extend_Islands)
func TestClassicExtendIslands(test *testing.T) {
    old := classicRoll
    defer func() { classicRoll = old }()

    kinds := make([][]classicKind, ClassicWidth)
    for x := range kinds {
        kinds[x] = make([]classicKind, ClassicHeight)
    }
    // two out of three, 1 + 1 = 2 attempts: east (dx 1, dy 0), then west (dx -1, dy 0)
    rolls := []int{3, 1, 3, 2, 1, 2}
    classicRoll = func(n int) int {
        roll := rolls[0]
        rolls = rolls[1:]
        return roll
    }
    classicExtendIslands(kinds, []ClassicNode{{X: 10, Y: 10}}, nil)
    if kinds[11][10] != classicGrass {
        test.Errorf("the square east of the node should be land")
    }
    if kinds[9][10] != classicOcean {
        test.Errorf("the square west of the node should stay sea")
    }
    if len(rolls) != 0 {
        test.Errorf("rolls left: %v", rolls)
    }

    // a node with land beside it grows nothing
    kinds[11][10] = classicGrass
    rolls = []int{}
    classicExtendIslands(kinds, []ClassicNode{{X: 10, Y: 10}}, []image.Point{})
}
