package game

// Text style of the right hand panel on the world map, matched to the original game.

import (
    "fmt"

    "github.com/kazzmir/master-of-magic/lib/font"
    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
)

// the Gold, Food and Mana income under the pictures: black border all around
var PanelIncomeOutline = font.OutlineFull
// the GP and MP reserves at the top of the panel: black border right and below
var PanelReserveOutline = font.OutlineBottomRight

// the panel's text area runs from 245 (after the dark left border) up to 314 (where the light right
// border starts), in original pixels, measured from a capture. the surveyor centers its helper messages
// on the middle of it, wrapped with a margin on each side. the original code centers its own panel
// text on 280
const SurveyorPanelCenterX = 279.5
const SurveyorHelperTextWidth = 60

// the GP and MP reserves, for the screens that draw their own copy of the world map panel
// (surveyor, road building, spell targeting). the world map's own hud draws them in MakeHudUI
func (game *Game) drawPanelReserves(screen *ebiten.Image) {
    player := game.Model.GetHumanPlayer()
    if player == nil {
        return
    }
    game.Fonts.WhiteFont.PrintOutlined(screen, 276, 68, font.FontOptions{Justify: font.FontJustifyRight, Scale: scale.ScaleAmount}, PanelReserveOutline, fmt.Sprintf("%v GP", player.Gold))
    game.Fonts.WhiteFont.PrintOutlined(screen, 314, 68, font.FontOptions{Justify: font.FontJustifyRight, Scale: scale.ScaleAmount}, PanelReserveOutline, fmt.Sprintf("%v MP", player.Mana))
}
