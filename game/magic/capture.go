package main

// Development instrument: write one frame of the running game to a png and exit.
// usage: magic -start -capture out.png -capture-frames 120

import (
    "image/png"
    "log"
    "os"

    "github.com/hajimehoshi/ebiten/v2"
)

type frameCapture struct {
    // png file to write. empty disables the capture
    Path string
    // frames to draw before capturing
    Frames int
    // open the game menu over the world map first
    Popup bool
    // deselect units first
    NoSelection bool
    // keep the starting city name prompt, which captures skip by default
    CityPrompt bool
    RevealAll bool
    DebugMenu bool
    // screen pixels
    PanX int
    PanY int
    // runs this many frames before the capture
    Later func()
    // small window in the lower right corner, without capturing anything
    Corner bool
    // world map zoom level. 0 leaves the default
    Zoom int
    // part way through a zoom animation
    ZoomAnimation float64
    drawn int
    done bool
}

var capture frameCapture

// frames between the delayed capture options and the capture itself, enough for a menu to fade in
const CaptureLaterFrames = 20

// called at the end of every Draw. returns true once the frame has been written
func (capture *frameCapture) Update(screen *ebiten.Image) bool {
    if capture.Path == "" || capture.done {
        return capture.done
    }

    capture.drawn += 1
    if capture.drawn == capture.Frames - CaptureLaterFrames && capture.Later != nil {
        capture.Later()
    }
    if capture.drawn < capture.Frames {
        return false
    }

    bounds := screen.Bounds()
    pixels := make([]byte, 4 * bounds.Dx() * bounds.Dy())
    screen.ReadPixels(pixels)
    out := ebiten.NewImage(bounds.Dx(), bounds.Dy())
    out.WritePixels(pixels)

    file, err := os.Create(capture.Path)
    if err != nil {
        log.Printf("Error: unable to capture frame: %v", err)
    } else {
        err = png.Encode(file, out)
        if err != nil {
            log.Printf("Error: unable to capture frame: %v", err)
        }
        file.Close()
        log.Printf("captured frame to %v (%vx%v) at %.1f fps", capture.Path, bounds.Dx(), bounds.Dy(), ebiten.ActualFPS())
    }

    capture.done = true
    return true
}
