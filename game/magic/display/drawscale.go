package display

// Native draw scale: the game draws at the window's real pixel size, a whole number of screen pixels
// per art pixel, instead of drawing at 3 and letting Ebiten stretch the picture to the window. Every
// art pixel lands on whole screen pixels, and the world map can move one screen pixel at a time.
// Real pixels count Windows display scaling: a 1776 x 1000 window at 150% is 2664 x 1500 pixels.
// The picture is centered in the window, and the pixels left over after the largest whole scale
// that fits become thin black margins.
//
// NativeDrawScale = false undoes all of it: the game draws at 3 again, Ebiten stretches it, the zoom
// levels go back to 3x, 2x and 1x, and panning blends between pixels again. See CLAUDE.md, Draw scale law.

import (
    "log"
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
)

const NativeDrawScale = true

// the draw scale when NativeDrawScale is off
const FixedDrawScale = 3.0

// development: a fixed draw scale, for runs in the small corner window. 0 is off, and below 0
// forces the native layout even in the corner window
var DrawScaleOverride float64

// where the picture sits in the window, in real pixels. 0 unless the native draw scale leaves margins
var frameX, frameY int

// the largest whole draw scale at which the picture fits a window of this many real pixels
func DrawScaleForPixels(width int, height int) float64 {
    if width <= 0 || height <= 0 {
        return FixedDrawScale
    }

    byHeight := math.Floor(float64(height) / data.ScreenHeight)
    byWidth := math.Floor(float64(width) / data.ScreenWidth)
    return max(1, min(byHeight, byWidth))
}

// Windows display scaling, 1.5 at 150%
func deviceScale() float64 {
    monitor := ebiten.Monitor()
    if monitor != nil && monitor.DeviceScaleFactor() > 0 {
        return monitor.DeviceScaleFactor()
    }
    return 1
}

func setDrawScale(drawScale float64, outsideWidth int, outsideHeight int) {
    if drawScale != scale.ScaleAmount {
        scale.UpdateScale(drawScale)
        logDrawScale(outsideWidth, outsideHeight, drawScale)
    }
}

// the game's Layout: sets the draw scale and the picture's width and position for a window of this
// size, and returns the size of the image the game draws into
func Layout(outsideWidth int, outsideHeight int) (int, int) {
    if DrawScaleOverride > 0 || !NativeDrawScale {
        drawScale := DrawScaleOverride
        if drawScale <= 0 {
            drawScale = FixedDrawScale
        }
        setDrawScale(drawScale, outsideWidth, outsideHeight)
        frameX, frameY = 0, 0
        // Ebiten stretches this to the window
        return scale.Scale2(UpdateLayout(outsideWidth, outsideHeight), data.ScreenHeight)
    }

    factor := deviceScale()
    physicalWidth := int(math.Round(float64(outsideWidth) * factor))
    physicalHeight := int(math.Round(float64(outsideHeight) * factor))

    drawScale := DrawScaleForPixels(physicalWidth, physicalHeight)
    setDrawScale(drawScale, outsideWidth, outsideHeight)
    whole := int(drawScale)

    width := data.ScreenWidth
    if Current.Widescreen {
        width = min(MaxLogicalWidth, physicalWidth / whole)
        width -= width % 2
        width = max(data.ScreenWidth, width)
    }
    logicalWidth = width

    frameX = max(0, (physicalWidth - width * whole) / 2)
    frameY = max(0, (physicalHeight - data.ScreenHeight * whole) / 2)

    // one image pixel per real pixel, nothing is stretched
    return physicalWidth, physicalHeight
}

// the left edge of the whole picture in the window, in real pixels, not counting black bars
func PictureX() int {
    return frameX
}

var startupFinished bool

// runs once the game is running and Windows display scaling is known: converts a settings file saved
// before window scales meant draw scales, and sizes the window now that real pixels can be worked out.
// development runs keep their small corner window
func FinishStartup(development bool) {
    if startupFinished {
        return
    }
    startupFinished = true

    if !NativeDrawScale || development {
        return
    }

    if !Current.ScaleIsDrawScale {
        // keep the window about the size it was: the old scale was in Windows units
        old := Current.WindowScale
        Current.WindowScale = max(MinWindowScale, int(math.Floor(float64(old) * deviceScale())))
        Current.ScaleIsDrawScale = true
        log.Printf("display settings: window scale %v in Windows units is draw scale %v", old, Current.WindowScale)
        Current.Apply()
        Current.Save()
        return
    }

    Current.Apply()
}

// true when the world map should be drawn between screen pixels while it moves.
// at the native draw scale one screen pixel is fine enough that it moves crisp instead
func BlendBetweenPixels() bool {
    return !NativeDrawScale
}

// records each change of draw scale in the log, with what it was based on
func logDrawScale(outsideWidth int, outsideHeight int, drawScale float64) {
    factor := 0.0
    monitorWidth, monitorHeight := 0, 0
    monitor := ebiten.Monitor()
    if monitor != nil {
        factor = monitor.DeviceScaleFactor()
        monitorWidth, monitorHeight = monitor.Size()
    }
    log.Printf("draw scale %v for window %vx%v, device scale factor %v, monitor %vx%v", drawScale, outsideWidth, outsideHeight, factor, monitorWidth, monitorHeight)
}
