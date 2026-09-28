package combat

// Which unit of the player is next, and picking one with a click. Not in the original (user,
// 2026-09-27).
//
// The game goes through the units of an army in the order they joined it, whatever their places.
// For the army a player leads, the next unit is now the next one on the field, read like the days
// of a calendar: along a row to the right, then the row below. A row is a line of tiles the way an
// army stands in its rows when the battle starts, which runs to the upper right on the screen; the
// row below is the next one toward the lower right.
//
// Armies the computer leads, and the army of the player on auto, keep the order of the game: what
// a unit decides depends on the units that went before it, see together.go.
//
// A click on a unit of the player that still has its turn picks it.

import (
    "cmp"
    "image"
    "slices"

    "github.com/kazzmir/master-of-magic/game/magic/pathfinding"
    "github.com/kazzmir/master-of-magic/lib/fraction"
)

// false: the order of the game
const CalendarOrder = true
// false: a click on a unit of the player does not pick it
const ClickSelectsUnit = true

// where the unit that was picked last stood, and for which turn of which army that is
type unitOrder struct {
    Set bool
    Place image.Point
    Turn int
    Team Team
}

// before in the order: the row first, then along the row
func placeBefore(a image.Point, b image.Point) int {
    return cmp.Or(cmp.Compare(a.Y, b.Y), cmp.Compare(a.X, b.X))
}

// the units in the order they are gone through, starting after a place. without a place from the
// first
func unitsInOrder(units []*ArmyUnit, after image.Point, hasAfter bool) []*ArmyUnit {
    sorted := slices.Clone(units)
    slices.SortStableFunc(sorted, func(a *ArmyUnit, b *ArmyUnit) int {
        return placeBefore(image.Pt(a.X, a.Y), image.Pt(b.X, b.Y))
    })

    if !hasAfter {
        return sorted
    }

    start := len(sorted)
    for index, unit := range sorted {
        if placeBefore(image.Pt(unit.X, unit.Y), after) > 0 {
            start = index
            break
        }
    }

    return append(slices.Clone(sorted[start:]), sorted[:start]...)
}

func (model *CombatModel) notePicked(unit *ArmyUnit) {
    model.order = unitOrder{Set: true, Place: image.Pt(unit.X, unit.Y), Turn: model.CurrentTurn, Team: unit.Team}
}

// the next unit of an army a player leads. does to units that sleep or are in a web what
// ChooseNextUnit does to them
func (model *CombatModel) chooseNextInOrder(army *Army, team Team) *ArmyUnit {
    // a new turn of the army starts with its first unit
    hasAfter := model.order.Set && model.order.Turn == model.CurrentTurn && model.order.Team == team

    for _, unit := range unitsInOrder(army.units, model.order.Place, hasAfter) {
        if unit.IsAsleep() || unit.ConfusionAction == ConfusionActionDoNothing {
            unit.LastTurn = model.CurrentTurn
        }

        if unit.LastTurn >= model.CurrentTurn {
            continue
        }

        // spend a turn to remove the web
        if unit.IsWebbed() {
            unit.ProcessWeb()
            continue
        }

        unit.Paths = make(map[image.Point]pathfinding.Path)
        model.order = unitOrder{Set: true, Place: image.Pt(unit.X, unit.Y), Turn: model.CurrentTurn, Team: team}
        return unit
    }

    return nil
}

// true if a click on the unit picks it: a unit of the army whose turn it is, led by the player,
// that still has its turn and can act
func (model *CombatModel) CanPick(unit *ArmyUnit) bool {
    if !ClickSelectsUnit || unit == nil || unit == model.SelectedUnit {
        return false
    }

    selected := model.SelectedUnit
    if selected == nil || model.IsAIControlled(selected) || selected.Moving {
        return false
    }

    if unit.Team != model.Turn || model.IsAIControlled(unit) || unit.GetHealth() <= 0 {
        return false
    }

    if unit.LastTurn >= model.CurrentTurn || unit.MovesLeft.LessThanEqual(fraction.FromInt(0)) {
        return false
    }

    return !unit.IsAsleep() && !unit.IsWebbed() && unit.ConfusionAction == ConfusionActionNone
}

// picks the unit, as if the player had waited until it came up. the unit that was selected keeps
// its turn
func (model *CombatModel) Pick(unit *ArmyUnit) bool {
    if !model.CanPick(unit) {
        return false
    }

    unit.Paths = make(map[image.Point]pathfinding.Path)
    model.SelectedUnit = unit
    // the next one is the one after it
    model.notePicked(unit)
    return true
}
