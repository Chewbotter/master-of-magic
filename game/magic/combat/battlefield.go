package combat

// The geometry of the battlefield, matched to the original game.
//
// The original's battlefield is a grid of 21 by 22 cells. Cell (cgx, cgy) is anchored on the screen at
//   x = (cgx - cgy) * 16 + 158
//   y = (cgx + cgy) * 8 - 80
// which is the top corner of the cell's diamond. The terrain picture of the cell is drawn 14 pixels to
// the left of the anchor, and a single figure stands 1 right and 8 below it, the middle of the diamond.
// (ReMoM: MoM/src/Combat.c Combat_Grid_Screen_Coordinates, Combat_Compose_Background, Spawn_Figure_Entity,
// Spawn_Structure_Entities, Deploy_Battle_Units; MoM/src/CMBTDEF.h. The code here is ours.)
//
// Our grid is 30 by 30 tiles and already lays out armies, city walls and the gate like the original
// under the mapping
//   cgx = tile y - 3
//   cgy = 22 - tile x
// The camera matrix maps a tile to the MIDDLE of its diamond, which is what the rest of the combat
// screen draws around.

import (
    "github.com/hajimehoshi/ebiten/v2"
)

// the original's grid
const OriginalGridWidth = 21
const OriginalGridHeight = 22

// tile = (cellTileX - cgy, cgx + cellTileY)
const cellTileX = 22
const cellTileY = 3

// one step along a grid axis, in original pixels
const CellStepX = 16
const CellStepY = 8

// where the anchor of cell (0, 0) is on the screen
const cellOriginX = 158
const cellOriginY = -80

// from the anchor of a cell to the middle of its diamond
const cellMiddleX = 1
const cellMiddleY = 8

// from the anchor of a cell to the top left of its terrain picture
const cellTerrainX = -14
const cellTerrainY = 0

// from the middle of a tile to the top left of its terrain picture
const TerrainOffsetX = cellTerrainX - cellMiddleX
const TerrainOffsetY = cellTerrainY - cellMiddleY

// the cell of the structure in the middle of a battlefield that has one: a cave, a tower, a node
const lairCellX = 6
const lairCellY = 11

// the picture of a structure is drawn this far left of the anchor of its cell, and its bottom edge
// this far below it
const structureAnchorX = 15
const structureAnchorBelow = 21
const natureNodeAnchorX = 16
const natureNodeAnchorY = 15

func CellToTile(cgx int, cgy int) (int, int) {
    return cellTileX - cgy, cgx + cellTileY
}

func TileToCell(x int, y int) (int, int) {
    return y - cellTileY, cellTileX - x
}

// the tile of the structure in the middle of the battlefield
func LairTile() (int, int) {
    return CellToTile(lairCellX, lairCellY)
}

// maps a tile to the middle of its diamond on the screen, in original pixels
func MakeBattlefieldMatrix() ebiten.GeoM {
    // cgx - cgy = x + y - (cellTileX + cellTileY)
    // cgx + cgy = y - x + (cellTileX - cellTileY)
    var matrix ebiten.GeoM
    matrix.SetElement(0, 0, CellStepX)
    matrix.SetElement(0, 1, CellStepX)
    matrix.SetElement(0, 2, float64(-(cellTileX + cellTileY) * CellStepX + cellOriginX + cellMiddleX))
    matrix.SetElement(1, 0, -CellStepY)
    matrix.SetElement(1, 1, CellStepY)
    matrix.SetElement(1, 2, float64((cellTileX - cellTileY) * CellStepY + cellOriginY + cellMiddleY))
    return matrix
}

// from the middle of the structure's tile to the top left of its picture
func structureOffset(width int, height int, natureNode bool) (float64, float64) {
    if natureNode {
        return float64(-natureNodeAnchorX - cellMiddleX), float64(-natureNodeAnchorY - cellMiddleY)
    }

    return float64(-structureAnchorX - cellMiddleX), float64(-(height - structureAnchorBelow) - cellMiddleY)
}
