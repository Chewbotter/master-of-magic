package player

// The player's side of the original's difficulty table (city/difficulty.go): who gets the bonuses,
// and the upkeep of armies and enchantments of computer players (ReMoM MoM/src/CITYCALC.c:
// Player_Resource_Income_Total). The code is ours.

import (
    "github.com/kazzmir/master-of-magic/game/magic/data"
    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
)

// what a game tells its players of its difficulty (the game model)
type difficultyProvider interface {
    GetDifficulty() data.DifficultySetting
}

// the owner of a city as the table needs it (citylib.OwnerDifficultyProvider)
func (player *Player) OwnerDifficulty() (citylib.OwnerDifficulty, bool) {
    provider, ok := player.GlobalEnchantmentsProvider.(difficultyProvider)
    if !ok {
        return citylib.OwnerDifficulty{}, false
    }

    neutral := player.IsNeutral()

    return citylib.OwnerDifficulty{
        Level: citylib.OriginalDifficulty(provider.GetDifficulty()),
        Computer: !player.IsHuman() && !neutral,
        Neutral: !player.IsHuman() && neutral,
    }, true
}

// the modifiers of a computer wizard, false for the human and the neutral player
func (player *Player) DifficultyModifiers() (citylib.DifficultyModifiers, bool) {
    if !citylib.ClassicDifficulty {
        return citylib.DifficultyModifiers{}, false
    }

    owner, ok := player.OwnerDifficulty()
    if !ok || !owner.Computer {
        return citylib.DifficultyModifiers{}, false
    }

    return citylib.DifficultyTable[owner.Level], true
}

// the gold, food and mana every player but the human pays for its units and enchantments, times the
// maintenance of the table, AS THE ORIGINAL'S SUMMARY COUNTS IT (Player_Resource_Income_Total, the
// neutral player too): the computer players plan with it; what they really pay is full
func (player *Player) planningUpkeep(planning bool, upkeep int) int {
    if !planning {
        return upkeep
    }
    return player.difficultyUpkeep(upkeep)
}

// Unit_Gold_Upkeep: the gold a unit of any player but the human costs, three quarters at Hard, half
// at Impossible (and Extreme), per unit, rounded down
func (player *Player) difficultyUnitGold(upkeep int) int {
    if !citylib.ClassicDifficulty {
        return upkeep
    }
    owner, ok := player.OwnerDifficulty()
    if !ok || (!owner.Computer && !owner.Neutral) {
        return upkeep
    }
    switch owner.Level {
        case 3: return upkeep * 3 / 4
        case 4: return upkeep / 2
    }
    return upkeep
}

func (player *Player) difficultyUpkeep(upkeep int) int {
    if !citylib.ClassicDifficulty {
        return upkeep
    }
    owner, ok := player.OwnerDifficulty()
    if !ok || (!owner.Computer && !owner.Neutral) {
        return upkeep
    }
    return citylib.ScaleByDifficulty(upkeep, citylib.DifficultyTable[owner.Level].Maintenance)
}
