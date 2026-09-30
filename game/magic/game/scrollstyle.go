package game

// The text of the scroll of events (Info, Chancellor, and at the start of a turn), matched to the
// original game.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/REPORT.c:
// Chancellor_Screen__WIP, Chancellor_Screen_Scroll_Draw__WIP). The code is ours. Upstream had
// both in flat reds of its own without the soft edge of the letters, and the lines in the middle
// of the scroll.
//
//   a heading: font 4, letters 47 with soft edge 240, its middle at 165 (NO EVENTS THIS MONTH
//     at 158); the first line 11 under it
//   a line: font 1, letters 55 with soft edge 247, spaces 2 wide, from x 70; lines 7 apart
//   no shadow
//   the text is in the middle of the scroll from top to bottom: around y 101
//
// ONE scroll has all events of a turn under their headings, see scrollevents.go. How the scroll
// opens and closes is still the fork's.

import (
    fontslib "github.com/kazzmir/master-of-magic/game/magic/fonts"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/lib/font"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: the text of upstream
var OriginalScrollText = true

const scrollLbx = "scroll.lbx"
const scrollPaperIndex = 5

const (
    scrollHeadingX = 165
    scrollNoEventsX = 158
    scrollLineX = 70
    // lines that are longer are broken, the original does not break them
    scrollLineWidth = 194
    scrollHeadingStep = 11
    scrollLineStep = 7
    // a heading counts this much for the height of the text
    scrollHeadingHeight = 12
    scrollMiddleY = 101
    // the text is at most this high
    scrollTextMost = 127
)

const scrollNoEvents = "NO EVENTS THIS MONTH"

type scrollStyle struct {
    Heading *font.StyledFont
    Line *font.StyledFont
}

func (game *Game) getScrollStyle() *scrollStyle {
    if !OriginalScrollText {
        return nil
    }

    original := fontslib.LoadOriginal(game.Cache, scrollLbx, scrollPaperIndex)
    if original == nil {
        return nil
    }

    style := &scrollStyle{
        Heading: original.Style(4, original.Flat(240, 47), font.ShadowNone, -1),
        Line: original.Style(1, original.Flat(247, 55), font.ShadowNone, -1),
    }
    if style.Heading == nil || style.Line == nil {
        return nil
    }
    style.Line.SpaceWidth = 2

    return style
}

// the lines of a text of the scroll
func (style *scrollStyle) lines(text string) []string {
    if text == "" {
        return nil
    }
    return style.Line.Wrap(text, scrollLineWidth, 1)
}

// how high heading and lines are together
func (style *scrollStyle) height(title string, lines []string) int {
    height := len(lines) * scrollLineStep
    if title != "" {
        height += scrollHeadingHeight
    }
    return min(height, scrollTextMost)
}

func (style *scrollStyle) draw(screen *ebiten.Image, colors ebiten.ColorScale, title string, lines []string) {
    var options ebiten.DrawImageOptions
    options.ColorScale = colors

    y := scrollMiddleY - style.height(title, lines) / 2
    if title != "" {
        x := scrollHeadingX
        if title == scrollNoEvents {
            x = scrollNoEventsX
        }
        style.Heading.Print(screen, x, y, font.FontOptions{Options: &options, Scale: scale.ScaleAmount, Justify: font.FontJustifyCenter}, title)
        y += scrollHeadingStep
    }

    for _, line := range lines {
        style.Line.Print(screen, scrollLineX, y, font.FontOptions{Options: &options, Scale: scale.ScaleAmount}, line)
        y += scrollLineStep
    }
}
