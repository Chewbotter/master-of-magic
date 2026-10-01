package combat

// THE FIELD IS SORTED BY DEPTH (user, 2026-09-30: "There's still some popping when moving
// diagonally- specifically because the priority of which one is in front changes between start and
// end. This also occurs during attack animations"). The original sorts by the row of cells, then
// the cell (DrawOrder). That is right for what stands still in the middle of a cell, but a figure
// that walks or lunges is between cells, and no cell gives it the right place: on a step into the
// next row it is in front of a neighbor from the start that it only passes at the end, and figures
// that close in for a strike are drawn whole over or under the figures of the other unit, whatever
// their places. Now everything is sorted by how far down the original's screen it stands (its
// feet), which changes smoothly while it moves, so two figures change places only where their feet
// are level. Things of the same depth keep the original's order.
//
// What belongs to a cell as a whole (a piece of wall, a house, the structure in the middle) gets
// the depth that puts it where the original's order has it against figures standing in the cells
// around it: a quarter of a cell down for what stands at its far edges (after the figures of the
// cells behind, before its own), three quarters for the near walls (after its own figures, before
// the ones of the cells in front). DepthSort false gives the original's order back.

import (
    "math"
)

const DepthSort = true

// the part of a cell down the screen at which what stands at its far edges is sorted, and what
// stands at its near edges
const depthFarEdge = 3.5
const depthNearEdge = 11.5

// steps of depth per art pixel
const depthSteps = 64
// room for the order of the original among things of the same depth
const depthTieRoom = 1 << 26

// the place in the order of a thing whose feet are at x, y on the original's screen. tie decides
// among things at the same depth: a position on the original's screen whose order of the original
// is taken
func depthOrder(x float64, y float64, tieX int, tieY int) int {
    if !DepthSort {
        return DrawOrder(tieX, tieY)
    }
    depth := int(math.Floor(y * depthSteps))
    return depth * depthTieRoom + DrawOrder(tieX, tieY) + depthTieRoom / 2
}

// the order of a thing that stands still at a position of the original's screen
func pointOrder(x int, y int) int {
    return depthOrder(float64(x), float64(y), x, y)
}

// the order of what belongs to a cell as a whole: part is how far down the cell it is sorted
// (depthFarEdge, depthNearEdge), x, y is its position for the original's order
func cellOrder(cgx int, cgy int, part float64, x int, y int) int {
    _, top := cellScreen(cgx, cgy, 0, 0)
    return depthOrder(float64(x), float64(top) + part, x, y)
}

// the cell a position of the original's screen is in
func screenCell(x int, y int) (int, int) {
    half := (x - cellOriginX) / 2
    cellX, _ := floorDivide(half + y - cellOriginY, CellStepX)
    cellY, _ := floorDivide(y - cellOriginY - half, CellStepX)
    return cellX, cellY
}
