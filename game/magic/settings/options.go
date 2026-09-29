package settings

// The single settings screen, shown from the main menu and from the in-game menu.
// It is upstream's settings screen (MakeSettingsUI, left untouched in settings.go) with
// the display controls added on top in the same visual style.

import (
    "context"
    "image"

    "github.com/kazzmir/master-of-magic/lib/lbx"
    "github.com/kazzmir/master-of-magic/lib/coroutine"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/display"
    fontslib "github.com/kazzmir/master-of-magic/game/magic/fonts"
    "github.com/kazzmir/master-of-magic/game/magic/util"
    uilib "github.com/kazzmir/master-of-magic/game/magic/ui"

    "github.com/hajimehoshi/ebiten/v2"
)

// display controls tunables. positions are in original 320x200 pixels.
// upstream's controls fill the left column, these sit in the free right column

// left edge of the display controls
const DisplayColumnX = 175
// y of the "Resolution" caption
const ResolutionLabelY = 40
// the box that shows the resolution in use and opens the list
const ResolutionBoxY = 50
const ResolutionBoxWidth = 84
const ResolutionBoxHeight = 13
// the rows of the settings, see textsize.go. left column: upstream's four checkboxes on the rows
// 0 to 3, then these
const SingleStrikesRow = 4
const HideCursorRow = 5
// right column
const FullscreenRow = 0
const WidescreenRow = 1
// the control type row: caption, and a box that switches between Modern and Classic when clicked
const ControlsRow = 2
// upstream's
const AggressiveRow = 3
const ControlsBoxX = DisplayColumnX + 32
const ControlsBoxWidth = 40
// the button of the keys, on the row of the controls and right of their box
const KeysBoxX = ControlsBoxX + ControlsBoxWidth + 4
const KeysBoxWidth = 26
// opacity of the resolution box while fullscreen makes it meaningless
const DisplayDisabledAlpha = 0.4
// ticks for the fade, matches upstream's settings screen
const SettingsFadeTicks = 7

const rolloutLayer = uilib.UILayer(6)

// development hook: the capture probe sets this to render the screen with the resolution list open
var OptionsStartWithRolloutOpen = false

