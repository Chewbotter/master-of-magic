package display

// The setting "Single strikes": how units strike in a battle.
//
// On, which it is unless it was turned off: an attack is one swing, the figures step back, wind
// up, rush forward and stay where their blow lands. Off: the figures strike again and again, fast,
// for as long as the attack lasts. See game/magic/combat/strikeswing.go and strikeclassic.go.
//
// The file of the settings keeps whether it was turned OFF, so that a file that says nothing of
// it, as every file from before the setting, means on.

// true when an attack is one swing
func SingleStrikes() bool {
    return Current.SingleStrikes()
}

func (settings *Settings) SingleStrikes() bool {
    return !settings.RepeatedStrikes
}

func (settings *Settings) SetSingleStrikes(single bool) {
    settings.RepeatedStrikes = !single
    settings.Save()
}
