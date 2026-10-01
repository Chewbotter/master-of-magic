package combat

// The spells of Chewbot's combat AI (see aichewbot.go): which spell the wizard and the units of a
// computer player cast in a battle, and at what, as the original game decides it. Written from
// the facts of the ReMoM project's reconstruction; the code is ours. Facts from MoM/src/Combat.c
// (AI_Select_Combat_Spell, AI_Score_Combat_Spell, AI_Build_Castable_Combat_Spell_List,
// AI_Prepare_Combat_Realm_Threats, AI_Build_Unit_Realm_Threat_Percentages, AITP_Combat_Spell,
// AITP_Healing, AITP_WarpWood, AITP_WarpCreature, AITP_DispelMagic, AITP_DarknessLight,
// AITP_HolyWord, Auto_Cast_Spell_And_Do_Combat_Turn) and MoM/src/CMBMAGIC.c (Combat_Cast_Spell,
// Spell_Resistance_Modifier, AITP_EarthToMud, AITP_Disrupt, AITP_CracksCall, AITP_RecallHero).
// The rules in words: docs/mod/ai-combat.md.
//
// How it fits the fork: the fork's own code casts the spell (InvokeSpell). The target the original
// would pick is handed to it through chewPending, which DoAITargetUnitSpell and
// DoAITargetTileSpell take before their own choice by chance.

import (
    "image"
    rand "github.com/kazzmir/master-of-magic/lib/chance"
    "slices"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    "github.com/kazzmir/master-of-magic/lib/fraction"
)

// the original's spell types, spelldat's type field (Spell.SpellType)
const (
    spellTypeSummoning = 0
    spellTypeUnitEnchantment = 1
    spellTypeDirectDamageFixed = 4
    spellTypeSpecial = 5
    spellTypeBattlefield = 10
    spellTypeDestroyUnit = 12
    spellTypeResistable = 13
    spellTypeUnresistable = 14
    spellTypeNormalOnly = 15
    spellTypeMundaneCurse = 16
    // from here on a spell can take more mana than its cost
    spellTypeInfusable = 17
    spellTypeDispels = 18
    spellTypeDisenchants = 19
    spellTypeCounterMagic = 21
    spellTypeDirectDamageVariable = 22
    spellTypeBanish = 23
)

// the values of AI_Score_Combat_Spell by the player mode (0 badly outnumbered to 5 far stronger):
// 1 summons, 2 curses of the mind, 3 help for the own army, 4 protection, 5 damage
var chewSpellGroups = [6][6]int{
    {},
    {0, 10, 20, 25, 10, -10},
    {-900, 30, 35, 15, 0, 0},
    {-900, 10, 20, 20, 20, 10},
    {-900, 30, 20, 15, 10, 0},
    {500, 0, 10, 20, 30, 40},
}

// a spell only counts when its score is above this
const chewSpellLeast = 0

// the most own units a summoning spell is cast for
const chewSummonMost = 9

// what a spell of Chewbot's is cast at: taken by DoAITargetUnitSpell or DoAITargetTileSpell
type chewPending struct {
    Unit *ArmyUnit
    Tile image.Point
    HasTile bool
}

// the target the original picks for a spell, or nothing
type chewTarget struct {
    Unit *ArmyUnit
    Tile image.Point
    HasTile bool
    // the spell has a target, a unit, a place or the whole battle
    Found bool
}

// the unit target Chewbot has picked for the spell being cast, if the fork allows it
func (model *CombatModel) chewPendingUnit(canTarget func(*ArmyUnit) bool) (*ArmyUnit, bool) {
    if model.chewbot == nil || model.chewbot.pending == nil || model.chewbot.pending.Unit == nil {
        return nil, false
    }
    unit := model.chewbot.pending.Unit
    if unit.GetHealth() <= 0 || !canTarget(unit) {
        return nil, false
    }
    return unit, true
}

func (model *CombatModel) chewPendingTile(canTarget func(int, int) bool) (image.Point, bool) {
    if model.chewbot == nil || model.chewbot.pending == nil || !model.chewbot.pending.HasTile {
        return image.Point{}, false
    }
    tile := model.chewbot.pending.Tile
    if !canTarget(tile.X, tile.Y) {
        return image.Point{}, false
    }
    return tile, true
}

// casts a spell at what Chewbot picked. the fork decides the rest
func (model *CombatModel) chewInvoke(spellSystem SpellSystem, army *Army, caster *ArmyUnit, spell spellbook.Spell, target chewTarget, done func(bool)) {
    chew := model.chewbotState()
    chew.pending = &chewPending{Unit: target.Unit, Tile: target.Tile, HasTile: target.HasTile}
    defer func() {
        chew.pending = nil
    }()
    model.InvokeSpell(spellSystem, army, caster, spell, done)
}

// ------------------------------------------------------------------------------------------------
// the wizard

// Auto_Cast_Spell_And_Do_Combat_Turn: a computer wizard casts a spell at the start of its side's
// turn, the one of the highest score; the player's own wizard does not while its army is on auto,
// nor does the neutral player
func (model *CombatModel) doAiCastChewbot(spellSystem SpellSystem, army *Army) bool {
    if !army.autoCastsSpells() || army.Casted || army.Player.IsHuman() || model.chewNeutral(army) {
        return false
    }

    defendingCity := model.Zone.City != nil && model.GetTeamForArmy(army) == TeamDefender
    manaLimit := min(army.ManaPool, int(float64(army.Player.GetMana()) / army.Range.ToFloat()))

    var castable []spellbook.Spell
    for _, spell := range army.Player.GetKnownSpells().CombatSpells(defendingCity).Spells {
        if army.Player.ComputeEffectiveSpellCost(spell, false) <= manaLimit {
            castable = append(castable, spell)
        }
    }

    spell, target, ok := model.chewSelectSpell(army, nil, castable)
    if !ok {
        return false
    }

    // a spell that can take more mana is cast with as much again as it costs, as far as the mana goes
    base := spell.BaseCost(false)
    if spell.SpellType >= spellTypeInfusable && spell.IsVariableCost() {
        extra := min(base, max(0, manaLimit - army.Player.ComputeEffectiveSpellCost(spell, false)))
        switch spell.Name {
            case "Life Drain", "Banish", "Counter Magic":
                // some fives of it, by chance
                extra = rand.IntN(extra / 5 + 1) * 5
        }
        if extra > 0 {
            spell.OverrideCost = base + extra
            for spell.OverrideCost > base && army.Player.ComputeEffectiveSpellCost(spell, false) > manaLimit {
                spell.OverrideCost -= 1
            }
        }
    }

    cost := army.Player.ComputeEffectiveSpellCost(spell, false)
    casted := false
    chewLog("the wizard of %v casts %v (cost %v) at %v", model.GetTeamForArmy(army), spell.Name, cost, chewName(target.Unit))

    model.chewInvoke(spellSystem, army, nil, spell, target, func(success bool) {
        army.ManaPool -= cost
        army.Player.UseMana(model.castMana(army, cost))
        army.Casted = true
        casted = true
        if success {
            spellSystem.PlaySound(spell)
        }
    })

    return casted
}

