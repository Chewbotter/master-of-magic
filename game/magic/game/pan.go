package game

// Panning the world map by holding the middle mouse button and dragging.
//
// The mouse only reports whole internal pixels, 60 times a second, so moving the camera straight
// to it steps visibly. Instead the camera itself follows the mouse exactly, and the map is drawn at
// a view position that glides toward the camera every frame, through fractions of a pixel. When the
// glide settles it lands exactly on the camera, and the camera is kept on whole pixels when no drag
// is going on, so the map comes to rest crisp.

import (
    "log"
    "math"
    "time"

    "github.com/kazzmir/master-of-magic/game/magic/camera"
    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/game/magic/inputmanager"
    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/inpututil"
)

const PanButton = ebiten.MouseButtonMiddle
// how far the map moves for each pixel the mouse moves. 1 keeps the map under the cursor
const PanGain = 1.25
// seconds for the view to cover about two thirds of the distance to the camera. higher is floatier
const PanSmoothTime = 0.03
// the glide ends when the view is this close to the camera, in screen pixels
const PanSettleDistance = 0.05
// longest frame the glide accounts for, in seconds, so a stall does not throw the view
const PanMaxFrameTime = 0.1

var panning bool
var panLastX, panLastY int

// the position the map is drawn at while it glides after the camera
type panView struct {
    game *Game
    active bool
    // tile offsets the map is drawn at
    x float64
    y float64
    // camera offsets the last pan left the camera at. any other value means something else moved it
    targetX float64
    targetY float64
    lastTime time.Time
}

var view panView

// development: log the view and camera positions every frame while the view glides
var PanTrace bool

// development: a simulated drag replaces the real button, so the simulation and the real input
// check do not fight over the drag state
var panSimulated bool
var simulatedHeld bool

// development: one tick of a simulated drag, as if the mouse moved by the given screen pixels
func (game *Game) SimulatePanStep(deltaX int, deltaY int, held bool) {
    panSimulated = true
    simulatedHeld = held
    if !held {
        return
    }

    if !panning {
        game.startPan(panLastX, panLastY)
    }

    game.panTick(float64(deltaX) * PanGain, float64(deltaY) * PanGain)
}

func (game *Game) startPan(mouseX int, mouseY int) {
    panning = true
    coasting = false
    panVelocityX, panVelocityY = 0, 0
    panCarryX, panCarryY = 0, 0
    panLastX, panLastY = mouseX, mouseY
    startCursorGlide(mouseX, mouseY)
}

// movement the drag owes the map that is less than a whole screen pixel, in screen pixels
var panCarryX, panCarryY float64

// one tick of a drag that moved the map by the given screen pixels, possibly none
func (game *Game) panTick(mapX float64, mapY float64) {
    tick := panTickSeconds()
    step := 1 - math.Exp(-tick / PanVelocitySmoothTime)
    panVelocityX += (mapX / tick - panVelocityX) * step
    panVelocityY += (mapY / tick - panVelocityY) * step

    // the camera only moves in whole screen pixels during a drag, so the map is crisp whenever the
    // mouse pauses. PanGain makes fractions, which wait here until they add up to a pixel
    game.snapCameraToPixel()
    panCarryX += mapX
    panCarryY += mapY
    wholeX := math.Trunc(panCarryX)
    wholeY := math.Trunc(panCarryY)
    panCarryX -= wholeX
    panCarryY -= wholeY

    if wholeX != 0 || wholeY != 0 {
        game.PanBy(wholeX, wholeY)
    }
}

// moves the camera while the pan button is held, and lets it coast after release. returns true while panning
func (game *Game) doInputPan() bool {
    if !display.ModernControls() {
        // the classic controls have no middle mouse pan
        panning = false
        coasting = false
        game.snapZoomToLevel()
        game.snapCameraToPixel()
        return false
    }

    mouseX, mouseY := inputmanager.MousePosition()

    pressed := ebiten.IsMouseButtonPressed(PanButton)
    if panSimulated {
        pressed = simulatedHeld
    }

    if !pressed {
        if panning {
            panning = false
            game.startCoast()
        }

        if !game.coastTick() {
            game.snapZoomToLevel()
            game.snapCameraToPixel()
        }
        return false
    }

    if !panning {
        // a drag only starts on the map, not on the panel
        if inpututil.IsMouseButtonJustPressed(PanButton) && game.InOverworldArea(mouseX, mouseY) {
            game.startPan(mouseX, mouseY)
        }
        return panning
    }

    if panSimulated {
        // the simulation moves the map through SimulatePanStep
        return true
    }

    deltaX := mouseX - panLastX
    deltaY := mouseY - panLastY
    panLastX, panLastY = mouseX, mouseY

    game.panTick(float64(deltaX) * PanGain, float64(deltaY) * PanGain)
    return true
}

