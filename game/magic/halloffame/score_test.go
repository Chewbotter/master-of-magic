package halloffame

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
)

// End_Of_Game_Score_Draw
func TestScore(test *testing.T) {
    score := Score{
        Spells: 10,
        Townsfolk: 57,
        Banished: []data.WizardBase{data.WizardMerlin, data.WizardKali},
        Fame: 37,
        Turn: 180,
        FortressStands: true,
        Difficulty: data.DifficultyAverage,
    }
    // 10 + 28 + 100 + 74 + (2000 - 360)
    if got := score.Final(); got != 1852 {
        test.Errorf("score %v, want 1852", got)
    }

    score.Difficulty = data.DifficultyImpossible
    if got := score.Final(); got != 1852 * 3 {
        test.Errorf("at Impossible %v, want %v", got, 1852 * 3)
    }

    // another wizard's Spell of Mastery: no time bonus
    score.Difficulty = data.DifficultyAverage
    score.OtherMastery = true
    if got := score.Final(); got != 212 {
        test.Errorf("after another's Spell of Mastery %v, want 212", got)
    }

    score.OtherMastery = false
    score.Mastery = true
    if got := score.Final(); got != 1852 + 250 {
        test.Errorf("with the player's Spell of Mastery %v, want %v", got, 1852 + 250)
    }
}

// a score goes in at the first place with less, the list keeps 10
func TestInsert(test *testing.T) {
    var entries []Entry
    for i := 0; i < Places; i++ {
        entries = append(entries, Entry{Name: "x", Score: 1000 - i * 100})
    }

    out, ok := insert(entries, Entry{Name: "new", Score: 750})
    if !ok || len(out) != Places || out[3].Name != "new" || out[Places - 1].Score != 200 {
        test.Errorf("750 should be 4th and the last dropped: %v %v", ok, out)
    }

    if _, ok := insert(entries, Entry{Name: "low", Score: 50}); ok {
        test.Errorf("a score below all 10 should get no place")
    }

    out, ok = insert(nil, Entry{Name: "first", Score: 5})
    if !ok || len(out) != 1 {
        test.Errorf("the first score should get the first place: %v", out)
    }
}
