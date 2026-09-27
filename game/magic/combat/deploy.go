package combat

// Where the units of both armies start, matched to the original game.
//
// Each army has 12 places in three rows of four. Units without a ranged attack fill the places
// first, front row first, then the units with one behind them. An army with more than five ranged
// units sends the ones with the strongest melee attack to the front until the front row is full.
// Defenders skip the places taken by the corners of a city wall, and outside of a city the place of
// the structure in the middle of the field.
// (ReMoM: MoM/src/Combat.c Deploy_Battle_Units, MoM/src/CMBTDEF.h DEPLOY_*. The code here is ours.)

import (
    "image"
)

// rows of the original's grid, front row first
var deployDefenderRows = []int{8, 7, 6}
var deployAttackerRows = []int{14, 15, 16}
// the places of a row, in the order they are filled
var deployColumns = []int{12, 11, 13, 10}

const deployPlaces = 12
// an army with more ranged units than this sends some to the front
const deployRangedLimit = 5
const deployRangedKept = 4
const deployFrontRow = 4

// the places taken by the corners of a city wall
var deployWallPlaces = map[int]bool{2: true, 3: true}
// the place of the structure in the middle of the field
const deployStructurePlace = 9

// the tile of a place
func deployTile(team Team, place int) image.Point {
    rows := deployDefenderRows
    if team == TeamAttacker {
        rows = deployAttackerRows
    }

    x, y := CellToTile(rows[place / len(deployColumns)], deployColumns[place % len(deployColumns)])
    return image.Pt(x, y)
}

// the units in the order they take their places
func deployOrder(units []*ArmyUnit) []*ArmyUnit {
    var melee []*ArmyUnit
    var ranged []*ArmyUnit

    for _, unit := range units {
        if unit.Unit.GetRangedAttacks() > 0 && unit.Unit.GetRangedAttackPower() > 0 {
            ranged = append(ranged, unit)
        } else {
            melee = append(melee, unit)
        }
    }

    if len(ranged) > deployRangedLimit {
        moves := len(ranged) - deployRangedKept
        for move := 0; move < moves && len(melee) < deployFrontRow; move++ {
            strongest := 0
            for index, unit := range ranged {
                if unit.Unit.GetMeleeAttackPower() > ranged[strongest].Unit.GetMeleeAttackPower() {
                    strongest = index
                }
            }

            melee = append(melee, ranged[strongest])
            ranged = append(ranged[:strongest], ranged[strongest + 1:]...)
        }
    }

    return append(melee, ranged...)
}

// puts the units on the original's places. returns the units that found no place
func deployUnits(units []*ArmyUnit, team Team, walled bool, city bool, legalLocation LegalLocation, facing func(*ArmyUnit)) []*ArmyUnit {
    var left []*ArmyUnit
    place := 0

    for _, unit := range deployOrder(units) {
        placed := false

        for place < deployPlaces && !placed {
            skip := false
            if team == TeamDefender {
                skip = (walled && deployWallPlaces[place]) || (!city && place == deployStructurePlace)
            }

            tile := deployTile(team, place)
            place += 1

            if skip || !legalLocation.IsLegalLocation(tile.X, tile.Y) {
                continue
            }

            unit.X = tile.X
            unit.Y = tile.Y
            facing(unit)
            placed = true
        }

        if !placed {
            left = append(left, unit)
        }
    }

    return left
}
