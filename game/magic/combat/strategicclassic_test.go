package combat

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/setup"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

func strategicArmies(attackers []units.Unit, defenders []units.Unit) (*Army, *Army) {
    defendingPlayer := playerlib.MakePlayer(setup.WizardCustom{Name: "Lair", Banner: data.BannerBrown}, false, 0, 0, nil, &noGlobalEnchantments{})
    attackingPlayer := playerlib.MakePlayer(setup.WizardCustom{Name: "AI", Banner: data.BannerRed}, false, 0, 0, nil, &noGlobalEnchantments{})
    attackingArmy := &Army{Player: attackingPlayer}
    defendingArmy := &Army{Player: defendingPlayer}
    for _, unit := range attackers {
        attackingArmy.AddUnit(units.MakeOverworldUnitFromUnit(unit, 1, 1, data.PlaneArcanus, attackingPlayer.Wizard.Banner, attackingPlayer.MakeExperienceInfo(), attackingPlayer.MakeUnitEnchantmentProvider()))
    }
    for _, unit := range defenders {
        defendingArmy.AddUnit(units.MakeOverworldUnitFromUnit(unit, 1, 1, data.PlaneArcanus, defendingPlayer.Wizard.Banner, defendingPlayer.MakeExperienceInfo(), defendingPlayer.MakeUnitEnchantmentProvider()))
    }
    var allSpells spellbook.Spells
    MakeCombatModel(allSpells, defendingArmy, attackingArmy, CombatLandscapeGrass, data.PlaneArcanus, ZoneType{}, data.MagicNone, 0, 0, make(chan CombatEvent, 10))
    return attackingArmy, defendingArmy
}

func alive(army *Army) int {
    count := 0
    for _, unit := range army.units {
        if unit.GetHealth() > 0 {
            count += 1
        }
    }
    return count
}

// Strategic_Combat: the loser loses every unit, the winner some; a computer wizard gets half as much
// again against the monsters of a lair
func TestClassicStrategicCombat(test *testing.T) {
    old := strategicRoll
    defer func() { strategicRoll = old }()
    strategicRoll = func(n int) int { return (n + 1) / 2 }

    wizard := StrategicSide{Wizard: true}
    monsters := StrategicSide{Neutral: true}

    attackers, defenders := strategicArmies([]units.Unit{units.GreatDrake, units.GreatDrake}, []units.Unit{units.LizardSwordsmen, units.LizardSwordsmen})
    state, lostAttackers, lostDefenders, _, _ := DoClassicStrategicCombat(attackers, defenders, wizard, monsters, StructureNone)
    if state != CombatStateAttackerWin || lostDefenders != 2 || alive(defenders) != 0 {
        test.Errorf("drakes against swordsmen: state %v, lost %v and %v", state, lostAttackers, lostDefenders)
    }
    if alive(attackers) + lostAttackers != 2 {
        test.Errorf("the attackers' losses do not add up: %v alive, %v lost", alive(attackers), lostAttackers)
    }

    attackers, defenders = strategicArmies([]units.Unit{units.HighMenSpearmen}, []units.Unit{units.GreatDrake})
    state, lostAttackers, _, _, _ = DoClassicStrategicCombat(attackers, defenders, wizard, monsters, StructureNone)
    if state != CombatStateDefenderWin || lostAttackers != 1 || alive(defenders) != 1 {
        test.Errorf("spearmen against a drake: state %v, lost %v", state, lostAttackers)
    }

    if StrategicCastMultiplier(0, false) != 5 || StrategicCastMultiplier(7, false) != 15 || StrategicCastMultiplier(30, false) != 30 || StrategicCastMultiplier(30, true) != 10 {
        test.Errorf("cast multipliers")
    }
}
