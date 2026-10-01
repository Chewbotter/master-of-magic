package game

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    herolib "github.com/kazzmir/master-of-magic/game/magic/hero"
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

    island := scenario.units(player, 2, 2, units.FloatingIsland, units.HighMenSpearmen, units.HighMenSpearmen)
    if len(classicDrowned(island)) != 0 {
        test.Errorf("a Floating Island carries its riders")
    }

    hero := herolib.MakeHero(units.MakeOverworldUnit(units.HeroBrax, 3, 3, data.PlaneArcanus), herolib.HeroBrax, "Brax")
    withHero := append(scenario.units(player, 3, 3, units.Trireme, units.HighMenSpearmen, units.HighMenSpearmen), player.AddUnit(hero))
    if drowned := classicDrowned(withHero); drowned[hero] {
        test.Errorf("a hero never drowns while there are seats, even after the last is taken")
    }
}
