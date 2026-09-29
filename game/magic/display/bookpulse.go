package display

// The setting "Pulsing spellbook text": in the spellbooks the spell under the mouse pulses to
// the color of its realm, and the spell that is being cast or researched pulses lighter. Off,
// nothing pulses, as in the original: the spell under the mouse is not marked, the one that is
// being cast or researched is lighter and stays so. See game/magic/spellbook/hover.go.
//
// On unless it was turned off (user, 2026-09-29). The file of settings keeps the opposite,
// "spellbook-text-still", so a file that says nothing of it means on.

// true when the text of the spellbooks pulses
func SpellbookPulse() bool {
    return Current.SpellbookPulse()
}

func (settings *Settings) SpellbookPulse() bool {
    return !settings.SpellbookTextStill
}

func (settings *Settings) SetSpellbookPulse(pulse bool) {
    settings.SpellbookTextStill = !pulse
    settings.Save()
}
