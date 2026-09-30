package relations

// What computer wizards do in diplomacy on their own: with each other, and toward the human. Facts
// from the ReMoM project's reconstruction (MoM/src/DIPLOMAC.c: NPC_To_NPC_Treaty_Negotiations,
// G_DIPL_NeedForWar, G_DIPL_SuperiorityWar, DIPL_GetOffMyLawn, NPC_To_Human_Diplomacy,
// Find_Worst_Modifier, Limit_Treaty_Modifiers, DIPL_HumanWarOrPeace, NPC_Diplo_s74420,
// Invader_Army_Strength_Comparison; MoM/src/Spells128.c: Get_Differential_Spell_List). The code
// is ours.

import (
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

// the personality's bonus in judgments of treaties (TBL_AI_PRS_IDK_Mod)
func treatyBonus(player *playerlib.Player) int {
    return personalityTreatyBonus[player.Personality]
}

// empire_mini_pops: the citizens of all cities, in tens
func population(player *playerlib.Player) int {
    total := 0
    for _, city := range player.Cities {
        total += city.Citizens()
    }
    return total / 10
}

// the spells one wizard knows that the other does not, in realms both have books of (arcane
// always), in random order (Get_Differential_Spell_List)
func spellsToGive(from *playerlib.Player, to *playerlib.Player) []spellbook.Spell {
    var candidates []spellbook.Spell
    for _, spell := range from.KnownSpells.Spells {
        if to.KnownSpells.Contains(spell) {
            continue
        }
        if spell.Magic != data.ArcaneMagic && (books(from, spell.Magic) == 0 || books(to, spell.Magic) == 0) {
            continue
        }
        candidates = append(candidates, spell)
    }
    for index := len(candidates) - 1; index > 0; index-- {
        other := roll(index + 1) - 1
        candidates[index], candidates[other] = candidates[other], candidates[index]
    }
    return candidates
}

// NPC_To_NPC_Treaty_Negotiations: every ordered pair of computer wizards that have met talks one
// time in 15 (one in 7 on the hardest level): relations, trust, the other's personality and a roll
// make an alliance or a pact above 150, else maybe an exchange of spells; then maybe goodwill, a
// peace when the will for peace is high enough, patience worn down, sides picked, a war declared
// for a reason, and a warning about units near cities. Then an ally joins its ally's wars
func (rules *Rules) computerToComputer() {
    wizards := rules.computerWizards()
    for _, first := range wizards {
        for _, second := range wizards {
            if first == second || !isActive(first) || !isActive(second) {
                continue
            }
            if roll(15 - rules.Difficulty * 2) != 1 {
                continue
            }
            relation := view(first, second)
            if relation == nil || view(second, first) == nil {
                continue
            }

            base := relation.HiddenRelation + relation.VisibleRelation + treatyBonus(second)
            total := base + roll(100) + (roll(5) + 3) * rules.Difficulty
            switch relation.Treaty {
                case data.TreatyPact: total += 10
                case data.TreatyAlliance: total += 20
            }

            if total + relation.TreatyInterest > 150 && relation.Treaty != data.TreatyAlliance {
                rules.StartTreaty(first, second, data.TreatyAlliance)
            } else if total + relation.TreatyInterest > 150 && relation.Treaty != data.TreatyPact {
                if quirkAllianceBecomesPact || relation.Treaty != data.TreatyAlliance {
                    rules.StartTreaty(first, second, data.TreatyPact)
                }
            } else if relation.TradeInterest + 80 < total {
                // the original hands each the spell it already has; meant: each learns one of the other's
                gives := spellsToGive(first, second)
                gets := spellsToGive(second, first)
                if len(gives) > 0 && len(gets) > 0 {
                    second.LearnSpell(gives[0])
                    first.LearnSpell(gets[0])
                }
            }

            if roll(20) == 1 {
                rules.ChangeRelations(roll(5) + 5, first, second, ActionComputerGoodwill, "", "")
            }
            // the original adds a number of its message text here, small: the will for peace decides
            if relation.Treaty == data.TreatyWar && relation.PeaceInterest > 100 {
                rules.DeclarePeace(first, second)
            }
            adjustModifiers(relation, 3)
            rules.pickSides()
            rules.needForWar(first, second)
            if relation.Action == ActionNone {
                rules.unitsNearCities(second, first)
            }
        }
    }

    for _, first := range wizards {
        for _, second := range wizards {
            if !isActive(first) || !isActive(second) || treaty(first, second) != data.TreatyAlliance {
                continue
            }
            for _, third := range wizards {
                if !isActive(third) || third == first || third == second {
                    continue
                }
                if treaty(third, first) == data.TreatyWar && treaty(third, second) != data.TreatyWar {
                    rules.DeclareWar(second, third)
                }
            }
        }
    }
}

// Invader_Army_Strength_Comparison: the units of an invader on the landmass of the owner's fortress
// against the owner's there: -100 and below when the owner has more, up from -50 when the invader has
func (rules *Rules) superiority(invader *playerlib.Player, owner *playerlib.Player) int {
    if rules.FortressLandCosts == nil {
        return 0
    }
    invasion, defense, ok := rules.FortressLandCosts(invader, owner)
    if !ok {
        return 0
    }
    if defense > invasion {
        if invasion <= 0 {
            return -100
        }
        return -(defense * 100 / invasion) - 50
    }
    if defense <= 0 {
        return 100
    }
    return invasion * 100 / defense - 50
}

// no peace to keep: a computer wizard's side counts no peace with others (the original keeps it on
// the human's side only)
func peaceAllows(victim *playerlib.Player, aggressor *playerlib.Player) bool {
    if !victim.IsHuman() {
        return true
    }
    relation := view(victim, aggressor)
    return relation == nil || relation.PeaceCounter < 1
}

// G_DIPL_NeedForWar: does the aggressor make war on the victim? The human is only threatened from
// turn 100. A chaotic wizard does now and then for nothing (up to 4 in 300, by the level); a
// militarist or expansionist one that is not lawful, one time in 20, and any, another time in 20,
// when it is much stronger on the victim's home landmass and the relation is bad; one allied with
// someone the human is at war with, one time in 10; and from the third level, while fewer computer
// wizards are at war with the human than the level, one that dislikes the human, the more the likelier
func (rules *Rules) needForWar(victim *playerlib.Player, aggressor *playerlib.Player) {
    victimView := view(victim, aggressor)
    if victimView == nil || victimView.Treaty >= data.TreatyWar || !isActive(victim) {
        return
    }
    if victim.IsHuman() && rules.Turn < 100 {
        return
    }

    if aggressor.Personality == playerlib.PersonalityChaotic && roll(300) <= rules.Difficulty && peaceAllows(victim, aggressor) {
        victimView.Action = ActionChaoticWar
        victimView.ActionStrength = 2000
        rules.DeclareWar(aggressor, victim)
    } else {
        for try := 0; try < 2; try++ {
            if roll(20) != 1 {
                continue
            }
            if try == 0 {
                objective := aggressor.Objective == playerlib.ObjectiveMilitarist || aggressor.Objective == playerlib.ObjectiveExpansionist
                if !objective || aggressor.Personality == playerlib.PersonalityLawful {
                    continue
                }
            }
            if !peaceAllows(victim, aggressor) {
                continue
            }
            // the victim's units on the aggressor's home landmass against the aggressor's own there: the
            // aggressor that is safe at home is the likelier
            score := victimView.VisibleRelation + rules.superiority(victim, aggressor) + treatyBonus(aggressor) + victimView.HiddenRelation
            if score <= -150 {
                rules.superiorityWar(aggressor, victim)
            }
        }
    }

    if human := rules.human(); human != nil {
        for _, other := range rules.computerWizards() {
            if treaty(human, other) == data.TreatyWar && treaty(aggressor, other) == data.TreatyAlliance {
                if peaceAllows(victim, aggressor) && roll(10) == 1 && treaty(aggressor, victim) != data.TreatyAlliance {
                    rules.superiorityWar(aggressor, victim)
                }
            }
        }

        if rules.Difficulty >= 2 {
            wars := 0
            for _, other := range rules.computerWizards() {
                if treaty(human, other) >= data.TreatyWar {
                    wars += 1
                }
            }
            if wars < rules.Difficulty && victimView.VisibleRelation < -30 && peaceAllows(victim, aggressor) {
                if roll(15) <= -victimView.VisibleRelation / 10 {
                    rules.superiorityWar(aggressor, victim)
                }
            }
        }
    }
}

// G_DIPL_SuperiorityWar: from the third level a wizard with a pact breaks it one time in 2 (an
// alliance one in 4), sooner with an expansionist victim, and likes it at most 30; without a
// treaty it declares war. The human is told
func (rules *Rules) superiorityWar(aggressor *playerlib.Player, victim *playerlib.Player) {
    aggressorView, victimView := view(aggressor, victim), view(victim, aggressor)
    if aggressorView == nil || victimView == nil {
        return
    }
    tell := func(action int) {
        if victim.IsHuman() {
            victimView.Action = action
            victimView.ActionStrength = 10000
        }
    }

    switch aggressorView.Treaty {
        case data.TreatyAlliance, data.TreatyPact:
            if rules.Difficulty < 2 {
                return
            }
            expansionist := victim.Objective == playerlib.ObjectiveExpansionist
            breaks := false
            if aggressorView.Treaty == data.TreatyAlliance {
                breaks = roll(4) == 1 || (roll(2) == 1 && expansionist)
            } else {
                breaks = roll(2) == 1 || expansionist
            }
            if breaks {
                tell(ActionSuperiorityBreak)
                rules.BreakTreaties(aggressor, victim)
                aggressorView.VisibleRelation = min(aggressorView.VisibleRelation, 30)
                victimView.VisibleRelation = aggressorView.VisibleRelation
            }
        default:
            if peaceAllows(victim, aggressor) {
                if aggressorView.VisibleRelation < 0 {
                    tell(ActionWarDeclared)
                } else {
                    tell(ActionSuperiorityWar)
                }
                rules.DeclareWar(aggressor, victim)
            }
    }
}

// DIPL_GetOffMyLawn: units of a wizard, no settlers or engineers, within 2 squares of a city of
// another. Under a treaty the city's owner warns (a small drop in relations), waits a turn, and the
// third turn breaks the treaty; without one, it minds a little one time in 17
func (rules *Rules) unitsNearCities(cityOwner *playerlib.Player, unitOwner *playerlib.Player) {
    unitView, ownerView := view(unitOwner, cityOwner), view(cityOwner, unitOwner)
    if unitView == nil || ownerView == nil || unitView.Treaty == data.TreatyWar || rules.UnitNearCity == nil || unitOwner.IsNeutral() {
        return
    }
    city, near := rules.UnitNearCity(cityOwner, unitOwner)
    if !near {
        unitView.WarningCounter = 0
        return
    }
    if unitView.Treaty == data.TreatyAlliance || unitView.Treaty == data.TreatyPact {
        switch unitView.WarningCounter {
            case 0:
                rules.ChangeRelations(-roll(5), unitOwner, cityOwner, ActionUnitsNearCity, city, "")
                unitView.WarningCounter = 1
            case 1:
                unitView.WarningCounter = 2
            default:
                unitView.WarningCounter = 0
                rules.ChangeRelations(-5, unitOwner, cityOwner, ActionTreatyBroken, city, "")
                rules.BreakTreaties(cityOwner, unitOwner)
                rules.BreakTreaties(unitOwner, cityOwner)
        }
        return
    }
    unitView.WarningCounter = 0
    if roll(100) < 6 {
        kind := ActionGoodwill
        if roll(5) == 1 {
            kind = ActionUnitsNearCity
        }
        rules.ChangeRelations(-roll(5), unitOwner, cityOwner, kind, city, "")
    }
}

// Find_Worst_Modifier as the original has it: the treaty patience when it is below the exchange
// patience, else 0, and the peace patience when that is more
func worstPatience(relation *playerlib.Relationship) int {
    worst := 0
    if relation.TreatyInterest < relation.TradeInterest {
        worst = relation.TreatyInterest
    }
    if worst < relation.PeaceInterest {
        worst = relation.PeaceInterest
    }
    return worst
}

// Limit_Treaty_Modifiers: the treaty patience stays within -200 and 120
func (rules *Rules) limitTreatyPatience() {
    for _, player := range rules.wizards() {
        for _, relation := range player.PlayerRelations {
            relation.TreatyInterest = clamp(relation.TreatyInterest, -200, 120)
        }
    }
}

// NPC_To_Human_Diplomacy: what every computer wizard decides to say to the human this turn. A new
// acquaintance greets; one that is angry enough (its relation twice, trust, patience and personality
// below -100) either reacts to the grievance of the turn (war, or a broken treaty), offers peace
// when at war, or makes a proposal; any other may find a reason for war, or mind the human's units
// near its cities
func (rules *Rules) computerToHuman() {
    human := rules.human()
    if human == nil {
        return
    }
    rules.limitTreatyPatience()
    for _, other := range rules.computerWizards() {
        relation := view(human, other)
        if relation == nil || view(other, human) == nil {
            continue
        }
        if !isActive(other) {
            relation.Action = ActionNone
            continue
        }

        switch {
            case relation.ContactStage == 1:
                relation.ContactStage = 2
                relation.Action = ActionGreeting + int(other.Personality)
            case relation.Action != ActionWarDeclared && relation.Action != ActionSuperiorityBreak && relation.Action != ActionChaoticWar && relation.Action != ActionSuperiorityWar:
                total := relation.HiddenRelation + relation.VisibleRelation + worstPatience(relation) + treatyBonus(other)
                if total + relation.VisibleRelation >= -100 {
                    relation.Action = ActionNone
                    break
                }
                grievance := relation.ActionStrength * 2
                if relation.ActionStrength < 0 {
                    if relation.ThreatCounter > 0 || roll(75) < abs(grievance) {
                        rules.humanWarOrPeace(other)
                    } else {
                        relation.Action = ActionNone
                    }
                } else if relation.Treaty == data.TreatyWar && 30 + roll(100) < relation.PeaceInterest {
                    rules.humanWarOrPeace(other)
                } else {
                    if relation.ThreatCounter == 0 {
                        grievance += 3
                    }
                    if roll(100) < grievance && roll(4) == 1 {
                        if relation.ThreatCounter > 0 {
                            relation.ThreatCounter = 0
                        } else {
                            rules.propose(other)
                        }
                    }
                    // the original's NPC_Required_Alliances only ever clears the order here
                }
        }

        if relation.Action == ActionNone {
            rules.needForWar(human, other)
        }
        if relation.Action == ActionNone {
            rules.unitsNearCities(other, human)
        }
    }
}

// DIPL_HumanWarOrPeace: an angry computer wizard not at war with the human declares war when the
// relation is -75 or worse; otherwise, when its judgment is bad enough, it warns (and after warnings,
// or without a treaty, may declare war by the size of the grievance) or breaks the treaty. At war
// it offers peace one time in 8 when it wants it, sometimes with gold or a spell
//
// What the wizard says: the reconstruction leaves the order at the grievance, so a warning and a
// broken treaty would both say what the grievance alone says. The messages of the original have a
// warning that names the treaty for every grievance (records 22 to 29, 15 more than the grievance)
// and one that breaks it (31 to 38, 24 more); those are said here (warningRecord, brokenRecord)
func (rules *Rules) humanWarOrPeace(other *playerlib.Player) {
    human := rules.human()
    relation := view(human, other)
    if relation == nil || relation.ActionStrength == 0 {
        if relation != nil {
            relation.Action = ActionNone
        }
        return
    }
    base := relation.HiddenRelation + relation.VisibleRelation + treatyBonus(other)

    if relation.Treaty < data.TreatyWar {
        if relation.VisibleRelation <= -75 {
            rules.DeclareWar(other, human)
            relation.Action = ActionWarDeclared
            return
        }
        if base - roll(100) + relation.TreatyInterest > -100 {
            return
        }
        // the first time a warning; after that war by chance, or the treaty is broken
        if relation.ThreatCounter <= 0 {
            relation.ThreatCounter = 1
            if relation.Treaty != data.TreatyNone {
                relation.Action = warningRecord(relation.Action)
            }
            return
        }
        relation.ThreatCounter += 1
        if relation.ThreatCounter > 2 || relation.Treaty == data.TreatyNone {
            limit := -(relation.ActionStrength + treatyBonus(other) / 2)
            if roll(100) < limit {
                rules.DeclareWar(other, human)
                relation.Action = ActionWarDeclared
                relation.ThreatCounter = 1
            }
        } else {
            rules.BreakTreaties(other, human)
            relation.Action = brokenRecord(relation.Action)
        }
        return
    }

    if base + roll(100) + relation.PeaceInterest >= 50 && roll(8) == 1 {
        relation.Action = ActionProposePeace
        humanPopulation := max(population(human), 1)
        if roll(100) <= population(other) * 30 / humanPopulation {
            if roll(4) > 1 {
                relation.OfferGold = ((population(other) / (roll(3) + 1)) + rules.Turn) / 25 * 25
            } else if spells := spellsToGive(other, human); len(spells) > 0 {
                relation.OfferSpell = spells[0].Name
            }
        }
    }
}

// the grievances that have a warning and a breaking of their own: units near cities (7), battles
// (8), attacked cities (9), the Spell of Mastery (10), a summoning (11), enchantments (12), curses on
// cities (13), expansion (14)
func grievanceHasTreatyWords(kind int) bool {
    return kind >= ActionConquest && kind <= ActionOverextension
}

// "another incident and I shall break our treaty"
func warningRecord(kind int) int {
    if grievanceHasTreatyWords(kind) {
        return kind + 15
    }
    return kind
}

// "our treaty is broken"
func brokenRecord(kind int) int {
    if grievanceHasTreatyWords(kind) {
        return kind + 24
    }
    return kind
}

// NPC_Diplo_s74420: a proposal to the human. A pact when there is none and the relation is above
// 15, when relations, trust, personality, a roll and the patience reach 50; an alliance over a pact
// above 50 when they reach 100; else a pact when the wizard likes the human and has fewer people.
// Now and then a kind personality adds gold or a spell to a treaty
func (rules *Rules) propose(other *playerlib.Player) {
    human := rules.human()
    relation := view(human, other)
    if relation == nil {
        return
    }
    if relation.Action != ActionNone && roll(2) == 1 {
        return
    }
    judgment := relation.HiddenRelation + relation.VisibleRelation + treatyBonus(other)

    proposal := ActionNone
    if relation.Treaty == data.TreatyNone && relation.VisibleRelation > 15 {
        if judgment + roll(100) + relation.TreatyInterest >= 50 {
            proposal = ActionProposePact
        }
    }
    if (relation.Treaty == data.TreatyPact || proposal == ActionProposePact) && relation.VisibleRelation > 50 {
        if judgment + roll(100) + relation.TreatyInterest >= 100 {
            proposal = ActionProposeAlliance
        }
    }
    // the proposal of an exchange of spells (the original's message 5) comes with the screens
    if proposal == ActionNone && relation.VisibleRelation > 0 && population(other) < population(human) && relation.Treaty == data.TreatyNone {
        proposal = ActionProposePact
    }
    if proposal == ActionNone {
        return
    }
    relation.Action = proposal

    if population(other) != 0 && other.Personality > playerlib.PersonalityRuthless {
        if roll(100) <= population(human) * 30 / population(other) && roll(8) == 1 {
            if roll(4) <= 1 {
                if spells := spellsToGive(other, human); len(spells) > 0 {
                    relation.OfferSpell = spells[0].Name
                }
            } else {
                relation.OfferGold = ((population(other) / (roll(3) + 1)) + rules.Turn) / 25 * 25
            }
        }
    }
}
