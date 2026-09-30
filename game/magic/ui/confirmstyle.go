package ui

// The text of the box that asks yes or no, matched to the original game. The Grand Vizier asks
// with it, and so does every other question of the game.
//
// The facts were read from the ReMoM project's reconstruction (MoX/src/GENDRAW.c:
// Confirmation_Box, Confirmation_Box_Draw). The code is ours. Upstream had the text in yellow
// with a drop shadow of less than an art pixel, in the middle of the box or from 15 of its left,
// the box at 67,68 whatever its height.
//
//   the box: at x 68, and as far from the top of the screen as from its bottom
//   the text: font 4 in its color set 4 with soft edge 18, from the left, in lines of 166, 9 from
//     the left and the top of the box; under it the same text all in 17, a pixel lower and once
//     more a pixel lower and right, as its shadow
//   under the text: 10 to the lower part of the box, 15 to the buttons; No 18 and Yes 101 from
//     the left

import (
    "github.com/kazzmir/master-of-magic/game/magic/data"
    fontslib "github.com/kazzmir/master-of-magic/game/magic/fonts"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/lib/font"
    "github.com/kazzmir/master-of-magic/lib/lbx"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: the text of upstream
var OriginalConfirmText = true

const (
    confirmBoxX = 68
    confirmTextX = 9
    confirmTextY = 9
    confirmTextWidth = 166
    // the box is this much higher than its text
    confirmBoxRoom = 34
    // from the top of the box: the rows of its upper part that are drawn, its lower part, the buttons
    confirmTopRows = 12
    confirmBottomY = 10
    confirmButtonsY = 15
)

const (
    confirmFontIndex = 4
    confirmColorBlock = 4
    confirmEdge = 18
    confirmShadow = 17
)

type confirmStyle struct {
    Text *font.StyledFont
    Shadow *font.StyledFont
}

var confirmStyles = make(map[*lbx.LbxCache]*confirmStyle)

func getConfirmStyle(cache *lbx.LbxCache) *confirmStyle {
    if !OriginalConfirmText || cache == nil {
        return nil
    }

    style, ok := confirmStyles[cache]
    if ok {
        return style
    }
    confirmStyles[cache] = nil

    original := fontslib.LoadOriginal(cache, "", 0)
    if original == nil {
        return nil
    }

    style = &confirmStyle{
        Text: original.Style(confirmFontIndex, original.Set(confirmColorBlock, confirmEdge), font.ShadowNone, -1),
        Shadow: original.Style(confirmFontIndex, original.Flat(confirmShadow, confirmShadow), font.ShadowNone, -1),
    }
    if style.Text == nil || style.Shadow == nil {
        return nil
    }

    confirmStyles[cache] = style
    return style
}

// the text in its lines, and how high they are together
func (style *confirmStyle) lines(message string) ([]string, int) {
    lines := style.Text.Wrap(message, confirmTextWidth, 1)
    return lines, len(lines) * style.Text.LineHeight()
}

// where the box is from the top of the screen
func confirmBoxY(textHeight int) int {
    return (data.ScreenHeight - (textHeight + confirmBoxRoom)) / 2
}

func (style *confirmStyle) draw(screen *ebiten.Image, x int, y int, alpha float32, lines []string) {
    var options ebiten.DrawImageOptions
    options.ColorScale.ScaleAlpha(alpha)
    print := font.FontOptions{Options: &options, Scale: scale.ScaleAmount}

    for index, line := range lines {
        lineY := y + confirmTextY + index * style.Text.LineHeight()
        style.Shadow.Print(screen, x + confirmTextX + 1, lineY + 1, print, line)
        style.Shadow.Print(screen, x + confirmTextX, lineY + 1, print, line)
        style.Text.Print(screen, x + confirmTextX, lineY, print, line)
    }
}
