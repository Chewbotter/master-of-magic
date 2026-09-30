package game

// The power of the nodes of a computer wizard times the mana of the original's difficulty table
// (ReMoM MoM/src/CITYCALC.c: Player_Magic_Power_Income_Total, applied after the masteries and
// before the power of the cities; city/difficulty.go has the table), and the original's cap of the
// reserves (MoM/src/NEXTTURN.c: Next_Turn_Calc, MAX_GOLD_RESERVE, MAX_MANA_RESERVE). The code is ours.

import (
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

func difficultyNodePower(player *playerlib.Player, power float64) float64 {
    modifiers, ok := player.DifficultyModifiers()
    if !ok || power <= 0 {
        return power
    }
    return power * float64(modifiers.Mana) / 100
}

// false: gold and mana without a cap, as the fork had them
var CapReserves = true

const reserveMost = 30000

// every player's gold and mana at most 30000 when a new turn starts, the human's too
func capReserves(players []*playerlib.Player) {
    if !CapReserves {
        return
    }
    for _, player := range players {
        player.Gold = min(player.Gold, reserveMost)
        player.Mana = min(player.Mana, reserveMost)
    }
}
