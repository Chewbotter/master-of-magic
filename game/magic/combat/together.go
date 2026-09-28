package combat

// Units that act together. Not in the original, where units act one after the other.
//
// When it is the turn of units the computer controls, all of them act at once: the enemy's army,
// and the player's own army when it is set to auto. Each unit decides and acts the way it always
// did, in a routine of its own, and all routines take one step per tick. What a turn does does not
// change, only that the units do not wait for each other.
//
// UNITS DECIDE AS THEY DO IN TURN (user, 2026-09-27: accuracy before speed, and no unit loses its
// turn to another). A unit decides only when the unit before it, in the order of their turns, is
// settled: everything its turn does to the battle has been done, where it stands, what it has hit.
// So every unit decides on the battle it would have found in its turn, and decides the same. What
// overlaps is what takes the time: a unit walks to a tile it was put on at once, an attack plays on
// after it has struck.
// - a unit that walks is put on the tile it walks to at once, tile by tile as in its turn, and its
//   picture walks there after
// - a unit is settled when it has no moves left and its last blow has struck, or when its turn ends
// - the clock of the battle (combat.Counter, animations, projectiles, input) moves once per tick, in
//   runTogether, and not once per unit. actions call actionTick, which does nothing while units act
//   together
//
// What is not the same as in turn: while its picture still walks, a unit can be passed through by
// the picture of a unit that walks where it stood. Wall of fire hurts a unit when it decides to
// walk through, not when its picture gets there.
//
// The army of the player can be taken off auto while its units act, see autotoggle.go.
//
// One unit of the player set to auto on its own, confused units and webbed units act in turn as before.

