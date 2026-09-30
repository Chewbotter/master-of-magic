package game

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// a banished computer wizard returns to its city with the strongest garrison; with no garrison
// anywhere it starts the Spell of Return again
func TestSpellOfReturnCity(test *testing.T) {
    scenario := makeChewScenario(test, 100, chewIsland...)
    wizard := scenario.wizard("Merlin", data.BannerRed)
    weak := scenario.city(wizard, 4, 5, 4, false)
    strong := scenario.city(wizard, 12, 6, 4, false)
    game := &Game{Model: scenario.Model}

    if city := game.returnCity(wizard); city != nil {
        test.Errorf("no garrison anywhere, yet %v", city.Name)
    }

    scenario.units(wizard, weak.X, weak.Y, units.HighMenSpearmen)
    scenario.units(wizard, strong.X, strong.Y, units.HighMenSwordsmen, units.HighMenSwordsmen)
    if city := game.returnCity(wizard); city != strong {
        test.Errorf("the city with the strongest garrison should be taken, not %v", city)
    }

    empty := scenario.wizard("Kali", data.BannerGreen)
    scenario.city(empty, 8, 8, 3, false)
    empty.KnownSpells.AddSpell(spellbook.Spell{Name: "Spell of Return"})
    game.aiSpellOfReturn(&GameEventSelectLocationForSpell{Player: empty}, func(int, int) {
        test.Errorf("no city should be picked")
    })
    if empty.CastingSpell.Name != "Spell of Return" {
        test.Errorf("the wizard should start the Spell of Return again, casts %q", empty.CastingSpell.Name)
    }
}
