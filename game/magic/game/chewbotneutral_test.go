package game

import (
    "image"
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// a neutral stack outside the cities goes for the city of a wizard worth the most: 10 less its
// units less its distance (NPC_Destinations)
func TestChewbotNeutralGoesForCity(test *testing.T) {
    scenario := makeChewScenario(test, 60, chewIsland...)
    wizard := scenario.wizard("Merlin", data.BannerRed)
    raiders := scenario.wizard("Raiders", data.BannerBrown)
    scenario.city(wizard, 4, 5, 4, true)
    scenario.units(wizard, 4, 5, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen)
    scenario.city(wizard, 15, 8, 4, false)
    // 5 from the guarded city (10 - 4 - 5 = 1), 6 from the empty one (10 - 0 - 6 = 4)
    stack := scenario.units(raiders, 9, 5, units.HighMenSwordsmen, units.HighMenSwordsmen)

    decisions := scenario.turn(raiders)
    to, ok := chewDestinationOf(decisions, stack[0])
    if !ok || to != image.Pt(15, 8) {
        test.Errorf("the raiders should go for the empty city at 15,8; orders: %v", chewDescribe(decisions))
    }
}

// a neutral stack of walkers with no city of a wizard on its landmass is disbanded
func TestChewbotNeutralWithNowhereToGo(test *testing.T) {
    scenario := makeChewScenario(test, 60,
        "~~~~~~~~~~~~~~~~~~~~",
        "~.....~~~~~~~......~",
        "~.....~~~~~~~......~",
        "~.....~~~~~~~......~",
        "~~~~~~~~~~~~~~~~~~~~",
    )
    wizard := scenario.wizard("Merlin", data.BannerRed)
    raiders := scenario.wizard("Raiders", data.BannerBrown)
    scenario.city(wizard, 2, 2, 4, true)
    scenario.units(raiders, 16, 2, units.HighMenSwordsmen)

    scenario.turn(raiders)
    count := 0
    for range raiders.Units() {
        count += 1
    }
    if count != 0 {
        test.Errorf("the raiders on the other island should be gone, %v units left", count)
    }
}

// a neutral city with more units than its people loses its cheapest one a turn (NPC_Excess_Garrison)
func TestChewbotNeutralExcessGarrison(test *testing.T) {
    scenario := makeChewScenario(test, 60, chewIsland...)
    wizard := scenario.wizard("Merlin", data.BannerRed)
    raiders := scenario.wizard("Raiders", data.BannerBrown)
    scenario.city(wizard, 4, 5, 4, true)
    city := scenario.city(raiders, 15, 8, 2, false)
    // the farmers of the turn need the city's land
    city.CatchmentProvider = scenario.Model.ArcanusMap
    city.CityServices = scenario.Model
    city.ReignProvider = raiders
    scenario.units(raiders, 15, 8, units.HighMenSwordsmen, units.HighMenSpearmen, units.HighMenSwordsmen)

    scenario.turn(raiders)
    left := map[string]int{}
    for unit := range raiders.Units() {
        left[unit.GetName()] += 1
    }
    if left["Spearmen"] != 0 || left["Swordsmen"] != 2 {
        test.Errorf("the spearmen should have gone from a city of 2 with 3 units, left %v", left)
    }
}
