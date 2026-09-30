package fonts

// What a screen needs to print its text as the original game does: the fonts as they are in the
// data, the color sets of the fonts, and the palette of the picture of the screen (the colors
// from 224 on are the picture's own). See lib/font/styled.go for the text itself and
// lib/font/colorblocks.go for the color sets.
//
// How the original sets a text up (ReMoM: MoX/src/Fonts.c):
//   Set_Font_Style*(font, set, ...)   the font, its colors from a color set, and by the name of
//                                     the function its shadow
//   Set_Font_Colors_15(font, colors)  fills the color set 15 with colors of its own and picks it.
//                                     so a later style with set 15 has THOSE colors, of any font
//   Set_Alias_Color(n)                the soft edge of the letters, color 0 of the set
//   Set_Outline_Color(n)              the color of the shadow or outline

import (
    "fmt"
    "image/color"
    "log"

    "github.com/kazzmir/master-of-magic/lib/font"
    "github.com/kazzmir/master-of-magic/lib/lbx"
)

type Original struct {
    Fonts []*font.LbxFont
    Blocks [][]int
    Palette color.Palette
}

type originalKey struct {
    Cache *lbx.LbxCache
    Picture string
}

var originals = make(map[originalKey]*Original)

// the fonts and colors for a screen whose picture is this entry of this archive. with an empty
// name the palette of the game. nil when something can not be read
func LoadOriginal(cache *lbx.LbxCache, pictureLbx string, pictureEntry int) *Original {
    if cache == nil {
        return nil
    }

    key := originalKey{Cache: cache, Picture: fmt.Sprintf("%v/%v", pictureLbx, pictureEntry)}
    out, ok := originals[key]
    if ok {
        return out
    }

    // also when it fails, so it is tried once
    originals[key] = nil

    fontsLbx, err := cache.GetLbxFile("fonts.lbx")
    if err != nil {
        log.Printf("Unable to read fonts.lbx: %v", err)
        return nil
    }

    lbxFonts, err := font.ReadFonts(fontsLbx, 0)
    if err != nil || len(lbxFonts) < 6 {
        log.Printf("Unable to read the fonts: %v", err)
        return nil
    }

    blocks, err := font.ReadColorBlocks(fontsLbx, font.GamePaletteEntry)
    if err != nil {
        log.Printf("Unable to read the color sets of the fonts: %v", err)
        return nil
    }

    palette := lbx.GetDefaultPalette()
    if pictureLbx != "" {
        pictureFile, err := cache.GetLbxFile(pictureLbx)
        if err == nil {
            own, err := pictureFile.GetPalette(pictureEntry)
            if err == nil && len(own) >= 256 {
                palette = own
            }
        }
    }
    if len(palette) < 256 {
        return nil
    }

    out = &Original{Fonts: lbxFonts, Blocks: blocks, Palette: palette}
    originals[key] = out
    return out
}

// the colors of palette indexes. an index below 0 is a pixel that is not drawn
func (original *Original) Colors(indexes ...int) []color.Color {
    var colors []color.Color
    for _, index := range indexes {
        if index < 0 || index >= len(original.Palette) {
            colors = append(colors, nil)
        } else {
            colors = append(colors, original.Palette[index])
        }
    }
    return colors
}

// the soft edge and ONE color for every other value of a pixel
func (original *Original) Flat(edge int, body int) []color.Color {
    indexes := []int{edge}
    for len(indexes) < font.ColorBlockSize {
        indexes = append(indexes, body)
    }
    return original.Colors(indexes...)
}

// the colors of a color set of the data, with this soft edge. an edge below 0 is the set's own
func (original *Original) Set(block int, edge int) []color.Color {
    if block < 0 || block >= len(original.Blocks) {
        return nil
    }

    indexes := append([]int{}, original.Blocks[block]...)
    if edge >= 0 {
        indexes[0] = edge
    }
    return original.Colors(indexes...)
}

// a text style. a shadow index below 0 is no shadow
func (original *Original) Style(fontIndex int, colors []color.Color, shadow font.ShadowMode, shadowIndex int) *font.StyledFont {
    if fontIndex < 0 || fontIndex >= len(original.Fonts) {
        return nil
    }

    if shadowIndex < 0 || shadow == font.ShadowNone {
        return font.MakeStyledFont(original.Fonts[fontIndex], colors, font.ShadowNone, nil)
    }
    return font.MakeStyledFont(original.Fonts[fontIndex], colors, shadow, original.Palette[shadowIndex])
}
