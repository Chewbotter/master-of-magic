package game

import (
    "testing"

    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

func TestNextToStack(test *testing.T) {
    cases := []struct {
        DistanceX int
        FromY int
        ToY int
        Next bool
    }{
        {0, 5, 5, true},
        {1, 5, 5, true},
        {-1, 5, 6, true},
        {1, 5, 4, true},
        {2, 5, 5, false},
        {0, 5, 7, false},
        {-2, 5, 6, false},
        {1, 5, 3, false},
    }

    for _, use := range cases {
        if nextToStack(use.DistanceX, use.FromY, use.ToY) != use.Next {
            test.Errorf("distance %v, row %v to %v: next to the stack should be %v", use.DistanceX, use.FromY, use.ToY, use.Next)
        }
    }
}

// a stack that lost all of its units in a fight has no place the camera could go to
func TestStackWithoutUnitsHasNoPlace(test *testing.T) {
    gone := playerlib.MakeUnitStack()
    if gone.X() != 0 || gone.Y() != 0 {
        test.Fatalf("a stack without units is expected to say 0, 0, it says %v, %v", gone.X(), gone.Y())
    }

    if stackHasPlace(gone) {
        test.Errorf("the camera would be sent to the stack without units, which is the corner of the map")
    }

    if stackHasPlace(nil) {
        test.Errorf("no stack has no place")
    }
}