// AI_Select_Combat_Spell: the spell of the highest score above 0, the first one of it in the
// order of the spells
func (model *CombatModel) chewSelectSpell(army *Army, caster *ArmyUnit, castable []spellbook.Spell) (spellbook.Spell, chewTarget, bool) {
    slices.SortStableFunc(castable, func(a spellbook.Spell, b spellbook.Spell) int {
        return a.Index - b.Index
    })

    mode := model.chewPlayerMode(army)
    threats := model.chewThreats(army)

    best := chewSpellLeast
    var chosen spellbook.Spell
    var chosenTarget chewTarget
    found := false

    for _, spell := range castable {
        score, target := model.chewScoreSpell(army, caster, spell, mode, threats)
        chewLog("  %v scores %v", spell.Name, score)
        if score > best {
            best = score
            chosen = spell
            chosenTarget = target
            found = true
        }
    }

    return chosen, chosenTarget, found
}

// what the enemy can do to the side: AI_Prepare_Combat_Realm_Threats. EnemyCastsAttacks is the
// original's "opp_spell_realms > 0"; the per realm bonus of those realms is always 0 in the
// original (its variables are never set), so only whether there are any counts
type chewThreatInfo struct {
    EnemyCastsAttacks bool
    // share of the enemy army's strength by realm, in percent
    Realms map[data.MagicType]int
}

func (model *CombatModel) chewThreats(army *Army) chewThreatInfo {
    other := model.GetOppositeArmyForTeam(model.GetTeamForArmy(army))
    info := chewThreatInfo{Realms: make(map[data.MagicType]int)}

    if other.Player != nil && !model.chewNeutral(other) {
        defendingCity := model.Zone.City != nil && other == model.DefendingArmy
        limit := min(other.ManaPool, int(float64(other.Player.GetMana()) / other.Range.ToFloat()))
        for _, spell := range other.Player.GetKnownSpells().CombatSpells(defendingCity).Spells {
            if other.Player.ComputeEffectiveSpellCost(spell, false) > limit {
                continue
            }
            switch spell.SpellType {
                case spellTypeDirectDamageFixed, spellTypeDirectDamageVariable, spellTypeDestroyUnit, spellTypeResistable, spellTypeMundaneCurse:
                    info.EnemyCastsAttacks = true
            }
            if spell.Name == "Creature Binding" {
                info.EnemyCastsAttacks = true
            }
        }
    }

    // AI_Build_Unit_Realm_Threat_Percentages
    total := 0
    realms := make(map[data.MagicType]int)
    for _, unit := range other.units {
        if unit.GetHealth() <= 0 {
            continue
        }
        figures := unit.Figures()
        total += (unit.GetMeleeAttackPower() + unit.GetRangedAttackPower()) * figures
        if unit.HasAbility(data.AbilityFireBreath) || unit.HasAbility(data.AbilityLightningBreath) {
            realms[data.ChaosMagic] += unit.GetRangedAttackPower() * figures
        }
        if unit.HasAbility(data.AbilityStoningGaze) {
            realms[data.NatureMagic] += unit.GetRangedAttackPower() * figures
        }
        if unit.HasAbility(data.AbilityFireballSpell) || unit.HasAbility(data.AbilityDoomBoltSpell) {
            total += 10
            realms[data.ChaosMagic] += 10
        }
        if unit.HasAbility(data.AbilityImmolation) || unit.HasEnchantment(data.UnitEnchantmentImmolation) {
            total += 20
            realms[data.ChaosMagic] += 20
        }
        if unit.HasAbility(data.AbilityLifeSteal) {
            amount := (int(unit.GetAbilityValue(data.AbilityLifeSteal)) + 3) * figures
            total += amount
            realms[data.DeathMagic] += amount
        }
        if unit.HasAbility(data.AbilityStoningTouch) {
            total += figures * 5
            realms[data.NatureMagic] += figures * 5
        }
        if unit.HasAbility(data.AbilityCauseFear) || unit.HasEnchantment(data.UnitEnchantmentCloakOfFear) {
            total += figures
            realms[data.DeathMagic] += figures
        }
        if unit.HasAbility(data.AbilityDeathTouch) {
            total += figures * 5
            realms[data.DeathMagic] += figures * 5
        }
        if unit.HasAbility(data.AbilityDispelEvil) {
            total += figures * 5
            realms[data.LifeMagic] += figures * 5
        }
    }
    if total > 0 {
        for realm, amount := range realms {
            info.Realms[realm] = amount * 100 / total
        }
    }

    return info
}

// ------------------------------------------------------------------------------------------------
// the units of both sides, as the spells see them

func (model *CombatModel) chewSideUnits(army *Army, own bool) []*ArmyUnit {
    side := army
    if !own {
        side = model.GetOppositeArmyForTeam(model.GetTeamForArmy(army))
    }
    var out []*ArmyUnit
    for _, unit := range side.units {
        if unit.GetHealth() > 0 {
            out = append(out, unit)
        }
    }
    return out
}

// a fantastic unit of a realm: the original's races from rt_Arcane on
func chewFantastic(unit *ArmyUnit) bool {
    return unit.GetRace() == data.RaceFantastic
}

func chewFantasticOf(unit *ArmyUnit, realm data.MagicType) bool {
    return chewFantastic(unit) && unit.GetRealm() == realm
}

func chewRighteous(unit *ArmyUnit) bool {
    return unit.HasEnchantment(data.UnitEnchantmentRighteousness)
}

// the spell's enchantment or curse is on the unit already
func chewHasSpell(unit *ArmyUnit, spell spellbook.Spell) bool {
    if enchantment := spell.GetUnitEnchantment(); enchantment != data.UnitEnchantmentNone && unit.HasEnchantment(enchantment) {
        return true
    }
    if curse := spell.GetUnitCurse(); curse != data.UnitEnchantmentNone && unit.HasCurse(curse) {
        return true
    }
    return false
}

