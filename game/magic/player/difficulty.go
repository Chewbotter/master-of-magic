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

// gold, food and mana a computer wizard pays for its units and enchantments, times the maintenance
// of the table
func (player *Player) difficultyUpkeep(upkeep int) int {
    modifiers, ok := player.DifficultyModifiers()
    if !ok {
        return upkeep
    }
    return citylib.ScaleByDifficulty(upkeep, modifiers.Maintenance)
}
