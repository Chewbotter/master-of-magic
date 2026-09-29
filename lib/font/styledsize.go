package font

// Styled text at a smaller named size, see textsize.go and styled.go.

import (
    "math"

    "github.com/hajimehoshi/ebiten/v2"
)

// like Print, at the text size of the options (FontOptions.TextSize, or the one of UseTextSize).
// x and y are art pixels and are where text of the full size would start: smaller text starts at
// the same x and is centered on the height of the full size, so it stays in line with what is
// next to it. the letters and their shadow are on whole screen pixels, the shadow is as far from
// the letters as one pixel of a letter is wide. text is not justified
func (styled *StyledFont) PrintSized(destination *ebiten.Image, x float64, y float64, options FontOptions, text string) {
    pixelScale := options.Scale
    if pixelScale == 0 {
        pixelScale = 1
    }

    // screen pixels per pixel of a letter
    glyph := glyphScale(pixelScale, options.textSize())

    var colorScale ebiten.ColorScale
    if options.Options != nil {
        colorScale = options.Options.ColorScale
    }

    startX := math.Round(x * pixelScale)
    startY := math.Round(y * pixelScale + float64(styled.Height()) * (pixelScale - glyph) / 2)

    draw := func(useFont *Font, offsetX int, offsetY int, colors ebiten.ColorScale) {
        spacing := styled.Face.internalFont.HorizontalSpacing
        cursor := 0
        for _, c := range text {
            glyphIndex := int(c) - 32
            if glyphIndex < 0 || glyphIndex >= len(useFont.Glyphs) {
                continue
            }

            if useFont.Glyphs[glyphIndex].Width > 0 {
                var use ebiten.DrawImageOptions
                use.GeoM.Scale(glyph, glyph)
                use.GeoM.Translate(startX + float64(cursor + offsetX) * glyph, startY + float64(offsetY) * glyph)
                use.ColorScale = colors
                destination.DrawImage(useFont.getGlyphImage(glyphIndex), &use)
            }

            cursor += styled.glyphWidth(glyphIndex) + spacing
        }
    }

    if styled.ShadowColor != nil {
        shadowScale := colorScale
        shadowScale.ScaleWithColor(styled.ShadowColor)
        for _, offset := range styled.Shadow.Offsets() {
            draw(styled.Mask, offset.X, offset.Y, shadowScale)
        }
    }

    draw(styled.Face, 0, 0, colorScale)
}
