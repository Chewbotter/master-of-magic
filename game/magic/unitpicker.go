package main

// Test Battle: a battle of one kind of unit against itself, picked from a list, for looking at
// changes to the pictures of a unit. One of the debug options of the start screen, see fastplay.go.
//
// The list has a row per race that opens and closes (a rollout), with the units of the race under
// it. It is drawn in the small text size over the whole window, in as many columns as it needs.
//
// Above the races are the ground of the battle and its coast (user, 2026-09-29), rollouts as the
// races are: a landscape or one of its biomes (combat/biomes.go), and the side the sea lies on
// (combat/coast.go), its river (combat/river.go) and the fields of a town (combat/farmland.go). A click on one of them picks it, the battle
// starts with the click on a unit.
// They are kept for the next battle and the next run of the game, see debugsaved.go.

import (
    "fmt"
    "image/color"
    "math"
    "slices"
    "strings"

    "github.com/kazzmir/master-of-magic/game/magic/combat"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/game/magic/inputmanager"
    "github.com/kazzmir/master-of-magic/game/magic/mod"
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

// the button "Test": art pixels between its rim and its text, left and right, above and below
const unitPickerButtonPadX = 4.0
const unitPickerButtonPadY = 2.0
// its rim, in art pixels
const unitPickerButtonRim = 1.0
// its face, the rim above and left, the rim below and right; and the same under the mouse
var unitPickerButtonFace = color.RGBA{R: 0x6a, G: 0x42, B: 0x0c, A: 0xff}
var unitPickerButtonLight = color.RGBA{R: 0xc7, G: 0x82, B: 0x1b, A: 0xff}
var unitPickerButtonDark = color.RGBA{R: 0x2e, G: 0x1c, B: 0x04, A: 0xff}
var unitPickerButtonFaceOver = color.RGBA{R: 0x9a, G: 0x62, B: 0x14, A: 0xff}
var unitPickerButtonLightOver = color.RGBA{R: 0xf2, G: 0xb4, B: 0x4a, A: 0xff}

// units of each army of a test battle, times the army size of the debug list
const TestBattleUnits = 4

// the races of the list in their order. the races of the towns first
var unitPickerRaces = []data.Race{
    data.RaceBarbarian, data.RaceBeastmen, data.RaceDarkElf, data.RaceDraconian, data.RaceDwarf, data.RaceGnoll, data.RaceHalfling,
    data.RaceHighElf, data.RaceHighMen, data.RaceKlackon, data.RaceLizard, data.RaceNomad, data.RaceOrc, data.RaceTroll,
    data.RaceFantastic, data.RaceHero, data.RaceAll,
}

// a ground a test battle can be fought on: a landscape, or a biome of one
type testGround struct {
    Name string
    Landscape combat.CombatLandscape
    Biome string
}

var testGrounds = []testGround{
    {Name: "Grass", Landscape: combat.CombatLandscapeGrass},
    {Name: mod.BiomeForest, Landscape: combat.CombatLandscapeGrass, Biome: mod.BiomeForest},
    {Name: mod.BiomeSwamp, Landscape: combat.CombatLandscapeGrass, Biome: mod.BiomeSwamp},
    {Name: mod.BiomeHills, Landscape: combat.CombatLandscapeGrass, Biome: mod.BiomeHills},
    {Name: "Desert", Landscape: combat.CombatLandscapeDesert},
    {Name: "Mountain", Landscape: combat.CombatLandscapeMountain},
    {Name: mod.BiomeVolcano, Landscape: combat.CombatLandscapeMountain, Biome: mod.BiomeVolcano},
    {Name: mod.BiomeSnowyMountain, Landscape: combat.CombatLandscapeMountain, Biome: mod.BiomeSnowyMountain},
    {Name: "Tundra", Landscape: combat.CombatLandscapeTundra},
}

// the sides of a coast with their names in the list, in the order of the list
var testCoasts = []combat.CoastSide{combat.CoastNone, combat.CoastAny, combat.CoastNorth, combat.CoastEast, combat.CoastSouth, combat.CoastWest}
var testCoastNames = map[combat.CoastSide]string{
    combat.CoastNone: "None",
    combat.CoastAny: "Any, a side by chance",
    combat.CoastNorth: "North, upper right",
    combat.CoastEast: "East, lower right",
    combat.CoastSouth: "South, lower left",
    combat.CoastWest: "West, upper left",
}

// the courses of a river with their names in the list, in the order of the list
var testRivers = []combat.RiverCourse{combat.RiverNone, combat.RiverAny, combat.RiverAcross, combat.RiverBeside}
var testRiverNames = map[combat.RiverCourse]string{
    combat.RiverNone: "None",
    combat.RiverAny: "Any, a course by chance",
    combat.RiverAcross: "Between the armies",
    combat.RiverBeside: "Beside the armies",
}

// the fields of a town with their names in the list, by how far the town is
var testFarmlandNames = []string{"None", "Next to a town, more houses", "Two tiles from a town"}

// the weather of a test battle with its names in the list (combat/weather.go)
var testWeatherNames = map[combat.Weather]string{
    combat.WeatherNone: "None",
    combat.WeatherLightRain: "Light rain",
    combat.WeatherHeavyRain: "Heavy rain",
    combat.WeatherLightSnow: "Light snow",
    combat.WeatherHeavySnow: "Heavy snow",
    combat.WeatherClouds: "Cloud shadows",
}
var testBattleWeather = combat.WeatherNone
var unitPickerWeatherOpen bool

// the race of the town of the fields of a test battle: the one of the unit that fights
var testBattleFarmland = combat.FarmlandNone
var unitPickerFarmlandOpen bool

// the ground, the coast and the river of the test battles: which of testGrounds, the side, the
// course
var testBattleGround = 0
var testBattleCoast = combat.CoastNone
var testBattleRiver = combat.RiverNone

// the rollouts of the ground, of the coast and of the river
var unitPickerGroundOpen bool
var unitPickerCoastOpen bool
var unitPickerRiverOpen bool

func testGroundByName(name string) int {
    for index, ground := range testGrounds {
        if strings.EqualFold(ground.Name, name) {
            return index
        }
    }
    return 0
}

func testCoastByName(name string) combat.CoastSide {
    for _, side := range testCoasts {
        if strings.EqualFold(side.String(), name) {
            return side
        }
    }
    return combat.CoastNone
}

func testRiverByName(name string) combat.RiverCourse {
    for _, course := range testRivers {
        if strings.EqualFold(course.String(), name) {
            return course
        }
    }
    return combat.RiverNone
}

// the ground of a test battle as it was picked
func testBattleZone() (combat.CombatLandscape, combat.ZoneType) {
    ground := testGrounds[testBattleGround]
    var zone combat.ZoneType
    zone.Ground.SetBiome(ground.Biome)
    zone.Ground.Coast = testBattleCoast
    zone.Ground.River = testBattleRiver
    zone.Ground.Farmland = testBattleFarmland
    zone.Ground.Weather = testBattleWeather
    if testBattleUnit != nil {
        zone.Ground.FarmRace = testBattleUnit.Race
    }
    return ground.Landscape, zone
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
    // the battle of the unit of the last test battle, on what is picked in the list now
    Test bool
    Race data.Race
    Unit *units.Unit
    // the rollouts of the ground and of the coast, and what is in them: a ground by its number
    // from 1, a coast by its side
    GroundTitle bool
    CoastTitle bool
    Ground int
    Coast combat.CoastSide
    IsCoast bool
    RiverTitle bool
    FarmlandTitle bool
    WeatherTitle bool
    Weather combat.Weather
    IsWeather bool
    // from 1
    Farmland int
    River combat.RiverCourse
    IsRiver bool
    // the one that is picked
    Picked bool
    // an empty row lies above it
    Apart bool

    // in art pixels of the picture of the window
    X float64
    Y float64
    Width float64
    Height float64
}

// a row that stands under the title of its rollout
func (row unitPickerRow) indented() bool {
    return row.Unit != nil || row.Ground > 0 || row.IsCoast || row.IsRiver || row.Farmland > 0 || row.IsWeather
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
    // the unit of before once more on what is picked here, without the way back to the debug list
    // (user, 2026-09-29). not there before the first test battle
    // the way back is the first row, the button stands apart from it (user, 2026-09-29)
    rows = append(rows, unitPickerRow{Text: "Back", Back: true})
    hasTest := testBattleLast != nil
    if hasTest {
        rows = append(rows, unitPickerRow{Text: fmt.Sprintf("Test: %v", testBattleLast.Name), Test: true, Apart: true})
    }

    // the ground and the coast of the battle
    mark := func(open bool) string {
        if open {
            return "-"
        }
        return "+"
    }
    rows = append(rows, unitPickerRow{Text: fmt.Sprintf("%v Biome: %v", mark(unitPickerGroundOpen), testGrounds[testBattleGround].Name), GroundTitle: true, Apart: true})
    if unitPickerGroundOpen {
        for index, ground := range testGrounds {
            rows = append(rows, unitPickerRow{Text: ground.Name, Ground: index + 1, Picked: index == testBattleGround})
        }
    }
    rows = append(rows, unitPickerRow{Text: fmt.Sprintf("%v Coast: %v", mark(unitPickerCoastOpen), testCoastNames[testBattleCoast]), CoastTitle: true})
    if unitPickerCoastOpen {
        for _, side := range testCoasts {
            rows = append(rows, unitPickerRow{Text: testCoastNames[side], IsCoast: true, Coast: side, Picked: side == testBattleCoast})
        }
    }
    rows = append(rows, unitPickerRow{Text: fmt.Sprintf("%v River: %v", mark(unitPickerRiverOpen), testRiverNames[testBattleRiver]), RiverTitle: true})
    if unitPickerRiverOpen {
        for _, course := range testRivers {
            rows = append(rows, unitPickerRow{Text: testRiverNames[course], IsRiver: true, River: course, Picked: course == testBattleRiver})
        }
    }
    rows = append(rows, unitPickerRow{Text: fmt.Sprintf("%v Farmland: %v", mark(unitPickerFarmlandOpen), testFarmlandNames[testBattleFarmland]), FarmlandTitle: true})
    if unitPickerFarmlandOpen {
        for index, name := range testFarmlandNames {
            rows = append(rows, unitPickerRow{Text: name, Farmland: index + 1, Picked: index == testBattleFarmland})
        }
    }
    rows = append(rows, unitPickerRow{Text: fmt.Sprintf("%v Weather: %v", mark(unitPickerWeatherOpen), testWeatherNames[testBattleWeather]), WeatherTitle: true})
    if unitPickerWeatherOpen {
        for _, weather := range combat.Weathers {
            rows = append(rows, unitPickerRow{Text: testWeatherNames[weather], IsWeather: true, Weather: weather, Picked: weather == testBattleWeather})
        }
    }

    // the races stand apart from what the battle is fought on
    apart := true
    for _, race := range unitPickerRaces {
        list := unitsOfRace(race)
        if len(list) == 0 {
            continue
        }

        mark := "+"
        if unitPickerExpanded[race] {
            mark = "-"
        }
        rows = append(rows, unitPickerRow{Text: fmt.Sprintf("%v %v (%v)", mark, race, len(list)), Race: race, Apart: apart})
        apart = false

        if unitPickerExpanded[race] {
            for _, unit := range list {
                rows = append(rows, unitPickerRow{Text: unit.Name, Unit: unit, Race: race})
            }
        }
    }

    x := unitPickerLeft
    y := unitPickerTop
    top := y
    columnWidth := 0.0

    for index := range rows {
        row := &rows[index]
        row.Height = height
        row.Width = unitPickerFont().TextWidth(row.Text, options)
        if row.Test {
            // a button: its text with room around it
            row.Height += 2 * (unitPickerButtonPadY + unitPickerButtonRim)
            row.Width += 2 * (unitPickerButtonPadX + unitPickerButtonRim)
        }

        if row.Apart && y > top {
            y += height + unitPickerRowGap
        }
        if y + height > unitPickerBottom {
            x += columnWidth + unitPickerColumnGap
            y = top
            columnWidth = 0
        }

        row.X = x
        if row.indented() {
            row.X += unitPickerIndent
        }
        row.Y = y
        if row.Test {
            // its rims are art pixels
            row.Y = math.Round(y)
        }

        columnWidth = max(columnWidth, row.X - x + row.Width)
        y = row.Y + row.Height + unitPickerRowGap
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
        case row.Test:
            testBattleUnit = testBattleLast
            unitPickerOpen = false
            return true
        case row.Back:
            unitPickerOpen = false
        case row.GroundTitle:
            unitPickerGroundOpen = !unitPickerGroundOpen
        case row.CoastTitle:
            unitPickerCoastOpen = !unitPickerCoastOpen
        case row.WeatherTitle:
            unitPickerWeatherOpen = !unitPickerWeatherOpen
        case row.IsWeather:
            testBattleWeather = row.Weather
            unitPickerWeatherOpen = false
            saveDebugSaved()
        case row.FarmlandTitle:
            unitPickerFarmlandOpen = !unitPickerFarmlandOpen
        case row.Farmland > 0:
            testBattleFarmland = row.Farmland - 1
            unitPickerFarmlandOpen = false
            saveDebugSaved()
        case row.RiverTitle:
            unitPickerRiverOpen = !unitPickerRiverOpen
        case row.IsRiver:
            testBattleRiver = row.River
            unitPickerRiverOpen = false
            saveDebugSaved()
        case row.Ground > 0:
            testBattleGround = row.Ground - 1
            unitPickerGroundOpen = false
            saveDebugSaved()
        case row.IsCoast:
            testBattleCoast = row.Coast
            unitPickerCoastOpen = false
            saveDebugSaved()
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

    for index, row := range unitPickerRows() {
        rowOptions := options

        if row.Test {
            drawUnitPickerButton(screen, row, offsetX, offsetY, index == unitPickerHover)
            continue
        }

        // titles, the row under the cursor and what is picked are white, the rest is in the color of
        // the debug list
        if index != unitPickerHover && !row.Picked && (row.indented() || row.Back) {
            var tint ebiten.DrawImageOptions
            tint.ColorScale.ScaleWithColor(unitPickerUnitColor)
            rowOptions.Options = &tint
        }

        unitPickerFont().PrintOutlined(screen, row.X + offsetX, row.Y + offsetY, rowOptions, font.OutlineFull, row.Text)
    }
}

// the button "Test": a face with a light rim above and left and a dark one below and right, all
// lighter under the mouse
func drawUnitPickerButton(screen *ebiten.Image, row unitPickerRow, offsetX float64, offsetY float64, over bool) {
    face, light := unitPickerButtonFace, unitPickerButtonLight
    if over {
        face, light = unitPickerButtonFaceOver, unitPickerButtonLightOver
    }

    fill := func(x float64, y float64, width float64, height float64, use color.Color) {
        vector.FillRect(screen, float32((x + offsetX) * scale.ScaleAmount), float32((y + offsetY) * scale.ScaleAmount), float32(width * scale.ScaleAmount), float32(height * scale.ScaleAmount), use, false)
    }

    rim := unitPickerButtonRim
    fill(row.X, row.Y, row.Width, row.Height, unitPickerButtonDark)
    fill(row.X, row.Y, row.Width - rim, row.Height - rim, light)
    fill(row.X + rim, row.Y + rim, row.Width - 2 * rim, row.Height - 2 * rim, face)

    unitPickerFont().PrintOutlined(screen, row.X + rim + unitPickerButtonPadX + offsetX, row.Y + rim + unitPickerButtonPadY + offsetY, unitPickerOptions(), font.OutlineFull, row.Text)
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
