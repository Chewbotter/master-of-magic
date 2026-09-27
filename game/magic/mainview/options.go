package mainview

// Shows the shared options screen (game/magic/settings/options.go) over the main menu backdrop.

import (
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    settingslib "github.com/kazzmir/master-of-magic/game/magic/settings"
    uilib "github.com/kazzmir/master-of-magic/game/magic/ui"

    "github.com/hajimehoshi/ebiten/v2"
)

// show the options screen in place of the main menu
func (main *MainScreen) ShowOptions() {
    main.CreditsPlaying = false

    ui := &uilib.UI{
        Draw: func(ui *uilib.UI, screen *ebiten.Image) {
            var options ebiten.DrawImageOptions

            top, err := main.ImageCache.GetImages("mainscrn.lbx", 0)
            if err == nil {
                use := top[(main.Counter / 4) % uint64(len(top))]
                scale.DrawScaled(screen, use, &options)
                options.GeoM.Translate(0, float64(use.Bounds().Dy()))
            }

            background, err := main.ImageCache.GetImage("mainscrn.lbx", 5, 0)
            if err == nil {
                scale.DrawScaled(screen, background, &options)
            }

            ui.StandardDraw(screen)
        },
    }
    ui.SetElementsFromArray(nil)

    // the key binding screen waits on the coroutine that is driving the menu at that moment
    yield := func() error {
        if main.Yield == nil {
            return nil
        }
        return main.Yield()
    }

    group, quit, closeScreen := settingslib.MakeOptionsUI(yield, ui, main.Cache, &main.ImageCache, main.Settings, main.Music)
    ui.AddGroup(group)

    ui.Update = func(ui *uilib.UI) {
        if quit.Err() != nil {
            main.HideOptions()
        }
    }

    ui.HandleKeys = func(keys []ebiten.Key){
        for _, key := range keys {
            if key == ebiten.KeyEscape {
                closeScreen()
            }
        }
    }

    if main.MenuUI == nil {
        main.MenuUI = main.UI
    }
    main.UI = ui
}

// return from the options screen to the main menu
func (main *MainScreen) HideOptions() {
    if main.MenuUI != nil {
        main.UI = main.MenuUI
        main.MenuUI = nil
    }
}
