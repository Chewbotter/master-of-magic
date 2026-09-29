package settings

import (
    "context"
    "fmt"
    "image"
    "image/color"

    "github.com/kazzmir/master-of-magic/lib/lbx"
    "github.com/kazzmir/master-of-magic/lib/coroutine"
    fontslib "github.com/kazzmir/master-of-magic/game/magic/fonts"
    "github.com/kazzmir/master-of-magic/game/magic/util"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    uilib "github.com/kazzmir/master-of-magic/game/magic/ui"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/vector"
)

const settingsLayer = uilib.UILayer(5)

// adds a checkbox + label pair to the group at the given position, reading/writing
// through get/set. shared by the small handful of boolean settings on this screen.
func addCheckbox(group *uilib.UIElementGroup, fonts *fontslib.SettingsFonts, getAlpha *util.AlphaFadeFunc, x int, y int, label string, get func() bool, set func(bool)) {
    // small, see textsize.go
    checkboxRect := image.Rect(x, y, x + SettingsCheckboxSize, y + SettingsCheckboxSize)

    group.AddElement(&uilib.UIElement{
        Layer: settingsLayer,
        Rect: checkboxRect,
        LeftClick: func(element *uilib.UIElement){
            set(!get())
        },
        // what the setting does, see tips.go
        Inside: func(element *uilib.UIElement, x int, y int){
            hoverTip(label, element, x, y)
        },
        Draw: func(element *uilib.UIElement, screen *ebiten.Image){
            rect := element.Rect

            vector.FillRect(screen, float32(scale.Scale(rect.Min.X)), float32(scale.Scale(rect.Min.Y)), float32(scale.Scale(rect.Dx())), float32(scale.Scale(rect.Dy())), color.NRGBA{R: 32, G: 32, B: 32, A: uint8(200 * (*getAlpha)())}, false)
            util.DrawRect(screen, scale.ScaleRect(rect), color.NRGBA{R: 255, G: 255, B: 255, A: uint8(200 * (*getAlpha)())})

            if get() {
                inner := rect.Inset(SettingsCheckboxMark)
                vector.FillRect(screen, float32(scale.Scale(inner.Min.X)), float32(scale.Scale(inner.Min.Y)), float32(scale.Scale(inner.Dx())), float32(scale.Scale(inner.Dy())), color.NRGBA{R: 255, G: 255, B: 255, A: uint8(220 * (*getAlpha)())}, false)
            }
        },
    })

    group.AddElement(&uilib.UIElement{
        Layer: settingsLayer,
        Draw: func(element *uilib.UIElement, screen *ebiten.Image){
            var options ebiten.DrawImageOptions
            options.ColorScale.ScaleAlpha((*getAlpha)())
            // small and at the middle of its checkbox, see textsize.go
            middle := float64(checkboxRect.Min.Y + checkboxRect.Max.Y) / 2
            fonts.OptionFont.PrintOptions(screen, float64(checkboxRect.Max.X + SettingsLabelGap), settingsTextMiddleY(fonts.OptionFont, middle), settingsText(&options), label)
        },
    })

    // the name shows what the setting does too
    labelRect := image.Rect(checkboxRect.Max.X, checkboxRect.Min.Y, checkboxRect.Max.X + SettingsLabelGap + settingsTextWidth(fonts.OptionFont, label), checkboxRect.Max.Y)
    addTipArea(group, labelRect, label)
}

type MusicSettings interface {
    GetVolume() float64
    SetVolume(float64)
    IsMusicEnabled() bool
    SetMusicEnabled(bool)
}

