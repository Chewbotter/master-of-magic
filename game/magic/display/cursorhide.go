package display

// The setting "Hide cursor on attack": in a battle the cursor fades away after a click that sends
// a unit to strike or to shoot, until the mouse moves. See game/magic/combat/cursorhide.go.
//
// On unless it was turned off (user, 2026-09-29). The file of settings keeps the opposite,
// "cursor-stays-on-attack", so a file that says nothing of it means on.

// true when the cursor goes away after such a click
func HideCursorOnAttack() bool {
    return Current.HideCursorOnAttack()
}

func (settings *Settings) HideCursorOnAttack() bool {
    return !settings.CursorStaysOnAttack
}

func (settings *Settings) SetHideCursorOnAttack(hide bool) {
    settings.CursorStaysOnAttack = !hide
    settings.Save()
}
