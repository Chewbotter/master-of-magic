package relations

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/setup"
    herolib "github.com/kazzmir/master-of-magic/game/magic/hero"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

func makeWizard(name string, banner data.BannerType, human bool, books ...data.WizardBook) *playerlib.Player {
    return playerlib.MakePlayer(setup.WizardCustom{Name: name, Banner: banner, Books: books}, human, 10, 10, make(map[herolib.HeroType]string), nil)
}

// three wizards that have all met: the human and two computer wizards
func makeThree() (*Rules, *playerlib.Player, *playerlib.Player, *playerlib.Player) {
    human := makeWizard("Merlin", data.BannerRed, true)
    kali := makeWizard("Kali", data.BannerPurple, false)
    jafar := makeWizard("Jafar", data.BannerGreen, false)
    players := []*playerlib.Player{human, kali, jafar}
    for _, a := range players {
        for _, b := range players {
            if a != b {
                a.AwarePlayer(b)
            }
        }
    }
    return &Rules{Players: players, Turn: 50, Difficulty: 2}, human, kali, jafar
}

// Change_Relations: only the computer wizard's feelings change, the human's follow; bad is doubled
// against a good relation, and without an alliance the relation is at most 65
func TestChangeRelations(test *testing.T) {
    rules, human, kali, _ := makeThree()
    // a personality that changes nothing here (maniacal, the first, takes it half again)
    kali.Personality = playerlib.PersonalityAggressive
    kali.PlayerRelations[human].VisibleRelation = 20
    rules.ChangeRelations(-10, human, kali, ActionBattle, "", "")
    if got := kali.PlayerRelations[human].VisibleRelation; got != 0 {
        test.Errorf("20 less twice 10: %v, want 0", got)
    }
    if got := human.PlayerRelations[kali].VisibleRelation; got != 0 {
        test.Errorf("the human's relation follows: %v", got)
    }
    if got := human.PlayerRelations[kali].ActionStrength; got != -20 {
        test.Errorf("the grievance of the turn: %v, want -20", got)
    }

    rules.ChangeRelations(100, human, kali, ActionGoodwill, "", "")
    if got := kali.PlayerRelations[human].VisibleRelation; got != 65 {
        test.Errorf("without an alliance at most 65: %v", got)
    }

    before := human.PlayerRelations[kali].VisibleRelation
    rules.ChangeRelations(-10, kali, human, ActionBattle, "", "")
    if got := human.PlayerRelations[kali].VisibleRelation; got != before {
        test.Errorf("the human's feelings never change: %v, was %v", got, before)
    }
}

// Declare_War: both at war, hostile, relations between -76 and -100, no patience left
func TestDeclareWar(test *testing.T) {
    rules, human, kali, _ := makeThree()
    rules.StartTreaty(human, kali, data.TreatyPact)
    rules.DeclareWar(kali, human)
    for _, relation := range []*playerlib.Relationship{kali.PlayerRelations[human], human.PlayerRelations[kali]} {
        if relation.Treaty != data.TreatyWar || relation.Hostility != 3 || relation.VisibleRelation > -76 || relation.TreatyInterest != -200 {
            test.Errorf("after the declaration: %+v", *relation)
        }
    }
    if human.PlayerRelations[kali].BrokenTreaty != data.TreatyPact {
        test.Errorf("the pact was broken first")
    }
}

// Break_Treaties: the breaker pays: its victim trusts it less for good, and every other wizard a little
func TestBreakTreaties(test *testing.T) {
    rules, human, kali, jafar := makeThree()
    kali.Personality = playerlib.PersonalityLawful
    rules.StartTreaty(kali, human, data.TreatyAlliance)
    rules.BreakTreaties(kali, human)
    // the original's signs and sides (quirkBreakTreatiesBackwards): the breaker's trust in its victim
    // rises, every other wizard trusts the victim less, the lasting relation rises and is shared
    if got := kali.PlayerRelations[human].HiddenRelation; got != 40 {
        test.Errorf("a lawful wizard breaking an alliance: the breaker's trust %v, want 40", got)
    }
    if got := jafar.PlayerRelations[human].HiddenRelation; got != -5 {
        test.Errorf("another wizard's trust in the victim: %v, want -5", got)
    }
    if got := human.PlayerRelations[kali].StartingRelation; got != kali.PlayerRelations[human].StartingRelation {
        test.Errorf("the lasting relation is shared: %v", got)
    }
    if kali.PlayerRelations[human].Treaty != data.TreatyNone {
        test.Errorf("the alliance is over")
    }
}

