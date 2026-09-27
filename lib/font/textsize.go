package font

// Smaller text that stays sharp.
//
// Text comes in a few NAMED sizes, relative to the art, and every change picks one of them, never a
// raw number, so text of the same kind is the same size all through the game. Retuning a name changes
// every place that uses it.
//
// The letters are drawn at a whole number of screen pixels per font pixel, so a size is rounded to the
// nearest one that is sharp: at a draw scale of 7, Compact is 6 screen pixels per font pixel and Small
// is 4. At small draw scales two names can round to the same size.
//
// A size is given per call in FontOptions.TextSize, or for a whole stretch of drawing with UseTextSize.
// Positions always stay in art pixels.

import (
    "math"
)

// the named sizes
const TextSizeFull = 1.0
const TextSizeCompact = 0.8
const TextSizeSmall = 0.6

// the size for text drawn without its own TextSize, see UseTextSize
var defaultTextSize = TextSizeFull

// draws all text that does not set its own size at this size, until the returned function is called:
// defer font.UseTextSize(font.TextSizeCompact)()
func UseTextSize(size float64) func() {
    old := defaultTextSize
    defaultTextSize = size
    return func() {
        defaultTextSize = old
    }
}

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

// the text size these options draw at
func (options FontOptions) textSize() float64 {
    if options.TextSize > 0 {
        return options.TextSize
    }
    return defaultTextSize
}

// the letter size for these options, relative to the art
func (options FontOptions) relativeSize() float64 {
    scale := options.Scale
    if scale == 0 {
        scale = 1
    }
    return RelativeTextSize(scale, options.textSize())
}

// the height of a line at the current size (see UseTextSize), in art pixels
func (font *Font) HeightAt(scale float64) float64 {
    return float64(font.Height()) * RelativeTextSize(scale, defaultTextSize)
}
