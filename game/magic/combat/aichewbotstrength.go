package combat

// The strength of a unit as the original's AI counts it on the world map, for the overland AI of
// Chewbot (ai/chewbotworld.go). Facts from the ReMoM project's reconstruction (MoM/src/NEXTTURN.c:
// Effective_Unit_Strength, Effective_Unit_Type_Strength); the parts are the ones of the battle AI
// (aichewbot.go). The code is ours.

import (
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// what the strength of a unit is made of: a unit of a battle, a unit of the world map, or a kind of
// unit (the guardians of a lair)
type chewStrengthUnit interface {
    GetRangedAttackDamageType() units.Damage
    GetRangedAttackPower() int
    GetMeleeAttackPower() int
    GetDefense() int
    HasAbility(data.AbilityType) bool
    HasEnchantment(data.UnitEnchantment) bool
}

type ChewStrengthUnit = chewStrengthUnit

// Effective_Unit_Strength: the hits it has left weighed by its defense, its melee threat (with a
// thrown attack, breath or gaze), and twice the threat of a missile, boulder or magic attack.
// health is what the figures have left all together
func ChewbotUnitStrength(unit ChewStrengthUnit, figures int, health int) int {
    thrown := 0
    gaze := chewGazeNone
    if chewShortRange(unit) {
        thrown = unit.GetRangedAttackPower()
        gaze = chewGaze(unit)
    }

    strength := chewEffectiveHits(health, unit.GetDefense())
    switch unit.GetRangedAttackDamageType() {
        case units.DamageRangedMagical, units.DamageRangedPhysical, units.DamageRangedBoulder:
            if unit.GetRangedAttackPower() > 0 {
                strength += chewFlagThreat(chewThreat(unit.GetRangedAttackPower(), figures, 40), figures, chewRangedFlags(unit), false) * 2
            }
    }
    strength += chewEffectiveMelee(unit.GetMeleeAttackPower(), thrown, figures, chewMeleeFlags(unit), gaze)
    return strength
}
