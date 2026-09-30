package artifact

// The original's random item (ReMoM MoM/src/ItemMake.c: Create_Random_Item, the item of a lair's
// treasure: Make_Item is called with its arguments swapped, so a lair's item is always random). The
// powers, their costs and the kinds they fit are the game's data (itempow.lbx, ReadPowers). The code
// is ours.
//
//   worth: the value given, or 800 to 1700 when none; a kind by chance of the 10 and a picture of it;
//   then basic powers by chance (attack, defense, to hit, movement, resistance, skill, spell save),
//   one of a kind of each, that fit the kind and cost at most 200 each (2 at most) for "power" 1,
//   else 20000 (4 at most); it stops at the most powers, once its cost is over the worth (so it goes
//   past it), or after 50 tries with a power

import (
    "math/rand/v2"

    "github.com/kazzmir/master-of-magic/lib/lbx"
)

var classicItemKinds = []ArtifactType{ArtifactTypeSword, ArtifactTypeMace, ArtifactTypeAxe, ArtifactTypeBow,
    ArtifactTypeStaff, ArtifactTypeWand, ArtifactTypeMisc, ArtifactTypeShield, ArtifactTypeChain, ArtifactTypePlate}

// the pictures of a kind in the item archive
func classicItemPicture(kind ArtifactType) int {
    between := func(low int, high int) int {
        return low + rand.N(high - low + 1)
    }
    switch kind {
        case ArtifactTypeSword: return between(0, 8)
        case ArtifactTypeMace: return between(9, 19)
        case ArtifactTypeAxe: return between(20, 28)
        case ArtifactTypeBow: return between(29, 37)
        case ArtifactTypeStaff: return between(38, 46)
        case ArtifactTypeWand: return between(107, 115)
        case ArtifactTypeMisc: return between(72, 106)
        case ArtifactTypeShield: return between(62, 71)
        case ArtifactTypeChain: return between(47, 54)
        case ArtifactTypePlate: return between(55, 61)
    }
    return 0
}

func classicBasicPower(power Power) bool {
    switch power.Type {
        case PowerTypeAttack, PowerTypeDefense, PowerTypeToHit, PowerTypeMovement, PowerTypeResistance, PowerTypeSpellSkill, PowerTypeSpellSave:
            return true
    }
    return false
}

// Create_Random_Item: an item for a "power" and a worth (0: 800 to 1700)
func MakeClassicRandomItem(cache *lbx.LbxCache, power int, value int) (Artifact, bool) {
    all, costs, fits, err := ReadPowers(cache)
    if err != nil {
        return Artifact{}, false
    }
    var basic []Power
    for _, candidate := range all {
        if classicBasicPower(candidate) {
            basic = append(basic, candidate)
        }
    }
    if len(basic) == 0 {
        return Artifact{}, false
    }

    if value == 0 {
        value = 700 + (rand.N(10) + 1) * 100
    }
    costMost, powersMost := 20000, 4
    if power == 1 {
        costMost, powersMost = 200, 2
    }

    item := Artifact{Type: classicItemKinds[rand.N(len(classicItemKinds))]}
    item.Image = classicItemPicture(item.Type)
    has := func(kind PowerType) bool {
        for _, other := range item.Powers {
            if other.Type == kind {
                return true
            }
        }
        return false
    }
    for tries := 0; len(item.Powers) < powersMost && tries < 1000; tries++ {
        if tries >= 50 && len(item.Powers) > 0 {
            break
        }
        candidate := basic[rand.N(len(basic))]
        kinds, ok := fits[candidate]
        if !ok || !kinds.Contains(item.Type) || costs[candidate] > costMost || has(candidate.Type) {
            continue
        }
        item.Powers = append(item.Powers, candidate)
        if calculateCost(&item, costs) > value {
            break
        }
    }
    item.Cost = calculateCost(&item, costs)
    item.Name = getName(&item, "")
    return item, true
}
