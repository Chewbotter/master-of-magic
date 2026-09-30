package city

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/lib/fraction"
)

// Apply_City_Changes: a whole person more at 1000, what is over is lost; one less below 0, the
// rest carried; a city of 1 is set to 50 people
func TestClassicApplyGrowth(test *testing.T) {
    city := MakeCity("Test", 10, 10, data.RaceHighMen, nil, &Catchment{Map: makeScenarioMap()}, &NoCities{}, &NoReign{TaxRate: fraction.FromInt(1)})

    for _, check := range []struct {
        Population, Growth, Want int
    }{
        {4600, 500, 5000},
        {4600, 300, 4900},
        {3050, -200, 2850},
        {1050, -200, 1050},
    } {
        city.Population = check.Population
        city.classicApplyGrowth(check.Growth)
        if city.Population != check.Want {
            test.Errorf("%v with %v: %v, want %v", check.Population, check.Growth, city.Population, check.Want)
        }
    }
}

// City_Growth_Rate: a city exactly at its maximum does not change, even starving
func TestClassicGrowthAtMaximum(test *testing.T) {
    city := MakeCity("Test", 10, 10, data.RaceHighMen, nil, &Catchment{Map: makeScenarioMap()}, &NoCities{}, &NoReign{TaxRate: fraction.FromInt(1)})
    city.Population = city.classicMaxSize() * 1000
    city.Farmers = 0
    city.Workers = city.Citizens()
    if city.FoodProductionRate() >= city.Citizens() {
        test.Fatalf("the city should be starving: food %v", city.FoodProductionRate())
    }
    if got := city.PopulationGrowthRate(); got != 0 {
        test.Errorf("growth at the maximum %v, want 0", got)
    }

    city.Population -= 1000
    if got := city.PopulationGrowthRate(); got >= 0 {
        test.Errorf("a starving city below its maximum should shrink, growth %v", got)
    }
}
