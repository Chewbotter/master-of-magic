package font

// Text drawn the way the original game draws it.
//
// The original sets a font, a set of up to 16 palette colors for the pixel values of its glyphs, a
// shadow mode and a shadow color, then prints at whole pixel positions. Pixel value 0 of a glyph is
// the "alias" pixel, the soft edge of a letter: it is drawn in the face but left out of the shadow.
// The shadow is the same text printed in one color at fixed offsets underneath the face.
//
// The offsets per mode, the alignment rules and the width rule (no letter gap after the last letter)
// are the original's, as documented by the ReMoM project (MoX/src/Fonts.c: Print_Display,
// Print_Right, Print_Centered, Get_String_Width). This is our own implementation of those rules.

import (
    "image"
    "image/color"
    "math"

    "github.com/hajimehoshi/ebiten/v2"
)

type ShadowMode int

const (
    ShadowNone ShadowMode = iota
    // below and right. text that looks chiseled into stone uses this with a light color
    ShadowDown
    // above and left
    ShadowUp
    // below and right, two pixels deep
    ShadowHeavy
    // all around
    ShadowOutline
    // all around, two pixels deep below and right
    ShadowOutlineHeavy
)

var shadowDown = []image.Point{{X: 1, Y: 1}, {X: 0, Y: 1}, {X: 1, Y: 0}}
var shadowUp = []image.Point{{X: -1, Y: 0}, {X: -1, Y: -1}, {X: 0, Y: -1}}
var shadowDeep = []image.Point{{X: 2, Y: 2}, {X: 1, Y: 2}, {X: 2, Y: 1}}
var shadowCorners = []image.Point{{X: 1, Y: -1}, {X: -1, Y: 1}}
var shadowDeepSides = []image.Point{{X: 2, Y: 0}, {X: 0, Y: 2}}

// where the shadow copies of the text go, in art pixels, in the order the original draws them
func (mode ShadowMode) Offsets() []image.Point {
    var out []image.Point

    if mode == ShadowNone {
        return nil
    }

    if mode != ShadowUp {
        out = append(out, shadowDown...)
    }
    if mode != ShadowDown && mode != ShadowHeavy {
        out = append(out, shadowUp...)
    }
    if mode == ShadowHeavy || mode == ShadowOutlineHeavy {
        out = append(out, shadowDeep...)
    }
    if mode > ShadowHeavy {
        out = append(out, shadowCorners...)
    }
    if mode == ShadowOutlineHeavy {
        out = append(out, shadowDeepSides...)
    }

    return out
}

type StyledFont struct {
    // the letters in their colors
    Face *Font
    // the letters without their alias pixels, in white, to be tinted with the shadow color
    Mask *Font
    Shadow ShadowMode
    ShadowColor color.Color
    // when above 0, the width of a space, in art pixels
    SpaceWidth int
    // art pixels between two lines of a paragraph, from the data of the font
    LineGap int
}

// from a line of a paragraph to the next, in art pixels
func (styled *StyledFont) LineHeight() int {
    return styled.Height() + styled.LineGap
}

// colors[k] is the color of glyph pixel value k. colors[0] is the alias color.
// missing colors are transparent
func MakeStyledFont(lbxFont *LbxFont, colors []color.Color, shadow ShadowMode, shadowColor color.Color) *StyledFont {
    transparent := color.RGBA{}
    white := color.RGBA{R: 255, G: 255, B: 255, A: 255}

    // the glyph images use palette index 0 for empty pixels and index k + 1 for pixel value k
    face := color.Palette{transparent}
    mask := color.Palette{transparent}
    for index := 0; index < 16; index++ {
        if index < len(colors) && colors[index] != nil {
            face = append(face, colors[index])
        } else {
            face = append(face, transparent)
        }

        if index == 0 {
            mask = append(mask, transparent)
        } else {
            mask = append(mask, white)
        }
    }

    return &StyledFont{
        Face: MakeOptimizedFontWithPalette(lbxFont, face),
        Mask: MakeOptimizedFontWithPalette(lbxFont, mask),
        Shadow: shadow,
        ShadowColor: shadowColor,
        LineGap: lbxFont.VerticalSpacing,
    }
}

func (styled *StyledFont) Height() int {
    return styled.Face.Height()
}

