package camera

// Zoom levels: whole numbers of screen pixels per art pixel, one apart, from the draw scale down,
// at most ZoomLevelCount of them and never below 1. At draw scale 5 that is 5x, 4x, 3x, 2x and 1x,
// at 7 it is 7x down to 3x, and at 3 it is 3x, 2x and 1x.

import (
    "math"
)

// the camera zoom that shows art pixels as this many screen pixels at the given draw scale
func ZoomForPixels(pixels float64, drawScale float64) int {
    if drawScale <= 0 {
        return ZoomMax
    }
    return int(math.Round(float64(ZoomMax) * pixels / drawScale))
}

// how many zoom levels there are, one screen pixel per art pixel apart, starting at the draw scale
const ZoomLevelCount = 5

// the zoom levels for a draw scale, closest first
func ZoomLevels(drawScale float64) []int {
    levels := []int{ZoomMax}
    top := math.Floor(drawScale + 1e-9)
    for pixels := top; pixels >= max(1, top - ZoomLevelCount + 1); pixels-- {
        zoom := ZoomForPixels(pixels, drawScale)
        if zoom < levels[len(levels) - 1] {
            levels = append(levels, zoom)
        }
    }
    return levels
}

// the next level closer (direction above 0) or farther (direction below 0) than the current zoom.
// returns the current zoom when there is no level that way
func NextZoom(current int, drawScale float64, direction int) int {
    levels := ZoomLevels(drawScale)

    if direction > 0 {
        for index := len(levels) - 1; index >= 0; index-- {
            if levels[index] > current {
                return levels[index]
            }
        }
        return current
    }

    for _, level := range levels {
        if level < current {
            return level
        }
    }
    return current
}

// the level nearest the current zoom
func NearestZoom(current int, drawScale float64) int {
    best := ZoomMax
    for _, level := range ZoomLevels(drawScale) {
        if math.Abs(float64(level - current)) < math.Abs(float64(best - current)) {
            best = level
        }
    }
    return best
}
