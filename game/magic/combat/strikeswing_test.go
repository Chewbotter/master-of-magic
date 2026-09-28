package combat

import (
    "math"
    "testing"
)

// the frames and places of a swing in the order they come, each once, at the middle of each step
func swingOrder(steps []swingStep) ([]int, []float64) {
    var frames []int
    var places []float64
    time := 0.0
    for _, step := range steps {
        frame, place := swingAt(steps, time + step.Time / 2)
        frames = append(frames, frame)
        places = append(places, place)
        time += step.Time
    }
    return frames, places
}

func sameFrames(a []int, b []int) bool {
    if len(a) != len(b) {
        return false
    }
    for index := range a {
        if a[index] != b[index] {
            return false
        }
    }
    return true
}

// the steps the user asked for, for a figure with the frames 3, 4 and 5
func TestSwingWithThreeFrames(test *testing.T) {
    steps := swingSteps([]int{4, 5})
    frames, places := swingOrder(steps)

    if !sameFrames(frames, []int{3, 3, 4, 4, 5, 3}) {
        test.Fatalf("frames %v", frames)
    }

    back := -strikeSlideBack
    forward := strikeSlideForward

    // stands, is on its way back, is back, is on its way forward, is forward, is on its way home
    if places[0] != 0 {
        test.Errorf("it stands at %v", places[0])
    }
    if places[1] >= 0 || places[1] <= back {
        test.Errorf("on its way back it is at %v", places[1])
    }
    if places[2] != back {
        test.Errorf("winding up it is at %v", places[2])
    }
    if places[3] <= back || places[3] >= forward {
        test.Errorf("rushing it is at %v", places[3])
    }
    if places[4] != forward {
        test.Errorf("where its blow lands it is at %v", places[4])
    }
    if places[5] <= 0 || places[5] >= forward {
        test.Errorf("on its way home it is at %v", places[5])
    }

    // it stays furthest forward longer than it takes to get there
    if steps[4].Time <= steps[3].Time || steps[4].Time <= swingLandTime {
        test.Errorf("it stays forward for %v, the rush takes %v", steps[4].Time, steps[3].Time)
    }

    // after the swing and before it the figure stands
    frame, place := swingAt(steps, swingTime(steps))
    if frame != 3 || place != 0 {
        test.Errorf("after the swing: frame %v at %v", frame, place)
    }
    frame, place = swingAt(steps, swingTime(steps) * 3)
    if frame != 3 || place != 0 {
        test.Errorf("long after the swing: frame %v at %v", frame, place)
    }
    frame, place = swingAt(steps, -1)
    if frame != 3 || place != 0 {
        test.Errorf("before the swing: frame %v at %v", frame, place)
    }

    // the blow lands when the figure gets furthest forward
    lands := swingLands(steps)
    frame, place = swingAt(steps, lands)
    if frame != 5 || place != forward {
        test.Errorf("when the blow lands: frame %v at %v", frame, place)
    }
    frame, _ = swingAt(steps, lands - 0.01)
    if frame != 4 {
        test.Errorf("just before the blow lands: frame %v", frame)
    }
    _, place = swingAt(steps, swingTime(steps) - 0.0001)
    if math.Abs(place) > 0.01 {
        test.Errorf("at the end of the swing it is at %v", place)
    }
}

// a figure with other frames does the same steps with what it has
func TestSwingWithOtherFrames(test *testing.T) {
    // 5 and then 6 while it is forward, then 3 and back
    steps := swingSteps([]int{4, 5, 6})
    frames, places := swingOrder(steps)
    if !sameFrames(frames, []int{3, 3, 4, 4, 5, 6, 3}) {
        test.Errorf("with 4, 5 and 6: %v", frames)
    }
    if places[4] != strikeSlideForward || places[5] != strikeSlideForward {
        test.Errorf("on 5 it is at %v, on 6 at %v", places[4], places[5])
    }
    // it stays on 6, not on 5
    if steps[5].Frame != 6 || steps[5].Time <= steps[4].Time {
        test.Errorf("frame %v shows for %v, frame %v for %v", steps[4].Frame, steps[4].Time, steps[5].Frame, steps[5].Time)
    }

    frames, _ = swingOrder(swingSteps([]int{4}))
    if !sameFrames(frames, []int{3, 3, 4, 4, 3, 3}) {
        test.Errorf("with 4: %v", frames)
    }

    frames, places = swingOrder(swingSteps(nil))
    if !sameFrames(frames, []int{1, 1, 1, 1, 3, 1}) {
        test.Errorf("with the frames of the game: %v", frames)
    }
    if places[4] != strikeSlideForward {
        test.Errorf("the strike of the game lands at %v", places[4])
    }
}

// it never jumps: from one moment to the next it moves a little
func TestSwingIsSmooth(test *testing.T) {
    for _, extra := range [][]int{nil, {4}, {4, 5}, {4, 5, 6}} {
        steps := swingSteps(extra)
        total := swingTime(steps)

        _, last := swingAt(steps, 0)
        for step := 1; step <= 1000; step++ {
            _, place := swingAt(steps, total * float64(step) / 1000)
            if math.Abs(place - last) > 0.12 {
                test.Fatalf("frames %v: from %v to %v at step %v", extra, last, place, step)
            }
            last = place
        }
    }
}