func (styled *StyledFont) glyphWidth(glyphIndex int) int {
    if glyphIndex == 0 && styled.SpaceWidth > 0 {
        return styled.SpaceWidth
    }
    return styled.Face.Glyphs[glyphIndex].Width
}

// the width of the text in art pixels, without a letter gap after the last letter
func (styled *StyledFont) Width(text string) int {
    width := 0
    spacing := styled.Face.internalFont.HorizontalSpacing
    for _, c := range text {
        glyphIndex := int(c) - 32
        if glyphIndex < 0 || glyphIndex >= len(styled.Face.Glyphs) {
            continue
        }
        width += styled.glyphWidth(glyphIndex) + spacing
    }

    if width > 0 {
        width -= spacing
    }

    return width
}

func (styled *StyledFont) printAt(destination *ebiten.Image, useFont *Font, x int, y int, pixelScale float64, colorScale ebiten.ColorScale, text string) {
    styled.printFineAt(destination, useFont, x, y, 0, 0, pixelScale, colorScale, text)
}

// fineX and fineY move the text by screen pixels
func (styled *StyledFont) printFineAt(destination *ebiten.Image, useFont *Font, x int, y int, fineX float64, fineY float64, pixelScale float64, colorScale ebiten.ColorScale, text string) {
    spacing := styled.Face.internalFont.HorizontalSpacing
    for _, c := range text {
        glyphIndex := int(c) - 32
        if glyphIndex < 0 || glyphIndex >= len(useFont.Glyphs) {
            continue
        }

        if useFont.Glyphs[glyphIndex].Width > 0 {
            var options ebiten.DrawImageOptions
            options.GeoM.Scale(pixelScale, pixelScale)
            options.GeoM.Translate(float64(x) * pixelScale + fineX, float64(y) * pixelScale + fineY)
            options.ColorScale = colorScale
            destination.DrawImage(useFont.getGlyphImage(glyphIndex), &options)
        }

        x += styled.glyphWidth(glyphIndex) + spacing
    }
}

// for text that moves: x and y are art pixels with fractions, and the text lands on the nearest
// whole screen pixel. the letters and their shadow stay on whole art pixels of each other
func (styled *StyledFont) PrintFine(destination *ebiten.Image, x float64, y float64, options FontOptions, text string) {
    pixelScale := options.Scale
    if pixelScale == 0 {
        pixelScale = 1
    }

    var colorScale ebiten.ColorScale
    if options.Options != nil {
        colorScale = options.Options.ColorScale
    }

    switch options.Justify {
        case FontJustifyCenter: x -= float64(styled.Width(text) / 2)
        case FontJustifyRight: x -= float64(styled.Width(text) - 1)
    }

    fineX := math.Round(x * pixelScale)
    fineY := math.Round(y * pixelScale)

    if styled.ShadowColor != nil {
        shadowScale := colorScale
        shadowScale.ScaleWithColor(styled.ShadowColor)
        for _, offset := range styled.Shadow.Offsets() {
            styled.printFineAt(destination, styled.Mask, offset.X, offset.Y, fineX, fineY, pixelScale, shadowScale, text)
        }
    }

    styled.printFineAt(destination, styled.Face, 0, 0, fineX, fineY, pixelScale, colorScale, text)
}

// x and y are whole art pixels, as in the original. Justify follows the original's rules:
// right aligned text ends on x, centered text starts at x minus half its width rounded down.
// Scale and Options.ColorScale of the options are used, the rest is ignored
func (styled *StyledFont) Print(destination *ebiten.Image, x int, y int, options FontOptions, text string) {
    pixelScale := options.Scale
    if pixelScale == 0 {
        pixelScale = 1
    }

    var colorScale ebiten.ColorScale
    if options.Options != nil {
        colorScale = options.Options.ColorScale
    }

    switch options.Justify {
        case FontJustifyCenter: x -= styled.Width(text) / 2
        case FontJustifyRight: x -= styled.Width(text) - 1
    }

    if styled.ShadowColor != nil {
        shadowScale := colorScale
        shadowScale.ScaleWithColor(styled.ShadowColor)
        for _, offset := range styled.Shadow.Offsets() {
            styled.printAt(destination, styled.Mask, x + offset.X, y + offset.Y, pixelScale, shadowScale, text)
        }
    }

    styled.printAt(destination, styled.Face, x, y, pixelScale, colorScale, text)
}