// the answer to the human's proposal: a pact wants a relation above 10
func TestProposePact(test *testing.T) {
    defer FixedRolls(100)()
    rules, human, kali, _ := makeThree()
    human.PlayerRelations[kali].VisibleRelation = 5
    kali.PlayerRelations[human].VisibleRelation = 5
    if answer := rules.ProposeTreaty(human, kali, data.TreatyPact); answer != AnswerRefused {
        test.Errorf("relation 5: %v, want refused", answer)
    }
    human.PlayerRelations[kali].VisibleRelation = 60
    kali.PlayerRelations[human].VisibleRelation = 60
    kali.Personality = playerlib.PersonalityPeaceful
    if answer := rules.ProposeTreaty(human, kali, data.TreatyPact); answer != AnswerAccepted {
        test.Errorf("relation 60, a peaceful wizard, the best roll: %v, want accepted", answer)
    }
    if kali.PlayerRelations[human].Treaty != data.TreatyPact || human.PlayerRelations[kali].Treaty != data.TreatyPact {
        test.Errorf("the pact holds both ways")
    }
}

// the personalities and objectives of the original by the realm of most books: life is never
// maniacal, death never peaceful, nobody a pragmatist
func TestPickPersonality(test *testing.T) {
    for realm, weights := range realmObjective {
        if weights[playerlib.ObjectivePragmatist] != 0 {
            test.Errorf("realm %v can give a pragmatist", realm)
        }
    }
    life := setup.WizardCustom{Books: []data.WizardBook{{Magic: data.LifeMagic, Count: 8}}, Retorts: []data.Retort{data.RetortInfernalPower}}
    for value := 1; value <= 20; value++ {
        undo := FixedRolls(value)
        personality, _ := PickPersonality(life)
        undo()
        if personality == playerlib.PersonalityManiacal {
            test.Errorf("a life wizard should never be maniacal (roll %v)", value)
        }
    }
    death := setup.WizardCustom{Books: []data.WizardBook{{Magic: data.DeathMagic, Count: 8}}}
    for value := 1; value <= 20; value++ {
        undo := FixedRolls(value)
        personality, objective := PickPersonality(death)
        undo()
        if personality == playerlib.PersonalityPeaceful || objective == playerlib.ObjectivePragmatist {
            test.Errorf("a death wizard: %v %v (roll %v)", personality, objective, value)
        }
    }
}

// the answer to a proposal on the hardest roll wears the patience down
func TestProposalsWearPatience(test *testing.T) {
    defer FixedRolls(1)()
    rules, human, kali, _ := makeThree()
    human.PlayerRelations[kali].VisibleRelation = 60
    before := human.PlayerRelations[kali].TreatyInterest
    rules.ProposeTreaty(human, kali, data.TreatyPact)
    if got := human.PlayerRelations[kali].TreatyInterest; got != before - 10 - 21 {
        test.Errorf("patience after a proposal: %v, want %v", got, before - 31)
    }
}

// computer wizards that like each other enough make an alliance; when the other of the two passes the
// test again in the same turn, the original makes it a pact (quirkAllianceBecomesPact)
func TestComputerWizardsAlly(test *testing.T) {
    defer FixedRolls(1)()
    rules, _, kali, jafar := makeThree()
    for _, pair := range [][2]*playerlib.Player{{kali, jafar}, {jafar, kali}} {
        pair[0].PlayerRelations[pair[1]].VisibleRelation = 100
        pair[0].PlayerRelations[pair[1]].TreatyInterest = 100
    }
    jafar.Personality = playerlib.PersonalityPeaceful
    kali.Personality = playerlib.PersonalityPeaceful
    rules.computerToComputer()
    want := data.TreatyAlliance
    if quirkAllianceBecomesPact {
        want = data.TreatyPact
    }
    if kali.PlayerRelations[jafar].Treaty != want {
        test.Errorf("two wizards in harmony, both talking: %v, want %v", kali.PlayerRelations[jafar].Treaty, want)
    }
}
