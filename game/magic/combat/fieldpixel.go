package combat

// Figures stand on the art pixels of the field. Not in the original, which has no places between
// its pixels.
//
// A unit that walks, a figure that trails its unit, lunges or closes ranks is at a place between
// art pixels most of the time. Drawn there, its pixels do not line up with the pixels of the
// figures and the ground around it, and thin edges show between them (user, 2026-09-28). A figure
// is drawn at the nearest art pixel of the field, as the cursors, the shadows and the corpses are.
// It moves by whole art pixels then.

import (
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/scale"
)

// false: figures are drawn where they are, between pixels.
// TRIED AND TAKEN BACK (user, 2026-09-28): on the art pixels the figures moved in steps, which
// looked worse than the thin edges, which show only now and then while figures move. figures stand
// on whole art pixels whenever they stand
const FiguresOnFieldPixels = false
// the size of an art pixel on the screen is taken as whole if it is this near to a whole number.
// while the zoom changes it is not, and figures are drawn where they are
const fieldPixelWhole = 0.001

// the place on the grid of art pixels that is nearest to a place. all in pixels of the screen, in
// one direction: start is where the grid starts, pixel the size of an art pixel
func nearestFieldPixel(position float64, start float64, pixel float64) float64 {
    if pixel <= 0 {
        return position
    }
    return math.Round((position - start) / pixel) * pixel + start
}

// a place of the picture of the game, as the matrix of the camera gives it, moved to the nearest
// art pixel of the field
func (combat *CombatScreen) onFieldPixel(x float64, y float64) (float64, float64) {
    if !FiguresOnFieldPixels || scale.ScaleAmount <= 0 {
        return x, y
    }

    exact := combat.CameraScale * scale.ScaleAmount
    pixel := math.Round(exact)
    if pixel < 1 || math.Abs(exact - pixel) > fieldPixelWhole {
        return x, y
    }

    field := combat.GetCameraMatrix()
    fieldX, fieldY := field.Apply(0, 0)
    startX := math.Round(fieldX * scale.ScaleAmount)
    startY := math.Round(fieldY * scale.ScaleAmount)

    x = nearestFieldPixel(x * scale.ScaleAmount, startX, pixel) / scale.ScaleAmount
    y = nearestFieldPixel(y * scale.ScaleAmount, startY, pixel) / scale.ScaleAmount
    return x, y
}
