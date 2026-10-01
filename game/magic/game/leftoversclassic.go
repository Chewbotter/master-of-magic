package game

// The last small rules of the original's turn: computer wizards dismissing weak units, the riders
// of a lost sea battle, the running events in the chancellor's scroll. Rules in words: the
// economy, movement and events pages of docs/mod.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/NEXTTURN.c:
// AI_Kill_Lame_Units, AI_Calculate_Average_Unit_Cost; Combat.c: Retreat_From_Combat, its ocean
// block; REPORT.c: Chancellor_Screen_Scroll_Draw__WIP). The code is ours.

import (
    "fmt"
    "strings"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    herolib "github.com/kazzmir/master-of-magic/game/magic/hero"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// the first turn computer wizards dismiss weak units, and how often the garrisons are looked at
const classicLameFirstTurn = 100
const classicLameRecount = 25

// AI_Calculate_Average_Unit_Cost leaves out settlers, spirits that meld, engineers and transports
func classicLameExempt(unit units.StackUnit) bool {
    return unit.HasAbility(data.AbilityCreateOutpost) || unit.HasAbility(data.AbilityMeld) ||
        unit.HasAbility(data.AbilityConstruction) || unit.HasAbility(data.AbilityTransport)
}

func classicUnitWorth(unit units.StackUnit) int {
    if hero, ok := unit.(*herolib.Hero); ok {
        // a hero's worth in the table is near its base fee (100 to 600); the fork keeps no table
        // cost for heroes (MY CALL)
        return 100 + hero.HeroType.GetRequiredFame() * 10
    }
    return unit.GetRawUnit().WorthCost()
}

// AI_Kill_Lame_Units: from turn 100 a computer wizard dismisses its units outside its cities that
// cost less than half its average; every 25 turns it counts the average anew and dismisses the
// first such unit of every garrison
func (game *Game) classicKillLameUnits(player *playerlib.Player) {
    turn := int(game.Model.TurnNumber)
    if !playerlib.ClassicEconomy || player.IsHuman() || player.IsNeutral() || player.Defeated || turn < classicLameFirstTurn {
        return
    }

    if turn % classicLameRecount == 0 {
        total, count := 0, 0
        for unit := range player.Units() {
            if classicLameExempt(unit) {
                continue
            }
            total += classicUnitWorth(unit)
            count += 1
        }
        player.AverageUnitCost = 0
        if count > 0 {
            player.AverageUnitCost = total / count
        }

        for _, city := range player.Cities {
            for _, unit := range player.GetUnits(city.X, city.Y, city.Plane) {
                if classicUnitWorth(unit) < player.AverageUnitCost / 2 {
                    playerlib.Note(player, "dismiss", unit.GetName() + " in " + city.Name, fmt.Sprintf("worth %v, under half the average %v (AI_Kill_Lame_Units)", classicUnitWorth(unit), player.AverageUnitCost))
                    dismissUnit(player, unit)
                    break
                }
            }
        }
    }

    if player.AverageUnitCost <= 0 {
        return
    }
    var lame []units.StackUnit
    for unit := range player.Units() {
        if unit.IsHero() || classicLameExempt(unit) || player.FindCity(unit.GetX(), unit.GetY(), unit.GetPlane()) != nil {
            continue
        }
        if classicUnitWorth(unit) < player.AverageUnitCost / 2 {
            lame = append(lame, unit)
        }
    }
    for _, unit := range lame {
        playerlib.Note(player, "dismiss", unit.GetName(), fmt.Sprintf("worth %v, under half the average %v, outside its cities (AI_Kill_Lame_Units)", classicUnitWorth(unit), player.AverageUnitCost))
        dismissUnit(player, unit)
    }
}

// Retreat_From_Combat, the ocean, for the side that lost: with no wind walker left, the seats of
// the transports that are left (ships, a Floating Island) carry the riders that can not fly or
// swim, in their order; heroes take no seat while there are seats; those beyond the seats drown,
// with no seats every rider
func classicDrowned(stack []units.StackUnit) map[units.StackUnit]bool {
    out := make(map[units.StackUnit]bool)
    seats := 0
    for _, unit := range stack {
        if unit.GetHealth() <= 0 {
            continue
        }
        if unit.HasAbility(data.AbilityWindWalking) {
            return out
        }
        seats += int(unit.GetAbilityValue(data.AbilityTransport))
    }
    anySeats := seats > 0
    for _, unit := range stack {
        if unit.GetHealth() <= 0 || !unit.IsLandWalker() {
            continue
        }
        if !anySeats {
            out[unit] = true
            continue
        }
        if unit.IsHero() {
            continue
        }
        if seats > 0 {
            seats -= 1
        } else {
            out[unit] = true
        }
    }
    return out
}

// a transport of the stack is left (a ship, or a Floating Island, which does not sail)
func transportLeft(stack []units.StackUnit) bool {
    for _, unit := range stack {
        if unit.GetHealth() > 0 && unit.GetAbilityValue(data.AbilityTransport) > 0 {
            return true
        }
    }
    return false
}

// Chancellor_Screen_Scroll_Draw: the running events the chancellor lists first
func (game *Game) classicRandomEventLines() *GameEventScroll {
    human := game.Model.GetHumanPlayer()
    var lines []string
    for _, kind := range []RandomEventType{RandomEventPlague, RandomEventPopulationBoom} {
        for _, event := range game.Model.RandomEvents {
            if event.Type != kind || event.TargetCity == nil || human == nil || human.FindCity(event.TargetCity.X, event.TargetCity.Y, event.TargetCity.Plane) == nil {
                continue
            }
            if kind == RandomEventPlague {
                lines = append(lines, "Plague in " + event.TargetCity.Name)
            } else {
                lines = append(lines, "Population Boom in " + event.TargetCity.Name)
            }
        }
    }
    names := []struct{Kind RandomEventType; Name string}{
        {RandomEventGoodMoon, "Good Moon"}, {RandomEventBadMoon, "Bad Moon"},
        {RandomEventConjunctionChaos, "Red Conjunction"}, {RandomEventConjunctionNature, "Green Conjunction"},
        {RandomEventConjunctionSorcery, "Blue Conjunction"}, {RandomEventManaShort, "Mana Short"},
    }
    for _, name := range names {
        for _, event := range game.Model.RandomEvents {
            if event.Type == name.Kind {
                lines = append(lines, name.Name)
                break
            }
        }
    }
    if len(lines) == 0 {
        return nil
    }
    return &GameEventScroll{Title: "RANDOM EVENTS", Text: strings.Join(lines, "\n")}
}
