package combat

import (
    "testing"
)

// the wind up frame takes as many places of a long strike as it shows longer
func TestWindUpShowsLonger(test *testing.T) {
    if strikeWindUpFrame != 4 || strikeWindUpHold != 2 {
        test.Fatalf("the wind up is frame %v for %v, the user asked for frame 4 for 2", strikeWindUpFrame, strikeWindUpHold)
    }
}