func MakeSettingsUI(yield coroutine.YieldFunc, parentUI *uilib.UI, cache *lbx.LbxCache, imageCache *util.ImageCache, settings *Settings, musicSettings MusicSettings) (*uilib.UIElementGroup, context.Context) {
    fonts := fontslib.MakeSettingsFonts(cache)

    group := uilib.MakeGroup()
    quit, cancel := context.WithCancel(context.Background())

    background, _ := imageCache.GetImage("load.lbx", 11, 0)

    getAlpha := group.MakeFadeIn(7)

    group.AddElement(&uilib.UIElement{
        Layer: 4,
        Draw: func(element *uilib.UIElement, screen *ebiten.Image){
            var backgroundOptions ebiten.DrawImageOptions
            backgroundOptions.ColorScale.ScaleAlpha(getAlpha())
            scale.DrawScaled(screen, background, &backgroundOptions)
        },
    })

    ok, _ := imageCache.GetImage("load.lbx", 4, 0)

    group.AddElement(&uilib.UIElement{
        Layer: settingsLayer,
        Rect: util.ImageRect(266, 176, ok),
        LeftClick: func(element *uilib.UIElement){
            getAlpha = group.MakeFadeOut(7)
            group.AddDelay(7, func(){
                cancel()
            })
        },
        Draw: func(element *uilib.UIElement, screen *ebiten.Image){
            var options ebiten.DrawImageOptions
            options.GeoM.Translate(float64(element.Rect.Min.X), float64(element.Rect.Min.Y))
            options.ColorScale.ScaleAlpha(getAlpha())
            scale.DrawScaled(screen, ok, &options)
        },
    })

    // next to the box of the controls (user, 2026-09-28), see options.go. upstream had it left
    // of Ok, at 214, 176 and 40 by 13
    keysTop := settingsRowBoxTop(ControlsRow)
    keysButtonRect := image.Rect(KeysBoxX, keysTop, KeysBoxX + KeysBoxWidth, keysTop + SettingsRowBoxHeight)
    group.AddElement(&uilib.UIElement{
        Layer: settingsLayer,
        Rect: keysButtonRect,
        Inside: func(element *uilib.UIElement, x int, y int){
            hoverTip("Keys", element, x, y)
        },
        LeftClick: func(element *uilib.UIElement){
            keysGroup, keysQuit := MakeKeysUI(yield, parentUI, cache, imageCache, settings.Keybindings)
            runNestedUI(yield, parentUI, keysGroup, keysQuit)
        },
        Draw: func(element *uilib.UIElement, screen *ebiten.Image){
            rect := element.Rect
            vector.FillRect(screen, float32(scale.Scale(rect.Min.X)), float32(scale.Scale(rect.Min.Y)), float32(scale.Scale(rect.Dx())), float32(scale.Scale(rect.Dy())), color.NRGBA{R: 96, G: 60, B: 20, A: uint8(255 * getAlpha())}, false)
            util.DrawRect(screen, scale.ScaleRect(rect), color.NRGBA{R: 255, G: 200, B: 100, A: uint8(255 * getAlpha())})

            var options ebiten.DrawImageOptions
            options.ColorScale.ScaleAlpha(getAlpha())
            fonts.OptionFont.PrintOptions(screen, float64(rect.Min.X + 5), settingsTextMiddleY(fonts.OptionFont, float64(rect.Min.Y + rect.Max.Y) / 2), settingsText(&options), "Keys")
        },
    })

    group.AddElement(&uilib.UIElement{
        Layer: settingsLayer,
        Draw: func(element *uilib.UIElement, screen *ebiten.Image){
            var options ebiten.DrawImageOptions
            options.ColorScale.ScaleAlpha(getAlpha())
            fonts.OptionFont.PrintOptions(screen, 30, settingsTextY(fonts.OptionFont, 40), settingsText(&options), fmt.Sprintf("Volume: %02d%%", int(musicSettings.GetVolume() * 100)))
        },
    })

    slider, _ := imageCache.GetImage("spellscr.lbx", 3, 0)

    volumeClicked := false
    group.AddElement(&uilib.UIElement{
        Layer: settingsLayer,
        Rect: image.Rect(30, 50, 30 + 80, 50 + slider.Bounds().Dy()),
        Inside: func(this *uilib.UIElement, x int, y int){
            hoverTip("Volume", this, x, y)
            if volumeClicked {
                musicSettings.SetVolume(min(1, float64(x) / float64(this.Rect.Dx() - 1)))
            }
        },
        LeftClick: func(element *uilib.UIElement){
            volumeClicked = true
        },
        LeftClickRelease: func(element *uilib.UIElement){
            volumeClicked = false
        },
        Draw: func(element *uilib.UIElement, screen *ebiten.Image){
            backgroundRect := element.Rect
            backgroundRect.Max.X += 5
            backgroundRect.Min.X -= 1
            backgroundRect.Min.Y -= 1

            vector.FillRect(screen, float32(scale.Scale(backgroundRect.Min.X)), float32(scale.Scale(backgroundRect.Min.Y)), float32(scale.Scale(backgroundRect.Dx())), float32(scale.Scale(backgroundRect.Dy())), color.NRGBA{R: 32, G: 32, B: 32, A: uint8(200 * getAlpha())}, false)
            util.DrawRect(screen, scale.ScaleRect(backgroundRect), color.NRGBA{R: 255, G: 255, B: 255, A: uint8(200 * getAlpha())})

            var options ebiten.DrawImageOptions
            options.ColorScale.ScaleAlpha(getAlpha())
            options.GeoM.Translate(float64(element.Rect.Min.X) + float64(element.Rect.Dx()) * musicSettings.GetVolume(), float64(element.Rect.Min.Y))
            options.GeoM.Translate(float64(-slider.Bounds().Dx()/2), 0)
            scale.DrawScaled(screen, slider, &options)

            // util.DrawRect(screen, scale.ScaleRect(element.Rect), color.RGBA{R: 255, A: 255})
        },
    })

    // the rows of textsize.go
    addCheckbox(group, fonts, &getAlpha, SettingsLeftColumnX, SettingsRowY(0), "End Of Turn Wait",
        func() bool { return settings.EndOfTurnWait },
        func(value bool) { settings.EndOfTurnWait = value },
    )

    addCheckbox(group, fonts, &getAlpha, SettingsLeftColumnX, SettingsRowY(1), "Strategic Combat Only",
        func() bool { return settings.StrategicCombatOnly },
        func(value bool) { settings.StrategicCombatOnly = value },
    )

    // maps to the single existing music on/off flag - the original game's separate
    // "Event Music" checkbox would need songs classified into background vs. event
    // categories, which doesn't exist yet, so it's left as a future addition.
    addCheckbox(group, fonts, &getAlpha, SettingsLeftColumnX, SettingsRowY(2), "Background Music",
        musicSettings.IsMusicEnabled,
        musicSettings.SetMusicEnabled,
    )
     // gates whether game.Model.DoRandomEvents() gets called each turn: when off,
     // no new random events are rolled while in-flight ones (plague, conjunctions)
     // still run through their normal decay path.
    addCheckbox(group, fonts, &getAlpha, SettingsLeftColumnX, SettingsRowY(3), "Random Events",
        func() bool { return settings.RandomEvents },
        func(value bool) { settings.RandomEvents = value },
    )

    // session-only, same as Random Events. Enemy2 reads this via AIServices.
    addCheckbox(group, fonts, &getAlpha, DisplayColumnX, SettingsRowY(AggressiveRow), "Aggressive AI",
        func() bool { return settings.AggressiveAI },
        func(value bool) { settings.AggressiveAI = value },
    )

    // what a setting does, see tips.go
    addTips(group, fonts, &getAlpha)

    return group, quit
}
