package game

// dev: a line for every battle no screen shows (-capture-battle-log): both sides before it, their
// power as the fork's quick resolution counts it and their strength as Chewbot counts it, how it
// ended, after how many turns, and what was left

import (
    "fmt"
    "log"
    "strings"

    "github.com/kazzmir/master-of-magic/game/magic/combat"
)

var BattleLog = false

func battleSide(army *combat.Army) (string, int, int) {
    counts := make(map[string]int)
    var names []string
    strength := 0
    for _, unit := range army.GetUnits() {
        name := unit.Unit.GetName()
        if counts[name] == 0 {
            names = append(names, name)
        }
        counts[name] += 1
        strength += combat.ChewbotUnitStrength(unit.Unit, unit.Unit.VisibleFigures(), unit.Unit.GetHealth()) / 10
    }
    var parts []string
    for _, name := range names {
        parts = append(parts, fmt.Sprintf("%v %v", counts[name], name))
    }
    return strings.Join(parts, ", "), army.GetPower(), strength
}

// what a battle starts with
type battleBefore struct {
    Line string
    AttackStrength int
    DefendStrength int
}

func battleStart(attackers *combat.Army, defenders *combat.Army, zone combat.ZoneType) battleBefore {
    if !BattleLog {
        return battleBefore{}
    }
    attackUnits, attackPower, attackStrength := battleSide(attackers)
    defendUnits, defendPower, defendStrength := battleSide(defenders)
    where := "field"
    switch {
        case zone.City != nil: where = "city"
        case zone.Encounter != combat.ZoneNone: where = "lair"
    }
    return battleBefore{
        Line: fmt.Sprintf("%v: %v (power %v, strength %v) against %v (power %v, strength %v), strength ratio %.2f", where,
            attackUnits, attackPower, attackStrength, defendUnits, defendPower, defendStrength, float64(attackStrength) / float64(max(defendStrength, 1))),
        AttackStrength: attackStrength,
        DefendStrength: defendStrength,
    }
}

func battleEnd(before battleBefore, state combat.CombatState, model *combat.CombatModel, quick bool) {
    if !BattleLog {
        return
    }
    result := "defender wins"
    switch state {
        case combat.CombatStateAttackerWin: result = "attacker wins"
        case combat.CombatStateAttackerFlee: result = "attacker flees"
        case combat.CombatStateDefenderFlee: result = "defender flees"
    }
    how := "fought"
    if quick {
        how = "quick"
    } else if model.CurrentTurn >= combat.MAX_TURNS {
        result += " (out of turns)"
    }
    left := func(army *combat.Army) int {
        count := 0
        for _, unit := range army.GetUnits() {
            if unit.GetHealth() > 0 {
                count += 1
            }
        }
        return count
    }
    log.Printf("battle %v: %v: %v after %v turns, left %v attackers, %v defenders", how, before.Line, result, model.CurrentTurn, left(model.AttackingArmy), left(model.DefendingArmy))
}
