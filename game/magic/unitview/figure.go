package unitview

// Drawing the figures of a unit on the battlefield one at a time, so each can take its own place in
// the order things on the field are drawn in (game/magic/combat/scenerydraw.go).

import (
    "github.com/kazzmir/master-of-magic/game/magic/data"
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

func RenderCombatFigureSemiInvisible(screen *ebiten.Image, use *ebiten.Image, options ebiten.DrawImageOptions, count int, lostCount int, lostColor *colorm.ColorM, timeCounter uint64, imageCache *util.ImageCache, figure int) {
    renderCombatSemiInvisible(screen, use, options, count, lostCount, lostColor, timeCounter, imageCache, figure)
}
