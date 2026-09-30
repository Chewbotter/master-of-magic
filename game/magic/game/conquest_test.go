package game

import (
    "fmt"
    "testing"

    buildinglib "github.com/kazzmir/master-of-magic/game/magic/building"
    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// a city of the scenario that can change owner
func conquestCity(scenario *chewScenario, player *playerlib.Player, x int, y int, citizens int, fortress bool) *citylib.City {
    city := citylib.MakeCity(fmt.Sprintf("%v %v,%v", player.Wizard.Name, x, y), x, y, data.RaceHighMen, nil, &NoCatchment{}, &NoServices{}, player)
    city.Plane = data.PlaneArcanus
    city.Population = citizens * 1000
    if fortress {
        city.Buildings.Insert(buildinglib.BuildingFortress)
    }
    player.AddCity(city)
    return city
}

// Banish_Wizard and Resolve_Wizard_Conquest (conquest.go)
func TestConquest(test *testing.T) {
    scenario := makeChewScenario(test, 100, chewIsland...)
    winner := scenario.wizard("Merlin", data.BannerRed)
    winner.Human = true
    loser := scenario.wizard("Kali", data.BannerGreen)
    scenario.wizard("Raiders", data.BannerBrown)
    game := &Game{Model: scenario.Model, headless: true}

    loser.KnownSpells.AddSpell(spellbook.Spell{Name: "Spell of Return", CastCost: 500})
    // casting skill 40: (sqrt(4 * power - 3) + 1) / 2
    loser.CastingSkillPower = 1600
    loser.Mana = 100
    loser.Gold = 300
    winner.Mana = 10

    fortress := conquestCity(scenario, loser, 4, 5, 6, true)
    other := conquestCity(scenario, loser, 12, 6, 4, false)
    scenario.units(loser, other.X, other.Y, units.HighMenSpearmen, units.WarBear)
    scenario.units(loser, 8, 8, units.HighMenSwordsmen)

    capture := func(city *citylib.City) {
        before := conquestBeforeCapture(loser, city)
        ChangeCityOwner(city, loser, winner, ChangeCityRemoveOwnerEnchantments)
        game.conquestAfterCapture(nil, loser, winner, city, before)
    }

    // the fortress with a city left: banished
    capture(fortress)
    if !loser.Banished || loser.Defeated {
        test.Fatalf("the loser should be banished, banished %v defeated %v", loser.Banished, loser.Defeated)
    }
    if loser.CastingSpell.Name != "Spell of Return" {
        test.Errorf("the loser should cast the Spell of Return, casts %q", loser.CastingSpell.Name)
    }
    if loser.Mana != 50 || winner.Mana != 60 {
        test.Errorf("the mana should be halved and given: loser %v winner %v", loser.Mana, winner.Mana)
    }
    if winner.Fame != 5 || len(winner.DefeatedWizards) != 1 {
        test.Errorf("the winner: fame %v, conquered %v", winner.Fame, winner.DefeatedWizards)
    }

    // its last city: defeated, and the human wins
    capture(other)
    if !loser.Defeated {
        test.Fatalf("the loser should be defeated")
    }
    if winner.Gold != 150 || winner.Mana != 85 {
        test.Errorf("the winner should have half the gold and mana: gold %v mana %v", winner.Gold, winner.Mana)
    }
    if loser.UnitCount() != 0 {
        test.Errorf("every unit of the loser should be gone, %v left", loser.UnitCount())
    }
    if len(winner.DefeatedWizards) != 1 || winner.Fame != 10 {
        test.Errorf("the loser counts once and the fame is 10: %v, %v", winner.DefeatedWizards, winner.Fame)
    }
    if game.State != GameStateQuit {
        test.Errorf("the last computer wizard defeated should end the game")
    }
}

// a computer wizard with a casting skill under 40 can not come back: defeated at once, and its
// other cities become the neutral player's
func TestConquestWeakWizard(test *testing.T) {
    scenario := makeChewScenario(test, 100, chewIsland...)
    winner := scenario.wizard("Merlin", data.BannerRed)
    loser := scenario.wizard("Kali", data.BannerGreen)
    other := scenario.wizard("Sharee", data.BannerBlue)
    neutral := scenario.wizard("Raiders", data.BannerBrown)
    game := &Game{Model: scenario.Model, headless: true}

    conquestCity(scenario, other, 8, 8, 4, true)
    loser.Mana = 100
    fortress := conquestCity(scenario, loser, 4, 5, 6, true)
    second := conquestCity(scenario, loser, 12, 6, 4, false)
    scenario.units(loser, second.X, second.Y, units.HighMenSpearmen, units.WarBear)

    before := conquestBeforeCapture(loser, fortress)
    ChangeCityOwner(fortress, loser, winner, ChangeCityRemoveOwnerEnchantments)
    game.conquestAfterCapture(nil, loser, winner, fortress, before)

    if !loser.Defeated {
        test.Fatalf("a wizard with casting skill under 40 should be defeated")
    }
    if _, ok := neutral.Cities[second.GetPlanePoint()]; !ok {
        test.Errorf("its other city should be the neutral player's")
    }
    if count := len(neutral.GetUnits(second.X, second.Y, second.Plane)); count != 1 {
        test.Errorf("the spearmen in the city should be the neutral player's, the bear dismissed: %v units", count)
    }
    if game.State == GameStateQuit {
        test.Errorf("a computer wizard with a city is left, the game goes on")
    }
}
