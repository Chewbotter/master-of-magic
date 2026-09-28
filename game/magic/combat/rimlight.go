package combat

// The light of a spell on the rim of the figures near it. Not in the original (user, 2026-09-28).
//
// A figure that stands in the light of a spell has the pixels of its rim that face the spell in
// the color of the light of the spell: the outermost pixels of its picture on the side the light
// comes from, one art pixel wide. No pixel inside of the figure is changed. The rim comes up when
// the spell gives light and is gone when the light is.
//
// The side is one of 8. For every picture and side the rim is made once, from the picture: a
// pixel of the figure belongs to it if the pixel next to it toward the light is empty.
//
// How much and in which color are the values rim-light and rim-color of the spell, see
// spellvalues.go.

import (
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/unitview"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: no light on the rims
const RimLight = true

// the sides the light can come from
const rimSides = 8
// a figure this near to the rim of the light, as a part of how far the light goes, has all of
// its rim light. nearer to the rim of the light it has less
const rimFullPart = 0.35
// the light falls on a figure this many art pixels over its feet
const rimFigureHeight = 10.0

// the way to the light for each side, across and down, in pixels of the picture
var rimSteps = [rimSides][2]int{
    {1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}, {0, -1}, {1, -1},
}

type rimKey struct {
    Picture *ebiten.Image
    Side int
}

// the side a light is on, seen from a figure. across and down the screen
func rimSide(towardX float64, towardY float64) int {
    angle := math.Atan2(towardY, towardX)
    side := int(math.Round(angle / (2 * math.Pi / rimSides)))
    return ((side % rimSides) + rimSides) % rimSides
}

// the rim of a picture on a side, in white: 4 numbers a pixel as the picture has them
func makeRimPixels(source []byte, width int, height int, side int) []byte {
    out := make([]byte, len(source))
    step := rimSteps[((side % rimSides) + rimSides) % rimSides]

    solid := func(x int, y int) bool {
        if x < 0 || y < 0 || x >= width || y >= height {
            return false
        }
        return source[(y * width + x) * 4 + 3] != 0
    }

    for y := range height {
        for x := range width {
            if !solid(x, y) {
                continue
            }

            // toward the light there is nothing of the figure. for a light from a corner: across,
            // down or both
            lit := !solid(x + step[0], y + step[1])
            if step[0] != 0 && step[1] != 0 {
                lit = lit || !solid(x + step[0], y) || !solid(x, y + step[1])
            }
            if !lit {
                continue
            }

            at := (y * width + x) * 4
            out[at] = 255
            out[at + 1] = 255
            out[at + 2] = 255
            out[at + 3] = 255
        }
    }

    return out
}

func (combat *CombatScreen) figureRim(picture *ebiten.Image, side int) *ebiten.Image {
    effects := &combat.effects
    key := rimKey{Picture: picture, Side: side}
    if rim, ok := effects.Rims[key]; ok {
        return rim
    }

    bounds := picture.Bounds()
    pixels := make([]byte, 4 * bounds.Dx() * bounds.Dy())
    picture.ReadPixels(pixels)

    rim := ebiten.NewImage(bounds.Dx(), bounds.Dy())
    rim.WritePixels(makeRimPixels(pixels, bounds.Dx(), bounds.Dy(), side))

    if effects.Rims == nil {
        effects.Rims = make(map[rimKey]*ebiten.Image)
    }
    effects.Rims[key] = rim
    return rim
}

// how much rim light a point on the original's screen has, 0 to 1, the side it comes from and
// its color
func rimLightAt(lights []spellLightSource, x float64, y float64) (float64, int, [3]float32) {
    best := 0.0
    side := 0
    tint := [3]float32{1, 1, 1}

    for _, light := range lights {
        if light.Radius <= 0 || light.Rim <= 0 || light.Strength <= 0 {
            continue
        }

        towardX := light.X - x
        towardY := light.Y - y
        // on the ground: what is up and down the screen is twice as far as it looks
        distance := math.Hypot(towardX, towardY / lightSquash)
        inside := 1 - distance / light.Radius
        if inside <= 0 {
            continue
        }

        amount := light.Rim * min(1, light.Strength) * min(1, inside / rimFullPart)
        if amount > best {
            best = amount
            side = rimSide(towardX, towardY)
            tint = light.RimColor
        }
    }

    return min(1, best), side, tint
}

// the rim of a figure of a unit that stands in the light of a spell. count is the number of
// figures the unit is laid out for
func (combat *CombatScreen) drawFigureRim(screen *ebiten.Image, unit *ArmyUnit, picture *ebiten.Image, options ebiten.DrawImageOptions, figure int, count int) {
    if !RimLight || !SpellEffects || picture == nil {
        return
    }

    lighting := &combat.effects.Lighting
    if lighting.ShadowSwap <= 0 || len(lighting.Lights) == 0 {
        return
    }

    points := unitview.CombatPoints(count)
    if figure < 0 || figure >= len(points) {
        return
    }

    x, y := float64(unit.X), float64(unit.Y)
    if unit.Moving {
        x, y = unit.MoveX, unit.MoveY
    }
    field := MakeBattlefieldMatrix()
    screenX, screenY := field.Apply(x, y)
    screenX += float64(points[figure].X)
    screenY += float64(points[figure].Y) - rimFigureHeight

    amount, side, tint := rimLightAt(lighting.Lights, screenX, screenY)
    // comes up and goes with the light of the spell, as its shadows do
    amount *= lighting.ShadowSwap
    if amount <= 0 {
        return
    }

    options.ColorScale.Scale(tint[0], tint[1], tint[2], 1)
    options.ColorScale.ScaleAlpha(float32(amount))
    unitview.RenderCombatFigure(screen, combat.figureRim(picture, side), options, unit.VisibleFigures(), unit.LostUnits, nil, data.UnitEnchantmentNone, combat.Counter, &combat.ImageCache, figure)
}
