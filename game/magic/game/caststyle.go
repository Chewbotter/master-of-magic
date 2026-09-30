package game

// The words under the mirror that shows a spell for the whole world being cast, matched to the
// original game.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/Spells137.c:
// Global_Enchantment_Animation_Draw). The code is ours.
//
//   the mirror (backgrnd 18) at 58,28, as upstream has it
//   the words font 5 in its color set 0 with an outline all around in 0 (black), their middle at
//     62,133 of the mirror: "You have completed casting..." or "<wizard> has cast...", then the
//     name of the spell at the same place

import (
    "image/color"

    "github.com/kazzmir/master-of-magic/lib/font"
    fontslib "github.com/kazzmir/master-of-magic/game/magic/fonts"
)

// false: the words of upstream
var OriginalGlobalCastText = true

const (
    globalCastX = 58
    globalCastY = 28
    globalCastTextMiddle = 62
    globalCastTextY = 133
)

// nil when the text can not be made
func (game *Game) globalCastText() *font.StyledFont {
    if !OriginalGlobalCastText {
        return nil
    }

    original := fontslib.LoadOriginal(game.Cache, "", 0)
    if original == nil || len(original.Fonts) <= 5 {
        return nil
    }
    return font.MakeStyledFont(original.Fonts[5], original.Set(0, -1), font.ShadowOutline, color.RGBA{A: 255})
}

func globalCastWords(human bool, wizard string) string {
    if human {
        return "You have completed casting..."
    }
    return wizard + " has cast..."
}
