package mod

// The beach and the water of a coast (combat/coast.go) by landscape. The folder Shore has the
// ones of every landscape. A landscape can have its own, in a folder named after it:
//
//   environment/Shore Grass/sand 1_0.png ...     the beach of grass land
//   environment/Shore Mountain/, Shore Tundra/, Shore Desert/
//
// with the names of the folder Shore. A battle takes the sand of the folder of its landscape if
// that has any, else the one of Shore, else the game's, and the same for the water.
//
// TUNDRA IS FROZEN (user, 2026-09-29: "river and ocean water should be frozen"): its sea is ice.
// The game makes the ice from its water until Shore Tundra has painted water.

import (
    "image"
    "image/color"
)

// the landscapes that can have a beach of their own, by the names of their folders
var ShoreLandscapes = []string{"Grass", "Desert", "Mountain", "Tundra"}
const ShoreFrozen = "Tundra"

// the folder of the beach of a landscape, by the folder of the landscape: "Shore Grass", "Shore
// Grass Myrror"
func ShoreFolder(set string) string {
    if set == "" {
        return ""
    }
    return FeatureShore + " " + set
}

// the color of ice, which a pixel of water of middle brightness gets
var iceColor = color.NRGBA{R: 186, G: 212, B: 232, A: 255}
// the brightness of water that gets that color, of 255
const iceMiddle = 58.0
// how much of the differences in brightness of the water the ice keeps
const iceContrast = 0.35

// a pixel of water as a pixel of ice
func IceTint(pixel color.NRGBA) color.NRGBA {
    if pixel.A == 0 {
        return pixel
    }

    brightness := 0.3 * float64(pixel.R) + 0.59 * float64(pixel.G) + 0.11 * float64(pixel.B)
    scale := 1 + (brightness / iceMiddle - 1) * iceContrast

    channel := func(value uint8) uint8 {
        return uint8(min(max(float64(value) * scale, 0), 255))
    }
    return color.NRGBA{R: channel(iceColor.R), G: channel(iceColor.G), B: channel(iceColor.B), A: pixel.A}
}

// a tile of water as a tile of ice
func IceTile(water image.Image) *image.NRGBA {
    bounds := water.Bounds()
    out := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
    for y := range bounds.Dy() {
        for x := range bounds.Dx() {
            pixel := color.NRGBAModel.Convert(water.At(bounds.Min.X + x, bounds.Min.Y + y)).(color.NRGBA)
            out.SetNRGBA(x, y, IceTint(pixel))
        }
    }
    return out
}