// screen pixels per tile at the current zoom
func (game *Game) tilePixels() (float64, float64) {
    mapUse := game.Model.CurrentMap()
    zoom := game.Camera.GetZoom()
    return float64(mapUse.TileWidth()) * zoom * scale.ScaleAmount, float64(mapUse.TileHeight()) * zoom * scale.ScaleAmount
}

// moves the map by the given number of screen pixels, as a drag of that length would
func (game *Game) PanBy(deltaX float64, deltaY float64) {
    tileWidth, tileHeight := game.tilePixels()
    if tileWidth <= 0 || tileHeight <= 0 {
        return
    }

    if !view.active || view.game != game {
        // the glide starts from where the map is drawn now
        view = panView{
            game: game,
            active: true,
            x: game.Camera.GetOffsetX(),
            y: game.Camera.GetOffsetY(),
            lastTime: time.Now(),
        }
    }

    // the map moves with the cursor, so the camera moves the other way
    offsetX := game.Camera.GetOffsetX() - deltaX / tileWidth
    offsetY := game.Camera.GetOffsetY() - deltaY / tileHeight

    game.setCameraOffset(offsetX, offsetY)
    offsetY = game.clampCameraY()
    game.setCameraOffset(game.Camera.GetOffsetX(), offsetY)

    view.targetX = game.Camera.GetOffsetX()
    view.targetY = game.Camera.GetOffsetY()
}

// places the camera at a fractional tile position, wrapping around the map horizontally
func (game *Game) setCameraOffset(offsetX float64, offsetY float64) {
    setOffsetOn(&game.Camera, offsetX, offsetY, game.Model.CurrentMap().WrapX)
}

func setOffsetOn(useCamera *camera.Camera, offsetX float64, offsetY float64, wrapX func(int) int) {
    wholeX := math.Floor(offsetX)
    wholeY := math.Floor(offsetY)

    useCamera.X = wrapX(int(wholeX))
    useCamera.Y = int(wholeY)
    useCamera.SetOffset(offsetX - wholeX, offsetY - wholeY)
}

// the same vertical limits the camera keeps when it moves to a tile. returns the allowed offset
func (game *Game) clampCameraY() float64 {
    offsetY := game.Camera.GetOffsetY()
    zoom := game.Camera.GetZoom()
    if zoom <= 0 {
        return offsetY
    }

    minY := math.Floor(-1 / zoom)
    height := float64(game.Model.CurrentMap().Height())

    if over := game.Camera.GetZoomedMaxY() - height; over > 0 {
        offsetY -= over
    }

    if under := minY - (offsetY - game.Camera.GetOffsetY() + game.Camera.GetZoomedY()); under > 0 {
        offsetY += under
    }

    return offsetY
}

// when no drag or zoom is going on, keeps the camera on whole screen pixels so the map is drawn crisp.
// a pan at one zoom level can leave the camera between pixels of another
func (game *Game) snapCameraToPixel() {
    if game.Camera.AnimatedZoom != 0 {
        return
    }

    tileWidth, tileHeight := game.tilePixels()
    if tileWidth <= 0 || tileHeight <= 0 {
        return
    }

    screenX := game.Camera.GetZoomedX() * tileWidth
    screenY := game.Camera.GetZoomedY() * tileHeight
    moveX := math.Round(screenX) - screenX
    moveY := math.Round(screenY) - screenY
    if math.Abs(moveX) < 1e-6 && math.Abs(moveY) < 1e-6 {
        return
    }

    offsetX := game.Camera.GetOffsetX() + moveX / tileWidth
    offsetY := game.Camera.GetOffsetY() + moveY / tileHeight
    game.setCameraOffset(offsetX, offsetY)

    if view.active && view.game == game {
        view.targetX = game.Camera.GetOffsetX()
        view.targetY = game.Camera.GetOffsetY()
    }
}

