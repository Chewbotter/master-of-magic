package mirror

// The text of the mirror (Info, Mirror), matched to the original game.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/WIZVIEW.c: Mirror_Screen,
// Mirror_Screen_Draw) and the color sets of the fonts from the game's data. The code is ours.
// Upstream printed everything with a drop shadow of less than an art pixel, in fonts and at
// places of its own, the panel at 50,1.
//
//   the panel is at 75,2. from its corner:
//   the name: font 4, letters 178 to 182 with soft edge 150, its middle at 114, y 11
//   gold and mana: the number in font 1 ending on 31 and on 159, y 75; GP and MP in font 0 from
//     34 and from 162
//   the books: 8 apart, their middle at 90, y 66
//   fame: font 2, the number ending on 79, the word from 85, y 96
//   the abilities: font 1, from 13,111 (109 with more than 5), in lines of 155 that are 7 apart,
//     every ability with its comma or "and" a piece that is not broken, a full stop at the end
//   "Heroes": font 2 from 74,131; the heroes in font 1 from 13,142, 8 apart
//   all text in the colors of the name and with a shadow below and right in 234. the original
//     names the color set 15 for the fonts 0, 1 and 2: that is no set of the data, it is the one
//     Set_Font_Colors_15 has filled last, here with the colors of the name

