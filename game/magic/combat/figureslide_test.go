package combat

import (
    "testing"
)

// back on the wind up, forward through the rest of the swing, at its place on the strike frame
func TestStrikeSlideByFrame(test *testing.T) {
    if strikeSlideAt(3) != 0 {
        test.Errorf("frame 3: %v, should be where the figure stands", strikeSlideAt(3))
    }
    if strikeSlideAt(4) >= 0 {
        test.Errorf("frame 4: %v, should be back", strikeSlideAt(4))
    }
    if !(strikeSlideAt(5) > 0 && strikeSlideAt(6) > strikeSlideAt(5)) {
        test.Errorf("frames 5 and 6: %v and %v, should go forward", strikeSlideAt(5), strikeSlideAt(6))
    }
    if strikeSlideAt(6) != strikeSlideForward {
        test.Errorf("frame 6: %v, should be all of the way forward", strikeSlideAt(6))
    }
}
