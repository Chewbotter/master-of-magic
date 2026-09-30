package ui

// The colors and the places of the help scroll, matched to the original game. See
// MakeHelpElementSized in dialogs.go, which draws it.
//
// The facts were read from the ReMoM project's reconstruction (MoX/src/TEXTBOX.c:
// Draw_Help_Entry__WIP). The code is ours.

import (
    "image/color"

    fontslib "github.com/kazzmir/master-of-magic/game/magic/fonts"
    "github.com/kazzmir/master-of-magic/lib/font"
    "github.com/kazzmir/master-of-magic/lib/lbx"
)

// false: the scroll of upstream
var OriginalHelpText = true

// places, in art pixels. the two of the text from the left of the scroll
const (
    helpScrollX = 50
    helpTitleX = 18
    helpTextX = 20
    helpTextWidth = 180
    helpPictureGap = 4
    helpEntryBelow = 6
    helpScrollBelow = 5
)

// palette indexes: the letters and their soft edge
const (
    helpTitleColor = 47
    helpTitleEdge = 185
    helpTextColor = 55
    helpTextEdge = 184
)

type helpStyle struct {
    Title *font.Font
    Text *font.Font
}

var helpStyles = make(map[*lbx.LbxCache]*helpStyle)

func getHelpStyle(cache *lbx.LbxCache) *helpStyle {
    if !OriginalHelpText || cache == nil {
        return nil
    }

    style, ok := helpStyles[cache]
    if ok {
        return style
    }
    helpStyles[cache] = nil

    original := fontslib.LoadOriginal(cache, "", 0)
    if original == nil {
        return nil
    }

    // a letter has the soft edge as the value 0 of its pixels, which is color 1 of its picture
    palette := func(edge int, body int) color.Palette {
        out := color.Palette{color.RGBA{}, original.Palette[edge]}
        for len(out) < 256 {
            out = append(out, original.Palette[body])
        }
        return out
    }

    style = &helpStyle{
        Title: font.MakeOptimizedFontWithPalette(original.Fonts[4], palette(helpTitleEdge, helpTitleColor)),
        Text: font.MakeOptimizedFontWithPalette(original.Fonts[1], palette(helpTextEdge, helpTextColor)),
    }

    helpStyles[cache] = style
    return style
}