import (
    "fmt"
    "image/color"
    "log"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/lib/font"
    "github.com/kazzmir/master-of-magic/lib/lbx"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: the text of upstream
var OriginalMirror = true

// the picture of the panel, which has the colors from 224 on
const mirrorBackgroundLbx = "backgrnd.lbx"
const mirrorBackgroundIndex = 4

// where the panel is on the screen
const mirrorX = 75
const mirrorY = 2

// palette indexes
var mirrorNameColors = []int{150, 178, 179, 180, 181, 182}
const mirrorShadow = 234

const mirrorNameFont = 4

// places from the corner of the panel
const (
    mirrorNameX = 114
    mirrorNameY = 11

    mirrorGoldX = 31
    mirrorGoldWordX = 34
    mirrorManaX = 159
    mirrorManaWordX = 162
    mirrorReserveY = 75

    mirrorBooksMiddle = 90
    mirrorBooksY = 66
    mirrorBookStep = 8

    mirrorFameX = 79
    mirrorFameWordX = 85
    mirrorFameY = 96

    mirrorAbilitiesX = 13
    mirrorAbilitiesY = 111
    // with more abilities than this they start higher
    mirrorAbilitiesMany = 5
    mirrorAbilitiesManyUp = 2
    mirrorAbilitiesWidth = 155
    mirrorAbilitiesLine = 7
    // between two pieces of the list
    mirrorAbilitiesGap = 1

    mirrorHeroesX = 74
    mirrorHeroesY = 131
    mirrorHeroX = 13
    mirrorHeroY = 142
    mirrorHeroStep = 8
)

type mirrorStyle struct {
    Name *font.StyledFont
    // fonts 0, 1 and 2
    Tiny *font.StyledFont
    Small *font.StyledFont
    Medium *font.StyledFont
}

var mirrorStyles = make(map[*lbx.LbxCache]*mirrorStyle)

// the styles of the text, nil when they can not be made. then the text of upstream is printed
func getMirrorStyle(cache *lbx.LbxCache) *mirrorStyle {
    if !OriginalMirror || cache == nil {
        return nil
    }

    style, ok := mirrorStyles[cache]
    if ok {
        return style
    }

    // also when it fails, so it is tried once
    mirrorStyles[cache] = nil

    fontsLbx, err := cache.GetLbxFile("fonts.lbx")
    if err != nil {
        log.Printf("Mirror: unable to read fonts.lbx: %v", err)
        return nil
    }

    lbxFonts, err := font.ReadFonts(fontsLbx, 0)
    if err != nil || len(lbxFonts) <= mirrorNameFont {
        log.Printf("Mirror: unable to read the fonts: %v", err)
        return nil
    }

    backgroundLbx, err := cache.GetLbxFile(mirrorBackgroundLbx)
    if err != nil {
        return nil
    }
    palette, err := backgroundLbx.GetPalette(mirrorBackgroundIndex)
    if err != nil || len(palette) < 256 {
        palette = lbx.GetDefaultPalette()
    }
    if len(palette) < 256 {
        return nil
    }

    colorsOf := func(indexes []int) []color.Color {
        var colors []color.Color
        for _, index := range indexes {
            colors = append(colors, palette[index])
        }
        return colors
    }

    make := func(fontIndex int, indexes []int) *font.StyledFont {
        return font.MakeStyledFont(lbxFonts[fontIndex], colorsOf(indexes), font.ShadowDown, palette[mirrorShadow])
    }

    style = &mirrorStyle{
        Name: make(mirrorNameFont, mirrorNameColors),
        Tiny: make(0, mirrorNameColors),
        Small: make(1, mirrorNameColors),
        Medium: make(2, mirrorNameColors),
    }

    mirrorStyles[cache] = style
    return style
}

// the pieces of the list of abilities: every ability with what follows it
func abilityPieces(abilities []data.Retort) []string {
    var pieces []string
    for index, ability := range abilities {
        piece := ability.String()
        switch {
            case index == len(abilities) - 1:
            case index == len(abilities) - 2: piece += " and "
            default: piece += ", "
        }
        pieces = append(pieces, piece)
    }
    return pieces
}

type mirrorText struct {
    Screen *ebiten.Image
    Options font.FontOptions
    X int
    Y int
}

func (style *mirrorStyle) drawName(text mirrorText, name string) {
    options := text.Options
    options.Justify = font.FontJustifyCenter
    style.Name.Print(text.Screen, text.X + mirrorNameX, text.Y + mirrorNameY, options, name)
}

func (style *mirrorStyle) drawReserves(text mirrorText, gold int, mana int) {
    right := text.Options
    right.Justify = font.FontJustifyRight

    style.Small.Print(text.Screen, text.X + mirrorGoldX, text.Y + mirrorReserveY, right, fmt.Sprintf("%v", gold))
    style.Tiny.Print(text.Screen, text.X + mirrorGoldWordX, text.Y + mirrorReserveY, text.Options, "GP")
    style.Small.Print(text.Screen, text.X + mirrorManaX, text.Y + mirrorReserveY, right, fmt.Sprintf("%v", mana))
    style.Tiny.Print(text.Screen, text.X + mirrorManaWordX, text.Y + mirrorReserveY, text.Options, "MP")
}

func (style *mirrorStyle) drawFame(text mirrorText, fame int) {
    right := text.Options
    right.Justify = font.FontJustifyRight

    style.Medium.Print(text.Screen, text.X + mirrorFameX, text.Y + mirrorFameY, right, fmt.Sprintf("%v", fame))
    style.Medium.Print(text.Screen, text.X + mirrorFameWordX, text.Y + mirrorFameY, text.Options, "Fame")
}

func (style *mirrorStyle) drawAbilities(text mirrorText, abilities []data.Retort) {
    if len(abilities) == 0 {
        return
    }

    start := text.X + mirrorAbilitiesX
    x := start
    y := text.Y + mirrorAbilitiesY
    if len(abilities) > mirrorAbilitiesMany {
        y -= mirrorAbilitiesManyUp
    }

    for _, piece := range abilityPieces(abilities) {
        width := style.Small.Width(piece)
        if x + width > start + mirrorAbilitiesWidth {
            x = start
            y += mirrorAbilitiesLine
        }
        style.Small.Print(text.Screen, x, y, text.Options, piece)
        x += width + mirrorAbilitiesGap
    }
    style.Small.Print(text.Screen, x, y, text.Options, ".")
}

func (style *mirrorStyle) drawHeroes(text mirrorText, heroes []string) {
    style.Medium.Print(text.Screen, text.X + mirrorHeroesX, text.Y + mirrorHeroesY, text.Options, "Heroes")
    for index, hero := range heroes {
        style.Small.Print(text.Screen, text.X + mirrorHeroX, text.Y + mirrorHeroY + index * mirrorHeroStep, text.Options, hero)
    }
}

// where the first of this many books is, from the corner of the panel
func mirrorBooksX(count int) int {
    return mirrorBooksMiddle - count * mirrorBookStep / 2
}

func scaledOptions(options *ebiten.DrawImageOptions) font.FontOptions {
    return font.FontOptions{Options: options, Scale: scale.ScaleAmount}
}
