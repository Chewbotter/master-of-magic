package combat

// Units that act together. Not in the original, where units act one after the other.
//
// When it is the turn of units the computer controls, all of them act at once: the enemy's army,
// and the player's own army when it is set to auto. Each unit decides and acts the way it always
// did, in a routine of its own, and all routines take one step per tick. What a turn does does not
// change, only that the units do not wait for each other.
//
// What acting at once needs that acting in turn did not:
// - the clock of the battle (combat.Counter, animations, projectiles, input) moves once per tick, in
//   runTogether, and not once per unit. actions call actionTick, which does nothing while units act
//   together
// - a unit claims the tile it steps to, so two units do not walk into the same tile. a unit that
//   finds its tile taken waits a moment and decides again
// - a unit whose target was killed by another stops its attack
//
// One unit of the player set to auto on its own, confused units and webbed units act in turn as before.

import (
    "image"
    "math"
    "math/rand/v2"

    "github.com/kazzmir/master-of-magic/game/magic/inputmanager"
    "github.com/kazzmir/master-of-magic/lib/coroutine"
    "github.com/kazzmir/master-of-magic/lib/fraction"

    "github.com/hajimehoshi/ebiten/v2"
)

// turns it off: units act in turn
const ActTogether = true

// units start up to this many seconds after each other, so they do not move as one
const togetherStartDelayMax = 0.3
// seconds a unit waits before it decides again after it could not do what it decided
const togetherRetryTime = 0.15
// a unit that could not do what it decided this often in a row ends its turn
const togetherRetries = 6

// the units that act together now, or nothing when units act in turn
func (combat *CombatScreen) togetherUnits() []*ArmyUnit {
    model := combat.Model
    selected := model.SelectedUnit

    if !ActTogether || selected == nil || !model.IsAIControlled(selected) {
        return nil
    }

    var out []*ArmyUnit
    hasSelected := false

    for _, unit := range model.GetArmyForTeam(model.Turn).units {
        if unit.LastTurn >= model.CurrentTurn || unit.GetHealth() <= 0 {
            continue
        }
        if unit.IsAsleep() || unit.IsWebbed() || unit.ConfusionAction != ConfusionActionNone {
            continue
        }
        if !model.IsAIControlled(unit) {
            continue
        }

        out = append(out, unit)
        if unit == selected {
            hasSelected = true
        }
    }

    if !hasSelected || len(out) < 2 {
        return nil
    }

    return out
}

// one tick of the clock of the battle, for an action of a unit that acts in turn
func (combat *CombatScreen) actionTick() {
    if combat.together {
        return
    }

    combat.Counter += 1
    combat.UpdateAnimations()
    combat.UpdateDamageIndicators()
    combat.ProcessInput()
}

// the unit steps to a tile: nobody else may. false if the tile is taken
func (combat *CombatScreen) claimTile(unit *ArmyUnit, x int, y int) bool {
    if !combat.together {
        return true
    }

    point := image.Pt(x, y)
    owner, claimed := combat.claimedTiles[point]
    if claimed && owner != unit {
        return false
    }

    other := combat.Model.GetUnit(x, y)
    if other != nil && other != unit && other.GetHealth() > 0 {
        return false
    }

    combat.claimedTiles[point] = unit
    return true
}

func (combat *CombatScreen) releaseTiles(unit *ArmyUnit) {
    for point, owner := range combat.claimedTiles {
        if owner == unit {
            delete(combat.claimedTiles, point)
        }
    }
}

func togetherTicks(seconds float64) int {
    return int(seconds * float64(max(1, ebiten.TPS())))
}

// the turn of one unit, in a routine of its own
func (combat *CombatScreen) unitRoutine(unit *ArmyUnit, startDelay int) *coroutine.Coroutine {
    model := combat.Model

    return coroutine.MakeCoroutine(func(yield coroutine.YieldFunc) error {
        stopped := false
        // an action that was stopped must not wait again
        wait := func() error {
            if stopped {
                return coroutine.CoroutineCancelled
            }
            err := yield()
            if err != nil {
                stopped = true
            }
            return err
        }

        pause := func(ticks int) {
            for range ticks {
                if wait() != nil {
                    return
                }
            }
        }

        pause(startDelay)

        actions := &CombatActions{
            AIUnitActions: &AIUnitActions{yield: wait, combat: combat},
            yield: wait,
            combat: combat,
        }

        failed := 0
        for !stopped && unit.GetHealth() > 0 && unit.MovesLeft.GreaterThan(fraction.FromInt(0)) && model.FinalState() == CombatStateRunning {
            moves := unit.MovesLeft
            x, y := unit.X, unit.Y

            doAI(model, combat, actions, unit)

            if unit.MovesLeft.Equals(moves) && unit.X == x && unit.Y == y {
                // what it decided could not be done: a tile was taken, a target was killed
                failed += 1
                if failed >= togetherRetries {
                    unit.MovesLeft = fraction.FromInt(0)
                    break
                }
                pause(togetherTicks(togetherRetryTime))
            } else {
                failed = 0
            }
        }

        combat.releaseTiles(unit)
        return nil
    })
}

// all units act, until the last is done
func (combat *CombatScreen) runTogether(yield coroutine.YieldFunc, units []*ArmyUnit) {
    model := combat.Model

    // the wizard casts first, as before the first unit of a turn
    if model.doAiCast(combat, model.GetArmy(units[0])) {
        combat.doProjectiles(yield)
    }

    combat.together = true
    combat.claimedTiles = make(map[image.Point]*ArmyUnit)
    defer func() {
        combat.together = false
        combat.claimedTiles = nil
    }()

    var routines []*coroutine.Coroutine
    for _, unit := range units {
        delay := rand.N(togetherTicks(togetherStartDelayMax) + 1)
        routines = append(routines, combat.unitRoutine(unit, delay))
    }

    stopAll := func() {
        for _, routine := range routines {
            if routine != nil {
                routine.Stop()
            }
        }
    }

    running := len(routines)
    for running > 0 {
        // the clock of the battle, once for all units
        combat.Counter += 1
        combat.UpdateAnimations()
        combat.UpdateDamageIndicators()
        combat.ProcessInput()
        combat.ProcessEvents(yield)
        model.UpdateProjectiles(combat.Counter)

        mouseX, mouseY := inputmanager.MousePosition()
        tileX, tileY := combat.ScreenToTile(float64(mouseX), float64(mouseY))
        combat.MouseTileX = int(math.Round(tileX))
        combat.MouseTileY = int(math.Round(tileY))

        for index, routine := range routines {
            if routine == nil {
                continue
            }
            if routine.Run() != nil {
                routines[index] = nil
                running -= 1
            }
        }

        if yield() != nil {
            stopAll()
            return
        }
    }

    for _, unit := range units {
        unit.LastTurn = model.CurrentTurn
    }

    if model.FinalState() == CombatStateRunning {
        model.NextUnit()
    }
}
