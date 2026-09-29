package combat

// Units whose turn is over are gray while Tab is held. Not in the original (user, 2026-09-29).
//
// Tab shows the unit whose turn it is (unitmarker.go). With it every unit that has acted in this
// turn, or has no moves left, loses its color, so what can still act stands out. Nothing gets
// darker, and the black outline of the figures stays: a lighter one was tried and taken back,
// it turned whole figures into gray shapes (user, 2026-09-29).
//
// The look is a change of the palette of the pictures of the figures, as the pulse of the
// outline of the unit under the mouse is (animation.go).

import (
    "image"
    "image/color"

    "github.com/kazzmir/master-of-magic/game/magic/util"
    "github.com/kazzmir/master-of-magic/lib/fraction"
)

// false: Tab shows the unit whose turn it is and nothing else
const DoneUnitsGray = true

// how much of its color a unit whose turn is over keeps, 0 (gray) to 1
const doneSaturation = 0.15

// true if the unit has acted in this turn or can not act any more in it
func (model *CombatModel) turnIsOver(unit *ArmyUnit) bool {
    if unit.GetHealth() <= 0 {
        return false
    }

    return unit.LastTurn >= model.CurrentTurn || unit.MovesLeft.LessThanEqual(fraction.FromInt(0))
}

// true if the unit is drawn gray now
func (combat *CombatScreen) showsDone(unit *ArmyUnit) bool {
    return DoneUnitsGray && combat.TabHeld && combat.Model.turnIsOver(unit)
}

// a color with less of its color and the same lightness
func doneColor(from color.Color) color.Color {
    r, g, b, a := from.RGBA()
    if a == 0 {
        return from
    }

    red := float64(r) / float64(a)
    green := float64(g) / float64(a)
    blue := float64(b) / float64(a)
    gray := 0.299 * red + 0.587 * green + 0.114 * blue

    mix := func(part float64) uint8 {
        value := gray + (part - gray) * doneSaturation
        return uint8(min(1, max(0, value)) * 255 + 0.5)
    }

    return color.NRGBA{R: mix(red), G: mix(green), B: mix(blue), A: uint8(a >> 8)}
}

// the look of a unit whose turn is over, after the banner colors have been applied
func withDoneLook(base util.ImageTransformFunc) util.ImageTransformFunc {
    return func(original *image.Paletted) image.Image {
        out := base(original)
        paletted, ok := out.(*image.Paletted)
        if ok {
            palette := util.ClonePalette(paletted.Palette)
            for index := range palette {
                palette[index] = doneColor(palette[index])
            }
            paletted.Palette = palette
        }
        return out
    }
}

func doneKey(bannerKey string) string {
    return bannerKey + "-done"
}
