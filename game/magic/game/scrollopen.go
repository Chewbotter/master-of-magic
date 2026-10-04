package game

// A click or a key while a scroll of events unrolls opens it at once; only one when it is open
// closes it (user 2026-10-02: a City Growth scroll was gone before its line could be read: it is
// a thin strip with its text behind the rolls while it unrolls, and a click then closed it). The
// original reads no input before the scroll is open.

import (
    "github.com/kazzmir/master-of-magic/game/magic/util"
)

// false: a click while the scroll unrolls closes it, as upstream
var ScrollOpensOnClick = true

// the animation of the scroll at its last picture and the paper as long as it is then
func finishScrollOpening(animation *util.Animation, length *int) {
    for animation.Next() {
        *length += 10
    }
}
