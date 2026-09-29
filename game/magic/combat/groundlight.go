package combat

// The light of a spell on the ground. Not in the original (user, 2026-09-29).
//
// The lights of spells cut holes into the dark of the field (spelllight.go), so the ground near a
// spell was as bright as by day and no brighter. Here the light is ADDED to the ground: every
// pixel of the ground in the light gets lighter by the color of the light, as much as the light
// is strong there. It is drawn after the ground and before what stands on it, so it lies on the
// terrain and not on the figures, which have the light on their rims (rimlight.go).
//
// The light is the picture of the hole, in steps toward its rim, on the art pixels of the ground.
// How much is the value `ground-light` of a spell, the color is its `rim-color`.

import (
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: the ground in the light of a spell is as bright as by day
const GroundLight = true

// draws the light of the spells on the ground. after the ground, before what stands on it
func (combat *CombatScreen) drawGroundLight(screen *ebiten.Image) {
    if !GroundLight || !SpellLight || !SpellEffects {
        return
    }

    lighting := &combat.effects.Lighting

    lit := false
    for _, light := range lighting.Lights {
        if light.Ground > 0 && light.Strength > 0 && light.Radius >= 1 {
            lit = true
            break
        }
    }
    if !lit {
        return
    }

    // the picture of art pixels, lined up with the ground as the one of the dark
    pixel := math.Max(1, math.Round(combat.CameraScale * scale.ScaleAmount))
    camera := combat.GetCameraMatrix()
    fieldX, fieldY := camera.Apply(0, 0)
    startX := math.Mod(math.Round(fieldX * scale.ScaleAmount), pixel)
    startY := math.Mod(math.Round(fieldY * scale.ScaleAmount), pixel)
    if startX > 0 {
        startX -= pixel
    }
    if startY > 0 {
        startY -= pixel
    }

    bounds := screen.Bounds()
    width := int(math.Ceil(float64(bounds.Dx()) / pixel)) + 2
    height := int(math.Ceil(float64(bounds.Dy()) / pixel)) + 2
    if lighting.GroundLayer == nil || lighting.GroundLayer.Bounds().Dx() != width || lighting.GroundLayer.Bounds().Dy() != height {
        lighting.GroundLayer = ebiten.NewImage(width, height)
    }
    layer := lighting.GroundLayer
    layer.Clear()

    onScreen := originalScreenMatrix(camera)
    for _, light := range lighting.Lights {
        radius := int(math.Round(light.Radius))
        if radius < 1 || light.Strength <= 0 || light.Ground <= 0 {
            continue
        }

        picture := combat.lightPicture(radius)
        screenX, screenY := onScreen.Apply(light.X, light.Y)
        atX := math.Round((screenX * scale.ScaleAmount - startX) / pixel) - float64(picture.Bounds().Dx() / 2)
        atY := math.Round((screenY * scale.ScaleAmount - startY) / pixel) - float64(picture.Bounds().Dy() / 2)

        much := float32(light.Ground * min(1, light.Strength))

        var options ebiten.DrawImageOptions
        options.GeoM.Translate(atX, atY)
        options.ColorScale.Scale(light.RimColor[0] * much, light.RimColor[1] * much, light.RimColor[2] * much, much)
        // the lights of two spells add up
        options.Blend = ebiten.BlendLighter
        layer.DrawImage(picture, &options)
    }

    var onto ebiten.DrawImageOptions
    onto.GeoM.Scale(pixel, pixel)
    onto.GeoM.Translate(startX, startY)
    // added to what is there
    onto.Blend = ebiten.BlendLighter
    screen.DrawImage(layer, &onto)
}
