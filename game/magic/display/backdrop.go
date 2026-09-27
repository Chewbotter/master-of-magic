package display

// A screen that has no widescreen layout is drawn 320 wide in the middle, with black bars. Such a
// screen can still fill the bars with a backdrop: a picture as wide as the window that is drawn
// beneath it. The battlefield does this: the field spans the window, the combat bar and everything
// that reads the mouse stay in the middle 320 columns.

import (
    "github.com/hajimehoshi/ebiten/v2"
)

// set by the screen on top while it has a backdrop. draws it on a picture LogicalWidth() wide, whose
// middle 320 columns are where the screen itself is drawn
var WideBackdrop func(screen *ebiten.Image)

// true while the screen on top draws a backdrop beneath itself
func IsWideBackdropActive() bool {
    return WideBackdrop != nil && ExtraWidth() > 0 && !IsWideContentActive()
}