// the side sees the unit: invisible units only for a side with a unit immune to illusions
func (model *CombatModel) chewVisibleTo(army *Army, unit *ArmyUnit) bool {
    if !unit.IsInvisible() {
        return true
    }
    for _, own := range army.units {
        if own.GetHealth() > 0 && own.HasAbility(data.AbilityIllusionsImmunity) {
            return true
        }
    }
    return false
}

// Spell_Resistance_Modifier
var chewResistModifier = map[string]int{
    "Warp Creature": -1, "Weakness": -2, "Black Sleep": -2, "Possession": -1, "Black Wind": -1,
    "Terror": 1, "Wrack": 1, "Word of Death": -5, "Death Spell": -2, "Dispel Evil": -4,
    "Holy Word": -2, "Banish": -3, "Confusion": -4, "Stasis": -5, "Creature Binding": -2,
    "Great Unsummoning": -3,
}

// a value scaled by the chance the unit fails its resistance
func chewByResistance(strength int, resistance int) int {
    return (strength * (10 - resistance) + 9) / 10
}

// what a spell of damage takes from a unit, roughly: the original computes the damage the spell
// would do (Compute_Battle_Unit_Damage_From_Spell); here its strength against the defense, at the
// original's 30 in 100 to hit and to block
func (model *CombatModel) chewSpellDamage(spell spellbook.Spell, target *ArmyUnit) int {
    if target.HasAbility(data.AbilityMagicImmunity) {
        return 0
    }
    switch spell.Name {
        case "Fire Bolt", "Fireball", "Fire Storm", "Flame Strike":
            if target.HasAbility(data.AbilityFireImmunity) {
                return 0
            }
        case "Ice Bolt", "Ice Storm":
            if target.HasAbility(data.AbilityColdImmunity) {
                return 0
            }
        case "Doom Bolt":
            // doom damage passes every defense
            return 10
    }

    strength := spell.Cost(false)
    defense := GetDefenseFor(target, spell.Magic)
    if target.HasAbility(data.AbilityLargeShield) {
        defense += 2
    }
    damage := max(0, (strength - defense) * 3 / 10)
    if spell.Name == "Fireball" {
        damage *= target.Figures()
    }
    return damage
}

// ------------------------------------------------------------------------------------------------
// targets: AITP_Combat_Spell

func (model *CombatModel) chewSpellTarget(army *Army, spell spellbook.Spell) chewTarget {
    best := -1
    var target chewTarget
    consider := func(unit *ArmyUnit, value int) {
        if value > best {
            best = value
            target = chewTarget{Unit: unit, Found: true}
        }
    }
    whole := chewTarget{Found: true}

    enemies := model.chewSideUnits(army, false)
    own := model.chewSideUnits(army, true)
    team := model.GetTeamForArmy(army)

    chaosOrDeath := spell.Magic == data.ChaosMagic || spell.Magic == data.DeathMagic

    switch spell.SpellType {
        case spellTypeDirectDamageFixed, spellTypeDirectDamageVariable:
            for _, unit := range enemies {
                if unit.HasAbility(data.AbilityMagicImmunity) || !model.chewVisibleTo(army, unit) {
                    continue
                }
                if spell.Name == "Star Fires" && !chewFantasticOf(unit, data.DeathMagic) && !chewFantasticOf(unit, data.ChaosMagic) {
                    continue
                }
                if spell.Name == "Psionic Blast" && unit.HasAbility(data.AbilityIllusionsImmunity) {
                    continue
                }
                if spell.Name == "Life Drain" && (chewFantasticOf(unit, data.DeathMagic) || unit.HasAbility(data.AbilityDeathImmunity)) {
                    continue
                }
                if chewRighteous(unit) && chaosOrDeath {
                    continue
                }
                value := 0
                if spell.Name == "Life Drain" {
                    value = max(0, 13 - GetResistanceFor(unit, data.DeathMagic))
                } else {
                    value = model.chewSpellDamage(spell, unit)
                }
                if value > 0 {
                    // the weaker the unit, the better
                    value += 100 - unit.Figures() * unit.GetHitPoints() + (unit.GetHitPoints() - unit.GetLeadUnitHealth())
                }
                consider(unit, value)
            }

        case spellTypeDestroyUnit, spellTypeBanish:
            for _, unit := range enemies {
                if unit.HasAbility(data.AbilityMagicImmunity) || !model.chewVisibleTo(army, unit) {
                    continue
                }
                if spell.Name == "Petrify" && unit.HasAbility(data.AbilityStoningImmunity) {
                    continue
                }
                if spell.Name == "Dispel Evil" && !chewFantasticOf(unit, data.DeathMagic) && !chewFantasticOf(unit, data.ChaosMagic) {
                    continue
                }
                if spell.Name == "Banish" && !chewFantastic(unit) {
                    continue
                }
                if chewRighteous(unit) && chaosOrDeath {
                    continue
                }
                resistance := GetResistanceFor(unit, spell.Magic) + chewResistModifier[spell.Name]
                if resistance >= 10 {
                    continue
                }
                if spell.Name == "Disintegrate" {
                    consider(unit, chewEffectiveStrength(unit))
                } else {
                    consider(unit, chewByResistance(chewEffectiveStrength(unit), resistance))
                }
            }

        case spellTypeUnitEnchantment, spellTypeNormalOnly:
            chew := model.chewbotState()
            for _, unit := range own {
                if chewHasSpell(unit, spell) {
                    continue
                }
                if spell.SpellType == spellTypeNormalOnly && chewFantastic(unit) {
                    continue
                }
                switch spell.Name {
                    case "Stone Skin":
                        if unit.HasEnchantment(data.UnitEnchantmentIronSkin) {
                            continue
                        }
                    case "Resist Elements":
                        if unit.HasEnchantment(data.UnitEnchantmentElementalArmor) {
                            continue
                        }
                    case "Resist Magic":
                        if unit.HasEnchantment(data.UnitEnchantmentWraithForm) {
                            continue
                        }
                    case "Flight":
                        if unit.IsFlying() {
                            continue
                        }
                    case "Heroism":
                        if unit.GetExperienceLevel() > units.ExperienceRegular {
                            continue
                        }
                    case "Berserk":
                        goal := chew.Targets[unit]
                        if goal == nil || goal == unit || fraction.FromInt(chewRange(unit, goal)).GreaterThan(unit.MovesLeft) || unit.Unit.IsHero() {
                            continue
                        }
                }
                consider(unit, unit.Figures() * 10 + unit.GetMeleeAttackPower())
            }

        case spellTypeResistable, spellTypeMundaneCurse:
            for _, unit := range enemies {
                if chewHasSpell(unit, spell) || unit.HasAbility(data.AbilityMagicImmunity) || !model.chewVisibleTo(army, unit) {
                    continue
                }
                if spell.Magic == data.SorceryMagic && unit.HasAbility(data.AbilityIllusionsImmunity) {
                    continue
                }
                if spell.SpellType == spellTypeResistable && spell.Magic == data.DeathMagic && unit.HasAbility(data.AbilityDeathImmunity) {
                    continue
                }
                if chewRighteous(unit) && chaosOrDeath {
                    continue
                }
                if spell.Name == "Creature Binding" && !chewFantastic(unit) {
                    continue
                }
                if spell.Name == "Shatter" && unit.GetMeleeAttackPower() <= 1 && unit.GetRangedAttackPower() <= 1 {
                    continue
                }
                if spell.SpellType == spellTypeMundaneCurse && chewFantastic(unit) {
                    continue
                }
                resistance := GetResistanceFor(unit, spell.Magic)
                if spell.SpellType == spellTypeResistable {
                    resistance += chewResistModifier[spell.Name]
                }
                if resistance >= 10 {
                    continue
                }
                consider(unit, chewByResistance(chewEffectiveStrength(unit), resistance))
            }

        case spellTypeUnresistable:
            for _, unit := range enemies {
                if spell.Name == "Web" {
                    if unit.HasAbility(data.AbilityNonCorporeal) || unit.IsWebbed() {
                        continue
                    }
                } else {
                    if chewHasSpell(unit, spell) || unit.HasAbility(data.AbilityMagicImmunity) {
                        continue
                    }
                    if spell.Magic == data.SorceryMagic && unit.HasAbility(data.AbilityIllusionsImmunity) {
                        continue
                    }
                }
                if chewRighteous(unit) && chaosOrDeath || !model.chewVisibleTo(army, unit) {
                    continue
                }
                value := chewEffectiveStrength(unit)
                if spell.Name == "Web" && unit.IsFlying() {
                    value += 2000
                }
                consider(unit, value)
            }

        case spellTypeBattlefield, spellTypeCounterMagic:
            if enchantment, ok := chewCombatEnchantment(spell); ok && model.IsEnchantmentActive(enchantment, team) {
                return chewTarget{}
            }
            switch spell.Name {
                case "True Light", "Darkness":
                    // any unit of life or death, of either side
                    for _, unit := range append(slices.Clone(own), enemies...) {
                        if chewFantasticOf(unit, data.LifeMagic) || chewFantasticOf(unit, data.DeathMagic) {
                            return whole
                        }
                    }
                    return chewTarget{}
            }
            return whole

        case spellTypeSpecial:
            switch spell.Name {
                case "Healing":
                    return model.chewHealingTarget(own)
                case "Mass Healing":
                    if model.chewHealingTarget(own).Found {
                        return whole
                    }
                    return chewTarget{}
                case "Warp Wood":
                    return model.chewWarpWoodTarget(army, enemies)
                case "Warp Creature":
                    return model.chewWarpCreatureTarget(army, enemies)
                case "Earth to Mud":
                    return model.chewEarthToMudTarget(own, enemies)
                case "Disrupt":
                    return model.chewWallTarget(army)
                case "Recall Hero":
                    return model.chewRecallHeroTarget(army, own)
                case "Cracks Call":
                    return model.chewCracksCallTarget(army, enemies)
                case "Raise Dead":
                    return whole
            }
            // any other special spell has no target the AI would pick
            return chewTarget{}

        case spellTypeDispels:
            return model.chewDispelTarget(army, own, enemies)

        case spellTypeSummoning:
            if len(own) >= chewSummonMost {
                return chewTarget{}
            }
            return whole

        case spellTypeDisenchants:
            return whole
    }

    return target
}

