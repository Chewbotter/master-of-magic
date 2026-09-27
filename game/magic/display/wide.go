package display

// Widescreen support.
//
// The game's screens are all built for a 320x200 picture. With widescreen on, the picture
// the game draws into becomes wider than 320 (the height stays 200) to match the shape of
// the window. A screen that has no widescreen layout is drawn in the middle 320 columns with
// black bars on both sides. A screen that does have one (the world map) says so through
// WideContent and draws across the whole width by itself.

import (
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

// how far right a screen without a widescreen layout is moved, in real screen pixels
func ContentOffsetX() int {
    if ExtraWidth() <= 0 || IsWideContentActive() {
        return 0
    }

    return scale.Scale(BarWidth())
}

// position of the cursor in real screen pixels, untouched. used to draw the cursor itself
func RawCursorPosition() (int, int) {
    return ebiten.CursorPosition()
}

// position of the cursor as the screen on top understands it
func CursorPosition() (int, int) {
    x, y := ebiten.CursorPosition()
    return x - ContentOffsetX(), y
}

// same adjustment for a touch position
func AdjustPosition(x int, y int) (int, int) {
    return x - ContentOffsetX(), y
}

// size of the window for a given scale, following the widescreen setting
func sizeForScale(windowScale int, widescreen bool) (int, int) {
    height := data.ScreenHeight * windowScale
    if widescreen {
        width := height * WideAspectX / WideAspectY
        width -= width % 2
        return width, height
    }

    return data.ScreenWidth * windowScale, height
}
