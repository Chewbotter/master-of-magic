package game

// Modern controls of the world map: what a click on the map picks. see display/controls.go

import (
    "github.com/kazzmir/master-of-magic/game/magic/display"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

// a left click on a stack of the player selects it. false: a left click always sends the selected
// stack to the tile, as in the classic controls, and stacks are selected with the right button
const LeftClickSelects = true

// the stack of the player a left click at this point of the screen selects, nil for none.
// the stack that is selected already is not one: a click on it clears its path, as always
func (game *Game) stackToSelect(player *playerlib.Player, mouseX int, mouseY int) *playerlib.UnitStack {
    if !LeftClickSelects || !display.ModernControls() || !game.InOverworldArea(mouseX, mouseY) {
        return nil
    }

    tileX, tileY := game.ScreenToTile(float64(mouseX), float64(mouseY))
    tileX = game.Model.CurrentMap().WrapX(tileX)

    stack := player.FindStack(tileX, tileY, game.Model.Plane)
    if stack == nil || stack == player.SelectedStack {
        return nil
    }

    return stack
}

// the picture of the selected stack stays and only the colored square behind it blinks.
// false, and with the classic controls: both blink together
const OnlyBackgroundBlinks = true

func onlyBackgroundBlinks() bool {
    return OnlyBackgroundBlinks && display.ModernControls()
}

// ticks the square behind the selected stack is shown, and as many it is not. the classic controls have 15
const SelectionBlinkTicks = 20

// the stack whose square blinks, and the tick its blinking started at
var selectionBlinkStack *playerlib.UnitStack
var selectionBlinkStart uint64

// true while the square behind the selected stack is shown. the blinking starts anew, with the square
// shown for its whole time, when another stack is selected and when the stack stops moving, so the
// square is never shown or gone for a part of its time only. moving is true while the stack walks
func selectionBlinkOn(stack *playerlib.UnitStack, counter uint64, moving bool) bool {
    if stack != selectionBlinkStack || moving || counter < selectionBlinkStart {
        selectionBlinkStack = stack
        selectionBlinkStart = counter
    }

    return (counter - selectionBlinkStart) / SelectionBlinkTicks % 2 == 0
}
