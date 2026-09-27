package combat

// Shadows of the figures on the ground. Not in the original.
//
// A shadow is the picture of the figure in black, laid on the ground: flattened and leaning to the
// upper right, at the angle the user drew (about 35 degrees above level). It uses the frame the
// figure shows, so it moves with it. All shadows are drawn on a picture of their own that is put on
// the ground once, so shadows that overlap are not darker than one.

import (
    "github.com/hajimehoshi/ebiten/v2"
)

// turns them off
const FigureShadows = true

// how dark a shadow is, 0 to 1
const shadowStrength = 0.4
// for every art pixel of height above its feet a point of the figure lies this far right and this
// far below the feet. below 0 is above them: the shadow runs 100 across for 69 up
const shadowLean = 0.64
const shadowLength = -0.44
// the shadow of a figure that flies lies this far from its feet, in art pixels, the same way
const shadowFlyingX = 8.0
const shadowFlyingY = -5.5

// the picture shadows are drawn on, as large as the picture the field is drawn on
func (combat *CombatScreen) shadowPicture(screen *ebiten.Image) *ebiten.Image {
    bounds := screen.Bounds()
    if combat.shadowLayer == nil || combat.shadowLayer.Bounds().Dx() != bounds.Dx() || combat.shadowLayer.Bounds().Dy() != bounds.Dy() {
        combat.shadowLayer = ebiten.NewImage(bounds.Dx(), bounds.Dy())
    }
    combat.shadowLayer.Clear()
    return combat.shadowLayer
}

// puts the shadows on the ground
func (combat *CombatScreen) drawShadowPicture(screen *ebiten.Image) {
    var options ebiten.DrawImageOptions
    options.ColorScale.ScaleAlpha(shadowStrength)
    screen.DrawImage(combat.shadowLayer, &options)
}