// the combat enchantment a battlefield spell puts on the battle
func chewCombatEnchantment(spell spellbook.Spell) (data.CombatEnchantment, bool) {
    for enchantment := data.CombatEnchantmentHighPrayer; enchantment < data.CombatEnchantmentHighPrayer + 40; enchantment++ {
        if enchantment.Name() == spell.Name {
            return enchantment, true
        }
    }
    return data.CombatEnchantmentNone, false
}

// AITP_Healing: the own unit that misses the most hits, times its attack
func (model *CombatModel) chewHealingTarget(own []*ArmyUnit) chewTarget {
    best := 0
    var target chewTarget
    for _, unit := range own {
        if chewFantasticOf(unit, data.DeathMagic) {
            continue
        }
        missing := unit.GetMaxHealth() - unit.GetHealth()
        value := 0
        if missing > 2 {
            value = missing * (unit.GetMeleeAttackPower() + unit.GetRangedAttackPower())
        }
        if value > best {
            best = value
            target = chewTarget{Unit: unit, Found: true}
        }
    }
    return target
}

// AITP_WarpWood: the enemy with the most arrows
func (model *CombatModel) chewWarpWoodTarget(army *Army, enemies []*ArmyUnit) chewTarget {
    best := -1
    var target chewTarget
    for _, unit := range enemies {
        if unit.HasAbility(data.AbilityMagicImmunity) || chewRighteous(unit) || !model.chewVisibleTo(army, unit) {
            continue
        }
        value := -10
        if chewMissile(unit) {
            value = unit.GetRangedAttackPower() * unit.RangedAttacks * unit.Figures()
        }
        if value > best {
            best = value
            target = chewTarget{Unit: unit, Found: true}
        }
    }
    return target
}

// AITP_WarpCreature: the normal enemy of the most attack, by its chance to resist
func (model *CombatModel) chewWarpCreatureTarget(army *Army, enemies []*ArmyUnit) chewTarget {
    best := -1
    var target chewTarget
    for _, unit := range enemies {
        if unit.HasAbility(data.AbilityMagicImmunity) || chewRighteous(unit) || chewFantastic(unit) || !model.chewVisibleTo(army, unit) {
            continue
        }
        if unit.HasCurse(data.UnitCurseWarpCreatureMelee) || unit.HasCurse(data.UnitCurseWarpCreatureDefense) || unit.HasCurse(data.UnitCurseWarpCreatureResistance) {
            continue
        }
        resistance := GetResistanceFor(unit, data.ChaosMagic)
        if resistance >= 10 {
            continue
        }
        value := chewByResistance((unit.GetMeleeAttackPower() + unit.GetRangedAttackPower()) * unit.Figures(), resistance)
        if value > best {
            best = value
            target = chewTarget{Unit: unit, Found: true}
        }
    }
    return target
}

