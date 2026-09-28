package combat

import (
    "math"
    "testing"
)

// a place is moved to the grid, by no more than half an art pixel, and a place on the grid stays
func TestNearestFieldPixel(test *testing.T) {
    for _, pixel := range []float64{3, 5, 7} {
        for _, start := range []float64{0, 2, -4, 13} {
            for step := 0; step < 200; step++ {
                position := 100 + float64(step) * 0.37
                at := nearestFieldPixel(position, start, pixel)

                steps := (at - start) / pixel
                if math.Abs(steps - math.Round(steps)) > 1e-9 {
                    test.Fatalf("pixel %v start %v: %v is moved to %v, between pixels", pixel, start, position, at)
                }
                if math.Abs(at - position) > pixel / 2 + 1e-9 {
                    test.Fatalf("pixel %v start %v: %v is moved to %v", pixel, start, position, at)
                }
                if again := nearestFieldPixel(at, start, pixel); again != at {
                    test.Fatalf("pixel %v start %v: %v on the grid is moved to %v", pixel, start, at, again)
                }
            }
        }
    }
}
