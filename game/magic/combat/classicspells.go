package combat

// THE LAST OF THE SPELLS IN BATTLE (user 2026-10-01: "Let's update all spells to be classic"). What
// the notes still listed as not ported, read anew from ReMoM by helpers: Combat.c
// Combat_Spell_Target_Screen (the targets the human may pick, checked against the program by
// ReMoM's review), COMBINIT.c Battle_Unit_Special_Stats (Metal Fires, Holy Weapon),
// Spells131.c Cast_Raise_Dead, Cast_Animate_Dead, Apply_Cracks_Call, Combat_Cast_Disenchant,
// Spells133.c Apply_Call_Lightning, Combat.c Begin_Combat_Turn. Under ClassicRules; the rules in
// words are in docs/mod/combat-rules.md "The last of the spells".

import (
    "math/rand/v2"
    "slices"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// the original builds the mask of "already has" of a curse from its first and third word only:
// Possession and Creature Binding, whose bits are in the second, are never refused for it
const quirkCurseMaskSkips = true

// the battle race of a unit is a normal one (below rt_Arcane): not a creature, not undead, not
// changed by Black or Chaos Channels (Apply_Enchantment_And_Mutation_Effects)
func classicNormalRace(unit *ArmyUnit) bool {
    // GetRace counts undead and Chaos Channels as fantastic already
    return unit.GetRace() != data.RaceFantastic && !unit.HasEnchantment(data.UnitEnchantmentBlackChannels)
}

// UA_FANTASTIC: creatures, undead and Black Channels units. Chaos Channels units are not
func classicFantasticFlag(unit *ArmyUnit) bool {
    return unit.Unit.GetRawUnit().Race == data.RaceFantastic || unit.IsUndead() || unit.HasEnchantment(data.UnitEnchantmentBlackChannels)
}

// Metal Fires gives a unit of its side +1 melee, +1 for arrows, rocks... and thrown, and counts
// as a magic weapon: not for UA_FANTASTIC units (Chaos Channels units do get it), not with Flame
// Blade. no to-hit
func (unit *ArmyUnit) metalFiresHolds() bool {
    if unit.Model == nil || !unit.Model.IsEnchantmentActive(data.CombatEnchantmentMetalFires, unit.Team) || unit.HasEnchantment(data.UnitEnchantmentFlameBlade) {
        return false
    }
    if ClassicRules {
        return !classicFantasticFlag(unit)
    }
    return unit.Unit.GetRace() != data.RaceFantastic
}

// Combat_Spell_Target_Screen: the units the human may pick for a spell cast on a unit in battle.
// false: the spell keeps upstream's filter
func classicTargetFilter(name string) (func(*ArmyUnit) bool, bool) {
    // a buff: any friendly unit that has none of it (from the world map, an item or this battle),
    // and none of the ability it gives by nature
    buff := func(enchantment data.UnitEnchantment, natural data.AbilityType) func(*ArmyUnit) bool {
        return func(target *ArmyUnit) bool {
            if target == nil || target.HasEnchantment(enchantment) {
                return false
            }
            if natural != data.AbilityNone && target.HasAbility(natural) {
                return false
            }
            return true
        }
    }
    // a buff for normal units only (scc_Unit_Enchantment_Normal_Only)
    normalBuff := func(enchantment data.UnitEnchantment) func(*ArmyUnit) bool {
        return func(target *ArmyUnit) bool {
            return target != nil && classicNormalRace(target) && !target.HasEnchantment(enchantment)
        }
    }
    // a curse: refused when it has it, and for an immunity of its realm (sorcery: illusions,
    // death: death). magic immunity, Righteousness and charms only raise the resistance
    curse := func(have data.UnitEnchantment, immunity data.AbilityType) func(*ArmyUnit) bool {
        return func(target *ArmyUnit) bool {
            if target == nil {
                return false
            }
            if have != data.UnitEnchantmentNone && target.HasCurse(have) {
                return false
            }
            if immunity != data.AbilityNone && target.HasAbility(immunity) {
                return false
            }
            return true
        }
    }

    switch name {
        case "Bless": return buff(data.UnitEnchantmentBless, data.AbilityNone), true
        case "Giant Strength": return buff(data.UnitEnchantmentGiantStrength, data.AbilityNone), true
        case "Lionheart": return buff(data.UnitEnchantmentLionHeart, data.AbilityNone), true
        case "Resist Elements": return buff(data.UnitEnchantmentResistElements, data.AbilityNone), true
        case "Resist Magic": return buff(data.UnitEnchantmentResistMagic, data.AbilityNone), true
        case "Elemental Armor": return buff(data.UnitEnchantmentElementalArmor, data.AbilityNone), true
        case "Righteousness": return buff(data.UnitEnchantmentRighteousness, data.AbilityNone), true
        case "Cloak of Fear": return buff(data.UnitEnchantmentCloakOfFear, data.AbilityCauseFear), true
        case "True Sight": return buff(data.UnitEnchantmentTrueSight, data.AbilityIllusionsImmunity), true
        case "Flight": return buff(data.UnitEnchantmentFlight, data.AbilityNone), true
        case "Invulnerability": return buff(data.UnitEnchantmentInvulnerability, data.AbilityNone), true
        // each looks at its own enchantment only: Stone Skin may go onto Iron Skin and back
        case "Iron Skin": return buff(data.UnitEnchantmentIronSkin, data.AbilityNone), true
        case "Stone Skin": return buff(data.UnitEnchantmentStoneSkin, data.AbilityNone), true
        case "Regeneration": return buff(data.UnitEnchantmentRegeneration, data.AbilityRegeneration), true
        case "Guardian Wind": return buff(data.UnitEnchantmentGuardianWind, data.AbilityNone), true
        case "Invisiblity": return buff(data.UnitEnchantmentInvisibility, data.AbilityInvisibility), true
        case "Magic Immunity": return buff(data.UnitEnchantmentMagicImmunity, data.AbilityMagicImmunity), true
        case "Haste": return buff(data.UnitEnchantmentHaste, data.AbilityNone), true
        case "Wraith Form": return buff(data.UnitEnchantmentWraithForm, data.AbilityNonCorporeal), true
        case "Berserk": return buff(data.UnitEnchantmentBerserk, data.AbilityNone), true
        case "Immolation": return buff(data.UnitEnchantmentImmolation, data.AbilityImmolation), true
        case "Spell Lock": return buff(data.UnitEnchantmentSpellLock, data.AbilityNone), true

        // heroes too, and no level check for the human
        case "Heroism": return normalBuff(data.UnitEnchantmentHeroism), true
        case "Holy Armor": return normalBuff(data.UnitEnchantmentHolyArmor), true
        case "Holy Weapon": return normalBuff(data.UnitEnchantmentHolyWeapon), true
        case "Eldritch Weapon": return normalBuff(data.UnitEnchantmentEldritchWeapon), true
        case "Flame Blade": return normalBuff(data.UnitEnchantmentFlameBlade), true

        case "Black Sleep": return curse(data.UnitCurseBlackSleep, data.AbilityDeathImmunity), true
        case "Weakness": return curse(data.UnitCurseWeakness, data.AbilityDeathImmunity), true
        case "Confusion": return curse(data.UnitCurseConfusion, data.AbilityIllusionsImmunity), true
        case "Vertigo": return curse(data.UnitCurseVertigo, data.AbilityIllusionsImmunity), true
        case "Mind Storm": return curse(data.UnitCurseMindStorm, data.AbilityIllusionsImmunity), true
        case "Shatter":
            // a mundane curse: normal units only
            shatter := curse(data.UnitCurseShatter, data.AbilityNone)
            return func(target *ArmyUnit) bool {
                return shatter(target) && classicNormalRace(target)
            }, true
        case "Possession":
            have := data.UnitCursePossession
            if quirkCurseMaskSkips {
                have = data.UnitEnchantmentNone
            }
            possession := curse(have, data.AbilityDeathImmunity)
            return func(target *ArmyUnit) bool {
                return possession(target) && classicNormalRace(target)
            }, true
        case "Creature Binding":
            have := data.UnitCurseCreatureBinding
            if quirkCurseMaskSkips {
                have = data.UnitEnchantmentNone
            }
            binding := curse(have, data.AbilityIllusionsImmunity)
            return func(target *ArmyUnit) bool {
                return binding(target) && !classicNormalRace(target)
            }, true
        case "Web":
            return func(target *ArmyUnit) bool {
                return target != nil && !target.HasAbility(data.AbilityNonCorporeal) && target.WebHealth <= 0
            }, true
    }
    return nil, false
}

// Apply_Cracks_Call: the square of the spell. every unit on it that does not fly and is not
// non-corporeal, of either side, invisible or merging too, is swallowed one time in 4 (200 doom
// damage: gone for good); a standing wall on it always falls. no resistance, no magic immunity
func (model *CombatModel) classicCracksCallEffect(x int, y int) func(*ArmyUnit) {
    return func(_ *ArmyUnit) {
        if unit := model.GetUnit(x, y); unit != nil && unit.GetHealth() > 0 {
            if !unit.IsFlying() && !unit.HasAbility(data.AbilityNonCorporeal) && rand.N(4) == 0 {
                model.RemoveUnit(unit)
            }
        }
        if model.DestroyWall(x, y) && model.SelectedUnit != nil {
            model.SelectedUnit.Paths = nil
        }
    }
}

// Chewbot picks a unit for Cracks Call: the spell takes its square
func (model *CombatModel) chewPendingUnitAsTile() {
    if model.chewbot == nil || model.chewbot.pending == nil {
        return
    }
    pending := model.chewbot.pending
    if !pending.HasTile && pending.Unit != nil {
        pending.Tile.X, pending.Tile.Y = pending.Unit.X, pending.Unit.Y
        pending.HasTile = true
    }
}

// Apply_Call_Lightning for the side that cast it, at the start of a round: 3 to 5 bolts; a try
// picks any unit of the battle (own, dead) and strikes it if it is an enemy, alive, without a
// Wraith Form cast in this battle, and wins a roll of 1 in 2; a bolt starts the count of tries
// anew, 30 failed tries in a row end it. a bolt is the spell data of Wall of Stone: strength 8,
// armor piercing, nature
func (model *CombatModel) classicCallLightning(caster Team) {
    var slots []*ArmyUnit
    for _, army := range []*Army{model.AttackingArmy, model.DefendingArmy} {
        slots = append(slots, army.units...)
        slots = append(slots, army.KilledUnits...)
        slots = append(slots, army.GoneUnits...)
    }
    if len(slots) == 0 {
        return
    }

    bolts := rand.N(3) + 3
    tries := 0
    for bolts > 0 && tries < 30 {
        tries += 1
        target := slots[rand.N(len(slots))]
        if target.Team == caster || target.GetHealth() <= 0 || !slices.Contains(model.GetArmyForTeam(target.Team).units, target) {
            continue
        }
        if quirkWraithFormSpares && target.HasEnchantmentOnly(data.UnitEnchantmentWraithForm) {
            continue
        }
        if rand.N(2) != 0 {
            continue
        }
        model.Events <- &CombatEventCreateLightningBolt{Target: target, Strength: 8, Nature: true}
        bolts -= 1
        tries = 0
    }
}

// the damage of a bolt of Call Lightning: one attack of strength 8, armor piercing, nature
func (model *CombatModel) CreateCallLightningEffect(strength int, damageIndicator AddDamageIndicators) func(*ArmyUnit) {
    return func(unit *ArmyUnit) {
        if unit.GetHealth() <= 0 {
            return
        }
        hurt, _ := ApplyDamage(unit, []int{spellRoll(unit, strength)}, units.DamageRangedMagical, DamageSourceSpell, DamageModifiers{ArmorPiercing: true, Magic: data.NatureMagic})
        damageIndicator.AddDamageIndicator(unit, hurt)
        if unit.GetHealth() <= 0 {
            model.KillUnit(unit)
        }
    }
}

// a warped node the battle is fought on (set by the game): Disenchant Area and True of the wizard
// who owns the node may lift the warp
type BattleNode struct {
    // the wizard the node is melded to
    Owner any
    Unwarp func()
}

// Combat_Cast_Disenchant, the rest: the caster's own warped node of the battle's square by a roll
// against Warp Node (75, no retorts), silent; the town's Heavenly Light and Cloud of Shadow for an
// attacker; a vortex on the square of a unit is rolled a second time
func (model *CombatModel) classicDisenchantNodeAndTown(caster ArmyPlayer, strength int, roll func(cost int, magic data.MagicType, owner ArmyPlayer) bool, costOf func(name string) int) {
    if node := model.WarpedNode; node != nil && node.Unwarp != nil && node.Owner == any(caster) {
        threshold := 250 * strength / max(1, strength + costOf("Warp Node"))
        if rand.N(250) + 1 <= threshold {
            node.Unwarp()
            model.WarpedNode = nil
        }
    }

    casterTeam := model.GetTeamForArmy(model.GetArmyForPlayer(caster))

    // Combat_Cast_Dispel on the square of every unit: a vortex there is rolled again
    var keep []*MagicVortex
    for _, vortex := range model.MagicVortexes {
        if vortex.Team != casterTeam && model.GetUnit(vortex.X, vortex.Y) != nil && roll(costOf("Magic Vortex"), data.ChaosMagic, model.GetArmyForTeam(vortex.Team).Player) {
            continue
        }
        keep = append(keep, vortex)
    }
    model.MagicVortexes = keep

    if model.Zone.City == nil || casterTeam == TeamDefender {
        return
    }
    defender := model.DefendingArmy.Player
    removeGlobal := func(enchantment data.CombatEnchantment) {
        if index := slices.Index(model.GlobalEnchantments, enchantment); index >= 0 {
            model.GlobalEnchantments = slices.Delete(model.GlobalEnchantments, index, index + 1)
        }
    }
    if model.Zone.City.HasEnchantment(data.CityEnchantmentHeavenlyLight) && roll(costOf("Heavenly Light"), data.LifeMagic, defender) {
        removeGlobal(data.CombatEnchantmentTrueLight)
    }
    if model.Zone.City.HasEnchantment(data.CityEnchantmentCloudOfShadow) && roll(costOf("Cloud of Shadow"), data.DeathMagic, defender) {
        removeGlobal(data.CombatEnchantmentDarkness)
    }
}

// the neutral player: its combat enchantments are never reached by Disenchant (the original skips
// the pass for it)
func classicIsNeutral(player ArmyPlayer) bool {
    wizard := player.GetWizard()
    return wizard != nil && wizard.Banner == data.BannerBrown
}

// the spells of the book of a battle: Cracks Call is not there in a battle at sea
func ClassicBattleSpells(spells spellbook.Spells, atSea bool) spellbook.Spells {
    if !ClassicRules || !atSea {
        return spells
    }
    var out spellbook.Spells
    for _, spell := range spells.Spells {
        if spell.Name != "Cracks Call" {
            out.AddSpell(spell)
        }
    }
    return out
}

// development (-capture-spell crackssquare): the original's Cracks Call at a square
func (model *CombatModel) DevCracksCallAt(spellSystem SpellSystem, x int, y int) {
    target := model.GetUnit(x, y)
    if target == nil {
        target = &ArmyUnit{X: x, Y: y}
    }
    projectile := spellSystem.CreateCracksCallProjectile(target)
    projectile.Effect = model.classicCracksCallEffect(x, y)
    model.AddProjectile(projectile)
}

// development (-capture-spell calllightning): the bolts of a side's Call Lightning
func (model *CombatModel) DevCallLightning(team Team) {
    model.classicCallLightning(team)
}
