package relations

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

// Diplomacy_Greeting: a warm wizard greets, a cold one grants an audience, one without trust and
// patience does not talk
func TestGreeting(test *testing.T) {
    rules, human, kali, _ := makeThree()
    kali.Personality = playerlib.PersonalityManiacal
    relation := human.PlayerRelations[kali]
    relation.VisibleRelation = 20
    if got := rules.Greeting(human, kali); got != RecordGreetingGood {
        test.Errorf("a relation of 20: %v, want %v", got, RecordGreetingGood)
    }
    relation.VisibleRelation = -60
    if got := rules.Greeting(human, kali); got != RecordGreetingCool {
        test.Errorf("a relation of -60: %v, want %v", got, RecordGreetingCool)
    }
    relation.HiddenRelation = -100

    if got := rules.Greeting(human, kali); got != RecordNoAudience {
        test.Errorf("no trust and no patience: %v, want %v", got, RecordNoAudience)
    }
}

// the human accepts a pact the wizard proposes: the treaty holds both ways, the offered gold comes,
// the patience for treaties wears down
func TestAnswerProposal(test *testing.T) {
    defer FixedRolls(1)()
    rules, human, kali, _ := makeThree()
    human.Gold = 10
    before := human.PlayerRelations[kali].TreatyInterest
    rules.AnswerProposal(human, kali, Message{From: kali, Action: ActionProposePact, OfferGold: 50}, true)
    if human.PlayerRelations[kali].Treaty != data.TreatyPact || kali.PlayerRelations[human].Treaty != data.TreatyPact {
        test.Errorf("the pact holds both ways")
    }
    if human.Gold != 60 {
        test.Errorf("gold after the offer: %v, want 60", human.Gold)
    }
    if got := human.PlayerRelations[kali].TreatyInterest; got != before - 10 - 21 {
        test.Errorf("patience: %v, want %v", got, before - 31)
    }
}

// the human attacks a partner: the treaty is over and the wizard minds it
func TestHumanAttacksPartner(test *testing.T) {
    rules, human, kali, _ := makeThree()
    rules.StartTreaty(human, kali, data.TreatyPact)
    before := kali.PlayerRelations[human].VisibleRelation
    rules.HumanAttacksPartner(human, kali)
    if kali.PlayerRelations[human].Treaty != data.TreatyNone {
        test.Errorf("the pact is over: %v", kali.PlayerRelations[human].Treaty)
    }
    if kali.PlayerRelations[human].VisibleRelation >= before {
        test.Errorf("the relation drops: %v, was %v", kali.PlayerRelations[human].VisibleRelation, before)
    }
}

// a threat with no army of the human on the wizard's land and the worst rolls: war
func TestThreatenWithoutArmy(test *testing.T) {
    defer FixedRolls(1)()
    rules, human, kali, _ := makeThree()
    rules.FortressLandCosts = func(invader *playerlib.Player, owner *playerlib.Player) (int, int, bool) {
        return 0, 500, true
    }
    human.PlayerRelations[kali].VisibleRelation = 0
    kali.PlayerRelations[human].VisibleRelation = 0
    answer, ok := rules.Threaten(human, kali)
    if !ok || answer.Record != ActionWarDeclared {
        test.Errorf("threat without an army: %+v %v, want war", answer, ok)
    }
    if human.PlayerRelations[kali].Treaty != data.TreatyWar {
        test.Errorf("at war after it: %v", human.PlayerRelations[kali].Treaty)
    }
}

// a warning names the treaty, a breaking says it is broken (records 15 and 24 past the grievance)
func TestWarningRecords(test *testing.T) {
    // the original says the grievance's words (quirkWarningWordsOfGrievance)
    if got := warningRecord(ActionBattle); got != ActionBattle {
        test.Errorf("warning for a battle: %v, want %v", got, ActionBattle)
    }
    if got := brokenRecord(ActionCityAttacked); got != ActionCityAttacked {
        test.Errorf("broken over an attacked city: %v, want %v", got, ActionCityAttacked)
    }
}

// a proposal that is close is taken one time in 2
func TestCloseProposalAccepted(test *testing.T) {
    defer FixedRolls(1)()
    rules, human, kali, _ := makeThree()
    kali.Personality = playerlib.PersonalityPeaceful
    relation := human.PlayerRelations[kali]
    relation.VisibleRelation = 30
    kali.PlayerRelations[human].VisibleRelation = 30
    // judgment: patience 0 + trust 0 + 30 + 50 (peaceful) + 1 - 75 - 50 (no army) = -44: close
    relation.TreatyInterest = 0
    if answer := rules.ProposeTreaty(human, kali, data.TreatyPact); answer != AnswerAccepted {
        test.Errorf("a close answer on the roll of 1: %v, want accepted", answer)
    }
}
