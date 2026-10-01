package combat

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// Combat_Spell_Target_Screen: the units the human may pick for a spell in battle
func TestClassicTargetFilter(test *testing.T) {
    _, attacker, defender := classicBattle(units.LizardSpearmen, units.WarBear)

    bless, _ := classicTargetFilter("Bless")
    if !bless(attacker) {
        test.Fatalf("Bless on a unit without it")
    }
    attacker.AddEnchantment(data.UnitEnchantmentBless)
    if bless(attacker) {
        test.Errorf("Bless on a unit that has it")
    }

    // each skin looks at its own
    attacker.AddEnchantment(data.UnitEnchantmentStoneSkin)
    ironSkin, _ := classicTargetFilter("Iron Skin")
    if !ironSkin(attacker) {
        test.Errorf("Iron Skin may go onto Stone Skin")
    }

    // normal units only
    heroism, _ := classicTargetFilter("Heroism")
    if !heroism(attacker) || heroism(defender) {
        test.Errorf("Heroism for normal units only")
    }
    binding, _ := classicTargetFilter("Creature Binding")
    if binding(attacker) || !binding(defender) {
        test.Errorf("Creature Binding for creatures only")
    }

    // Lionheart, Righteousness, True Sight: no refusal by the realm of the unit
    lionheart, _ := classicTargetFilter("Lionheart")
    if !lionheart(defender) {
        test.Errorf("Lionheart on any unit that lacks it")
    }

    // a web holds: no second one
    web, _ := classicTargetFilter("Web")
    defender.WebHealth = 3
    if web(defender) {
        test.Errorf("Web on a unit in a web")
    }
}

// Apply_Call_Lightning: 3 to 5 bolts, all at living enemies, of nature
func TestClassicCallLightning(test *testing.T) {
    model, _, defender := classicBattle(units.LizardSpearmen, classicTarget())
    for range 50 {
        model.Events = make(chan CombatEvent, 100)
        model.classicCallLightning(TeamAttacker)
        close(model.Events)
        count := 0
        for event := range model.Events {
            bolt, ok := event.(*CombatEventCreateLightningBolt)
            if !ok {
                continue
            }
            count += 1
            if bolt.Target != defender || !bolt.Nature || bolt.Strength != 8 {
                test.Fatalf("a bolt at %v, nature %v, strength %v", bolt.Target, bolt.Nature, bolt.Strength)
            }
        }
        if count < 3 || count > 5 {
            test.Fatalf("%v bolts", count)
        }
    }
}

// Battle_Unit_Special_Stats: Metal Fires for Chaos Channels units, not for undead or creatures
func TestClassicMetalFires(test *testing.T) {
    model, attacker, defender := classicBattle(units.LizardSpearmen, units.WarBear)
    model.AttackingArmy.Enchantments = append(model.AttackingArmy.Enchantments, data.CombatEnchantmentMetalFires)
    model.DefendingArmy.Enchantments = append(model.DefendingArmy.Enchantments, data.CombatEnchantmentMetalFires)
    if !attacker.metalFiresHolds() {
        test.Errorf("Metal Fires for a normal unit")
    }
    if defender.metalFiresHolds() {
        test.Errorf("no Metal Fires for a creature")
    }
    attacker.Unit.SetUndead()
    if attacker.metalFiresHolds() {
        test.Errorf("no Metal Fires for the undead")
    }
}

// Cast_Raise_Dead: a unit of one figure comes back with its hits without its enchantments, less
// half its hits with them
func TestClassicRaiseDeadLionheart(test *testing.T) {
    kind := classicTarget()
    kind.HitPoints = 10
    kind.Count = 1
    _, attacker, _ := classicBattle(kind, classicTarget())
    attacker.AddEnchantment(data.UnitEnchantmentLionHeart)
    withIt := attacker.GetMaxHealth()
    attacker.RaiseFromDead()
    if want := 10 - withIt / 2; attacker.GetHealth() != want {
        test.Errorf("raised with %v hits, the original %v (with Lion Heart %v)", attacker.GetHealth(), want, withIt)
    }
}
