package game

// Modern controls of the world map: the cursor is a red X over a tile the selected stack can not go
// to, and a click there does nothing. With the classic controls the whole screen flashes red after
// the click (blinkRed). see display/controls.go

import (
    "image"
    "log"
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/game/magic/inputmanager"
    "github.com/kazzmir/master-of-magic/game/magic/mouse"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/lib/coroutine"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: the cursor stays as it is and a click on such a tile flashes the screen, as with the classic controls
const InvalidMoveCursor = true

// the point of the X that is on the mouse position, in art pixels. the same as on the battlefield (combat/cursor.go)
var worldCursorHotCross = image.Pt(7, 7)

// what was asked last, so the path is searched once for a tile and not every tick
type moveCheck struct {
    Stack *playerlib.UnitStack
    FromX int
    FromY int
    ToX int
    ToY int
    Active int
    Turn uint64
    Plane int
}

var lastMoveCheck moveCheck
var lastMoveCheckValid bool

func invalidMoveCursor() bool {
    return InvalidMoveCursor && display.ModernControls()
}

// held with a left click: the selected stack goes to the tile, also when a stack of the player is there
func moveKeyHeld() bool {
    return ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight)
}

// true if a left click at this point of the screen would send the selected stack to a tile it can not go to
func (game *Game) overInvalidMove(player *playerlib.Player, mouseX int, mouseY int) bool {
    stack := player.SelectedStack
    if stack == nil || stack.Plane() != game.Model.Plane || !game.InOverworldArea(mouseX, mouseY) {
        return false
    }

    if game.HudUI != nil && game.HudUI.GetHighestLayerValue() > 0 {
        // a window lies over the map
        return false
    }

    mapUse := game.Model.CurrentMap()
    tileX, tileY := game.ScreenToTile(float64(mouseX), float64(mouseY))
    tileX = mapUse.WrapX(tileX)

    if tileX == stack.X() && tileY == stack.Y() {
        return false
    }

    if tileY < 0 || tileY >= mapUse.Height() {
        return true
    }

    if !moveKeyHeld() && game.stackToSelect(player, mouseX, mouseY) != nil {
        // the click selects
        return false
    }

    check := moveCheck{
        Stack: stack,
        FromX: stack.X(),
        FromY: stack.Y(),
        ToX: tileX,
        ToY: tileY,
        Active: len(stack.ActiveUnits()),
        Turn: game.Model.TurnNumber,
        Plane: int(game.Model.Plane),
    }

    if check == lastMoveCheck {
        return !lastMoveCheckValid
    }

    // the search of doPlayerUpdate
    var pathStack playerlib.PathStack = stack
    if check.Active == 0 {
        pathStack = playerlib.MakeUnitStackFromUnits(stack.Units())
    }
    _, ok := game.Model.FindPath(stack.X(), stack.Y(), tileX, tileY, player, pathStack, player.GetFog(game.Model.Plane))

    lastMoveCheck = check
    lastMoveCheckValid = ok
    if inputmanager.FixedMouse {
        // development: a capture with -capture-mouse-at
        log.Printf("move cursor: from %v,%v to %v,%v can go %v", check.FromX, check.FromY, tileX, tileY, ok)
    }
    return !ok
}

// true: over the map the cursor is as large as the map is drawn, so its pixels are the size of the
// pixels of the map at every zoom level, and lie on them. as the cursors of the battlefield do
// (combat/cursor.go). over the panel and in windows the cursor is as always
const WorldCursorScales = true

// the point of the hand that is on the mouse position: the tip of the wand
var worldCursorHotHand = image.Pt(0, 0)

// true if the mouse is over the map and no window lies over it
func (game *Game) mouseOverMap() bool {
    if game.HudUI != nil && game.HudUI.GetHighestLayerValue() > 0 {
        return false
    }

    mouseX, mouseY := inputmanager.MousePosition()
    return game.InOverworldArea(mouseX, mouseY)
}

