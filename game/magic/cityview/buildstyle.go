package cityview

// The screen that picks what a city builds, matched to the original game.
//
// Every number here is the original's: fonts, palette indexes and places, read from the ReMoM
// project's reconstruction (MoM/src/ProdScr.c: Production_Screen_Draw_, Production_Screen_Add_Fields;
// MoM/src/UnitStat.c: Thing_View_Draw, the view of a building) and checked against the game's
// palette. The code is ours. The text of the window itself is the one of the window of a unit,
// see unitview/style.go.

import (
    "fmt"
    "image"
    "image/color"
    "log"
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/game/magic/unitview"
    "github.com/kazzmir/master-of-magic/lib/font"
    "github.com/kazzmir/master-of-magic/lib/lbx"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: the screen of upstream
var OriginalBuildScreen = true

// development: the city screen opens the build screen at once. "unit" and "building" show the
// first unit or building of the lists, anything else what the city builds. see -capture-screen build
var CaptureBuild = ""

// the window in the middle, from the left edge of the screen
const buildWindowX = 74
// everything in the window is this far right of where it is in the window of a unit
const buildWindowShift = 4

// places, in original pixels of the screen
const (
    // the lists: buildings on the left, units on the right
    buildListY = 4
    buildListStep = 14
    buildListLeftX = 0
    buildListRightX = 240
    buildListTextY = 1
    buildListLeftTextX = 2
    buildListRightTextX = 3

    // the picture of a building is put in the middle of a field of this size
    buildPictureX = buildWindowX + 6
    buildPictureY = 4
    buildPictureWidth = 44
    buildPictureHeight = 36

    buildNameX = buildWindowX + buildWindowShift + 51
    buildNameY = 11
    // a name wider than this goes on two lines, which start higher
    buildNameWidth = 100
    buildNameTwoLinesY = 7
    // the name of a unit is higher in this screen
    buildUnitNameY = 7

    buildCostY = 34
    buildCostValueX = buildWindowX + buildWindowShift + 73

    buildLabelX = buildWindowX + buildWindowShift + 8
    buildMaintenanceY = 47
    // no maintenance: the 0 ends here
    buildMaintenanceZeroX = buildWindowX + buildWindowShift + 92
    buildMaintenanceZeroY = 48
    buildCoinX = buildWindowX + buildWindowShift + 89
    buildCoinY = 49
    buildCoinStep = 9

    buildAllowsY = 58
    buildAllowsTextX = buildWindowX + buildWindowShift + 60
    buildAllowsTextY = 59
    buildAllowsWidth = 100

    buildDescriptionX = buildWindowX + buildWindowShift + 9
    buildDescriptionY = 107
    buildDescriptionWidth = 147
    // the field of the description ends above the buttons
    buildDescriptionBottom = 178

    // the statistics and the list of a unit: the origin upstream's functions of the window of a
    // unit are given, which puts the names of the statistics at buildLabelX, 47
    buildStatsX = buildLabelX + 2
    buildStatsY = 50
    buildAbilitiesX = 85
    buildAbilitiesY = 108
    buildUnitPictureX = 103
    buildUnitPictureY = 28

    buildCancelX = 99
    buildOkX = 172
    buildButtonY = 181
)

// the space between words, in art pixels
const (
    buildNameSpace = 1
    buildDescriptionSpace = 2
)

// true: the descriptions of the settings that are no buildings (Housing, Trade Goods) are printed
// smaller than the original has them, every line in the middle of their field. TRIED AND TAKEN BACK
// by the user (2026-09-29): "in this case the text was better large"
const BuildSettingsCompact = false
var BuildSettingTextSize = font.TextSizeCompact
// true: the descriptions of all buildings are printed that way
const BuildAllDescriptionsCompact = false

// palette indexes of the names in the lists: the color of the soft edge of the letters, the
// color of the letters, the color of their shadow below and right
var buildTabColors = []int{251, 1}
const buildTabShadow = 244
var buildTabPickedColors = []int{244, 255}
const buildTabPickedShadow = 248
const buildTabFontIndex = 2
const buildTabSpace = 1

// the picture that carries the palette colors 224 and up of the screen
const buildPaletteLbx = "unitview.lbx"
const buildPaletteIndex = 0

type buildStyle struct {
    // a name in the lists, and the one that is picked
    Tab *font.StyledFont
    TabPicked *font.StyledFont
    // the text of the window
    Title *font.StyledFont
    Small *font.StyledFont
    List *font.StyledFont
}

var buildStyles = make(map[*lbx.LbxCache]*buildStyle)

// the styles of the text, nil when they can not be made. then the screen of upstream is shown
func getBuildStyle(cache *lbx.LbxCache) *buildStyle {
    if !OriginalBuildScreen || cache == nil {
        return nil
    }

    style, ok := buildStyles[cache]
    if ok {
        return style
    }

    // also when it fails, so it is tried once
    buildStyles[cache] = nil

    title, small, list := unitview.ViewFonts(cache)
    if title == nil || small == nil || list == nil {
        return nil
    }

    fontsLbx, err := cache.GetLbxFile("fonts.lbx")
    if err != nil {
        log.Printf("Build screen: unable to read fonts.lbx: %v", err)
        return nil
    }

    lbxFonts, err := font.ReadFonts(fontsLbx, 0)
    if err != nil || len(lbxFonts) <= buildTabFontIndex {
        log.Printf("Build screen: unable to read the fonts: %v", err)
        return nil
    }

    paletteLbx, err := cache.GetLbxFile(buildPaletteLbx)
    if err != nil {
        log.Printf("Build screen: unable to read %v: %v", buildPaletteLbx, err)
        return nil
    }

    palette, err := paletteLbx.GetPalette(buildPaletteIndex)
    if err != nil || len(palette) < 256 {
        log.Printf("Build screen: unable to read the palette of the screen: %v", err)
        return nil
    }

    make := func(indexes []int, shadow int) *font.StyledFont {
        var colors []color.Color
        for _, index := range indexes {
            colors = append(colors, palette[index])
        }
        styled := font.MakeStyledFont(lbxFonts[buildTabFontIndex], colors, font.ShadowDown, palette[shadow])
        styled.SpaceWidth = buildTabSpace
        return styled
    }

    style = &buildStyle{
        Tab: make(buildTabColors, buildTabShadow),
        TabPicked: make(buildTabPickedColors, buildTabPickedShadow),
        Title: title,
        Small: small,
        List: list,
    }

    buildStyles[cache] = style
    return style
}

func printBuild(styled *font.StyledFont, screen *ebiten.Image, x int, y int, text string) {
    styled.Print(screen, x, y, font.FontOptions{Scale: scale.ScaleAmount}, text)
}

// a style with another space between its words
func withSpace(styled *font.StyledFont, space int) *font.StyledFont {
    out := *styled
    out.SpaceWidth = space
    return &out
}

// the name of a thing in a list
func (style *buildStyle) drawTabName(screen *ebiten.Image, x int, y int, right bool, picked bool, name string) {
    use := style.Tab
    if picked {
        use = style.TabPicked
    }

    textX := x + buildListLeftTextX
    if right {
        textX = x + buildListRightTextX
    }

    printBuild(use, screen, textX, y + buildListTextY, name)
}

// the list of what a building allows, as the original writes it: with commas, "and" before the
// last, and a full stop
func allowsText(names []string) string {
    if len(names) == 0 {
        return ""
    }

    return combineStrings(names) + "."
}

// where the picture of a building goes so that it is in the middle of its field
func buildingPicturePlace(picture image.Rectangle) (int, int) {
    return buildPictureX + (buildPictureWidth - picture.Dx()) / 2, buildPictureY + (buildPictureHeight - picture.Dy()) / 2
}

// name, cost, maintenance, what it allows and the description of a building
func (style *buildStyle) drawBuilding(screen *ebiten.Image, name string, cost int, maintenance int, coin *ebiten.Image, allows string, description string, setting bool) {
    title := withSpace(style.Title, buildNameSpace)

    if title.Width(name) > buildNameWidth {
        for index, line := range title.Wrap(name, buildNameWidth, 1) {
            printBuild(title, screen, buildNameX, buildNameTwoLinesY + index * (title.Height() + 1), line)
        }
    } else {
        printBuild(title, screen, buildNameX, buildNameY, name)
    }

    printBuild(style.Small, screen, buildNameX, buildCostY, "Cost")
    printBuild(style.Small, screen, buildCostValueX, buildCostY, fmt.Sprintf("%v", cost))

    printBuild(title, screen, buildLabelX, buildMaintenanceY, "Maintenance")
    if maintenance == 0 {
        style.List.Print(screen, buildMaintenanceZeroX, buildMaintenanceZeroY, font.FontOptions{Scale: scale.ScaleAmount, Justify: font.FontJustifyRight}, "0")
    } else if coin != nil {
        for index := 0; index < maintenance; index++ {
            var options ebiten.DrawImageOptions
            options.GeoM.Translate(float64(buildCoinX + index * buildCoinStep), float64(buildCoinY))
            scale.DrawScaled(screen, coin, &options)
        }
    }

    printBuild(title, screen, buildLabelX, buildAllowsY, "Allows")
    for index, line := range style.List.Wrap(allows, buildAllowsWidth, 1) {
        printBuild(style.List, screen, buildAllowsTextX, buildAllowsTextY + index * (style.List.Height() + 1), line)
    }

    style.drawDescription(screen, description, setting)
}

// the description of a building, in the lower field of the window
func (style *buildStyle) drawDescription(screen *ebiten.Image, description string, setting bool) {
    text := withSpace(style.Title, buildDescriptionSpace)

    if !(setting && BuildSettingsCompact) && !BuildAllDescriptionsCompact {
        // as the original has it
        for index, line := range text.Wrap(description, buildDescriptionWidth, 1) {
            printBuild(text, screen, buildDescriptionX, buildDescriptionY + index * (text.Height() + 1), line)
        }
        return
    }

    // smaller, every line in the middle of the field and the lines in the middle of its height
    size := font.RelativeTextSize(scale.ScaleAmount, BuildSettingTextSize)
    lines := text.Wrap(description, buildDescriptionWidth, size)
    step := float64(text.Height() + 1) * size
    // PrintSized puts smaller letters in the middle of a line of the full size
    lead := float64(text.Height()) * (1 - size) / 2

    height := float64(len(lines)) * step - size
    top := float64(buildDescriptionY) + (float64(buildDescriptionBottom - buildDescriptionY) - height) / 2
    middle := float64(buildDescriptionX) + float64(buildDescriptionWidth) / 2

    for index, line := range lines {
        x := middle - text.WidthAt(line, size) / 2
        y := top + float64(index) * step - lead
        // on whole pixels of the screen
        x = math.Round(x * scale.ScaleAmount) / scale.ScaleAmount
        text.PrintSized(screen, x, y, font.FontOptions{Scale: scale.ScaleAmount, TextSize: BuildSettingTextSize}, line)
    }
}
