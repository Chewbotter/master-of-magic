package display

// Window and fullscreen preferences. These are per machine, not per save, so they
// live in their own small file next to the executable's working directory.

import (
    "encoding/json"
    "fmt"
    "log"
    "os"

    "github.com/hajimehoshi/ebiten/v2"
)

// window sizes are whole multiples of the original 320x200 screen so pixels stay even
// 5 is 1776 x 1000 in widescreen, 1600 x 1000 without
const DefaultWindowScale = 5
const MinWindowScale = 2
const MaxWindowScale = 12
const DefaultFullscreen = false

// vertical space left free for the window title bar and the taskbar when
// deciding which window sizes fit on the monitor
const MonitorMarginY = 60

// file the preferences are stored in, relative to the working directory
var SettingsFile = "display.json"
// development runs read and write this one instead, so they never touch the player's preferences
const DevelopmentSettingsFile = "display-dev.json"

type Settings struct {
    Fullscreen bool `json:"fullscreen"`
    WindowScale int `json:"window-scale"`
    Widescreen bool `json:"widescreen"`
}

// the preferences in use by the running game
var Current *Settings = MakeDefault()

func MakeDefault() *Settings {
    return &Settings{
        Fullscreen: DefaultFullscreen,
        WindowScale: DefaultWindowScale,
        Widescreen: DefaultWidescreen,
    }
}

// size of the window for the given scale, following the widescreen setting in use
func SizeForScale(windowScale int) (int, int) {
    return sizeForScale(windowScale, Current.Widescreen)
}

func (settings *Settings) WindowSize() (int, int) {
    return sizeForScale(settings.WindowScale, settings.Widescreen)
}

func ResolutionName(windowScale int) string {
    width, height := SizeForScale(windowScale)
    return fmt.Sprintf("%v x %v", width, height)
}

// window scales that fit on the current monitor, smallest first. always contains at least one entry
func AvailableScales() []int {
    monitorWidth, monitorHeight := 0, 0
    monitor := ebiten.Monitor()
    if monitor != nil {
        monitorWidth, monitorHeight = monitor.Size()
    }

    var out []int
    for windowScale := MinWindowScale; windowScale <= MaxWindowScale; windowScale++ {
        width, height := SizeForScale(windowScale)
        // an unknown monitor size allows everything up to the default
        if monitorWidth <= 0 || monitorHeight <= 0 {
            if windowScale <= DefaultWindowScale {
                out = append(out, windowScale)
            }
            continue
        }

        if width <= monitorWidth && height + MonitorMarginY <= monitorHeight {
            out = append(out, windowScale)
        }
    }

    if len(out) == 0 {
        out = append(out, MinWindowScale)
    }

    return out
}

// keep the window scale inside the set that fits the monitor
func (settings *Settings) Clamp() {
    available := AvailableScales()
    largest := available[len(available) - 1]
    smallest := available[0]

    if settings.WindowScale > largest {
        settings.WindowScale = largest
    }
    if settings.WindowScale < smallest {
        settings.WindowScale = smallest
    }
}

// push the preferences to the actual window
func (settings *Settings) Apply() {
    ebiten.SetFullscreen(settings.Fullscreen)
    if settings.Fullscreen {
        return
    }

    settings.Clamp()
    width, height := settings.WindowSize()
    ebiten.SetWindowSize(width, height)

    monitor := ebiten.Monitor()
    if monitor != nil {
        monitorWidth, monitorHeight := monitor.Size()
        if monitorWidth > 0 && monitorHeight > 0 {
            ebiten.SetWindowPosition((monitorWidth - width) / 2, max(0, (monitorHeight - height) / 2))
        }
    }
}

func (settings *Settings) SetFullscreen(fullscreen bool) {
    settings.Fullscreen = fullscreen
    settings.Apply()
    settings.Save()
}

func (settings *Settings) SetWidescreen(widescreen bool) {
    settings.Widescreen = widescreen
    settings.Apply()
    settings.Save()
}

func (settings *Settings) SetWindowScale(windowScale int) {
    settings.WindowScale = windowScale
    settings.Apply()
    settings.Save()
}

func (settings *Settings) Save() {
    file, err := os.Create(SettingsFile)
    if err != nil {
        log.Printf("Unable to save display settings: %v", err)
        return
    }
    defer file.Close()

    encoder := json.NewEncoder(file)
    encoder.SetIndent("", "  ")
    err = encoder.Encode(settings)
    if err != nil {
        log.Printf("Unable to save display settings: %v", err)
    }
}

// read the preferences file into Current. a missing or broken file leaves the defaults in place
func Load() *Settings {
    settings := MakeDefault()

    file, err := os.Open(SettingsFile)
    if err == nil {
        defer file.Close()
        err = json.NewDecoder(file).Decode(settings)
        if err != nil {
            log.Printf("Unable to read display settings: %v", err)
            settings = MakeDefault()
        }
    }

    if settings.WindowScale < MinWindowScale || settings.WindowScale > MaxWindowScale {
        settings.WindowScale = DefaultWindowScale
    }

    Current = settings
    return settings
}
