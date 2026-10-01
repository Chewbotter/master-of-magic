package game

import (
    "testing"

    buildinglib "github.com/kazzmir/master-of-magic/game/magic/building"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    herolib "github.com/kazzmir/master-of-magic/game/magic/hero"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/setup"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
)

// a building sold or destroyed is remembered for the turn: it still keeps Wall of Stone off and
// counts for the Nightshade; the count is the one of the start of the turn and is never cleared
func TestClassicRemovedThisTurn(test *testing.T) {
    scenario := makeChewScenario(test, 50, chewIsland...)
    owner := scenario.wizard("Kali", data.BannerGreen)
    city := scenario.city(owner, 5, 5, 4, false)
    city.CatchmentProvider = scenario.Model.ArcanusMap

    city.AddBuilding(buildinglib.BuildingCityWalls)
    city.AddBuilding(buildinglib.BuildingShrine)
    city.RemoveBuilding(buildinglib.BuildingCityWalls)
    city.RemoveBuilding(buildinglib.BuildingShrine)
    if city.Buildings.Contains(buildinglib.BuildingCityWalls) || !city.HasOrRemovedThisTurn(buildinglib.BuildingCityWalls) {
        test.Fatalf("walls sold this turn are gone and remembered")
    }
    if classicNightshadeCount(city) < 0 {
        test.Errorf("a shrine sold this turn still lets the city count its Nightshade")
    }

    city.NightshadeCount = 3
    game := &Game{Model: scenario.Model}
    game.classicCityRecords()
    if city.HasOrRemovedThisTurn(buildinglib.BuildingCityWalls) {
        test.Errorf("the turn forgets what was sold")
    }
    if city.NightshadeCount != 0 {
        test.Errorf("the shrine sold in the turn still counted: the city has no Nightshade, count %v", city.NightshadeCount)
    }

    // no building of religion or learning: the count is left as it was
    city.NightshadeCount = 2
    game.classicCityRecords()
    if city.NightshadeCount != 2 {
        test.Errorf("a city without the buildings keeps its old count, has %v", city.NightshadeCount)
    }
}

// Spell Blast cast by the human costs the progress of a spell of both books, whatever its discount
func TestClassicSpellBlastHumanPrice(test *testing.T) {
    scenario := makeChewScenario(test, 50, chewIsland...)
    human := playerlib.MakePlayer(setup.WizardCustom{Name: "Merlin", Banner: data.BannerRed}, true, scenario.Width, scenario.Height, make(map[herolib.HeroType]string), scenario.Model)
    scenario.Model.Players = append(scenario.Model.Players, human)
    target := scenario.wizard("Kali", data.BannerGreen)

    target.CastingSpell = spellbook.Spell{Name: "Fire Elemental", Magic: data.ChaosMagic, CastCost: 100, Eligibility: spellbook.EligibilityBoth}
    target.CastingSpellProgress = 60
    human.Mana = 60
    if !classicSpellBlast(human, target) {
        test.Fatalf("60 mana blasts a spell 60 into its casting")
    }
    if human.Mana != 0 {
        test.Errorf("paid %v, the progress is 60", 60 - human.Mana)
    }
}
