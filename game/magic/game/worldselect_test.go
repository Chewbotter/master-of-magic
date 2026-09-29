package game

import (
    "testing"

    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

func TestSelectionBlinkStartsWhole(test *testing.T) {
    first := &playerlib.UnitStack{}
    second := &playerlib.UnitStack{}

    // whatever the clock of the game says when a stack is selected, its square is shown for a whole time
    for _, start := range []uint64{0, 7, 19, 33, 1234} {
        selectionBlinkStack = nil
        for tick := uint64(0); tick < SelectionBlinkTicks; tick++ {
            if !selectionBlinkOn(first, start + tick, false) {
                test.Fatalf("selected at %v: the square is gone after %v ticks", start, tick)
            }
        }
        for tick := uint64(SelectionBlinkTicks); tick < SelectionBlinkTicks * 2; tick++ {
            if selectionBlinkOn(first, start + tick, false) {
                test.Fatalf("selected at %v: the square is shown at tick %v", start, tick)
            }
        }

        // another stack in the middle of the time the square is gone
        if !selectionBlinkOn(second, start + SelectionBlinkTicks * 2 - 3, false) {
            test.Fatalf("selected at %v: the square of a newly selected stack is not shown", start)
        }
    }

    // shown while the stack walks, and for a whole time after it
    selectionBlinkStack = nil
    for tick := uint64(0); tick < 100; tick++ {
        if !selectionBlinkOn(first, tick, true) {
            test.Fatalf("the square is gone while the stack walks")
        }
    }
    for tick := uint64(100); tick < 99 + SelectionBlinkTicks; tick++ {
        if !selectionBlinkOn(first, tick, false) {
            test.Fatalf("the square is gone %v ticks after the walk", tick - 99)
        }
    }
}
