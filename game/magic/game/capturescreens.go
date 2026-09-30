package game

// Development: open a screen by name, so a frame capture can show it without clicking.

import (
    "github.com/kazzmir/master-of-magic/game/magic/maplib"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    "fmt"
    "image"
    "log"
    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    "github.com/kazzmir/master-of-magic/game/magic/cityview"
    buildinglib "github.com/kazzmir/master-of-magic/game/magic/building"
    "github.com/kazzmir/master-of-magic/game/magic/unitview"
    "github.com/kazzmir/master-of-magic/lib/coroutine"
)

// the names CaptureOpenScreen accepts
var CaptureScreenNames = []string{"newbuilding", "event", "notice", "lair", "lairempty", "lairnode", "chancellorevents", "chancellormany", "unit", "cityunit", "citybuilding", "build", "buildunit", "buildbuilding", "armies", "cities", "magic", "spellbook", "city", "surveyor", "cartographer", "advisors", "blink", "nextunit", "research", "spellinfo", "chancellor", "apprentice", "historian", "astrologer", "taxcollector", "vizier", "mirror"}

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

// development: plans a move of the selected stack to the tile dx,dy away, as a first click on it does.
// returns the number of points of the path and how many of them the stack reaches in this turn
func (game *Game) CapturePlan(deltaX int, deltaY int) string {
    player := game.Model.GetHumanPlayer()
    if player == nil || player.SelectedStack == nil {
        return "no stack is selected"
    }

    stack := player.SelectedStack
    targetX := game.Model.CurrentMap().WrapX(stack.X() + deltaX)
    targetY := stack.Y() + deltaY
    walks := game.clickMoves(player, stack, targetX, targetY)
    path := game.plannedPath(stack)
    return fmt.Sprintf("walks at once %v, points of the path %v, reached in this turn %v", walks, len(path), game.pathReach(player, stack, path))
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
        case "auras", "unitaura":
            // a spell of another realm on every unit of the player, for their outlines; and
            // the window of the first unit of the selected stack. see the package aura
            player := game.Model.GetHumanPlayer()
            if player == nil {
                return false
            }
            spells := []data.UnitEnchantment{data.UnitEnchantmentBless, data.UnitEnchantmentStoneSkin, data.UnitEnchantmentFlight, data.UnitEnchantmentFlameBlade, data.UnitEnchantmentCloakOfFear}
            index := 0
            for unit := range player.Units() {
                unit.AddEnchantment(spells[index % len(spells)])
                index += 1
            }
            if name == "unitaura" && player.SelectedStack != nil && len(player.SelectedStack.Units()) > 0 {
                game.HudUI.AddGroup(unitview.MakeUnitContextMenu(game.Cache, game.HudUI, player.SelectedStack.Units()[0], func(){}))
            }
            return true
        case "unit":
            // the window of the first unit of the selected stack, as a right click on it in the panel opens
            player := game.Model.GetHumanPlayer()
            if player == nil || player.SelectedStack == nil || len(player.SelectedStack.Units()) == 0 {
                return false
            }
            game.HudUI.AddGroup(unitview.MakeUnitContextMenu(game.Cache, game.HudUI, player.SelectedStack.Units()[0], func(){}))
            return true
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
        case "newbuilding", "event":
            // the box with the animal of the realm: a building that is finished, an event
            player := game.Model.GetHumanPlayer()
            if player == nil || len(player.Cities) == 0 {
                return false
            }
            var city *citylib.City
            for _, check := range player.Cities {
                city = check
                break
            }
            event = &GameEventInvokeRoutine{
                Routine: func(yield coroutine.YieldFunc) {
                    if name == "event" {
                        game.doRandomEvent(yield, MakeDisjunctionEvent(1), true, player.Wizard)
                    } else {
                        game.showNewBuilding(yield, city, buildinglib.BuildingArmory, player)
                    }
                },
            }
        case "notice":
            // the box of a message
            event = &GameEventInvokeRoutine{
                Routine: func(yield coroutine.YieldFunc) {
                    game.doNotice(yield, game.HudUI, "You do not have enough gold to buy that. Perhaps you should raise your taxes.")
                },
            }
        case "lair", "lairempty", "lairnode":
            // the box of what a stack has found
            encounter := &maplib.ExtraEncounter{Type: maplib.EncounterTypeAncientTemple}
            if name == "lairnode" {
                encounter.Type = maplib.EncounterTypeSorceryNode
            }
            if name != "lairempty" {
                for index := range units.AllUnits {
                    if units.AllUnits[index].Name == "Sky Drake" || (name == "lairnode" && units.AllUnits[index].Name == "Phantom Warriors") {
                        encounter.Units = append(encounter.Units, units.AllUnits[index])
                        break
                    }
                }
            }
            event = &GameEventInvokeRoutine{
                Routine: func(yield coroutine.YieldFunc) {
                    game.confirmLairEncounter(yield, encounter)
                },
            }
        case "chancellorevents", "chancellormany":
            // the scroll with events of several kinds, and with more than it has room for
            game.Model.ScrollEvents = []*GameEventScroll{
                {Title: "CITY GROWTH", Text: "Hadrian's Wall has grown to a population of 5"},
                {Title: "UNITS DISBANDED", Text: "Swordsmen - deserted: lack of gold.\nBowmen - deserted: lack of food."},
                {Title: "CITY DEATHS", Text: "Rivendell now has a population of 3"},
                {Title: "CITY GROWTH", Text: "Camelot has grown to a population of 9"},
            }
            if name == "chancellormany" {
                for index := range 14 {
                    game.Model.ScrollEvents = append(game.Model.ScrollEvents, &GameEventScroll{Title: "CITY GROWTH", Text: fmt.Sprintf("Town %v has grown to a population of %v", index + 1, index + 2)})
                }
            }
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
        case "city", "cityunit", "citybuilding", "build", "buildunit", "buildbuilding":
            switch name {
                case "cityunit", "citybuilding": cityview.CaptureBuild = name
                case "build": cityview.CaptureBuild = "setting"
                case "buildunit": cityview.CaptureBuild = "unit"
                case "buildbuilding": cityview.CaptureBuild = "building"
            }
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
