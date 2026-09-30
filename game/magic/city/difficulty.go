package city

// The original's bonuses of computer wizards by difficulty (ReMoM MoX/src/MOM_DAT.c:
// difficulty_modifiers_table; used in MoM/src/CITYCALC.c: Do_City_Calculations, City_Growth_Rate,
// Outpost growth in Do_All_Cities_Next_Turn, Player_Resource_Income_Total and the node power of
// Player_Magic_Power_Income_Total). The rules in words: docs/mod/difficulty.md. The code is ours.

import (
    "github.com/kazzmir/master-of-magic/game/magic/data"
)

// false: every owner's cities as the clone has them, whatever the difficulty
var ClassicDifficulty = true

// percent of what the city or the wizard would have
type DifficultyModifiers struct {
    PopulationGrowth int
    OutpostGrowth int
    Production int
    Gold int
    Mana int
    Research int
    Food int
    Maintenance int
}

// by the original's level: intro, easy, normal, hard, impossible
var DifficultyTable = [5]DifficultyModifiers{
    {PopulationGrowth: 100, OutpostGrowth: 100, Production: 100, Gold: 100, Mana: 100, Research: 100, Food: 100, Maintenance: 100},
    {PopulationGrowth: 100, OutpostGrowth: 100, Production: 125, Gold: 125, Mana: 125, Research: 100, Food: 125, Maintenance: 90},
    {PopulationGrowth: 150, OutpostGrowth: 150, Production: 150, Gold: 150, Mana: 150, Research: 100, Food: 150, Maintenance: 75},
    {PopulationGrowth: 200, OutpostGrowth: 200, Production: 200, Gold: 200, Mana: 200, Research: 110, Food: 200, Maintenance: 60},
    {PopulationGrowth: 400, OutpostGrowth: 400, Production: 400, Gold: 400, Mana: 400, Research: 150, Food: 400, Maintenance: 30},
}

// the original's level of a difficulty of the fork, which has Extreme between Hard and
// Impossible: Extreme counts as Impossible (MY CALL, as in the rules of diplomacy)
func OriginalDifficulty(difficulty data.DifficultySetting) int {
    switch difficulty {
        case data.DifficultyIntro: return 0
        case data.DifficultyEasy: return 1
        case data.DifficultyAverage: return 2
        case data.DifficultyHard: return 3
    }
    return 4
}

// who owns a city, as the table needs it
type OwnerDifficulty struct {
    // the original's level, 0 to 4
    Level int
    // a computer wizard (not the neutral player)
    Computer bool
    Neutral bool
}

// the owner of a city (the player) tells the city what the table needs to know of it
type OwnerDifficultyProvider interface {
    // false when there is no difficulty (a battle of the debug list)
    OwnerDifficulty() (OwnerDifficulty, bool)
}

// scale a value by a percent of the table, as the original does (value * percent / 100)
func ScaleByDifficulty(value int, percent int) int {
    return value * percent / 100
}

func (city *City) ownerDifficulty() (OwnerDifficulty, bool) {
    if !ClassicDifficulty || city.ReignProvider == nil {
        return OwnerDifficulty{}, false
    }

    provider, ok := city.ReignProvider.(OwnerDifficultyProvider)
    if !ok {
        return OwnerDifficulty{}, false
    }

    return provider.OwnerDifficulty()
}

// the modifiers of a city of a computer wizard, false for the human and the neutral player
func (city *City) computerModifiers() (DifficultyModifiers, bool) {
    owner, ok := city.ownerDifficulty()
    if !ok || !owner.Computer {
        return DifficultyModifiers{}, false
    }

    return DifficultyTable[owner.Level], true
}

// food, production, gold, mana and research of a computer wizard's city times the table
func (city *City) difficultyYield(value int, percent func(DifficultyModifiers) int) int {
    modifiers, ok := city.computerModifiers()
    if !ok {
        return value
    }
    return ScaleByDifficulty(value, percent(modifiers))
}

// City_Growth_Rate: growth of a computer wizard's city times the table; a neutral city grows half as
// fast and not at all from (level + 1) * 2 citizens on. Only growth, a city that starves loses what
// it loses (the original keeps its growth at 0 or more and shrinks cities elsewhere)
func (city *City) difficultyGrowth(growth int) int {
    if growth <= 0 {
        return growth
    }

    owner, ok := city.ownerDifficulty()
    if !ok {
        return growth
    }

    if owner.Neutral {
        growth /= 2
        if city.Citizens() >= (owner.Level + 1) * 2 {
            return 0
        }
        return growth
    }

    if owner.Computer {
        return ScaleByDifficulty(growth, DifficultyTable[owner.Level].PopulationGrowth)
    }

    return growth
}

// the chance an outpost grows, times the table for EVERY owner, the human's outposts too (as the
// original has it)
func (city *City) difficultyOutpostGrowth(chance float64) float64 {
    owner, ok := city.ownerDifficulty()
    if !ok {
        return chance
    }
    return chance * float64(DifficultyTable[owner.Level].OutpostGrowth) / 100
}
