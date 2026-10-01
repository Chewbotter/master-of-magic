package units

// Units as the original game has them. Rules in words: docs/mod/units.md.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/UNITTYPE.c: the table of
// unit types, checked against the program of the Steam release; COMBINIT.c: Unit_Hit_Points;
// NEXTTURN.c: Heal_Unit, Heal_All_Units; CITYCALC.c: Calc_Unit_Level). The code is ours.

import (
    "math/rand/v2"

    "github.com/kazzmir/master-of-magic/game/magic/data"
)

// false: upstream's unit rules and data
var ClassicUnits = true

// a fix of the table: the unit of this archive entry, its name to be sure, and what changes
type classicUnitFix struct {
    LbxFile string
    Index int
    Name string
    Fix func(unit *Unit)
}

// the original's table where the fork's differs (UNITTYPE.c, the same as the program's bytes)
var classicUnitFixes = []classicUnitFix{
    {"units2.lbx", 66, "Gorgons", func(unit *Unit) { unit.Count = 4; unit.TableCost = 599 }},
    {"units2.lbx", 1, "Swordsmen", func(unit *Unit) { unit.HitPoints = 2 }},
    {"units2.lbx", 60, "Sprites", func(unit *Unit) { unit.MeleeAttackPower = 2 }},
    {"units2.lbx", 49, "Demon", func(unit *Unit) { unit.Defense = 5; unit.TableCost = 80 }},
    {"units1.lbx", 10, "Greyfairer", func(unit *Unit) { unit.HitPoints = 5 }},
    {"units1.lbx", 100, "Settlers", func(unit *Unit) { unit.MeleeAttackPower = 0 }},
    {"units1.lbx", 118, "Settlers", func(unit *Unit) { unit.HitPoints = 10 }},
}

// the table's cost of creatures whose spell costs another (the game reads the table's for the
// worth of a creature: lair guardians, the fame of a rare foe)
var classicTableCosts = map[string]int{
    "Chaos Spawn": 400,
    "WereWolves": 250,
    "Guardian Spirit": 50,
    "Demon Lord": 900,
    "Nagas": 120,
    "Fire Elemental": 100,
    "Earth Elemental": 160,
    "Air Elemental": 170,
    "Phantom Beast": 225,
    "Phantom Warriors": 20,
    "Zombies": 30,
}

// the named units of the fixes, so a unit made from them has the original's numbers too
func classicNamedUnits() []*Unit {
    return []*Unit{&Gorgon, &LizardSwordsmen, &Sprites, &Demon, &HeroGreyfairer, &HighElfSettlers, &KlackonSettlers,
        &ChaosSpawn, &WereWolf, &GuardianSpirit, &DemonLord, &Nagas, &FireElemental, &EarthElemental,
        &AirElemental, &PhantomBeast, &PhantomWarrior, &Zombie}
}

func applyClassicFix(unit *Unit) {
    for _, fix := range classicUnitFixes {
        if unit.LbxFile == fix.LbxFile && unit.Index == fix.Index && unit.Name == fix.Name {
            fix.Fix(unit)
        }
    }
    if unit.Race == data.RaceFantastic {
        if cost, ok := classicTableCosts[unit.Name]; ok {
            unit.TableCost = cost
        }
    }
}

// puts the original's numbers into the table of units (AllUnits, which saves read) and the
// named units; called once at the start when ClassicUnits is on
func ApplyClassicUnitData() {
    for i := range AllUnits {
        applyClassicFix(&AllUnits[i])
    }
    for _, unit := range classicNamedUnits() {
        applyClassicFix(unit)
    }
}

// the worth of a unit as the table has it: what the game reads for the worth of creatures
func (unit Unit) WorthCost() int {
    if unit.TableCost > 0 {
        return unit.TableCost
    }
    if unit.ProductionCost > 0 {
        return unit.ProductionCost
    }
    return unit.CastingCost
}

// Unit_Hit_Points: Charm of Life adds a quarter of the hits, at least 1
func classicCharmOfLife(hits int) int {
    return hits + max(1, hits / 4)
}

// ClassicCharmOfLife for the hero package
func ClassicCharmOfLife(hits int) int {
    return classicCharmOfLife(hits)
}

// Heal_Unit: a part of the whole of the hits, 1 in divisor, the rest as a chance; no least
func ClassicHealAmount(total int, divisor int) int {
    if divisor < 1 || total < 1 {
        return 0
    }
    heal := total / divisor
    if rand.N(divisor) + 1 <= total % divisor {
        heal += 1
    }
    return heal
}

// Calc_Unit_Level: Crusade raises the level of a normal unit that is not undead
func (unit *OverworldUnit) classicCrusade() bool {
    if unit.ExperienceInfo == nil || !unit.ExperienceInfo.Crusade() {
        return false
    }
    return !unit.IsUndead()
}
