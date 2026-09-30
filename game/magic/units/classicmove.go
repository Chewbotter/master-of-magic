package units

// The moves of a unit on the world map as the original game has them. Rules in words:
// docs/mod/movement.md.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/NEXTTURN.c: Unit_Moves2).
// The code is ours. Upstream: Endurance before Flight, Haste doubling the moves on the world map
// too, the Magic Spirit 2 moves.

import (
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/lib/fraction"
)

// false: upstream's movement on the world map (and the pathfinder's costs, game/movement.go)
var ClassicMovement = true

// Unit_Moves2: the unit's base, Flight at least 3, the wings of Chaos Channels at least 2,
// Endurance 1 more; Haste nothing
func (unit *OverworldUnit) classicOverlandSpeed() fraction.Fraction {
    return ClassicOverlandMoves(unit.Unit.MovementSpeed, unit.Enchantments)
}

// the moves of a base (the unit's, a hero's with its items) with these enchantments
func ClassicOverlandMoves(moves int, enchantments []data.UnitEnchantment) fraction.Fraction {
    for _, enchantment := range enchantments {
        if enchantment == data.UnitEnchantmentFlight {
            moves = max(moves, 3)
        }
    }
    for _, enchantment := range enchantments {
        if enchantment == data.UnitEnchantmentChaosChannelsDemonWings {
            moves = max(moves, 2)
        }
    }
    for _, enchantment := range enchantments {
        if enchantment == data.UnitEnchantmentEndurance {
            moves += 1
        }
    }
    return fraction.FromInt(moves)
}
