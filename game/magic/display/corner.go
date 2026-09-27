package display

// Development only: keeps automated runs out of the way of whatever is in the middle of the screen.
// A normal launch never calls this and stays centered.

import (
    "github.com/hajimehoshi/ebiten/v2"
)

// window scale for a run placed in the corner. 1 is the original 320x200
const CornerWindowScale = 1
// space left between the window and the right edge of the monitor
const CornerMarginX = 8
// space left between the window and the bottom edge of the monitor, enough to clear the taskbar
const CornerMarginY = 56

// development: when above 0, the corner window is this many pixels wide, to test other screen shapes
var CornerWidthOverride int

// make the window small and put it in the lower right corner of the monitor.
// the shape of the window still follows the widescreen setting, so captures are unchanged
func PlaceInCorner() {
    ebiten.SetFullscreen(false)

    width, height := sizeForScale(CornerWindowScale, Current.Widescreen)
    if CornerWidthOverride > 0 {
        width = CornerWidthOverride
    }
    ebiten.SetWindowSize(width, height)

    monitor := ebiten.Monitor()
    if monitor == nil {
        return
    }

    monitorWidth, monitorHeight := monitor.Size()
    if monitorWidth <= 0 || monitorHeight <= 0 {
        return
    }

    ebiten.SetWindowPosition(max(0, monitorWidth - width - CornerMarginX), max(0, monitorHeight - height - CornerMarginY))
}
