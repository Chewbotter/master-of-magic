package settings

// What a setting does, shown in a small panel next to the mouse that rests on it (user,
// 2026-09-28). The texts are in one place, settingTips, by the name of the setting.
//
// The interface of upstream has a tooltip of its own (UIElement.Tooltip): one line of letters at
// the mouse without a panel behind it, at the full text size, drawn by the screen that shows the
// interface. This one is an element of the settings screen itself, so it is the same wherever the
// screen is shown.

import (
    "image"
    "image/color"

    "github.com/kazzmir/master-of-magic/lib/font"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    fontslib "github.com/kazzmir/master-of-magic/game/magic/fonts"
    "github.com/kazzmir/master-of-magic/game/magic/util"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    uilib "github.com/kazzmir/master-of-magic/game/magic/ui"

    "github.com/hajimehoshi/ebiten/v2"
)

// one switch: false shows no panels
var SettingsTips = true

// ticks of the game the mouse rests on a setting before its panel shows, and the ticks it takes
// to show fully
const SettingsTipDelay = 25
const SettingsTipFade = 8
// art pixels. how wide the text of a panel is at most, the room between text and rim, how far
// the panel is from the mouse, and how close it comes to the edge of the screen
const SettingsTipWidth = 110
const SettingsTipPadding = 3
// 16 is the width of the picture of a cursor (8 first, the cursor lay over the panel)
const SettingsTipBesideMouse = 16
const SettingsTipBelowMouse = 10
const SettingsTipScreenEdge = 2
// drawn after everything else of the screen
const settingsTipOrder = 100

var settingsTipFill = color.NRGBA{R: 20, G: 16, B: 28, A: 240}
var settingsTipRim = color.NRGBA{R: 255, G: 200, B: 100, A: 255}

// by the name of the setting as the screen shows it
var settingTips = map[string]string{
    "Volume": "How loud the music is.",
    "End Of Turn Wait": "On: a turn ends when you press Next Turn. Off: it also ends by itself once all of your units have moved.",
    "Strategic Combat Only": "Your battles are decided at once, without the battlefield.",
    "Background Music": "Plays music.",
    "Random Events": "Events such as plagues and conjunctions can start. Events under way run their course when this is turned off.",
    "Hide cursor on attack": "In a battle the cursor fades away when you click to attack or shoot, and is back when you move the mouse. With the modern controls.",
    "Pulsing spellbook text": "In the spellbooks the spell under the mouse pulses to the color of its realm. Off: nothing pulses, as in the original.",
    "Damage numbers": "In a battle a number rises from a unit that is hit: bright red over the enemy's units, darker over yours.",
    "Single strikes": "How units attack in a battle. On: one heavy swing per attack. Off: fast strikes again and again.",
    "Resolution": "The size of the window. Not used in fullscreen.",
    "Fullscreen": "Fills the desktop, without a window border.",
    "Widescreen": "The picture is as wide as the window. Off: the shape of the original, with bars at the sides.",
    "Controls": "Modern: drag with the middle mouse button to move the map. Classic: the controls of the original.",
    "Keys": "Shows and changes the keys.",
    "Enemy AI": "Chewbot: the original game's AI, ported, then changed (battles so far). Clone original: the AI this remake came with.",
    "Aggressive AI": "Computer wizards settle earlier and attack towns more readily. For this session only.",
}

// what the mouse rests on
type settingsTipState struct {
    Name string
    // ticks of the game: when the mouse came to it and when it was seen on it last
    Since int64
    Last int64
    // where the mouse is, art pixels
    Mouse image.Point
}

// one settings screen is shown at a time
var settingsTip settingsTipState

// the mouse is on a setting
func (tip *settingsTipState) hover(name string, mouse image.Point, tick int64) {
    // a tick without the mouse on it starts the wait again
    if name != tip.Name || tick - tip.Last > 1 {
        tip.Name = name
        tip.Since = tick
    }
    tip.Last = tick
    tip.Mouse = mouse
}

