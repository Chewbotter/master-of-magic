package combat

// Variety between the figures of a unit. Not in the original, where all figures of a unit move and
// strike as one.
//
// Each figure has a delay of its own, picked when it is first drawn. While the unit walks the
// figure trails it by that delay and catches up when the unit stops. Its walking and striking
// frames are out of step with the other figures by a part of their cycle. A unit of one figure
// has no delays: it moves as before.

import (
    "math"
    "math/rand/v2"

    "github.com/hajimehoshi/ebiten/v2"
)

// turns all of it off
const FigureVariety = true

// the longest a figure trails its unit, in seconds. a step to the next tile takes about 0.44
const figureTrailMax = 0.09
// the furthest the frames of a figure are out of step, in redraws of the original. a walk cycle is
// 4 of them, a strike cycle 4 at half speed
const figurePhaseMax = 3.0
// a unit that moves further than this many tiles at once did not walk: its figures go with it
const figureTrailSnap = 2.0
// a figure this close to its unit, in tiles, has arrived
const figureTrailArrived = 0.01

type figureState struct {
    // seconds
    Trail float64
    // redraws of the original
    Phase float64
    // where the figure is, as the position of its unit, in tiles
    X float64
    Y float64
    // see figurefall.go
    Fall figureFall
}

type unitFigures struct {
    Figures []figureState
    // the tick the positions are from
    Counter uint64
}

func (combat *CombatScreen) figuresOf(unit *ArmyUnit, count int, x float64, y float64) *unitFigures {
    if combat.figureStates == nil {
        combat.figureStates = make(map[*ArmyUnit]*unitFigures)
    }

    state, ok := combat.figureStates[unit]
    if !ok {
        state = &unitFigures{Counter: combat.Counter}
        combat.figureStates[unit] = state
    }

    for len(state.Figures) < count {
        figure := figureState{X: x, Y: y}
        if count > 1 {
            figure.Trail = rand.Float64() * figureTrailMax
            figure.Phase = rand.Float64() * figurePhaseMax
        }
        state.Figures = append(state.Figures, figure)
    }

    return state
}

// where a figure is drawn, as the position of its unit in tiles. x, y is where the unit is
func (combat *CombatScreen) figurePosition(unit *ArmyUnit, figure int, count int, x float64, y float64) (float64, float64) {
    if !FigureVariety || figure < 0 || figure >= count {
        return x, y
    }

    state := combat.figuresOf(unit, count, x, y)

    if state.Counter != combat.Counter {
        ticks := min(combat.Counter - state.Counter, 10)
        state.Counter = combat.Counter
        seconds := float64(ticks) / float64(max(1, ebiten.TPS()))

        for index := range state.Figures {
            each := &state.Figures[index]
            distance := math.Hypot(x - each.X, y - each.Y)
            if each.Trail <= 0 || distance > figureTrailSnap || distance < figureTrailArrived {
                each.X, each.Y = x, y
                continue
            }

            step := 1 - math.Exp(-seconds / each.Trail)
            each.X += (x - each.X) * step
            each.Y += (y - each.Y) * step
        }
    }

    return state.Figures[figure].X, state.Figures[figure].Y
}

// how far the frames of a figure are out of step, in redraws of the original
func (combat *CombatScreen) figurePhase(unit *ArmyUnit, figure int, count int) float64 {
    if !FigureVariety || figure < 0 || figure >= count {
        return 0
    }

    x, y := float64(unit.X), float64(unit.Y)
    if unit.Moving {
        x, y = unit.MoveX, unit.MoveY
    }

    return combat.figuresOf(unit, count, x, y).Figures[figure].Phase
}