// AITP_EarthToMud: the cell with the most enemies within 2 and the fewest own, those that walk
func (model *CombatModel) chewEarthToMudTarget(own []*ArmyUnit, enemies []*ArmyUnit) chewTarget {
    if model.Ground == nil {
        return chewTarget{}
    }
    walks := func(unit *ArmyUnit) bool {
        return !unit.IsFlying() && !unit.CanTeleport() && !unit.HasAbility(data.AbilityMerging) && !unit.HasAbility(data.AbilityNonCorporeal)
    }
    near := func(cell image.Point, unit *ArmyUnit) int {
        distance := max(abs(cell.X - unit.X), abs(cell.Y - unit.Y))
        if distance <= 2 {
            return 3 - distance
        }
        return 0
    }

    best := 0
    var target chewTarget
    for x := 0; x < model.MaxWidth(); x++ {
        for y := 0; y < model.MaxHeight(); y++ {
            cell := image.Pt(x, y)
            value := 0
            for _, unit := range enemies {
                if walks(unit) {
                    value += near(cell, unit)
                }
            }
            for _, unit := range own {
                if walks(unit) {
                    value -= near(cell, unit)
                }
            }
            if value > best {
                best = value
                target = chewTarget{Tile: cell, HasTile: true, Found: true}
            }
        }
    }
    return target
}

// AITP_Disrupt: an attacker's spell at a piece of the city wall that still stands
func (model *CombatModel) chewWallTarget(army *Army) chewTarget {
    if model.GetTeamForArmy(army) == TeamDefender || model.Zone.City == nil {
        return chewTarget{}
    }
    for _, cell := range []image.Point{{8, 13}, {8, 11}, {8, 10}, {7, 13}, {7, 10}} {
        tile := fromCell(cell)
        if model.ContainsWall(tile.X, tile.Y) {
            return chewTarget{Tile: tile, HasTile: true, Found: true}
        }
    }
    return chewTarget{}
}

// AITP_CracksCall: the strongest enemy that walks, more so on the wall of the city; else a wall
func (model *CombatModel) chewCracksCallTarget(army *Army, enemies []*ArmyUnit) chewTarget {
    best := 0
    var target chewTarget
    attacker := model.GetTeamForArmy(army) == TeamAttacker
    for _, unit := range enemies {
        if unit.IsFlying() || unit.HasAbility(data.AbilityMerging) || unit.HasAbility(data.AbilityNonCorporeal) || !model.chewVisibleTo(army, unit) {
            continue
        }
        value := chewEffectiveStrength(unit)
        if attacker && model.ContainsWall(unit.X, unit.Y) {
            // the wall falls under it
            value += 30
        }
        if value > best {
            best = value
            target = chewTarget{Unit: unit, Tile: chewPoint(unit), HasTile: true, Found: true}
        }
    }
    if !target.Found && attacker {
        return model.chewWallTarget(army)
    }
    return target
}

// AITP_RecallHero: the hero in the most danger, never from the battle at the own fortress
func (model *CombatModel) chewRecallHeroTarget(army *Army, own []*ArmyUnit) chewTarget {
    if model.Zone.City != nil && model.Zone.City == army.Player.FindFortressCity() {
        return chewTarget{}
    }
    best := 0
    var target chewTarget
    for _, unit := range own {
        if !unit.Unit.IsHero() {
            continue
        }
        danger := chewEffectiveStrength(unit)
        if unit.GetHealth() > 0 {
            danger = danger * unit.GetMaxHealth() / unit.GetHealth()
        }
        if danger > best {
            best = danger
            target = chewTarget{Unit: unit, Found: true}
        }
    }
    return target
}

// AITP_DispelMagic: an own unit under curses, or an enemy under enchantments, the most valuable
func (model *CombatModel) chewDispelTarget(army *Army, own []*ArmyUnit, enemies []*ArmyUnit) chewTarget {
    best := 0
    var target chewTarget

    curseValue := map[data.UnitEnchantment]int{
        data.UnitCurseVertigo: 20, data.UnitCurseConfusion: 30, data.UnitCurseMindStorm: 40,
        data.UnitCurseShatter: 10, data.UnitCurseWeakness: 10, data.UnitCurseBlackSleep: 40,
        data.UnitCurseWarpCreatureMelee: 15, data.UnitCurseWarpCreatureDefense: 10, data.UnitCurseWarpCreatureResistance: 10,
    }
    enchantmentValue := map[data.UnitEnchantment]int{
        data.UnitEnchantmentImmolation: 20, data.UnitEnchantmentGuardianWind: 10, data.UnitEnchantmentCloakOfFear: 10,
        data.UnitEnchantmentWraithForm: 10, data.UnitEnchantmentStoneSkin: 10, data.UnitEnchantmentIronSkin: 20,
        data.UnitEnchantmentSpellLock: -10, data.UnitEnchantmentInvisibility: 10, data.UnitEnchantmentMagicImmunity: 25,
        data.UnitEnchantmentLionHeart: 30, data.UnitEnchantmentInvulnerability: 40,
    }

    for _, unit := range own {
        curses := unit.GetCurses()
        if len(curses) == 0 {
            continue
        }
        value := chewEffectiveStrength(unit)
        for _, curse := range curses {
            value += curseValue[curse]
        }
        if value > best {
            best = value
            target = chewTarget{Unit: unit, Found: true}
        }
    }
    for _, unit := range enemies {
        enchantments := unit.GetEnchantments()
        if len(enchantments) == 0 || !model.chewVisibleTo(army, unit) {
            continue
        }
        value := chewEffectiveStrength(unit)
        for _, enchantment := range enchantments {
            value += enchantmentValue[enchantment]
        }
        if value > best {
            best = value
            target = chewTarget{Unit: unit, Found: true}
        }
    }
    return target
}

// ------------------------------------------------------------------------------------------------
// the score of a spell: AI_Score_Combat_Spell

