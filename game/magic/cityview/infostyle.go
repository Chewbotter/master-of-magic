package cityview

// The text of the screen of a city, matched to the original game: the race and the population at
// the top, and in the field of what the city builds the turns it takes, the name of the thing and
// the description of the settings Housing and Trade Goods.
//
// The numbers are the original's: fonts, palette indexes and places, read from the ReMoM project's
// reconstruction (MoM/src/CityScr.c: City_Screen_Draw, City_Screen_Draw2) and checked against the
// game's palette. The code is ours. Upstream printed all of it with a drop shadow of less than an
// art pixel, at places of its own. The headings (Resources, Buildings, ...) are painted on the
// picture of the screen.

import (
    "fmt"
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

// the field of what the city builds
const (
    // the number of turns ends here, the word starts after it
    cityTurnsNumberX = 286
    cityTurnsWordX = 291
    cityTurnsY = 140

    // the name of what is built: lines of this width from this column, in the middle of it, the
    // last line in this row
    cityProductX = 216
    cityProductWidth = 44
    cityProductY = 179
    // a name wider than this is printed in the smaller font
    cityProductWide = 42

    // the description of a setting
    cityDescriptionX = 262
    cityDescriptionY = 159
    cityDescriptionWidth = 50
)

const cityProductFontIndex = 1
const citySmallFontIndex = 0
// the name and the description: white letters with this soft edge, and a shadow above and left
const cityProductEdge = 26
const cityProductShadow = 4
const cityProductSpace = 1

// what the original says about the settings
const cityHousingText = "Increases population growth rate."
const cityTradeGoodsText = "Converts production to gold."

type cityInfoStyle struct {
    Race *font.StyledFont
    Population *font.StyledFont
    // the name of what is built, and the same in the smaller font for long names
    Product *font.StyledFont
    ProductSmall *font.StyledFont
    // the description of a setting
    Description *font.StyledFont
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

    white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
    makeWhite := func(fontIndex int) *font.StyledFont {
        colors := []color.Color{palette[cityProductEdge]}
        for len(colors) < 16 {
            colors = append(colors, white)
        }
        styled := font.MakeStyledFont(lbxFonts[fontIndex], colors, font.ShadowUp, palette[cityProductShadow])
        styled.SpaceWidth = cityProductSpace
        return styled
    }

    style = &cityInfoStyle{
        Race: make(),
        Population: make(),
        Product: makeWhite(cityProductFontIndex),
        ProductSmall: makeWhite(citySmallFontIndex),
        Description: makeWhite(citySmallFontIndex),
    }
    style.Population.SpaceWidth = cityPopulationSpace
    // the description has the spaces of its font
    style.Description.SpaceWidth = 0

    cityInfoStyles[cache] = style
    return style
}

func (style *cityInfoStyle) draw(screen *ebiten.Image, race string, population string) {
    style.Race.Print(screen, cityRaceX, cityInfoY, font.FontOptions{Scale: scale.ScaleAmount}, race)
    style.Population.Print(screen, cityPopulationRightX, cityInfoY, font.FontOptions{Scale: scale.ScaleAmount, Justify: font.FontJustifyRight}, population)
}

// the turns it takes to build what the city builds
func (style *cityInfoStyle) drawTurns(screen *ebiten.Image, turns int) {
    word := "Turns"
    if turns <= 1 {
        word = "Turn"
    }

    style.Race.Print(screen, cityTurnsNumberX, cityTurnsY, font.FontOptions{Scale: scale.ScaleAmount, Justify: font.FontJustifyRight}, fmt.Sprintf("%v", turns))
    style.Race.Print(screen, cityTurnsWordX, cityTurnsY, font.FontOptions{Scale: scale.ScaleAmount}, word)
}

// lines in the middle of a field that starts at x and is this wide, the first at y
func drawCentered(styled *font.StyledFont, screen *ebiten.Image, x int, y int, width int, lines []string) {
    for index, line := range lines {
        styled.Print(screen, x + width / 2, y + index * styled.Height(), font.FontOptions{Scale: scale.ScaleAmount, Justify: font.FontJustifyCenter}, line)
    }
}

// the name of what the city builds, under its picture. the last line is in the row of the original
func (style *cityInfoStyle) drawProduct(screen *ebiten.Image, name string) {
    use := style.Product
    lines := use.Wrap(name, cityProductWidth, 1)

    widest := 0
    for _, line := range lines {
        widest = max(widest, use.Width(line))
    }

    if widest > cityProductWide || len(lines) > 2 {
        use = style.ProductSmall
        lines = use.Wrap(name, cityProductWidth, 1)
    }

    drawCentered(use, screen, cityProductX, cityProductY - (len(lines) - 1) * use.Height(), cityProductWidth, lines)
}

// the description of Housing or Trade Goods, in the place of the coins
func (style *cityInfoStyle) drawDescription(screen *ebiten.Image, text string) {
    lines := style.Description.Wrap(text, cityDescriptionWidth, 1)
    drawCentered(style.Description, screen, cityDescriptionX, cityDescriptionY, cityDescriptionWidth, lines)
}
