package units

import (
    "image"
    "image/color"
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
)

func greensPalette() color.Palette {
    palette := make(color.Palette, 256)
    for index := range palette {
        palette[index] = color.RGBA{A: 0xff}
    }
    palette[215] = color.RGBA{R: 116, G: 228, B: 112, A: 0xff}
    palette[216] = color.RGBA{R: 0, G: 188, B: 0, A: 0xff}
    palette[217] = color.RGBA{R: 0, G: 164, B: 0, A: 0xff}
    palette[218] = color.RGBA{R: 0, G: 124, B: 0, A: 0xff}
    return palette
}

// four greens of a picture give four tones of the color of any wizard, from light to dark
func TestBannerTonesAreApart(test *testing.T) {
    for _, banner := range []data.BannerType{data.BannerBlue, data.BannerGreen, data.BannerPurple, data.BannerRed, data.BannerYellow, data.BannerBrown} {
        picture := image.NewPaletted(image.Rect(0, 0, 1, 1), greensPalette())
        colored, ok := MakeUpdateUnitColorsFunc(banner)(picture).(*image.Paletted)
        if !ok {
            test.Fatalf("%v: the picture is not one with a palette any more", banner)
        }

        last := 256 * 3
        for index := BannerColorFirst; index < BannerColorFirst + BannerColorCount; index++ {
            red, green, blue, _ := colored.Palette[index].RGBA()
            sum := int(red >> 8) + int(green >> 8) + int(blue >> 8)
            if sum >= last {
                test.Errorf("%v: color %v is not darker than the one before it: %v", banner, index, colored.Palette[index])
            }
            last = sum
        }
    }
}

// the greens of the game stay close to what they are for the green wizard
func TestBannerTonesKeepGreen(test *testing.T) {
    picture := image.NewPaletted(image.Rect(0, 0, 1, 1), greensPalette())
    colored := MakeUpdateUnitColorsFunc(data.BannerGreen)(picture).(*image.Paletted)
    original := greensPalette()

    for index := BannerColorFirst; index < BannerColorFirst + BannerColorCount; index++ {
        red, green, blue, _ := colored.Palette[index].RGBA()
        wantRed, wantGreen, wantBlue, _ := original[index].RGBA()
        for _, pair := range [][2]uint32{{red, wantRed}, {green, wantGreen}, {blue, wantBlue}} {
            difference := int(pair[0] >> 8) - int(pair[1] >> 8)
            if difference < -6 || difference > 6 {
                test.Errorf("color %v of the green wizard is %v, the game has %v", index, colored.Palette[index], original[index])
                break
            }
        }
    }
}
