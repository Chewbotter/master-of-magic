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
// camera positions closer than this to a whole screen pixel are drawn directly, in screen pixels
const ZoomSubpixelTolerance = 0.01

var zoomCanvas *ebiten.Image
var drawingZoomCanvas bool

func getZoomCanvas(width int, height int) *ebiten.Image {
    if zoomCanvas == nil || zoomCanvas.Bounds().Dx() < width || zoomCanvas.Bounds().Dy() < height {
        // room for the largest canvas this screen size can need, so zooming does not reallocate
        zoomCanvas = ebiten.NewImage(width * 2, height * 2)
    }

    return zoomCanvas.SubImage(image.Rect(0, 0, width, height)).(*ebiten.Image)
}

// how far a value is from the nearest whole number
func offWhole(value float64) float64 {
    return math.Abs(value - math.Round(value))
}

// draws the world map through the canvas when the zoom level is not a whole multiple, or when the
// camera is between two screen pixels (while the view glides after a pan, see pan.go).
// returns false when the map should be drawn directly
func (overworld *Overworld) drawCleanZoom(screen *ebiten.Image, geom ebiten.GeoM) bool {
    if !OverworldCleanZoom || drawingZoomCanvas {
        return false
    }

    zoom := overworld.Camera.GetAnimatedZoom()
    if zoom <= 0 {
        return false
    }

    tileWidth := float64(overworld.Map.TileWidth())
    tileHeight := float64(overworld.Map.TileHeight())

    target := scale.ScaleAmount * zoom
    wholeZoom := offWhole(target) < ZoomWholeTolerance

    // where the map starts, in screen pixels
    screenX := overworld.Camera.GetZoomedX() * tileWidth * target
    screenY := overworld.Camera.GetZoomedY() * tileHeight * target
    betweenPixels := offWhole(screenX) > ZoomSubpixelTolerance || offWhole(screenY) > ZoomSubpixelTolerance

    if wholeZoom && !betweenPixels {
        return false
    }

    whole := max(1, math.Ceil(target))
    if wholeZoom {
        whole = math.Round(target)
    }
    // how much larger the canvas is than the screen, between 1 and 2
    ratio := whole / target

    // draw the map on whole canvas pixels, and move it by the leftover fraction when the canvas
    // is put on the screen. the smooth filter spreads that fraction over neighboring pixels
    canvasX := overworld.Camera.GetZoomedX() * tileWidth * whole
    canvasY := overworld.Camera.GetZoomedY() * tileHeight * whole
    fractionX := canvasX - math.Floor(canvasX)
    fractionY := canvasY - math.Floor(canvasY)

    oldCamera := overworld.Camera
    overworld.Camera.DX -= fractionX / (tileWidth * whole)
    overworld.Camera.DY -= fractionY / (tileHeight * whole)
    defer func() {
        overworld.Camera = oldCamera
    }()

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
    // the camera was moved back by the fraction, so the map sits that much too far right and down
    options.GeoM.Translate(-margin - fractionX, -margin - fractionY)
    options.GeoM.Scale(1 / ratio, 1 / ratio)
    options.Filter = ebiten.FilterLinear
    screen.DrawImage(canvas, &options)

    return true
}
