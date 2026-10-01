package game

// Prepare_All_Battle_Units (ReMoM Combat.c): every unit of a computer player that goes into a
// battle on the battlefield loses its order ("NO ORDERS"), attackers and defenders alike, so the
// ones that live through it (a stack that fled, or held out to the turn limit) plan anew the next
// turn. The battlefield is only ever the human's battles in the original; the quick resolution of
// the others leaves the loser nothing. Before this a computer stack that lost to the human without
// dying kept its target and attacked again every turn (found by test runs with every battle fought
// out, 2026-10-01).

import (
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// the original's rule (false: orders end only on arrival or death, as before)
var BattleEndsOrders = true

// the AI of a player that keeps orders from turn to turn (Chewbot)
type orderKeeper interface {
    EndOrders([]units.StackUnit)
}

// the orders of the units of both sides end, for the players an AI plays (the stand-in too)
func battleEndsOrders(sides ...orderSide) {
    if !BattleEndsOrders {
        return
    }
    for _, side := range sides {
        if side.player == nil || side.stack == nil {
            continue
        }
        keeper, ok := side.player.AIBehavior.(orderKeeper)
        if !ok {
            continue
        }
        keeper.EndOrders(side.stack.Units())
    }
}

type orderSide struct {
    player *playerlib.Player
    stack *playerlib.UnitStack
}
