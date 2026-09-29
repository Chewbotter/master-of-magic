package combat

// The strike as it was before the swing of steps (strikeswing.go), for the setting "Single
// strikes" when it is off (user, 2026-09-28).
//
// A unit that attacks strikes again and again, fast, for as long as the attack lasts:
// - its frames go around by the clock of the battle: 3 and the frames after it that the
//   replacement folder has, 4 twice as long as the others (3 4 4, 3 4 4 5, 3 4 4 5 6). Without
//   frame 4 the two of the game, 3 and 1
// - it lunges by its frames: back on 3, held on 4, forward on 5 and 6. With 4 alone back on 4 and
//   forward on 3. With the frames of the game back on 1 and forward on 3
// - the unit that is attacked shows the same frames half of the way around behind, and stands
// - an attack lasts a second, and the blow is struck a third of a second into it
// - units stay on their tiles, they do not close in across a corner
//
// This is the state of commit 20b4600b, from before the trials with the unit that is attacked.

import (
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/game/magic/mod"
)

// the setting: an attack is one swing, see strikeswing.go. false: strikes again and again
func singleStrikes() bool {
    return display.SingleStrikes()
}

// frame 4 shows this many times as long as the other frames of a long strike
const strikeWindUpHold = 2
// ticks of the battle an attack lasts, and the tick of it the blow is struck at
const classicAttackTicks = 60
const classicBlowTick = 20

// the frames of the long strike of the figure of the unit, as it faces: the strike frame and the
// frames after it that the replacement folder has, up to the first that is missing. nothing if the
// first of them is missing: the figure strikes as the original does
func longStrikeFrames(unit *ArmyUnit) []int {
    return longStrikeFramesOf(extraStrikeFrames(unit))
}

// the same, from the frames after the strike frame the figure has
func longStrikeFramesOf(extra []int) []int {
    var frames []int
    for _, frame := range extra {
        if frames == nil {
            frames = append(frames, mod.FrameStrike)
        }

        // a frame that shows longer is in the list more than once
        hold := 1
        if frame == strikeWindUpFrame {
            hold = strikeWindUpHold
        }
        for range hold {
            frames = append(frames, frame)
        }
    }

    return frames
}

// the frame a unit that strikes shows. tick is the redraws of the original
func classicStrikeFrame(unit *ArmyUnit, tick uint64) int {
    step := tick / attackTicksPerFrame

    if long := longStrikeFrames(unit); len(long) > 0 {
        if unit.Defending && !unit.Attacking {
            step += uint64(len(long) / 2)
        }
        return long[step % uint64(len(long))]
    }

    if unit.Attacking {
        return figureAttackFrames[step % uint64(len(figureAttackFrames))]
    }
    return figureDefendFrames[step % uint64(len(figureDefendFrames))]
}

// where the figure is at the end of each frame of its swing, in art pixels toward what it faces from
// where it stands. frames are the frames of the swing in their order, a frame that shows longer
// more than once
func strikeSlideEnds(frames []int) []float64 {
    // the frames the figure moves forward on, 5 and 6
    forward := 0
    for _, frame := range frames {
        if frame > strikeWindUpFrame {
            forward += 1
        }
    }

    ends := make([]float64, len(frames))
    done := 0

    for index, frame := range frames {
        switch {
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

// how far a figure of a unit that attacks has lunged. ticks is the redraws of the original, with
// the part of one that has gone by. the unit that is attacked stands
func classicStrikeSlide(unit *ArmyUnit, ticks float64) float64 {
    if !unit.Attacking {
        return 0
    }

    frames := longStrikeFrames(unit)
    if len(frames) == 0 {
        // the strike of the game
        frames = figureAttackFrames
    }

    // the whole part of this is the frame that shows, the rest is how far into it
    step := ticks / attackTicksPerFrame
    place := math.Mod(step, float64(len(frames)))
    index := int(place)
    part := place - float64(index)

    // through a frame the figure goes from where the frame before it ended to where this one ends
    ends := strikeSlideEnds(frames)
    from := ends[(index + len(ends) - 1) % len(ends)]
    to := ends[index]
    return from + (to - from) * easeInOut(part)
}
