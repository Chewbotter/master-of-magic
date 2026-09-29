package display

// The setting "Single strikes": how units strike in a battle.
//
// On: an attack is one swing, the figures step back, wind up, rush forward and stay where their
// blow lands. Off, which it is unless it was turned on (user, 2026-09-28; it was on at first): the
// figures strike again and again, fast, for as long as the attack lasts. See
// game/magic/combat/strikeswing.go and strikeclassic.go.
//
// A file of settings that says nothing of it means off. The first days of the setting the file
// kept it as "repeated-strikes"; that is not read any more, so everybody starts with off.

// true when an attack is one swing
func SingleStrikes() bool {
    return Current.SingleStrikes()
}

func (settings *Settings) SingleStrikes() bool {
    return settings.SingleStrikesOn
}

func (settings *Settings) SetSingleStrikes(single bool) {
    settings.SingleStrikesOn = single
    settings.Save()
}
