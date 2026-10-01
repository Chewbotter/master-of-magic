package game

// Development: battles fought out on the battlefield in a run without a window (-sim-tactical).
// The original resolves every battle the human does not watch by its quick resolution
// (strategicclassic.go), so the battlefield, its rules and Chewbot's battle AI only ever ran in the
// battles of the human. With SimTactical every battle of a run is fought out by combat.Run, Chewbot
// leading both armies (the human's on auto), and before it the quick resolution is asked what it
// would have said: on armies made anew from the same units, whose health is put back after. The
// journal's note of the battle tells both (simjournal.go).

import (
    "fmt"

    "github.com/kazzmir/master-of-magic/game/magic/combat"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// every battle of a run without a window is fought out on the battlefield
var SimTactical = false

// what the quick resolution would say of a battle, without a trace on the units
func (game *Game) simQuickWould(model *combat.CombatModel, attacker *playerlib.Player, attackerStack *playerlib.UnitStack, defender *playerlib.Player, defenderStack *playerlib.UnitStack, zone combat.ZoneType, createArmy func(*playerlib.Player, *playerlib.UnitStack) *combat.Army) combat.CombatState {
    saved := make(map[units.StackUnit]int)
    for _, stack := range []*playerlib.UnitStack{attackerStack, defenderStack} {
        for _, unit := range stack.Units() {
            saved[unit] = unit.GetHealth()
        }
    }
    attackers := createArmy(attacker, attackerStack)
    defenders := createArmy(defender, defenderStack)
    // the units ask the battle for the node that helps them; the quick resolution has none
    influence := model.Influence
    model.Influence = data.MagicNone
    defer func() { model.Influence = influence }()
    for _, army := range []*combat.Army{attackers, defenders} {
        for _, unit := range army.GetUnits() {
            unit.Model = model
        }
    }
    x, y, plane := defenderStack.X(), defenderStack.Y(), defenderStack.Plane()
    attackerSide := game.strategicSide(attacker, false, x, y, plane)
    defenderSide := game.strategicSide(defender, zone.Encounter != combat.ZoneNone, x, y, plane)
    state, _, _, _, _ := combat.DoClassicStrategicCombat(attackers, defenders, attackerSide, defenderSide, game.strategicStructure(x, y, plane, zone))
    for unit, health := range saved {
        if change := health - unit.GetHealth(); change != 0 {
            unit.AdjustHealth(change)
        }
    }
    return state
}

// the words of the note of a battle fought out: its turns, how it ended, the quick resolution's say
func simTacticalWords(model *combat.CombatModel, state combat.CombatState, quickWould combat.CombatState) string {
    how := ""
    if model.CurrentTurn >= combat.MAX_TURNS {
        how = ", the turn limit"
    }
    words := fmt.Sprintf("tactical %v turns%v", model.CurrentTurn, how)
    if quickWould != combat.CombatStateRunning {
        words += fmt.Sprintf(", quick would be: %v", simStateName(quickWould))
    }
    return words
}
