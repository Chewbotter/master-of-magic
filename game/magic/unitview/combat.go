package unitview

import (
    "image"
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/aura"
    "github.com/kazzmir/master-of-magic/game/magic/util"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/colorm"
)

// where each figure of a unit stands, relative to the middle of its tile, for units of 1 to 8 figures.
// these are the original's positions (ReMoM: MoM/src/CMBTDEF.h BATTLE_UNIT_FIGURE_POSITION), which it
// gives relative to the top corner of the tile's diamond, 1 left and 8 above the middle
func CombatPoints(count int) []image.Point {
    switch count {
        case 0: return nil
        case 1: return []image.Point{image.Pt(0, 0)}
        case 2: return []image.Point{image.Pt(-8, 1), image.Pt(6, 1)}
        case 3: return []image.Point{image.Pt(-1, -4), image.Pt(-7, 2), image.Pt(6, 2)}
        case 4: return []image.Point{image.Pt(0, -4), image.Pt(-8, 0), image.Pt(7, 0), image.Pt(0, 3)}
        case 5: return []image.Point{image.Pt(0, -4), image.Pt(-8, 0), image.Pt(0, 0), image.Pt(7, 0), image.Pt(0, 3)}
        case 6: return []image.Point{image.Pt(0, -4), image.Pt(3, -1), image.Pt(-9, 0), image.Pt(8, 0), image.Pt(-4, 1), image.Pt(0, 3)}
        case 7: return []image.Point{image.Pt(0, -4), image.Pt(5, -2), image.Pt(-9, 0), image.Pt(0, 0), image.Pt(9, 0), image.Pt(-4, 3), image.Pt(0, 3)}
        case 8: return []image.Point{image.Pt(0, -4), image.Pt(5, -2), image.Pt(-3, -1), image.Pt(-9, 0), image.Pt(9, 0), image.Pt(2, 1), image.Pt(-4, 3), image.Pt(0, 3)}
    }

    // more figures than the original ever has: a plain block
    rows := math.Round(math.Sqrt(float64(count)))
    columns := math.Round(float64(count) / rows)

    var points []image.Point
    total := 0

    startRow := -int(rows) * 4 / 2
    rowGap := 4
    startColumn := -int(columns) * 4 / 2
    columnGap := 4

    for row := range int(rows) {
        for column := range int(columns) {
            if total >= count {
                break
            }

            points = append(points, image.Pt(startColumn + column * columnGap, startRow + row * rowGap))
            total += 1
        }
    }

    return points
}

// the pictures of combat figures are 28 by 30, and the original draws them with this point of the
// picture, the feet, on the figure's position
const FigureWidth = 28
const FigureHeight = 30
const FigureAnchorX = 13
const FigureAnchorY = 23

// from a figure's position to the top left of its picture
func figureOffset(use *ebiten.Image) (float64, float64) {
    if use.Bounds().Dx() == FigureWidth && use.Bounds().Dy() == FigureHeight {
        return -FigureAnchorX, -FigureAnchorY
    }

    // a picture of another size, or one cut short while a unit rises from the ground: feet 6 above the bottom
    if use.Bounds().Dx() == FigureWidth {
        return -FigureAnchorX, -FigureAnchorY
    }

    return -float64(use.Bounds().Dx() / 2), -float64(use.Bounds().Dy()) + 6
}

// draws the unit semi-transparently in a solid greyish color
func RenderCombatSemiInvisible(screen *ebiten.Image, use *ebiten.Image, options ebiten.DrawImageOptions, count int, lostCount int, lostColor *colorm.ColorM, timeCounter uint64, imageCache *util.ImageCache) {
    renderCombatSemiInvisible(screen, use, options, count, lostCount, lostColor, timeCounter, imageCache, AllFigures)
}

// only: the figure to draw, or AllFigures. see figure.go
func renderCombatSemiInvisible(screen *ebiten.Image, use *ebiten.Image, options ebiten.DrawImageOptions, count int, lostCount int, lostColor *colorm.ColorM, timeCounter uint64, imageCache *util.ImageCache, only int) {

    var greyScale colorm.ColorM
    greyScale.Scale(0, 0, 0, 0.45)
    greyScale.Scale(0, 0, 0, float64(options.ColorScale.A()))
    greyScale.Translate(32, 32, 32, 0)
    var greyOptions colorm.DrawImageOptions

    geoM := options.GeoM

    var dyingOptions colorm.DrawImageOptions

    for i, point := range CombatPoints(count + lostCount) {
        if only != AllFigures && i != only {
            continue
        }

        greyOptions.GeoM.Reset()
        greyOptions.GeoM.Translate(float64(point.X), float64(point.Y))
        greyOptions.GeoM.Translate(figureOffset(use))

        greyOptions.GeoM.Concat(geoM)
        greyOptions.GeoM.Scale(scale.ScaleAmount, scale.ScaleAmount)

        if i >= count && lostColor != nil {
            dyingOptions.GeoM = greyOptions.GeoM
            colorm.DrawImage(screen, use, *lostColor, &dyingOptions)
        } else if i < count {
            // screen.DrawImage(use, &options)
            colorm.DrawImage(screen, use, greyScale, &greyOptions)
        }
    }
}

