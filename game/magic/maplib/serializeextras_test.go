package maplib

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
)

// a loaded map has a table of extras on every square, as a new map has: a mineral put on a square
// without extras crashed a loaded game
func TestLoadedMapTakesNewExtras(test *testing.T) {
    serialized := SerializedMap{Map: [][]int{{0, 0}, {0, 0}}}
    loaded := ReconstructMap(serialized, nil, nil, nil)
    loaded.SetBonus(1, 1, data.BonusGoldOre)
    if loaded.GetBonusTile(1, 1) != data.BonusGoldOre {
        test.Errorf("the mineral is not on the square")
    }
    loaded.SetCorruption(0, 1)
    if !loaded.HasCorruption(0, 1) {
        test.Errorf("the corruption is not on the square")
    }
}
