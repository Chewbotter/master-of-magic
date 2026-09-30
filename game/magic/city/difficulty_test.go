package city

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/lib/fraction"
)

// a reign with an owner for the difficulty table
type difficultyReign struct {
    NoReign
    Owner OwnerDifficulty
    Known bool
}

func (reign *difficultyReign) OwnerDifficulty() (OwnerDifficulty, bool) {
    return reign.Owner, reign.Known
}

func makeDifficultyCity(owner OwnerDifficulty, known bool) *City {
    reign := &difficultyReign{NoReign: NoReign{TaxRate: fraction.FromInt(1)}, Owner: owner, Known: known}
    city := MakeCity("Test City", 10, 10, data.RaceHighMen, nil, &Catchment{Map: makeSimpleMap()}, &NoCities{}, reign)
    city.Population = 6000
    city.Farmers = 3
    city.Workers = 3
    city.ResetCitizens()
    return city
}

func TestDifficultyTable(test *testing.T) {
    human := makeDifficultyCity(OwnerDifficulty{Level: 2}, true)
    computer := makeDifficultyCity(OwnerDifficulty{Level: 2, Computer: true}, true)
    unknown := makeDifficultyCity(OwnerDifficulty{Level: 2, Computer: true}, false)

    if human.FoodProductionRate() * 150 / 100 != computer.FoodProductionRate() {
        test.Errorf("food of a computer wizard at normal: %v, of the human %v", computer.FoodProductionRate(), human.FoodProductionRate())
    }
    if human.WorkProductionRate() * 1.5 != computer.WorkProductionRate() {
        test.Errorf("production of a computer wizard at normal: %v, of the human %v", computer.WorkProductionRate(), human.WorkProductionRate())
    }
    if unknown.FoodProductionRate() != human.FoodProductionRate() {
        test.Errorf("a city without a difficulty got a bonus")
    }

    ClassicDifficulty = false
    if computer.FoodProductionRate() != human.FoodProductionRate() {
        test.Errorf("the switch did not take the bonus away")
    }
    ClassicDifficulty = true

    // growth: times the table for computer wizards
    if growth := human.difficultyGrowth(60); growth != 60 {
        test.Errorf("growth of the human changed to %v", growth)
    }
    if growth := computer.difficultyGrowth(60); growth != 90 {
        test.Errorf("growth of a computer wizard at normal: %v, want 90", growth)
    }
    if growth := computer.difficultyGrowth(-50); growth != -50 {
        test.Errorf("starving changed to %v", growth)
    }

    // neutral cities: half, none from (level + 1) * 2 citizens on
    neutral := makeDifficultyCity(OwnerDifficulty{Level: 2, Neutral: true}, true)
    if growth := neutral.difficultyGrowth(60); growth != 0 {
        test.Errorf("a neutral city of 6 at normal grew by %v", growth)
    }
    neutral.Population = 5000
    neutral.Farmers = 3
    neutral.Workers = 2
    neutral.ResetCitizens()
    if growth := neutral.difficultyGrowth(60); growth != 30 {
        test.Errorf("a neutral city of 5 at normal grew by %v, want 30", growth)
    }

    // outposts: every owner
    if chance := human.difficultyOutpostGrowth(0.1); chance < 0.1499 || chance > 0.1501 {
        test.Errorf("outpost of the human at normal: %v, want 0.15", chance)
    }

    if OriginalDifficulty(data.DifficultyExtreme) != 4 || OriginalDifficulty(data.DifficultyAverage) != 2 {
        test.Errorf("levels of the fork")
    }
}
