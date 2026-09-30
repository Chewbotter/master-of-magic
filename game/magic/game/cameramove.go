package game

// The camera's own moves, such as a right click on the map or a jump to a unit.
//
// A move starts exactly where the map is drawn: after a middle mouse pan the camera sits part way
// into a tile, and the map may still be gliding behind it. The original move started from the whole
// tile, so its first frame jumped back by up to a tile, sometimes against the direction of travel.
// The move eases in and out.

import (
    "math"

    "github.com/kazzmir/master-of-magic/lib/coroutine"

    "github.com/hajimehoshi/ebiten/v2"
)

// seconds a camera move takes, whatever the distance
const CameraMoveTime = 0.3
// a move shorter than this, in tiles, happens at once
const CameraMoveThreshold = 0.01

// while a unit walks one tile: the camera stays centered on it. the offset is how far the unit is
// drawn from its new tile, in tiles, as in showMovement
func (game *Game) followMovingStack(stackX int, stackY int, offsetX float64, offsetY float64) {
    view.active = false
    coasting = false
    game.setCameraOffset(float64(stackX) + offsetX, float64(stackY) + offsetY)
    game.setCameraOffset(game.Camera.GetOffsetX(), game.clampCameraY())
    if CameraMoveTrace != nil {
        CameraMoveTrace(game.Camera.GetOffsetX(), game.Camera.GetOffsetY())
    }
}

// where the map is drawn right now, in tile offsets: the gliding view after a pan, or the camera
func (game *Game) shownCameraOffset() (float64, float64) {
    if view.active && view.game == game {
        return view.x, view.y
    }
    return game.Camera.GetOffsetX(), game.Camera.GetOffsetY()
}

// eases in and out: slow at both ends, fastest in the middle
func easeInOut(value float64) float64 {
    value = min(1, max(0, value))
    return value * value * (3 - 2 * value)
}

// development: called with the camera offsets on every frame of a move
var CameraMoveTrace func(x float64, y float64)

// true while a move of the camera of the game's own runs. the cursor of the map is not put on
// the pixels of the map then, see worldcursor.go
var cameraMoves bool

// moves the camera to center on a tile
func (game *Game) animateCameraTo(yield coroutine.YieldFunc, x int, y int) {
    mapUse := game.Model.CurrentMap()
    width := float64(mapUse.Width())

    startX, startY := game.shownCameraOffset()
    // the move takes over from any pan glide or coast
    view.active = false
    coasting = false

    deltaX := wrapDelta(float64(x) - startX, width)
    deltaY := float64(y) - startY

    // already there: no waiting
    if math.Abs(deltaX) < CameraMoveThreshold && math.Abs(deltaY) < CameraMoveThreshold {
        game.setCameraOffset(float64(x), float64(y))
        return
    }

    cameraMoves = true
    defer func(){
        cameraMoves = false
    }()

    frames := max(1, int(math.Round(CameraMoveTime * float64(ebiten.TPS()))))
    for frame := 1; frame < frames; frame++ {
        eased := easeInOut(float64(frame) / float64(frames))
        game.setCameraOffset(startX + deltaX * eased, startY + deltaY * eased)
        if CameraMoveTrace != nil {
            CameraMoveTrace(game.Camera.GetOffsetX(), game.Camera.GetOffsetY())
        }
        if yield() != nil {
            break
        }
    }

    game.Camera.SetOffset(0, 0)
    game.Camera.Center(mapUse.WrapX(x), y)
    if CameraMoveTrace != nil {
        CameraMoveTrace(game.Camera.GetOffsetX(), game.Camera.GetOffsetY())
    }
}
