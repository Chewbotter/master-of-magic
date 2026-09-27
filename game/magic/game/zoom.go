package game

// Clean world map zoom.
//
// The map art is drawn at whole multiples of its size and nothing else, so every art pixel
// becomes the same number of screen pixels. At a zoom level between two whole multiples the
// map is first drawn at the next whole multiple up into an offscreen canvas, and that canvas
// is then shrunk to the zoom level in one smooth step (with mipmaps when it shrinks by more
// than half). Drawing tile by tile with a smoothing filter instead leaves seams between tiles.

import (
    "image"
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
)

// false restores the original zoom, which drops or doubles rows and columns of art pixels
var OverworldCleanZoom = true
// pixels of margin around the canvas so the smooth shrink has map to sample at the edges
const ZoomCanvasMargin = 2
// scales closer than this to a whole multiple are treated as whole
const ZoomWholeTolerance = 1e-6

var zoomCanvas *ebiten.Image
var drawingZoomCanvas bool

func getZoomCanvas(width int, height int) *ebiten.Image {
    if zoomCanvas == nil || zoomCanvas.Bounds().Dx() < width || zoomCanvas.Bounds().Dy() < height {
        // room for the largest canvas this screen size can need, so zooming does not reallocate
        zoomCanvas = ebiten.NewImage(width * 2, height * 2)
    }

    return zoomCanvas.SubImage(image.Rect(0, 0, width, height)).(*ebiten.Image)
}

// draws the world map through the canvas when the zoom level is not a whole multiple.
// returns false when the map should be drawn directly
func (overworld *Overworld) drawCleanZoom(screen *ebiten.Image, geom ebiten.GeoM) bool {
    if !OverworldCleanZoom || drawingZoomCanvas {
        return false
    }

    zoom := overworld.Camera.GetAnimatedZoom()
    if zoom <= 0 {
        return false
    }

    target := scale.ScaleAmount * zoom
    if math.Abs(target - math.Round(target)) < ZoomWholeTolerance {
        return false
    }

    whole := max(1, math.Ceil(target))
    // how much larger the canvas is than the screen, between 1 and 2
    ratio := whole / target

    bounds := screen.Bounds()
    margin := float64(ZoomCanvasMargin)
    width := int(math.Ceil(float64(bounds.Max.X) * ratio + margin * 2))
    height := int(math.Ceil(float64(bounds.Max.Y) * ratio + margin * 2))
    canvas := getZoomCanvas(width, height)

    // everything on the map is drawn through scale.DrawScaled, so changing the scale for this one
    // pass makes every art pixel exactly `whole` canvas pixels
    oldScale := scale.ScaleAmount
    scale.UpdateScale(whole / zoom)
    drawingZoomCanvas = true

    // move the map away from the canvas edge by the margin. the passed geom is scaled by `whole` later
    canvasGeom := geom
    canvasGeom.Translate(margin / whole, margin / whole)
    overworld.DrawOverworld(canvas, canvasGeom)

    drawingZoomCanvas = false
    scale.UpdateScale(oldScale)

    var options ebiten.DrawImageOptions
    options.GeoM.Translate(-margin, -margin)
    options.GeoM.Scale(1 / ratio, 1 / ratio)
    options.Filter = ebiten.FilterLinear
    screen.DrawImage(canvas, &options)

    return true
}
