package combat

// Shadows of the figures on the ground. Not in the original.
//
// A shadow is the picture of the figure in black, laid on the ground: flattened and leaning to the
// lower right, the way the shadows painted into the trees and rocks fall. It uses the frame the
// figure shows, so it moves with it.
//
// Shadows are made of pixels of the art: they are drawn on a small picture in which one pixel is
// one art pixel of the field as the camera shows it, lined up with the pixels of the ground, and
// that picture is put on the ground enlarged. Leaning a picture at the size of the screen gives
// edges that cut through the art's pixels, which looks unrelated to the rest (user, 2026-09-27).
// Putting all shadows on the ground as one picture also keeps shadows that overlap from being
// darker than one.

import (
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
)

// turns them off
const FigureShadows = true

// how dark a shadow is, 0 to 1
const shadowStrength = 0.4
// for every art pixel of height above its feet a point of the figure lies this far right and this
// far below the feet. the upper right was tried (user, 2026-09-27) and taken back
const shadowLean = 0.7
const shadowLength = 0.35
// the shadow of a figure that flies lies this far from its feet, in art pixels
const shadowFlyingX = 9.0
const shadowFlyingY = 7.0

// the picture shadows are drawn on. sets up how it lies on the picture the field is drawn on
func (combat *CombatScreen) shadowPicture(screen *ebiten.Image) *ebiten.Image {
    // screen pixels per art pixel of the field
    combat.shadowPixel = math.Max(1, math.Round(combat.CameraScale * scale.ScaleAmount))

    // where the art pixels of the field start on the screen
    matrix := combat.GetCameraMatrix()
    fieldX, fieldY := matrix.Apply(0, 0)
    startX := math.Mod(math.Round(fieldX * scale.ScaleAmount), combat.shadowPixel)
    startY := math.Mod(math.Round(fieldY * scale.ScaleAmount), combat.shadowPixel)
    if startX > 0 {
        startX -= combat.shadowPixel
    }
    if startY > 0 {
        startY -= combat.shadowPixel
    }
    combat.shadowStartX = startX
    combat.shadowStartY = startY

    bounds := screen.Bounds()
    width := int(math.Ceil(float64(bounds.Dx()) / combat.shadowPixel)) + 2
    height := int(math.Ceil(float64(bounds.Dy()) / combat.shadowPixel)) + 2

    if combat.shadowLayer == nil || combat.shadowLayer.Bounds().Dx() != width || combat.shadowLayer.Bounds().Dy() != height {
        combat.shadowLayer = ebiten.NewImage(width, height)
    }
    combat.shadowLayer.Clear()
    return combat.shadowLayer
}

// from a position on the picture the field is drawn on to the picture of the shadows
func (combat *CombatScreen) shadowMatrix() ebiten.GeoM {
    var matrix ebiten.GeoM
    matrix.Translate(-combat.shadowStartX, -combat.shadowStartY)
    matrix.Scale(1 / combat.shadowPixel, 1 / combat.shadowPixel)
    return matrix
}

// puts the shadows on the ground
func (combat *CombatScreen) drawShadowPicture(screen *ebiten.Image) {
    var options ebiten.DrawImageOptions
    options.GeoM.Scale(combat.shadowPixel, combat.shadowPixel)
    options.GeoM.Translate(combat.shadowStartX, combat.shadowStartY)
    // harder near a spell that gives light, see spelllight.go
    options.ColorScale.ScaleAlpha(combat.shadowStrengthNow())
    screen.DrawImage(combat.shadowLayer, &options)
}
