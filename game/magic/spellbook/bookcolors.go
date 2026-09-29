package spellbook

// The colors of the letters of the spellbooks, matched to the original game.
//
// A letter of the game's fonts is made of pixels with values: value 0 is the soft edge of the
// letter, the values above it are its body, in some fonts in shades. The original gives every
// value a color of the palette, so the soft edges of the letters of a page are lighter than their
// bodies and the headers have shades. Upstream gave all values one color and left the soft edges
// out, which made the letters flat.
//
// The palette indexes are the original's, read from the ReMoM project's reconstruction
// (MoM/src/Spellbook.c: SmlBook_Compose, SmlBook_Draw, BigBook_Compose). The code is ours.

import (
    "image/color"

    "github.com/kazzmir/master-of-magic/lib/lbx"
)

// false: the colors of upstream
var OriginalBookColors = true

// the book for casting: the names of the spells, their costs and "Instant" (font 1)
var castTextColors = []int{184, 187}
// the headers of its pages (font 4)
var castTitleColors = []int{53, 46}

// the book of research: the headers of its pages (font 5), which have shades
var researchTitleColors = []int{184, 55, 47, 39, 46}
// the names of its spells (fonts 4 and 7): the soft edge has the color of the body
var researchNameColors = []int{186, 186}
// the descriptions and costs (fonts 0 and 6)
var researchTextColors = []int{184, 185, 186}

// the palette of a font: the first of the colors is the one of the soft edge, the last one is
// the color of every value that has none of its own. nil when the colors of upstream are wanted
func bookPalette(indexes []int) color.Palette {
    if !OriginalBookColors || len(indexes) == 0 {
        return nil
    }

    // the colors are below 224, which are the ones of the palette of the game
    game := lbx.GetDefaultPalette()
    if len(game) < 256 {
        return nil
    }

    // the pictures of the letters have the index 0 for no pixel and the index k + 1 for the value k
    out := color.Palette{color.RGBA{}}
    for value := 0; value < 16; value++ {
        index := indexes[min(value, len(indexes) - 1)]
        out = append(out, game[index])
    }

    return out
}

// the palette of the original, or the one of upstream when there is none
func bookPaletteOr(indexes []int, upstream color.Palette) color.Palette {
    palette := bookPalette(indexes)
    if palette == nil {
        return upstream
    }
    return palette
}