// the cursor as it always is
func (game *Game) resetWorldCursor() {
    if game.MouseData != nil && display.ModernControls() {
        mouse.Mouse.SetImage(game.MouseData.Normal)
    }
}

// sets the cursor of the world map, every tick of the player's turn
func (game *Game) updateMoveCursor(invalid bool) {
    if !display.ModernControls() || game.MouseData == nil {
        return
    }

    picture := game.MouseData.Normal
    hot := worldCursorHotHand
    if invalid && invalidMoveCursor() && game.MouseData.Error != nil {
        picture = game.MouseData.Error
        hot = worldCursorHotCross
    }

    if WorldCursorScales && game.mouseOverMap() {
        game.setMapCursor(picture, hot)
        return
    }

    if hot == worldCursorHotHand {
        mouse.Mouse.SetImage(picture)
        return
    }

    mouse.Mouse.SetImageFunc(func(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
        var use ebiten.DrawImageOptions
        use.GeoM.Translate(float64(-hot.X), float64(-hot.Y))
        use.GeoM.Concat(options.GeoM)
        use.ColorScale = options.ColorScale
        scale.DrawScaled(screen, picture, &use)
    })
}

// where a cursor is drawn so that its pixels are pixels of the map. position is where the mouse is
// on the screen, start where a pixel of the map starts, both in pixels of the screen. pixel is the
// size of an art pixel of the map, hot the point of the cursor that is on the mouse
func cursorOnMapPixel(position float64, start float64, pixel float64, hot int) float64 {
    corner := position - float64(hot) * pixel
    return math.Round((corner - start) / pixel) * pixel + start
}

// a cursor over the map: as large as the map is drawn and on its pixels
func (game *Game) setMapCursor(picture *ebiten.Image, hot image.Point) {
    mouse.Mouse.SetImageFunc(func(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
        // screen pixels per art pixel of the map
        pixel := math.Max(1, math.Round(game.Camera.GetAnimatedZoom() * scale.ScaleAmount))

        mouseX, mouseY := options.GeoM.Apply(0, 0)
        x := mouseX * scale.ScaleAmount - float64(hot.X) * pixel
        y := mouseY * scale.ScaleAmount - float64(hot.Y) * pixel

        // while the map is panned the cursor glides between pixels on a small picture of its own
        // (mouse/smooth.go), where the pixels of the map are not known
        onScreen := screen.Bounds().Dx() > scale.Scale(mapCursorCanvasLimit)
        if onScreen {
            useCamera := game.Camera
            game.roundToPixel(&useCamera)
            startX := float64(display.ContentOffsetX()) - useCamera.GetZoomedX() * float64(game.Model.CurrentMap().TileWidth()) * pixel
            startY := float64(display.ContentOffsetY()) - useCamera.GetZoomedY() * float64(game.Model.CurrentMap().TileHeight()) * pixel
            x = cursorOnMapPixel(mouseX * scale.ScaleAmount, startX, pixel, hot.X)
            y = cursorOnMapPixel(mouseY * scale.ScaleAmount, startY, pixel, hot.Y)
        }

        var use ebiten.DrawImageOptions
        use.GeoM.Scale(pixel / scale.ScaleAmount, pixel / scale.ScaleAmount)
        use.GeoM.Translate(x / scale.ScaleAmount, y / scale.ScaleAmount)
        use.ColorScale = options.ColorScale
        scale.DrawScaled(screen, picture, &use)
    })
}

// a picture narrower than this, in art pixels, is not the screen but the one the gliding cursor is drawn on
const mapCursorCanvasLimit = 100

// a move that can not be made: with the classic controls the screen flashes. with the modern controls
// nothing happens, for a click and for the keys (user, 2026-09-28)
func (game *Game) invalidMove(yield coroutine.YieldFunc, byClick bool) {
    if invalidMoveCursor() {
        return
    }

    game.blinkRed(yield)
}
