package maplib

import (
    "image"
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/terrain"
)

// Create_Lair: at most 8 of the first kind, 9 in all, a second kind other than the first, treasure
// at least 50
func TestClassicCreateLair(test *testing.T) {
    for _, realm := range []data.MagicType{data.ChaosMagic, data.DeathMagic, data.LifeMagic, data.NatureMagic, data.SorceryMagic} {
        if len(classicGuardianPool(realm)) == 0 {
            test.Errorf("no guardians of %v", realm)
        }
    }

    for range 300 {
        for _, plane := range []data.Plane{data.PlaneArcanus, data.PlaneMyrror} {
            for _, weak := range []bool{false, true} {
                encounter := classicLairEncounter(weak, data.DifficultyImpossible, plane)
                if len(encounter.Units) > 9 {
                    test.Fatalf("%v guardians", len(encounter.Units))
                }
                if encounter.Budget < 50 {
                    test.Fatalf("treasure %v", encounter.Budget)
                }
                first := 0
                for _, unit := range encounter.Units {
                    if unit.Equals(encounter.Units[0]) {
                        first += 1
                    } else if first == 0 {
                        test.Fatalf("the second kind before the first")
                    }
                }
                if first > 8 {
                    test.Fatalf("%v of the first kind", first)
                }
                if weak && len(encounter.Units) > 0 && encounter.Units[0].CastingCost >= 250 {
                    test.Fatalf("a weak lair of Impossible with %v", encounter.Units[0].Name)
                }
            }
        }
    }

    // a node of power 10 on a normal magic setting at Impossible: 100 x 5 to 15 x 1 x 5 / 4
    magicNode, encounter := classicNodeEncounter(terrain.ClassicNode{X: 5, Y: 5, Kind: terrain.ClassicChaos, Power: 3,
        Aura: []image.Point{{5, 5}, {6, 5}, {5, 6}}}, data.MagicSettingNormal, data.DifficultyImpossible, data.PlaneArcanus)
    if magicNode.Kind != MagicNodeChaos || len(magicNode.Zone) != 3 || magicNode.Zone[1] != image.Pt(1, 0) {
        test.Errorf("node %v %v", magicNode.Kind, magicNode.Zone)
    }
    for _, unit := range encounter.Units {
        if unit.Realm != data.ChaosMagic {
            test.Errorf("a guardian of a chaos node of %v", unit.Realm)
        }
    }
}
