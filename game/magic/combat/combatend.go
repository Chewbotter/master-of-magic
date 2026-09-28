package combat

// The end of a battle. Not in the original (user, 2026-09-28).
//
// When a battle is won or lost the battlefield is shown for a moment longer, so that how the
// fight has ended can be seen: the last figures fall, the last spell plays out. Then the result
// comes up as always. A battle that ends because an army flees ends at once.

import (
    "github.com/hajimehoshi/ebiten/v2"
)

// seconds
const combatEndDelay = 1.5

func combatEndTicks() int {
    return int(combatEndDelay * float64(max(1, ebiten.TPS())))
}

// true while the battle is still shown after its end. one tick of it
func (combat *CombatScreen) holdEnd(state CombatState) bool {
    if state != CombatStateAttackerWin && state != CombatStateDefenderWin {
        return false
    }

    if combat.endTicks >= combatEndTicks() {
        return false
    }
    combat.endTicks += 1

    // the clock of the battle goes on: figures fall, particles fly, numbers rise
    combat.Counter += 1
    combat.Model.UpdateProjectiles(combat.Counter)
    combat.UpdateDamageIndicators()
    combat.UpdateAnimations()
    combat.ProcessInput()
    combat.MouseState = CombatClickHud
    combat.UpdateMouseState()

    return true
}
