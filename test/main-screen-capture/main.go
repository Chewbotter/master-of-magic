package main

// Development probe: renders the main menu for a number of frames, writes a png and exits.
// usage: main-screen-capture -data <lbx dir> -out <png> [-frames N] [-credits]

import (
    "flag"
    "fmt"
    "image/png"
    "log"
    "os"

    "github.com/kazzmir/master-of-magic/lib/lbx"
    "github.com/kazzmir/master-of-magic/game/magic/mainview"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    settingslib "github.com/kazzmir/master-of-magic/game/magic/settings"
    musiclib "github.com/kazzmir/master-of-magic/game/magic/music"

    "github.com/hajimehoshi/ebiten/v2"
)

type Engine struct {
    MainScreen *mainview.MainScreen
    Frame int
    MaxFrames int
    Output string
    Saved bool
}

func (engine *Engine) Update() error {
    engine.MainScreen.Update(nil)
    engine.Frame += 1
    if engine.Saved {
        return ebiten.Termination
    }
    return nil
}

func (engine *Engine) Draw(screen *ebiten.Image) {
    engine.MainScreen.Draw(screen)

    if engine.Frame >= engine.MaxFrames && !engine.Saved {
        bounds := screen.Bounds()
        pixels := make([]byte, 4 * bounds.Dx() * bounds.Dy())
        screen.ReadPixels(pixels)
        out := ebiten.NewImage(bounds.Dx(), bounds.Dy())
        out.WritePixels(pixels)

        file, err := os.Create(engine.Output)
        if err != nil {
            log.Printf("Error: %v", err)
        } else {
            err = png.Encode(file, out)
            if err != nil {
                log.Printf("Error: %v", err)
            }
            file.Close()
            log.Printf("wrote %v", engine.Output)
        }
        engine.Saved = true
    }
}

func (engine *Engine) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
    return scale.Scale2(data.ScreenWidth, data.ScreenHeight)
}

func main(){
    log.SetFlags(log.Ldate | log.Lshortfile | log.Lmicroseconds)

    var dataPath string
    var output string
    var frames int
    var credits bool
    flag.StringVar(&dataPath, "data", "", "path to lbx files")
    flag.StringVar(&output, "out", "main-screen.png", "png file to write")
    flag.IntVar(&frames, "frames", 60, "frames to run before capturing")
    flag.BoolVar(&credits, "credits", false, "start the credits before capturing")
    var options bool
    var rollout bool
    flag.BoolVar(&options, "options", false, "show the options screen before capturing")
    flag.BoolVar(&rollout, "rollout", false, "open the resolution rollout on the options screen")
    var mouseAt string
    flag.StringVar(&mouseAt, "mouse-at", "", "x,y in art pixels: the interface sees the mouse there")
    var drawScale float64
    flag.Float64Var(&drawScale, "scale", 0, "screen pixels per art pixel, instead of 3")
    flag.Parse()

    if drawScale > 0 {
        scale.UpdateScale(drawScale)
    }

    if mouseAt != "" {
        var x, y int
        _, err := fmt.Sscanf(mouseAt, "%d,%d", &x, &y)
        if err != nil {
            log.Printf("Error: -mouse-at wants x,y: %v", err)
            return
        }
        display.FixedCursor = true
        display.FixedCursorX = scale.Scale(x)
        display.FixedCursorY = scale.Scale(y)
    }

    cache := lbx.CacheFromPath(dataPath)
    if cache == nil {
        log.Printf("Error: unable to load data from '%v'", dataPath)
        return
    }

    display.PlaceInCorner()
    ebiten.SetWindowTitle("main screen capture")

    music := musiclib.MakeMusic(cache)
    music.Enabled = false
    mainScreen := mainview.MakeMainScreen(cache, nil, music, settingslib.MakeSettings(cache))
    if credits {
        mainScreen.ToggleCredits()
    }
    if options {
        settingslib.OptionsStartWithRolloutOpen = rollout
        mainScreen.ShowOptions()
    }

    err := ebiten.RunGame(&Engine{MainScreen: mainScreen, MaxFrames: frames, Output: output})
    if err != nil {
        log.Printf("Error: %v", err)
    }
}
