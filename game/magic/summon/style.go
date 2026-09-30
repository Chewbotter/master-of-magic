package summon

// The places and the title of the picture of a summoning, matched to the original game.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/Spells137.c:
// Summon_Animation_Draw). The code is ours.
//
//   the frame at 30,42 (upstream 30,40); from it: the wizard at 8,3 (upstream 7,3), of which
//     nothing shows above row 8; the circle behind at 55,54 (upstream 53,54) and the one in
//     front at 64,80; what is summoned at 76 (upstream 75)
//   the title: font 4 in its color set 4 with soft edge 190, a shadow below and right in 16, its
//     middle at 90,117 of the frame
//   an item is "Item Enchanted" (upstream "Artifact Summoned")
//
// AS UPSTREAM HAS IT: how what is summoned comes up (the original moves the picture up 3 a
// frame from under row 105, upstream uncovers it from its top).

import (
    "github.com/kazzmir/master-of-magic/lib/font"
    "github.com/kazzmir/master-of-magic/lib/lbx"
    fontslib "github.com/kazzmir/master-of-magic/game/magic/fonts"
)

// false: the picture of upstream
var OriginalSummon = true

const (
    summonFrameY = 42
    summonWizardX = 8
    summonWizardTop = 8
    summonCircleX = 55
    summonSubjectX = 76
    summonTitleMiddle = 90
    summonTitleY = 117
    summonTitleEdge = 190
    summonTitleShadow = 16

    summonItemTitle = "Item Enchanted"
)

// the title, nil when it can not be made
func summonTitle(cache *lbx.LbxCache) *font.StyledFont {
    if !OriginalSummon {
        return nil
    }

    original := fontslib.LoadOriginal(cache, "", 0)
    if original == nil {
        return nil
    }
    return original.Style(4, original.Set(4, summonTitleEdge), font.ShadowDown, summonTitleShadow)
}
