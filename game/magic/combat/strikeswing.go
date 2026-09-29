package combat

// The swing of a figure that strikes: which frame it shows and where it is, step by step. Not in
// the original, which flips between the standing and the strike frame and does not move.
//
// The steps (user, 2026-09-28), for a figure with the frames 3, 4 and 5 of the replacement folder:
//   1. it stands where it stands, in frame 3
//   2. it steps back, still in frame 3
//   3. back there it shows frame 4, the wind up
//   4. it rushes forward, still in frame 4
//   5. furthest forward it shows frame 5 and stays there for a moment
//   6. it shows frame 3 again and goes back to where it stands
// With frame 6 as well it shows 5 and then 6 while it is furthest forward, stays on 6, and then
// shows 3 and goes back.
//
// A figure that has not all of these frames does the same steps with the frames it has:
// - frame 4 and nothing after it: furthest forward it shows frame 3, the strike of the game
// - the frames of the game only: it steps back and rushes forward in frame 1, how it stands, and
//   furthest forward it shows frame 3
//
// The unit that is attacked shows the same frames in the same steps, half a swing later, and does
// not move (see DefenderSlides in figureslide.go).
//
// AN ATTACK IS WHOLE SWINGS (user, 2026-09-28), SwingsPerAttack of them, one after the other and
// none cut short. The first starts when the attack does, and the attack lasts until the unit
// that attacks and the one that is attacked have both done theirs. The blow is struck when the
// figures of the attacker are furthest forward in the swing SwingOfTheBlow. Before its swings
// start and after they are over a figure stands.

import (
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/mod"

    "github.com/hajimehoshi/ebiten/v2"
)

// how long the steps take, in frames of a strike (attackTicksPerFrame redraws of the original each)
const swingStandTime = 1.0
const swingBackTime = 1.0
const swingWindUpTime = 1.0
const swingRushTime = 0.6
const swingLandTime = 1.0
// the moment it stays furthest forward on its last frame, after the time the frame has anyway
const swingStayTime = 1.0
const swingReturnTime = 1.0

// the swings of an attack
// 2 was tried (user, 2026-09-28) and taken back for now
const SwingsPerAttack = 1
// the swing the blow is struck in, from 1: what the attack does to the units, the numbers, the
// blood and the figures that fall come when this swing lands. the last one, so nobody strikes at
// figures that have fallen already
const SwingOfTheBlow = SwingsPerAttack

// one step of a swing
type swingStep struct {
    Frame int
    // where the figure is when the step starts and when it ends, in art pixels toward what it faces
    // from where it stands
    From float64
    To float64
    // in frames of a strike
    Time float64
    // it rushes: slowly at first and then fast, and stops at once
    Rush bool
}

// the steps of a swing. extra is the frames after the strike frame the figure has, in their order
// and without a gap: nothing, 4, 4 5, or 4 5 6
func swingSteps(extra []int) []swingStep {
    back := -strikeSlideBack
    forward := strikeSlideForward

    // what it shows while it stands, steps back and returns, and while it winds up
    ready := figureStandFrame
    windUp := figureStandFrame
    // what it shows furthest forward
    lands := []int{mod.FrameStrike}

    if len(extra) > 0 {
        ready = mod.FrameStrike
        windUp = extra[0]
        if len(extra) > 1 {
            lands = extra[1:]
        }
    }

    steps := []swingStep{
        {Frame: ready, From: 0, To: 0, Time: swingStandTime},
        {Frame: ready, From: 0, To: back, Time: swingBackTime},
        {Frame: windUp, From: back, To: back, Time: swingWindUpTime},
        {Frame: windUp, From: back, To: forward, Time: swingRushTime, Rush: true},
    }

    for index, frame := range lands {
        time := swingLandTime
        if index == len(lands) - 1 {
            time += swingStayTime
        }
        steps = append(steps, swingStep{Frame: frame, From: forward, To: forward, Time: time})
    }

    steps = append(steps, swingStep{Frame: ready, From: forward, To: 0, Time: swingReturnTime})
    return steps
}

func swingTime(steps []swingStep) float64 {
    total := 0.0
    for _, step := range steps {
        total += step.Time
    }
    return total
}

