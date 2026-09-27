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

// a figure that has fallen: turned around its feet by angle, in radians, and moved by pushX and
// pushY art pixels from its place in the unit. total is the number of figures of the unit with the
// lost ones. the color scale of the options fades it
func RenderCombatFigureFallen(screen *ebiten.Image, use *ebiten.Image, options ebiten.DrawImageOptions, total int, figure int, angle float64, pushX float64, pushY float64) {
    points := CombatPoints(total)
    if figure < 0 || figure >= len(points) {
        return
    }

    var draw ebiten.DrawImageOptions
    // the feet on the origin, so the figure turns around them
    draw.GeoM.Translate(figureOffset(use))
    draw.GeoM.Rotate(angle)
    draw.GeoM.Translate(float64(points[figure].X) + pushX, float64(points[figure].Y) + pushY)
    draw.GeoM.Concat(options.GeoM)
    draw.GeoM.Scale(scale.ScaleAmount, scale.ScaleAmount)
    draw.ColorScale = options.ColorScale

    screen.DrawImage(use, &draw)
}

func RenderCombatFigureSemiInvisible(screen *ebiten.Image, use *ebiten.Image, options ebiten.DrawImageOptions, count int, lostCount int, lostColor *colorm.ColorM, timeCounter uint64, imageCache *util.ImageCache, figure int) {
    renderCombatSemiInvisible(screen, use, options, count, lostCount, lostColor, timeCounter, imageCache, figure)
}
