package game

// Development: open a screen by name, so a frame capture can show it without clicking.

import (
    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    buildinglib "github.com/kazzmir/master-of-magic/game/magic/building"
    "github.com/kazzmir/master-of-magic/lib/coroutine"
)

// the names CaptureOpenScreen accepts
var CaptureScreenNames = []string{"armies", "cities", "magic", "spellbook", "city", "surveyor", "cartographer", "advisors", "blink"}

// puts the camera at a column, keeping its row
func (game *Game) CaptureSetCameraX(x float64) {
    game.setCameraOffset(x, game.Camera.GetOffsetY())
}

// sends the event that opens the named screen. returns false for an unknown name
func (game *Game) CaptureOpenScreen(name string) bool {
    var event GameEvent

    switch name {
        case "armies": event = &GameEventArmyView{}
        case "cities": event = &GameEventCityListView{}
        case "magic": event = &GameEventMagicView{}
        case "spellbook": event = &GameEventCastSpellBook{}
        case "surveyor": event = &GameEventSurveyor{}
        case "cartographer": event = &GameEventCartographer{}
        case "blink":
            // the red flash of an invalid move
            event = &GameEventInvokeRoutine{
                Routine: func(yield coroutine.YieldFunc) {
                    game.blinkRed(yield)
                },
            }
        case "advisors":
            game.HudUI.AddElements(game.MakeInfoUI(60, 25))
            return true
        case "city":
            player := game.Model.GetHumanPlayer()
            if player == nil || len(player.Cities) == 0 {
                return false
            }
            var city *citylib.City
            for _, candidate := range player.Cities {
                city = candidate
                break
            }
            event = &GameEventInvokeRoutine{
                Routine: func(yield coroutine.YieldFunc) {
                    game.doCityScreen(yield, city, player, buildinglib.BuildingNone)
                },
            }
        default:
            return false
    }

    select {
        case game.Events <- event:
        default:
    }
    return true
}
