package main

// Frames per second counter in the upper right corner of the real screen. Hidden until its key
// is pressed, and hidden again at the next start of the game (user, 2026-09-29).

import (
    "fmt"

    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    fontslib "github.com/kazzmir/master-of-magic/game/magic/fonts"
    "github.com/kazzmir/master-of-magic/lib/font"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/inpututil"
)

// shown now
var ShowFPS = false
// the key that shows and hides it, on every screen
var FPSKey = ebiten.KeyF10

func updateFPSKey() {
    if inpututil.IsKeyJustPressed(FPSKey) {
        ShowFPS = !ShowFPS
    }
}
// distance from the right and top edge of the screen, in original pixels
const FPSMarginX = 3
const FPSMarginY = 2
// font from game/magic/fonts
const FPSFontName = fontslib.SmallWhite

func (game *MagicGame) drawFPS(screen *ebiten.Image) {
    if !ShowFPS || game.Cache == nil {
        return
    }

    if game.FPSFont == nil {
        loader, err := fontslib.Loader(game.Cache)
        if err != nil {
            return
        }
        game.FPSFont = loader(FPSFontName)
        if game.FPSFont == nil {
            return
        }
    }

    // the right edge of the picture, which may sit inside margins in the window
    x := float64(display.LogicalWidth() - FPSMarginX) + float64(display.PictureX()) / scale.ScaleAmount
    y := float64(FPSMarginY) + float64(display.ContentOffsetY()) / scale.ScaleAmount
    text := fmt.Sprintf("%.0f FPS", ebiten.ActualFPS())
    game.FPSFont.PrintOptions(screen, x, y, font.FontOptions{DropShadow: true, Scale: scale.ScaleAmount, Justify: font.FontJustifyRight}, text)
}
