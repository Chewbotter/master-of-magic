package game

// Debug button in the upper left of the world map, and the options it opens.
// In widescreen the button sits in the empty stretch of the top bar. Without widescreen
// there is no free space, so it sits in the top left corner of the map instead.

import (
    "image"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    fontslib "github.com/kazzmir/master-of-magic/game/magic/fonts"
    uilib "github.com/kazzmir/master-of-magic/game/magic/ui"
    "github.com/kazzmir/master-of-magic/lib/font"

    "github.com/hajimehoshi/ebiten/v2"
)

const ShowDebugButton = true
const DebugButtonLabel = "Debug"

// position of the button text in widescreen, centered in the empty stretch of the top bar between
// the bar's left border and the Game button. original pixels
const DebugButtonBarY = 7
// left edge of the Game button in the hud
const HudGameButtonX = 7
// position without widescreen, or when the stretch of bar is too short for the button
const DebugButtonMapX = 4
const DebugButtonMapY = 22
// extra clickable pixels around the text
const DebugButtonPadding = 2
// top left corner of the debug menu. in widescreen, menus are drawn in the middle of the screen,
// so this is relative to the middle 320 pixels
const DebugMenuX = 4
const DebugMenuY = 22
// space between an option and its On or Off, the menu has no gap of its own
const DebugMenuStateGap = "   "

// debug options, for this session only

// draw the world map as if every tile were explored and visible. nothing about the game changes
var DebugRevealAll = false

var debugButtons = map[*Game]*uilib.UIElement{}
var debugFont *font.Font
var debugFontHighlight *font.Font

func (game *Game) loadDebugFonts() bool {
    if debugFont != nil {
        return true
    }

    loader, err := fontslib.Loader(game.Cache)
    if err != nil {
        return false
    }

    debugFont = loader(fontslib.SmallYellow)
    debugFontHighlight = loader(fontslib.SmallWhite)
    return debugFont != nil && debugFontHighlight != nil
}

// where the button is on the screen, in original pixels
func (game *Game) debugButtonScreenRect() image.Rectangle {
    if !game.loadDebugFonts() {
        return image.Rectangle{}
    }

    width := int(debugFont.MeasureTextWidth(DebugButtonLabel, 1))
    height := debugFont.Height()

    x, y := DebugButtonMapX, DebugButtonMapY
    extra := game.WideExtra()
    stretchLeft := HudBarBorderWidth
    stretchRight := extra + HudGameButtonX
    if stretchRight - stretchLeft >= width + DebugButtonPadding * 2 {
        x, y = (stretchLeft + stretchRight - width) / 2, DebugButtonBarY
    }

    return image.Rect(x, y, x + width, y + height).Inset(-DebugButtonPadding)
}

// the button as a hud element. its rect is kept in the hud's own coordinates, which in widescreen
// start at the left edge of the shifted hud, so the rect there is to the left of zero
func (game *Game) makeDebugButton() *uilib.UIElement {
    inside := false

    button := &uilib.UIElement{
        LeftClick: func(element *uilib.UIElement){
            game.HudUI.AddElements(game.makeDebugMenu())
        },
        Inside: func(element *uilib.UIElement, x int, y int){
            inside = true
        },
        NotInside: func(element *uilib.UIElement){
            inside = false
        },
        Draw: func(element *uilib.UIElement, screen *ebiten.Image){
            if !game.loadDebugFonts() {
                return
            }

            use := debugFont
            if inside {
                use = debugFontHighlight
            }

            rect := element.Rect.Inset(DebugButtonPadding)
            use.PrintOutlined(screen, float64(rect.Min.X), float64(rect.Min.Y), font.FontOptions{Scale: scale.ScaleAmount}, font.OutlineFull, DebugButtonLabel)
        },
    }

    debugButtons[game] = button
    game.updateDebugButtonRect()
    return button
}

func (game *Game) updateDebugButtonRect() {
    button := debugButtons[game]
    if button == nil {
        return
    }

    button.Rect = game.debugButtonScreenRect().Sub(image.Pt(game.WideExtra(), 0))
}

// in widescreen the button is outside the shifted hud, so the hud draws it on the screen directly
func (game *Game) drawDebugButtonOnScreen(screen *ebiten.Image, extra int) {
    button := debugButtons[game]
    if button == nil || button.Draw == nil {
        return
    }

    rect := button.Rect
    button.Rect = rect.Add(image.Pt(extra, 0))
    button.Draw(button, screen)
    button.Rect = rect
}

func onOffText(value bool) string {
    if value {
        return "On"
    }
    return "Off"
}

func (game *Game) makeDebugMenu() []*uilib.UIElement {
    choices := []uilib.Selection{
        uilib.Selection{
            Name: "Reveal All",
            Hotkey: DebugMenuStateGap + onOffText(DebugRevealAll),
            Action: func(){
                DebugRevealAll = !DebugRevealAll
            },
        },
    }

    return uilib.MakeSelectionUI(game.HudUI, game.Cache, &game.ImageCache, DebugMenuX, DebugMenuY, "Debug", choices, true)
}

// development: the debug menu elements, for a capture
func (game *Game) DebugMenuForCapture() []*uilib.UIElement {
    return game.makeDebugMenu()
}

var revealedFog data.FogMap

// a fog map with every tile visible, the size of the given map
func revealedFogFor(width int, height int) data.FogMap {
    if len(revealedFog) == width && (width == 0 || len(revealedFog[0]) == height) {
        return revealedFog
    }

    revealedFog = make(data.FogMap, width)
    for x := range revealedFog {
        revealedFog[x] = make([]data.FogType, height)
        for y := range revealedFog[x] {
            revealedFog[x][y] = data.FogTypeVisible
        }
    }

    return revealedFog
}
