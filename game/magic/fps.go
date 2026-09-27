package main

// Frames per second counter in the upper right corner of the real screen.

import (
    "fmt"

    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    fontslib "github.com/kazzmir/master-of-magic/game/magic/fonts"
    "github.com/kazzmir/master-of-magic/lib/font"

    "github.com/hajimehoshi/ebiten/v2"
)

const ShowFPS = true
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

    // the right edge of the real screen, outside of any black bars
    x := float64(display.LogicalWidth() - FPSMarginX)
    text := fmt.Sprintf("%.0f FPS", ebiten.ActualFPS())
    game.FPSFont.PrintOptions(screen, x, FPSMarginY, font.FontOptions{DropShadow: true, Scale: scale.ScaleAmount, Justify: font.FontJustifyRight}, text)
}
