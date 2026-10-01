package relations

// The original game's diplomacy between wizards: how their relations change, treaties, war and
// peace. Facts from the ReMoM project's reconstruction (MoM/src/DIPLOMAC.c: Change_Relations,
// Start_Treaty, Declare_Peace, Declare_War, Break_Treaties, Adjust_Diplomat_Modifiers,
// G_DIPL_PickSides; MoX/src/MOM_DAT.c: TBL_AI_PRS_IDK_Mod, TBL_AI_PRS_War_Mod); the code is ours.
// The rules in words: docs/mod/diplomacy.md.
//
// The data of a pair is player.Relationship: a.PlayerRelations[b] is a's view of b (the original's
// _players[a].Dipl.X[b]). The visible relation is kept the same both ways.

import (
    "math/rand/v2"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

// what happened, for what a wizard says about it (the original's dipact_* and diplomatic orders)
const (
    ActionNone = 0
    // goodwill of a treaty, Aura of Majesty
    ActionGoodwill = 0
    ActionComputerGoodwill = 1
    ActionTreatyBroken = 5
    ActionConquest = 7
    ActionBattle = 8
    ActionCityAttacked = 9
    ActionSpellOfMastery = 10
    ActionEnchantment = 12
    ActionCityCurse = 13
    ActionOverextension = 14
    // 15 to 20: the greeting of a wizard of each personality
    ActionGreeting = 15
    // a treaty broken because the other is stronger
    ActionSuperiorityBreak = 30
    // war declared, relations bad
    ActionWarDeclared = 39
    // war declared by a chaotic wizard, for no reason
    ActionChaoticWar = 40
    // war declared, relations good (the other is stronger)
    ActionSuperiorityWar = 41
    // proposals to the human
    ActionProposePact = 45
    ActionProposeAlliance = 46
    ActionProposeWarOn = 47
    ActionProposeExchange = 48
    ActionProposePeace = 49
    // units near a city
    ActionUnitsNearCity = 73
)

// the personality's bonus in every judgment of treaties (TBL_AI_PRS_IDK_Mod)
var personalityTreatyBonus = map[playerlib.Personality]int{
    playerlib.PersonalityManiacal: 0,
    playerlib.PersonalityRuthless: 10,
    playerlib.PersonalityAggressive: 20,
    playerlib.PersonalityChaotic: 30,
    playerlib.PersonalityLawful: 40,
    playerlib.PersonalityPeaceful: 50,
}

// the personality's bonus in the chance of hostility (TBL_AI_PRS_War_Mod)
// TBL_AI_PRS_War_Mod of a personality, for other parts of the AI
func WarBonus(personality playerlib.Personality) int {
    return personalityWarBonus[personality]
}

var personalityWarBonus = map[playerlib.Personality]int{
    playerlib.PersonalityManiacal: 40,
    playerlib.PersonalityRuthless: 20,
    playerlib.PersonalityAggressive: 10,
    playerlib.PersonalityChaotic: 10,
    playerlib.PersonalityLawful: -40,
    playerlib.PersonalityPeaceful: -20,
}

// The original's own mistakes, kept where they show in play
const (
    // NPC_To_NPC_Treaty_Negotiations: an alliance that passes the test for a treaty again is made a
    // wizard pact
    quirkAllianceBecomesPact = true
    // Declare_Peace adds 40 to the relation twice and then overwrites it with the other side's:
    // peace brings no better relations
    quirkPeaceNoGain = true
)

// every roll: 1 to n (the original's Random); tests put a fixed one here
var roll = func(n int) int {
    if n <= 0 {
        return 1
    }
    return 1 + rand.IntN(n)
}

// a roll of 1 to n for the game's side of the rules
func Roll(n int) int {
    return roll(n)
}

// development: every roll gives this (at most n); returns the function that undoes it
func FixedRolls(value int) func() {
    old := roll
    roll = func(n int) int {
        return max(1, min(value, max(n, 1)))
    }
    return func() {
        roll = old
    }
}

// the world the rules act in. The game fills it in (game/relations.go)
type Rules struct {
    // every player in the game's order, the human first; the rules skip the neutral player
    Players []*playerlib.Player
    Turn int
    // 0 intro to 4 impossible
    Difficulty int
    // 0 small to 2 large
    LandSize int

    // the production cost of the units of a wizard and of another on the landmass and plane of that
    // other's fortress; false when it has none (Invader_Army_Strength_Comparison)
    FortressLandCosts func(invader *playerlib.Player, owner *playerlib.Player) (int, int, bool)
    // a unit of a wizard, no settler or engineer, within 2 squares of a city of another: the city
    UnitNearCity func(cityOwner *playerlib.Player, unitOwner *playerlib.Player) (string, bool)
    // the city enchantments each of two wizards has on the other's cities end
    CancelCityEnchantments func(a *playerlib.Player, b *playerlib.Player)
    // the astrologer's army strength of a wizard: the upkeep of its units
    ArmyStrength func(player *playerlib.Player) int
    // what a computer wizard says to the human: called for every order of this turn (the screens)
    Tell func(message Message)
}

// what a computer wizard says to the human at the end of a turn (Npc_Diplomacy_Screen)
type Message struct {
    From *playerlib.Player
    Action int
    Strength int
    Spell string
    City string
    OfferGold int
    OfferSpell string
    ExchangeSpells []string
    WantSpell string
    // the treaty between the two when it was said, and the one broken last (for the words)
    Treaty data.TreatyType
    BrokenTreaty data.TreatyType
    // the size of the city named, filled in by the game
    CitySize string
}

// a wizard, not the neutral player
func isWizard(player *playerlib.Player) bool {
    return player != nil && !player.IsNeutral()
}

// a wizard still in the game (the original's active fortress)
func isActive(player *playerlib.Player) bool {
    return isWizard(player) && !player.Defeated && !player.Banished
}

// a's view of b, nil before they met
func view(a *playerlib.Player, b *playerlib.Player) *playerlib.Relationship {
    if a == nil || b == nil || a == b {
        return nil
    }
    return a.PlayerRelations[b]
}

func treaty(a *playerlib.Player, b *playerlib.Player) data.TreatyType {
    if relation := view(a, b); relation != nil {
        return relation.Treaty
    }
    return data.TreatyNone
}

func clamp(value int, low int, high int) int {
    return max(low, min(high, value))
}

func abs(value int) int {
    if value < 0 {
        return -value
    }
    return value
}

// the countdown until a wizard looks at its hostility again: 16 to 25 turns
func resetHostilityCountdown(player *playerlib.Player) {
    player.HostilityCountdown = 15 + roll(10)
}

// the visible relation of a is b's too
func mirrorVisible(a *playerlib.Player, b *playerlib.Player) {
    if from, to := view(a, b), view(b, a); from != nil && to != nil {
        to.VisibleRelation = from.VisibleRelation
    }
}

func (rules *Rules) wizards() []*playerlib.Player {
    var out []*playerlib.Player
    for _, player := range rules.Players {
        if isWizard(player) {
            out = append(out, player)
        }
    }
    return out
}

func (rules *Rules) human() *playerlib.Player {
    for _, player := range rules.Players {
        if player.IsHuman() {
            return player
        }
    }
    return nil
}

// Change_Relations: what the attacker did changes the defender's relation to it. Only a computer
// wizard's feelings change; the human's follow them (the visible relation is the same both ways).
// The amount is doubled when it goes against the relation, and smaller when it goes with it;
// lawful wizards take offence twice as hard, maniacal ones half again and are pleased less; a
// charismatic attacker counts double for good and half for bad. At war, harm changes nothing but
// the will for peace. Between computer wizards after turn 200, good counts double and bad less on
// harder levels. Without an alliance the relation is at most 65, at war at most -25. What the human
// did is remembered for the turn (the grievance) when it is the strongest of the turn
func (rules *Rules) ChangeRelations(value int, attacker *playerlib.Player, defender *playerlib.Player, kind int, city string, spell string) {
    if !isWizard(attacker) || !isWizard(defender) || attacker == defender || value == 0 || defender.IsHuman() {
        return
    }
    attackerView, defenderView := view(attacker, defender), view(defender, attacker)
    if attackerView == nil || defenderView == nil {
        return
    }

    // a war declared or a treaty broken for superiority: only remembered for the message
    if value == -10000 {
        if attacker.IsHuman() && kind != ActionGoodwill {
            attackerView.ActionStrength = 10000
            attackerView.Action = kind
            attackerView.ActionSpell = spell
            attackerView.ActionCity = city
        }
        return
    }

    relation := defenderView.VisibleRelation
    if value > 0 {
        if relation < 0 {
            value *= 2
            if relation + value > 10 {
                value = 10 - relation
            }
        } else {
            value /= max(relation / 25 + 1, 1)
        }
    } else {
        if relation > 0 {
            value *= 2
        } else {
            value /= max(relation / -25 + 1, 1)
        }
    }

    switch defender.Personality {
        case playerlib.PersonalityLawful:
            if value < 0 {
                value *= 2
            }
        case playerlib.PersonalityManiacal:
            if value < 0 {
                value = value * 3 / 2
            } else {
                value = value * 3 / 4
            }
    }
    beforeCharisma := value

    if attacker.Wizard.RetortEnabled(data.RetortCharismatic) {
        if value > 0 {
            value *= 2
        } else {
            value /= 2
        }
    }

    if defenderView.Treaty == data.TreatyWar && kind <= ActionOverextension && value < 0 {
        if attacker.IsHuman() {
            defenderView.PeaceInterest += value / 4
            attackerView.PeaceInterest -= value / 4
        }
        value = 0
    }

    if !attacker.IsHuman() && !defender.IsHuman() && rules.Turn > 200 {
        if value > 0 {
            value *= 2
        } else if value < 0 {
            value /= rules.Difficulty / 2 + 1
        }
    }

    if value != 0 {
        defenderView.VisibleRelation = clamp(defenderView.VisibleRelation + value, -100, 100)
        defenderView.TreatyInterest += value
        defenderView.TradeInterest += value
        attackerView.TreatyInterest = defenderView.TreatyInterest
        attackerView.TradeInterest = defenderView.TradeInterest
        if beforeCharisma < 0 {
            attackerView.PeaceInterest -= beforeCharisma / 4
            defenderView.PeaceInterest += beforeCharisma / 4
        } else {
            defenderView.PeaceInterest += beforeCharisma
        }
        if attacker.IsHuman() && kind != ActionGoodwill && abs(value) > abs(attackerView.ActionStrength) {
            attackerView.ActionStrength = value
            attackerView.Action = kind
            attackerView.ActionSpell = spell
            attackerView.ActionCity = city
        }
    }

    if defenderView.Treaty >= data.TreatyWar {
        defenderView.VisibleRelation = min(defenderView.VisibleRelation, -25)
    }
    if defenderView.Treaty != data.TreatyAlliance {
        defenderView.VisibleRelation = min(defenderView.VisibleRelation, 65)
    }
    attackerView.VisibleRelation = defenderView.VisibleRelation

    switch defenderView.Treaty {
        case data.TreatyWar: defenderView.Hostility = 3
        case data.TreatyAlliance: defenderView.Hostility = 0
    }
    resetHostilityCountdown(defender)
}

// Start_Treaty: a treaty both ways, and no hostility
func (rules *Rules) StartTreaty(a *playerlib.Player, b *playerlib.Player, kind data.TreatyType) {
    aView, bView := view(a, b), view(b, a)
    if aView == nil || bView == nil {
        return
    }
    aView.Treaty = kind
    bView.Treaty = kind
    aView.Hostility = 0
    bView.Hostility = 0
}

// Declare_Peace: the war ends; the human keeps the peace for 8 to 13 turns (the original counts it
// on the human's side only), city curses of both on the other's cities end
func (rules *Rules) DeclarePeace(a *playerlib.Player, b *playerlib.Player) {
    aView, bView := view(a, b), view(b, a)
    if aView == nil || bView == nil {
        return
    }
    rules.StartTreaty(a, b, data.TreatyNone)
    if !quirkPeaceNoGain {
        aView.VisibleRelation = min(aView.VisibleRelation + 80, 100)
        bView.VisibleRelation = aView.VisibleRelation
    }
    if a.IsHuman() {
        aView.PeaceCounter = 7 + roll(6)
    }
    if b.IsHuman() {
        bView.PeaceCounter = 7 + roll(6)
    }
    if rules.CancelCityEnchantments != nil {
        rules.CancelCityEnchantments(a, b)
    }
    resetHostilityCountdown(a)
    resetHostilityCountdown(b)
}

// Declare_War: the attacker breaks any treaty, both like each other 5 less for good, the relation
// drops to -76 to -100, the patience for proposals is gone, both are at war and hostile
func (rules *Rules) DeclareWar(attacker *playerlib.Player, defender *playerlib.Player) {
    attackerView, defenderView := view(attacker, defender), view(defender, attacker)
    if attackerView == nil || defenderView == nil || !isWizard(attacker) || !isWizard(defender) {
        return
    }
    if attacker.IsHuman() && attackerView.ContactStage == 0 {
        attackerView.ContactStage = 1
    }
    if defender.IsHuman() && defenderView.ContactStage == 0 {
        defenderView.ContactStage = 1
    }
    if attackerView.Treaty >= data.TreatyWar {
        return
    }

    rules.BreakTreaties(attacker, defender)
    attackerView.StartingRelation -= 5
    defenderView.StartingRelation -= 5
    attackerView.VisibleRelation = -75 - roll(25)
    defenderView.VisibleRelation = attackerView.VisibleRelation
    // the original sets the defender's relation where it means its treaty; both are at war
    attackerView.Treaty = data.TreatyWar
    defenderView.Treaty = data.TreatyWar
    for _, relation := range []*playerlib.Relationship{attackerView, defenderView} {
        relation.TreatyInterest = -200
        relation.TradeInterest = -200
        relation.PeaceInterest = -130
        relation.Hostility = 3
    }
    resetHostilityCountdown(attacker)
    resetHostilityCountdown(defender)
}

// Break_Treaties: the breaker ends a pact or alliance. The size is 10 (20 for an alliance, twice
// that when the breaker is lawful). As the original has it (quirkBreakTreatiesBackwards): the
// BREAKER's hidden relation to the victim goes up by it, every other wizard's to the VICTIM drops by
// 5, after an alliance the breaker's lasting relation goes up by it, and the lasting relation is
// copied to the victim's view for any treaty. (Meant: the breaker pays; see improvements.md.) The
// relation drops by 1 to 20, the patience for proposals is gone
func (rules *Rules) BreakTreaties(breaker *playerlib.Player, victim *playerlib.Player) {
    breakerView, victimView := view(breaker, victim), view(victim, breaker)
    if breakerView == nil || victimView == nil || breakerView.Treaty >= data.TreatyWar {
        return
    }
    if quirkBreakTreatiesBackwards {
        rules.breakTreatiesBackwards(breaker, victim, breakerView, victimView)
        return
    }

    penalty := 0
    switch breakerView.Treaty {
        case data.TreatyPact: penalty = -10
        case data.TreatyAlliance: penalty = -20
    }
    if breaker.Personality == playerlib.PersonalityLawful {
        penalty *= 2
    }
    victimView.HiddenRelation += penalty
    for _, other := range rules.wizards() {
        if other == breaker || other == victim {
            continue
        }
        if otherView := view(other, breaker); otherView != nil {
            otherView.HiddenRelation -= 5
        }
    }
    if breakerView.Treaty == data.TreatyAlliance {
        victimView.StartingRelation = max(victimView.StartingRelation + penalty, -100)
        breakerView.StartingRelation = victimView.StartingRelation
    }
    if penalty != 0 {
        breakerView.BrokenTreaty = breakerView.Treaty
        victimView.BrokenTreaty = breakerView.Treaty
        breakerView.VisibleRelation -= roll(20)
    }
    breakerView.Treaty = data.TreatyNone
    victimView.Treaty = data.TreatyNone
    breakerView.VisibleRelation = max(breakerView.VisibleRelation, -100)
    victimView.VisibleRelation = breakerView.VisibleRelation
    for _, relation := range []*playerlib.Relationship{breakerView, victimView} {
        relation.TreatyInterest = -200
        relation.TradeInterest = -200
        relation.PeaceInterest = -200
    }
    resetHostilityCountdown(breaker)
    resetHostilityCountdown(victim)
}

// Break_Treaties: the signs and sides of the original's program (the reconstruction marks them
// as its bugs)
const quirkBreakTreatiesBackwards = true

func (rules *Rules) breakTreatiesBackwards(breaker *playerlib.Player, victim *playerlib.Player, breakerView *playerlib.Relationship, victimView *playerlib.Relationship) {
    size := 0
    switch breakerView.Treaty {
        case data.TreatyPact: size = 10
        case data.TreatyAlliance: size = 20
    }
    if breaker.Personality == playerlib.PersonalityLawful {
        size *= 2
    }
    // the original subtracts a negative number
    breakerView.HiddenRelation += size
    for _, other := range rules.wizards() {
        if other == breaker || other == victim {
            continue
        }
        if otherView := view(other, victim); otherView != nil {
            otherView.HiddenRelation -= 5
        }
    }
    if breakerView.Treaty == data.TreatyAlliance {
        breakerView.StartingRelation += size
    }
    breakerView.StartingRelation = max(breakerView.StartingRelation, -100)
    victimView.StartingRelation = breakerView.StartingRelation
    if size != 0 {
        breakerView.BrokenTreaty = breakerView.Treaty
        victimView.BrokenTreaty = breakerView.Treaty
        breakerView.VisibleRelation -= roll(20)
    }
    breakerView.Treaty = data.TreatyNone
    victimView.Treaty = data.TreatyNone
    breakerView.VisibleRelation = max(breakerView.VisibleRelation, -100)
    victimView.VisibleRelation = breakerView.VisibleRelation
    for _, relation := range []*playerlib.Relationship{breakerView, victimView} {
        relation.TreatyInterest = -200
        relation.TradeInterest = -200
        relation.PeaceInterest = -200
    }
    resetHostilityCountdown(breaker)
    resetHostilityCountdown(victim)
}

// Subversion: every other wizard's relation to the target drops by 25, and any peace ends. (The
// original lowers it and then overwrites it with the target's own, so only the peace ends; the
// spell is meant to do both)
func (rules *Rules) Subversion(caster *playerlib.Player, target *playerlib.Player) {
    if ClassicSubversion {
        // Apply_Subversion: every computer wizard but the target (the caster too) that has met it
        // drops its view of the target 25, then takes the target's view of it; the peace ends
        for _, other := range rules.wizards() {
            if other == target || other.IsHuman() {
                continue
            }
            otherView, targetView := view(other, target), view(target, other)
            if otherView == nil || targetView == nil {
                continue
            }
            otherView.VisibleRelation = max(otherView.VisibleRelation - 25, -100)
            otherView.VisibleRelation = targetView.VisibleRelation
            otherView.PeaceCounter = 0
            targetView.PeaceCounter = 0
        }
        return
    }
    for _, other := range rules.wizards() {
        if other == caster || other == target {
            continue
        }
        otherView, targetView := view(other, target), view(target, other)
        if otherView == nil || targetView == nil {
            continue
        }
        otherView.VisibleRelation = max(otherView.VisibleRelation - 25, -100)
        targetView.VisibleRelation = otherView.VisibleRelation
        otherView.PeaceCounter = 0
        targetView.PeaceCounter = 0
    }
}

// Resolve_Wizard_Conquest: every other wizard likes the conqueror 40 less; nobody has a treaty with
// the conquered any more
func (rules *Rules) Conquered(winner *playerlib.Player, loser *playerlib.Player) {
    for _, other := range rules.wizards() {
        if other == winner || other == loser {
            continue
        }
        rules.ChangeRelations(-40, winner, other, ActionConquest, "", "")
    }
    for _, other := range rules.wizards() {
        if relation := view(other, loser); relation != nil {
            relation.Treaty = data.TreatyNone
        }
        if relation := view(loser, other); relation != nil {
            relation.Treaty = data.TreatyNone
        }
    }
}

// Banish_Wizard: every other wizard likes the banisher 20 less
func (rules *Rules) Banished(winner *playerlib.Player, loser *playerlib.Player) {
    for _, other := range rules.wizards() {
        if other == winner || other == loser {
            continue
        }
        rules.ChangeRelations(-20, winner, other, ActionConquest, "", "")
    }
}

// Adjust_Diplomat_Modifiers: every talk wears the patience down by 10
func adjustModifiers(relation *playerlib.Relationship, times int) {
    if relation == nil {
        return
    }
    relation.TreatyInterest -= 10 * times
    relation.TradeInterest -= 10 * times
    relation.PeaceInterest -= 10 * times
}

// G_DIPL_PickSides: a computer wizard allied with two that are at war breaks with both
func (rules *Rules) pickSides() {
    wizards := rules.computerWizards()
    for i := 0; i < len(wizards); i++ {
        for j := i + 1; j < len(wizards); j++ {
            for k := i + 1; k < len(wizards); k++ {
                if treaty(wizards[j], wizards[k]) == data.TreatyWar && treaty(wizards[i], wizards[j]) == data.TreatyAlliance && treaty(wizards[i], wizards[k]) == data.TreatyAlliance {
                    rules.BreakTreaties(wizards[i], wizards[j])
                    rules.BreakTreaties(wizards[i], wizards[k])
                }
            }
        }
    }
}

func (rules *Rules) computerWizards() []*playerlib.Player {
    var out []*playerlib.Player
    for _, player := range rules.wizards() {
        if !player.IsHuman() {
            out = append(out, player)
        }
    }
    return out
}

// false: upstream's Subversion as this port first had it (the fork's own reading)
var ClassicSubversion = true
