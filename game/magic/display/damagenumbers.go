package display

// The setting "Damage numbers": in a battle a number rises from a unit that is hit. The original
// has none. See game/magic/combat/damagenumbers.go.
//
// On unless it was turned off (user, 2026-09-29). The file of settings keeps the opposite,
// "damage-numbers-off", so a file that says nothing of it means on.

// true when the numbers are shown
func DamageNumbers() bool {
    return Current.DamageNumbers()
}

func (settings *Settings) DamageNumbers() bool {
    return !settings.DamageNumbersOff
}

func (settings *Settings) SetDamageNumbers(show bool) {
    settings.DamageNumbersOff = !show
    settings.Save()
}
