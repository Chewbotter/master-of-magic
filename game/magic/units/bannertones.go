package units

// The color of a wizard on figures, unit pictures of the world map and markers.
//
// Pictures carry the color of their wizard as four greens of the palette, 215 (light and pale) to
// 218 (dark). Upstream made the four colors of another wizard by changing the color of the banner
// by a tenth at a time (util.Lighten), which for red, blue and yellow gives four colors that are
// nearly one, so the shading of the art was lost. Here every green keeps how light and how pale it
// is, and only takes the hue of the banner.

import (
    "image/color"

    "github.com/kazzmir/master-of-magic/game/magic/util"
)

// false: the colors of upstream
var BannerTones = true

// the colors of the palette that carry the color of a wizard
const BannerColorFirst = 215
const BannerColorCount = 4

// the palette with its four greens turned into tones of the color of a banner
func BannerTonesOf(palette color.Palette, banner color.RGBA) color.Palette {
    out := util.ClonePalette(palette)
    if len(out) < BannerColorFirst + BannerColorCount {
        return out
    }

    base := [3]float64{float64(banner.R), float64(banner.G), float64(banner.B)}
    strongest := max(base[0], base[1], base[2], 1)

    for index := BannerColorFirst; index < BannerColorFirst + BannerColorCount; index++ {
        red, green, blue, _ := palette[index].RGBA()
        shade := [3]float64{float64(red >> 8), float64(green >> 8), float64(blue >> 8)}
        light := max(shade[0], shade[1], shade[2])
        if light <= 0 {
            out[index] = color.RGBA{A: 0xff}
            continue
        }
        pale := min(shade[0], shade[1], shade[2]) / light

        var channels [3]uint8
        for channel := range channels {
            full := base[channel] / strongest
            channels[channel] = uint8(min(255, light * (pale + (1 - pale) * full) + 0.5))
        }
        out[index] = color.RGBA{R: channels[0], G: channels[1], B: channels[2], A: 0xff}
    }

    return out
}
