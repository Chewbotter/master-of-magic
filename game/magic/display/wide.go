package display

// Widescreen support.
//
// The game's screens are all built for a 320x200 picture. With widescreen on, the picture
// the game draws into becomes wider than 320 (the height stays 200) to match the shape of
// the window. A screen that has no widescreen layout is drawn in the middle 320 columns with
// black bars on both sides. A screen that does have one (the world map) says so through
// WideContent and draws across the whole width by itself.

import (
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
)

const DefaultWidescreen = true

// shape of the window in windowed widescreen mode
const WideAspectX = 16
const WideAspectY = 9

// widest picture the game will draw, in original pixels. 480 covers 21:9 monitors
const MaxLogicalWidth = 480

// width of the picture being drawn this frame, in original pixels. always even
var logicalWidth int = data.ScreenWidth

// set by the running game. returns true while the screen on top can fill the wide picture by itself
var WideContent func() bool

// called from Layout with the size of the window. returns the width of the picture in original pixels
func UpdateLayout(outsideWidth int, outsideHeight int) int {
    width := data.ScreenWidth

    if Current.Widescreen && outsideWidth > 0 && outsideHeight > 0 {
        width = data.ScreenHeight * outsideWidth / outsideHeight
        // an even width keeps both bars the same size
        width -= width % 2
        width = max(data.ScreenWidth, min(MaxLogicalWidth, width))
    }

    logicalWidth = width
    return width
}

// width of the picture in original pixels
func LogicalWidth() int {
    return logicalWidth
}

// how much wider than the original 320 the picture is, in original pixels
func ExtraWidth() int {
    return logicalWidth - data.ScreenWidth
}

// width of one black bar, in original pixels
func BarWidth() int {
    return ExtraWidth() / 2
}

// true when the screen on top is drawing across the whole wide picture
func IsWideContentActive() bool {
    return ExtraWidth() > 0 && WideContent != nil && WideContent()
}

// how far right the screen on top is drawn, in real screen pixels: the picture's margin in the
// window (see drawscale.go), plus a black bar when the screen has no widescreen layout
func ContentOffsetX() int {
    offset := frameX
    if ExtraWidth() > 0 && !IsWideContentActive() {
        offset += scale.Scale(BarWidth())
    }
    return offset
}

// how far down the screen on top is drawn, in real screen pixels
func ContentOffsetY() int {
    return frameY
}

// true when the screen on top is not drawn at the top left of the window
func HasContentOffset() bool {
    return ContentOffsetX() > 0 || ContentOffsetY() > 0
}

// position of the cursor as the screen on top understands it
func CursorPosition() (int, int) {
    x, y := ebiten.CursorPosition()
    return x - ContentOffsetX(), y - ContentOffsetY()
}

// same adjustment for a touch position
func AdjustPosition(x int, y int) (int, int) {
    return x - ContentOffsetX(), y - ContentOffsetY()
}

// width of the picture in a 16:9 window, in original pixels. even, so both bars are the same size
const WideArtWidth = 354

// size of the picture for a draw scale, in real pixels
func realSizeForScale(drawScale int, widescreen bool) (int, int) {
    width := data.ScreenWidth
    if widescreen {
        width = WideArtWidth
    }
    return width * drawScale, data.ScreenHeight * drawScale
}

// the window, in device independent pixels, that is just large enough to hold the picture at this
// draw scale in real pixels, so nothing is left over. factor is Windows display scaling
func windowSizeFor(drawScale int, widescreen bool, factor float64) (int, int) {
    width, height := realSizeForScale(drawScale, widescreen)
    return int(math.Ceil(float64(width) / factor - 1e-9)), int(math.Ceil(float64(height) / factor - 1e-9))
}

// size of the window for a given scale, following the widescreen setting, in device independent pixels
func sizeForScale(windowScale int, widescreen bool) (int, int) {
    if NativeDrawScale {
        return windowSizeFor(windowScale, widescreen, deviceScale())
    }

    height := data.ScreenHeight * windowScale
    if widescreen {
        width := height * WideAspectX / WideAspectY
        width -= width % 2
        return width, height
    }

    return data.ScreenWidth * windowScale, height
}
