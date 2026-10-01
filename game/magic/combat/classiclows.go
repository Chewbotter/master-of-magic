package combat

// The smaller rules of a battle as the original game has them (the LOWs of the rules survey).
// Rules in words: docs/mod/combat-rules.md, section "Smaller rules".
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/Combat.c: End_Of_Combat,
// Retreat_From_Combat, Battle_Unit_Process_Attack, Battle_Unit_Defense_Special,
// Battle_Unit_Commit_Damage, Check_Attack_Melee, Check_Attack_Ranged, Battle_Unit_Attack,
// Battle_Unit_Attack_Target, Begin_Combat_Turn, Apply_Mana_Leak, Summon_Demon; COMBINIT.c:
// Battle_Unit_Regular_Stats, Battle_Unit_Item_Stats; CMBMAGIC.c: Combat_Cast_Spell,
// Do_Legal_Spell_Check; Spells131.c: Apply_Holy_Word, Apply_Death_Spell, Apply_Flame_Strike,
// Combat_Cast_Disenchant, Cast_Raise_Dead; Spells133.c: Apply_Call_Lightning). The code is ours.

import (
    rand "github.com/kazzmir/master-of-magic/lib/chance"

    "github.com/kazzmir/master-of-magic/game/magic/artifact"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// Apply_Holy_Word, Apply_Death_Spell, Apply_Flame_Strike, Apply_Call_Lightning spare Wraith Form
// (the reconstruction marks it a mistake: it should not protect)
const quirkWraithFormSpares = true

// Combat_Cast_Spell: a spell stopped by Counter Magic or a node costs the whole skill but only the
// range's tenth in mana (the original's mistake)
const quirkCounteredCheap = true

// Combat_Cast_Spell: the human pays the full cost of a fixed cost spell in battle, the spellbook
// shows the reduced one (the caster test is above 20, the human is 20)
const quirkHumanNoCombatDiscount = true

// the units of a side a wizard controls at most when it summons or raises (Do_Legal_Spell_Check)
const classicCombatUnitsMost = 9

// the artifacts of a hero, for the to-hit of its items by the kind of attack
type artifactCarrier interface {
    GetArtifacts() []*artifact.Artifact
}

// Battle_Unit_Item_Stats: the to-hit of the items that do not count for this kind of attack
func itemToHitNotFor(unit *ArmyUnit, ranged units.Damage) int {
    carrier, ok := unit.Unit.(artifactCarrier)
    if !ok {
        return 0
    }
    out := 0
    for _, item := range carrier.GetArtifacts() {
        if item == nil {
            continue
        }
        switch item.Type {
            case artifact.ArtifactTypeSword, artifact.ArtifactTypeMace, artifact.ArtifactTypeAxe:
                // melee items; the axe's thrown use is never read (the original's)
                out += item.ToHitBonus()
            case artifact.ArtifactTypeBow:
                if ranged != units.DamageRangedPhysical {
                    out += item.ToHitBonus()
                }
            case artifact.ArtifactTypeStaff, artifact.ArtifactTypeWand:
                if ranged != units.DamageRangedMagical {
                    out += item.ToHitBonus()
                }
        }
    }
    return out
}

// Battle_Unit_Regular_Stats: the to-hit of a ranged shot: the base, items for shooting, the weapon
// and Holy Weapon only for missiles and rocks
func classicRangedToHit(unit *ArmyUnit, defender *ArmyUnit) int {
    toHit := unit.GetToHitMelee(defender)
    kind := unit.Unit.GetRangedAttackDamageType()
    toHit -= itemToHitNotFor(unit, kind)
    if kind == units.DamageRangedMagical {
        // the weapon the unit was built with: Metal Fires gives no to-hit
        if unit.Unit.GetWeaponBonus() != data.WeaponNone {
            toHit -= 10
        }
        if unit.HasEnchantment(data.UnitEnchantmentHolyWeapon) {
            toHit -= 10
        }
    }
    return toHit
}

// Battle_Unit_Process_Attack: thrown and breath strike with the base to-hit alone: no weapon, no
// Holy Weapon, no items
func classicThrownToHit(unit *ArmyUnit, defender *ArmyUnit) int {
    toHit := unit.GetToHitMelee(defender)
    if carrier, ok := unit.Unit.(artifactCarrier); ok {
        for _, item := range carrier.GetArtifacts() {
            if item != nil {
                toHit -= item.ToHitBonus()
            }
        }
    }
    if unit.Unit.GetWeaponBonus() != data.WeaponNone {
        toHit -= 10
    }
    if unit.HasEnchantment(data.UnitEnchantmentHolyWeapon) {
        toHit -= 10
    }
    return toHit
}

// Battle_Unit_Commit_Damage: irreversible damage wins ties, drain only when it is more than the
// irreversible and at least the normal
func (unit *ArmyUnit) classicDeathReason() DamageType {
    if unit.IrreversableDamage >= unit.NormalDamage && unit.IrreversableDamage >= unit.UndeadDamage && unit.IrreversableDamage > 0 {
        return DamageIrreversable
    }
    if unit.UndeadDamage > unit.IrreversableDamage && unit.UndeadDamage >= unit.NormalDamage {
        invalid := unit.Unit.IsHero() || unit.GetRealm() == data.DeathMagic || unit.Summoned || unit.HasAbility(data.AbilityMagicImmunity)
        if !invalid {
            return DamageUndead
        }
    }
    return DamageNormal
}

// Begin_Combat_Turn: a web is torn by the melee (missiles: the larger of melee and ranged; breath
// and gaze: both)
func (unit *ArmyUnit) classicTearWeb() {
    if unit.WebHealth <= 0 {
        return
    }
    damage := unit.GetMeleeAttackPower()
    switch {
        case unit.Unit.GetRangedAttackDamageType() == units.DamageRangedPhysical:
            damage = max(damage, unit.GetRangedAttackPower())
        case unit.HasAbility(data.AbilityFireBreath) || unit.HasAbility(data.AbilityLightningBreath):
            damage += int(unit.GetAbilityValue(data.AbilityFireBreath)) + int(unit.GetAbilityValue(data.AbilityLightningBreath))
        case unit.HasAbility(data.AbilityStoningGaze) || unit.HasAbility(data.AbilityDeathGaze) || unit.HasAbility(data.AbilityDoomGaze):
            damage += unit.GetRangedAttackPower()
    }
    unit.WebHealth -= damage
}

// Check_Attack_Melee: a unit that does not fly strikes a flier only with a ranged attack left; with
// thrown, breath or gaze it hurts it, with bows, rocks or magic it does not
func classicStrikesFlier(unit *ArmyUnit) bool {
    if unit.HasAbility(data.AbilityThrown) || unit.HasAbility(data.AbilityFireBreath) || unit.HasAbility(data.AbilityLightningBreath) ||
       unit.HasAbility(data.AbilityStoningGaze) || unit.HasAbility(data.AbilityDeathGaze) || unit.HasAbility(data.AbilityDoomGaze) {
        return true
    }
    return unit.CanRangeAttack()
}

func classicHurtsFlier(unit *ArmyUnit) bool {
    return unit.HasAbility(data.AbilityThrown) || unit.HasAbility(data.AbilityFireBreath) || unit.HasAbility(data.AbilityLightningBreath) ||
       unit.HasAbility(data.AbilityStoningGaze) || unit.HasAbility(data.AbilityDeathGaze) || unit.HasAbility(data.AbilityDoomGaze)
}

// the mana of a spell cast in battle: the cost times the range, or only the range's tenth when it
// was countered
func (model *CombatModel) castMana(army *Army, cost int) int {
    if ClassicRules && quirkCounteredCheap && model.LastCountered {
        return int(army.Range.ToFloat() * 10) / 10
    }
    return int(float64(cost) * army.Range.ToFloat())
}

// Do_Legal_Spell_Check: the human controls at most 9 units when it summons or raises
func (model *CombatModel) classicTooManyUnits(army *Army) bool {
    if army.IsAI() {
        return false
    }
    count := 0
    for _, unit := range army.units {
        if unit.GetHealth() > 0 {
            count += 1
        }
    }
    if count >= classicCombatUnitsMost {
        model.Events <- &CombatEventMessage{Message: "You may only control 9 units in combat at one time"}
        return true
    }
    return false
}

// Combat_Cast_Disenchant: the vortexes of the other side and the walls of fire and darkness of a city
// of the other side go too, each by a roll
func (model *CombatModel) classicDisenchantMore(caster ArmyPlayer, strength int, roll func(cost int, magic data.MagicType, owner ArmyPlayer) bool, costOf func(name string) int) {
    casterTeam := model.GetTeamForArmy(model.GetArmyForPlayer(caster))
    var keep []*MagicVortex
    for _, vortex := range model.MagicVortexes {
        if vortex.Team != casterTeam && roll(costOf("Magic Vortex"), data.ChaosMagic, model.GetArmyForTeam(vortex.Team).Player) {
            continue
        }
        keep = append(keep, vortex)
    }
    model.MagicVortexes = keep

    if model.Zone.City == nil || casterTeam == TeamDefender {
        return
    }
    defender := model.DefendingArmy.Player
    if model.Zone.City.HasEnchantment(data.CityEnchantmentWallOfFire) && roll(costOf("Wall of Fire"), data.ChaosMagic, defender) {
        model.removeWallOfFire()
    }
    if model.Zone.City.HasEnchantment(data.CityEnchantmentWallOfDarkness) && roll(costOf("Wall of Darkness"), data.DeathMagic, defender) {
        model.removeWallOfDarkness()
    }
}

// Apply_Call_Lightning: 3 to 5 bolts, each try a unit by chance that must be an enemy without
// Wraith Form and win a roll of 1 in 2; up to 30 failed tries
func (model *CombatModel) classicLightningTargets(enemies []*ArmyUnit) []*ArmyUnit {
    var out []*ArmyUnit
    if len(enemies) == 0 {
        return out
    }
    bolts := rand.N(3) + 3
    failed := 0
    for len(out) < bolts && failed < 30 {
        target := enemies[rand.N(len(enemies))]
        if target.GetHealth() <= 0 || (quirkWraithFormSpares && target.HasEnchantment(data.UnitEnchantmentWraithForm)) || rand.N(2) != 0 {
            failed += 1
            continue
        }
        out = append(out, target)
    }
    return out
}

// the wall of fire of the battle is gone (Combat_Cast_Disenchant), the city keeps its enchantment
func (model *CombatModel) removeWallOfFire() {
    for y := range model.Tiles {
        for x := range model.Tiles[y] {
            model.Tiles[y][x].Fire = nil
            model.Tiles[y][x].InsideFire = false
        }
    }
}

// the wall of darkness of the battle is gone
func (model *CombatModel) removeWallOfDarkness() {
    for y := range model.Tiles {
        for x := range model.Tiles[y] {
            model.Tiles[y][x].Darkness = nil
            model.Tiles[y][x].InsideDarkness = false
        }
    }
}
