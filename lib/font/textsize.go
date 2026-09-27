package font

// Smaller text that stays sharp. FontOptions.TextSize makes the letters smaller than the art around
// them (0.5 is about half size) while their position stays in art pixels. The letters are drawn at a
// whole number of screen pixels per font pixel, so a size is rounded to the nearest one that is
// sharp: at a draw scale of 7, half size is 4 screen pixels per font pixel.

import (
    "math"
)

// the size of each font pixel in screen pixels
func glyphScale(scale float64, textSize float64) float64 {
    if textSize <= 0 || textSize >= 1 {
        return scale
    }

    return max(1, math.Round(scale * textSize))
}

// how large the letters come out compared to the art, after rounding to whole screen pixels.
// wrapping and line spacing use this, see CreateWrappedText and RenderWrapped
func RelativeTextSize(scale float64, textSize float64) float64 {
    if scale <= 0 {
        return 1
    }
    return glyphScale(scale, textSize) / scale
}

// the letter size for these options, relative to the art
func (options FontOptions) relativeSize() float64 {
    scale := options.Scale
    if scale == 0 {
        scale = 1
    }
    return RelativeTextSize(scale, options.TextSize)
}
