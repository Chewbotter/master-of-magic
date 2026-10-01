package combat

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/setup"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// the defender moves first, as in the original, unless the attacking army of a test battle is
// given the first move
func TestFirstMove(test *testing.T) {
    for _, attackerFirst := range []bool{false, true} {
        defendingPlayer := playerlib.MakePlayer(setup.WizardCustom{Name: "Defenders", Banner: data.BannerBrown}, false, 0, 0, nil, &noGlobalEnchantments{})
        attackingPlayer := playerlib.MakePlayer(setup.WizardCustom{Name: "Attackers", Banner: data.BannerRed}, false, 0, 0, nil, &noGlobalEnchantments{})

        attackingArmy := &Army{Player: attackingPlayer, MovesFirst: attackerFirst}
        defendingArmy := &Army{Player: defendingPlayer}
        attackingArmy.AddUnit(units.MakeOverworldUnitFromUnit(units.LizardSwordsmen, 1, 1, data.PlaneArcanus, attackingPlayer.Wizard.Banner, attackingPlayer.MakeExperienceInfo(), attackingPlayer.MakeUnitEnchantmentProvider()))
        defendingArmy.AddUnit(units.MakeOverworldUnitFromUnit(units.LizardSwordsmen, 1, 1, data.PlaneArcanus, defendingPlayer.Wizard.Banner, defendingPlayer.MakeExperienceInfo(), defendingPlayer.MakeUnitEnchantmentProvider()))

        var allSpells spellbook.Spells
        model := MakeCombatModel(allSpells, defendingArmy, attackingArmy, CombatLandscapeGrass, data.PlaneArcanus, ZoneType{}, data.MagicNone, 0, 0, make(chan CombatEvent, 10))

        want := TeamDefender
        if attackerFirst {
            want = TeamAttacker
        }
        if model.Turn != want || model.SelectedUnit == nil || model.SelectedUnit.Team != want {
            test.Fatalf("attacker first %v: turn %v, selected %v", attackerFirst, model.Turn, model.SelectedUnit)
        }
    }
}
