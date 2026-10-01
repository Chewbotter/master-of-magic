package combat

import (
    "math"
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/unitview"
)

// what stands at the edges of a cell keeps the original's place against the figures standing in the
// cells around it: a far wall after the figures of the cells behind and before its own, a near wall
// after its own and before the ones of the cells in front
func TestDepthOfCellEdges(test *testing.T) {
    field := MakeBattlefieldMatrix()
    cgx, cgy := 7, 12

    // the orders of every figure of a full unit standing in a cell
    figures := func(cellX int, cellY int) []int {
        tileX, tileY := CellToTile(cellX, cellY)
        middleX, middleY := field.Apply(float64(tileX), float64(tileY))
        var out []int
        for _, point := range unitview.CombatPoints(8) {
            x, y := math.Floor(middleX) + float64(point.X), math.Floor(middleY) + float64(point.Y)
            out = append(out, depthOrder(x, y, int(x), int(y)))
        }
        return out
    }

    x, y := cellScreen(cgx, cgy, 0, 0)
    far := cellOrder(cgx, cgy, depthFarEdge, x, y)
    near := cellOrder(cgx, cgy, depthNearEdge, x, y + 14)

    for _, behind := range [][2]int{{cgx - 1, cgy}, {cgx, cgy - 1}} {
        for _, order := range figures(behind[0], behind[1]) {
            if order >= far {
                test.Fatalf("a figure of the cell %v is not behind the far wall", behind)
            }
        }
    }
    for _, order := range figures(cgx, cgy) {
        if order <= far || order >= near {
            test.Fatalf("a figure of the cell is not between its walls")
        }
    }
    for _, front := range [][2]int{{cgx + 1, cgy}, {cgx, cgy + 1}} {
        for _, order := range figures(front[0], front[1]) {
            if order <= near {
                test.Fatalf("a figure of the cell %v is not in front of the near wall", front)
            }
        }
    }
}

// a figure that walks down the screen past one that stands goes in front of it once, where their
// feet are level, and stays there
func TestDepthWhileWalking(test *testing.T) {
    standing := depthOrder(100, 50, 100, 50)
    changes := 0
    before := depthOrder(110, 40, 110, 40) < standing
    for step := 0; step <= 100; step++ {
        y := 40 + float64(step) / 5
        now := depthOrder(110, y, 110, 40) < standing
        if now != before {
            changes += 1
            if math.Abs(y - 50) > 0.25 {
                test.Fatalf("changed places at %v, the feet are level at 50", y)
            }
        }
        before = now
    }
    if changes != 1 {
        test.Fatalf("changed places %v times", changes)
    }
}
