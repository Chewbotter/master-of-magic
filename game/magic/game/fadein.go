package game

// The world map comes back out of the black. A screen that fills the window, as the book of
// research, fades to black when it closes; the map behind it was there at once after that, which
// looked like a pop (user, 2026-09-29).

import (
    "image/color"

    "github.com/kazzmir/master-of-magic/lib/coroutine"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/vector"
)

// false: the map is there at once
const FadeInFromBlack = true
// seconds the map takes to come out of the black
const fadeInFromBlackTime = 0.2

// shows what is on the screen now under a black that goes away. call it when the screen that
// faded to black is off the stack of drawers
func (game *Game) fadeInFromBlack(yield coroutine.YieldFunc) {
    under := game.LastDrawer()
    if !FadeInFromBlack || under == nil {
        return
    }

    ticks := max(1, int(fadeInFromBlackTime * float64(max(1, ebiten.TPS()))))
    black := 1.0

    game.PushDrawer(func (screen *ebiten.Image){
        under(screen)
        bounds := screen.Bounds()
        vector.FillRect(screen, 0, 0, float32(bounds.Dx()), float32(bounds.Dy()), color.NRGBA{A: uint8(black * 255)}, false)
    })
    // the map keeps its wide layout under it, see wide.go
    unmark := game.markOverlayDrawer()
    defer unmark()
    defer game.PopDrawer()

    for tick := range ticks {
        // slowly at first
        part := float64(tick + 1) / float64(ticks)
        black = 1 - part * part
        if yield() != nil {
            return
        }
    }
}
