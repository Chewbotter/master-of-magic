package spellbook

import (
    "testing"
)

func TestPageComesUp(test *testing.T) {
    if pageComesUp(0) != 0 || pageComesUp(0.5) != 0 {
        test.Errorf("in the first half of the turn: %v %v", pageComesUp(0), pageComesUp(0.5))
    }
    if pageComesUp(1) != 1 {
        test.Errorf("at the end: %v", pageComesUp(1))
    }

    last := float32(0)
    for step := 50; step <= 100; step++ {
        now := pageComesUp(float64(step) / 100)
        if now < last {
            test.Errorf("less at %v: %v after %v", step, now, last)
        }
        last = now
    }

    if pageTurnPart(12, 24) != 0.5 || pageTurnPart(30, 24) != 1 || pageTurnPart(3, 0) != 1 {
        test.Errorf("how far a turn has come")
    }
}
