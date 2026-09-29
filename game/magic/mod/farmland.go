package mod

// Farmland (combat/farmland.go): the fields around a town. A feature, as the coast and the river
// are. Its folder holds the tiles of its crops:
//
//   environment/Farmland/crop 1_0.png ... crop 4_0.png   in place of the game's, crop 5 on are added
//
// The game has no pictures of crops. Until there are some it makes its own from the tiles of grass
// of the landscape, in the colors of ripe grain (user, 2026-09-29: "for now let's just duplicate
// and use a golden tinted version of the grass tiles").

import (
    "image"
    "image/color"
)

const FeatureFarmland = "Farmland"

// the color of the crops, which a pixel of grass of middle brightness gets
var cropColor = color.NRGBA{R: 204, G: 168, B: 72, A: 255}
// the brightness of grass that gets that color, of 255. brighter grass gets brighter crops
const cropMiddle = 92.0
// how much of the differences in brightness of the grass the crops keep
const cropContrast = 0.8

// a pixel of grass as a pixel of crops
func CropTint(pixel color.NRGBA) color.NRGBA {
    if pixel.A == 0 {
        return pixel
    }

    brightness := 0.3 * float64(pixel.R) + 0.59 * float64(pixel.G) + 0.11 * float64(pixel.B)
    scale := 1 + (brightness / cropMiddle - 1) * cropContrast

    channel := func(value uint8) uint8 {
        return uint8(min(max(float64(value) * scale, 0), 255))
    }
    return color.NRGBA{R: channel(cropColor.R), G: channel(cropColor.G), B: channel(cropColor.B), A: pixel.A}
}

// a tile of grass as a tile of crops
func CropTile(grass image.Image) *image.NRGBA {
    bounds := grass.Bounds()
    out := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
    for y := range bounds.Dy() {
        for x := range bounds.Dx() {
            pixel := color.NRGBAModel.Convert(grass.At(bounds.Min.X + x, bounds.Min.Y + y)).(color.NRGBA)
            out.SetNRGBA(x, y, CropTint(pixel))
        }
    }
    return out
}
