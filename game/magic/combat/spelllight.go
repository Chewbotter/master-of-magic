package combat

// The light of a spell. Not in the original (user, 2026-09-28).
//
// While a spell plays the battlefield gets dark, and the spell is what gives light:
// - the field is darkened, all of it, when the spell starts. the dark has a little of the color
//   of the spell
// - around the spell it is not: the spell lights the ground and the units near it
// - the pictures of the spell, its particles and the numbers are drawn over the dark, as bright as
//   they are
// - the shadows of the units near the spell turn away from it and get harder, as if it was the
//   spell that cast them
// - when the spell hits the dark starts to lift, slowly at first. The light of the spell stays
//   where it hit a little longer and goes out, and the shadows turn back. What the spell left on
//   the ground, marks and corpses, stays
//
// The dark is a picture of art pixels lined up with the ground (as the shadows, see shadows.go)
// that the lights cut holes into. A light falls off in steps toward its rim, and from step to step
// in a pattern of single pixels, as the pictures of the game do it.
//
// How dark, how far the light goes and the rest are values of the spell, see spellvalues.go.

import (
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: spells play in the light of day
const SpellLight = true

// seconds the field takes to get dark, and to get light again once the spell has hit
const darkInTime = 0.15
const darkOutTime = 0.7
// the steps a light falls off in
const lightBands = 5
// the ground is seen from above at an angle: a light is this much as high as it is wide
const lightSquash = 0.5
// the radius of a light changes at most every this many ticks, when it flickers
const lightFlickerTicks = 3
// how dark the shadows of the units near a spell get, 0 to 1. see shadowStrength for the usual
const spellShadowStrength = 0.7
// art pixels a shadow is long for every art pixel the figure is high, near a spell
const spellShadowLength = 0.9
// a shadow that points to the side is not thinner than this, of the height of the figure
const spellShadowThin = 0.12
// the shadows of the day go out and the ones of the spell come up when a spell starts, in this
// many seconds: the new ones are there at once and do not turn in (user, 2026-09-28)
const shadowSwapInTime = 0.12
// seconds the shadows take for most of their way back when the light of the spell is gone
const shadowSwapOutTime = 0.4
// a unit this near to the rim of the reach of a light, as a part of the reach, has a shadow of
// full length. nearer to the rim it is shorter, so it does not come up at once at the rim
const shadowFullPart = 0.4

// the pattern of single pixels between two steps of a light, 4 by 4
var lightPattern = [4][4]float64{
    {0, 8, 2, 10},
    {12, 4, 14, 6},
    {3, 11, 1, 9},
    {15, 7, 13, 5},
}

type spellLightSource struct {
    // the middle, on the original's screen in art pixels
    X float64
    Y float64
    // art pixels, across
    Radius float64
    // how much of the dark it takes away in its middle, 0 to 1
    Strength float64
    // how far from it the shadows of units turn away from it, art pixels
    Reach float64
    // how white the rims of the figures in its light get, 0 to 1. see rimlight.go
    Rim float64
}

// a light that stays where a spell has hit and goes out
type spellGlow struct {
    Light spellLightSource
    Start uint64
    Ticks uint64
}

type spellLighting struct {
    // how far the field has gone dark, 0 to 1, how dark it goes, and the color of the dark
    Level float64
    Dark float64
    Color [3]float32
    // the dark lifts: since when, and from how dark
    Lifting bool
    LiftStart uint64
    LiftFrom float64

    Lights []spellLightSource
    Glows []spellGlow
    // pictures of lights by their radius
    Pictures map[int]*ebiten.Image
    Layer *ebiten.Image
    // what the radius of the lights is changed by now, when they flicker
    Flicker float64

    // how far the shadows of the day have gone out and the ones of the spell have come up, 0 to 1
    ShadowSwap float64
}

// the pixels of a light of a radius: only how much they show counts, 4 numbers a pixel
func makeLightPixels(radius int) ([]byte, int, int) {
    radiusY := max(1, int(math.Round(float64(radius) * lightSquash)))
    width := radius * 2 + 1
    height := radiusY * 2 + 1
    pixels := make([]byte, 4 * width * height)

    for y := range height {
        for x := range width {
            far := math.Hypot(float64(x - radius) / float64(radius), float64(y - radiusY) / float64(radiusY))
            if far >= 1 {
                continue
            }

            // all of it near the middle, less and less toward the rim
            part := easeInOut(1 - far)

            // in steps, from one step to the next in a pattern
            steps := part * lightBands
            whole := math.Floor(steps)
            if steps - whole > (lightPattern[y % 4][x % 4] + 0.5) / 16 {
                whole += 1
            }

            value := byte(math.Round(min(1, whole / lightBands) * 255))
            at := (y * width + x) * 4
            pixels[at] = value
            pixels[at + 1] = value
            pixels[at + 2] = value
            pixels[at + 3] = value
        }
    }

    return pixels, width, height
}

func (combat *CombatScreen) lightPicture(radius int) *ebiten.Image {
    lighting := &combat.effects.Lighting
    if picture, ok := lighting.Pictures[radius]; ok {
        return picture
    }

    pixels, width, height := makeLightPixels(radius)
    picture := ebiten.NewImage(width, height)
    picture.WritePixels(pixels)

    if lighting.Pictures == nil {
        lighting.Pictures = make(map[int]*ebiten.Image)
    }
    lighting.Pictures[radius] = picture
    return picture
}

// the step of a spell up to which it keeps the field dark: until it hits. a spell that hits when
// it starts to show, as the ones on a unit, keeps it dark for the first half of its pictures
func holdsDark(step int, impactStep int, steps int) bool {
    return step < max(impactStep, steps / 2)
}

// the light a spell gives now, if it gives any
func (combat *CombatScreen) lightOf(projectile *Projectile) (spellLightSource, SpellValues, bool) {
    if !projectile.Scripted || !projectile.Started || len(projectile.Pictures) == 0 || projectile.Target == nil {
        return spellLightSource{}, SpellValues{}, false
    }

    values := combat.valuesOf(projectile.Name)
    if values.Dark <= 0 {
        return spellLightSource{}, values, false
    }

    light := spellLightSource{Radius: values.LightRadius, Strength: values.LightStrength, Reach: values.ShadowReach, Rim: values.RimLight}

    // where it hits, unless it is in flight
    placeX, placeY := spellPlace(projectile.Target.X, projectile.Target.Y)
    light.X = float64(placeX)
    light.Y = float64(placeY) - values.ImpactHeight

    if projectile.Step >= 0 && projectile.Step < len(projectile.Steps) {
        step := projectile.Steps[projectile.Step]
        if step.GlideX != 0 || step.GlideY != 0 {
            x, y, frame, ok := projectile.placeAt(combat.Counter)
            if ok {
                bounds := projectile.Pictures[frame].Bounds()
                light.X = float64(x) + float64(bounds.Dx()) / 2
                light.Y = float64(y) + float64(bounds.Dy()) / 2
            }
        }
    }

    return light, values, true
}

// a spell has hit: its light stays there for a while
func (combat *CombatScreen) addGlow(projectile *Projectile, values SpellValues) {
    if values.Dark <= 0 || values.LightLinger <= 0 || values.LightRadius <= 0 {
        return
    }

    placeX, placeY := spellPlace(projectile.Target.X, projectile.Target.Y)
    lighting := &combat.effects.Lighting
    lighting.Glows = append(lighting.Glows, spellGlow{
        Light: spellLightSource{
            X: float64(placeX),
            Y: float64(placeY) - values.ImpactHeight,
            Radius: values.LightRadius,
            Strength: values.LightStrength,
            Reach: values.ShadowReach,
            Rim: values.RimLight,
        },
        Start: combat.effects.Tick,
        Ticks: effectTicks(values.LightLinger),
    })
}

// how far the dark has lifted by a part of its time, 0 to 1: slowly at first and at the end
func liftPart(part float64) float64 {
    return easeInOut(max(0, min(1, part)))
}

// the part of the way to where it should be something goes in one tick
func easeStep(seconds float64) float64 {
    ticks := seconds * float64(max(1, ebiten.TPS()))
    if ticks <= 1 {
        return 1
    }
    // most of the way in that time
    return 1 - math.Pow(0.05, 1 / ticks)
}

// one tick of the light. goes on while the battle stands still
func (combat *CombatScreen) lightTick() {
    effects := &combat.effects
    lighting := &effects.Lighting

    lighting.Lights = lighting.Lights[:0]
    holds := false

    if SpellLight && SpellEffects {
        flicker := 0.0
        dark := 0.0
        for _, projectile := range combat.Model.Projectiles {
            light, values, ok := combat.lightOf(projectile)
            if !ok {
                continue
            }

            if holdsDark(projectile.Step, projectile.ImpactStep, len(projectile.Steps)) {
                holds = true
                if values.Dark > dark {
                    dark = values.Dark
                    lighting.Dark = dark
                    lighting.Color = colorParts(values.DarkColor.R, values.DarkColor.G, values.DarkColor.B)
                }
            }

            flicker = max(flicker, values.LightFlicker)
            if light.Radius > 0 && light.Strength > 0 {
                lighting.Lights = append(lighting.Lights, light)
            }
        }

        if effects.Tick % lightFlickerTicks == 0 {
            lighting.Flicker = randomPart(-flicker, flicker)
        }
        for index := range lighting.Lights {
            lighting.Lights[index].Radius = max(1, lighting.Lights[index].Radius + lighting.Flicker)
        }

        kept := lighting.Glows[:0]
        for _, glow := range lighting.Glows {
            if effects.Tick - glow.Start >= glow.Ticks {
                continue
            }
            kept = append(kept, glow)

            // smaller and weaker until it is out
            left := 1 - liftPart(float64(effects.Tick - glow.Start) / float64(glow.Ticks))
            light := glow.Light
            light.Radius = max(1, light.Radius * (0.5 + 0.5 * left))
            light.Strength *= left
            lighting.Lights = append(lighting.Lights, light)
        }
        lighting.Glows = kept
    } else {
        lighting.Glows = nil
    }

    ticksPerSecond := float64(max(1, ebiten.TPS()))
    if holds {
        lighting.Lifting = false
        lighting.Level = min(1, lighting.Level + 1 / (darkInTime * ticksPerSecond))
    } else if lighting.Level > 0 {
        if !lighting.Lifting {
            lighting.Lifting = true
            lighting.LiftStart = effects.Tick
            lighting.LiftFrom = lighting.Level
        }
        gone := float64(effects.Tick - lighting.LiftStart) / (darkOutTime * ticksPerSecond)
        lighting.Level = lighting.LiftFrom * (1 - liftPart(gone))
    }

    combat.shadowTick()
}

// the shadows of the day go out and the ones of the spell come up, or the other way around
func (combat *CombatScreen) shadowTick() {
    lighting := &combat.effects.Lighting

    lit := false
    for _, light := range lighting.Lights {
        if (light.Reach > 0 || light.Rim > 0) && light.Strength > 0 {
            lit = true
            break
        }
    }

    if lit {
        lighting.ShadowSwap = min(1, lighting.ShadowSwap + 1 / (shadowSwapInTime * float64(max(1, ebiten.TPS()))))
    } else if lighting.ShadowSwap > 0 {
        lighting.ShadowSwap -= lighting.ShadowSwap * easeStep(shadowSwapOutTime)
        if lighting.ShadowSwap < shadowSwapGone {
            lighting.ShadowSwap = 0
        }
    }
}

// less than this of the shadows of a spell is none
const shadowSwapGone = 0.01

// how far the field is dark now, 0 to 1
func (combat *CombatScreen) darkNow() float64 {
    lighting := &combat.effects.Lighting
    return lighting.Level * lighting.Dark
}

// draws the dark with the holes the lights cut into it. after what stands on the field, before
// the pictures of spells
func (combat *CombatScreen) drawSpellDark(screen *ebiten.Image) {
    dark := combat.darkNow()
    if dark <= 0 {
        return
    }

    lighting := &combat.effects.Lighting

    // the picture of art pixels, lined up with the ground as the one of the shadows
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
    if lighting.Layer == nil || lighting.Layer.Bounds().Dx() != width || lighting.Layer.Bounds().Dy() != height {
        lighting.Layer = ebiten.NewImage(width, height)
    }
    layer := lighting.Layer

    layer.Clear()
    var fill ebiten.DrawImageOptions
    fill.GeoM.Scale(float64(width), float64(height))
    // black, with a little of the color of the spell
    fill.ColorScale.Scale(lighting.Color[0], lighting.Color[1], lighting.Color[2], 1)
    fill.ColorScale.ScaleAlpha(float32(dark))
    layer.DrawImage(combat.effectPixel(), &fill)

    onScreen := originalScreenMatrix(camera)
    for _, light := range lighting.Lights {
        radius := int(math.Round(light.Radius))
        if radius < 1 || light.Strength <= 0 {
            continue
        }

        picture := combat.lightPicture(radius)
        screenX, screenY := onScreen.Apply(light.X, light.Y)
        atX := math.Round((screenX * scale.ScaleAmount - startX) / pixel) - float64(picture.Bounds().Dx() / 2)
        atY := math.Round((screenY * scale.ScaleAmount - startY) / pixel) - float64(picture.Bounds().Dy() / 2)

        var options ebiten.DrawImageOptions
        options.GeoM.Translate(atX, atY)
        options.ColorScale.ScaleAlpha(float32(min(1, light.Strength)))
        // takes away of the dark what the light shows
        options.Blend = ebiten.BlendDestinationOut
        layer.DrawImage(picture, &options)
    }

    var onto ebiten.DrawImageOptions
    onto.GeoM.Scale(pixel, pixel)
    onto.GeoM.Translate(startX, startY)
    screen.DrawImage(layer, &onto)
}

// how the shadow a spell gives a unit lies: lean and length as unitview.RenderCombatFigureShadow
// takes them. false if the unit is too far from any light to have one
func (combat *CombatScreen) spellShadowShape(unit *ArmyUnit) (float64, float64, bool) {
    x, y := float64(unit.X), float64(unit.Y)
    if unit.Moving {
        x, y = unit.MoveX, unit.MoveY
    }

    lean, length, part := spellShadow(combat.effects.Lighting.Lights, x, y)
    return lean, length, part > 0
}

// how the shadow of a unit at a tile lies by the lights of spells: away from the light that
// counts the most for it. the last value is how much that light counts, 0 to 1
func spellShadow(lights []spellLightSource, x float64, y float64) (float64, float64, float64) {
    field := MakeBattlefieldMatrix()
    unitX, unitY := field.Apply(x, y)

    best := 0.0
    lean := 0.0
    length := 0.0

    for _, light := range lights {
        if light.Reach <= 0 || light.Strength <= 0 {
            continue
        }

        // on the ground: what is up and down the screen is twice as far as it looks
        awayX := unitX - light.X
        awayY := (unitY - light.Y) / lightSquash
        distance := math.Hypot(awayX, awayY)
        weight := 1 - distance / light.Reach
        if weight <= best {
            continue
        }
        best = weight

        if distance < 1 {
            // under the light: a short shadow as it is
            lean = 0
            length = spellShadowThin
            continue
        }

        // the shadow of a point lies away from the light, further the higher the point is
        lean = awayX / distance * spellShadowLength
        length = awayY / distance * spellShadowLength * lightSquash
        if math.Abs(length) < spellShadowThin {
            if length < 0 {
                length = -spellShadowThin
            } else {
                length = spellShadowThin
            }
        }
    }

    if best <= 0 {
        return 0, 0, 0
    }

    // shorter near the rim of the reach
    long := min(1, best / shadowFullPart)
    return lean * long, length * long, best
}

// how dark the shadows of the day are now, and the ones of the spell
func (combat *CombatScreen) shadowStrengths() (float32, float32) {
    swap := combat.effects.Lighting.ShadowSwap
    return float32(shadowStrength * (1 - swap)), float32(spellShadowStrength * swap)
}
