package game

// Widescreen layout of the world map.
//
// The hud frame (top bar and right panel) is one piece of art, so it moves as one piece:
// it is drawn against the right edge of the wide picture, and the map grows to the right
// underneath it. Popups that open over the map are drawn in the middle of the picture.
// The map itself keeps its position, tile size and zoom, it only shows more columns.

import (
    "fmt"
    "image"
    "image/color"
    "math"
    "path/filepath"
    "runtime"
    "strings"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    uilib "github.com/kazzmir/master-of-magic/game/magic/ui"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/vector"
)

// the world map area of the original screen, in original pixels
const OverworldAreaWidth = 240
const OverworldAreaTop = 18
// tiles across the original world map area
const OverworldTilesX = 12

// development: where each drawer layer was pushed from, per game
var drawerOrigins = map[*Game]map[int]string{}

func (game *Game) noteDrawerPush() {
    _, file, line, ok := runtime.Caller(2)
    if !ok {
        return
    }
    if drawerOrigins[game] == nil {
        drawerOrigins[game] = map[int]string{}
    }
    drawerOrigins[game][len(game.Drawers) - 1] = fmt.Sprintf("%v:%v", filepath.Base(file), line)
}

// a newly pushed layer is a screen of its own until it is marked as a popup, whatever sat at its
// place in the stack before
func (game *Game) clearOverlayMark() {
    delete(overlayDrawers[game], len(game.Drawers) - 1)
}

// development: the drawer layers above the world map and where they came from
func (game *Game) DescribeDrawers() string {
    var parts []string
    for index := 1; index < len(game.Drawers); index++ {
        kind := "screen"
        if overlayDrawers[game][index] {
            kind = "popup"
        }
        parts = append(parts, fmt.Sprintf("%v from %v", kind, drawerOrigins[game][index]))
    }
    popupLayer := 0
    if game.HudUI != nil {
        popupLayer = int(game.HudUI.GetHighestLayerValue())
    }
    return fmt.Sprintf("%v layers [%v], hud popup layer %v, player %v", len(game.Drawers), strings.Join(parts, "; "), popupLayer, game.Model.CurrentPlayer)
}

// drawer layers, per game, that are small popups over the world map rather than screens of their own
var overlayDrawers = map[*Game]map[int]bool{}

// marks the drawer on top as a popup over the world map, so the map stays wide beneath it.
// returns the function that removes the mark, to be deferred before the drawer is popped
func (game *Game) markOverlayDrawer() func() {
    index := len(game.Drawers) - 1
    if overlayDrawers[game] == nil {
        overlayDrawers[game] = map[int]bool{}
    }
    overlayDrawers[game][index] = true

    return func() {
        delete(overlayDrawers[game], index)
        if len(overlayDrawers[game]) == 0 {
            delete(overlayDrawers, game)
        }
    }
}

// true while the plain world map is the screen on top, possibly with popups over it. every other screen gets black bars
func (game *Game) IsWideOverland() bool {
    if len(game.Drawers) == 0 || game.WatchMode {
        return false
    }

    for index := 1; index < len(game.Drawers); index++ {
        if !overlayDrawers[game][index] {
            return false
        }
    }

    return true
}

var overlayCanvas *ebiten.Image

// draws a popup ui in the middle of the screen, over the wide world map
func (game *Game) drawOverlayUI(ui *uilib.UI, screen *ebiten.Image) {
    extra := game.WideExtra()
    if extra == 0 {
        ui.Draw(ui, screen)
        return
    }

    canvas := getCanvas(&overlayCanvas)
    ui.Draw(ui, canvas)

    var options ebiten.DrawImageOptions
    options.GeoM.Translate(float64(scale.Scale(extra / 2)), 0)
    screen.DrawImage(canvas, &options)
}

// converts the cursor for a popup drawn by drawOverlayUI
func (game *Game) overlayCursorTransform(x int, y int) (int, int) {
    return x - scale.Scale(game.WideExtra() / 2), y
}

// extra columns the world map screen has to fill, in original pixels. 0 when not in widescreen
func (game *Game) WideExtra() int {
    if display.IsWideContentActive() && game.IsWideOverland() {
        return display.ExtraWidth()
    }

    return 0
}

// width of the visible world map area in original pixels
func (game *Game) OverworldWidth() int {
    return OverworldAreaWidth + game.WideExtra()
}