func (model *CombatModel) chewScoreSpell(army *Army, caster *ArmyUnit, spell spellbook.Spell, mode int, threats chewThreatInfo) (int, chewTarget) {
    enemies := model.chewSideUnits(army, false)
    own := model.chewSideUnits(army, true)
    if len(enemies) == 0 {
        return -100, chewTarget{}
    }

    target := model.chewSpellTarget(army, spell)
    if !target.Found {
        return -1, target
    }

    group := func(which int) int {
        return chewSpellGroups[which][mode]
    }
    count := func(units []*ArmyUnit, what func(*ArmyUnit) bool) int {
        total := 0
        for _, unit := range units {
            if what(unit) {
                total += 1
            }
        }
        return total
    }
    team := model.GetTeamForArmy(army)
    attacker := team == TeamAttacker
    walled := model.Zone.City != nil && model.GetCityGateCoordinatesKnown()
    ocean := model.Ground == nil
    walks := func(unit *ArmyUnit) bool {
        return !unit.IsFlying() && !unit.CanTeleport() && !unit.HasAbility(data.AbilityMerging)
    }
    // a value, or -100 when there is nothing for the spell
    valueOr := func(value int) int {
        if value > 0 {
            return value
        }
        return value - 100
    }

    value := 0

    switch spell.Name {
        case "Earth to Mud":
            ownRanged, enemyRanged := 0, 0
            for _, unit := range own {
                if chewHasRanged(unit) {
                    ownRanged += unit.GetRangedAttackPower() * unit.Figures()
                }
            }
            for _, unit := range enemies {
                if chewHasRanged(unit) {
                    enemyRanged += unit.GetRangedAttackPower() * unit.Figures()
                }
            }
            if ownRanged * 2 > enemyRanged * 3 {
                value += 15
            } else {
                value -= 100
            }
        case "Resist Elements":
            value = (threats.Realms[data.NatureMagic] + threats.Realms[data.ChaosMagic]) / 4
            if value != 0 {
                value += group(4)
            } else {
                value -= 100
            }
        case "Wall of Stone":
            if team == TeamDefender {
                value += group(4) * count(enemies, walks)
            } else {
                value = -100
            }
        case "Web":
            value += group(2)
            if count(enemies, func(unit *ArmyUnit) bool { return unit.IsFlying() }) > 0 {
                value += 10
            }
        case "Stone Skin":
            value += group(4) + 15
        case "Cracks Call":
            if !ocean {
                value += group(5)
                if attacker && walled {
                    value += 20
                }
            } else {
                value = -100
            }
        case "Ice Bolt", "Petrify", "Psionic Blast", "Banish", "Fire Bolt", "Lightning Bolt", "Warp Lightning",
             "Doom Bolt", "Magic Vortex", "Star Fires", "Dispel Evil", "Word of Death":
            value += group(5)
        case "Elemental Armor":
            value = valueOr((threats.Realms[data.NatureMagic] + threats.Realms[data.ChaosMagic]) / 2)
            if value > 0 {
                value += group(4)
            }
        case "Iron Skin":
            value += group(4) + 35
        case "Earth Elemental", "Fire Elemental":
            if ocean {
                value -= 100
            } else {
                value += group(1)
            }
        case "Regeneration", "Shatter", "Holy Armor", "Cloak of Fear", "Wraith Form":
            value += group(4)
        case "Call Lightning":
            if mode > 1 {
                value += len(enemies) * 4
            }
            value += group(5)
        case "Resist Magic", "Magic Immunity":
            total := 0
            for _, share := range threats.Realms {
                total += share
            }
            if spell.Name == "Resist Magic" {
                value = total / 4
            } else {
                value = total / 2
            }
            if value > 0 {
                value += group(4)
            } else {
                value -= 100
            }
        case "Dispel Magic", "Dispel Magic True":
            cursed := count(own, func(unit *ArmyUnit) bool { return len(unit.GetCurses()) > 0 })
            enchanted := count(enemies, func(unit *ArmyUnit) bool { return len(unit.GetEnchantments()) > 0 })
            if cursed + enchanted > 0 {
                if spell.Name == "Dispel Magic" {
                    value += 25
                } else {
                    value += 35
                }
            } else {
                value -= 100
            }
        case "Guardian Wind":
            arrows := 0
            for _, unit := range enemies {
                if chewMissile(unit) {
                    arrows += unit.GetRangedAttackPower()
                }
            }
            if arrows > 0 {
                value += group(4) + arrows
            }
        case "Phantom Warriors":
            if ocean {
                value -= 100
            } else {
                value += group(1)
            }
            if attacker && model.Zone.City != nil && model.Zone.City.HasWallOfFire() {
                value = -100
            }
        case "Confusion", "Vertigo", "Mind Storm", "Warp Wood", "Warp Creature", "Weakness":
            value += group(2)
        case "Counter Magic":
            if threats.EnemyCastsAttacks {
                value += 30 + group(4)
            } else {
                value = -100
            }
        case "Disenchant True", "Disenchant Area":
            cursed := count(own, func(unit *ArmyUnit) bool { return len(unit.GetCurses()) > 0 })
            enchanted := count(enemies, func(unit *ArmyUnit) bool { return len(unit.GetEnchantments()) > 0 })
            value = (cursed + enchanted) * 6
            other := model.GetOppositeArmyForTeam(team)
            active := 0
            for enchantment := data.CombatEnchantmentHighPrayer; enchantment < data.CombatEnchantmentHighPrayer + 40; enchantment++ {
                if enchantment.Name() != "" && model.IsEnchantmentActive(enchantment, model.GetTeamForArmy(other)) {
                    active += 1
                }
            }
            value += active * 30
            if value == 0 {
                value = -100
            }
        case "Flight":
            flyers := count(enemies, func(unit *ArmyUnit) bool { return unit.IsFlying() })
            value += group(4) + flyers * 2
            if flyers == 0 {
                value += 20
            }
        case "Phantom Beast", "Air Elemental":
            value += group(1)
        // the game's data spells it "Invisiblity"
        case "Invisibility", "Invisiblity", "Mass Invisibility":
            seen := 0
            for _, unit := range enemies {
                if unit.HasAbility(data.AbilityIllusionsImmunity) {
                    continue
                }
                seen += 1
                if chewHasRanged(unit) {
                    seen += unit.GetRangedAttackPower()
                }
            }
            if seen > 0 {
                value += group(4) + seen
                if spell.Name != "Mass Invisibility" {
                    value += 5
                }
            } else {
                value = -100
            }
        case "Haste", "Heroism":
            value += group(3)
        case "Creature Binding":
            if count(enemies, chewFantastic) == 1 {
                value += group(5)
            } else {
                value += group(1) + group(5) + 20
            }
        case "Disrupt":
            if attacker && walled {
                value += 20
            } else {
                value -= 100
            }
        case "Eldritch Weapon", "Flame Blade", "Holy Weapon":
            if count(enemies, func(unit *ArmyUnit) bool { return unit.HasEnchantment(data.UnitEnchantmentInvulnerability) }) > 0 {
                value += 25
            }
            value += group(3)
        case "Wall of Fire":
            if team == TeamDefender {
                value += group(3) * (count(enemies, walks) + 1) / 2
            } else {
                value = -100
            }
        case "Fireball":
            most := 0
            for _, unit := range enemies {
                most = max(most, unit.Figures())
            }
            value += group(5)
            if most > 3 {
                value += (most - 3) * 5
            }
        case "Immolation", "Berserk":
            value += group(3) + 10
        case "Metal Fires":
            normal := count(own, func(unit *ArmyUnit) bool { return !chewFantastic(unit) })
            if normal > 0 {
                value += group(3) + normal * 5
            } else {
                value -= 100
            }
        case "Warp Reality":
            balance := count(own, func(unit *ArmyUnit) bool { return chewFantasticOf(unit, data.ChaosMagic) }) - count(enemies, func(unit *ArmyUnit) bool { return chewFantasticOf(unit, data.ChaosMagic) })
            if balance > 0 {
                value += group(5) + balance * 5
            } else {
                value -= 100
            }
        case "Flame Strike":
            value += group(5) + len(enemies) * 2
        case "Disintegrate":
            value += group(5) + 25
        case "Bless":
            value = (threats.Realms[data.DeathMagic] + threats.Realms[data.ChaosMagic]) / 4
            if value != 0 {
                value += group(4)
            } else {
                value -= 100
            }
        case "Healing":
            if target.Unit != nil {
                value += chewEffectiveStrength(target.Unit) / 4
            }
        case "True Light", "Darkness":
            good, bad := data.LifeMagic, data.DeathMagic
            if spell.Name == "Darkness" {
                good, bad = data.DeathMagic, data.LifeMagic
            }
            helped := count(own, func(unit *ArmyUnit) bool { return chewFantasticOf(unit, good) }) + count(enemies, func(unit *ArmyUnit) bool { return chewFantasticOf(unit, bad) })
            if helped == 0 {
                value = -100
            } else {
                value += helped * 3
            }
        case "True Sight":
            value = threats.Realms[data.SorceryMagic] / 2
            if value != 0 {
                value += group(4)
            } else {
                value -= 100
            }
        case "Raise Dead":
            // the original counts its own normal units that are gone; the fork keeps no dead units in a battle
            value -= 100
        case "Prayer":
            value += group(3) + len(own) * 2
        case "High Prayer":
            value += group(3) + len(own) * 4
        case "Invulnerability":
            value += group(4) + 30
        case "Righteousness":
            value = threats.Realms[data.DeathMagic] + threats.Realms[data.ChaosMagic]
            if value > 0 {
                value += group(4)
            }
        case "Mass Healing":
            hurt := count(own, func(unit *ArmyUnit) bool { return unit.GetHealth() < unit.GetMaxHealth() })
            if hurt > 0 {
                value += (hurt - 1) * 10 + 5
            } else {
                value = -1000
            }
        case "Holy Word":
            figures := 0
            for _, unit := range enemies {
                if chewFantastic(unit) && GetResistanceFor(unit, data.LifeMagic) - 2 < 10 {
                    figures += unit.Figures()
                }
            }
            if figures > 0 {
                value += group(5) + figures * 3
            } else {
                value -= 100
            }
        case "Black Sleep":
            value += group(2) + group(5)
        case "Life Drain":
            value += group(5) + 10
        case "Terror":
            afraid := count(enemies, func(unit *ArmyUnit) bool {
                return GetResistanceFor(unit, data.DeathMagic) + 1 < 10 && !unit.HasAbility(data.AbilityDeathImmunity)
            })
            if afraid > 0 {
                value += group(2) + afraid * 3
            } else {
                value -= 100
            }
        case "Mana Leak":
            casters := 0
            for _, unit := range enemies {
                if unit.CastingSkill > 0 || chewMagicRanged(unit) {
                    casters += unit.Figures()
                }
            }
            if casters > 0 {
                value += 10 - model.CurrentTurn * 2
            } else {
                value = -100
            }
        case "Possession":
            value += group(5) + group(1)
        case "Black Prayer":
            value += group(4) + len(enemies) * 4
        case "Wall of Darkness":
            if team == TeamDefender {
                shots := 0
                for _, unit := range enemies {
                    if chewHasRanged(unit) {
                        shots += unit.GetRangedAttackPower()
                    }
                }
                value += group(4) + shots * 2
            } else {
                value = -100
            }
        case "Wrack", "Death Spell":
            figures := 0
            for _, unit := range enemies {
                resistance := GetResistanceFor(unit, data.DeathMagic)
                if spell.Name == "Wrack" {
                    resistance += 1
                } else {
                    resistance -= 2
                }
                if resistance < 10 && !unit.HasAbility(data.AbilityDeathImmunity) {
                    figures += unit.Figures()
                }
            }
            if figures > 0 {
                value += group(5)
                if spell.Name == "Wrack" {
                    value += figures * 5
                } else {
                    value += figures * 3
                }
            } else {
                value -= 100
            }
        case "Recall Hero":
            if army.Player.IsHuman() {
                value = -100
            } else {
                endangered := count(own, func(unit *ArmyUnit) bool {
                    return unit.Unit.IsHero() && (mode < 2 || unit.GetHitPoints() - unit.GetLeadUnitHealth() + 3 >= unit.GetHitPoints())
                })
                if endangered > 0 {
                    value += 100
                } else {
                    value = -100
                }
            }
    }

    // spells that cost the wizard less are worth more, and a little chance
    if caster == nil && spell.BaseCost(false) > 0 {
        reduction := (spell.BaseCost(false) - army.Player.ComputeEffectiveSpellCost(spell, false)) * 100 / spell.BaseCost(false)
        value += reduction / 2
    }
    value += 1 + rand.IntN(20)

    return value, target
}