// the time of a swing at which the figure gets furthest forward: when its blow lands
func swingLands(steps []swingStep) float64 {
    time := 0.0
    for _, step := range steps {
        time += step.Time
        if step.Rush {
            return time
        }
    }
    return time
}

// the frame a figure shows and where it is at a time of its swings, in frames of a strike. it
// swings a number of times, one swing after the other. before and after them the figure stands,
// in the frame a swing starts with
func swingAt(steps []swingStep, time float64, swings int) (int, float64) {
    total := swingTime(steps)
    if len(steps) == 0 || total <= 0 {
        return figureStandFrame, 0
    }

    if time < 0 || time >= total * float64(swings) {
        return steps[0].Frame, 0
    }
    time = math.Mod(time, total)

    for _, step := range steps {
        if time >= step.Time {
            time -= step.Time
            continue
        }

        part := time / step.Time
        if step.Rush {
            part = part * part
        } else {
            part = easeInOut(part)
        }
        return step.Frame, step.From + (step.To - step.From) * part
    }

    last := steps[len(steps) - 1]
    return last.Frame, last.To
}

// the frames after the strike frame the figure of the unit has, as it faces: the ones of the
// replacement folder up to the first that is missing
func extraStrikeFrames(unit *ArmyUnit) []int {
    archive := unit.Unit.GetCombatLbxFile()
    entry := unit.Unit.GetCombatIndex(unit.Facing)

    var frames []int
    for frame := mod.FrameStrike + 1; frame <= mod.FrameStrikeLast; frame++ {
        if !mod.HasFrame(archive, entry, frame) {
            break
        }
        frames = append(frames, frame)
    }
    return frames
}

// the frame a figure of a unit that strikes shows and how far it has lunged. ticks is the redraws
// of the original since the swing of the unit started
func unitSwing(unit *ArmyUnit, ticks float64) (int, float64) {
    steps := swingSteps(extraStrikeFrames(unit))

    defends := unit.Defending && !unit.Attacking
    if defends {
        ticks -= unit.SwingDelay
    }

    frame, place := swingAt(steps, ticks / attackTicksPerFrame, SwingsPerAttack)
    if defends && !DefenderSlides {
        place = 0
    }
    return frame, place
}

// the redraws of the original a swing of a unit takes
func swingRedraws(unit *ArmyUnit) float64 {
    return swingTime(swingSteps(extraStrikeFrames(unit))) * attackTicksPerFrame
}

// the ticks of the battle for a number of redraws of the original
func redrawTicks(redraws float64) int {
    return int(math.Ceil(redraws * float64(max(1, ebiten.TPS())) / OriginalTicksPerSecond))
}

// starts the swings of an attack. defender can be nil, for an attack on a wall. returns how many
// ticks of the battle the attack lasts, and the tick of it the blow is struck at
func (combat *CombatScreen) startSwing(attacker *ArmyUnit, defender *ArmyUnit) (int, int) {
    attacker.SwingStart = combat.Counter
    attacker.SwingDelay = 0

    if !singleStrikes() {
        // again and again for a fixed time, and nobody closes in. see strikeclassic.go
        if defender != nil {
            defender.SwingStart = combat.Counter
            defender.SwingDelay = 0
        }
        combat.startApproach(nil, nil, classicAttackTicks)
        attacker.ApproachX, attacker.ApproachY = 0, 0
        if defender != nil {
            defender.ApproachX, defender.ApproachY = 0, 0
        }
        return classicAttackTicks, classicBlowTick
    }

    // the figures of a unit are out of step by up to this, see figurevariety.go
    lasts := swingRedraws(attacker) * SwingsPerAttack + figurePhaseMax
    lands := swingRedraws(attacker) * (SwingOfTheBlow - 1)
    lands += swingLands(swingSteps(extraStrikeFrames(attacker))) * attackTicksPerFrame + figurePhaseMax / 2

    if defender != nil {
        defender.SwingStart = combat.Counter
        // half a swing of the attacker later
        defender.SwingDelay = swingRedraws(attacker) / 2
        lasts = max(lasts, defender.SwingDelay + swingRedraws(defender) * SwingsPerAttack + figurePhaseMax)
    }

    // units across a corner close in on each other, see strikeapproach.go
    combat.startApproach(attacker, defender, redrawTicks(lasts))

    return redrawTicks(lasts), redrawTicks(lands)
}
