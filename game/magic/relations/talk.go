package relations

// What a talk of the human with a computer wizard changes, for the screens of diplomacy
// (diplomacy/). Facts from the ReMoM project's reconstruction (MoM/src/DIPLOMAC.c:
// Diplomacy_Greeting, Get_Main_Diplomacy_Choices, Diplomacy_Display_Response,
// Diplomacy_Break_Treaty, Diplomacy_Offer_Tribute, Diplomacy_Exchange_Spell__WIP,
// Npc_Diplomacy_Screen, Npc_Proposal__WIP; MoM/src/Combat.c: the attack on a partner). The code is
// ours.

import (
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

// the records of the original's messages (DIPLOMSG.LBX) the screens show; the grievances,
// greetings and proposals are the orders of the same numbers (Action...)
const (
    RecordTributeThanks = 1
    RecordTreatyBrokenByHuman = 5
    RecordGreetingGood = 42
    RecordGreetingCool = 43
    RecordNoAudience = 44
    RecordBreakYourTreaty = 47
    RecordPactAccepted = 51
    RecordAllianceAccepted = 52
    RecordPeaceAccepted = 53
    RecordGiftRemembered = 54
    RecordRefused = 57
    RecordRefusedBroken = 58
    RecordThreatIgnored = 67
    RecordThreatGold = 69
    RecordThreatSpell = 70
    RecordImpatient = 71
    RecordNothingToTrade = 72
)

// Diplomacy_Greeting: how a wizard receives the human who opens a talk: its patience, trust,
// relation and personality; with the trust and patience alone at -100 or worse it does not talk
func (rules *Rules) Greeting(human *playerlib.Player, other *playerlib.Player) int {
    relation := view(human, other)
    if relation == nil {
        return RecordNoAudience
    }
    level := worstPatience(relation) + relation.HiddenRelation + relation.VisibleRelation + treatyBonus(other)
    if level - relation.VisibleRelation <= -100 {
        return RecordNoAudience
    }
    if level > -50 {
        return RecordGreetingGood
    }
    return RecordGreetingCool
}

// Get_Main_Diplomacy_Choices: the wizard has had enough of talk for now
func (rules *Rules) Impatient(human *playerlib.Player, other *playerlib.Player) bool {
    relation := view(human, other)
    return relation != nil && worstPatience(relation) < -100
}

// Diplomacy_Display_Response for most answers: one time in 4 a wizard the human gave a spell as
// tribute answers with its thanks for that spell instead, and forgets the gift
func (rules *Rules) GiftAnswer(human *playerlib.Player, other *playerlib.Player) (string, bool) {
    relation := view(human, other)
    if roll(4) == 1 && relation != nil && relation.TributeSpell != "" {
        spell := relation.TributeSpell
        relation.TributeSpell = ""
        return spell, true
    }
    return "", false
}

// Diplomacy_Display_Response for a proposal that was refused: one time in 4 after a broken treaty
// the refusal names that (and it is forgotten)
func (rules *Rules) RefusalRecord(human *playerlib.Player, other *playerlib.Player) int {
    relation := view(human, other)
    if relation != nil && relation.BrokenTreaty != data.TreatyNone && roll(4) == 1 {
        relation.BrokenTreaty = data.TreatyNone
        return RecordRefusedBroken
    }
    return RecordRefused
}

// the human breaks its pact or alliance with the wizard (Diplomacy_Break_Treaty)
func (rules *Rules) HumanBreaksTreaty(human *playerlib.Player, other *playerlib.Player) {
    rules.BreakTreaties(human, other)
}

// Diplomacy_Break_Treaty: whatever the human picks in the list of its actions, Forget It too, the
// wizard's patience wears down ten times over
func (rules *Rules) AfterThreats(human *playerlib.Player, other *playerlib.Player) {
    adjustModifiers(view(human, other), 10)
}

// what a threat of the human brings
type ThreatAnswer struct {
    Record int
    Gold int
    Spell string
}

// Diplomacy_Break_Treaty, "Threaten To Attack": the threat of the human's army on the landmass of
// the wizard's fortress (as the reconstruction reads it: the more the two armies there differ,
// whichever is the larger, the more it counts; -100 without the human's), trust, relation,
// personality and two rolls of 100. The threat always costs relation and patience. Below 0 the
// wizard declares war, from 90 it pays 75 to 250 gold (as far as it has it), from 200 it gives a
// spell the human lacks instead; either keeps the peace for a while. false: the wizard has no
// fortress, and nothing is said
func (rules *Rules) Threaten(human *playerlib.Player, other *playerlib.Player) (ThreatAnswer, bool) {
    relation := view(human, other)
    if relation == nil || rules.FortressLandCosts == nil {
        return ThreatAnswer{}, false
    }
    invasion, defense, ok := rules.FortressLandCosts(human, other)
    if !ok {
        return ThreatAnswer{}, false
    }
    threat := 0
    if invasion < defense {
        if invasion > 0 {
            threat = -50 + defense * 100 / invasion
        } else {
            threat = -100
        }
    } else {
        if defense > 0 {
            threat = -50 + invasion * 100 / defense
        } else {
            threat = 100
        }
    }
    judgment := relation.HiddenRelation + relation.VisibleRelation + treatyBonus(other) + threat + roll(100) + roll(100)

    relation.ActionStrength -= roll(5)
    relation.ThreatCounter += 1
    relation.VisibleRelation = max(relation.VisibleRelation - roll(15), -100)
    relation.TreatyInterest = -120
    mirrorVisible(human, other)

    answer := ThreatAnswer{Record: RecordThreatIgnored}
    switch {
        case judgment < 0:
            rules.DeclareWar(human, other)
            answer.Record = ActionWarDeclared
        case judgment >= 200:
            if spells := spellsToGive(other, human); len(spells) > 0 {
                relation.PeaceCounter = roll(15)
                human.LearnSpell(spells[0])
                answer.Record = RecordThreatSpell
                answer.Spell = spells[0].Name
            }
        case judgment >= 90:
            amount := (2 + roll(8)) * 25
            if other.Gold < amount {
                amount = max(other.Gold, 0)
            }
            if amount != 0 {
                relation.PeaceCounter = roll(15)
                // the original gives the gold to the human and takes none from the wizard; meant: it pays
                other.Gold -= amount
                human.Gold += amount
                answer.Record = RecordThreatGold
                answer.Gold = amount
            }
    }
    return answer, true
}

// the spells the human may give as tribute: 4 at most it knows and the wizard does not
// (Diplomacy_Offer_Tribute)
func (rules *Rules) TributeSpells(human *playerlib.Player, other *playerlib.Player) []spellbook.Spell {
    spells := spellsToGive(human, other)
    if len(spells) > 4 {
        spells = spells[:4]
    }
    return spells
}

// a spell of the wizard and the spells of the human it would take for it
type ExchangeOffer struct {
    Spell spellbook.Spell
    For []spellbook.Spell
}

// Diplomacy_Exchange_Spell__WIP: what the wizard would trade: every spell it knows and the human
// does not (5 at most) for which the human has spells it lacks of at least the same worth (4 at most
// each). Worth is the research cost here: the original's comparison of worth is broken (the
// reconstruction notes the values do not fit their bytes). Asking wears the patience down
func (rules *Rules) ExchangeOffers(human *playerlib.Player, other *playerlib.Player) []ExchangeOffer {
    relation := view(human, other)
    if relation == nil {
        return nil
    }
    var offers []ExchangeOffer
    theirs := spellsToGive(other, human)
    mine := spellsToGive(human, other)
    for _, spell := range theirs {
        var trade []spellbook.Spell
        for _, candidate := range mine {
            if candidate.ResearchCost >= spell.ResearchCost {
                trade = append(trade, candidate)
            }
        }
        if len(trade) == 0 {
            continue
        }
        if len(trade) > 4 {
            trade = trade[:4]
        }
        offers = append(offers, ExchangeOffer{Spell: spell, For: trade})
        if len(offers) >= 5 {
            break
        }
    }
    if len(offers) > 0 {
        adjustModifiers(relation, 1)
        relation.TradeInterest -= 20 + roll(50)
    }
    return offers
}

// the trade: both learn the other's spell; when the wizard gave the cheaper one it is pleased
func (rules *Rules) Exchange(human *playerlib.Player, other *playerlib.Player, theirs spellbook.Spell, mine spellbook.Spell) {
    relation := view(human, other)
    if relation == nil {
        return
    }
    human.LearnSpell(theirs)
    other.LearnSpell(mine)
    if theirs.ResearchCost < mine.ResearchCost {
        adjustModifiers(relation, 2)
        rules.ChangeRelations(5, human, other, ActionGoodwill, "", "")
    }
}

// the end of a talk of an exchange, whatever came of it: the patience wears down, the will to trade
// is at most 50
func (rules *Rules) EndExchange(human *playerlib.Player, other *playerlib.Player) {
    relation := view(human, other)
    if relation == nil {
        return
    }
    adjustModifiers(relation, 3)
    relation.TradeInterest = min(relation.TradeInterest, 50)
}

// Npc_Proposal__WIP: before its proposal a wizard at war with an ally of the human asks the human
// to end that alliance. The ally, the last of them in the game's order; nil for none
func (rules *Rules) AllyToAbandon(human *playerlib.Player, other *playerlib.Player) *playerlib.Player {
    var found *playerlib.Player
    for _, third := range rules.wizards() {
        if third == human || third == other {
            continue
        }
        if treaty(other, third) >= data.TreatyWar && treaty(human, third) == data.TreatyAlliance {
            found = third
        }
    }
    return found
}

// the answer to that: agreed, the human breaks the alliance; refused, the wizard likes it 6 to 11
// less (the reconstruction's further lines that set relations to -100 are ones it doubts itself)
func (rules *Rules) AnswerAbandon(human *playerlib.Player, other *playerlib.Player, ally *playerlib.Player, agree bool) {
    if agree {
        rules.BreakTreaties(human, ally)
        return
    }
    relation := view(human, other)
    if relation == nil {
        return
    }
    relation.VisibleRelation = max(relation.VisibleRelation - (5 + roll(6)), -100)
    mirrorVisible(human, other)
}

// Npc_Diplomacy_Screen: the human answers a proposal of a wizard (pact, alliance, peace). Accepted:
// the treaty, and the gold and spell the wizard offered with it. Either way the patience wears down
func (rules *Rules) AnswerProposal(human *playerlib.Player, other *playerlib.Player, message Message, accept bool) {
    relation := view(human, other)
    if relation == nil {
        return
    }
    if accept {
        switch message.Action {
            case ActionProposePact: rules.StartTreaty(human, other, data.TreatyPact)
            case ActionProposeAlliance: rules.StartTreaty(human, other, data.TreatyAlliance)
            case ActionProposePeace:
                rules.DeclarePeace(human, other)
                // the original raises the relation by 20 and then overwrites it (quirkPeaceNoGain)
                if !quirkPeaceNoGain && relation.VisibleRelation < 80 {
                    relation.VisibleRelation += 20
                    mirrorVisible(human, other)
                }
                adjustModifiers(relation, 1)
                adjustModifiers(view(other, human), 1)
        }
        human.Gold += message.OfferGold
        if message.OfferSpell != "" {
            if spell := other.KnownSpells.FindByName(message.OfferSpell); spell.Valid() {
                human.LearnSpell(spell)
            }
        }
    }
    adjustModifiers(relation, 1)
    switch message.Action {
        case ActionProposePact, ActionProposeAlliance:
            relation.TreatyInterest -= 20 + roll(30)
        case ActionProposePeace:
            relation.PeaceInterest -= 50 + roll(50)
    }
}

// the human attacks a wizard it has a pact or alliance with (Combat.c, after its question): the
// wizard likes it 40 less, the treaty is over. The original calls Break_Treaties with the sides the
// other way round; here the human broke it
func (rules *Rules) HumanAttacksPartner(human *playerlib.Player, other *playerlib.Player) {
    rules.ChangeRelations(-40, human, other, ActionTreatyBroken, "", "")
    rules.BreakTreaties(human, other)
}