// keep the camera's idea of how many tiles are visible in step with the width of the map area
func (game *Game) updateCameraSize() {
    tileWidth := game.Model.CurrentMap().TileWidth()
    if tileWidth <= 0 {
        return
    }

    game.Camera.SizeX = max(OverworldTilesX, int(math.Ceil(float64(game.OverworldWidth()) / float64(tileWidth))))
}

// where the minimap is on the real screen, in real pixels
func (game *Game) GetMinimapScreenRect() image.Rectangle {
    return game.GetMinimapRect().Add(image.Pt(scale.Scale(game.WideExtra()), 0))
}

var hudCanvas *ebiten.Image
var popupCanvas *ebiten.Image

func getCanvas(canvas **ebiten.Image) *ebiten.Image {
    width, height := scale.Scale2(data.ScreenWidth, data.ScreenHeight)
    if *canvas == nil || (*canvas).Bounds().Dx() != width || (*canvas).Bounds().Dy() != height {
        *canvas = ebiten.NewImage(width, height)
    }

    (*canvas).Clear()
    return *canvas
}

func isBaseLayer(layer uilib.UILayer) bool {
    return layer <= 0
}

func isPopupLayer(layer uilib.UILayer) bool {
    return layer > 0
}

func (game *Game) drawHudFrame(screen *ebiten.Image) {
    var options ebiten.DrawImageOptions
    mainHud, err := game.ImageCache.GetImage("main.lbx", 0, 0)
    if err == nil {
        scale.DrawScaled(screen, mainHud, &options)
    }
}

// draws the hud of the world map. this is the Draw function of the hud ui
func (game *Game) drawHud(ui *uilib.UI, screen *ebiten.Image) {
    game.updateDebugButtonRect()
    extra := game.WideExtra()

    if extra == 0 {
        game.drawHudFrame(screen)
        ui.StandardDraw(screen)
        return
    }

    hasPopup := ui.GetHighestLayerValue() > 0

    // the frame and everything docked in it, against the right edge
    base := getCanvas(&hudCanvas)
    game.drawHudFrame(base)
    ui.DrawLayers(base, isBaseLayer)
    if !hasPopup {
        ui.RenderTooltip(base)
    }

    var baseOptions ebiten.DrawImageOptions
    baseOptions.GeoM.Translate(float64(scale.Scale(extra)), 0)
    screen.DrawImage(base, &baseOptions)

    game.drawTopBarFiller(screen, extra)
    game.drawDebugButtonOnScreen(screen, extra)

    // popups, in the middle
    if hasPopup {
        popup := getCanvas(&popupCanvas)
        ui.DrawLayers(popup, isPopupLayer)
        ui.RenderTooltip(popup)

        hudPopupBars.draw(screen, popup, extra)

        var popupOptions ebiten.DrawImageOptions
        popupOptions.GeoM.Translate(float64(scale.Scale(extra / 2)), 0)
        screen.DrawImage(popup, &popupOptions)
    } else {
        hudPopupBars = popupBars{}
    }
}

// frames between looks at the popup to see if it covers the whole original screen
const PopupBarProbeFrames = 4
// how far in from the left and right edge of the popup the looks are taken, in real pixels
const PopupBarProbeInset = 2

// the black bars beside a popup, for one kind of popup
type popupBars struct {
    alpha uint8
    counter uint64
}

// popups in the hud, and popups that push their own layer (drawCentered)
var hudPopupBars popupBars
var centeredPopupBars popupBars

