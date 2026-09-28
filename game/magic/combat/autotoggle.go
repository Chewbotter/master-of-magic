package combat

// Auto combat on a key. Not in the original (user, 2026-09-27).
//
// Space sets the army of the player to auto, and takes it off auto when pressed again. When auto
// is taken off while units act, the units that have started their turn finish it, the units that
// have not keep their turn, and the player goes on with them as always.

import (
    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/inpututil"
)

// turns the key off
const AutoToggle = true
var AutoToggleKey = ebiten.KeySpace

// the army a player at the keyboard leads, or nil
func (model *CombatModel) humanArmy() *Army {
    for _, army := range []*Army{model.AttackingArmy, model.DefendingArmy} {
        if army != nil && army.Player != nil && !army.Player.IsAI() {
            return army
        }
    }

    return nil
}

// looks at the key. called wherever the clock of the battle moves, counts once per tick of the game
func (combat *CombatScreen) updateAutoToggle() {
    if !AutoToggle {
        return
    }

    tick := ebiten.Tick()
    if combat.autoToggleSeen && combat.autoToggleTick == tick {
        return
    }
    combat.autoToggleSeen = true
    combat.autoToggleTick = tick

    if !inpututil.IsKeyJustPressed(AutoToggleKey) {
        return
    }

    // not while a window is open over the battle
    if combat.UI != nil && combat.UI.GetHighestLayerValue() > 0 {
        return
    }

    combat.ToggleAuto()
}

func (combat *CombatScreen) ToggleAuto() {
    army := combat.Model.humanArmy()
    if army != nil {
        army.Auto = !army.Auto
    }
}

// a unit that acts together with others is about to start its turn: true if it still should. a
// unit of an army that was taken off auto keeps its turn for the player
func (combat *CombatScreen) startsTogether(unit *ArmyUnit) bool {
    if combat.Model.IsAIControlled(unit) {
        return true
    }

    if combat.heldUnits != nil {
        combat.heldUnits[unit] = true
    }
    return false
}
