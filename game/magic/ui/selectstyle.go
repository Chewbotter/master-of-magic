package ui

// The text of the box a choice is made in (Info, Game, the debug menu), matched to the original
// game.
//
// The facts were read from the ReMoM project's reconstruction (MoX/src/GENDRAW.c:
// Selection_Box_Draw, Selection_Box_Coords; MoM/src/AdvsrScr.c for the keys of the advisors).
// The code is ours. Upstream had the title in yellow at the left, the choices in plain black
// letters 2 from the rim, and the keys at the right rim.
//
//   the title: font 4, in the middle over the box at y 4, letters 178 179 180 180 with soft edge
//     20, and before it the same in 251 a pixel right and down as its shadow
//   a choice: font 4 in its color set 1 with soft edge 53, and a light edge below and right of
//     the letters in 182 (as chiseled); under the mouse soft edge 182 and the edge in 178
//   a choice starts 4 from the rim at y 2 of its button, its key 81 further right
//   the box is as wide as its widest text and 8, and in the middle of the screen
//   the buttons: there are 5 pictures of them and of their right ends, taken in turn from the
//     top; the last button of a box is always the fifth, which has the lower rim

import (
    "image/color"
    "log"

    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/lib/font"
    "github.com/kazzmir/master-of-magic/lib/lbx"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: the text of upstream
var OriginalSelectionText = true

const selectionFontIndex = 4
const selectionColorBlock = 1

// palette indexes
var selectionTitleColors = []int{20, 178, 179, 180, 180}
const selectionTitleShadow = 251
const (
    selectionEdge = 53
    selectionLight = 182
    selectionOverEdge = 182
    selectionOverLight = 178
)

// places, in art pixels
const (
    selectionTitleY = 4
    selectionTextX = 4
    selectionTextY = 2
    // the key of a choice starts here, from where the choice starts
    selectionKeyX = 81
    // the box is wider than its widest text by this
    selectionRoom = 8
    // between a name that is as long as the column of the keys is far, and its key
    selectionKeyGap = 1
)

type selectionStyle struct {
    Title *font.StyledFont
    TitleShadow *font.StyledFont
    Choice *font.StyledFont
    ChoiceOver *font.StyledFont
}

var selectionStyles = make(map[*lbx.LbxCache]*selectionStyle)

// the styles of the text, nil when they can not be made. then the text of upstream is printed
func getSelectionStyle(cache *lbx.LbxCache) *selectionStyle {
    if !OriginalSelectionText || cache == nil {
        return nil
    }

    style, ok := selectionStyles[cache]
    if ok {
        return style
    }

    // also when it fails, so it is tried once
    selectionStyles[cache] = nil

    fontsLbx, err := cache.GetLbxFile("fonts.lbx")
    if err != nil {
        log.Printf("Selection box: unable to read fonts.lbx: %v", err)
        return nil
    }

    lbxFonts, err := font.ReadFonts(fontsLbx, 0)
    if err != nil || len(lbxFonts) <= selectionFontIndex {
        log.Printf("Selection box: unable to read the fonts: %v", err)
        return nil
    }

    blocks, err := font.ReadColorBlocks(fontsLbx, font.GamePaletteEntry)
    if err != nil || len(blocks) <= selectionColorBlock {
        log.Printf("Selection box: unable to read the color sets of the fonts: %v", err)
        return nil
    }

    palette := lbx.GetDefaultPalette()
    if len(palette) < 256 {
        return nil
    }

    colorsOf := func(indexes []int, edge int) []color.Color {
        var colors []color.Color
        for place, index := range indexes {
            if place == 0 {
                index = edge
            }
            colors = append(colors, palette[index])
        }
        return colors
    }

    allOf := func(index int) []color.Color {
        var colors []color.Color
        for range font.ColorBlockSize {
            colors = append(colors, palette[index])
        }
        return colors
    }

    use := lbxFonts[selectionFontIndex]
    block := blocks[selectionColorBlock]

    style = &selectionStyle{
        Title: font.MakeStyledFont(use, colorsOf(selectionTitleColors, selectionTitleColors[0]), font.ShadowNone, nil),
        TitleShadow: font.MakeStyledFont(use, allOf(selectionTitleShadow), font.ShadowNone, nil),
        Choice: font.MakeStyledFont(use, colorsOf(block, selectionEdge), font.ShadowDown, palette[selectionLight]),
        ChoiceOver: font.MakeStyledFont(use, colorsOf(block, selectionOverEdge), font.ShadowDown, palette[selectionOverLight]),
    }

    selectionStyles[cache] = style
    return style
}

// the pictures of resource.lbx of the button of a choice and of its right end
const selectionButtonFirst = 12
const selectionEndFirst = 22
const selectionButtonKinds = 5

func selectionButton(index int, count int) (int, int) {
    kind := index % selectionButtonKinds
    if index == count - 1 {
        kind = selectionButtonKinds - 1
    }
    return selectionButtonFirst + kind, selectionEndFirst + kind
}

// how wide the buttons of a box with these texts are
func (style *selectionStyle) width(title string, choices []Selection) int {
    widest := style.Title.Width(title)
    for _, choice := range choices {
        width := style.Choice.Width(choice.Name)
        if choice.Hotkey != "" {
            // the original does not count what stands after the tab. it is counted here, so
            // nothing can reach out of the box
            width = style.keyX(choice) + style.Choice.Width(choice.Hotkey)
        }
        widest = max(widest, width)
    }

    return widest + selectionRoom
}

// where the key of a choice starts, from where the choice starts
func (style *selectionStyle) keyX(choice Selection) int {
    column := selectionKeyX
    if choice.HotkeyX > 0 {
        column = choice.HotkeyX
    }
    // a name that reaches the column of the keys has its key right after it
    return max(column, style.Choice.Width(choice.Name) + selectionKeyGap)
}

// the title, in the middle of a box from left to right
func (style *selectionStyle) drawTitle(screen *ebiten.Image, left int, right int, top int, options ebiten.DrawImageOptions, title string) {
    middle := (left + right) / 2
    print := font.FontOptions{Options: &options, Scale: scale.ScaleAmount, Justify: font.FontJustifyCenter}
    style.TitleShadow.Print(screen, middle + 1, top + selectionTitleY + 1, print, title)
    style.Title.Print(screen, middle, top + selectionTitleY, print, title)
}

// a choice on its button, whose upper left corner is given
func (style *selectionStyle) drawChoice(screen *ebiten.Image, x int, y int, options ebiten.DrawImageOptions, choice Selection, over bool) {
    use := style.Choice
    if over {
        use = style.ChoiceOver
    }

    print := font.FontOptions{Options: &options, Scale: scale.ScaleAmount}
    use.Print(screen, x + selectionTextX, y + selectionTextY, print, choice.Name)
    if choice.Hotkey != "" {
        use.Print(screen, x + selectionTextX + style.keyX(choice), y + selectionTextY, print, choice.Hotkey)
    }
}
