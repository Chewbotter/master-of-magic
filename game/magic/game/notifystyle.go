package game

// The text of the box with the animal of the wizard's realm, which tells of a building that is
// finished and of an event, matched to the original game.
//
// The facts were read from the ReMoM project's reconstruction (MoX/src/GENDRAW.c: Notify2,
// Notify2_Draw; MoM/src/CityScr.c: City_Built_Building_Message; MoM/src/EVENTS.c). The code is
// ours. Upstream had the text in flat yellows of its own without the soft edge of the letters
// and without a shadow, at one place for every realm.
//
//   the text: font 4, letters 178 179 180 180 with soft edge 25, from the left in lines of 175,
//     8 under the top of the box; under it the same all in 240, a pixel lower and a pixel lower
//     and right
//   where it starts depends on the animal, which takes more or less room: chaos 78, death 79,
//     life 90, nature 74, sorcery 76
//   the box of a building is at y 60, the one of an event at y 70
//
// STILL AS UPSTREAM HAS IT: the pictures of the box. The original cuts the box off after its
// text and puts the box of the picture there, so the whole is as wide as its text; the fork's is
// as wide as its picture.

import (
    "strings"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    fontslib "github.com/kazzmir/master-of-magic/game/magic/fonts"
    "github.com/kazzmir/master-of-magic/game/magic/setup"
)

// false: the text of upstream
var OriginalNotifyText = true

const (
    notifyLbx = "resource.lbx"
    notifyIndex = 40
    notifyTextWidth = 175
    notifyTextDown = 8
    notifyBuildingY = 60
    notifyEventY = 70
    notifyShadow = 240
)

var notifyTextColors = []int{25, 178, 179, 180, 180}

var notifyTextX = map[data.MagicType]int{
    data.ChaosMagic: 78,
    data.DeathMagic: 79,
    data.LifeMagic: 90,
    data.NatureMagic: 74,
    data.SorceryMagic: 76,
}

// the text of the box, nil when it can not be made
func (game *Game) notifyText() *fontslib.Shadowed {
    if !OriginalNotifyText {
        return nil
    }

    original := fontslib.LoadOriginal(game.Cache, notifyLbx, notifyIndex)
    if original == nil {
        return nil
    }

    return original.ShadowedColors(4, notifyTextColors, notifyShadow)
}

// where the text starts for a wizard
func notifyTextStart(wizard setup.WizardCustom) int {
    x, ok := notifyTextX[wizard.MostBooks()]
    if !ok {
        // the animal of chaos is the one a wizard without books has
        return notifyTextX[data.ChaosMagic]
    }
    return x
}

// "a" or "an" before a name, as the original picks it
func articleFor(name string) string {
    if name != "" && strings.ContainsAny(name[:1], "AEIOUaeiou") {
        return "an"
    }
    return "a"
}

// the words of the original for a building that is finished
func newBuildingMessage(size string, city string, building string) string {
    // the original tells of one wall
    if building == "City Walls" {
        building = "City Wall"
    }
    return "The " + size + " of " + city + " has completed the construction of " + articleFor(building) + " " + building + "."
}
