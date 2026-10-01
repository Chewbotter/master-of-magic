package combat

// RIGHT CLICK CANCELS A SPELL (user, 2026-09-30: "When a spell is selected and the 'cancel' button is
// active, also make right click cancel the spell"), modern controls: while the target of a spell is
// picked (doSelectUnit, doSelectTile), a right click does what the cancel button does. Not while a
// window is open over the battle, where a right click is that window's

import (
    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/game/magic/inputmanager"
)

const RightClickCancelsSpell = true

// true when a right click of this tick cancels the spell whose target is being picked
func (combat *CombatScreen) rightClickCancels() bool {
    if !RightClickCancelsSpell || !display.ModernControls() {
        return false
    }
    return combat.UI.GetHighestLayerValue() == 0 && inputmanager.RightClick()
}
