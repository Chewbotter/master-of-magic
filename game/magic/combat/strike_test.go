package combat

import (
    "testing"
)

// the wind up frame takes as many places of a long strike as it shows longer
func TestWindUpShowsLonger(test *testing.T) {
    if strikeWindUpFrame != 4 {
        test.Fatalf("the wind up is frame %v, the user asked for frame 4", strikeWindUpFrame)
    }

    // it shows while the figure is back and while it rushes forward: longer than any other frame
    // but the one it stays forward on
    steps := swingSteps([]int{4, 5})
    shown := make(map[int]float64)
    for _, step := range steps {
        shown[step.Frame] += step.Time
    }
    if shown[4] <= swingLandTime {
        test.Errorf("the wind up shows for %v", shown[4])
    }
}
