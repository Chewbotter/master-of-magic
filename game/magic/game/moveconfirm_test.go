package game

import (
    "testing"
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
