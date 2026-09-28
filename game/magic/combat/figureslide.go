package combat

// Figures that slide. Not in the original.
//
// The lunge of a strike: a figure that attacks steps back and then forward, toward what it faces
// (user, 2026-09-27).
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

    "github.com/kazzmir/master-of-magic/game/magic/mod"
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

// where the figure is at the end of each frame of its swing, in art pixels toward what it faces from
// where it stands. frames are the frames of the swing in their order, a frame that shows longer
// more than once
func strikeSlideEnds(frames []int) []float64 {
    // a frame that shows longer is in the list more than once: the figure is where it was
    again := func(index int) bool {
        return index > 0 && frames[index] == frames[index - 1]
    }

    // the frames the figure moves forward on, 5 and 6
    forward := 0
    for index, frame := range frames {
        if frame > strikeWindUpFrame && !again(index) {
            forward += 1
        }
    }

    ends := make([]float64, len(frames))
    done := 0

    for index, frame := range frames {
        switch {
            case again(index):
                // it stays: held back on its wind up, and forward where its blow has landed
                ends[index] = ends[index - 1]
            case frame > strikeWindUpFrame:
                // forward, by the same part on each of these frames
                done += 1
                ends[index] = -strikeSlideBack + (strikeSlideBack + strikeSlideForward) * float64(done) / float64(forward)
            case forward > 0:
                // 3 steps back, 4 holds there
                ends[index] = -strikeSlideBack
            case frame == mod.FrameStrike:
                // nothing to move forward on but the strike frame
                ends[index] = strikeSlideForward
            default:
                // the standing frame of the game, or the wind up without anything after it
                ends[index] = -strikeSlideBack
        }
    }

    return ends
}

func easeInOut(part float64) float64 {
    return part * part * (3 - 2 * part)
}

// how far the figure is from where it stands, across and down in art pixels, for its lunge
func (combat *CombatScreen) strikeSlide(unit *ArmyUnit, phase float64) (float64, float64) {
    defends := unit.Defending && !unit.Attacking
    if !StrikeSlide || !(unit.Attacking || unit.Defending) || (defends && !DefenderSlides) {
        return 0, 0
    }

    // the frames of the swing and where in them the unit starts, as figureFrame shows them
    frames := longStrikeFrames(unit)
    ahead := 0
    switch {
        case len(frames) > 0 && defends:
            ahead = len(frames) / 2
        case len(frames) > 0:
        case defends:
            frames = figureDefendFrames
        default:
            frames = gameStrikeFrames()
    }

    // where in the swing the figure is, with the part of the frame that has gone by
    tps := float64(max(1, ebiten.TPS()))
    ticks := float64(combat.Counter) * OriginalTicksPerSecond / tps + phase
    // the whole part of this is the frame figureFrame shows, the rest is how far into it
    step := ticks / attackTicksPerFrame + float64(ahead)
    place := math.Mod(step, float64(len(frames)))
    index := int(place)
    part := place - float64(index)

    // through a frame the figure goes from where the frame before it ended to where this one ends
    ends := strikeSlideEnds(frames)
    from := ends[(index + len(ends) - 1) % len(ends)]
    to := ends[index]
    distance := from + (to - from) * easeInOut(part)
    if defends {
        distance *= defenderSlidePart
    }

    faceX, faceY := facingOnScreen(unit.Facing)
    return faceX * distance, faceY * distance
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
