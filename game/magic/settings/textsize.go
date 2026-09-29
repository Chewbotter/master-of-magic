package settings

// The size of the text of the settings screen.
//
// Small (user, 2026-09-28): more settings are to come, and small text leaves room for them. The
// places of the rows are as they were; a line of small text stands at the middle of the height the
// full line had, so it is still in line with its checkbox or its box.

import (
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/lib/font"

    "github.com/hajimehoshi/ebiten/v2"
)

// a named size of lib/font/textsize.go
const SettingsTextSize = font.TextSizeSmall

// how the text of the settings screen is printed
func settingsText(options *ebiten.DrawImageOptions) font.FontOptions {
    return font.FontOptions{Scale: scale.ScaleAmount, DropShadow: true, Options: options, TextSize: SettingsTextSize}
}

// where a line of the text of the settings screen starts, for a line of full size that started at y
func settingsTextY(use *font.Font, y int) float64 {
    if use == nil {
        return float64(y)
    }

    full := float64(use.Height())
    small := full * font.RelativeTextSize(scale.ScaleAmount, SettingsTextSize)
    return float64(y) + (full - small) / 2
}
