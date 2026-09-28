package combat

import (
    "math"
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/scale"
)

// the tile picked for a point on the screen is the tile whose diamond is drawn there, at every zoom
// level and position of the camera
// a cursor is drawn on the pixels of the field, and never further than half of one from where the
// mouse has it
func TestCursorOnFieldPixels(test *testing.T) {
    for _, pixel := range []float64{3, 5, 7} {
        for _, start := range []float64{0, 2, -3, 11} {
            for position := 100.0; position < 140; position++ {
                at := cursorOnFieldPixel(position, 40, start, pixel, 5)

                steps := (at - 40 - start) / pixel
                if steps != math.Round(steps) {
                    test.Errorf("pixel %v start %v mouse %v: drawn at %v, between pixels of the field", pixel, start, position, at)
                }

                exact := position - 5 * pixel
                if math.Abs(at - exact) > pixel / 2 {
                    test.Errorf("pixel %v start %v mouse %v: drawn at %v, %v from where the mouse has it", pixel, start, position, at, at - exact)
                }
            }
        }
    }
}

func TestCursorPicksTheTileDrawnUnderIt(test *testing.T) {
    saved := scale.ScaleAmount
    defer func() {
        scale.ScaleAmount = saved
    }()
    scale.ScaleAmount = 7

    combat := &CombatScreen{}

    for _, level := range []int{7, 6, 5, 4, 3} {
        for _, pan := range [][2]float64{{0, 0}, {133, -57}, {-260, 90}} {
            combat.Camera.SetView(level, pan[0], pan[1])
            combat.syncCamera()
            matrix := combat.GetCameraMatrix()

            for x := range BattlefieldWidth {
                for y := range BattlefieldHeight {
                    middleX, middleY := matrix.Apply(float64(x), float64(y))

                    // points well inside the diamond, in art pixels at the size it is drawn
                    size := combat.CameraScale
                    for _, offset := range [][2]float64{{0, 0}, {-12, 0}, {12, 0}, {0, -6}, {0, 6}, {-6, -3}, {6, 3}} {
                        screenX := (middleX + offset[0] * size) * scale.ScaleAmount
                        screenY := (middleY + offset[1] * size) * scale.ScaleAmount

                        tileX, tileY := combat.ScreenToTile(screenX, screenY)
                        gotX := int(math.Round(tileX))
                        gotY := int(math.Round(tileY))
                        if gotX != x || gotY != y {
                            test.Fatalf("level %v pan %v: point %v of tile %v,%v picked tile %v,%v", level, pan, offset, x, y, gotX, gotY)
                        }
                    }
                }
            }
        }
    }
}
