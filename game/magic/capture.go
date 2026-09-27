package main

// Development instrument: write one frame of the running game to a png and exit.
// usage: magic -start -capture out.png -capture-frames 120

import (
    "github.com/kazzmir/master-of-magic/game/magic/display"

    "image/png"
    "log"
    "os"
    "strings"

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
    // runs this many frames before the capture. the frame just before it runs is written
    // next to the capture with _before in its name, to compare the two
    Later func()
    // small window in the lower right corner, without capturing anything
    Corner bool
    // world map zoom level. 0 leaves the default
    Zoom int
    // part way through a zoom animation
    ZoomAnimation float64
    // called on every frame after Later, with the number of frames since Later
    EachFrame func(frame int)
    // simulate a drag of this many frames, DragSpeed screen pixels to the right each frame
    Drag int
    DragSpeed int
    // x,y in screen pixels
    CursorAt string
    // a screen to open, see game.CaptureScreenNames
    Screen string
    // camera column, below 0 is off
    CameraX float64
    // press Next Turn and trace the layout
    NextTurn bool
    // dx,dy in tiles
    CameraMove string
    // dx,dy in tiles
    Walk string
    TraceWide bool
    // development: describes what is on screen, for the layout trace
    Describe func() string
    lastWide bool
    lastDescription string
    drawn int
    done bool
}

var capture frameCapture

// frames between the delayed capture options and the capture itself, enough for a menu to fade in
const CaptureLaterFrames = 20

func writeFrame(screen *ebiten.Image, path string) {
    bounds := screen.Bounds()
    pixels := make([]byte, 4 * bounds.Dx() * bounds.Dy())
    screen.ReadPixels(pixels)
    out := ebiten.NewImage(bounds.Dx(), bounds.Dy())
    out.WritePixels(pixels)

    file, err := os.Create(path)
    if err != nil {
        log.Printf("Error: unable to capture frame: %v", err)
        return
    }
    defer file.Close()

    err = png.Encode(file, out)
    if err != nil {
        log.Printf("Error: unable to capture frame: %v", err)
        return
    }

    log.Printf("captured frame to %v (%vx%v) at %.1f fps", path, bounds.Dx(), bounds.Dy(), ebiten.ActualFPS())
}

// frames between Later and the capture. a simulated drag gets room to finish and coast to a stop
func (capture *frameCapture) laterFrames() int {
    if capture.Drag > 0 {
        return capture.Drag + CaptureCoastFrames
    }
    if capture.NextTurn || capture.Walk != "" || capture.Screen == "nextunit" {
        return CaptureNextTurnFrames
    }
    return CaptureLaterFrames
}

// frames after a simulated drag, enough for a coast to end
const CaptureCoastFrames = 80
// frames after pressing Next Turn, enough for the other players' turns
const CaptureNextTurnFrames = 600
// frames between the delayed capture options and a simulated walk
const CaptureWalkDelayFrames = 60

// called at the end of every Draw. returns true once the frame has been written
func (capture *frameCapture) Update(screen *ebiten.Image) bool {
    if capture.Path == "" || capture.done {
        return capture.done
    }

    capture.drawn += 1
    if capture.Describe != nil {
        wide := display.IsWideContentActive()
        description := capture.Describe()
        if wide != capture.lastWide || description != capture.lastDescription {
            log.Printf("frame %v: wide %v, %v", capture.drawn, wide, description)
            capture.lastWide = wide
            capture.lastDescription = description
        }
    }
    if capture.drawn == capture.Frames - capture.laterFrames() && capture.Later != nil {
        writeFrame(screen, strings.TrimSuffix(capture.Path, ".png") + "_before.png")
        capture.Later()
    }
    if capture.EachFrame != nil && capture.drawn > capture.Frames - capture.laterFrames() {
        capture.EachFrame(capture.drawn - (capture.Frames - capture.laterFrames()))
    }
    if capture.drawn < capture.Frames {
        return false
    }

    writeFrame(screen, capture.Path)

    capture.done = true
    return true
}
