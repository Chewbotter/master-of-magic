package combat

// Where the units of both armies start, matched to the original game.
//
// Each army has 12 places in three rows of four. Units without a ranged attack fill the places
// first, front row first, then the units with one behind them. An army with more than five ranged
// units sends the ones with the strongest melee attack to the front until the front row is full.
// Defenders skip the places taken by the corners of a city wall, and outside of a city the place of
// the structure in the middle of the field.
// (ReMoM: MoM/src/Combat.c Deploy_Battle_Units, MoM/src/CMBTDEF.h DEPLOY_*. The code here is ours.)
//
// Armies of more than 12 units do not exist in the original. Ours get more places: rows twice as
// wide and two more rows behind, see largeArmy. An army of up to 12 units stands as in the original.

import (
    "image"
)

// rows of the original's grid, front row first. the first three are the original's
var deployDefenderRows = []int{8, 7, 6, 5, 4}
var deployAttackerRows = []int{14, 15, 16, 17, 18}
// the places of a row, in the order they are filled. the first four are the original's
var deployColumns = []int{12, 11, 13, 10, 14, 9, 15, 8}

// the original's army
const deployRows = 3
const deployRowWidth = 4
const deployPlaces = deployRows * deployRowWidth

// an army with more ranged units than this sends some to the front
const deployRangedLimit = 5
const deployRangedKept = 4

// the cells of the corners of a city wall that a front row reaches
const deployWallRow = 8
var deployWallColumns = map[int]bool{10: true, 13: true}

// the cells an army takes, in the order they are filled
func deployCells(team Team, units int, city bool) []image.Point {
    rows := deployDefenderRows
    if team == TeamAttacker {
        rows = deployAttackerRows
    }

    var out []image.Point

    if units <= deployPlaces {
        for _, row := range rows[:deployRows] {
            for _, column := range deployColumns[:deployRowWidth] {
                out = append(out, image.Pt(row, column))
            }
        }
        return out
    }

    // a large army. the defenders of a city first fill what lies inside of it, row by row, and
    // then the places to both sides. everybody else stands in wide rows
    if team == TeamDefender && city {
        for _, row := range rows {
            for _, column := range deployColumns[:deployRowWidth] {
                out = append(out, image.Pt(row, column))
            }
        }
        for _, row := range rows {
            for _, column := range deployColumns[deployRowWidth:] {
                out = append(out, image.Pt(row, column))
            }
        }
        return out
    }

    for _, row := range rows {
        for _, column := range deployColumns {
            out = append(out, image.Pt(row, column))
        }
    }

    return out
}

// the units in the order they take their places. frontRow is how many stand in the front row
func deployOrder(units []*ArmyUnit, frontRow int) []*ArmyUnit {
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
        for move := 0; move < moves && len(melee) < frontRow; move++ {
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

// AN ARMY THAT STANDS APART (user, 2026-09-30: "When starting a debug test army with 'all', let's
// separate each unit by one empty tile, like a checkerboard pattern"): the places are every second
// cell of rows behind the front row, the middle of the front row first, so no two units stand side
// by side. Only for armies that ask for it (Army.SpreadOut)

// the columns of the places of an army that stands apart, in the order they are filled
var spreadColumns = []int{12, 11, 13, 10, 14, 9, 15, 8, 16, 7, 17, 6, 18}
// how many rows it takes, front row first
const spreadRows = 6

// the cells of an army that stands apart, in the order they are filled
func spreadCells(team Team) []image.Point {
    front, back := deployDefenderRows[0], -1
    if team == TeamAttacker {
        front, back = deployAttackerRows[0], 1
    }

    var out []image.Point
    for step := range spreadRows {
        row := front + back * step
        for _, column := range spreadColumns {
            // the cells of one color of the checkerboard, the one of the middle of the front row
            if (row + column) % 2 == (front + spreadColumns[0]) % 2 {
                out = append(out, image.Pt(row, column))
            }
        }
    }
    return out
}

// puts the units on the original's places. returns the units that found no place
func deployUnits(units []*ArmyUnit, team Team, walled bool, city bool, spread bool, legalLocation LegalLocation, facing func(*ArmyUnit)) []*ArmyUnit {
    var left []*ArmyUnit

    cells := deployCells(team, len(units), city)
    frontRow := deployRowWidth
    if len(units) > deployPlaces {
        frontRow = len(deployColumns)
    }
    if spread {
        cells = spreadCells(team)
        frontRow = (len(spreadColumns) + 1) / 2
    }

    place := 0

    for _, unit := range deployOrder(units, frontRow) {
        placed := false

        for place < len(cells) && !placed {
            cell := cells[place]
            place += 1

            skip := false
            if team == TeamDefender {
                wallCorner := walled && cell.X == deployWallRow && deployWallColumns[cell.Y]
                structure := !city && cell.X == lairCellX && cell.Y == lairCellY
                skip = wallCorner || structure
            }

            x, y := CellToTile(cell.X, cell.Y)
            if skip || !legalLocation.IsLegalLocation(x, y) {
                continue
            }

            unit.X = x
            unit.Y = y
            facing(unit)
            placed = true
        }

        if !placed {
            left = append(left, unit)
        }
    }

    return left
}
