package unitview

// The text of the window that shows a unit, matched to the original game.
//
// The original prints all of it with an outline of one art pixel around the letters, in colors of
// its palette, at whole art pixels. Upstream printed it with a drop shadow of less than an art
// pixel, at places of its own. The numbers here are the original's: fonts, palette indexes and
// positions, read from the ReMoM project's reconstruction (MoM/src/UnitStat.c: Thing_View_Draw,
// USW_DrawListPage) and checked against the game's palette. The code is ours.

import (
    "fmt"
    "image/color"
    "log"
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/game/magic/util"
    "github.com/kazzmir/master-of-magic/lib/font"
    "github.com/kazzmir/master-of-magic/lib/lbx"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: the text of upstream
var OriginalText = true

// the size of the names in the list of what a unit has
var ViewListTextSize = font.TextSizeCompact

// the picture of the window, which carries the palette colors 224 and up that its text uses
const viewBackgroundLbx = "unitview.lbx"
const viewBackgroundIndex = 1

// fonts of the original, by index
const (
    viewSmallFontIndex = 1
    viewListFontIndex = 2
    viewTitleFontIndex = 4
)

// palette indexes. a list gives the color of every pixel value of a letter, the first is the one of
// the soft edge of the letters
var viewTitleColors = []int{197, 14, 255, 80, 255}
var viewSmallColors = []int{197, 255}
var viewListColors = []int{197, 255, 80, 255}
const viewOutlineColor = 251
// the width of a space in the names of the list
const viewListSpaceWidth = 1

// places, in original pixels from the upper left corner of the window
const (
    viewNameX = 51
    viewNameY = 11
    // a hero has the name on a line of its own above
    viewHeroNameY = 6
    viewHeroTitleY = 17
    viewMovesY = 27
    viewMovesIconX = 84
    viewMovesIconY = 25
    viewMovesIconStep = 8
    viewUpkeepY = 34
    viewUpkeepIconX = 85
    viewUpkeepIconY = 33

    viewStatX = 8
    viewStatIconX = 49

    // in the window of the things a city can build the lines are closer to the top
    viewBuildNameRaise = 4
    viewBuildLinesRaise = 7

    // in a battle: the number of the damage ends here
    viewBattleDamageX = 90

    // upstream's own lines, which the original does not have in this window
    viewDamageX = 165
    viewBuildCostStep = 7
    // in the window of what a city builds: the number of the cost
    viewBuildCostValueX = 73
)

// the rows of Melee, Range, Armor, Resist, Hits
var viewStatY = []int{47, 58, 68, 79, 90}

// where upstream's callers put the origin of the name and of the statistics, from the corner of the window
const (
    callerNameX = 51
    callerNameY = 6
    callerStatX = 10
    callerStatY = 50
)

type viewStyle struct {
    // the name of the unit and the names of the statistics
    Title *font.StyledFont
    // Moves, Upkeep, Damage
    Small *font.StyledFont
    // the names of abilities, items and spells in the list
    List *font.StyledFont
}

var viewStyles = make(map[*lbx.LbxCache]*viewStyle)

// the styles of the text, nil when they can not be made. then the text of upstream is printed
func getViewStyle(cache *lbx.LbxCache) *viewStyle {
    if !OriginalText || cache == nil {
        return nil
    }

    style, ok := viewStyles[cache]
    if ok {
        return style
    }

    // also when it fails, so it is tried once
    viewStyles[cache] = nil

    fontsLbx, err := cache.GetLbxFile("fonts.lbx")
    if err != nil {
        log.Printf("Unit view: unable to read fonts.lbx: %v", err)
        return nil
    }

    lbxFonts, err := font.ReadFonts(fontsLbx, 0)
    if err != nil || len(lbxFonts) <= viewTitleFontIndex {
        log.Printf("Unit view: unable to read the fonts: %v", err)
        return nil
    }

    backgroundLbx, err := cache.GetLbxFile(viewBackgroundLbx)
    if err != nil {
        log.Printf("Unit view: unable to read %v: %v", viewBackgroundLbx, err)
        return nil
    }

    palette, err := backgroundLbx.GetPalette(viewBackgroundIndex)
    if err != nil || len(palette) < 256 {
        log.Printf("Unit view: unable to read the palette of the window: %v", err)
        return nil
    }

    make := func(fontIndex int, indexes []int) *font.StyledFont {
        var colors []color.Color
        for _, index := range indexes {
            colors = append(colors, palette[index])
        }
        return font.MakeStyledFont(lbxFonts[fontIndex], colors, font.ShadowOutline, palette[viewOutlineColor])
    }

    style = &viewStyle{
        Title: make(viewTitleFontIndex, viewTitleColors),
        Small: make(viewSmallFontIndex, viewSmallColors),
        List: make(viewListFontIndex, viewListColors),
    }
    style.List.SpaceWidth = viewListSpaceWidth

    viewStyles[cache] = style
    return style
}

// prints at a place in art pixels, with the alpha of the options. styled is the style of the
// original, plain the font of upstream that is used when there is no style
func printView(styled *font.StyledFont, plain *font.Font, screen *ebiten.Image, x float64, y float64, options ebiten.DrawImageOptions, text string) {
    if styled == nil {
        plain.PrintOptions(screen, x, y, font.FontOptions{DropShadow: true, Options: &options, Scale: scale.ScaleAmount}, text)
        return
    }

    styled.Print(screen, int(math.Round(x)), int(math.Round(y)), font.FontOptions{Options: &options, Scale: scale.ScaleAmount}, text)
}

// the name of a statistic
func printStat(imageCache *util.ImageCache, plain *font.Font, screen *ebiten.Image, x float64, y float64, options ebiten.DrawImageOptions, text string) {
    style := getViewStyle(imageCache.LbxCache)
    if style == nil {
        printView(nil, plain, screen, x, y, options, text)
        return
    }

    printView(style.Title, plain, screen, x, y, options, text)
}

// a name in the list of what a unit has
func printList(imageCache *util.ImageCache, plain *font.Font, screen *ebiten.Image, x float64, y float64, options ebiten.DrawImageOptions, text string) {
    style := getViewStyle(imageCache.LbxCache)
    if style == nil {
        printView(nil, plain, screen, x, y, options, text)
        return
    }

    // smaller than the original has it, which leaves room for longer names (user, 2026-09-29).
    // the pictures and the rows of the list stay where they are
    style.List.PrintSized(screen, x, y, font.FontOptions{Options: &options, Scale: scale.ScaleAmount, TextSize: ViewListTextSize}, text)
}

// the window of a unit in a battle: the name, Moves, and Damage in the place of the upkeep, as the
// original has it there. the options are at the place upstream's callers give, callerNameX and
// callerNameY from the corner of the window. false when the styles can not be made and nothing was drawn
func RenderUnitInfoBattle(screen *ebiten.Image, imageCache *util.ImageCache, name string, moves int, flying bool, swimmer bool, damage int, options ebiten.DrawImageOptions) bool {
    style := getViewStyle(imageCache.LbxCache)
    if style == nil {
        return false
    }

    cornerX, cornerY := options.GeoM.Apply(float64(-callerNameX), float64(-callerNameY))

    at := func(x int, y int) (float64, float64) {
        return cornerX + float64(x), cornerY + float64(y)
    }

    print := func(styled *font.StyledFont, x int, y int, justify font.FontJustify, text string) {
        printX, printY := at(x, y)
        styled.Print(screen, int(math.Round(printX)), int(math.Round(printY)), font.FontOptions{Options: &options, Scale: scale.ScaleAmount, Justify: justify}, text)
    }

    print(style.Title, viewNameX, viewNameY, font.FontJustifyLeft, name)
    print(style.Small, viewNameX, viewMovesY, font.FontJustifyLeft, "Moves")

    index := 24
    if flying {
        index = 25
    } else if swimmer {
        index = 26
    }
    picture, err := imageCache.GetImage("unitview.lbx", index, 0)
    if err == nil {
        for step := 0; step < moves; step++ {
            var use ebiten.DrawImageOptions
            use.ColorScale = options.ColorScale
            use.GeoM.Translate(at(viewMovesIconX + step * viewMovesIconStep, viewMovesIconY))
            scale.DrawScaled(screen, picture, &use)
        }
    }

    print(style.Small, viewNameX, viewUpkeepY, font.FontJustifyLeft, "Damage")
    print(style.Small, viewBattleDamageX, viewUpkeepY, font.FontJustifyRight, fmt.Sprintf("%v", damage))

    return true
}

// the picture of how a unit moves
func movementPicture(imageCache *util.ImageCache, unit UnitView) *ebiten.Image {
    index := 24
    if unit.IsFlying() {
        index = 25
    } else if unit.IsSwimmer() {
        index = 26
    }

    picture, err := imageCache.GetImage("unitview.lbx", index, 0)
    if err != nil {
        return nil
    }

    return picture
}

// name, Moves, Upkeep and Damage of the window of a unit. the options are at the place upstream's
// callers give, callerNameX and callerNameY from the corner of the window
func renderUnitInfoOriginal(screen *ebiten.Image, imageCache *util.ImageCache, unit UnitView, extraTitle string, name string, style *viewStyle, options ebiten.DrawImageOptions) {
    cornerX, cornerY := options.GeoM.Apply(float64(-callerNameX), float64(-callerNameY))

    at := func(x int, y int) (float64, float64) {
        return cornerX + float64(x), cornerY + float64(y)
    }

    print := func(styled *font.StyledFont, x int, y int, text string) {
        printX, printY := at(x, y)
        printView(styled, nil, screen, printX, printY, options, text)
    }

    if extraTitle != "" {
        print(style.Title, viewNameX, viewHeroNameY, name)
        print(style.Title, viewNameX, viewHeroTitleY, "The " + extraTitle)
    } else {
        print(style.Title, viewNameX, viewNameY, name)
    }

    print(style.Small, viewNameX, viewMovesY, "Moves")

    picture := movementPicture(imageCache, unit)
    if picture != nil {
        // FIXME: draw half a movement icon if the unit has 0.5 movement?
        for index := 0; index < unit.GetMovementSpeed(true).ToInt(); index++ {
            var use ebiten.DrawImageOptions
            use.ColorScale = options.ColorScale
            use.GeoM.Translate(at(viewMovesIconX + index * viewMovesIconStep, viewMovesIconY))
            scale.DrawScaled(screen, picture, &use)
        }
    }

    print(style.Small, viewNameX, viewUpkeepY, "Upkeep")

    var upkeep ebiten.DrawImageOptions
    upkeep.ColorScale = options.ColorScale
    upkeep.GeoM.Translate(at(viewUpkeepIconX, viewUpkeepIconY))
    renderUpkeep(screen, imageCache, unit, upkeep)

    // the original shows the damage in a battle only, in the place of the upkeep. upstream shows both
    print(style.Small, viewDamageX, viewUpkeepY, fmt.Sprintf("Damage: %v", unit.GetDamage()))
}

// the same for the window of the things a city can build. the options are at the place of the name
func renderUnitInfoBuildOriginal(screen *ebiten.Image, imageCache *util.ImageCache, unit UnitView, style *viewStyle, options ebiten.DrawImageOptions, discountedCost int) {
    nameX, nameY := options.GeoM.Apply(0, 0)
    // the row of the name in the original, so the rows below it are as far from it as there
    nameRow := viewNameY - viewBuildNameRaise

    at := func(x int, y int) (float64, float64) {
        return nameX + float64(x - viewNameX), nameY + float64(y - viewBuildLinesRaise - nameRow)
    }

    print := func(x int, y int, text string) {
        printX, printY := at(x, y)
        printView(style.Small, nil, screen, printX, printY, options, text)
    }

    printView(style.Title, nil, screen, nameX, nameY, options, unit.GetName())

    print(viewNameX, viewMovesY, "Moves")

    picture := movementPicture(imageCache, unit)
    if picture != nil {
        for index := 0; index < unit.GetMovementSpeed(true).ToInt(); index++ {
            var use ebiten.DrawImageOptions
            use.ColorScale = options.ColorScale
            use.GeoM.Translate(at(viewMovesIconX + index * viewMovesIconStep, viewMovesIconY))
            scale.DrawScaled(screen, picture, &use)
        }
    }

    print(viewNameX, viewUpkeepY, "Upkeep")

    var upkeep ebiten.DrawImageOptions
    upkeep.ColorScale = options.ColorScale
    upkeep.GeoM.Translate(at(viewUpkeepIconX, viewUpkeepIconY))
    renderUpkeep(screen, imageCache, unit, upkeep)

    // what it costs in this city, and in brackets what it costs by itself
    print(viewNameX, viewUpkeepY + viewBuildCostStep, "Cost")
    print(viewBuildCostValueX, viewUpkeepY + viewBuildCostStep, fmt.Sprintf("%v(%v)", discountedCost, unit.GetProductionCost()))
}

// the styles of the text of the window of a unit: the name and the names of the statistics, the
// small lines (Moves, Upkeep, Cost), the names of the list. nil when they can not be made
func ViewFonts(cache *lbx.LbxCache) (*font.StyledFont, *font.StyledFont, *font.StyledFont) {
    style := getViewStyle(cache)
    if style == nil {
        return nil, nil, nil
    }

    return style.Title, style.Small, style.List
}
