package game

// The limit of units in the world, as the original has it. Facts from the ReMoM project's
// reconstruction (MoX/src/MOX_DEF.h MAX_UNIT_COUNT; MoM/src/NEXTTURN.c Create_Unit,
// City_Apply_Production). The code is ours.
//
//   the world holds 1000 units at most; a unit is not made for a computer player (the neutral one
//     too) when there are more than 950, nor for anyone when there are more than 980
//   made anyway: guardians of a lair and units summoned in a battle (the original's R_Param 2000)
//   a city whose unit can not be made: a computer player's picks something else (AUTOBUILD), the
//     human's keeps what it builds
//   offers of mercenaries and heroes look at the same limit (Determine_Offer)
//
// The fork had no limit: test games of five computer wizards reached 1,407 units.

import (
    "log"

    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

// false: no limit (the fork's)
var ClassicUnitLimit = true

const (
    unitLimitComputers = 950
    unitLimitAll = 980
    unitLimitMost = 1000
)

// the units of every player
func (model *GameModel) UnitCount() int {
    count := 0
    for _, player := range model.Players {
        for range player.Units() {
            count += 1
        }
    }
    return count
}

// can a unit be made for the owner (Create_Unit without R_Param 2000)
func (game *Game) roomForUnit(owner *playerlib.Player) bool {
    if !ClassicUnitLimit {
        return true
    }
    count := game.Model.UnitCount()
    if count >= unitLimitMost || count > unitLimitAll {
        return false
    }
    if !owner.IsHuman() && count > unitLimitComputers {
        return false
    }
    return true
}

// a unit that could not be made, for the log and the journal of a run without a window
func (game *Game) noRoomForUnit(owner *playerlib.Player, what string) {
    log.Printf("No room for %v of %v: %v units in the world", what, owner.Wizard.Name, game.Model.UnitCount())
    playerlib.Note(owner, "unit limit", what, "the world holds too many units (the original's limit)")
}
