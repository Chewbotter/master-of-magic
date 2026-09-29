package cityview

// The race and the population at the top of the screen of a city, matched to the original game.
//
// The numbers are the original's: font, palette indexes and places, read from the ReMoM project's
// reconstruction (MoM/src/CityScr.c: City_Screen_Draw2) and checked against the game's palette.
// The code is ours. Upstream printed both with a drop shadow of less than an art pixel.

import (
    "image/color"
    "log"

    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/lib/font"
    "github.com/kazzmir/master-of-magic/lib/lbx"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: the text of upstream
var OriginalCityInfo = true

const cityInfoFontIndex = 1
// palette indexes: the soft edge of the letters, the letters, their shadow below and right
var cityInfoColors = []int{190, 179}
const cityInfoShadow = 19
// the space between words of the population, in art pixels
const cityPopulationSpace = 1

// places, in original pixels
const (
    // the name of the city has its middle here
    cityTitleMiddleX = 105
    cityTitleY = 3
    cityRaceX = 6
    cityInfoY = 19
    // the population ends here
    cityPopulationRightX = 208
)

type cityInfoStyle struct {
    Race *font.StyledFont
    Population *font.StyledFont
}

var cityInfoStyles = make(map[*lbx.LbxCache]*cityInfoStyle)

// the styles of the text, nil when they can not be made. then the text of upstream is printed
func getCityInfoStyle(cache *lbx.LbxCache) *cityInfoStyle {
    if !OriginalCityInfo || cache == nil {
        return nil
    }

    style, ok := cityInfoStyles[cache]
    if ok {
        return style
    }

    // also when it fails, so it is tried once
    cityInfoStyles[cache] = nil

    fontsLbx, err := cache.GetLbxFile("fonts.lbx")
    if err != nil {
        log.Printf("City screen: unable to read fonts.lbx: %v", err)
        return nil
    }

    lbxFonts, err := font.ReadFonts(fontsLbx, 0)
    if err != nil || len(lbxFonts) <= cityInfoFontIndex {
        log.Printf("City screen: unable to read the fonts: %v", err)
        return nil
    }

    // the colors are below 224, which are the ones of the palette of the game
    palette := lbx.GetDefaultPalette()
    if len(palette) < 256 {
        return nil
    }

    make := func() *font.StyledFont {
        var colors []color.Color
        for _, index := range cityInfoColors {
            colors = append(colors, palette[index])
        }
        return font.MakeStyledFont(lbxFonts[cityInfoFontIndex], colors, font.ShadowDown, palette[cityInfoShadow])
    }

    style = &cityInfoStyle{
        Race: make(),
        Population: make(),
    }
    style.Population.SpaceWidth = cityPopulationSpace

    cityInfoStyles[cache] = style
    return style
}

func (style *cityInfoStyle) draw(screen *ebiten.Image, race string, population string) {
    style.Race.Print(screen, cityRaceX, cityInfoY, font.FontOptions{Scale: scale.ScaleAmount}, race)
    style.Population.Print(screen, cityPopulationRightX, cityInfoY, font.FontOptions{Scale: scale.ScaleAmount, Justify: font.FontJustifyRight}, population)
}
