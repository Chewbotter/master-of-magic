package game

// How far along its path a stack gets in a turn (for the camera, walkcamera.go), and the moment the
// camera rests on a stack that goes on along its path when a turn starts.

import (
    "github.com/kazzmir/master-of-magic/game/magic/maplib"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/lib/coroutine"
    "github.com/kazzmir/master-of-magic/lib/fraction"

    "github.com/hajimehoshi/ebiten/v2"
)

// seconds the camera rests on a stack that goes on along its path by itself, before it walks. 0: none
const AutoMoveHoldTime = 0.4

// how far along a path the stack gets in this turn: the number of points of the path it reaches,
// and the last of them. a point the stack stands on counts as reached
func (game *Game) reachAlong(player *playerlib.Player, stack *playerlib.UnitStack, path worldPath, mapUse *maplib.Map, getStack func(int, int) (playerlib.PathStack, bool)) (int, int, int, bool) {
    moves := stack.GetRemainingMoves()
    x := stack.X()
    y := stack.Y()
    found := false
    reached := 0

    for index, step := range path {
        if step.X == x && step.Y == y {
            reached = index + 1
            continue
        }

        if !moves.GreaterThan(fraction.Zero()) {
            break
        }

        cost, canMove := game.Model.ComputeTerrainCost(stack, x, y, step.X, step.Y, mapUse, getStack)
        if !canMove {
            break
        }
        cost = debugMoveCost(player, cost)

        x = step.X
        y = step.Y
        found = true
        reached = index + 1
        moves = moves.Subtract(cost)

        if mapUse.GetEncounter(mapUse.WrapX(step.X), step.Y) != nil {
            // the stack stops there
            break
        }
    }

    return reached, x, y, found
}

// the camera rests on the stack for a moment. called before a stack walks on by itself
func (game *Game) holdBeforeAutoMove(yield coroutine.YieldFunc) {
    ticks := int(AutoMoveHoldTime * float64(ebiten.TPS()))
    for tick := 0; tick < ticks; tick++ {
        game.Counter += 1
        if yield() != nil {
            return
        }
    }
}
