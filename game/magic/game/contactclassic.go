package game

// When two wizards meet, as the original has it. Facts from the ReMoM project's reconstruction
// (MoM/src/Explore.c: Update_Scouted_And_Contacted, Contact_Other_Player; MoM/src/NEXTTURN.c:
// All_AI_Players_Contacted). The code is ours.
//
//   the human meets a wizard when a city of it or a stack with a unit that is not invisible is on a
//     square the human sees now (the original's scouting: what the units and cities of the
//     current player see, and that is always the human)
//   a wizard with Nature's Awareness meets every wizard that has a unit that is not invisible
//   computer wizards never meet each other by sight (kept for ComputersMeetBySight false; the user
//     took it out, see below)
//
// The fork's own rule, every wizard meeting every wizard it sees, is kept for ClassicContact false.

import (
    "github.com/kazzmir/master-of-magic/game/magic/data"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

// false: every wizard meets every wizard it can see (the fork's rule)
var ClassicContact = true

// computer wizards meet each other by what they see, as the human meets them; the human is still
// met only by what the human sees (user 2026-10-01:
// "let's ignore the Nature's Awareness rule for AI diplomacy as it seems like strictly a
// downgrade"). The original's computer wizards met only through Nature's Awareness, so they never
// made treaties or wars among themselves in an all-AI game. False: the original's rule
var ComputersMeetBySight = true

// a stack with a unit that can be seen
func stackSeen(stack *playerlib.UnitStack) bool {
    for _, unit := range stack.Units() {
        if !unit.IsInvisible() {
            return true
        }
    }
    return false
}

// does the observer meet the other by the original's rules
func classicMeets(observer *playerlib.Player, other *playerlib.Player) bool {
    if observer.GlobalEnchantments.Contains(data.EnchantmentNatureAwareness) {
        for _, stack := range other.Stacks {
            if stackSeen(stack) {
                return true
            }
        }
    }
    if !observer.IsHuman() && (!ComputersMeetBySight || other.IsHuman()) {
        // the human is met by what the human sees, as in the original
        return false
    }
    for _, city := range other.GetCities() {
        if observer.IsVisible(city.X, city.Y, city.Plane) {
            return true
        }
    }
    for _, stack := range other.Stacks {
        if stackSeen(stack) && observer.IsVisible(stack.X(), stack.Y(), stack.Plane()) {
            return true
        }
    }
    return false
}
