package game

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// the checks of every turn find gold below 0, a square over 9 and two players on one square
func TestSimCheck(test *testing.T) {
    scenario := makeChewScenario(test, 50, chewIsland...)
    first := scenario.wizard("Merlin", data.BannerRed)
    second := scenario.wizard("Kali", data.BannerGreen)
    game := &Game{Model: scenario.Model, headless: true, Stats: MakeSimStats()}

    game.SimCheck()
    if len(game.Stats.ViolationList()) != 0 {
        test.Fatalf("a clean game has no broken states: %+v", game.Stats.ViolationList())
    }

    first.Gold = -5
    var kinds []units.Unit
    for range 10 {
        kinds = append(kinds, units.HighMenSpearmen)
    }
    scenario.units(first, 5, 5, kinds...)
    scenario.units(second, 7, 7, units.HighMenSpearmen)
    // a unit of the second wizard on the first's square, in a stack of its own
    stray := scenario.units(second, 7, 7, units.HighMenSpearmen)[0]
    stray.SetX(5)
    stray.SetY(5)

    game.SimCheck()
    found := make(map[string]bool)
    for _, violation := range game.Stats.ViolationList() {
        found[violation.Kind] = true
    }
    for _, kind := range []string{"gold below 0", "more than 9 units on a square", "unit away from its stack"} {
        if !found[kind] {
            test.Errorf("the checks should find %q, found %v", kind, found)
        }
    }
}
