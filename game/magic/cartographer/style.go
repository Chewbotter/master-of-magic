package cartographer

// The text of the cartographer, matched to the original game.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/AdvsrScr.c:
// Cartographer_Screen__WIP, Cartographer_Screen_Draw__WIP, Cartograph_Screen_Draw_Map__WIP;
// MoX/src/MOX_DEF.h for the colors of the banners). The code is ours. Upstream had the title and
// the names in plain black fonts of its own, the names of the legend 2 right of their flags and
// 12 apart, the name of a city with a drop shadow of less than an art pixel, and the map 220 by
// 162 at 25,30.
//
//   the map: 4 pixels a tile, 240 by 160, at 16,27
//   the title: font 5, all of its letters in 1 with soft edge 25, an outline in 239, its middle
//     at 160, y 12
//   the legend: a flag at 260 and its name at 269, 2 lower, 9 from one to the next from y 80;
//     font 1 in its color set 1 with soft edge 238
//   the name of the city under the mouse: font 0 in the color of its banner with an outline in
//     1, its middle 2 right of the corner of its tile on the map and 6 above it

import (
    "image/color"
    "log"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/lib/font"
    "github.com/kazzmir/master-of-magic/lib/lbx"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: the text and the map of upstream
var OriginalCartographer = true

// the picture of the screen, which has the colors from 224 on
const cartographerLbx = "reload.lbx"
const cartographerIndex = 2

const (
    mapX = 16
    mapY = 27
    mapTile = 4
    mapWidth = 240
    mapHeight = 160

    titleX = 160
    titleY = 12

    legendFlagX = 260
    legendNameX = 269
    legendY = 80
    legendNameDown = 2
    legendStep = 9

    cityNameRight = 2
    cityNameUp = 6
)

const (
    titleFontIndex = 5
    titleColor = 1
    titleEdge = 25
    titleOutline = 239

    legendFontIndex = 1
    legendColorBlock = 1
    legendEdge = 238

    cityFontIndex = 0
    cityOutline = 1
)

// palette indexes of the banners on the map
var bannerIndexes = map[data.BannerType]int{
    data.BannerBlue: 172,
    data.BannerGreen: 216,
    data.BannerPurple: 123,
    data.BannerRed: 201,
    data.BannerYellow: 179,
    data.BannerBrown: 50,
}

type cartographerStyle struct {
    Title *font.StyledFont
    Legend *font.StyledFont
    City map[data.BannerType]*font.StyledFont
}

var cartographerStyles = make(map[*lbx.LbxCache]*cartographerStyle)

// the styles of the text, nil when they can not be made. then the text of upstream is printed
func getStyle(cache *lbx.LbxCache) *cartographerStyle {
    if !OriginalCartographer || cache == nil {
        return nil
    }

    style, ok := cartographerStyles[cache]
    if ok {
        return style
    }

    // also when it fails, so it is tried once
    cartographerStyles[cache] = nil

    fontsLbx, err := cache.GetLbxFile("fonts.lbx")
    if err != nil {
        log.Printf("Cartographer: unable to read fonts.lbx: %v", err)
        return nil
    }

    lbxFonts, err := font.ReadFonts(fontsLbx, 0)
    if err != nil || len(lbxFonts) <= titleFontIndex {
        log.Printf("Cartographer: unable to read the fonts: %v", err)
        return nil
    }

    blocks, err := font.ReadColorBlocks(fontsLbx, font.GamePaletteEntry)
    if err != nil || len(blocks) <= legendColorBlock {
        log.Printf("Cartographer: unable to read the color sets of the fonts: %v", err)
        return nil
    }

    pictureLbx, err := cache.GetLbxFile(cartographerLbx)
    if err != nil {
        return nil
    }
    palette, err := pictureLbx.GetPalette(cartographerIndex)
    if err != nil || len(palette) < 256 {
        palette = lbx.GetDefaultPalette()
    }
    if len(palette) < 256 {
        return nil
    }

    // the soft edge and the same color for every other value of a pixel
    flat := func(edge color.Color, body color.Color) []color.Color {
        colors := []color.Color{edge}
        for len(colors) < font.ColorBlockSize {
            colors = append(colors, body)
        }
        return colors
    }

    var legend []color.Color
    for place, index := range blocks[legendColorBlock] {
        if place == 0 {
            index = legendEdge
        }
        legend = append(legend, palette[index])
    }

    style = &cartographerStyle{
        Title: font.MakeStyledFont(lbxFonts[titleFontIndex], flat(palette[titleEdge], palette[titleColor]), font.ShadowOutline, palette[titleOutline]),
        Legend: font.MakeStyledFont(lbxFonts[legendFontIndex], legend, font.ShadowNone, nil),
        City: make(map[data.BannerType]*font.StyledFont),
    }
    style.Legend.SpaceWidth = 1

    for banner, index := range bannerIndexes {
        // the soft edge of the letters is not drawn
        style.City[banner] = font.MakeStyledFont(lbxFonts[cityFontIndex], flat(nil, palette[index]), font.ShadowOutline, palette[cityOutline])
    }

    cartographerStyles[cache] = style
    return style
}

func (style *cartographerStyle) print(options *ebiten.DrawImageOptions, justify font.FontJustify) font.FontOptions {
    var use ebiten.DrawImageOptions
    use.ColorScale = options.ColorScale
    return font.FontOptions{Options: &use, Scale: scale.ScaleAmount, Justify: justify}
}

func (style *cartographerStyle) drawTitle(screen *ebiten.Image, options *ebiten.DrawImageOptions, title string) {
    style.Title.Print(screen, titleX, titleY, style.print(options, font.FontJustifyCenter), title)
}

// the name of an entry of the legend, by its place in it from 0
func (style *cartographerStyle) drawLegend(screen *ebiten.Image, options *ebiten.DrawImageOptions, place int, name string) {
    style.Legend.Print(screen, legendNameX, legendY + place * legendStep + legendNameDown, style.print(options, font.FontJustifyLeft), name)
}

// the name of a city at a tile
func (style *cartographerStyle) drawCity(screen *ebiten.Image, options *ebiten.DrawImageOptions, banner data.BannerType, tileX int, tileY int, name string) {
    use, ok := style.City[banner]
    if !ok {
        use = style.City[data.BannerBrown]
    }
    use.Print(screen, mapX + tileX * mapTile + cityNameRight, mapY + tileY * mapTile - cityNameUp, style.print(options, font.FontJustifyCenter), name)
}