// a popup that covers the whole original screen (the game menu, for one) is a screen of its own:
// it gets black bars like any other screen without a widescreen layout. a small popup leaves the map visible.
// the bars are as solid as the popup is at its left and right edge, so they fade in and out with it
func (bars *popupBars) draw(screen *ebiten.Image, popup *ebiten.Image, extra int) {
    popupBarAlpha := bars.alpha
    popupBarCounter := bars.counter
    defer func() {
        bars.alpha = popupBarAlpha
        bars.counter = popupBarCounter
    }()

    if popupBarCounter % PopupBarProbeFrames == 0 {
        bounds := popup.Bounds()
        lowest := uint32(0xffff)
        for _, x := range []int{bounds.Min.X + PopupBarProbeInset, bounds.Max.X - 1 - PopupBarProbeInset} {
            for _, y := range []int{bounds.Min.Y + bounds.Dy() / 4, bounds.Min.Y + bounds.Dy() * 3 / 4} {
                _, _, _, alpha := popup.At(x, y).RGBA()
                lowest = min(lowest, alpha)
            }
        }
        popupBarAlpha = uint8(lowest >> 8)
    }
    popupBarCounter += 1

    if popupBarAlpha == 0 {
        return
    }

    barWidth := float32(scale.Scale(extra / 2))
    height := float32(screen.Bounds().Dy())
    black := color.NRGBA{R: 0, G: 0, B: 0, A: popupBarAlpha}
    vector.FillRect(screen, 0, 0, barWidth, height, black, false)
    vector.FillRect(screen, float32(screen.Bounds().Dx()) - barWidth, 0, barWidth, height, black, false)
}

var centeredCanvas *ebiten.Image

// draws a popup that works in the original 320 wide coordinates in the middle of the wide world map,
// with black bars that fade in when it covers the whole original width. the layer that calls this
// must be marked with markOverlayDrawer
func (game *Game) drawCentered(screen *ebiten.Image, draw func(*ebiten.Image)) {
    extra := game.WideExtra()
    if extra == 0 {
        draw(screen)
        return
    }

    canvas := getCanvas(&centeredCanvas)
    draw(canvas)
    centeredPopupBars.draw(screen, canvas, extra)

    var options ebiten.DrawImageOptions
    options.GeoM.Translate(float64(scale.Scale(extra / 2)), 0)
    screen.DrawImage(canvas, &options)
}

// a map mode (surveyor, and later road building and spell targeting) draws its own copy of the hud
// frame and a panel in original coordinates. in the wide layout that goes against the right edge
// like the world map's hud, with the top bar extended to the left edge
func (game *Game) drawWideHud(ui *uilib.UI, screen *ebiten.Image) {
    extra := game.WideExtra()
    if extra == 0 {
        ui.Draw(ui, screen)
        return
    }

    canvas := getCanvas(&hudCanvas)
    ui.Draw(ui, canvas)

    var options ebiten.DrawImageOptions
    options.GeoM.Translate(float64(scale.Scale(extra)), 0)
    screen.DrawImage(canvas, &options)
    game.drawTopBarFiller(screen, extra)
}

// the cursor for a ui drawn by drawWideHud
func (game *Game) shiftedCursorTransform(x int, y int) (int, int) {
    return x - scale.Scale(game.WideExtra()), y
}

// the top bar of the hud frame, in original pixels
// height of the bar including its stone edge
const HudBarHeight = 20
// width of the stone border at the left end of the bar
const HudBarBorderWidth = 6
// a column of the bar without any button in it. it is repeated to make the bar longer
const HudBarFillerColumn = 6

// the hud frame sits against the right edge, which leaves the top bar short of the left edge.
// extend the bar to the left edge: its left border moves to the far left and an empty stretch of bar fills the rest
func (game *Game) drawTopBarFiller(screen *ebiten.Image, extra int) {
    mainHud, err := game.ImageCache.GetImage("main.lbx", 0, 0)
    if err != nil {
        return
    }

    column := mainHud.SubImage(image.Rect(HudBarFillerColumn, 0, HudBarFillerColumn + 1, HudBarHeight)).(*ebiten.Image)
    var columnOptions ebiten.DrawImageOptions
    // covers the gap and the border the frame brought along with it
    columnOptions.GeoM.Scale(float64(extra), 1)
    columnOptions.GeoM.Translate(float64(HudBarBorderWidth), 0)
    scale.DrawScaled(screen, column, &columnOptions)

    border := mainHud.SubImage(image.Rect(0, 0, HudBarBorderWidth, HudBarHeight)).(*ebiten.Image)
    var borderOptions ebiten.DrawImageOptions
    scale.DrawScaled(screen, border, &borderOptions)
}

// converts the cursor from real screen pixels to the position the hud elements were built for
func (game *Game) hudCursorTransform(x int, y int) (int, int) {
    extra := game.WideExtra()
    if extra == 0 {
        return x, y
    }

    if game.HudUI != nil && game.HudUI.GetHighestLayerValue() > 0 {
        return x - scale.Scale(extra / 2), y
    }

    return x - scale.Scale(extra), y
}
