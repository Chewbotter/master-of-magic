package combat

// Figures that slide. Not in the original.
//
// The lunge of a strike: a figure that attacks with a long strike (frames of the replacement folder,
// see game/magic/mod) slides back while it winds up on frame 4, and forward through the rest of the
// swing, toward what it faces.
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
// how far a figure slides back on its wind up and forward at the end of its swing, in art pixels
// from where it stands
const strikeSlideBack = 3.0
const strikeSlideForward = 4.0

// turns closing ranks off: the figures are at their new places at once
const RegroupSlide = true
// art pixels a second a figure slides to its new place in the tile
const regroupSpeed = 30.0

// where a figure is from where it stands while it shows a frame of a long strike, in art pixels
// toward what it faces. the swing goes forward from the wind up to its last frame
func strikeSlideAt(frame int) float64 {
    switch {
        case frame == strikeWindUpFrame: return -strikeSlideBack
        case frame > strikeWindUpFrame && frame <= mod.FrameStrikeLast:
            // frame 5 is half of the way, frame 6 all of it
            part := float64(frame - strikeWindUpFrame) / float64(mod.FrameStrikeLast - strikeWindUpFrame)
            return strikeSlideForward * part
    }
    // the strike frame itself: where the figure stands
    return 0
}

func easeInOut(part float64) float64 {
    return part * part * (3 - 2 * part)
}

// how far the figure is from where it stands, across and down in art pixels, for its lunge
func (combat *CombatScreen) strikeSlide(unit *ArmyUnit, phase float64) (float64, float64) {
    if !StrikeSlide || !unit.Attacking {
        return 0, 0
    }

    frames := longStrikeFrames(unit)
    if len(frames) == 0 {
        return 0, 0
    }

    // where in the swing the figure is, with the part of the frame that has gone by
    tps := float64(max(1, ebiten.TPS()))
    ticks := float64(combat.Counter) * OriginalTicksPerSecond / tps + phase
    // the whole part of this is the frame figureFrame shows, the rest is how far into it
    step := ticks / attackTicksPerFrame
    place := math.Mod(step, float64(len(frames)))
    index := int(place)
    part := place - float64(index)

    from := strikeSlideAt(frames[index])
    to := strikeSlideAt(frames[(index + 1) % len(frames)])
    distance := from + (to - from) * easeInOut(part)

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
