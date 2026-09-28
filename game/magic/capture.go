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
    // the frame of a random battle at which every unit loses about half of its figures, 0 is never
    KillFigures int
    // multiplies the units of both armies of a random battle, 0 leaves the setting of the debug list
    ArmyScale int
    // a spell that is shown in a random battle, and the frame of the last frames before the capture
    // it starts at: name@frame. see captureSpell in fastplay.go
    Spell string
    // a test battle of this unit
    UnitBattle string
    // the list of units of the test battle, with these races open
    UnitPicker string
    // units act in turn, as before they acted together
    InTurn bool
    // a random battle with the same armies on the same ground every time, and the places of all
    // units in the log at the start of every turn. to compare two ways of running a battle
    SameBattle bool
    // the army of the player of a random battle is set to auto
    Auto bool
    // the look of the move area, by its number. below 0: as it is
    MoveArea int
    // the frame at which auto is toggled as by its key, 0 for never
    AutoToggle int
    // keep damage numbers over the units of a random battle
    DamageNumbers bool
    // x,y in screen pixels: the game sees the mouse there
    MouseAt string
    // a screen to open, see game.CaptureScreenNames
    Screen string
    // camera column, below 0 is off
    CameraX float64
    // press Next Turn and trace the layout
    NextTurn bool
    // start a random battle instead of the start screen
    RandomBattle bool
    // start a random battle for a city. a list of: walls, fortress, fire, darkness, outpost, myrror, size=N
    CityBattle string
    // the view of the battlefield: screen pixels per art pixel, and position in screen pixels
    BattleLevel int
    BattlePanX int
    BattlePanY int
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

// frames before the capture in which effects of a battle can be started
const CaptureEffectFrames = 100

// frames between Later and the capture. a simulated drag gets room to finish and coast to a stop
func (capture *frameCapture) laterFrames() int {
    if capture.Drag > 0 {
        return capture.Drag + CaptureCoastFrames
    }
    if capture.Spell != "" || capture.KillFigures > 0 || capture.AutoToggle > 0 {
        // room for a spell to play and for figures to fall and lie
        return CaptureEffectFrames
    }
    if capture.NextTurn || capture.Walk != "" || capture.Screen == "nextunit" || capture.Screen == "research" {
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
