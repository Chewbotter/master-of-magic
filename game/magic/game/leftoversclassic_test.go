package game

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// Retreat_From_Combat: a trireme of 2 seats carries 2 riders, the third drowns; heroes take no seat
func TestClassicDrowned(test *testing.T) {
    scenario := makeChewScenario(test, 50, chewIsland...)
    player := scenario.wizard("Merlin", data.BannerRed)
    stack := scenario.units(player, 0, 0, units.Trireme, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen)
    drowned := classicDrowned(stack)
    if len(drowned) != 1 || !drowned[stack[3]] {
        test.Errorf("the rider beyond the seats drowns, drowned %v", len(drowned))
    }

    alone := scenario.units(player, 1, 1, units.HighMenSpearmen)
    if len(classicDrowned(alone)) != 1 {
        test.Errorf("with no ship every rider drowns")
    }
}
