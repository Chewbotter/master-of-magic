package game

// Development: what the game notes into the journal of a run without a window (-sim-journal, see
// player/journal.go and sim.go of the main package): battles with the strength of both sides,
// cities taken and founded, events, spells that land, conquests. Nothing is noted when no journal
// is kept.

import (
    "fmt"

    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    "github.com/kazzmir/master-of-magic/game/magic/combat"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/maplib"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// the strength of a stack by Chewbot's measure (a tenth, as its orders count it) and its units
func simStackStrength(stack *playerlib.UnitStack) (int, int) {
    if stack == nil {
        return 0, 0
    }
    strength, count := 0, 0
    for _, unit := range stack.Units() {
        if unit.GetHealth() <= 0 {
            continue
        }
        strength += combat.ChewbotUnitStrength(unit, unit.VisibleFigures(), unit.GetHealth()) / 10
        count += 1
    }
    return strength, count
}

func simStateName(state combat.CombatState) string {
    switch state {
        case combat.CombatStateAttackerWin: return "attacker won"
        case combat.CombatStateDefenderWin: return "defender won"
        case combat.CombatStateAttackerFlee: return "attacker fled"
        case combat.CombatStateDefenderFlee: return "defender fled"
    }
    return "no result"
}

// a battle as it is about to start, to be noted with its result
type simBattle struct {
    attacker *playerlib.Player
    defender *playerlib.Player
    where string
    attackStrength int
    attackUnits int
    defendStrength int
    defendUnits int
    attackStack *playerlib.UnitStack
    defendStack *playerlib.UnitStack
    // the attacker's units by name, and how many of them went into the battle (doCombat leaves out
    // ships on land and walkers on water)
    attackNames string
    fought int
    // fought on a square of water (a sea battle)
    atSea bool
}

// the units of a stack by name and count, "2 Swordsmen, Trireme"
func simUnitNames(stack *playerlib.UnitStack) string {
    if stack == nil {
        return ""
    }
    var order []string
    counts := make(map[string]int)
    for _, unit := range stack.Units() {
        name := unit.GetName()
        if counts[name] == 0 {
            order = append(order, name)
        }
        counts[name] += 1
    }
    out := ""
    for i, name := range order {
        if i > 0 {
            out += ", "
        }
        if counts[name] > 1 {
            out += fmt.Sprintf("%v ", counts[name])
        }
        out += name
    }
    return out
}

func simBattleStart(attacker *playerlib.Player, attackStack *playerlib.UnitStack, defender *playerlib.Player, defendStack *playerlib.UnitStack, where string) *simBattle {
    if !playerlib.Noting() {
        return nil
    }
    battle := &simBattle{attacker: attacker, defender: defender, where: where, attackStack: attackStack, defendStack: defendStack, fought: -1}
    battle.attackNames = simUnitNames(attackStack)
    battle.attackStrength, battle.attackUnits = simStackStrength(attackStack)
    battle.defendStrength, battle.defendUnits = simStackStrength(defendStack)
    return battle
}

func (battle *simBattle) end(state combat.CombatState) {
    if battle == nil {
        return
    }
    _, attackLeft := simStackStrength(battle.attackStack)
    _, defendLeft := simStackStrength(battle.defendStack)
    defenderName := "monsters"
    if battle.defender != nil {
        defenderName = battle.defender.Wizard.Name
    }
    what := fmt.Sprintf("%v at %v against %v: %v", "battle", battle.where, defenderName, simStateName(state))
    why := fmt.Sprintf("strength %v (%v units) against %v (%v units); units left %v and %v; attackers %v", battle.attackStrength, battle.attackUnits, battle.defendStrength, battle.defendUnits, attackLeft, defendLeft, battle.attackNames)
    if battle.fought >= 0 && battle.fought < battle.attackUnits {
        why += fmt.Sprintf("; only %v of them in the battle", battle.fought)
    }
    if battle.atSea {
        why += "; at sea"
    }
    playerlib.Note(battle.attacker, "battle", what, why)
}

// where a battle is: a city, a lair, or the square
func (game *Game) simWhere(stack *playerlib.UnitStack) string {
    if stack == nil {
        return "?"
    }
    if city, _ := game.Model.FindCity(stack.X(), stack.Y(), stack.Plane()); city != nil {
        return "the city " + city.Name
    }
    return fmt.Sprintf("%v,%v", stack.X(), stack.Y())
}

func simEncounterName(encounter *maplib.ExtraEncounter) string {
    if encounter == nil {
        return "an encounter"
    }
    return fmt.Sprintf("a %v of %v guardians", encounter.Type.Name(), len(encounter.Units))
}

func simCityNote(player *playerlib.Player, kind string, city *citylib.City, why string) {
    if !playerlib.Noting() || city == nil {
        return
    }
    playerlib.Note(player, kind, fmt.Sprintf("%v (%v people) at %v,%v", city.Name, city.Citizens(), city.X, city.Y), why)
}

// the names of the random events for the journal
func randomEventName(kind RandomEventType) string {
    switch kind {
        case RandomEventBadMoon: return "Bad Moon"
        case RandomEventConjunctionChaos: return "Red Conjunction"
        case RandomEventConjunctionNature: return "Green Conjunction"
        case RandomEventConjunctionSorcery: return "Blue Conjunction"
        case RandomEventDepletion: return "Depletion"
        case RandomEventDiplomaticMarriage: return "Diplomatic Marriage"
        case RandomEventDisjunction: return "Disjunction"
        case RandomEventDonation: return "Donation"
        case RandomEventEarthquake: return "Earthquake"
        case RandomEventGift: return "Gift"
        case RandomEventGoodMoon: return "Good Moon"
        case RandomEventGreatMeteor: return "Great Meteor"
        case RandomEventManaShort: return "Mana Short"
        case RandomEventNewMinerals: return "New Minerals"
        case RandomEventPiracy: return "Piracy"
        case RandomEventPlague: return "Plague"
        case RandomEventPopulationBoom: return "Population Boom"
        case RandomEventRebellion: return "Rebellion"
    }
    return fmt.Sprintf("event %v", int(kind))
}

// a fight for a lair, node or tower: the guardians measured as Chewbot measures a stack
func simEncounterStart(player *playerlib.Player, stack *playerlib.UnitStack, encounter *maplib.ExtraEncounter, plane data.Plane) *simBattle {
    if !playerlib.Noting() || encounter == nil {
        return nil
    }
    battle := &simBattle{attacker: player, where: simEncounterName(encounter), attackStack: stack, fought: -1}
    battle.attackNames = simUnitNames(stack)
    battle.attackStrength, battle.attackUnits = simStackStrength(stack)
    for _, guardian := range encounter.Units {
        made := units.MakeOverworldUnit(guardian, 0, 0, plane)
        battle.defendStrength += combat.ChewbotUnitStrength(made, made.GetCount(), made.GetMaxHealth()) / 10
        battle.defendUnits += 1
    }
    return battle
}
