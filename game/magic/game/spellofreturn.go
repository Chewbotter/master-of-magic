package game

// Where a banished wizard's Spell of Return puts its new fortress (ReMoM MoM/src/Spells132.c
// Cast_Spell_Of_Return; MoM/src/AISPELL.c AITP_Move_Fortress). The human picks one of its cities and
// can not leave the choice; a computer player takes its city with the strongest garrison. Before,
// a computer player's Spell of Return found no city (its AI had no answer for the spell) and the
// wizard stayed banished for good. The code is ours.

import (
    "cmp"
    "slices"

    "github.com/kazzmir/master-of-magic/game/magic/combat"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

// the strength of a city's garrison as the original's AI counts it (settlers, engineers and ships
// count nothing)
func (game *Game) garrisonStrength(player *playerlib.Player, city *citylib.City) int {
    stack := player.FindStack(city.X, city.Y, city.Plane)
    if stack == nil {
        return 0
    }
    strength := 0
    for _, unit := range stack.Units() {
        if unit.HasAbility(data.AbilityCreateOutpost) || unit.HasAbility(data.AbilityConstruction) || unit.HasAbility(data.AbilityTransport) {
            continue
        }
        strength += combat.ChewbotUnitStrength(unit, unit.VisibleFigures(), unit.GetHealth()) / 10
    }
    return strength
}

// AITP_Move_Fortress for a returning wizard: its city with the strongest garrison, which must be
// stronger than the old fortress's; the old fortress is lost here, so any garrison at all (MY CALL:
// the original reads the city at the old place, which is another's now or gone). Cities in the order
// of plane, row and column, the first of equals
func (game *Game) returnCity(player *playerlib.Player) *citylib.City {
    var cities []*citylib.City
    for _, city := range player.CitiesInOrder() {
        cities = append(cities, city)
    }
    slices.SortFunc(cities, func(a *citylib.City, b *citylib.City) int {
        return cmp.Or(cmp.Compare(a.Plane, b.Plane), cmp.Compare(a.Y, b.Y), cmp.Compare(a.X, b.X))
    })
    var best *citylib.City
    strongest := 0
    for _, city := range cities {
        if strength := game.garrisonStrength(player, city); strength > strongest {
            strongest = strength
            best = city
        }
    }
    return best
}

// a computer player's Spell of Return: to the city of returnCity; with no such city the spell does
// nothing and the wizard starts it again (MY CALL: the original's spell fails, and a banished computer
// player is still played while it casts the spell)
func (game *Game) aiSpellOfReturn(event *GameEventSelectLocationForSpell, selected func(x int, y int)) {
    player := event.Player
    city := game.returnCity(player)
    if city == nil {
        player.CastSpellOfReturn()
        return
    }
    game.runAiCast(city.X, city.Y, city.Plane, func() {
        selected(city.X, city.Y)
    })
}