// the shortest way from one horizontal tile offset to another on a map that wraps around
func wrapDelta(delta float64, width float64) float64 {
    if width <= 0 {
        return delta
    }
    delta = math.Mod(delta + width / 2, width)
    if delta < 0 {
        delta += width
    }
    return delta - width / 2
}

// the camera to draw the world map with this frame: the real camera, or the gliding view after a pan
func (game *Game) displayCamera() camera.Camera {
    now := time.Now()
    useCamera := game.Camera

    if !view.active || view.game != game {
        game.roundToPixel(&useCamera)
        return useCamera
    }

    mapUse := game.Model.CurrentMap()
    width := float64(mapUse.Width())
    targetX := game.Camera.GetOffsetX()
    targetY := game.Camera.GetOffsetY()

    // something other than a pan moved the camera, such as a jump to a unit: follow it directly
    if math.Abs(wrapDelta(targetX - view.targetX, width)) > 1e-6 || math.Abs(targetY - view.targetY) > 1e-6 {
        view.active = false
        game.roundToPixel(&useCamera)
        return useCamera
    }

    frameTime := min(PanMaxFrameTime, max(0, now.Sub(view.lastTime).Seconds()))
    view.lastTime = now

    step := 1 - math.Exp(-frameTime / PanSmoothTime)
    remainingX := wrapDelta(targetX - view.x, width)
    remainingY := targetY - view.y
    view.x += remainingX * step
    view.y += remainingY * step
    remainingX -= remainingX * step
    remainingY -= remainingY * step

    tileWidth, tileHeight := game.tilePixels()
    if coasting {
        // the camera is still moving on its own
        game.setViewOn(&useCamera)
        return useCamera
    }
    if PanTrace {
        log.Printf("pan view x %.3f px, camera x %.3f px, behind by %.3f px, held %v", view.x * tileWidth, targetX * tileWidth, remainingX * tileWidth, panning)
    }
    if !panning && math.Abs(remainingX) * tileWidth < PanSettleDistance && math.Abs(remainingY) * tileHeight < PanSettleDistance {
        // settled: land exactly on the camera
        view.active = false
        game.roundToPixel(&useCamera)
        return useCamera
    }

    view.x = math.Mod(view.x, width)
    if view.x < 0 {
        view.x += width
    }

    game.setViewOn(&useCamera)
    return useCamera
}

// puts the gliding view on the camera that draws the map. at the native draw scale the view is
// rounded to whole screen pixels, so the map moves crisp; otherwise it is drawn between pixels
func (game *Game) setViewOn(useCamera *camera.Camera) {
    mapUse := game.Model.CurrentMap()
    setOffsetOn(useCamera, view.x, view.y, mapUse.WrapX)
    game.roundToPixel(useCamera)
}

// at the native draw scale, moves a camera that draws the map onto whole screen pixels
func (game *Game) roundToPixel(useCamera *camera.Camera) {
    if display.BlendBetweenPixels() {
        return
    }

    mapUse := game.Model.CurrentMap()
    tileWidth, tileHeight := game.tilePixels()
    if tileWidth <= 0 || tileHeight <= 0 {
        return
    }

    screenX := useCamera.GetZoomedX() * tileWidth
    screenY := useCamera.GetZoomedY() * tileHeight
    offsetX := useCamera.GetOffsetX() + (math.Round(screenX) - screenX) / tileWidth
    offsetY := useCamera.GetOffsetY() + (math.Round(screenY) - screenY) / tileHeight
    setOffsetOn(useCamera, offsetX, offsetY, mapUse.WrapX)
}

// a change of draw scale, such as a new window size, can leave the zoom between levels
func (game *Game) snapZoomToLevel() {
    if game.Camera.AnimatedZoom != 0 {
        return
    }

    game.Camera.Zoom = camera.NearestZoom(game.Camera.Zoom, scale.ScaleAmount)
}
