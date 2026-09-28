package combat

// The light of a spell on the rim of the figures near it. Not in the original (user, 2026-09-28).
//
// A figure that stands in the light of a spell has the pixels of its rim that face the spell in
// the color of the light of the spell: the outermost pixels of its picture on the side the light
// comes from, one art pixel wide. No pixel inside of the figure is changed. The rim comes up when
// the spell gives light. When the light is gone, or gets weaker at once, the rim goes out fast
// but not at once.
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
// seconds the rim of a figure takes for most of its way to how much it should be lit: coming up,
// and going out
const rimInTime = 0.1
const rimOutTime = 0.22
// less than this is none
const rimGone = 0.01

// a figure of a unit
type rimFigure struct {
    Unit *ArmyUnit
    Figure int
}

// how the rim of a figure is lit now
type rimState struct {
    Amount float64
    Side int
    Color [3]float32
}

// where a figure of a unit is on the original's screen, at the height the light falls on it. count
// is the number of figures the unit is laid out for
func rimFigurePlace(unit *ArmyUnit, figure int, count int) (float64, float64, bool) {
    points := unitview.CombatPoints(count)
    if figure < 0 || figure >= len(points) {
        return 0, 0, false
    }

    x, y := float64(unit.X), float64(unit.Y)
    if unit.Moving {
        x, y = unit.MoveX, unit.MoveY
    }
    field := MakeBattlefieldMatrix()
    screenX, screenY := field.Apply(x, y)
    return screenX + float64(points[figure].X), screenY + float64(points[figure].Y) - rimFigureHeight, true
}

// how much a rim is lit after a tick: on its way from how much it is to how much it should be
func rimStep(now float64, should float64) float64 {
    time := rimOutTime
    if should > now {
        time = rimInTime
    }
    return now + (should - now) * easeStep(time)
}

// one tick of the rims: each goes a part of its way to how much it should be lit
func (combat *CombatScreen) rimTick() {
    lighting := &combat.effects.Lighting
    if !RimLight || !SpellEffects {
        lighting.Rims = nil
        return
    }

    if len(lighting.Lights) == 0 && len(lighting.Rims) == 0 {
        return
    }

    if combat.Model == nil || combat.Model.AttackingArmy == nil || combat.Model.DefendingArmy == nil {
        return
    }

    if lighting.Rims == nil {
        lighting.Rims = make(map[rimFigure]rimState)
    }

    seen := make(map[rimFigure]bool)
    for _, army := range []*Army{combat.Model.AttackingArmy, combat.Model.DefendingArmy} {
        for _, unit := range army.units {
            count := unit.VisibleFigures() + unit.LostUnits
            for figure := range unit.VisibleFigures() {
                x, y, ok := rimFigurePlace(unit, figure, count)
                if !ok {
                    continue
                }

                key := rimFigure{Unit: unit, Figure: figure}
                state, has := lighting.Rims[key]
                should, side, tint := rimLightAt(lighting.Lights, x, y)
                if !has && should <= 0 {
                    continue
                }

                state.Amount = rimStep(state.Amount, should)
                if should > 0 {
                    // while it goes out it stays on its side and in its color
                    state.Side = side
                    state.Color = tint
                }

                if should <= 0 && state.Amount < rimGone {
                    delete(lighting.Rims, key)
                    continue
                }

                lighting.Rims[key] = state
                seen[key] = true
            }
        }
    }

    for key := range lighting.Rims {
        if !seen[key] {
            delete(lighting.Rims, key)
        }
    }
}

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

    state, ok := combat.effects.Lighting.Rims[rimFigure{Unit: unit, Figure: figure}]
    if !ok || state.Amount <= 0 {
        return
    }

    options.ColorScale.Scale(state.Color[0], state.Color[1], state.Color[2], 1)
    options.ColorScale.ScaleAlpha(float32(min(1, state.Amount)))
    unitview.RenderCombatFigure(screen, combat.figureRim(picture, state.Side), options, unit.VisibleFigures(), unit.LostUnits, nil, data.UnitEnchantmentNone, combat.Counter, &combat.ImageCache, figure)
}
