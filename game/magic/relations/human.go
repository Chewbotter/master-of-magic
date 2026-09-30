package relations

// How a computer wizard answers the human, how its personality and objective are picked, and when
// it razes a city it took. Facts from the ReMoM project's reconstruction (MoM/src/DIPLOMAC.c:
// Diplomacy_Test, Diplomacy_Propose_Treaty__WIP; MoM/src/INITGAME.c:
// Init_Magic_Personalities_Objectives; MoM/src/NewGame.c: TBL_AI_Realm_PRS, TBL_AI_Realm_OBJ;
// MoM/src/Combat.c: Raze_Check). The code is ours.

import (
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/setup"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

// the answer to a proposal of the human (Diplomacy_Test): 0 far off, 1 and 2 close, 3 accepted
const (
    AnswerRefused = 0
    AnswerCloseFar = 1
    AnswerClose = 2
    AnswerAccepted = 3
)

// the threat of the human's army: its units on the landmass of the wizard's fortress against the
// wizard's own there: -50 when it has none, up the stronger it is
func (rules *Rules) humanThreat(human *playerlib.Player, other *playerlib.Player) int {
    if rules.FortressLandCosts == nil {
        return -50
    }
    invasion, defense, ok := rules.FortressLandCosts(human, other)
    if !ok {
        return -50
    }
    if defense < invasion {
        if defense > 0 {
            return -50 + invasion * 100 / defense
        }
        return 50
    }
    if invasion > 0 {
        return -50 + defense * 100 / invasion
    }
    return -50
}

// Diplomacy_Test: the wizard's judgment of a proposal of the human: its patience, trust, relation
// and personality, a roll, the threat of the human's army, 40 more for an ally, less the price of
// the proposal. Every proposal wears the patience down
func (rules *Rules) judge(human *playerlib.Player, other *playerlib.Player, price int) int {
    relation := view(human, other)
    if relation == nil {
        return AnswerRefused
    }
    judgment := relation.TreatyInterest + relation.HiddenRelation + relation.VisibleRelation + treatyBonus(other)
    final := judgment + roll(100) - price + rules.humanThreat(human, other)
    if relation.Treaty == data.TreatyAlliance {
        final += 40
    }
    adjustModifiers(relation, 1)
    relation.TreatyInterest -= 20 + roll(30)

    switch {
        case final < -75: return AnswerRefused
        case final < -50: return AnswerCloseFar
        case final < 0: return AnswerClose
    }
    return AnswerAccepted
}

// the human proposes a treaty (Diplomacy_Propose_Treaty): a pact needs a relation above 10 and a
// judgment against 75, an alliance above 50 and against 125, peace against 60. An answer that is
// close is accepted one time in 2: the original makes a counteroffer there, which the
// reconstruction does not have, and then decides by that roll
func (rules *Rules) ProposeTreaty(human *playerlib.Player, other *playerlib.Player, kind data.TreatyType) int {
    relation := view(human, other)
    if relation == nil {
        return AnswerRefused
    }
    judge := func(price int) int {
        answer := rules.judge(human, other, price)
        if (answer == AnswerCloseFar || answer == AnswerClose) && roll(2) == 1 {
            answer = AnswerAccepted
        }
        return answer
    }
    answer := AnswerRefused
    switch kind {
        case data.TreatyPact:
            if relation.Treaty != data.TreatyNone || relation.VisibleRelation <= 10 {
                return AnswerRefused
            }
            answer = judge(75)
            if answer == AnswerAccepted {
                rules.StartTreaty(human, other, data.TreatyPact)
            }
        case data.TreatyAlliance:
            if relation.Treaty >= data.TreatyAlliance || relation.VisibleRelation <= 50 {
                return AnswerRefused
            }
            answer = judge(125)
            if answer == AnswerAccepted {
                rules.StartTreaty(human, other, data.TreatyAlliance)
            }
        case data.TreatyNone:
            // peace
            if relation.Treaty < data.TreatyWar {
                return AnswerRefused
            }
            answer = judge(60)
            if answer == AnswerAccepted {
                rules.DeclarePeace(human, other)
            }
            adjustModifiers(relation, 2)
    }
    return answer
}

// TBL_AI_Realm_PRS: the chances of the personalities by the realm of most books (Maniacal,
// Ruthless, Aggressive, Chaotic, Lawful, Peaceful)
var realmPersonality = map[data.MagicType][6]int{
    data.NatureMagic: {0, 1, 1, 1, 5, 2},
    data.SorceryMagic: {1, 2, 3, 1, 2, 1},
    data.ChaosMagic: {1, 2, 2, 5, 0, 0},
    data.LifeMagic: {0, 1, 2, 0, 2, 5},
    data.DeathMagic: {3, 3, 2, 1, 1, 0},
}

// TBL_AI_Realm_OBJ: the chances of the objectives by the realm of most books (Pragmatist,
// Militarist, Theurgist, Perfectionist, Expansionist)
var realmObjective = map[data.MagicType][5]int{
    data.NatureMagic: {0, 2, 2, 4, 2},
    data.SorceryMagic: {0, 1, 4, 2, 3},
    data.ChaosMagic: {0, 3, 2, 2, 3},
    data.LifeMagic: {0, 2, 3, 4, 1},
    data.DeathMagic: {0, 4, 1, 1, 4},
}

// the realm of most books, the first of nature, sorcery, chaos, life, death on a tie; nature
// without books
func primaryRealm(wizard setup.WizardCustom) data.MagicType {
    best := data.NatureMagic
    most := 0
    for _, magic := range []data.MagicType{data.NatureMagic, data.SorceryMagic, data.ChaosMagic, data.LifeMagic, data.DeathMagic} {
        count := 0
        for _, book := range wizard.Books {
            if book.Magic == magic {
                count += book.Count
            }
        }
        if count > most {
            most = count
            best = magic
        }
    }
    return best
}

// the original's weighted pick: a roll of 1 to the sum, the first whose weights reach it; the first
// when all are 0
func weightedPick(weights []int) int {
    total := 0
    for _, weight := range weights {
        total += weight
    }
    if total <= 0 {
        return 0
    }
    left := roll(total) - weights[0]
    index := 0
    for left > 0 && index < len(weights) - 1 {
        index += 1
        left -= weights[index]
    }
    return index
}

// Init_Magic_Personalities_Objectives: a computer wizard's personality and objective, by the realm
// of its most books and its retorts (Warlord: ruthless, aggressive, militarist, expansionist; Chaos
// Mastery: chaotic, theurgist; Nature Mastery: lawful, theurgist; Infernal Power: maniacal; Divine
// Power: peaceful; Channeler: aggressive; Alchemy: perfectionist; Archmage: theurgist; Myrran:
// expansionist; Conjurer: militarist). Life is never maniacal, death never peaceful
func PickPersonality(wizard setup.WizardCustom) (playerlib.Personality, playerlib.Objective) {
    realm := primaryRealm(wizard)
    personality := realmPersonality[realm]
    objective := realmObjective[realm]

    retort := wizard.RetortEnabled
    if retort(data.RetortWarlord) {
        personality[playerlib.PersonalityRuthless] += 2
        personality[playerlib.PersonalityAggressive] += 3
        objective[playerlib.ObjectiveMilitarist] += 2
        objective[playerlib.ObjectiveExpansionist] += 3
    }
    if retort(data.RetortChaosMastery) {
        personality[playerlib.PersonalityChaotic] += 3
        objective[playerlib.ObjectiveTheurgist] += 1
    }
    if retort(data.RetortNatureMastery) {
        personality[playerlib.PersonalityLawful] += 3
        objective[playerlib.ObjectiveTheurgist] += 1
    }
    if retort(data.RetortInfernalPower) {
        personality[playerlib.PersonalityManiacal] += 4
    }
    if retort(data.RetortDivinePower) {
        personality[playerlib.PersonalityPeaceful] += 4
    }
    if retort(data.RetortChanneler) {
        personality[playerlib.PersonalityAggressive] += 2
    }
    if retort(data.RetortAlchemy) {
        objective[playerlib.ObjectivePerfectionist] += 2
    }
    if retort(data.RetortArchmage) {
        objective[playerlib.ObjectiveTheurgist] += 4
    }
    if retort(data.RetortMyrran) {
        objective[playerlib.ObjectiveExpansionist] += 4
    }
    if retort(data.RetortConjurer) {
        objective[playerlib.ObjectiveMilitarist] += 3
    }
    if realm == data.LifeMagic {
        personality[playerlib.PersonalityManiacal] = 0
    }
    if realm == data.DeathMagic {
        personality[playerlib.PersonalityPeaceful] = 0
    }

    return playerlib.Personality(weightedPick(personality[:])), playerlib.Objective(weightedPick(objective[:]))
}

// Raze_Check: a computer wizard that took a city of another wizard (never a neutral one) razes it
// by its personality (maniacal 25, ruthless 10, lawful 25 when it mistrusts the owner else no,
// peaceful no), 10 more with fewer than 3 units left, no when the owner has no units on the
// landmass, else by its units there against the owner's; a chaotic one at least one time in 10
func RazeCity(conqueror *playerlib.Player, owner *playerlib.Player, survivors int, ownUnits int, ownerUnits int) bool {
    if owner == nil || owner.IsNeutral() {
        return false
    }
    score := 0
    switch conqueror.Personality {
        case playerlib.PersonalityManiacal: score += 25
        case playerlib.PersonalityRuthless: score += 10
        case playerlib.PersonalityLawful:
            if relation := view(conqueror, owner); relation != nil && relation.HiddenRelation != 0 {
                score += 25
            } else {
                score -= 200
            }
        case playerlib.PersonalityPeaceful: score -= 200
    }
    if survivors < 3 {
        score += 10
    }
    if ownerUnits == 0 {
        score -= 200
    } else {
        score += ownUnits * 50 / ownerUnits - 50
    }
    if conqueror.Personality == playerlib.PersonalityChaotic {
        score = max(score, 10)
    }
    return roll(100) <= score
}

// the amounts of gold the human may offer as tribute (Diplomacy_Offer_Tribute): with less than 100,
// 25 to 100 in steps of 25 as far as the gold goes; else a quarter, a half, three quarters and all
// of it, in 25s (the original takes the computer wizard's gold here, meant: the human's)
func TributeGoldAmounts(gold int) []int {
    gold = gold / 25 * 25
    var amounts []int
    if gold < 100 {
        for amount := 25; amount <= gold; amount += 25 {
            amounts = append(amounts, amount)
        }
        return amounts
    }
    for quarter := 1; quarter <= 4; quarter++ {
        amounts = append(amounts, gold * quarter / 4 / 25 * 25)
    }
    return amounts
}

// gold as tribute: the relation rises by a little, more for the larger amounts (choice 0 to 3), and
// that twice unless it passes 100 the first time
func (rules *Rules) TributeGold(human *playerlib.Player, other *playerlib.Player, choice int, amount int) {
    humanView := view(human, other)
    if humanView == nil || amount <= 0 || human.Gold < amount {
        return
    }
    human.Gold -= amount
    other.Gold += amount
    weight := 8
    if rules.ArmyStrength != nil && rules.ArmyStrength(other) != 0 {
        weight = (choice + 1) * 4
    }
    gain := min(weight * roll(3) / 10, 100)
    humanView.VisibleRelation += gain
    if humanView.VisibleRelation > 100 {
        humanView.VisibleRelation = 100
    } else {
        humanView.VisibleRelation = min(humanView.VisibleRelation + gain, 100)
    }
    mirrorVisible(human, other)
}

// a spell as tribute: the wizard learns it; the relation rises the more the worse it was, the will
// for peace as much, and the trust by 3 to 10 (at most 30); the wizard remembers the gift
func (rules *Rules) TributeSpell(human *playerlib.Player, other *playerlib.Player, spell spellbook.Spell) {
    humanView := view(human, other)
    if humanView == nil {
        return
    }
    other.LearnSpell(spell)
    weight := 20
    if humanView.VisibleRelation >= 0 {
        weight = (100 - humanView.VisibleRelation) / 10
    }
    gain := weight * (roll(8) + roll(8)) / 10
    humanView.VisibleRelation = min(humanView.VisibleRelation + gain, 100)
    mirrorVisible(human, other)
    humanView.PeaceInterest = min(humanView.PeaceInterest + gain, 100)
    humanView.HiddenRelation = min(humanView.HiddenRelation + 2 + roll(8), 30)
    humanView.TributeSpell = spell.Name
}
