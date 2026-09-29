package spellbook

import (
    "testing"
)

func TestTurnIsClearOfTheTabs(test *testing.T) {
    books := []leafSize{
        {Width: castBookRightPage.Dx(), Reach: castTabReach},
        {Width: researchBookRightPage.Dx(), Reach: researchTabReach},
    }

    for _, book := range books {
        clear := pageTurnClear(book.Width, book.Reach)
        if clear <= 0 || clear >= 0.5 {
            test.Fatalf("width %v: the leaf is clear of the tab at %v", book.Width, clear)
        }

        for _, mirror := range []bool{false, true} {
            // from the first tick on no part of the leaf is under the tab of the side it leaves
            for tick := 0; tick < PageTurnTicks; tick++ {
                turned := turnedAt(pageTurnPart(uint64(tick), PageTurnTicks), book.Width, book.Reach)
                for _, strip := range leafStrips(book.Width, turned, mirror) {
                    // the side the leaf leaves: the right one, in a mirror the left one
                    column := strip.Column
                    if mirror {
                        column = -strip.Column - 1
                    }
                    if column >= book.Width - book.Reach {
                        test.Fatalf("width %v, mirror %v, tick %v: column %v of the leaf is under the tab it leaves", book.Width, mirror, tick, strip.Column)
                    }
                }
            }
        }

        if turnedAt(1, book.Width, book.Reach) != 1 {
            test.Errorf("width %v: the turn does not end with the leaf down", book.Width)
        }

        // it never goes back
        last := 0.0
        for tick := 0; tick <= PageTurnTicks; tick++ {
            turned := turnedAt(pageTurnPart(uint64(tick), PageTurnTicks), book.Width, book.Reach)
            if turned < last {
                test.Errorf("width %v: the leaf goes back at tick %v", book.Width, tick)
            }
            last = turned
        }
    }
}

func TestNewTurnEndsTheOneThatRuns(test *testing.T) {
    var turn runningTurn
    ended := 0

    turn.finish = func() {
        ended += 1
    }
    turn.end()
    turn.end()
    if ended != 1 {
        test.Errorf("a turn ended %v times", ended)
    }
}
