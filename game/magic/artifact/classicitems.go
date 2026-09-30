package artifact

// Items made by the wizard as the original game has them (ReMoM MoM/src/ItemMake.c:
// Itam_Make_Screen_Build_Weapon_Powers_List, the cost of an item). The code is ours.

// false: upstream's rules of making items
var ClassicItems = true

// the basic powers of an item: the only ones jewelry pays twice for
func isBasicPower(power Power) bool {
    switch power.Type {
        case PowerTypeAttack, PowerTypeDefense, PowerTypeToHit, PowerTypeMovement, PowerTypeResistance, PowerTypeSpellSkill, PowerTypeSpellSave:
            return true
    }
    return false
}

// Enchant Item offers no power over 200, the special powers too
func enchantItemPowers(powers []Power, costs map[Power]int, creationType CreationScreen) []Power {
    if !ClassicItems || creationType != CreationEnchantItem {
        return powers
    }
    var out []Power
    for _, power := range powers {
        if costs[power] <= CreationScreenCostThreshold {
            out = append(out, power)
        }
    }
    return out
}
