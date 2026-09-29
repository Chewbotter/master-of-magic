package combat

// The cursor goes away after a click that sends a unit to strike (the crossed swords) or to
// shoot (the arrow), so the fight is not under it, and is back when the mouse moves. Not in the
// original (user, 2026-09-29; the shots a little later: "let's extend it to ranged attacks,
// too"). With the modern controls.

import (
    "image"

    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/game/magic/inputmanager"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: the cursor stays, whatever the setting "Hide cursor on attack" says (display/cursorhide.go)
const CursorHidesOnAttack = true
// seconds the cursor takes to fade out
const cursorHideTime = 0.12

type cursorHide struct {
    Hidden bool
    // where the mouse was at the click, and the tick of the battle of the click
    At image.Point
    Start uint64
}

// the click that starts a strike or a shot
func (combat *CombatScreen) hideCursor() {
    if !CursorHidesOnAttack || !display.ModernControls() || !display.HideCursorOnAttack() {
        return
    }

    x, y := inputmanager.MousePosition()
    combat.cursorHidden = cursorHide{Hidden: true, At: image.Pt(x, y), Start: combat.Counter}
}

// how much of the cursor shows, 1 to 0
func (combat *CombatScreen) cursorShows() float32 {
    hide := &combat.cursorHidden
    if !hide.Hidden {
        return 1
    }

    x, y := inputmanager.MousePosition()
    if x != hide.At.X || y != hide.At.Y {
        // the mouse has moved
        hide.Hidden = false
        return 1
    }

    seconds := 0.0
    if combat.Counter > hide.Start {
        seconds = float64(combat.Counter - hide.Start) / float64(max(1, ebiten.TPS()))
    }

    return float32(max(0, 1 - seconds / cursorHideTime))
}
