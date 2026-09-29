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

    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
)

// with the mouse over it a tab is lifted by this many art pixels, and leaves a shadow this dark,
// 0 to 1, where it lay. there is no look for a tab that is pressed
const pageTabLift = 1
const pageTabShadow = 0.35

// draws a tab that turns the pages at a place in art pixels
func drawPageTab(screen *ebiten.Image, tab *ebiten.Image, place image.Point, alpha float32, over bool) {
    if tab == nil {
        return
    }

    var options ebiten.DrawImageOptions
    options.ColorScale.ScaleAlpha(alpha)
    options.GeoM.Translate(float64(place.X), float64(place.Y))

    if over {
        shadow := options
        shadow.ColorScale.Scale(0, 0, 0, pageTabShadow)
        scale.DrawScaled(screen, tab, &shadow)

        options.GeoM.Translate(0, -pageTabLift)
    }

    scale.DrawScaled(screen, tab, &options)
}

// false: the turn starts and ends flat on the page, as before
const PageTurnClearsTabs = true
// false: only the start of the turn is clear of its tab, the leaf comes down under the other one
const PageTurnClearsLandingTab = true

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
