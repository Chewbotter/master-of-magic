package combat

// The rules of tactical combat as the original game has them. Rules in words:
// docs/mod/combat-rules.md.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/Combat.c:
// Battle_Unit_Attack, Battle_Unit_Attack_Target, Battle_Unit_Process_Attack, Apply_Fear_Attack,
// Check_Wall_Of_Fire_Attack, Battle_Unit_Defense_Special, Combat_Roll_Damage_Dealt; COMBINIT.c:
// Battle_Unit_Special_Stats). The code is ours.

import (
    "math/rand/v2"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/lib/fraction"
)

// false: upstream's rules of tactical combat
var ClassicRules = true

// Battle_Unit_Attack_Target: a second fear roll is taken off the attacker's own striking figures
// (kept, the original's)
const quirkFearHitsOwnFigures = true

// Battle_Unit_Process_Attack: in melee the defender's to-block is taken off the attacker's to-hit
// and counts again in the block (kept, the original's)
const quirkToBlockDoubleCount = true

// Blur: the loop runs over the damage while it removes from it, so it takes about half at most;
// and an illusion-immune DEFENDER loses its side's Blur (kept, the original's)
const quirkBlurLoop = true

// a natural 10 always hits: no attack hits on fewer than 10 percent
const classicMinimumToHit = 10

// Battle_Unit_Attack: a melee attack costs half the full moves, rounded up in half moves; no
// floor, so a unit of 1 move can strike twice
func classicMeleeCost(unit *ArmyUnit) fraction.Fraction {
    halfMoves := int(unit.GetMovementSpeed().Multiply(fraction.FromInt(2)).ToFloat())
    return fraction.Make((halfMoves + 1) / 2, 2)
}

// the defender's to-block of Lucky, Prayer and High Prayer, in percent (Battle_Unit_Special_Stats)
func classicToBlock(unit *ArmyUnit) int {
    block := 0
    if unit.HasAbility(data.AbilityLucky) {
        block += 10
    }
    if unit.Model.IsEnchantmentActive(data.CombatEnchantmentHighPrayer, unit.Team) || unit.Model.IsEnchantmentActive(data.CombatEnchantmentPrayer, unit.Team) {
        block += 10
    }
    return block
}

// Blur: each point of damage of a figure goes one time in 10, before the defense roll
func classicBlur(attacker *ArmyUnit, defender *ArmyUnit, damage int) int {
    if !attacker.Model.IsEnchantmentActive(data.CombatEnchantmentBlur, defender.Team) {
        return damage
    }
    if quirkBlurLoop {
        if defender.HasAbility(data.AbilityIllusionsImmunity) {
            return damage
        }
        for i := 0; i < damage; i++ {
            if rand.N(10) == 9 {
                damage--
            }
        }
        return damage
    }
    if attacker.HasAbility(data.AbilityIllusionsImmunity) {
        return damage
    }
    out := damage
    for range damage {
        if rand.N(10) == 9 {
            out--
        }
    }
    return out
}

// the to-hit of an attack roll, at least 10 percent
func classicToHit(toHit int) int {
    return max(classicMinimumToHit, toHit)
}

// Check_Wall_Of_Fire_Attack: only the attacker burns, when it strikes from outside the wall at a
// unit inside it; fliers, teleporters and mergers never
func (model *CombatModel) classicWallOfFireBurns(attacker *ArmyUnit, defender *ArmyUnit) bool {
    if attacker.IsFlying() || attacker.CanTeleport() || attacker.HasAbility(data.AbilityMerging) {
        return false
    }
    return !model.InsideWallOfFire(attacker.X, attacker.Y) && model.InsideWallOfFire(defender.X, defender.Y)
}

// Battle_Unit_Attack: an adjacent enemy is shot only when the ranged attack is more than half the
// melee, else it is struck
func (model *CombatModel) shootsAt(attacker *ArmyUnit, defender *ArmyUnit) bool {
    if !model.withinArrowRange(attacker, defender) || !model.canRangeAttack(attacker, defender) {
        return false
    }
    if ClassicRules && model.withinMeleeRange(attacker, defender) && model.canMeleeAttack(attacker, defender, true) {
        return attacker.GetRangedAttackPower() > attacker.GetMeleeAttackPower() / 2
    }
    return true
}

// Do_Legal_Spell_Check: Dispel Evil is not checked for its target, any enemy unit (kept, the
// original's)
const quirkDispelEvilAnyTarget = true

// Combat_Cast_Dispel: the Runemaster's double reads the wrong player, so combat dispels are never
// doubled (kept, the original's)
const quirkNoRunemasterDispel = true

// Apply_Wrack: a unit with n figures failing takes n times n (kept, the original's); Wraith Form
// spares a unit (the original's)
const quirkWrackSquared = true

// a wizard who gains casting skill in battle (Life Drain)
type skillGainer interface {
    AddCastingSkillPower(int)
}

// damage of a spell: its strength rolled at 30 percent, all of it against a sleeping unit
func spellRoll(unit UnitDamage, strength int) int {
    if ClassicRules && unit.IsAsleep() {
        return strength
    }
    return ComputeRoll(strength, 30)
}

// Apply_Wrack: undead of death and units in Wraith Form are spared
func classicWrackSpares(unit *ArmyUnit) bool {
    if !ClassicRules {
        return false
    }
    return unit.HasAbility(data.AbilityDeathImmunity) || unit.HasEnchantment(data.UnitEnchantmentWraithForm)
}

// Combat_Casting_Cost_Multiplier: the larger of the two distances, east and west around the world
func classicFortressDistance(x1 int, y1 int, x2 int, y2 int, provider any) int {
    dx := x1 - x2
    if measure, ok := provider.(interface{ XDistance(int, int) int }); ok {
        dx = measure.XDistance(x1, x2)
    }
    dy := y1 - y2
    return max(max(dx, -dx), max(dy, -dy))
}
