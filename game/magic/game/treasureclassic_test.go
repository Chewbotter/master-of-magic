package game

import (
    "testing"
)

// Create_Lair's hoard: at most 3 items, a book or retort takes the place of all else, a tower
// always has a spell or a book or retort, a small budget gives gold or mana only
func TestClassicHoard(test *testing.T) {
    for range 2000 {
        hoard := classicRollHoard(5000, false)
        if len(hoard.Items) > 3 {
            test.Fatalf("%v items", len(hoard.Items))
        }
        if hoard.SpellOrSpecial > 4 && (hoard.Gold > 0 || hoard.Mana > 0 || len(hoard.Items) > 0 || hoard.Prisoner) {
            test.Fatalf("a special with other loot: %+v", hoard)
        }
        if hoard.SpellOrSpecial > 6 {
            test.Fatalf("spell or special %v", hoard.SpellOrSpecial)
        }

        tower := classicRollHoard(400, true)
        if tower.SpellOrSpecial == 0 {
            test.Fatalf("a tower without a spell: %+v", tower)
        }

        small := classicRollHoard(60, false)
        if len(small.Items) > 0 || small.Prisoner || small.SpellOrSpecial > 1 {
            test.Fatalf("a hoard of 60 with %+v", small)
        }
        if small.Gold > 60 || small.Mana > 60 {
            test.Fatalf("a hoard of 60 with %+v", small)
        }
    }
}
