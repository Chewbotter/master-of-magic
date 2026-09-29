package game

// Modern controls of the world map: the cursor is a red X over a tile the selected stack can not go
// to, and a click there does nothing. With the classic controls the whole screen flashes red after
// the click (blinkRed). see display/controls.go

import (
    "image"
    "log"

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

// sets the cursor of the world map, every tick of the player's turn
func (game *Game) updateMoveCursor(invalid bool) {
    if !invalidMoveCursor() || game.MouseData == nil {
        return
    }

    if !invalid || game.MouseData.Error == nil {
        mouse.Mouse.SetImage(game.MouseData.Normal)
        return
    }

    picture := game.MouseData.Error
    mouse.Mouse.SetImageFunc(func(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
        var use ebiten.DrawImageOptions
        use.GeoM.Translate(float64(-worldCursorHotCross.X), float64(-worldCursorHotCross.Y))
        use.GeoM.Concat(options.GeoM)
        use.ColorScale = options.ColorScale
        scale.DrawScaled(screen, picture, &use)
    })
}

// a move that can not be made: with the classic controls, and for the keys, the screen flashes
func (game *Game) invalidMove(yield coroutine.YieldFunc, byClick bool) {
    if byClick && invalidMoveCursor() {
        return
    }

    game.blinkRed(yield)
}
