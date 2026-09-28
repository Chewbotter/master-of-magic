package combat

// Spells that give light. Not in the original (user, 2026-09-27).
//
// Two things, for the pictures of spells only:
// - the picture is drawn a second time so that its light is added to what is under it: its bright
//   pixels get brighter, up to white, its dark ones add nothing
// - a halo around it: the light of the picture spread over the pixels around it, added to what is
//   under it, so it also lies on the ground and on the units it is near
//
// The halo is made of art pixels, as the picture is, and its light falls off in a few steps in
// place of evenly, so it looks like a part of the art. It is made once for every frame of a picture.
//
// SpellGlowKey turns it off and on in a battle, to compare.

import (
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/inpututil"
)

var SpellGlow = true
var SpellGlowKey = ebiten.KeyF7

// how much of the light of the picture is added on top of it, 0 to 1
const spellEmissive = 0.35
// how far the halo goes from the pixels of the picture, in art pixels
const spellHaloReach = 5
// the steps its light falls off in
const spellHaloBands = 3
// how much of the halo is added, 0 to 1
const spellHaloStrength = 0.55
// the part of the pixels within reach that has to give light for the halo to be at its brightest.
// lower: a wider and brighter halo
const spellHaloFull = 0.3
// pixels darker than this, of 255 for their brightest part, give no light: the black edges
const spellHaloDark = 60

func (combat *CombatScreen) updateSpellGlowKey() {
    if inpututil.IsKeyJustPressed(SpellGlowKey) {
        SpellGlow = !SpellGlow
        if SpellGlow {
            combat.CastMessage("Spell glow: on")
        } else {
            combat.CastMessage("Spell glow: off")
        }
    }
}

// the halo of a picture: pixels with their colors multiplied by how much they show, 4 numbers a
// pixel, reach wider than the picture on every side. source is the same of the picture
func makeHalo(source []byte, width int, height int, reach int, bands int) ([]byte, int, int) {
    outWidth := width + reach * 2
    outHeight := height + reach * 2
    out := make([]byte, 4 * outWidth * outHeight)

    type light struct {
        X, Y int
        Red, Green, Blue float64
    }

    // the pixels that give light, with their colors as they are
    var lights []light
    for y := range height {
        for x := range width {
            at := (y * width + x) * 4
            alpha := float64(source[at + 3])
            if alpha == 0 {
                continue
            }
            red := float64(source[at]) * 255 / alpha
            green := float64(source[at + 1]) * 255 / alpha
            blue := float64(source[at + 2]) * 255 / alpha
            if math.Max(red, math.Max(green, blue)) < spellHaloDark {
                continue
            }
            lights = append(lights, light{X: x + reach, Y: y + reach, Red: red, Green: green, Blue: blue})
        }
    }

    if len(lights) == 0 {
        return out, outWidth, outHeight
    }

    // what a pixel gets from a light by how far it is, and what it gets when all within reach
    // give light
    weights := make(map[int]float64)
    all := 0.0
    for dy := -reach; dy <= reach; dy++ {
        for dx := -reach; dx <= reach; dx++ {
            distance := math.Hypot(float64(dx), float64(dy))
            if distance > float64(reach) {
                continue
            }
            weight := 1 - distance / float64(reach + 1)
            weights[dy * (reach * 2 + 1) + dx] = weight
            all += weight
        }
    }

    sums := make([]float64, 4 * outWidth * outHeight)
    for _, each := range lights {
        for dy := -reach; dy <= reach; dy++ {
            for dx := -reach; dx <= reach; dx++ {
                weight, ok := weights[dy * (reach * 2 + 1) + dx]
                x := each.X + dx
                y := each.Y + dy
                if !ok || x < 0 || y < 0 || x >= outWidth || y >= outHeight {
                    continue
                }
                at := (y * outWidth + x) * 4
                sums[at] += each.Red * weight
                sums[at + 1] += each.Green * weight
                sums[at + 2] += each.Blue * weight
                sums[at + 3] += weight
            }
        }
    }

    for index := 0; index < len(sums); index += 4 {
        weight := sums[index + 3]
        if weight <= 0 {
            continue
        }

        // how bright, in steps
        bright := math.Min(1, weight / (all * spellHaloFull))
        step := math.Ceil(bright * float64(bands)) / float64(bands)

        // the color of the lights around it
        out[index] = uint8(math.Min(255, sums[index] / weight * step))
        out[index + 1] = uint8(math.Min(255, sums[index + 1] / weight * step))
        out[index + 2] = uint8(math.Min(255, sums[index + 2] / weight * step))
        out[index + 3] = uint8(255 * step)
    }

    return out, outWidth, outHeight
}

// the halo of a frame of a spell, made when it is first asked for
func (combat *CombatScreen) spellHalo(picture *ebiten.Image) *ebiten.Image {
    if halo, ok := combat.spellHalos[picture]; ok {
        return halo
    }

    width := picture.Bounds().Dx()
    height := picture.Bounds().Dy()
    source := make([]byte, 4 * width * height)
    picture.ReadPixels(source)

    pixels, outWidth, outHeight := makeHalo(source, width, height, spellHaloReach, spellHaloBands)
    halo := ebiten.NewImage(outWidth, outHeight)
    halo.WritePixels(pixels)

    if combat.spellHalos == nil {
        combat.spellHalos = make(map[*ebiten.Image]*ebiten.Image)
    }
    combat.spellHalos[picture] = halo
    return halo
}

// adds the light of a picture to what is on the field. x and y are on the original's screen
func (combat *CombatScreen) addLightOnField(screen *ebiten.Image, picture *ebiten.Image, x int, y int, strength float32) {
    matrix := originalScreenMatrix(combat.GetCameraMatrix())

    var options ebiten.DrawImageOptions
    options.GeoM.Translate(float64(x), float64(y))
    options.GeoM.Concat(matrix)
    options.ColorScale.ScaleAlpha(strength)
    options.Blend = ebiten.BlendLighter
    scale.DrawScaled(screen, picture, &options)
}

// a frame of a spell with its light
func (combat *CombatScreen) drawSpellPicture(screen *ebiten.Image, picture *ebiten.Image, x int, y int) {
    if SpellGlow {
        combat.addLightOnField(screen, combat.spellHalo(picture), x - spellHaloReach, y - spellHaloReach, spellHaloStrength)
    }

    combat.drawOnField(screen, picture, x, y)

    if SpellGlow {
        combat.addLightOnField(screen, picture, x, y, spellEmissive)
    }
}
