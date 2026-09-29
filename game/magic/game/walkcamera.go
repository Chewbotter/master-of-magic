package game

// The camera while a stack of the player walks (modern controls).
//
// Before, the camera went to where the stack stood and then followed every step, so a player who
// had looked elsewhere saw the map pan to the stack and then shift again with it. Now the camera
// makes ONE move, to the tile the stack will stop on in this turn, and stays there while the
// stack walks to it.

import (
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/game/magic/maplib"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/lib/coroutine"
    "github.com/kazzmir/master-of-magic/lib/fraction"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: the camera goes to the stack and follows its steps, as with the classic controls
const WalkCameraToDestination = true
// true: the camera moves while the stack already walks. false: the camera moves first, then the stack walks
const WalkCameraWithWalk = true

type walkCameraMove struct {
    // a stack of the player walks and the camera is moved from here
    Walking bool
    // the move of the camera is not done yet
    Active bool
    TargetX int
    TargetY int
    StartX float64
    StartY float64
    DeltaX float64
    DeltaY float64
    Frame int
    Frames int
}

var walkCamera walkCameraMove

func walkCameraToDestination(player *playerlib.Player) bool {
    return WalkCameraToDestination && display.ModernControls() && player != nil && player.IsHuman()
}

// the tile the stack will stop on in this turn if nothing stops it before: as far along its path as its moves go
func (game *Game) walkDestination(player *playerlib.Player, stack *playerlib.UnitStack, mapUse *maplib.Map, getStack func(int, int) (playerlib.PathStack, bool)) (int, int, bool) {
    moves := stack.GetRemainingMoves()
    x := stack.X()
    y := stack.Y()
    found := false

    for _, step := range stack.CurrentPath {
        if step.X == x && step.Y == y {
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
        moves = moves.Subtract(cost)

        if mapUse.GetEncounter(mapUse.WrapX(step.X), step.Y) != nil {
            // the stack stops there
            break
        }
    }

    return x, y, found
}

// the row the camera ends on when it is sent to this tile: kept inside the map, as doMoveCamera does
func (game *Game) cameraRowFor(x int, y int) int {
    camera := game.Camera
    camera.SetOffset(0, 0)

    camera.Center(x, y)
    minY := math.Floor(-1 / camera.GetZoom())
    for camera.GetZoomedY() < minY {
        y += 1
        camera.Center(x, y)
    }

    for camera.GetZoomedMaxY() >= float64(game.Model.CurrentMap().Height()) && camera.Y > 0 {
        y -= 1
        camera.Center(x, y)
    }

    return min(y, game.Model.CurrentMap().Height())
}

// called when a stack of the player starts to walk along its path
func (game *Game) startWalkCamera(yield coroutine.YieldFunc, player *playerlib.Player, stack *playerlib.UnitStack, mapUse *maplib.Map, getStack func(int, int) (playerlib.PathStack, bool)) {
    walkCamera = walkCameraMove{}
    if !walkCameraToDestination(player) || stack.Plane() != game.Model.Plane {
        return
    }

    x, y, found := game.walkDestination(player, stack, mapUse, getStack)
    if !found {
        return
    }

    walkCamera.Walking = true
    game.aimWalkCamera(x, y)

    if !WalkCameraWithWalk {
        game.finishWalkCamera(yield)
    }
}

// starts the move of the camera to a tile. it goes on with stepWalkCamera
func (game *Game) aimWalkCamera(x int, y int) {
    y = game.cameraRowFor(x, y)
    width := float64(game.Model.CurrentMap().Width())

    startX, startY := game.shownCameraOffset()
    // the move takes over from any pan glide or coast
    view.active = false
    coasting = false

    walkCamera.TargetX = x
    walkCamera.TargetY = y
    walkCamera.StartX = startX
    walkCamera.StartY = startY
    walkCamera.DeltaX = wrapDelta(float64(x) - startX, width)
    walkCamera.DeltaY = float64(y) - startY
    walkCamera.Frame = 0
    walkCamera.Frames = max(1, int(math.Round(CameraMoveTime * float64(ebiten.TPS()))))
    walkCamera.Active = true

    if math.Abs(walkCamera.DeltaX) < CameraMoveThreshold && math.Abs(walkCamera.DeltaY) < CameraMoveThreshold {
        // already there
        walkCamera.Frame = walkCamera.Frames
        game.stepWalkCamera()
    }
}

// one tick of the move of the camera, eased as every camera move is (cameramove.go)
func (game *Game) stepWalkCamera() {
    if !walkCamera.Active {
        return
    }

    if view.active || coasting {
        // the player pans the map by hand: the camera is theirs
        walkCamera.Active = false
        return
    }

    walkCamera.Frame += 1
    if walkCamera.Frame >= walkCamera.Frames {
        game.Camera.SetOffset(0, 0)
        game.Camera.Center(game.Model.CurrentMap().WrapX(walkCamera.TargetX), walkCamera.TargetY)
        walkCamera.Active = false
    } else {
        eased := easeInOut(float64(walkCamera.Frame) / float64(walkCamera.Frames))
        game.setCameraOffset(walkCamera.StartX + walkCamera.DeltaX * eased, walkCamera.StartY + walkCamera.DeltaY * eased)
    }

    if CameraMoveTrace != nil {
        CameraMoveTrace(game.Camera.GetOffsetX(), game.Camera.GetOffsetY())
    }
}

// lets the move of the camera come to its end
func (game *Game) finishWalkCamera(yield coroutine.YieldFunc) {
    for walkCamera.Active {
        game.stepWalkCamera()
        if walkCamera.Active && yield() != nil {
            break
        }
    }
}

// called when the stack has stopped. a stack that was stopped before the tile the camera went to
// (a fight, a lair, a wizard that is met) gets the camera
func (game *Game) endWalkCamera(yield coroutine.YieldFunc, stack *playerlib.UnitStack) {
    if !walkCamera.Walking {
        return
    }

    game.finishWalkCamera(yield)
    handed := view.active || coasting
    targetX := game.Model.CurrentMap().WrapX(walkCamera.TargetX)
    targetY := walkCamera.TargetY
    walkCamera = walkCameraMove{}

    if handed || stack == nil || stack.Plane() != game.Model.Plane {
        return
    }

    if game.Model.CurrentMap().WrapX(stack.X()) != targetX || game.cameraRowFor(stack.X(), stack.Y()) != targetY {
        game.doMoveCamera(yield, stack.X(), stack.Y())
    }
}
