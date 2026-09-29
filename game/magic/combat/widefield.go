package combat

// The battlefield in widescreen: the field spans the whole width of the window, the combat bar and
// all windows stay in the middle 320 columns where they are, see display/backdrop.go.
//
// The screen is drawn in two passes. The field goes on the wide picture, moved right by the width
// of a bar so that it lines up with the middle. Everything else goes on the usual picture, which is
// left transparent where the field shows. Positions of the mouse and of the tiles it picks do not
// change: they are relative to the middle 320 columns and can be less than 0 or more than 320.

import (
    "image/color"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/vector"
)

type drawPass int
const (
    // the whole screen on one picture
    drawPassAll drawPass = iota
    drawPassField
    drawPassInterface
)

func (combat *CombatScreen) drawsField() bool {
    return combat.pass != drawPassInterface
}

func (combat *CombatScreen) drawsInterface() bool {
    return combat.pass != drawPassField
}

// the field on a picture as wide as the window
func (combat *CombatScreen) drawWideField(screen *ebiten.Image) {
    combat.pass = drawPassField
    combat.fieldShift = float64(display.BarWidth())

    combat.Drawer(screen)

    // the dark behind the spellbook, left and right of the middle where the book draws its own
    dark := spellbook.TakeCastBookDark()
    if dark > 0 {
        bar := float32(scale.Scale(display.BarWidth()))
        middle := float32(scale.Scale(data.ScreenWidth))
        height := float32(screen.Bounds().Dy())
        shade := color.RGBA{A: uint8(dark * 255)}
        vector.FillRect(screen, 0, 0, bar, height, shade, false)
        vector.FillRect(screen, bar + middle, 0, float32(screen.Bounds().Dx()) - bar - middle, height, shade, false)
    }

    combat.pass = drawPassAll
    combat.fieldShift = 0
    combat.syncCamera()
}

// from now on the field fills the width of the window. call the result when the battle is over
func (combat *CombatScreen) UseWideField() func() {
    combat.wideField = true
    display.WideBackdrop = combat.drawWideField

    return func() {
        combat.wideField = false
        display.WideBackdrop = nil
    }
}
