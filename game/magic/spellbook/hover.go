package spellbook

// The names of the book for casting that have a color of their own: the one under the mouse, the
// one that is being cast, the ones that cost too much in a battle.
//
// The original marks the name under the mouse not at all. Upstream let it pulse to bright red,
// which reads as an error (user, 2026-09-29): it pulses to a blue, slower and less far.
//
// Such a name is printed twice: in the colors of the page, and over that in its own colors, as
// see-through as the pulse says. Its own colors are a pair as the ones of the page are, a lighter
// one for the soft edge of the letters and a darker one for their body, so the letters keep their
// shape while they change color. Letters of one flat color, as upstream's, lost it (user: "the
// colored pulse kills the new subtlety").

import (
    "image/color"
    "math"
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

// under the mouse
var hoverColors = colorsOf(color.RGBA{R: 40, G: 100, B: 232, A: 255})
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

// the palette of a font with these colors
func (colors nameColors) palette() color.Palette {
    out := color.Palette{color.RGBA{}, colors.Edge}
    for len(out) < 17 {
        out = append(out, colors.Body)
    }
    return out
}

// how much of its own colors a name has at this tick, between low and high
func pulse(counter uint64, ticks uint64, low float64, high float64) float32 {
    wave := math.Cos(float64(counter) * 2 * math.Pi / float64(ticks)) / 2 + 0.5
    return float32(low + (high - low) * wave)
}
