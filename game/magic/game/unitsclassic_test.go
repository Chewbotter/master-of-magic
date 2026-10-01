package game

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/maplib"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// Next_Turn_Process_Purify: one priest takes 5 turns, two 3, four 2
func TestClassicPurify(test *testing.T) {
    for _, check := range []struct{Priests int; Turns int}{{1, 5}, {2, 3}, {3, 3}, {4, 2}} {
        scenario := makeChewScenario(test, 50, chewIsland...)
        player := scenario.wizard("Merlin", data.BannerRed)
        mapUse := scenario.Model.ArcanusMap
        mapUse.SetCorruption(5, 5)
        var kinds []units.Unit
        for range check.Priests {
            kinds = append(kinds, units.HighMenPriest)
        }
        for _, unit := range scenario.units(player, 5, 5, kinds...) {
            unit.SetBusy(units.BusyStatusPurify)
        }
        turns := 0
        for mapUse.HasCorruption(5, 5) && turns < 10 {
            scenario.Model.classicPurify(player)
            turns += 1
        }
        if turns != check.Turns {
            test.Errorf("%v priests purified in %v turns, the original %v", check.Priests, turns, check.Turns)
        }
    }
}

// Evict_Weakest_Unit: the tenth unit on a square is the weakest, it goes to the first free square
// around from the upper left
func TestClassicEvictWeakest(test *testing.T) {
    scenario := makeChewScenario(test, 50, chewIsland...)
    player := scenario.wizard("Merlin", data.BannerRed)
    game := &Game{Model: scenario.Model, headless: true}
    var kinds []units.Unit
    for range 9 {
        kinds = append(kinds, units.HighMenMagician)
    }
    scenario.units(player, 5, 5, kinds...)
    weak := scenario.units(player, 5, 5, units.HighMenSpearmen)[0]

    game.classicEvictWeakest(5, 5, data.PlaneArcanus)

    stack := player.FindStack(5, 5, data.PlaneArcanus)
    if len(stack.Units()) != 9 {
        test.Fatalf("the square keeps 9 units, it has %v", len(stack.Units()))
    }
    if weak.GetX() != 4 || weak.GetY() != 4 {
        test.Errorf("the spearmen should go to 4,4, they are at %v,%v", weak.GetX(), weak.GetY())
    }
}

// Heal_Unit: no least of 1, the rest is a chance
func TestClassicHealAmount(test *testing.T) {
    healed := 0
    for range 2000 {
        healed += units.ClassicHealAmount(10, 20)
    }
    // 10 / 20 is 0 and a chance of 10 in 20: about 1000
    if healed < 850 || healed > 1150 {
        test.Errorf("a unit of 10 hits should heal 1 half of the time, healed %v in 2000", healed)
    }
    if units.ClassicHealAmount(40, 10) != 4 {
        test.Errorf("40 hits at 10 heal 4")
    }
}

// Army_Do_Meld: a Guardian Spirit guards the wizard's own node; a Magic Spirit can not meld it again
func TestClassicMeldOwnNode(test *testing.T) {
    scenario := makeChewScenario(test, 50, chewIsland...)
    player := scenario.wizard("Merlin", data.BannerRed)
    node := &maplib.ExtraMagicNode{MeldingWizard: player}

    magic := scenario.units(player, 5, 5, units.MagicSpirit)
    if classicCanMeld(magic, player, node) {
        test.Errorf("a Magic Spirit can not meld the wizard's own node")
    }
    guardian := scenario.units(player, 6, 6, units.GuardianSpirit)
    if !classicCanMeld(guardian, player, node) {
        test.Errorf("a Guardian Spirit guards the wizard's own node")
    }
    node.GuardianSpiritMeld = true
    if classicCanMeld(guardian, player, node) {
        test.Errorf("a guarded node of the wizard's own takes no more spirits")
    }
}
