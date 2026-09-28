package main

// Test Battle: a battle of one kind of unit against itself, picked from a list, for looking at
// changes to the pictures of a unit. One of the debug options of the start screen, see fastplay.go.
//
// The list has a row per race that opens and closes (a rollout), with the units of the race under
// it. It is drawn in the small text size over the whole window, in as many columns as it needs.

import (
    "fmt"
    "image/color"
    "slices"
    "strings"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/game/magic/inputmanager"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    "github.com/kazzmir/master-of-magic/lib/font"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/inpututil"
    "github.com/hajimehoshi/ebiten/v2/vector"
)

// the text size of the list
const UnitPickerTextSize = font.TextSizeSmall

// where the list starts and ends, in art pixels of the picture of the window
const unitPickerLeft = 6.0
const unitPickerTop = 6.0
const unitPickerBottom = 196.0
// art pixels between rows, between columns, and how far units stand in from their race
const unitPickerRowGap = 1.5
const unitPickerColumnGap = 8.0
const unitPickerIndent = 5.0
// how dark the start screen behind the list is, of 255
const unitPickerShade = 215

// units of each army of a test battle, times the army size of the debug list
const TestBattleUnits = 4

// the races of the list in their order. the races of the towns first
var unitPickerRaces = []data.Race{
    data.RaceBarbarian, data.RaceBeastmen, data.RaceDarkElf, data.RaceDraconian, data.RaceDwarf, data.RaceGnoll, data.RaceHalfling,
    data.RaceHighElf, data.RaceHighMen, data.RaceKlackon, data.RaceLizard, data.RaceNomad, data.RaceOrc, data.RaceTroll,
    data.RaceFantastic, data.RaceHero, data.RaceAll,
}

var unitPickerOpen bool
var unitPickerExpanded = make(map[data.Race]bool)
// the row under the cursor, -1 for none
var unitPickerHover = -1

// the unit of the test battle that is about to start, and the one of the last
var testBattleUnit *units.Unit
var testBattleLast *units.Unit

type unitPickerRow struct {
    Text string
    // what a click does: one of these is set
    Back bool
    Race data.Race
    Unit *units.Unit

    // in art pixels of the picture of the window
    X float64
    Y float64
    Width float64
    Height float64
}

func (row unitPickerRow) contains(x float64, y float64) bool {
    return x >= row.X - 1 && x < row.X + row.Width + 1 && y >= row.Y - unitPickerRowGap / 2 && y < row.Y + row.Height + unitPickerRowGap / 2
}

var unitPickerUnits map[data.Race][]*units.Unit

// the units that have figures for battles, by race, by name
func unitsOfRace(race data.Race) []*units.Unit {
    if unitPickerUnits == nil {
        unitPickerUnits = make(map[data.Race][]*units.Unit)
        seen := make(map[string]bool)

        for index := range units.AllUnits {
            unit := &units.AllUnits[index]
            key := fmt.Sprintf("%v/%v", unit.CombatLbxFile, unit.CombatIndex)
            if unit.CombatLbxFile == "" || seen[key] {
                continue
            }
            seen[key] = true
            unitPickerUnits[unit.Race] = append(unitPickerUnits[unit.Race], unit)
        }

        for _, list := range unitPickerUnits {
            slices.SortFunc(list, func(a *units.Unit, b *units.Unit) int {
                return strings.Compare(a.Name, b.Name)
            })
        }
    }

    return unitPickerUnits[race]
}

// the font of the list. the smallest font of the game is hard to read at this size, its w and m
// lose their shape: the next one is used, which the debug list has in white
func unitPickerFont() *font.Font {
    return fastPlayFontHighlight
}

// the color of the units in the list, the one of the debug list
var unitPickerUnitColor = color.RGBA{R: 0xc7, G: 0x82, B: 0x1b, A: 0xff}

func unitPickerOptions() font.FontOptions {
    return font.FontOptions{Scale: scale.ScaleAmount, TextSize: UnitPickerTextSize}
}

