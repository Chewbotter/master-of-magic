package maplib

// What is inside the lairs, towers and nodes of the original's worlds: their guardians and the
// budget of their treasure (ReMoM MoM/src/MAPGEN.c: Generate_Lairs, Create_Lair, Generate_Nodes,
// Generate_Towers). The rules in words: docs/mod/worlds.md. The code is ours.

import (
    "image"
    rand "github.com/kazzmir/master-of-magic/lib/chance"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/terrain"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    "github.com/kazzmir/master-of-magic/lib/set"
)

const (
    // Create_Lair: tries of a guardian, and the most of the first kind
    classicGuardianTries = 200
    classicGuardiansMost = 8
    classicStackMost = 9
    // Create_Lair: the least treasure
    classicTreasureLeast = 50
)

// the original's Random(n): 1 to n
func classicLairRoll(n int) int {
    return rand.N(max(n, 1)) + 1
}

// the original's five levels (Extreme counts as Impossible)
func classicLevel(difficulty data.DifficultySetting) int {
    switch difficulty {
        case data.DifficultyIntro: return 0
        case data.DifficultyEasy: return 1
        case data.DifficultyAverage: return 2
        case data.DifficultyHard: return 3
    }
    return 4
}

// the worth of a guardian: the table's cost with the original's units (units/classicunits.go), else
// the casting cost
func guardianCost(unit *units.Unit) int {
    if units.ClassicUnits {
        return unit.WorthCost()
    }
    return unit.CastingCost
}

// the creatures a realm has for guardians, in the game's order: summoned creatures of the realm
// that are not ships (Floating Island)
func classicGuardianPool(realm data.MagicType) []units.Unit {
    var out []units.Unit
    for _, unit := range units.AllUnits {
        if unit.Race == data.RaceFantastic && unit.Realm == realm && guardianCost(&unit) > 0 && !unit.HasAbility(data.AbilityTransport) {
            out = append(out, unit)
        }
    }
    return out
}

// the costliest creature of the pool that costs less than the share, the first of equals; skipping
// one kind
func classicCostliestBelow(pool []units.Unit, share int, not *units.Unit) (units.Unit, bool) {
    var best units.Unit
    found := false
    for _, unit := range pool {
        if not != nil && unit.Equals(*not) {
            continue
        }
        if guardianCost(&unit) < share && (!found || guardianCost(&unit) > guardianCost(&best)) {
            best = unit
            found = true
        }
    }
    return best, found
}

// the realm of the guardians of a lair or tower (nodes have their own)
func classicLairRealm(kind EncounterType) data.MagicType {
    switch kind {
        case EncounterTypePlaneTower:
            switch classicLairRoll(6) {
                case 1, 2: return data.DeathMagic
                case 3: return data.ChaosMagic
                case 4: return data.LifeMagic
                case 5: return data.NatureMagic
            }
            return data.SorceryMagic
        case EncounterTypeAncientTemple, EncounterTypeFallenTemple, EncounterTypeRuins:
            if classicLairRoll(4) == 4 {
                return data.LifeMagic
            }
            return data.DeathMagic
    }
    // cave, dungeon, keep, monster lair: meant death 2 in 5, chaos 2 in 5, nature 1 in 5; the
    // original's choice falls through to nature every time (kept, the roll is still made)
    roll := classicLairRoll(5)
    if quirkLairsAllNature {
        return data.NatureMagic
    }
    switch roll {
        case 1, 2: return data.DeathMagic
        case 3, 4: return data.ChaosMagic
    }
    return data.NatureMagic
}

// Create_Lair: the realm of a cave, dungeon, keep or monster lair falls through to nature (see
// docs/mod/improvements.md)
const quirkLairsAllNature = true

