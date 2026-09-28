package combat

// What a step on the battlefield costs, as in the original (ReMoM Set_Movement_Cost_Maps,
// Scatter_Tree_Scenery, Apply_Earth_To_Mud, Combat_Move_Path_Find, Set_Movement_Cost_Map). The
// code is ours.
//
// In halves of a move, the cell a step goes into costs: grass and dirt 2, rough 4, a road 1. Every
// tree in the cell adds 1, up to 4. Mud costs 12. A diagonal step costs 1 more, a step up or down
// the slope of a plateau 2 more (plateau.go, not in the original). Flying units pay 2
// for every cell, whatever lies there. A unit can take a step as long as it has any movement left
// before it (CanFollowPath), so a step into rough or mud with half a move left is allowed and uses
// it up.

import (
    "image"

    "github.com/kazzmir/master-of-magic/lib/fraction"
)

// false moves as before the classic costs: every cell 1, a diagonal step 1.5
var ClassicMoveCosts = true

const (
    moveHalvesGround = 2
    moveHalvesRough = 4
    moveHalvesRoad = 1
    moveHalvesTreesMax = 4
    moveHalvesMud = 12
    moveHalvesDiagonal = 1
    // going up or down the slope of a plateau, see plateau.go
    moveHalvesSlope = 2
)

// the path search gives up on a path that costs more than this. with the costs of the ground a walk
// across the field can cost much more than the 50 that were enough with 1 a cell
const maxPathCost = 300

// what going into a cell costs, in halves of a move, not counting a diagonal step
func (model *CombatModel) cellMoveHalves(x int, y int, flying bool) int {
    tile := model.GetTile(x, y)
    if flying || tile == nil {
        return moveHalvesGround
    }

    if tile.Mud {
        return moveHalvesMud
    }
    if tile.Road {
        return moveHalvesRoad
    }

    halves := moveHalvesGround
    if tile.Ground == TerrainRough {
        halves = moveHalvesRough
    }

    if model.Ground != nil && halves < moveHalvesTreesMax {
        cgx, cgy := TileToCell(x, y)
        halves = min(moveHalvesTreesMax, halves + model.Ground.TreesAt(cgx, cgy))
    }

    return halves
}

// what a step from one tile to the next costs, in moves
func (model *CombatModel) StepCost(from image.Point, to image.Point, flying bool) fraction.Fraction {
    if !ClassicMoveCosts || from == to {
        return pathCost(from, to)
    }

    halves := model.cellMoveHalves(to.X, to.Y, flying)
    if from.X != to.X && from.Y != to.Y {
        halves += moveHalvesDiagonal
    }
    if !flying && model.Ground != nil {
        fromX, fromY := TileToCell(from.X, from.Y)
        toX, toY := TileToCell(to.X, to.Y)
        if model.Ground.isSlope(fromX, fromY, toX, toY) {
            halves += moveHalvesSlope
        }
    }

    return fraction.Make(halves, 2)
}

func (unit *ArmyUnit) stepCost(from image.Point, to image.Point) fraction.Fraction {
    if unit.Model == nil {
        return pathCost(from, to)
    }
    return unit.Model.StepCost(from, to, unit.IsFlying())
}
