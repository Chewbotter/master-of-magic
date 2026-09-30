package game

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// Unit_Gold_Upkeep: a computer player's unit costs three quarters of its gold at Hard, half at
// Impossible, each unit rounded down; the human pays in full
func TestDifficultyUnitGold(test *testing.T) {
    scenario := makeChewScenario(test, 100, chewIsland...)
    wizard := scenario.wizard("Merlin", data.BannerRed)
    army := scenario.units(wizard, 8, 5, units.HighMenCavalry, units.HighMenCavalry, units.HighMenSwordsmen)

    full := 0
    threeQuarters := 0
    half := 0
    for _, unit := range army {
        full += unit.GetUpkeepGold()
        threeQuarters += unit.GetUpkeepGold() * 3 / 4
        half += unit.GetUpkeepGold() / 2
    }

    for _, check := range []struct {
        Difficulty data.DifficultySetting
        Want int
    }{
        {data.DifficultyAverage, full},
        {data.DifficultyHard, threeQuarters},
        {data.DifficultyImpossible, half},
    } {
        scenario.Model.Settings.Difficulty = check.Difficulty
        got := wizard.TotalUnitUpkeepGold() + wizard.GetFame()
        if got != check.Want {
            test.Errorf("difficulty %v: gold upkeep %v, want %v", check.Difficulty, got, check.Want)
        }
    }

    wizard.Human = true
    scenario.Model.Settings.Difficulty = data.DifficultyImpossible
    if got := wizard.TotalUnitUpkeepGold() + wizard.GetFame(); got != full {
        test.Errorf("the human pays in full: %v, want %v", got, full)
    }
}