// Create_Lair: guardians for a budget scaled by the difficulty, and the budget of the treasure,
// which is worth what the guardians are
func classicCreateLair(kind EncounterType, realm data.MagicType, budget int, difficulty data.DifficultySetting, plane data.Plane) *ExtraEncounter {
    level := classicLevel(difficulty)
    budget = budget * (level + 1) / 4
    pool := classicGuardianPool(realm)

    var guardians []units.Unit
    first, found := units.Unit{}, false
    for range classicGuardianTries {
        if first, found = classicCostliestBelow(pool, budget / classicLairRoll(4), nil); found {
            break
        }
    }
    firstCount, secondCount := 0, 0
    var second units.Unit
    if found {
        firstCount = budget / guardianCost(&first)
        if firstCount > 1 && classicLairRoll(2) == 2 {
            firstCount -= 1
        }
        firstCount = min(firstCount, classicGuardiansMost)
        budget -= firstCount * guardianCost(&first)

        secondFound := false
        for range classicGuardianTries {
            if second, secondFound = classicCostliestBelow(pool, budget / classicLairRoll(classicStackMost + 1 - firstCount), &first); secondFound {
                break
            }
        }
        if secondFound {
            secondCount = min(budget / guardianCost(&second), classicStackMost - firstCount)
        }
    }
    for range firstCount {
        guardians = append(guardians, first)
    }
    for range secondCount {
        guardians = append(guardians, second)
    }

    // the treasure: the first kind of guardians and half the second (the original's rule), times
    // the difficulty back out (not at Impossible), times 50 to 124 percent (Myrror 76 to 174),
    // at least 50
    treasure := 0
    if firstCount > 0 {
        treasure += guardianCost(&first) * firstCount
    }
    if secondCount > 0 {
        treasure += guardianCost(&second) * secondCount / 2
    }
    if level < 4 {
        treasure = treasure * 4 / (level + 1)
    }
    if plane == data.PlaneMyrror {
        treasure = treasure * (75 + classicLairRoll(100)) / 100
    } else {
        treasure = treasure * (49 + classicLairRoll(76)) / 100
    }
    treasure = max(treasure, classicTreasureLeast)

    return &ExtraEncounter{
        Type: kind,
        Units: guardians,
        Budget: treasure,
        ExploredBy: set.MakeSet[Wizard](),
    }
}

// Generate_Towers: one set of guardians for a tower, the same on both planes; a budget of 700 to
// 1200
func classicTowerEncounter(difficulty data.DifficultySetting) *ExtraEncounter {
    budget := 650 + 50 * classicLairRoll(11)
    return classicCreateLair(EncounterTypePlaneTower, classicLairRealm(EncounterTypePlaneTower), budget, difficulty, data.PlaneArcanus)
}

// Generate_Nodes: power squared times 5 to 15 times the magic setting (half, one, one and a half)
func classicNodeEncounter(node terrain.ClassicNode, magic data.MagicSetting, difficulty data.DifficultySetting, plane data.Plane) (*ExtraMagicNode, *ExtraEncounter) {
    magicLevel := 1
    switch magic {
        case data.MagicSettingWeak: magicLevel = 0
        case data.MagicSettingPowerful: magicLevel = 2
    }
    budget := node.Power * node.Power * (4 + classicLairRoll(11)) * (magicLevel + 1) / 2

    kind, realm, encounterType := MagicNodeNature, data.NatureMagic, EncounterTypeNatureNode
    switch node.Kind {
        case terrain.ClassicSorcery: kind, realm, encounterType = MagicNodeSorcery, data.SorceryMagic, EncounterTypeSorceryNode
        case terrain.ClassicChaos: kind, realm, encounterType = MagicNodeChaos, data.ChaosMagic, EncounterTypeChaosNode
    }

    var zone []image.Point
    for _, point := range node.Aura {
        zone = append(zone, image.Pt(point.X - node.X, point.Y - node.Y))
    }
    return &ExtraMagicNode{Kind: kind, Zone: zone}, classicCreateLair(encounterType, realm, budget, difficulty, plane)
}

// Generate_Lairs: a strong lair has 100 to 1500 (Myrror 200 to 2500), a weak one 10 to 100 (Myrror
// 10 to 200), of a kind by chance
func classicLairEncounter(weak bool, difficulty data.DifficultySetting, plane data.Plane) *ExtraEncounter {
    var budget int
    switch {
        case weak && plane == data.PlaneMyrror: budget = 10 * classicLairRoll(20)
        case weak: budget = 10 * classicLairRoll(10)
        case plane == data.PlaneMyrror: budget = 100 + 100 * classicLairRoll(24)
        default: budget = 50 + 50 * classicLairRoll(29)
    }
    kind := randomEncounterType()
    return classicCreateLair(kind, classicLairRealm(kind), budget, difficulty, plane)
}
