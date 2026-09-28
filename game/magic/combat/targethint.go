package combat

// The panel that asks for the target of a spell, matched to the original game.
//
// The original draws a panel over the right end of the combat bar, the text in it and the cancel
// button under the text. The fork drew the text without the panel, 9 further right, where it did
// not fit into the bar any more. (ReMoM: MoM/src/Combat.c Combat_Spell_Target_Screen_Draw and
// Combat_Spell_Target_Screen. The code here is ours.)
//
// The original has the player always on the right. Here the defender has the left end of the bar,
// so the panel of a defender is at the same distance from the left.
//
// NOT MATCHED: the style of the letters. The original sets font 0 with a shadow below, ours is the
// white font the fork used.

import (
    "fmt"
    "image"

    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/lib/font"

    "github.com/hajimehoshi/ebiten/v2"
)

const targetPanelLbx = "cmbtfx.lbx"
const targetPanelIndex = 28
// places on the original's screen
const targetPanelX = 238
const targetPanelY = 164
const targetTextX = 241
const targetTextY = 168
const targetTextWidth = 75
const targetCancelX = 263
const targetCancelY = 186
const targetScreenWidth = 320

// how far the panel of this side is from where the original has it
func (combat *CombatScreen) targetPanelShift(selecter Team) int {
    if selecter != TeamDefender {
        return 0
    }

    panel, err := combat.ImageCache.GetImage(targetPanelLbx, targetPanelIndex, 0)
    if err != nil {
        return 0
    }

    // as far from the left as the original's is from the right
    left := targetScreenWidth - targetPanelX - panel.Bounds().Dx()
    return left - targetPanelX
}

func (combat *CombatScreen) targetCancelPlace(selecter Team) image.Point {
    return image.Pt(targetCancelX + combat.targetPanelShift(selecter), targetCancelY)
}

func (combat *CombatScreen) drawTargetHint(screen *ebiten.Image, selecter Team, spellName string) {
    shift := combat.targetPanelShift(selecter)

    panel, err := combat.ImageCache.GetImage(targetPanelLbx, targetPanelIndex, 0)
    if err == nil {
        var options ebiten.DrawImageOptions
        options.GeoM.Translate(float64(targetPanelX + shift), targetPanelY)
        scale.DrawScaled(screen, panel, &options)
    }

    text := fmt.Sprintf("Select a target for a %v spell.", spellName)
    combat.Fonts.WhiteFont.PrintWrap(screen, float64(targetTextX + shift), targetTextY, targetTextWidth, font.FontOptions{Scale: scale.ScaleAmount}, text)
}
