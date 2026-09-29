package settings

// The text and the rows of the settings screen.
//
// Small text, small checkboxes and rows close together (user, 2026-09-28): more settings are to
// come, and this way many fit below each other. Both columns use the same rows: a setting is put
// on a row with SettingsRowY, never at a place of its own.
//
// A line of text stands at the middle of what it belongs to, a checkbox or a box.

import (
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/lib/font"

    "github.com/hajimehoshi/ebiten/v2"
)

// a named size of lib/font/textsize.go
const SettingsTextSize = font.TextSizeSmall

// art pixels. a checkbox, how far its mark is from its rim, and how far its label is from it.
// upstream's checkbox was 12 with its mark 3 from the rim and its label 6 from it
const SettingsCheckboxSize = 7
const SettingsCheckboxMark = 2
const SettingsLabelGap = 4

// the rows of settings: where the first starts and how far they are apart. upstream's rows
// started at 84 and were 22 apart
const SettingsFirstRowY = 76
const SettingsRowHeight = 11
// the left edge of the left column. the right one is DisplayColumnX
const SettingsLeftColumnX = 30
// a box on a row, as the one of the controls, is this high
const SettingsRowBoxHeight = 9

// where a row of settings starts, counted from 0
func SettingsRowY(row int) int {
    return SettingsFirstRowY + row * SettingsRowHeight
}

// where a box on a row starts, its middle at the middle of a checkbox of the row
func settingsRowBoxTop(row int) int {
    return SettingsRowY(row) - (SettingsRowBoxHeight - SettingsCheckboxSize) / 2
}

// how the text of the settings screen is printed
func settingsText(options *ebiten.DrawImageOptions) font.FontOptions {
    return font.FontOptions{Scale: scale.ScaleAmount, DropShadow: true, Options: options, TextSize: SettingsTextSize}
}

// how high a line of the text of the settings screen is
func settingsTextHeight(use *font.Font) float64 {
    if use == nil {
        return 0
    }
    return float64(use.Height()) * font.RelativeTextSize(scale.ScaleAmount, SettingsTextSize)
}

// where a line of the text of the settings screen starts, for a line of full size that started at y
func settingsTextY(use *font.Font, y int) float64 {
    if use == nil {
        return float64(y)
    }
    return float64(y) + (float64(use.Height()) - settingsTextHeight(use)) / 2
}

// where a line of the text of the settings screen starts that has its middle at a height
func settingsTextMiddleY(use *font.Font, middle float64) float64 {
    return middle - settingsTextHeight(use) / 2
}
