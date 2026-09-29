package spellbook

// The turn of a page and the tabs in the outer corners of the pages.
//
// The tabs that turn the pages stay where they are while a leaf turns, so the rim of the leaf
// was seen moving under them at the start of a turn and at its end. The leaf now starts far
// enough into its turn to be clear of the tab it leaves, and is put down at once when it reaches
// the tab of the other side (user, 2026-09-29: "make the first few frames of the page turn come
// out faster so that the first frame clears the tab completely").

import (
    "image"
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/mod"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    uilib "github.com/kazzmir/master-of-magic/game/magic/ui"

    "github.com/hajimehoshi/ebiten/v2"
)

// with the mouse over it a tab is lifted by this many art pixels, and leaves a shadow this dark,
// 0 to 1, where it lay. there is no look for a tab that is pressed.
// KEEP IN STEP with mod.TabHoverLift and mod.TabHoverShadow, which the export makes the pictures with
const pageTabLift = mod.TabHoverLift
const pageTabShadow = mod.TabHoverShadow

// the pictures of the tabs under the mouse of the replacement folder, by name. nil for none
var hoverTabs = make(map[string]*ebiten.Image)

// the pictures are read again when they are asked for next
func forgetHoverTabs() {
    clear(hoverTabs)
}

func hoverTab(name string) *ebiten.Image {
    picture, ok := hoverTabs[name]
    if ok {
        return picture
    }

    read := mod.ReadMarker(mod.TabHoverName(name))
    if read != nil {
        picture = ebiten.NewImageFromImage(read)
    }
    hoverTabs[name] = picture
    return picture
}

// draws a tab that turns the pages at a place in art pixels. name is the one of mod.Tab...
func drawPageTab(screen *ebiten.Image, tab *ebiten.Image, name string, place image.Point, alpha float32, over bool) {
    if tab == nil {
        return
    }

    var options ebiten.DrawImageOptions
    options.ColorScale.ScaleAlpha(alpha)
    options.GeoM.Translate(float64(place.X), float64(place.Y))

    if over {
        painted := hoverTab(name)
        if painted != nil {
            // its lower left corner on the lower left corner of the tab, so a picture that is
            // higher reaches up
            options.GeoM.Translate(0, float64(tab.Bounds().Dy() - painted.Bounds().Dy()))
            scale.DrawScaled(screen, painted, &options)
            return
        }

        shadow := options
        shadow.ColorScale.Scale(0, 0, 0, pageTabShadow)
        scale.DrawScaled(screen, tab, &shadow)

        options.GeoM.Translate(0, -pageTabLift)
    }

    scale.DrawScaled(screen, tab, &options)
}

// the turn of a leaf that runs. a new turn puts the leaf of the one before down at once, so
// pages can be turned as fast as the tab is clicked (user, 2026-09-29)
type runningTurn struct {
    finish func()
    number int
}

// ends the turn that runs, if one does
func (turn *runningTurn) end() {
    if turn.finish != nil {
        finish := turn.finish
        turn.finish = nil
        finish()
    }
}

// finish is called when the ticks are over, or before that by end
func (turn *runningTurn) start(ui *uilib.UI, ticks uint64, finish func()) {
    turn.end()
    turn.number += 1
    number := turn.number
    turn.finish = finish

    ui.AddDelay(ticks, func(){
        if turn.number == number {
            turn.end()
        }
    })
}

// false: the turn starts and ends flat on the page, as before
const PageTurnClearsTabs = true
// true: the leaf is put down at once where it reaches the tab of the other side. TRIED AND TAKEN BACK
// (user, 2026-09-29: "the page landing now has a pop"): the leaf that comes down covers that tab slowly,
// which hides what happens under it. only the lift was to be faster
const PageTurnClearsLandingTab = false

// columns from the outer rim of a page that a tab covers, with one to spare. the tabs are 14
// wide and end 1 before the rim in the book for casting (spells.lbx 1, 2), 16 wide and end 1 or
// 2 before it in the book of research (scroll.lbx 7, 8)
const castTabReach = 16
const researchTabReach = 18

// steps a turn is looked at in to find where the leaf is clear of a tab
const pageTurnClearSteps = 1000

type leafSize struct {
    Width int
    Reach int
}

var pageTurnClearAt = make(map[leafSize]float64)

// how far a leaf of this width has to be turned, 0 to 1, for none of it to lie in the outer
// columns of the page it leaves
func pageTurnClear(width int, reach int) float64 {
    size := leafSize{Width: width, Reach: reach}
    clear, ok := pageTurnClearAt[size]
    if ok {
        return clear
    }

    clear = 0
    for step := 1; step < pageTurnClearSteps / 2; step++ {
        turned := float64(step) / pageTurnClearSteps
        outer := math.MinInt
        for _, strip := range leafStrips(width, turned, false) {
            outer = max(outer, strip.Column)
        }

        if outer < width - reach {
            clear = turned
            break
        }
    }

    pageTurnClearAt[size] = clear
    return clear
}

// fast at the start, slow at the end
func easeTurnOut(part float64) float64 {
    part = max(0, min(1, part))
    return 1 - (1 - part) * (1 - part)
}

// how far a leaf is turned, 0 to 1, when this part of the time of its turn is over
func turnedAt(part float64, width int, reach int) float64 {
    if !PageTurnClearsTabs {
        return easeTurn(part)
    }

    if part >= 1 {
        return 1
    }

    from := pageTurnClear(width, reach)
    to := 1.0
    if PageTurnClearsLandingTab {
        to = 1 - from
    }

    return from + (to - from) * easeTurnOut(part)
}
