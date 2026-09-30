package cityview

// The text of the window of a city of another wizard, matched to the original game.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/City_ovr55.c:
// Enemy_City_Screen_Draw). The code is ours.
//
//   the window (reload 26) at 59,0, as upstream has it; from its corner:
//   the title font 5 in its color set 5, a shadow below and right in 0 (black), spaces 1 wide,
//     its middle at 107,3 (upstream from the left at 20,3)
//   the race and "Units" in the text of the race of the city screen (infostyle.go) at 7,19 and
//     7,43, the name of the unit under the mouse at 37,43 (upstream 6, and the name after three
//     spaces)
//   the units from 6,52, 22 apart (upstream from 8, 20 apart)

import (
    "image/color"

    "github.com/kazzmir/master-of-magic/lib/font"
    "github.com/kazzmir/master-of-magic/lib/lbx"
    fontslib "github.com/kazzmir/master-of-magic/game/magic/fonts"
)

// false: the text of upstream
var OriginalEnemyCity = true

const (
    enemyTitleMiddle = 107
    enemyTitleY = 3
    enemyTitleSpace = 1
    enemyTextX = 7
    enemyRaceY = 19
    enemyUnitsY = 43
    enemyUnitNameX = 37
    enemyUnitX = 6
    enemyUnitStep = 22
)

type enemyCityStyle struct {
    Title *font.StyledFont
    Text *font.StyledFont
}

// nil when the text can not be made
func getEnemyCityStyle(cache *lbx.LbxCache) *enemyCityStyle {
    if !OriginalEnemyCity {
        return nil
    }

    info := getCityInfoStyle(cache)
    original := fontslib.LoadOriginal(cache, "reload.lbx", 26)
    if info == nil || original == nil || len(original.Fonts) <= 5 {
        return nil
    }

    title := font.MakeStyledFont(original.Fonts[5], original.Set(5, -1), font.ShadowDown, color.RGBA{A: 255})
    if title == nil {
        return nil
    }
    title.SpaceWidth = enemyTitleSpace

    return &enemyCityStyle{Title: title, Text: info.Race}
}
