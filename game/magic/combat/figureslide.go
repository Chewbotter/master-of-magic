package combat

// Figures that slide. Not in the original.
//
// The lunge of a strike: a figure that attacks steps back and then forward, toward what it faces
// (user, 2026-09-27). Which step of its swing it is at, which frame it shows then and where it is,
// is in strikeswing.go. What follows here was the first rule, by frames:
// - with frames of a long strike (4, 5, 6 of the replacement folder, see game/magic/mod): it steps
//   back on 3, holds on 4, and moves forward on 5 and 6
// - with 4 and nothing after it there is no frame to move forward on: it steps back on 4 and
//   forward on 3
// - with the frames of the game only: it steps back on 1 and forward on 3
//
// Closing ranks: when figures of a unit were lost and the ones that are left take their new places
// in the tile, they slide there.

import (
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/unitview"

    "github.com/hajimehoshi/ebiten/v2"
)

// turns the lunge off
const StrikeSlide = true
// false: only the unit that attacks lunges, the one that is attacked stands. on as a trial (user,
// 2026-09-28): the unit that is attacked strikes back with the same frames and lunges the same way
// TRIED AND TAKEN BACK (user, 2026-09-28): at the full distance and at half of it. the unit that
// is attacked standing still looks best and is the easiest to read
const DefenderSlides = false
// how far the unit that is attacked lunges, as a part of how far the attacker does
const defenderSlidePart = 0.5
// how far a figure slides back on its wind up and forward at the end of its swing, in art pixels
// from where it stands
const strikeSlideBack = 2.0
const strikeSlideForward = 2.0

// turns closing ranks off: the figures are at their new places at once
const RegroupSlide = true
// art pixels a second a figure slides to its new place in the tile
const regroupSpeed = 30.0

func easeInOut(part float64) float64 {
    return part * part * (3 - 2 * part)
}

// how far the figure is from where it stands, across and down in art pixels, for its lunge
func (combat *CombatScreen) strikeSlide(unit *ArmyUnit, phase float64) (float64, float64) {
    if !(unit.Attacking || unit.Defending) {
        return 0, 0
    }

    // units across a corner have closed in on each other, see strikeapproach.go
    nearX, nearY := combat.strikeApproach(unit)
    if !StrikeSlide {
        return nearX, nearY
    }

    // the steps of its swing, see strikeswing.go
    _, distance := unitSwing(unit, combat.swingTicks(unit, phase))

    faceX, faceY := facingOnScreen(unit.Facing)
    return nearX + faceX * distance, nearY + faceY * distance
}

// how far the figure is from its place in the tile, across and down in art pixels, while it slides
// to it. count is the number of figures of the unit with the lost ones
func (combat *CombatScreen) regroupSlide(unit *ArmyUnit, figure int, count int) (float64, float64) {
    points := unitview.CombatPoints(count)
    if !RegroupSlide || figure < 0 || figure >= len(points) {
        return 0, 0
    }

    state := combat.figuresOf(unit, count, float64(unit.X), float64(unit.Y))
    each := &state.Figures[figure]

    placeX := float64(points[figure].X)
    placeY := float64(points[figure].Y)

    if !each.Placed {
        each.Placed = true
        each.PlaceX = placeX
        each.PlaceY = placeY
        each.PlaceCounter = combat.Counter
    }

    if each.PlaceCounter != combat.Counter {
        ticks := min(combat.Counter - each.PlaceCounter, 10)
        each.PlaceCounter = combat.Counter

        reach := regroupSpeed * float64(ticks) / float64(max(1, ebiten.TPS()))
        distance := math.Hypot(placeX - each.PlaceX, placeY - each.PlaceY)
        if distance <= reach {
            each.PlaceX = placeX
            each.PlaceY = placeY
        } else {
            each.PlaceX += (placeX - each.PlaceX) * reach / distance
            each.PlaceY += (placeY - each.PlaceY) * reach / distance
        }
    }

    return each.PlaceX - placeX, each.PlaceY - placeY
}
