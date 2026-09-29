package spellbook

// The name of the spell under the mouse in the book for casting.
//
// The original does not mark it. Upstream let it pulse to bright red, which reads as an error
// (user, 2026-09-29). It pulses to a blue, slower and less far.

import (
    "math"
)

// ticks one pulse takes. upstream's red took 31
const hoverPulseTicks = 80
// the color the name has when the pulse is at its height, 0 to 255
const hoverRed = 20
const hoverGreen = 70
const hoverBlue = 190
// how much of that color the name has when the pulse is at its lowest, 0 to 1. 0: the black of the
// other names
const hoverLow = 0.45

// the color of the name under the mouse, as the values the letters are scaled with: the letters
// of the book have the color 1 of 255, so a value is the color it gives
func hoverTint(counter uint64) (float32, float32, float32) {
    wave := math.Cos(float64(counter) * 2 * math.Pi / hoverPulseTicks) / 2 + 0.5
    part := hoverLow + (1 - hoverLow) * wave
    return float32(hoverRed * part), float32(hoverGreen * part), float32(hoverBlue * part)
}