// the rows of the list with their places. they run down and then into the next column
func unitPickerRows() []unitPickerRow {
    options := unitPickerOptions()
    height := float64(unitPickerFont().Height()) * font.RelativeTextSize(scale.ScaleAmount, UnitPickerTextSize)

    var rows []unitPickerRow
    rows = append(rows, unitPickerRow{Text: "Back", Back: true})

    for _, race := range unitPickerRaces {
        list := unitsOfRace(race)
        if len(list) == 0 {
            continue
        }

        mark := "+"
        if unitPickerExpanded[race] {
            mark = "-"
        }
        rows = append(rows, unitPickerRow{Text: fmt.Sprintf("%v %v (%v)", mark, race, len(list)), Race: race})

        if unitPickerExpanded[race] {
            for _, unit := range list {
                rows = append(rows, unitPickerRow{Text: unit.Name, Unit: unit, Race: race})
            }
        }
    }

    x := unitPickerLeft
    // under the title
    y := unitPickerTop + height + unitPickerRowGap * 3
    top := y
    columnWidth := 0.0

    for index := range rows {
        row := &rows[index]
        row.Height = height
        row.Width = unitPickerFont().TextWidth(row.Text, options)

        if y + height > unitPickerBottom {
            x += columnWidth + unitPickerColumnGap
            y = top
            columnWidth = 0
        }

        row.X = x
        if row.Unit != nil {
            row.X += unitPickerIndent
        }
        row.Y = y

        columnWidth = max(columnWidth, row.X - x + row.Width)
        y += height + unitPickerRowGap
    }

    return rows
}

// the cursor in art pixels of the picture of the window, with fractions
func unitPickerCursor() (float64, float64) {
    x, y := display.CursorPosition()
    x += display.ContentOffsetX() - display.PictureX()
    return float64(x) / scale.ScaleAmount, float64(y) / scale.ScaleAmount
}

// one tick of the list. true when a unit was picked: testBattleUnit is the unit
func updateUnitPicker() bool {
    unitPickerHover = -1

    if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
        unitPickerOpen = false
        return false
    }

    rows := unitPickerRows()
    x, y := unitPickerCursor()
    for index, row := range rows {
        if row.contains(x, y) {
            unitPickerHover = index
        }
    }

    if unitPickerHover < 0 || !inputmanager.LeftClick() {
        return false
    }

    row := rows[unitPickerHover]
    switch {
        case row.Back:
            unitPickerOpen = false
        case row.Unit != nil:
            testBattleUnit = row.Unit
            unitPickerOpen = false
            return true
        default:
            unitPickerExpanded[row.Race] = !unitPickerExpanded[row.Race]
    }

    return false
}

func drawUnitPicker(screen *ebiten.Image) {
    vector.FillRect(screen, 0, 0, float32(screen.Bounds().Dx()), float32(screen.Bounds().Dy()), color.RGBA{A: unitPickerShade}, false)

    // positions are in art pixels of the picture of the window, so add its margin
    offsetX := float64(display.PictureX()) / scale.ScaleAmount
    offsetY := float64(display.ContentOffsetY()) / scale.ScaleAmount
    options := unitPickerOptions()

    unitPickerFont().PrintOutlined(screen, unitPickerLeft + offsetX, unitPickerTop + offsetY, options, font.OutlineFull, "Test Battle: pick a unit. It fights its own kind.")

    for index, row := range unitPickerRows() {
        rowOptions := options

        // races and the row under the cursor are white, units are in the color of the debug list
        if index != unitPickerHover && (row.Unit != nil || row.Back) {
            var tint ebiten.DrawImageOptions
            tint.ColorScale.ScaleWithColor(unitPickerUnitColor)
            rowOptions.Options = &tint
        }

        unitPickerFont().PrintOutlined(screen, row.X + offsetX, row.Y + offsetY, rowOptions, font.OutlineFull, row.Text)
    }
}

// the name of a unit with its race, as the folders of its pictures are named
func unitFullName(unit *units.Unit) string {
    return fmt.Sprintf("%v %v", unit.Race, unit.Name)
}

// development: the unit of a name as unitFullName gives it, or of its name alone
func findUnit(name string) *units.Unit {
    for _, race := range unitPickerRaces {
        for _, unit := range unitsOfRace(race) {
            if strings.EqualFold(unitFullName(unit), name) || strings.EqualFold(unit.Name, name) {
                return unit
            }
        }
    }
    return nil
}