// the battle is at a city with a wall of stone
func (model *CombatModel) GetCityGateCoordinatesKnown() bool {
    _, y := model.GetCityGateCoordinates()
    return y >= 0
}

// ------------------------------------------------------------------------------------------------
// the spells of units: Choose_Target_And_Action and the spell actions of AI_Execute_Unit_Action

// a spell the unit carries charges of, by name
func chewCharge(unit *ArmyUnit, name string) (spellbook.Spell, bool) {
    for spell, charges := range unit.SpellCharges {
        if spell.Name == name && charges > 0 {
            return spell, true
        }
    }
    return spellbook.Spell{}, false
}

// the charges of a hero's items: the charge spells that are none of the abilities of units
func chewItemCharge(unit *ArmyUnit) (spellbook.Spell, bool) {
    if !unit.Unit.IsHero() {
        return spellbook.Spell{}, false
    }
    var names []spellbook.Spell
    for spell, charges := range unit.SpellCharges {
        switch spell.Name {
            case "Doom Bolt", "Fireball", "Healing", "Web":
                continue
        }
        if charges > 0 {
            names = append(names, spell)
        }
    }
    if len(names) == 0 {
        return spellbook.Spell{}, false
    }
    slices.SortFunc(names, func(a spellbook.Spell, b spellbook.Spell) int {
        return a.Index - b.Index
    })
    return names[0], true
}

