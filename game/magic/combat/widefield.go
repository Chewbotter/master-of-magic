package combat

// The battlefield in widescreen: the field spans the whole width of the window, the combat bar and
// all windows stay in the middle 320 columns where they are, see display/backdrop.go.
//
// The screen is drawn in two passes. The field goes on the wide picture, moved right by the width
// of a bar so that it lines up with the middle. Everything else goes on the usual picture, which is
// left transparent where the field shows. Positions of the mouse and of the tiles it picks do not
// change: they are relative to the middle 320 columns and can be less than 0 or more than 320.

import (
    "github.com/kazzmir/master-of-magic/game/magic/display"

    "github.com/hajimehoshi/ebiten/v2"
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
