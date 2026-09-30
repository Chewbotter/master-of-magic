package combat

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    herolib "github.com/kazzmir/master-of-magic/game/magic/hero"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/setup"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    "github.com/kazzmir/master-of-magic/lib/fraction"
)

func classicBattle(attackerKind units.Unit, defenderKind units.Unit) (*CombatModel, *ArmyUnit, *ArmyUnit) {
    defendingArmy := &Army{
        Player: playerlib.MakePlayer(setup.WizardCustom{}, false, 1, 1, map[herolib.HeroType]string{}, &playerlib.NoGlobalEnchantments{}),
    }
    attackingArmy := &Army{
        Player: playerlib.MakePlayer(setup.WizardCustom{}, false, 1, 1, map[herolib.HeroType]string{}, &playerlib.NoGlobalEnchantments{}),
    }
    defendingArmy.AddUnit(units.MakeOverworldUnitFromUnit(defenderKind, 0, 0, data.PlaneArcanus, data.BannerRed, &units.NoExperienceInfo{}, &units.NoEnchantments{}))
    attackingArmy.AddUnit(units.MakeOverworldUnitFromUnit(attackerKind, 0, 0, data.PlaneArcanus, data.BannerBlue, &units.NoExperienceInfo{}, &units.NoEnchantments{}))

    model := &CombatModel{
        Tiles: makeTiles(30, 30, CombatLandscapeGrass, data.PlaneArcanus, ZoneType{}),
        Turn: TeamDefender,
        DefendingArmy: defendingArmy,
        AttackingArmy: attackingArmy,
    }
    model.Initialize(spellbook.Spells{}, 0, 0)
    return model, attackingArmy.units[0], defendingArmy.units[0]
}

// a wall of meat: never dies, never strikes back
func classicTarget() units.Unit {
    target := units.LizardSpearmen
    target.MeleeAttackPower = 0
    target.HitPoints = 10000
    return target
}

// Battle_Unit_Attack: half the full moves rounded up in half moves, paid once
func TestClassicMeleeCost(test *testing.T) {
    model, attacker, defender := classicBattle(units.LizardSpearmen, classicTarget())
    attacker.MovesLeft = fraction.FromInt(1)
    model.meleeAttack(attacker, defender)
    if !attacker.MovesLeft.Equals(fraction.Make(1, 2)) {
        test.Errorf("a unit of 1 move should have half a move after a strike, has %v", attacker.MovesLeft)
    }
    if !model.canMeleeAttack(attacker, defender, false) {
        test.Errorf("with half a move left it strikes again")
    }
}

// Battle_Unit_Attack_Target: First Strike with Haste strikes once
func TestClassicFirstStrikeHaste(test *testing.T) {
    kind := units.LizardSpearmen
    kind.Abilities = append(append([]data.Ability{}, kind.Abilities...), data.MakeAbility(data.AbilityFirstStrike))
    model, attacker, defender := classicBattle(kind, classicTarget())
    attacker.AddEnchantment(data.UnitEnchantmentHaste)

    strikes := 0
    model.Observer.AddObserver(&TestObserver{
        Melee: func(meleeAttacker *ArmyUnit, meleeDefender *ArmyUnit, damageRoll []int) {
            if meleeAttacker == attacker {
                strikes += 1
            }
        },
    })
    model.meleeAttack(attacker, defender)
    if strikes != 1 {
        test.Errorf("First Strike with Haste should strike once, struck %v times", strikes)
    }
}

// Battle_Unit_Attack: an adjacent enemy is shot only when the ranged attack is more than half the melee
func TestClassicShootsAdjacent(test *testing.T) {
    shooter := units.HighMenBowmen
    model, attacker, defender := classicBattle(shooter, classicTarget())
    attacker.X, attacker.Y = 10, 10
    defender.X, defender.Y = 11, 10
    attacker.MovesLeft = fraction.FromInt(1)
    if !model.canRangeAttack(attacker, defender) {
        test.Fatalf("the bowmen should be able to shoot")
    }
    want := attacker.GetRangedAttackPower() > attacker.GetMeleeAttackPower() / 2
    if model.shootsAt(attacker, defender) != want {
        test.Errorf("bowmen next to an enemy: shoot %v, want %v (ranged %v, melee %v)", !want, want, attacker.GetRangedAttackPower(), attacker.GetMeleeAttackPower())
    }
}