// the spells a unit knows and can pay for with its own mana: the spells of its realm for a
// creature, the wizard's and its own for a hero; the fork has them in ArmyUnit.Spells
func chewUnitCastable(unit *ArmyUnit) []spellbook.Spell {
    var out []spellbook.Spell
    for _, spell := range unit.Spells.Spells {
        if spell.BaseCost(false) <= int(unit.CastingSkill) {
            out = append(out, spell)
        }
    }
    return out
}

// the spell abilities of Choose_Target_And_Action, before the unit looks for a target to attack.
// done is true when the unit has picked an action for which it looks no further; else best and
// target are where the search for a target to attack starts
func (model *CombatModel) chewChooseAbility(unit *ArmyUnit, noSpells bool) (action chewAction, target *ArmyUnit, best int, spell spellbook.Spell, done bool) {
    army := model.GetArmy(unit)
    best = chewNoValue
    action = chewReady

    if summon, ok := chewCharge(unit, chewSummonDemonName); ok {
        if len(model.chewSideUnits(army, true)) < chewSummonMost {
            return chewSummonDemon, nil, best, summon, true
        }
    }

    if doom, ok := chewCharge(unit, "Doom Bolt"); ok {
        picked := model.chewSpellTarget(army, doom)
        if picked.Unit != nil {
            // not final: an attack of more value goes first
            action, target, best, spell = chewDoomBolt, picked.Unit, chewDoomBoltValue, doom
        }
    }

    if fireball, ok := chewCharge(unit, "Fireball"); ok {
        picked := model.chewSpellTarget(army, fireball)
        if picked.Unit != nil {
            defense := GetDefenseFor(picked.Unit, data.ChaosMagic)
            if picked.Unit.HasAbility(data.AbilityFireImmunity) || picked.Unit.HasAbility(data.AbilityMagicImmunity) {
                defense = 50
            }
            action, target, best, spell = chewFireball, picked.Unit, chewFireballValue - defense, fireball
        }
    }

    if healing, ok := chewCharge(unit, "Healing"); ok {
        if model.chewHealingTarget(model.chewSideUnits(army, true)).Found {
            return chewHealing, nil, best, healing, true
        }
    }

    if web, ok := chewCharge(unit, "Web"); ok {
        picked := model.chewSpellTarget(army, web)
        if picked.Unit != nil {
            return chewWeb, picked.Unit, best, web, true
        }
    }

    if item, ok := chewItemCharge(unit); ok {
        picked := model.chewSpellTarget(army, item)
        if picked.Found {
            return chewUseItem, picked.Unit, best, item, true
        }
    }

    // a caster casts more often the weaker its magic attack: (ranged - 5) up to a roll of 1 to 15
    if unit.CastingSkill > 2 && !noSpells && len(unit.Spells.Spells) > 0 {
        if unit.GetRangedAttackPower() - 5 <= 1 + rand.IntN(15) {
            return chewCastSpell, nil, best, spellbook.Spell{}, true
        }
    }

    return action, target, best, spell, false
}

// the name of the fork's spell of the ability Summon Demons, see makeSummonDemonSpell
var chewSummonDemonName = makeSummonDemonSpell().Name

// values of Choose_Target_And_Action for the bolts of units
const (
    chewDoomBoltValue = 30
    chewFireballValue = 16
)

// the spell actions of AI_Execute_Unit_Action. true when the unit's turn is over
func (model *CombatModel) chewCastAction(spellSystem SpellSystem, actions AIUnitActionsInterface, unit *ArmyUnit, plan *chewbotPlan) bool {
    army := model.GetArmy(unit)
    chew := model.chewbotState()
    action := plan.Actions[unit]

    useCharge := func(spell spellbook.Spell, target chewTarget) {
        model.chewInvoke(spellSystem, army, unit, spell, target, func(success bool) {
            if unit.SpellCharges[spell] > 0 {
                unit.SpellCharges[spell] -= 1
            }
            unit.Casted = true
            if success {
                spellSystem.PlaySound(spell)
            }
        })
        actions.DoProjectiles()
    }

    switch action {
        case chewDoomBolt, chewFireball, chewWeb:
            target := chew.Targets[unit]
            chewLog("%v uses %v at %v", unit.Unit.GetName(), plan.Spells[unit].Name, chewName(target))
            useCharge(plan.Spells[unit], chewTarget{Unit: target, Found: target != nil})
            return true

        case chewSummonDemon:
            chewLog("%v summons a demon", unit.Unit.GetName())
            useCharge(plan.Spells[unit], chewTarget{Found: true})
            return true

        case chewHealing:
            // see quirkHealingWastesTurn: the ability is used up and nothing happens
            if quirkHealingWastesTurn {
                unit.SpellCharges[plan.Spells[unit]] = 0
                chewLog("%v picks its Healing and does nothing", unit.Unit.GetName())
                return true
            }
            target := model.chewHealingTarget(model.chewSideUnits(army, true))
            useCharge(plan.Spells[unit], target)
            return true

        case chewUseItem:
            spell := plan.Spells[unit]
            chewLog("%v uses the %v of its item", unit.Unit.GetName(), spell.Name)
            useCharge(spell, model.chewSpellTarget(army, spell))
            return true

        case chewCastSpell:
            spell, target, ok := model.chewSelectSpell(army, unit, chewUnitCastable(unit))
            if !ok {
                // it meant to cast and found nothing: its turn is gone
                chewLog("%v meant to cast and found no spell", unit.Unit.GetName())
                return true
            }
            base := spell.BaseCost(false)
            cost := base
            if spell.SpellType >= spellTypeInfusable && spell.IsVariableCost() {
                extra := min(base, int(unit.CastingSkill) - base)
                if extra > 0 {
                    spell.OverrideCost = base + extra
                    cost = base + extra
                }
            }
            chewLog("%v casts %v (cost %v) at %v", unit.Unit.GetName(), spell.Name, cost, chewName(target.Unit))
            model.chewInvoke(spellSystem, army, unit, spell, target, func(success bool) {
                unit.CastingSkill -= float32(cost)
                unit.Casted = true
                if success {
                    spellSystem.PlaySound(spell)
                }
            })
            actions.DoProjectiles()
            return true
    }

    return false
}