func RenderCombatUnitGrey(screen *ebiten.Image, use *ebiten.Image, options ebiten.DrawImageOptions, count int, lostCount int, lostColor *colorm.ColorM, enchantment data.UnitEnchantment, timeCounter uint64, imageCache *util.ImageCache){
    renderCombatUnitGrey(screen, use, options, count, lostCount, lostColor, enchantment, timeCounter, imageCache, AllFigures)
}

func renderCombatUnitGrey(screen *ebiten.Image, use *ebiten.Image, options ebiten.DrawImageOptions, count int, lostCount int, lostColor *colorm.ColorM, enchantment data.UnitEnchantment, timeCounter uint64, imageCache *util.ImageCache, only int){

    var greyScale colorm.ColorM
    greyScale.Scale(1, 1, 1, float64(options.ColorScale.A()))
    greyScale.ChangeHSV(0, 0, 1)
    var greyOptions colorm.DrawImageOptions

    var dyingOptions colorm.DrawImageOptions

    geoM := options.GeoM

    for i, point := range CombatPoints(count + lostCount) {
        if only != AllFigures && i != only {
            continue
        }

        greyOptions.GeoM.Reset()
        greyOptions.GeoM.Translate(float64(point.X), float64(point.Y))
        greyOptions.GeoM.Translate(figureOffset(use))

        greyOptions.GeoM.Concat(geoM)
        greyOptions.GeoM.Scale(scale.ScaleAmount, scale.ScaleAmount)

        // screen.DrawImage(use, &options)
        if i >= count && lostColor != nil {
            dyingOptions.GeoM = greyOptions.GeoM
            colorm.DrawImage(screen, use, *lostColor, &dyingOptions)
        } else if i < count {
            colorm.DrawImage(screen, use, greyScale, &greyOptions)
            // the outline of the original, see the package aura
            aura.DrawUnit(screen, use, greyOptions.GeoM, options.ColorScale, []data.UnitEnchantment{enchantment})
        }
    }
}

func RenderCombatUnit(screen *ebiten.Image, use *ebiten.Image, options ebiten.DrawImageOptions, count int, lostCount int, lostColor *colorm.ColorM, enchantment data.UnitEnchantment, timeCounter uint64, imageCache *util.ImageCache){
    renderCombatUnit(screen, use, options, count, lostCount, lostColor, enchantment, timeCounter, imageCache, AllFigures)
}

func renderCombatUnit(screen *ebiten.Image, use *ebiten.Image, options ebiten.DrawImageOptions, count int, lostCount int, lostColor *colorm.ColorM, enchantment data.UnitEnchantment, timeCounter uint64, imageCache *util.ImageCache, only int){

    totalCount := count + lostCount

    var dyingOptions colorm.DrawImageOptions

    geoM := options.GeoM
    for i, point := range CombatPoints(totalCount) {
        if only != AllFigures && i != only {
            continue
        }

        options.GeoM.Reset()
        options.GeoM.Translate(float64(point.X), float64(point.Y))
        options.GeoM.Translate(figureOffset(use))

        options.GeoM.Concat(geoM)
        options.GeoM.Scale(scale.ScaleAmount, scale.ScaleAmount)

        /*
        x, y := options.GeoM.Apply(0, 0)
        vector.DrawFilledCircle(screen, float32(x), float32(y), 1, color.RGBA{255, 0, 0, 255}, true)
        */

        // options.GeoM.Translate(-float64(use.Bounds().Dx() / 2), -float64(use.Bounds().Dy()) + groundHeight)
        // options.GeoM.Translate(-13, -22)

        // draw the rest of the units in the dying color
        if i >= count && lostColor != nil {
            dyingOptions.GeoM = options.GeoM
            colorm.DrawImage(screen, use, *lostColor, &dyingOptions)
        } else if i < count {
            screen.DrawImage(use, &options)
            // the outline of the original, see the package aura
            aura.DrawUnit(screen, use, options.GeoM, options.ColorScale, []data.UnitEnchantment{enchantment})
        }
    }
}

func RenderCombatTile(screen *ebiten.Image, imageCache *util.ImageCache, options ebiten.DrawImageOptions){
    // FIXME: make the tile image a parameter
    grass, err := imageCache.GetImage("cmbgrass.lbx", 0, 0)
    if err == nil {
        options.GeoM.Translate(-float64(grass.Bounds().Dx() / 2), -float64(grass.Bounds().Dy() / 2))
        screen.DrawImage(grass, scale.ScaleOptions(options))
    }
}

