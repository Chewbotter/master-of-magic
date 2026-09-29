package game

// Modern controls of the world map: a move of more than one tile is made with two clicks.
//
// The first click on a tile that is not next to the selected stack only plans the move: the path
// is shown and nothing walks. A second click on the same tile sends the stack. A click on another
// tile plans anew. A tile next to the stack is walked to with one click, and the keys move at once.
//
// The planned path is not the path of the stack (UnitStack.CurrentPath): a stack goes on along its
// own path in a later turn, and a move that was never confirmed must not be walked.

import (
    "github.com/kazzmir/master-of-magic/game/magic/display"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/pathfinding"
)

// a path on the world map
type worldPath = pathfinding.Path

// false: every click on the map sends the selected stack at once, as with the classic controls
const ConfirmLongMoves = true

// a move that waits for its second click
type plannedMove struct {
    Stack *playerlib.UnitStack
    FromX int
    FromY int
    ToX int
    ToY int
    Turn uint64
    Path pathfinding.Path
}

var planned plannedMove

func confirmLongMoves() bool {
    return ConfirmLongMoves && display.ModernControls()
}

// true if the tile is the one the stack stands on or one of the 8 around it. distanceX is the
// distance along the map, which goes around
func nextToStack(distanceX int, fromY int, toY int) bool {
    return max(distanceX, -distanceX) <= 1 && max(toY - fromY, fromY - toY) <= 1
}

// the plan is for this stack as it stands, in this turn
func (game *Game) plannedFor(stack *playerlib.UnitStack) bool {
    return planned.Stack != nil && planned.Stack == stack && planned.FromX == stack.X() && planned.FromY == stack.Y() && planned.Turn == game.Model.TurnNumber
}

// the path to show for a move that waits for its second click, nil for none
func (game *Game) plannedPath(stack *playerlib.UnitStack) pathfinding.Path {
    if stack == nil || !confirmLongMoves() || !game.plannedFor(stack) {
        return nil
    }

    return planned.Path
}

// called for a click on a tile of the map while a stack is selected. true: the stack is sent there
// now. false: the click has planned the move, or the tile can not be gone to, and nothing walks
func (game *Game) clickMoves(player *playerlib.Player, stack *playerlib.UnitStack, tileX int, tileY int) bool {
    if !confirmLongMoves() {
        return true
    }

    mapUse := game.Model.CurrentMap()
    if nextToStack(mapUse.XDistance(stack.X(), tileX), stack.Y(), tileY) {
        planned = plannedMove{}
        return true
    }

    if game.plannedFor(stack) && planned.ToX == tileX && planned.ToY == tileY {
        // the second click
        planned = plannedMove{}
        return true
    }

    planned = plannedMove{}

    // the search of doPlayerUpdate
    var pathStack playerlib.PathStack = stack
    if len(stack.ActiveUnits()) == 0 {
        pathStack = playerlib.MakeUnitStackFromUnits(stack.Units())
    }

    path, ok := game.Model.FindPath(stack.X(), stack.Y(), tileX, tileY, player, pathStack, player.GetFog(game.Model.Plane))
    if !ok || len(path) == 0 {
        return false
    }

    planned = plannedMove{
        Stack: stack,
        FromX: stack.X(),
        FromY: stack.Y(),
        ToX: tileX,
        ToY: tileY,
        Turn: game.Model.TurnNumber,
        Path: path,
    }

    return false
}