// how much of the panel shows, 0 to 1
func (tip *settingsTipState) shows(tick int64) float32 {
    if tip.Name == "" || tick - tip.Last > 1 {
        return 0
    }
    rested := tick - tip.Since - SettingsTipDelay
    if rested <= 0 {
        return 0
    }
    if rested >= SettingsTipFade {
        return 1
    }
    return float32(rested) / SettingsTipFade
}

// where the panel of a size goes for a mouse: below and right of it, kept on the screen, above
// the mouse when there is no room below
func settingsTipPlace(mouse image.Point, width int, height int) image.Point {
    x := mouse.X + SettingsTipBesideMouse
    y := mouse.Y + SettingsTipBelowMouse

    x = max(SettingsTipScreenEdge, min(x, data.ScreenWidth - SettingsTipScreenEdge - width))
    if y + height > data.ScreenHeight - SettingsTipScreenEdge {
        y = mouse.Y - SettingsTipBelowMouse / 2 - height
    }
    y = max(SettingsTipScreenEdge, y)

    return image.Pt(x, y)
}

// gives an element the panel of a setting: call it from Inside of the element
func hoverTip(name string, element *uilib.UIElement, x int, y int) {
    settingsTip.hover(name, image.Pt(element.Rect.Min.X + x, element.Rect.Min.Y + y), ebiten.Tick())
}

// an area that only shows the panel of a setting, as the name next to a checkbox
func addTipArea(group *uilib.UIElementGroup, rect image.Rectangle, name string) {
    group.AddElement(&uilib.UIElement{
        Layer: settingsLayer,
        Rect: rect,
        Inside: func(element *uilib.UIElement, x int, y int){
            hoverTip(name, element, x, y)
        },
    })
}

// how wide a line of the text of the settings screen is, art pixels rounded up
func settingsTextWidth(use *font.Font, text string) int {
    if use == nil {
        return 0
    }
    var options ebiten.DrawImageOptions
    width := use.TextWidth(text, settingsText(&options))
    whole := int(width)
    if float64(whole) < width {
        whole += 1
    }
    return whole
}

// the element that draws the panel
func addTips(group *uilib.UIElementGroup, fonts *fontslib.SettingsFonts, getAlpha *util.AlphaFadeFunc) {
    settingsTip = settingsTipState{}

    // the lines of a text are made once
    wrapped := make(map[string]font.WrappedText)
    relative := font.RelativeTextSize(scale.ScaleAmount, SettingsTextSize)

    group.AddElement(&uilib.UIElement{
        Layer: settingsLayer,
        Order: settingsTipOrder,
        Draw: func(element *uilib.UIElement, screen *ebiten.Image){
            if !SettingsTips || fonts.OptionFont == nil {
                return
            }

            alpha := settingsTip.shows(ebiten.Tick()) * (*getAlpha)()
            text := settingTips[settingsTip.Name]
            if alpha <= 0 || text == "" {
                return
            }

            lines, ok := wrapped[text]
            if !ok {
                lines = fonts.OptionFont.CreateWrappedText(SettingsTipWidth, relative, text)
                wrapped[text] = lines
            }

            width := 0
            for _, line := range lines.Lines {
                width = max(width, settingsTextWidth(fonts.OptionFont, line))
            }
            lineHeight := (float64(fonts.OptionFont.Height()) + 1) * relative
            height := int(lineHeight * float64(len(lines.Lines)) + 0.999)

            place := settingsTipPlace(settingsTip.Mouse, width + SettingsTipPadding * 2, height + SettingsTipPadding * 2)
            panel := image.Rect(place.X, place.Y, place.X + width + SettingsTipPadding * 2, place.Y + height + SettingsTipPadding * 2)

            drawArtBox(screen, panel, settingsTipFill, settingsTipRim, alpha)

            var options ebiten.DrawImageOptions
            options.ColorScale.ScaleAlpha(alpha)
            fonts.OptionFont.RenderWrapped(screen, float64(panel.Min.X + SettingsTipPadding), float64(panel.Min.Y + SettingsTipPadding), lines, settingsText(&options))
        },
    })
}
