package combat

// Splashes (user, 2026-09-29): a unit that walks into the water of a river throws up drops, single
// art pixels as all particles are (particles.go). Not in the original.
//
// "Relatively subtle and scaled for groups entering the water, so all 8 units don't make an
// enormous amount of splashing while single entities still make a noticeable splash": a unit of
// one figure throws splashDrops (5), a unit of more figures that many times the root of their number,
// so 8 figures throw less than 3 times what one does (13), from the places of its figures.
//
// A unit that comes from the land splashes as a whole. One that wades on from one cell of the
// river into the next throws splashWade of it. The banks are no water, and units that fly stay
// dry.

import (
    "image"
    "image/color"
    "math"
    "math/rand/v2"

    "github.com/kazzmir/master-of-magic/game/magic/unitview"
)

// turns them off
var RiverSplash = true

// the drops of a unit of one figure that walks into the water (first 9; user, 2026-09-29: "half
// the water splash particles")
const splashDrops = 4.5
// of them, for a step from water into water
const splashWade = 0.4
// the drops of a unit are the ones of a figure times its figures to the power of this
const splashGroup = 0.5

// art pixels around the feet of a figure the drops start in, across and down the screen
const splashSpreadX = 4.0
const splashSpreadY = 1.5
// art pixels a second: along the ground, and upward. the drops spray out over the water more than
// they go up (user, 2026-09-29: "spray outward more instead of upward"; first 11 along the ground,
// 18 to 40 upward, gravity 150)
const splashSpeed = 30.0
const splashLiftMin = 12.0
const splashLiftMax = 24.0
// art pixels a second more with every second
const splashGravity = 90.0

// the colors a drop goes through
var splashColors = []color.RGBA{
    {R: 238, G: 246, B: 255, A: 255},
    {R: 186, G: 214, B: 244, A: 255},
    {R: 120, G: 164, B: 216, A: 255},
}

// how many drops a unit of so many figures throws. part is 1 for a unit that walks into the
// water, splashWade for one that wades on
func splashCount(figures int, part float64) int {
    if figures <= 0 {
        return 0
    }
    return max(1, int(math.Round(splashDrops * math.Pow(float64(figures), splashGroup) * part)))
}

// drops that are thrown up from a place on the water and fall back into it, where they are gone
func (system *particleSystem) emitSplash(x float64, y float64, count int) {
    for range count {
        angle := rand.Float64() * 2 * math.Pi
        own := splashSpeed * randomPart(0.3, 1)
        lift := randomPart(splashLiftMin, splashLiftMax)

        system.add(particle{
            X: x + randomPart(-splashSpreadX, splashSpreadX),
            Y: y + randomPart(-splashSpreadY, splashSpreadY),
            SpeedX: math.Cos(angle) * own,
            SpeedY: math.Sin(angle) * own * particleGroundSquash - lift,
            Gravity: splashGravity,
            // up and down again
            Life: 2 * lift / splashGravity,
            Colors: splashColors,
        })
    }
}

// the tile a unit that walks is over
func wadingTile(unit *ArmyUnit) image.Point {
    return image.Pt(int(math.Round(unit.MoveX)), int(math.Round(unit.MoveY)))
}

// one tick of the battle: the units that have walked into the water of a river since the last
// splash. called with the effects, see spelleffects.go
func (combat *CombatScreen) splashTick() {
    effects := &combat.effects
    ground := combat.Model.Ground
    // ice does not splash
    if !RiverSplash || ground == nil || ground.Banks == nil || ground.Frozen {
        return
    }
    if effects.Wading == nil {
        effects.Wading = make(map[*ArmyUnit]image.Point)
    }

    field := MakeBattlefieldMatrix()

    for _, army := range []*Army{combat.Model.AttackingArmy, combat.Model.DefendingArmy} {
        if army == nil {
            continue
        }
        for _, unit := range army.units {
            if !unit.Moving {
                delete(effects.Wading, unit)
                continue
            }

            tile := wadingTile(unit)
            last, known := effects.Wading[unit]
            effects.Wading[unit] = tile
            if !known || last == tile || !combat.Model.IsRiverTile(tile.X, tile.Y) || unit.IsFlying() {
                continue
            }

            part := 1.0
            if combat.Model.IsRiverTile(last.X, last.Y) {
                part = splashWade
            }

            figures := max(1, unit.VisibleFigures())
            points := unitview.CombatPoints(figures)
            if len(points) == 0 {
                continue
            }
            screenX, screenY := field.Apply(unit.MoveX, unit.MoveY)

            // drop by drop, from figures by chance
            for range splashCount(figures, part) {
                point := points[rand.N(len(points))]
                effects.Particles.emitSplash(screenX + float64(point.X), screenY + float64(point.Y), 1)
            }
        }
    }
}
