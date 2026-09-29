package main

// Test Battle: a battle of one kind of unit against itself, picked from a list, for looking at
// changes to the pictures of a unit. One of the debug options of the start screen, see fastplay.go.
//
// The list has a row per race that opens and closes (a rollout), with the units of the race under
// it. It is drawn in the small text size over the whole window, in as many columns as it needs.
//
// Above the races are the ground of the battle and its coast (user, 2026-09-29), rollouts as the
// races are: a landscape or one of its biomes (combat/biomes.go), and the side the sea lies on
// (combat/coast.go), and its river (combat/river.go). A click on one of them picks it, the battle
// starts with the click on a unit.
// They are kept for the next battle and the next run of the game, see debugsaved.go.

import (
    "fmt"
    "image/color"
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
    River combat.RiverCourse
    IsRiver bool
    // the one that is picked
    Picked bool

    // in art pixels of the picture of the window
    X float64
    Y float64
    Width float64
    Height float64
}

// a row that stands under the title of its rollout
func (row unitPickerRow) indented() bool {
    return row.Unit != nil || row.Ground > 0 || row.IsCoast || row.IsRiver
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

    // the ground and the coast of the battle
    mark := func(open bool) string {
        if open {
            return "-"
        }
        return "+"
    }
    rows = append(rows, unitPickerRow{Text: fmt.Sprintf("%v Biome: %v", mark(unitPickerGroundOpen), testGrounds[testBattleGround].Name), GroundTitle: true})
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
        if row.indented() {
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
        case row.GroundTitle:
            unitPickerGroundOpen = !unitPickerGroundOpen
        case row.CoastTitle:
            unitPickerCoastOpen = !unitPickerCoastOpen
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

    unitPickerFont().PrintOutlined(screen, unitPickerLeft + offsetX, unitPickerTop + offsetY, options, font.OutlineFull, "Test Battle: pick a unit. It fights its own kind, on the biome and with the coast and the river picked here.")

    for index, row := range unitPickerRows() {
        rowOptions := options

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
