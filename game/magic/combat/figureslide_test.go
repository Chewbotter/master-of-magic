package combat

import (
    "testing"
)

func slideEnds(test *testing.T, name string, frames []int, want []float64) {
    got := strikeSlideEnds(frames)
    if len(got) != len(want) {
        test.Fatalf("%v: %v places for %v frames", name, len(got), len(want))
    }
    for index := range want {
        if got[index] != want[index] {
            test.Errorf("%v: frame %v ends at %v, should be %v. all: %v", name, frames[index], got[index], want[index], got)
        }
    }
}

// where the figure is at the end of each frame of its swing
func TestStrikeSlide(test *testing.T) {
    back := -strikeSlideBack
    forward := strikeSlideForward
    middle := (back + forward) / 2

    // back on 3, hold on 4, forward on 5 and 6
    slideEnds(test, "3 4 4 5 6", []int{3, 4, 4, 5, 6}, []float64{back, back, back, middle, forward})
    slideEnds(test, "3 4 4 5", []int{3, 4, 4, 5}, []float64{back, back, back, forward})
    // nothing after the wind up: back on 4, forward on 3
    slideEnds(test, "3 4 4", []int{3, 4, 4}, []float64{forward, back, back})
    // the frames of the game: back on 1, forward on 3
    slideEnds(test, "3 1", figureAttackFrames, []float64{forward, back})

    // the frame the blow lands on shows longer: the figure stays forward
    slideEnds(test, "3 4 4 5 6 6", []int{3, 4, 4, 5, 6, 6}, []float64{back, back, back, middle, forward, forward})
    slideEnds(test, "3 3 4 4", []int{3, 3, 4, 4}, []float64{forward, forward, back, back})
    slideEnds(test, "3 3 1", []int{3, 3, 1}, []float64{forward, forward, back})
}

// the frame the blow lands on is in the swing as often as it is held
func TestFollowHold(test *testing.T) {
    same := func(name string, got []int, want []int) {
        if len(got) != len(want) {
            test.Errorf("%v: %v, should be %v", name, got, want)
            return
        }
        for index := range want {
            if got[index] != want[index] {
                test.Errorf("%v: %v, should be %v", name, got, want)
                return
            }
        }
    }

    same("3 4 4 5 6", withFollowHold([]int{3, 4, 4, 5, 6}), []int{3, 4, 4, 5, 6, 6})
    same("3 4 4 5", withFollowHold([]int{3, 4, 4, 5}), []int{3, 4, 4, 5, 5})
    same("3 4 4", withFollowHold([]int{3, 4, 4}), []int{3, 3, 4, 4})
    same("3 1", gameStrikeFrames(), []int{3, 3, 1})
}
