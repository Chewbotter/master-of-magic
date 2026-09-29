package combat

import (
    "testing"
)

// a pixel of the ground has the same place in the pattern wherever the light is drawn
func TestLightPatternIsOnTheGround(test *testing.T) {
    const pixel = 7.0
    // the ground starts at pixel 35 of the screen, the layer at -7: 6 pixels of the layer before it
    field, start := 35.0, -7.0
    ground := 40

    want := -1
    for at := -9; at <= 30; at++ {
        phase := lightPhase(float64(at), field, start, pixel)
        if phase < 0 || phase > 3 {
            test.Fatalf("phase %v of a picture at %v", phase, at)
        }

        // the pixel of the picture that lies on the pixel of the ground, which is on the pixel
        // ground + 6 of the layer
        inPicture := ground + 6 - at
        if inPicture < 0 {
            continue
        }
        place := (inPicture + phase) % 4
        if want == -1 {
            want = place
        }
        if place != want {
            test.Errorf("picture at %v: the pixel of the ground has place %v of the pattern, want %v", at, place, want)
        }
    }
}
