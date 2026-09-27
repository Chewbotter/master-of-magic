package font

// Text with a solid border made of whole art pixels, the way the original game draws it:
// the glyphs are drawn in the border color at each offset, then once more on top in their own colors.

import (
    "image"
    "image/color"

    "github.com/hajimehoshi/ebiten/v2"
)

type OutlineStyle []image.Point

// a 1 pixel border on every side, including the corners. the resource totals on the main screen panel
var OutlineFull = OutlineStyle{
    {X: -1, Y: -1}, {X: 0, Y: -1}, {X: 1, Y: -1},
    {X: -1, Y: 0}, {X: 1, Y: 0},
    {X: -1, Y: 1}, {X: 0, Y: 1}, {X: 1, Y: 1},
}

// a 1 pixel border on the right and bottom. the gold and mana reserves on the main screen panel
var OutlineBottomRight = OutlineStyle{
    {X: 1, Y: 0}, {X: 0, Y: 1}, {X: 1, Y: 1},
}

var OutlineColor color.Color = color.Black

// like PrintOptions, but with a border instead of a drop shadow. DropShadow in options is ignored.
// offsets are in art pixels, so the border stays one art pixel wide at any scale
func (font *Font) PrintOutlined(destination *ebiten.Image, x float64, y float64, options FontOptions, style OutlineStyle, text string) {
    useOptions := options.Options
    if useOptions == nil {
        useOptions = &ebiten.DrawImageOptions{}
    }

    scale := options.Scale
    if scale == 0 {
        scale = 1
    }

    useX, useY := x * scale, y * scale

    switch options.Justify {
        case FontJustifyLeft:
        case FontJustifyCenter:
            useX -= font.MeasureTextWidth(text, scale) / 2
        case FontJustifyRight:
            useX -= font.MeasureTextWidth(text, scale)
    }

    border := useOptions.ColorScale
    border.ScaleWithColor(OutlineColor)

    for _, offset := range style {
        font.doPrint(destination, useX + float64(offset.X) * scale, useY + float64(offset.Y) * scale, scale, border, false, nil, 0, text)
    }

    font.doPrint(destination, useX, useY, scale, useOptions.ColorScale, false, nil, 0, text)
}