// returns the group to show, a context that is done when the screen has closed, and a function that closes the screen
func MakeOptionsUI(yield coroutine.YieldFunc, parentUI *uilib.UI, cache *lbx.LbxCache, imageCache *util.ImageCache, settings *Settings, musicSettings MusicSettings) (*uilib.UIElementGroup, context.Context, func()) {
    group, upstreamQuit := MakeSettingsUI(yield, parentUI, cache, imageCache, settings, musicSettings)

    // done when upstream's Ok button finishes, or when closeScreen is called
    quit, cancel := context.WithCancel(upstreamQuit)

    fonts := fontslib.MakeSettingsFonts(cache)
    displaySettings := display.Current

    getAlpha := group.MakeFadeIn(SettingsFadeTicks)
    closing := false

    closeScreen := func() {
        if closing {
            return
        }
        closing = true
        getAlpha = group.MakeFadeOut(SettingsFadeTicks)
        group.AddDelay(SettingsFadeTicks, func(){
            cancel()
        })
    }

    drawBox := func(screen *ebiten.Image, rect image.Rectangle, alpha float32, highlight bool) {
        fill := settingsBoxFill
        if highlight {
            fill = settingsBoxLit
        }
        // of art pixels, see artbox.go
        drawArtBox(screen, rect, fill, settingsBoxRim, alpha)
    }

    // text at the middle of the height of a box or a row
    printInBox := func(screen *ebiten.Image, x int, rect image.Rectangle, alpha float32, text string) {
        var options ebiten.DrawImageOptions
        options.ColorScale.ScaleAlpha(alpha)
        middle := float64(rect.Min.Y + rect.Max.Y) / 2
        fonts.OptionFont.PrintOptions(screen, float64(x), settingsTextMiddleY(fonts.OptionFont, middle), settingsText(&options), text)
    }

    printText := func(screen *ebiten.Image, x int, y int, alpha float32, text string) {
        var options ebiten.DrawImageOptions
        options.ColorScale.ScaleAlpha(alpha)
        // small, see textsize.go
        fonts.OptionFont.PrintOptions(screen, float64(x), settingsTextY(fonts.OptionFont, y), settingsText(&options), text)
    }

    // caption
    group.AddElement(&uilib.UIElement{
        Layer: settingsLayer,
        Draw: func(element *uilib.UIElement, screen *ebiten.Image){
            printText(screen, DisplayColumnX, ResolutionLabelY, getAlpha(), "Resolution")
        },
    })

    var rolloutElements []*uilib.UIElement

    closeRollout := func() {
        group.RemoveElements(rolloutElements)
        rolloutElements = nil
    }

    openRollout := func() {
        closeRollout()

        // anything clicked outside the list closes it
        rolloutElements = append(rolloutElements, &uilib.UIElement{
            Layer: rolloutLayer,
            Order: -1,
            Rect: image.Rect(0, 0, data.ScreenWidth, data.ScreenHeight),
            LeftClick: func(element *uilib.UIElement){
                closeRollout()
            },
            RightClick: func(element *uilib.UIElement){
                closeRollout()
            },
        })

        y := ResolutionBoxY + ResolutionBoxHeight
        for _, windowScale := range display.AvailableScales() {
            rect := image.Rect(DisplayColumnX, y, DisplayColumnX + ResolutionBoxWidth, y + ResolutionBoxHeight)
            inside := false
            rolloutElements = append(rolloutElements, &uilib.UIElement{
                Layer: rolloutLayer,
                Rect: rect,
                LeftClick: func(element *uilib.UIElement){
                    displaySettings.SetWindowScale(windowScale)
                    closeRollout()
                },
                Inside: func(element *uilib.UIElement, x, y int) {
                    inside = true
                },
                NotInside: func(element *uilib.UIElement){
                    inside = false
                },
                Draw: func(element *uilib.UIElement, screen *ebiten.Image){
                    drawBox(screen, rect, getAlpha(), inside || windowScale == displaySettings.WindowScale)
                    printInBox(screen, rect.Min.X + 6, rect, getAlpha(), display.ResolutionName(windowScale))
                },
            })
            y += ResolutionBoxHeight
        }

        group.AddElements(rolloutElements)
    }

    // the box showing the resolution in use. click to open the list
    resolutionRect := image.Rect(DisplayColumnX, ResolutionBoxY, DisplayColumnX + ResolutionBoxWidth, ResolutionBoxY + ResolutionBoxHeight)
    group.AddElement(&uilib.UIElement{
        Layer: settingsLayer,
        Rect: resolutionRect,
        LeftClick: func(element *uilib.UIElement){
            if closing || displaySettings.Fullscreen {
                return
            }
            openRollout()
        },
        Inside: func(element *uilib.UIElement, x int, y int){
            hoverTip("Resolution", element, x, y)
        },
        Draw: func(element *uilib.UIElement, screen *ebiten.Image){
            alpha := getAlpha()
            text := display.ResolutionName(displaySettings.WindowScale)
            if displaySettings.Fullscreen {
                alpha *= DisplayDisabledAlpha
                text = "Desktop"
            }
            drawBox(screen, resolutionRect, alpha, false)
            printInBox(screen, resolutionRect.Min.X + 6, resolutionRect, alpha, text)
        },
    })

    addCheckbox(group, fonts, &getAlpha, DisplayColumnX, SettingsRowY(FullscreenRow), "Fullscreen",
        func() bool { return displaySettings.Fullscreen },
        func(value bool) {
            displaySettings.SetFullscreen(value)
        },
    )

    addCheckbox(group, fonts, &getAlpha, DisplayColumnX, SettingsRowY(WidescreenRow), "Widescreen",
        func() bool { return displaySettings.Widescreen },
        func(value bool) {
            displaySettings.SetWidescreen(value)
        },
    )

    // control type
    group.AddElement(&uilib.UIElement{
        Layer: settingsLayer,
        Draw: func(element *uilib.UIElement, screen *ebiten.Image){
            // at the height of a checkbox of the row
            row := image.Rect(DisplayColumnX, SettingsRowY(ControlsRow), DisplayColumnX, SettingsRowY(ControlsRow) + SettingsCheckboxSize)
            printInBox(screen, DisplayColumnX, row, getAlpha(), "Controls")
        },
    })

    controlsTop := settingsRowBoxTop(ControlsRow)
    controlsRect := image.Rect(ControlsBoxX, controlsTop, ControlsBoxX + ControlsBoxWidth, controlsTop + SettingsRowBoxHeight)
    controlsInside := false
    group.AddElement(&uilib.UIElement{
        Layer: settingsLayer,
        Rect: controlsRect,
        LeftClick: func(element *uilib.UIElement){
            if closing {
                return
            }
            if displaySettings.Controls() == display.ControlsModern {
                displaySettings.SetControlType(display.ControlsClassic)
            } else {
                displaySettings.SetControlType(display.ControlsModern)
            }
        },
        Inside: func(element *uilib.UIElement, x int, y int){
            controlsInside = true
            hoverTip("Controls", element, x, y)
        },
        NotInside: func(element *uilib.UIElement){
            controlsInside = false
        },
        Draw: func(element *uilib.UIElement, screen *ebiten.Image){
            drawBox(screen, controlsRect, getAlpha(), controlsInside)
            printInBox(screen, controlsRect.Min.X + 5, controlsRect, getAlpha(), displaySettings.Controls().Name())
        },
    })

    // how units strike in a battle, see display/strikes.go
    addCheckbox(group, fonts, &getAlpha, SettingsLeftColumnX, SettingsRowY(SingleStrikesRow), "Single strikes",
        func() bool { return displaySettings.SingleStrikes() },
        func(value bool) {
            displaySettings.SetSingleStrikes(value)
        },
    )

    // the cursor of a battle after a click that starts an attack, see display/cursorhide.go
    addCheckbox(group, fonts, &getAlpha, SettingsLeftColumnX, SettingsRowY(HideCursorRow), "Hide cursor on attack",
        func() bool { return displaySettings.HideCursorOnAttack() },
        func(value bool) {
            displaySettings.SetHideCursorOnAttack(value)
        },
    )

    if OptionsStartWithRolloutOpen && !displaySettings.Fullscreen {
        openRollout()
    }


    return group, quit, closeScreen
}
