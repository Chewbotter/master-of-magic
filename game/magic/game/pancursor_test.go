package game

import (
    "testing"
    "time"
)

// the mouse reports a new whole pixel position 60 times a second while frames are drawn 120 times a
// second. the drawn cursor must move on every frame of the drag, never stand still for a frame and
// then jump, and after the drag it must land exactly on the mouse and hand back to normal drawing
func TestCursorGlideMovesEveryFrame(test *testing.T) {
    start := time.Unix(1000, 0)
    frame := time.Second / 120

    panning = true
    defer func() {
        panning = false
        glide = cursorGlide{}
    }()

    startCursorGlide(100, 100)
    glide.lastTime = start

    mouseX := 100
    lastX := 100.0
    now := start
    for index := 1; index <= 60; index++ {
        now = now.Add(frame)
        // a slow drag: one pixel per tick, a tick every second frame
        if index % 2 == 0 {
            mouseX += 1
        }

        x, _, ok := glideCursorAt(mouseX, 100, now)
        if !ok {
            test.Fatalf("frame %v: the glide stopped during the drag", index)
        }
        if index > 1 && x <= lastX {
            test.Fatalf("frame %v: the cursor stood still at %.3f", index, x)
        }
        if x > float64(mouseX) {
            test.Fatalf("frame %v: the cursor overshot the mouse, %.3f past %v", index, x, mouseX)
        }
        lastX = x
    }

    // release: the cursor settles on the mouse and normal drawing takes over
    panning = false
    settled := false
    for index := 0; index < 60; index++ {
        now = now.Add(frame)
        x, _, ok := glideCursorAt(mouseX, 100, now)
        if !ok {
            settled = true
            break
        }
        if x > float64(mouseX) {
            test.Fatalf("after release: the cursor overshot the mouse, %.3f past %v", x, mouseX)
        }
    }

    if !settled {
        test.Fatalf("the cursor did not settle within half a second of the release")
    }
}
