package combat

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/display"
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

// the strike of before the swing of steps: where the figure is at the end of each frame
func TestClassicStrikeSlide(test *testing.T) {
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
}

// its frames: 3 and the ones after it, 4 twice
func TestClassicStrikeFrames(test *testing.T) {
    for _, each := range []struct{Extra []int; Frames []int}{
        {nil, nil},
        {[]int{4}, []int{3, 4, 4}},
        {[]int{4, 5}, []int{3, 4, 4, 5}},
        {[]int{4, 5, 6}, []int{3, 4, 4, 5, 6}},
    } {
        if got := longStrikeFramesOf(each.Extra); !sameFrames(got, each.Frames) {
            test.Errorf("with %v: %v, should be %v", each.Extra, got, each.Frames)
        }
    }
}

// the setting is on unless it was turned off
func TestSingleStrikesSetting(test *testing.T) {
    if !display.MakeDefault().SingleStrikes() {
        test.Errorf("single strikes are off in new settings")
    }

    settings := display.Settings{}
    if !settings.SingleStrikes() {
        test.Errorf("single strikes are off in settings that do not say")
    }
    settings.RepeatedStrikes = true
    if settings.SingleStrikes() {
        test.Errorf("single strikes are on after they were turned off")
    }
}
