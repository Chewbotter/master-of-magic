package unitview

// Drawing the figures of a unit on the battlefield one at a time, so each can take its own place in
// the order things on the field are drawn in (game/magic/combat/scenerydraw.go).

import (
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/game/magic/util"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/colorm"
)

// in place of the number of a figure: every figure of the unit
const AllFigures = -1

func RenderCombatFigure(screen *ebiten.Image, use *ebiten.Image, options ebiten.DrawImageOptions, count int, lostCount int, lostColor *colorm.ColorM, enchantment data.UnitEnchantment, timeCounter uint64, imageCache *util.ImageCache, figure int) {
    renderCombatUnit(screen, use, options, count, lostCount, lostColor, enchantment, timeCounter, imageCache, figure)
}

func RenderCombatFigureGrey(screen *ebiten.Image, use *ebiten.Image, options ebiten.DrawImageOptions, count int, lostCount int, lostColor *colorm.ColorM, enchantment data.UnitEnchantment, timeCounter uint64, imageCache *util.ImageCache, figure int) {
    renderCombatUnitGrey(screen, use, options, count, lostCount, lostColor, enchantment, timeCounter, imageCache, figure)
}

// from the feet of a figure to the top left of its picture
func FigureOffset(use *ebiten.Image) (float64, float64) {
    return figureOffset(use)
}

// the shadow of a figure: its picture in black, laid on the ground. a point of the figure lies
// lean to the right and length below the feet for every art pixel it is above them. offsetX and
// offsetY move the shadow away from the feet. total is the number of figures of the unit
func RenderCombatFigureShadow(screen *ebiten.Image, use *ebiten.Image, options ebiten.DrawImageOptions, total int, figure int, lean float64, length float64, offsetX float64, offsetY float64) {
    points := CombatPoints(total)
    if figure < 0 || figure >= len(points) {
        return
    }

    var ground ebiten.GeoM
    ground.SetElement(0, 0, 1)
    ground.SetElement(0, 1, -lean)
    ground.SetElement(1, 0, 0)
    ground.SetElement(1, 1, -length)

    var draw ebiten.DrawImageOptions
    // the feet on the origin
    draw.GeoM.Translate(figureOffset(use))
    draw.GeoM.Concat(ground)
    draw.GeoM.Translate(float64(points[figure].X) + offsetX, float64(points[figure].Y) + offsetY)
    draw.GeoM.Concat(options.GeoM)
    draw.GeoM.Scale(scale.ScaleAmount, scale.ScaleAmount)
    draw.ColorScale.Scale(0, 0, 0, 1)

    screen.DrawImage(use, &draw)
}

func RenderCombatFigureSemiInvisible(screen *ebiten.Image, use *ebiten.Image, options ebiten.DrawImageOptions, count int, lostCount int, lostColor *colorm.ColorM, timeCounter uint64, imageCache *util.ImageCache, figure int) {
    renderCombatSemiInvisible(screen, use, options, count, lostCount, lostColor, timeCounter, imageCache, figure)
}
