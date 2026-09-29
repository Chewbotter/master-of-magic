package spellbook

// The names of the book for casting that have a color of their own: the one under the mouse, the
// one that is being cast, the ones that cost too much in a battle.
//
// The original marks the name under the mouse not at all. Upstream let it pulse to bright red,
// which reads as an error (user, 2026-09-29): it pulsed to a blue, slower and less far, and
// pulses to the color of the realm of the spell now (user, 2026-09-29: "glow based on the school
// of magic (Red for chaos, etc)"), in both books. The pulse starts when the mouse comes over the
// name, from its lowest: on the clock of the book a name came up at any point of it, so going
// from name to name flickered.
//
// Such a name is printed twice: in the colors of the page, and over that in its own colors, as
// see-through as the pulse says. Its own colors are a pair as the ones of the page are, a lighter
// one for the soft edge of the letters and a darker one for their body, so the letters keep their
// shape while they change color. Letters of one flat color, as upstream's, lost it (user: "the
// colored pulse kills the new subtlety").

import (
    "image/color"
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/game/magic/data"
)

// the colors of a name: the soft edge of its letters and their body
type nameColors struct {
    Edge color.RGBA
    Body color.RGBA
}

// the paper of the pages of the book, measured on its picture (spells.lbx 0)
var pageColor = color.RGBA{R: 146, G: 106, B: 53, A: 255}
// the soft edge of a letter is between the paper and the body of the letter, this far from the
// body, 0 to 1. the page has its edges at about 0.7. an edge that was lighter than the body AND
// lighter than the paper made the letters glow at their rims (user, 2026-09-29)
const edgeTowardPage = 0.6

// the colors of a name whose letters have this color
func colorsOf(body color.RGBA) nameColors {
    mix := func(from uint8, to uint8) uint8 {
        return uint8(float64(from) + (float64(to) - float64(from)) * edgeTowardPage + 0.5)
    }

    return nameColors{
        Body: body,
        Edge: color.RGBA{R: mix(body.R, pageColor.R), G: mix(body.G, pageColor.G), B: mix(body.B, pageColor.B), A: 255},
    }
}

// under the mouse, a spell of no realm
var hoverColors = colorsOf(color.RGBA{R: 40, G: 100, B: 232, A: 255})

// under the mouse, by the realm of the spell: the color of the letters
var realmColors = map[data.MagicType]color.RGBA{
    data.NatureMagic: {R: 40, G: 150, B: 44, A: 255},
    data.SorceryMagic: {R: 40, G: 100, B: 232, A: 255},
    data.ChaosMagic: {R: 220, G: 44, B: 28, A: 255},
    data.LifeMagic: {R: 250, G: 248, B: 236, A: 255},
    data.DeathMagic: {R: 140, G: 52, B: 184, A: 255},
    data.ArcaneMagic: {R: 176, G: 176, B: 176, A: 255},
}

// the realms that have a color
var hoverRealms = []data.MagicType{data.NatureMagic, data.SorceryMagic, data.ChaosMagic, data.LifeMagic, data.DeathMagic, data.ArcaneMagic}

func hoverColorsOf(magic data.MagicType) nameColors {
    body, ok := realmColors[magic]
    if !ok {
        return hoverColors
    }
    return colorsOf(body)
}
// ticks one pulse takes. upstream's red took 31
const hoverPulseTicks = 80
// how much of its own colors the name has when the pulse is at its lowest and at its height, 0 to 1
const hoverLow = 0.35
const hoverHigh = 1

// the spell that is being cast: lighter than the page. the pulse is upstream's
var castingColors = colorsOf(color.RGBA{R: 216, G: 196, B: 156, A: 255})
const castingPulseTicks = 31
const castingLow = 0.2
const castingHigh = 0.85

// a spell that costs more than can be paid in a battle: grey, and no pulse
var costlyColors = colorsOf(color.RGBA{R: 68, G: 66, B: 62, A: 255})

// the palette of a font with these colors. for every value a pixel of a letter can have: the
// large letters of the names of the book of research have values the palette did not reach at
// first (it had 17 colors), and those pixels kept the dark of the page while the name pulsed
// (user, 2026-09-29)
func (colors nameColors) palette() color.Palette {
    out := color.Palette{color.RGBA{}, colors.Edge}
    for len(out) < 256 {
        out = append(out, colors.Body)
    }
    return out
}

// the same colors for letters that have no soft edge on the page: the names of the book of
// research are printed in ONE color there (researchNameColors), all of their pixels. with a soft
// edge toward the paper in the pulse those pixels were browner and darker than the letter, clear
// dark spots on a light color as the gray of arcane (user, 2026-09-29)
func (colors nameColors) flat() nameColors {
    return nameColors{Edge: colors.Body, Body: colors.Body}
}

// false while the setting "Pulsing spellbook text" is off: nothing pulses, as in the original
func pulses() bool {
    return display.SpellbookPulse()
}

// how much of its own colors the name under the mouse has at this tick, when the mouse came
// over it at the tick start: from low up to high and back
func pulseFrom(counter uint64, start uint64, ticks uint64, low float64, high float64) float32 {
    if !pulses() {
        // the spell under the mouse is not marked
        return 0
    }

    since := uint64(0)
    if counter > start {
        since = counter - start
    }
    wave := 0.5 - math.Cos(float64(since) * 2 * math.Pi / float64(ticks)) / 2
    return float32(low + (high - low) * wave)
}

// ticks the spell under the mouse takes to get its color in the book of research. there it does
// not pulse: it fades up to all of its color and stays so until the mouse leaves (user,
// 2026-09-29)
const researchHoverTicks = 14

// how much of the color of its realm the spell under the mouse has in the book of research, when
// the mouse came over it at the tick start
func fadeUpFrom(counter uint64, start uint64) float32 {
    if !pulses() {
        // the spell under the mouse is not marked
        return 0
    }

    since := uint64(0)
    if counter > start {
        since = counter - start
    }
    return float32(hoverHigh * min(1, float64(since) / researchHoverTicks))
}

// how much of its own colors a name has at this tick, between low and high
func pulse(counter uint64, ticks uint64, low float64, high float64) float32 {
    if !pulses() {
        // the spell that is being cast is lighter and stays so
        return float32(high)
    }

    wave := math.Cos(float64(counter) * 2 * math.Pi / float64(ticks)) / 2 + 0.5
    return float32(low + (high - low) * wave)
}
