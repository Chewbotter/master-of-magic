package camera

import (
    "testing"
)

// screen pixels per art pixel for each level
func levelPixels(drawScale float64) []float64 {
    var out []float64
    for _, level := range ZoomLevels(drawScale) {
        out = append(out, drawScale * float64(level) / float64(ZoomMax))
    }
    return out
}

func TestZoomLevelsAreWholePixels(test *testing.T) {
    cases := map[float64][]float64{
        5: {5, 4, 3, 2, 1},
        3: {3, 2, 1},
        4: {4, 3, 2, 1},
        6: {6, 5, 4, 3, 2},
        7: {7, 6, 5, 4, 3},
        1: {1},
    }

    for drawScale, expected := range cases {
        got := levelPixels(drawScale)
        if len(got) != len(expected) {
            test.Fatalf("draw scale %v: levels %v, expected %v", drawScale, got, expected)
        }
        for index := range expected {
            if got[index] != expected[index] {
                test.Fatalf("draw scale %v: levels %v, expected %v", drawScale, got, expected)
            }
        }
    }
}

func TestNextZoomSteps(test *testing.T) {
    zoom := ZoomMax
    var steps []int
    for range 6 {
        next := NextZoom(zoom, 5, -1)
        steps = append(steps, next)
        zoom = next
    }
    // five levels: four steps out, then it stays at 1x
    if steps[3] != ZoomForPixels(1, 5) || steps[4] != steps[3] {
        test.Fatalf("zooming out at draw scale 5 went %v", steps)
    }

    for range 6 {
        zoom = NextZoom(zoom, 5, 1)
    }
    if zoom != ZoomMax {
        test.Fatalf("zooming back in ended at %v, expected %v", zoom, ZoomMax)
    }

    // a zoom left between levels by a change of draw scale steps to the neighboring levels
    between := ZoomForPixels(3.5, 5)
    if NextZoom(between, 5, 1) != ZoomForPixels(4, 5) || NextZoom(between, 5, -1) != ZoomForPixels(3, 5) {
        test.Fatalf("zoom between levels did not step to its neighbors")
    }
    if NearestZoom(ZoomForPixels(3.2, 5), 5) != ZoomForPixels(3, 5) {
        test.Fatalf("nearest level to 3.2x was not 3x")
    }
}