import (
    "context"
    "image"
    "log"
    "math"
    "time"

    "github.com/kazzmir/master-of-magic/game/magic/inputmanager"
    "github.com/kazzmir/master-of-magic/game/magic/pathfinding"
    "github.com/kazzmir/master-of-magic/lib/coroutine"
    "github.com/kazzmir/master-of-magic/lib/fraction"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: units act in turn
var ActTogether = true

// seconds a unit waits after the unit before it is settled, so they do not set off as one
const togetherStartStep = 0.08
// seconds a unit waits before it decides again after it could not do what it decided
const togetherRetryTime = 0.15
// the last unit that acts ends its turn when it could not do what it decided this often in a row.
// when units act in turn that cannot happen, this keeps a battle from standing still if it does
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
    combat.updateAutoToggle(true)
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

// everything the turn of the unit does to the battle has been done
func (combat *CombatScreen) settle(unit *ArmyUnit) {
    if combat.together && combat.settledUnits != nil {
        combat.settledUnits[unit] = true
    }
}

// for an action: the unit has done what it does to the battle, if it has no moves left to do more
func (combat *CombatScreen) settleIfDone(unit *ArmyUnit) {
    if unit.GetHealth() <= 0 || unit.MovesLeft.LessThanEqual(fraction.FromInt(0)) {
        combat.settle(unit)
    }
}

// the turn of one unit, in a routine of its own. before is the unit that decides before it
func (combat *CombatScreen) unitRoutine(unit *ArmyUnit, before *ArmyUnit, startDelay int) *coroutine.Coroutine {
    model := combat.Model

    return coroutine.MakeCoroutine(func(yield coroutine.YieldFunc) error {
        defer combat.settle(unit)

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

        for before != nil && !combat.settledUnits[before] {
            if wait() != nil {
                return nil
            }
        }
        pause(startDelay)

        // the army was taken off auto before this unit started: it keeps its turn. see autotoggle.go
        if stopped || !combat.startsTogether(unit) {
            return nil
        }

        acted := false
        actions := &togetherActions{
            CombatActions: &CombatActions{
                AIUnitActions: &AIUnitActions{yield: wait, combat: combat},
                yield: wait,
                combat: combat,
            },
            acted: &acted,
        }

        failed := 0
        for !stopped && unit.GetHealth() > 0 && unit.MovesLeft.GreaterThan(fraction.FromInt(0)) && model.FinalState() == CombatStateRunning {
            moves := unit.MovesLeft
            x, y := unit.X, unit.Y
            acted = false

            doAI(model, combat, actions, unit)

            if unit.GetHealth() <= 0 {
                break
            }

            // it found nothing to do, or what it decided could not be done
            nothingDone := !acted || (unit.MovesLeft.Equals(moves) && unit.X == x && unit.Y == y)
            if !nothingDone {
                failed = 0
                continue
            }

            // it decided on the battle it would have found in its turn: nothing to do is nothing to do
            if !acted {
                break
            }

            failed += 1
            if failed >= togetherRetries {
                unit.MovesLeft = fraction.FromInt(0)
                break
            }
            pause(togetherTicks(togetherRetryTime))
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
        combat.settledUnits = nil
        combat.heldUnits = nil
    }()

    combat.settledUnits = make(map[*ArmyUnit]bool)
    // units that did not start because their army was taken off auto, see autotoggle.go
    held := make(map[*ArmyUnit]bool)
    combat.heldUnits = held

    var routines []*coroutine.Coroutine
    var before *ArmyUnit
    for _, unit := range units {
        routines = append(routines, combat.unitRoutine(unit, before, togetherTicks(togetherStartStep)))
        before = unit
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
        combat.togetherRunning = running
        // the clock of the battle, once for all units
        combat.Counter += 1
        combat.UpdateAnimations()
        combat.UpdateDamageIndicators()
        combat.ProcessInput()
        combat.updateAutoToggle(true)
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
        if !held[unit] {
            unit.LastTurn = model.CurrentTurn
        }
    }

    if model.FinalState() == CombatStateRunning {
        model.NextUnit()
    }
}

// the actions of a unit that acts together with others. notes whether the unit did anything
type togetherActions struct {
    *CombatActions
    acted *bool
}

func (actions *togetherActions) Teleport(mover *ArmyUnit, x int, y int, merge bool) {
    *actions.acted = true
    actions.CombatActions.Teleport(mover, x, y, merge)
}

func (actions *togetherActions) RangeAttack(attacker *ArmyUnit, defender RangeTarget) {
    *actions.acted = true
    actions.CombatActions.RangeAttack(attacker, defender)
}

func (actions *togetherActions) MeleeAttack(attacker *ArmyUnit, defender *ArmyUnit) {
    *actions.acted = true
    actions.CombatActions.MeleeAttack(attacker, defender)
}

func (actions *togetherActions) MeleeAttackWall(attacker *ArmyUnit, x int, y int) {
    *actions.acted = true
    actions.CombatActions.MeleeAttackWall(attacker, x, y)
}

func (actions *togetherActions) MoveMagicVortex(vortex *MagicVortex, path pathfinding.Path) {
    *actions.acted = true
    actions.CombatActions.MoveMagicVortex(vortex, path)
}

func (actions *togetherActions) MoveUnit(unit *ArmyUnit, path pathfinding.Path) {
    *actions.acted = true
    actions.CombatActions.MoveUnit(unit, path)
}

func (actions *togetherActions) DoProjectiles() {
    *actions.acted = true
    actions.CombatActions.DoProjectiles()
}

// a unit walks while units act together: it is put on the tile it walks to at once, tile by tile
// as when it walks in its turn, and then its picture walks there
func (combat *CombatScreen) moveTogether(yield coroutine.YieldFunc, mover *ArmyUnit, path pathfinding.Path) {
    fromX, fromY := float64(mover.X), float64(mover.Y)

    var steps []image.Point
    for len(path) > 0 && mover.MovesLeft.GreaterThan(fraction.FromInt(0)) {
        x, y := path[0].X, path[0].Y

        other := combat.Model.GetUnit(x, y)
        if other != nil && other != mover && other.GetHealth() > 0 {
            break
        }

        died := combat.Model.MoveUnit(mover, x, y)
        if died {
            return
        }

        steps = append(steps, image.Pt(x, y))
        path = path[1:]
    }

    if len(steps) == 0 {
        return
    }

    // from here on only its picture walks
    combat.settleIfDone(mover)

    mover.Moving = true
    mover.MoveX = fromX
    mover.MoveY = fromY
    mover.CurrentPath = nil
    defer func() {
        mover.Moving = false
        mover.MoveX = float64(mover.X)
        mover.MoveY = float64(mover.Y)
        mover.Paths = make(map[image.Point]pathfinding.Path)
    }()

    quit, cancel := context.WithCancel(combat.Quit)
    defer cancel()

    sound, err := combat.AudioCache.GetSound(mover.Unit.GetMovementSound().LbxIndex())
    if err == nil && combat.IsUnitVisible(mover) {
        // the sound of its steps until it has arrived
        go func() {
            for quit.Err() == nil {
                err := sound.Rewind()
                if err != nil {
                    log.Printf("Unable to rewind sound for %v: %v", mover.Unit.GetMovementSound(), err)
                }
                sound.Play()
                for sound.IsPlaying() {
                    select {
                        case <-quit.Done():
                            sound.Pause()
                            return
                        case <-time.After(10 * time.Millisecond):
                    }
                }
            }
        }()
    }

    for _, step := range steps {
        targetX, targetY := float64(step.X), float64(step.Y)
        startX, startY := mover.MoveX, mover.MoveY

        angle := math.Atan2(targetY - startY, targetX - startX)
        // the way it faces on the screen, as in doMoveUnit
        mover.Facing = computeFacing(-(angle - math.Pi/4))

        ticks := max(1, int(math.Round(moveTicksPerCell())))
        for tick := 1; tick <= ticks; tick++ {
            part := float64(tick) / float64(ticks)
            mover.MoveX = startX + (targetX - startX) * part
            mover.MoveY = startY + (targetY - startY) * part

            if yield() != nil {
                return
            }
        }
    }
}
