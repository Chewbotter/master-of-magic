package fonts

// Text the original prints in passes: first all in one dark color a pixel lower and right, then
// a pixel lower, then the letters (its boxes of messages do so, ReMoM MoX/src/GENDRAW.c). With a
// paragraph that is broken into lines.

import (
    "image/color"
    "strings"

    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/lib/font"

    "github.com/hajimehoshi/ebiten/v2"
)

type Shadowed struct {
    Text *font.StyledFont
    // nil for none
    Shadow *font.StyledFont
    // art pixels from a line to the next
    LineStep int
}

// text of a font with letters of a color set of the data and this soft edge, over a shadow all
// in one color. a shadow below 0 is none
func (original *Original) Shadowed(fontIndex int, block int, edge int, shadow int) *Shadowed {
    return original.shadowed(fontIndex, original.Set(block, edge), shadow)
}

// the same with letters of palette indexes of their own, the first the soft edge
func (original *Original) ShadowedColors(fontIndex int, indexes []int, shadow int) *Shadowed {
    return original.shadowed(fontIndex, original.Colors(indexes...), shadow)
}

func (original *Original) shadowed(fontIndex int, colors []color.Color, shadow int) *Shadowed {
    out := &Shadowed{}
    out.Text = original.Style(fontIndex, colors, font.ShadowNone, -1)
    if out.Text == nil {
        return nil
    }
    if shadow >= 0 {
        out.Shadow = original.Style(fontIndex, original.Flat(shadow, shadow), font.ShadowNone, -1)
    }
    out.LineStep = out.Text.LineHeight()
    return out
}

// breaks a text into lines. widthOf gives the width the line of a number has. spaces are kept
// as they are, so two spaces after a full stop stay two
func (text *Shadowed) Flow(message string, widthOf func(line int) int) []string {
    var lines []string
    for _, paragraph := range strings.Split(message, "\n") {
        line := ""
        started := false
        for _, word := range strings.Split(paragraph, " ") {
            longer := word
            if started {
                longer = line + " " + word
            }

            if started && strings.TrimSpace(line) != "" && word != "" && text.Text.Width(longer) > widthOf(len(lines)) {
                lines = append(lines, strings.TrimRight(line, " "))
                line = word
            } else {
                line = longer
            }
            started = true
        }
        lines = append(lines, strings.TrimRight(line, " "))
    }
    return lines
}

func (text *Shadowed) Wrap(message string, width int) []string {
    return text.Flow(message, func(int) int { return width })
}

func (text *Shadowed) Height(lines []string) int {
    return len(lines) * text.LineStep
}

func (text *Shadowed) Widest(lines []string) int {
    widest := 0
    for _, line := range lines {
        widest = max(widest, text.Text.Width(line))
    }
    return widest
}

// prints a line. x is where it starts, its middle or its end, by the way it is justified
func (text *Shadowed) Print(screen *ebiten.Image, x int, y int, justify font.FontJustify, alpha float32, line string) {
    var options ebiten.DrawImageOptions
    options.ColorScale.ScaleAlpha(alpha)
    print := font.FontOptions{Options: &options, Scale: scale.ScaleAmount, Justify: justify}

    if text.Shadow != nil {
        text.Shadow.Print(screen, x + 1, y + 1, print, line)
        text.Shadow.Print(screen, x, y + 1, print, line)
    }
    text.Text.Print(screen, x, y, print, line)
}

// prints lines, each where xOf says
func (text *Shadowed) PrintLines(screen *ebiten.Image, xOf func(line int) int, y int, justify font.FontJustify, alpha float32, lines []string) {
    for index, line := range lines {
        text.Print(screen, xOf(index), y + index * text.LineStep, justify, alpha, line)
    }
}
