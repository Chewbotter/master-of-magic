package units

import (
    "testing"
)

// UNITTYPE.c: the numbers of the original's table where the fork's differed
func TestClassicUnitData(test *testing.T) {
    ApplyClassicUnitData()
    find := func(lbx string, index int) Unit {
        for _, unit := range AllUnits {
            if unit.LbxFile == lbx && unit.Index == index {
                return unit
            }
        }
        test.Fatalf("no unit %v %v", lbx, index)
        return Unit{}
    }
    if find("units2.lbx", 66).Count != 4 || Gorgon.Count != 4 {
        test.Errorf("Gorgons are 4 figures")
    }
    if find("units2.lbx", 1).HitPoints != 2 {
        test.Errorf("Lizardman Swordsmen have 2 hits")
    }
    if find("units2.lbx", 60).MeleeAttackPower != 2 {
        test.Errorf("Sprites have melee 2")
    }
    if find("units2.lbx", 49).Defense != 5 {
        test.Errorf("a Demon has defense 5")
    }
    if find("units1.lbx", 118).HitPoints != 10 {
        test.Errorf("Klackon Settlers have 10 hits")
    }
    if Gorgon.WorthCost() != 599 || EarthElemental.WorthCost() != 160 || HighMenSpearmen.WorthCost() != HighMenSpearmen.ProductionCost {
        test.Errorf("worth: Gorgons 599, Earth Elemental 160, a normal unit its production cost")
    }
    if classicCharmOfLife(5) != 6 || classicCharmOfLife(2) != 3 || classicCharmOfLife(8) != 10 {
        test.Errorf("Charm of Life adds a quarter, at least 1")
    }
}
