package combat

// Units that fight across a corner close in on each other. Not in the original.
//
// Two tiles that touch with a side are 18 art pixels apart on the screen. Two that touch with a
// corner to the left and right of each other are 32 apart, and units that fight across such a
// corner swing at air (user, 2026-09-28). For the time of the attack both go toward each other
// until they are as far apart as units on tiles that touch with a side. They go there as the
// attack starts and back as it ends.
//
// Units that are near enough stay where they are: on tiles that touch with a side, and with a
// corner above and below each other, which are 16 apart.

import (
    "math"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: units stay on their tiles
const StrikeApproach = true
// units that fight are no further apart than this, in art pixels on the screen: as far as two
// tiles that touch with a side are
const approachDistance = 18.0
// seconds a unit takes to close in, and to go back
const approachTime = 0.18

// how far a unit goes toward another to fight it, across and down in art pixels. the units are at
// these tiles
func approachFor(x int, y int, otherX int, otherY int) (float64, float64) {
    field := MakeBattlefieldMatrix()
    fromX, fromY := field.Apply(float64(x), float64(y))
    toX, toY := field.Apply(float64(otherX), float64(otherY))

    towardX := toX - fromX
    towardY := toY - fromY
    distance := math.Hypot(towardX, towardY)
    if distance <= approachDistance {
        return 0, 0
    }

    // each of the two goes half of what is too far
    far := (distance - approachDistance) / 2
    return towardX / distance * far, towardY / distance * far
}

// how much of its way a unit has closed in, 0 to 1. ticks since the attack started, the ticks
// the attack lasts, and the ticks the way takes
func approachPart(since float64, lasts float64, way float64) float64 {
    if since < 0 || since >= lasts || lasts <= 0 {
        return 0
    }

    way = min(way, lasts / 2)
    if way <= 0 {
        return 1
    }

    switch {
        case since < way: return easeInOut(since / way)
        case since > lasts - way: return easeInOut((lasts - since) / way)
    }
    return 1
}

// the units of an attack close in on each other for as long as it lasts
func (combat *CombatScreen) startApproach(attacker *ArmyUnit, defender *ArmyUnit, lasts int) {
    for _, unit := range []*ArmyUnit{attacker, defender} {
        if unit != nil {
            unit.ApproachX = 0
            unit.ApproachY = 0
            unit.SwingLasts = lasts
        }
    }

    if !StrikeApproach || attacker == nil || defender == nil {
        return
    }

    attacker.ApproachX, attacker.ApproachY = approachFor(attacker.X, attacker.Y, defender.X, defender.Y)
    defender.ApproachX, defender.ApproachY = approachFor(defender.X, defender.Y, attacker.X, attacker.Y)
}

// how far a unit that fights is from its tile now, across and down in art pixels
func (combat *CombatScreen) strikeApproach(unit *ArmyUnit) (float64, float64) {
    if !StrikeApproach || (unit.ApproachX == 0 && unit.ApproachY == 0) || !(unit.Attacking || unit.Defending) {
        return 0, 0
    }

    since := 0.0
    if combat.Counter > unit.SwingStart {
        since = float64(combat.Counter - unit.SwingStart)
    }

    way := approachTime * float64(max(1, ebiten.TPS()))
    part := approachPart(since, float64(unit.SwingLasts), way)
    return unit.ApproachX * part, unit.ApproachY * part
}
