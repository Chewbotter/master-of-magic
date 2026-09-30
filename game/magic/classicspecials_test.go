package main

import (
    "image"
    "slices"
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
)

// Path_Wrap: the squares after the start up to the end, the shorter way around the world
func TestClassicLine(test *testing.T) {
    line := classicLine(2, 5, 6, 7, 60)
    want := []image.Point{{3, 6}, {4, 6}, {5, 7}, {6, 7}}
    if !slices.Equal(line, want) {
        test.Errorf("line 2,5 to 6,7: %v, want %v", line, want)
    }
    around := classicLine(58, 3, 1, 3, 60)
    wantAround := []image.Point{{59, 3}, {0, 3}, {1, 3}}
    if !slices.Equal(around, wantAround) {
        test.Errorf("line around the world: %v, want %v", around, wantAround)
    }
}

// the mineral tables give every roll of their plane a mineral of the original's odds
func TestClassicSpecialTables(test *testing.T) {
    old := classicSpecialRoll
    defer func() { classicSpecialRoll = old }()

    count := func(table classicSpecialTable, plane data.Plane, sides int) map[data.BonusType]int {
        out := make(map[data.BonusType]int)
        for roll := 1; roll <= sides; roll++ {
            classicSpecialRoll = func(int) int { return roll }
            out[table.pickFor(plane)] += 1
        }
        return out
    }

    hills := count(classicHillsSpecials, data.PlaneArcanus, 18)
    if hills[data.BonusIronOre] != 6 || hills[data.BonusSilverOre] != 4 || hills[data.BonusCoal] != 3 || hills[data.BonusGoldOre] != 4 || hills[data.BonusMithrilOre] != 1 {
        test.Errorf("hills of Arcanus: %v", hills)
    }
    mountains := count(classicMountainSpecials, data.PlaneMyrror, 10)
    if mountains[data.BonusAdamantiumOre] != 2 || mountains[data.BonusMithrilOre] != 3 || mountains[data.BonusGoldOre] != 2 {
        test.Errorf("mountains of Myrror: %v", mountains)
    }
    desert := count(classicDesertSpecials, data.PlaneMyrror, 5)
    if desert[data.BonusGem] != 1 || desert[data.BonusQuorkCrystal] != 3 || desert[data.BonusCrysxCrystal] != 1 {
        test.Errorf("desert of Myrror: %v", desert)
    }
}
