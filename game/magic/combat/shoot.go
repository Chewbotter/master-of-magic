package combat

// The frames of a figure that shoots (user, 2026-09-30: "Extra attack frames are added but not seen
// in the game, probably because ranged attacks behave differently from melee. Let's hook these up
// to work with the same convention, using all the frames. Ranged units don't need to slide when
// attacking.").
//
// A ranged attack showed no frames at all: the unit stood while its missile flew. A unit whose
// figures have frames after the strike frame in the replacement folder (4, 4 5, 4 5 6, see
// strikeswing.go) now goes through the steps of a swing with them before it shoots, and does not
// move:
//   frame 3, frame 4 the draw, then 5 (and 6) where a swing lands, where the missile is loosed,
//   a moment on the last of them, and frame 3 again
// The missile leaves when the figures show the frame a swing lands on. The same with the setting
// "Single strikes" on or off: a shot is one. A unit with the frames of the game only shoots as it
// did, without frames.

import (
    "github.com/kazzmir/master-of-magic/lib/coroutine"
)

// turns it off
var ShotFrames = true

// the steps of a shot: the ones of a swing, where the figure stays where it stands
func shotSteps(extra []int) []swingStep {
    steps := swingSteps(extra)
    for index := range steps {
        steps[index].From = 0
        steps[index].To = 0
    }
    return steps
}

// true if a unit shows frames when it shoots
func showsShot(unit *ArmyUnit) bool {
    return ShotFrames && len(extraStrikeFrames(unit)) > 0
}

// the frame a figure of a unit that shoots shows. ticks is the redraws of the original since the
// shot started
func unitShot(unit *ArmyUnit, ticks float64) int {
    frame, _ := swingAt(shotSteps(extraStrikeFrames(unit)), ticks / attackTicksPerFrame, 1)
    return frame
}

// starts the frames of a shot. returns the ticks of the battle until the missile leaves, and until
// the shot is over
func (combat *CombatScreen) startShot(unit *ArmyUnit) (int, int) {
    unit.Shooting = true
    unit.SwingStart = combat.Counter
    unit.SwingDelay = 0

    steps := shotSteps(extraStrikeFrames(unit))
    // the figures of a unit are out of step by up to this, see figurevariety.go
    looses := swingLands(steps) * attackTicksPerFrame + figurePhaseMax / 2
    lasts := swingTime(steps) * attackTicksPerFrame + figurePhaseMax
    return redrawTicks(looses), redrawTicks(lasts)
}

// lets the clock of the battle run for a number of its ticks, as the attacks do
func (combat *CombatScreen) waitTicks(yield coroutine.YieldFunc, ticks int) bool {
    for range ticks {
        // the clock of the battle, see together.go
        combat.actionTick()
        if !combat.together {
            combat.ProcessEvents(yield)
        }
        if yield() != nil {
            return false
        }
    }
    return true
}
