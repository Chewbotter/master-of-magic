package game

// The cursor glides the same way the world map does during a pan, so the two move in step instead
// of the map gliding and the cursor stepping. Outside of a pan the cursor is drawn where the mouse
// is, with no delay.

import (
    "math"
    "time"

    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/game/magic/mouse"
)

type cursorGlide struct {
    active bool
    x float64
    y float64
    lastTime time.Time
}

var glide cursorGlide

func init() {
    mouse.SmoothPosition = glidingCursorPosition
}

func startCursorGlide(x int, y int) {
    glide = cursorGlide{
        active: true,
        x: float64(x),
        y: float64(y),
        lastTime: time.Now(),
    }
}

// where to draw the cursor this frame. false once the glide has caught up with the mouse
func glidingCursorPosition(x int, y int) (float64, float64, bool) {
    glideX, glideY, ok := glideCursorAt(x, y, time.Now())
    if ok && !display.BlendBetweenPixels() {
        // at the native draw scale a screen pixel is fine enough to move crisp
        return math.Round(glideX), math.Round(glideY), true
    }
    return glideX, glideY, ok
}

func glideCursorAt(x int, y int, now time.Time) (float64, float64, bool) {
    if !glide.active {
        return 0, 0, false
    }

    frameTime := min(PanMaxFrameTime, max(0, now.Sub(glide.lastTime).Seconds()))
    glide.lastTime = now

    step := 1 - math.Exp(-frameTime / PanSmoothTime)
    glide.x += (float64(x) - glide.x) * step
    glide.y += (float64(y) - glide.y) * step

    remainingX := float64(x) - glide.x
    remainingY := float64(y) - glide.y
    if !panning && math.Abs(remainingX) < PanSettleDistance && math.Abs(remainingY) < PanSettleDistance {
        glide.active = false
        return 0, 0, false
    }

    return glide.x, glide.y, true
}
