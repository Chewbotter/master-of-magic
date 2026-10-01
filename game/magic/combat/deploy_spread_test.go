package combat

import (
    "image"
    "testing"
)

// the places of an army that stands apart: no two next to each other across or along, room for
// the largest test army, and the front row is the one of the original
func TestSpreadCells(test *testing.T) {
    for _, team := range []Team{TeamDefender, TeamAttacker} {
        cells := spreadCells(team)
        if len(cells) < 36 {
            test.Fatalf("team %v: %v places", team, len(cells))
        }

        taken := make(map[image.Point]bool)
        for _, cell := range cells {
            if taken[cell] {
                test.Fatalf("team %v: %v twice", team, cell)
            }
            taken[cell] = true
            if cell.X < 0 || cell.Y < 0 || cell.X >= OriginalGridWidth || cell.Y >= OriginalGridHeight {
                test.Fatalf("team %v: %v off the grid", team, cell)
            }
        }
        for cell := range taken {
            for _, side := range []image.Point{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
                if taken[cell.Add(side)] {
                    test.Fatalf("team %v: %v and %v side by side", team, cell, cell.Add(side))
                }
            }
        }

        front := deployDefenderRows[0]
        if team == TeamAttacker {
            front = deployAttackerRows[0]
        }
        if cells[0] != image.Pt(front, deployColumns[0]) {
            test.Fatalf("team %v: the first place is %v", team, cells[0])
        }
    }
}
