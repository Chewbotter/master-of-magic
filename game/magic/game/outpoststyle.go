package game

// The text and the houses of the window of an outpost, matched to the original game.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/Outpost.c:
// Outpost_Screen_Draw). The code is ours.
//
//   the window (backgrnd 32) at 30,50, as upstream has it
//   the title font 5 in its color set 5 with soft edge 26 and a shadow below and right in 1, its
//     middle at 124,6 of the window (upstream from the left at 20,5)
//   the race font 1, letters 179 with soft edge 190, shadow below and right in 19, at 6,23
//   the ten houses from 8,31, 14 apart (upstream from 7, as far apart as a house is wide and 1)

import (
    "github.com/kazzmir/master-of-magic/lib/font"
    fontslib "github.com/kazzmir/master-of-magic/game/magic/fonts"
)

// false: the text of upstream
var OriginalOutpostText = true

const (
    outpostTitleMiddle = 124
    outpostTitleY = 6
    outpostTitleEdge = 26
    outpostTitleShadow = 1
    outpostRaceX = 6
    outpostRaceY = 23
    outpostRaceEdge = 190
    outpostRaceColor = 179
    outpostRaceShadow = 19
    outpostHouseX = 8
    outpostHouseStep = 14
)

type outpostStyle struct {
    Title *font.StyledFont
    Race *font.StyledFont
}

// nil when the text can not be made
func (game *Game) outpostStyle() *outpostStyle {
    if !OriginalOutpostText {
        return nil
    }

    original := fontslib.LoadOriginal(game.Cache, "backgrnd.lbx", 32)
    if original == nil {
        return nil
    }

    style := &outpostStyle{
        Title: original.Style(5, original.Set(5, outpostTitleEdge), font.ShadowDown, outpostTitleShadow),
        Race: original.Style(1, original.Flat(outpostRaceEdge, outpostRaceColor), font.ShadowDown, outpostRaceShadow),
    }
    if style.Title == nil || style.Race == nil {
        return nil
    }
    return style
}
