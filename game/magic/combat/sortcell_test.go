package combat

import (
    "math"
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// a unit that walks is sorted by one cell for all of a step: the order of its figures does not
// change while it is between two cells, and it is the cell of the two that is drawn last
func TestSortCellWhileWalking(test *testing.T) {
    field := MakeBattlefieldMatrix()
    orderOf := func(x float64, y float64, facing units.Facing) int {
        middleX, middleY := sortCellMiddle(x, y, facing)
        return DrawOrder(int(math.Floor(middleX)), int(math.Floor(middleY)))
    }
    cellOrder := func(x float64, y float64) int {
        middleX, middleY := field.Apply(x, y)
        return DrawOrder(int(math.Floor(middleX)), int(math.Floor(middleY)))
    }

    for _, step := range [][2]float64{{1, 0}, {0, 1}, {1, 1}, {-1, 0}, {0, -1}, {-1, 1}, {1, -1}, {-1, -1}} {
        fromX, fromY := 10.0, 12.0
        toX, toY := fromX + step[0], fromY + step[1]
        want := max(cellOrder(fromX, fromY), cellOrder(toX, toY))
        facing := faceTowards(int(fromX), int(fromY), int(toX), int(toY))
        for part := 0.05; part < 1; part += 0.05 {
            if order := orderOf(fromX + step[0] * part, fromY + step[1] * part, facing); order != want {
                test.Fatalf("step %v at %v: order %v, the cell drawn last has %v", step, part, order, want)
            }
        }
    }

    // standing: its own cell
    if orderOf(10, 12, units.FacingUp) != cellOrder(10, 12) {
        test.Fatalf("a unit that stands is not sorted by its cell")
    }
}
