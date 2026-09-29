package settings

// The boxes of the settings screen, made of art pixels (user, 2026-09-28: "pixel style").
//
// Upstream drew them with lines of one screen pixel, which get thinner against the art the
// larger the window is. Here a rim is one art pixel wide and everything starts and ends on
// whole art pixels.

import (
    "image"
    "image/color"

    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/vector"
)

// art pixels
const SettingsRimWidth = 1

// upstream's colors: checkboxes and the bar of the volume
var settingsDarkFill = color.NRGBA{R: 32, G: 32, B: 32, A: 200}
var settingsLightRim = color.NRGBA{R: 255, G: 255, B: 255, A: 200}
var settingsMark = color.NRGBA{R: 255, G: 255, B: 255, A: 220}
// boxes that are clicked: Resolution, Controls, Keys
var settingsBoxFill = color.NRGBA{R: 96, G: 60, B: 20, A: 255}
var settingsBoxLit = color.NRGBA{R: 140, G: 90, B: 30, A: 255}
var settingsBoxRim = color.NRGBA{R: 255, G: 200, B: 100, A: 255}

func fillArtRect(screen *ebiten.Image, rect image.Rectangle, fill color.NRGBA, alpha float32) {
    if rect.Empty() {
        return
    }
    fill.A = uint8(float32(fill.A) * alpha)
    vector.FillRect(screen, float32(scale.Scale(rect.Min.X)), float32(scale.Scale(rect.Min.Y)), float32(scale.Scale(rect.Dx())), float32(scale.Scale(rect.Dy())), fill, false)
}

// the four sides of a rim inside of a rectangle, none over another
func rimRects(rect image.Rectangle, width int) []image.Rectangle {
    return []image.Rectangle{
        image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Min.Y + width),
        image.Rect(rect.Min.X, rect.Max.Y - width, rect.Max.X, rect.Max.Y),
        image.Rect(rect.Min.X, rect.Min.Y + width, rect.Min.X + width, rect.Max.Y - width),
        image.Rect(rect.Max.X - width, rect.Min.Y + width, rect.Max.X, rect.Max.Y - width),
    }
}

// a filled box with a rim
func drawArtBox(screen *ebiten.Image, rect image.Rectangle, fill color.NRGBA, rim color.NRGBA, alpha float32) {
    fillArtRect(screen, rect.Inset(SettingsRimWidth), fill, alpha)
    for _, side := range rimRects(rect, SettingsRimWidth) {
        fillArtRect(screen, side, rim, alpha)
    }
}
