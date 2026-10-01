package combat

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/mod"
)

// a shot goes through all the frames of a strike in their order, the missile leaves on the frame a
// swing lands on, and the figure does not move
func TestShotSteps(test *testing.T) {
    for _, extra := range [][]int{{4}, {4, 5}, {4, 5, 6}} {
        steps := shotSteps(extra)
        var seen []int
        for _, step := range steps {
            if step.From != 0 || step.To != 0 {
                test.Fatalf("frames %v: the figure moves in %+v", extra, step)
            }
            if len(seen) == 0 || seen[len(seen) - 1] != step.Frame {
                seen = append(seen, step.Frame)
            }
        }

        // 3, the frames after it in their order, and 3 again
        want := append(append([]int{mod.FrameStrike}, extra...), mod.FrameStrike)
        if len(extra) == 1 {
            // with 4 alone it lands on the game's strike frame, which is also the one it ends on
            want = []int{mod.FrameStrike, 4, mod.FrameStrike}
        }
        if len(seen) != len(want) {
            test.Fatalf("frames %v: shows %v", extra, seen)
        }
        for index := range want {
            if seen[index] != want[index] {
                test.Fatalf("frames %v: shows %v", extra, seen)
            }
        }

        // the missile leaves when the frame after the draw shows
        frame, _ := swingAt(steps, swingLands(steps) + 0.01, 1)
        wantFrame := mod.FrameStrike
        if len(extra) > 1 {
            wantFrame = extra[1]
        }
        if frame != wantFrame {
            test.Fatalf("frames %v: it looses on frame %v", extra, frame)
        }
    }
}

// a unit with the frames of the game only stands, shows its strike frame where the missile is
// loosed and stands again: every unit that shoots or casts shows frames
func TestShotOfGameFrames(test *testing.T) {
    steps := shotSteps(nil)
    var seen []int
    for _, step := range steps {
        if step.From != 0 || step.To != 0 {
            test.Fatalf("the figure moves in %+v", step)
        }
        if len(seen) == 0 || seen[len(seen) - 1] != step.Frame {
            seen = append(seen, step.Frame)
        }
    }
    if len(seen) != 3 || seen[0] != figureStandFrame || seen[1] != mod.FrameStrike || seen[2] != figureStandFrame {
        test.Fatalf("shows %v", seen)
    }
    if frame, _ := swingAt(steps, swingLands(steps) + 0.01, 1); frame != mod.FrameStrike {
        test.Fatalf("looses on frame %v", frame)
    }
    if !showsShot(&ArmyUnit{}) {
        test.Fatalf("a unit without frames of its own shows no shot")
    }
}
