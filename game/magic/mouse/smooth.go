package mouse

// Drawing the cursor between screen pixels, so it can glide in step with the world map while a pan
// settles (see game/magic/game/pan.go). The cursor is drawn crisp on a small canvas and the canvas
// is placed at the fractional position with the smooth filter.

import (
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
)

// optional. while it returns true, the cursor is drawn at the returned position instead of at the
// mouse, in screen pixels. it is given the mouse position in screen pixels
var SmoothPosition func(x int, y int) (float64, float64, bool)

// room on the canvas for the largest cursor, in original pixels
const smoothCanvasSize = 32
// empty screen pixels around the cursor on the canvas, so the smooth filter has room at the edges
const smoothCanvasMargin = 2

var smoothCanvas *ebiten.Image

func (mouse *GlobalMouse) drawBetweenPixels(screen *ebiten.Image, x float64, y float64) {
    size := scale.Scale(smoothCanvasSize) + smoothCanvasMargin * 2
    if smoothCanvas == nil || smoothCanvas.Bounds().Dx() != size {
        smoothCanvas = ebiten.NewImage(size, size)
    }
    smoothCanvas.Clear()

    // the cursor on whole canvas pixels, away from the edge by the margin
    mouse.Options.GeoM.Reset()
    mouse.Options.GeoM.Translate(scale.Unscale(float64(smoothCanvasMargin)), scale.Unscale(float64(smoothCanvasMargin)))
    mouse.DrawFunc(smoothCanvas, &mouse.Options)

    var options ebiten.DrawImageOptions
    options.GeoM.Translate(x - smoothCanvasMargin, y - smoothCanvasMargin)
    if x != math.Floor(x) || y != math.Floor(y) {
        options.Filter = ebiten.FilterLinear
    }
    screen.DrawImage(smoothCanvas, &options)
}
