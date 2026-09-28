package game

// Development: open a screen by name, so a frame capture can show it without clicking.

import (
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    "image"
    "log"
    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    buildinglib "github.com/kazzmir/master-of-magic/game/magic/building"
    "github.com/kazzmir/master-of-magic/lib/coroutine"
)

// the names CaptureOpenScreen accepts
var CaptureScreenNames = []string{"armies", "cities", "magic", "spellbook", "city", "surveyor", "cartographer", "advisors", "blink", "nextunit", "research", "spellinfo", "chancellor", "apprentice", "historian", "astrologer", "taxcollector", "vizier", "mirror"}

// development: sends the selected stack walking to the tile dx,dy away, as a left click would.
// returns false when there is no selected stack or no path
func (game *Game) CaptureWalk(deltaX int, deltaY int) bool {
    player := game.Model.GetHumanPlayer()
    if player == nil || player.SelectedStack == nil {
        return false
    }

    stack := player.SelectedStack
    targetX := game.Model.CurrentMap().WrapX(stack.X() + deltaX)
    targetY := stack.Y() + deltaY
    path, ok := game.Model.FindPath(stack.X(), stack.Y(), targetX, targetY, player, stack, player.GetFog(game.Model.Plane))
    if !ok {
        return false
    }

    stack.CurrentPath = path
    select {
        case game.Events <- &GameEventMoveUnit{Player: player}:
        default:
            log.Printf("capture walk: the event queue is full")
            return false
    }
    log.Printf("capture walk: stack at %v, %v with %v steps, active units %v, out of moves %v", stack.X(), stack.Y(), len(path), len(stack.ActiveUnits()), stack.AnyOutOfMoves())
    return true
}

// development: the tile the surveyor looks at in a capture, nil to follow the mouse
var CaptureSurveyorTile *image.Point

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
        case "nextunit":
            // selects the next unit with moves and moves the camera to it, as after Next Turn
            player := game.Model.GetHumanPlayer()
            if player == nil {
                return false
            }
            game.DoNextUnit(player)
            return true
        case "spellinfo":
            // the research spellbook with the info of a spell open
            player := game.Model.GetHumanPlayer()
            if player == nil {
                return false
            }
            spellbook.CaptureSpellInfo = "Star Fires"
            event = &GameEventInvokeRoutine{
                Routine: func(yield coroutine.YieldFunc) {
                    game.ResearchNewSpell(yield, player)
                },
            }
        case "research":
            // the spellbook for choosing new research, with spell descriptions
            player := game.Model.GetHumanPlayer()
            if player == nil {
                return false
            }
            event = &GameEventInvokeRoutine{
                Routine: func(yield coroutine.YieldFunc) {
                    game.ResearchNewSpell(yield, player)
                },
            }
        case "surveyorcity":
            // the surveyor pointed at the player's first city, where it shows the most text
            player := game.Model.GetHumanPlayer()
            if player == nil {
                return false
            }
            for _, city := range player.Cities {
                point := image.Pt(city.X, city.Y)
                CaptureSurveyorTile = &point
                break
            }
            event = &GameEventSurveyor{}
        case "surveyornear":
            // the surveyor pointed next to the player's first city, where it shows a helper message
            player := game.Model.GetHumanPlayer()
            if player == nil {
                return false
            }
            for _, city := range player.Cities {
                point := image.Pt(game.Model.CurrentMap().WrapX(city.X + 1), city.Y)
                CaptureSurveyorTile = &point
                break
            }
            event = &GameEventSurveyor{}
        case "chancellor":
            game.DoChancellor()
            return true
        case "apprentice": event = &GameEventApprenticeUI{}
        case "historian": event = &GameEventHistorian{}
        case "astrologer": event = &GameEventAstrologer{}
        case "taxcollector":
            game.ShowTaxCollectorUI(50, 35)
            return true
        case "vizier":
            game.ShowGrandVizierUI()
            return true
        case "mirror":
            game.ShowMirror()
            return true
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
