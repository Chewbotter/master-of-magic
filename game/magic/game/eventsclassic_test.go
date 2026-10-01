package game

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

// Determine_Event: nothing before turn 50, nothing in the first 5 turns of a session
func TestClassicEventPressure(test *testing.T) {
    scenario := makeChewScenario(test, 40, chewIsland...)
    model := scenario.Model
    model.Settings.Difficulty = data.DifficultyImpossible
    for range 100 {
        if model.classicEventFires() {
            test.Fatalf("no event before turn 50")
        }
    }

    model.TurnNumber = 1000
    model.eventDelay = 0
    for range 5 {
        if model.classicEventFires() {
            test.Fatalf("no event in the first 5 turns of a session")
        }
    }
    // pressure 950 over 512: always
    if !model.classicEventFires() {
        test.Errorf("after the delay an event at a pressure over 512 is sure")
    }
}

// Get_Event_Victim: a bad event goes to the strong
func TestClassicEventVictim(test *testing.T) {
    scenario := makeChewScenario(test, 100, chewIsland...)
    strong := scenario.wizard("Merlin", data.BannerRed)
    weak := scenario.wizard("Kali", data.BannerGreen)
    strong.PowerHistory = []playerlib.WizardPower{{Army: 400, Magic: 100, SpellResearch: 0}}
    weak.PowerHistory = []playerlib.WizardPower{{Army: 0, Magic: 0, SpellResearch: 0}}

    for range 100 {
        if scenario.Model.classicEventVictim(false) != strong {
            test.Fatalf("a bad event should hit the only wizard with power")
        }
    }
    hitStrong := 0
    for range 1000 {
        if scenario.Model.classicEventVictim(true) == strong {
            hitStrong += 1
        }
    }
    // weights 100 and 600, halved to 50 and 300: the strong one about one time in 7
    if hitStrong > 250 {
        test.Errorf("a good event should mostly go to the weak, the strong got %v of 1000", hitStrong)
    }

    // a player that sits out (the human of a headless run) gets nothing: its windows wait for a click
    weak.Skip = true
    for range 100 {
        if scenario.Model.classicEventVictim(true) == weak {
            test.Fatalf("a player that sits out got an event")
        }
    }
}

// Event_Twiddle: a moon never ends before its 5th turn; the donation is 105 to 600
func TestClassicEventEndsAndDonation(test *testing.T) {
    for turns := uint64(0); turns < 5; turns++ {
        for range 50 {
            if classicMoonEnds(turns) {
                test.Fatalf("a moon ended after %v turns", turns)
            }
        }
    }
    for range 1000 {
        gold := classicDonation()
        if gold < 105 || gold > 600 || gold % 5 != 0 {
            test.Fatalf("a donation of %v", gold)
        }
    }
}
