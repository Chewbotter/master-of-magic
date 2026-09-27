package game

// Panning the world map by holding the middle mouse button and dragging.
// The map follows the cursor exactly, at any zoom level.

import (
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/inputmanager"
    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/inpututil"
)

const PanButton = ebiten.MouseButtonMiddle

var panning bool
var panLastX, panLastY int

// moves the camera while the pan button is held. returns true while panning
func (game *Game) doInputPan() bool {
    mouseX, mouseY := inputmanager.MousePosition()

    if !ebiten.IsMouseButtonPressed(PanButton) {
        panning = false
        return false
    }

    if !panning {
        // a drag only starts on the map, not on the panel
        if inpututil.IsMouseButtonJustPressed(PanButton) && game.InOverworldArea(mouseX, mouseY) {
            panning = true
            panLastX, panLastY = mouseX, mouseY
        }
        return panning
    }

    deltaX := mouseX - panLastX
    deltaY := mouseY - panLastY
    panLastX, panLastY = mouseX, mouseY

    if deltaX == 0 && deltaY == 0 {
        return true
    }

    game.PanBy(deltaX, deltaY)
    return true
}

// moves the map by the given number of screen pixels, as a drag of that length would
func (game *Game) PanBy(deltaX int, deltaY int) {
    mapUse := game.Model.CurrentMap()
    zoom := game.Camera.GetZoom()
    // screen pixels per tile
    tileWidth := float64(mapUse.TileWidth()) * zoom * scale.ScaleAmount
    tileHeight := float64(mapUse.TileHeight()) * zoom * scale.ScaleAmount
    if tileWidth <= 0 || tileHeight <= 0 {
        return
    }

    // the map moves with the cursor, so the camera moves the other way
    offsetX := game.Camera.GetOffsetX() - float64(deltaX) / tileWidth
    offsetY := game.Camera.GetOffsetY() - float64(deltaY) / tileHeight

    game.setCameraOffset(offsetX, offsetY)
    offsetY = game.clampCameraY()
    game.setCameraOffset(game.Camera.GetOffsetX(), offsetY)
}

// places the camera at a fractional tile position, wrapping around the map horizontally
func (game *Game) setCameraOffset(offsetX float64, offsetY float64) {
    wholeX := math.Floor(offsetX)
    wholeY := math.Floor(offsetY)

    game.Camera.X = game.Model.CurrentMap().WrapX(int(wholeX))
    game.Camera.Y = int(wholeY)
    game.Camera.SetOffset(offsetX - wholeX, offsetY - wholeY)
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
