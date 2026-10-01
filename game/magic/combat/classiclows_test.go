package combat

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    "github.com/kazzmir/master-of-magic/lib/fraction"
)

// Battle_Unit_Commit_Damage: irreversible damage wins a tie
func TestClassicDeathReason(test *testing.T) {
    _, attacker, _ := classicBattle(units.LizardSpearmen, classicTarget())
    attacker.NormalDamage = 5
    attacker.IrreversableDamage = 5
    if attacker.DeathReason() != DamageIrreversable {
        test.Errorf("a tie of normal and irreversible damage is irreversible")
    }
    attacker.IrreversableDamage = 0
    if attacker.DeathReason() != DamageNormal {
        test.Errorf("normal damage alone is normal")
    }
}

// Cast_Raise_Dead: half the figures of a unit of several come back whole
func TestClassicRaiseDead(test *testing.T) {
    _, attacker, _ := classicBattle(units.LizardSpearmen, classicTarget())
    figures := attacker.Unit.GetCount()
    perFigure := attacker.GetMaxHealth() / figures
    attacker.RaiseFromDead()
    if attacker.GetHealth() != (figures / 2) * perFigure {
        test.Errorf("raised with %v hits, the original %v", attacker.GetHealth(), (figures / 2) * perFigure)
    }
}

// Check_Attack_Melee, Battle_Unit_Process_Attack: a bowman may strike a flier, for nothing
func TestClassicMeleeOnFliers(test *testing.T) {
    flier := classicTarget()
    flier.Flying = true
    model, attacker, defender := classicBattle(units.HighMenBowmen, flier)
    attacker.MovesLeft = fraction.FromInt(1)
    if !model.canMeleeAttack(attacker, defender, false) {
        test.Fatalf("a unit with arrows left may strike a flier")
    }
    rolls, _ := attacker.ComputeMeleeDamage(defender, 0, false)
    if len(rolls) != 0 {
        test.Errorf("a bowman strikes a flier for nothing")
    }
}

// Begin_Combat_Turn: Entangle takes a move of the turn from fliers too
func TestClassicEntangle(test *testing.T) {
    flier := classicTarget()
    flier.Flying = true
    flier.MovementSpeed = 3
    model, _, defender := classicBattle(units.LizardSpearmen, flier)
    model.AttackingArmy.Enchantments = append(model.AttackingArmy.Enchantments, data.CombatEnchantmentEntangle)
    defender.ResetTurnData()
    if !defender.MovesLeft.Equals(defender.GetMovementSpeed().Subtract(fraction.FromInt(1))) {
        test.Errorf("Entangle takes a move from a flier, moves %v of %v", defender.MovesLeft, defender.GetMovementSpeed())
    }
}
