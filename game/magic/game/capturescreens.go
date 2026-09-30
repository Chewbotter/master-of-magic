package game

// Development: open a screen by name, so a frame capture can show it without clicking.

import (
    "github.com/kazzmir/master-of-magic/game/magic/maplib"
    herolib "github.com/kazzmir/master-of-magic/game/magic/hero"
    "github.com/kazzmir/master-of-magic/game/magic/summon"
    "github.com/kazzmir/master-of-magic/game/magic/diplomacy"
    "github.com/kazzmir/master-of-magic/game/magic/relations"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/artifact"
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
    "github.com/kazzmir/master-of-magic/game/magic/halloffame"
    "github.com/kazzmir/master-of-magic/game/magic/mastery"

    "github.com/hajimehoshi/ebiten/v2"
)

// the names CaptureOpenScreen accepts
var CaptureScreenNames = []string{"vault", "vaultitem", "summon", "summonitem", "globalcast", "banish", "enemycity", "levelup", "outpost", "outpostnew", "treasure", "treasuremany", "treasurenone", "hirehero", "hireprisoner", "heroname", "mercenaries", "merchant", "cityname", "outpostname", "newbuilding", "event", "notice", "lair", "lairempty", "lairnode", "chancellorevents", "chancellormany", "unit", "cityunit", "citybuilding", "build", "buildunit", "buildbuilding", "armies", "cities", "magic", "spellbook", "city", "surveyor", "cartographer", "advisors", "blink", "nextunit", "research", "spellinfo", "chancellor", "apprentice", "historian", "astrologer", "taxcollector", "vizier", "mirror", "score", "halloffame", "lose", "resign"}

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

// development: the human player skips its turns until this many have passed. the returned
// function, called every frame, ends the skipping then
func (game *Game) CaptureSkipTurns(turns int) func() {
    human := game.Model.GetHumanPlayer()
    if human == nil {
        return func() {}
    }
    start := game.Model.TurnNumber
    human.Skip = true
    game.captureSkipping = true
    select {
        case game.Events <- &GameEventNextTurn{}:
        default:
    }
    return func() {
        if human.Skip && game.Model.TurnNumber >= start + uint64(turns) {
            human.Skip = false
            game.captureSkipping = false
            log.Printf("capture: %v turns have passed", turns)
        }
    }
}

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
        case "hirehero", "hireprisoner", "heroname", "mercenaries", "merchant", "cityname", "outpostname":
            // the windows of hiring and the box of a name
            player := game.Model.GetHumanPlayer()
            if player == nil {
                return false
            }
            player.Gold = 5000
            event = &GameEventInvokeRoutine{
                Routine: func(yield coroutine.YieldFunc) {
                    switch name {
                        case "hirehero", "hireprisoner":
                            for _, hero := range player.HeroPool {
                                cost := 0
                                if name == "hirehero" {
                                    cost = 250
                                }
                                game.doHireHero(yield, cost, hero, player, false, data.PlanePoint{})
                                break
                            }
                        case "heroname": game.doInput(yield, "Hero Name", "Brax", 70, 50)
                        case "cityname": game.doInput(yield, "New Starting City", "Gatewood", 60, 28)
                        case "outpostname": game.doInput(yield, "New Outpost", "Gatewood", 80, 100)
                        case "mercenaries":
                            var hired []*units.OverworldUnit
                            for range 3 {
                                hired = append(hired, units.MakeOverworldUnitFromUnit(units.HighMenSwordsmen, 0, 0, data.PlaneArcanus, player.Wizard.Banner, player.MakeExperienceInfo(), player.MakeUnitEnchantmentProvider()))
                            }
                            game.doHireMercenaries(yield, 300, hired, player)
                        case "merchant":
                            item := artifact.MakeRandomArtifact(game.Cache)
                            game.doMerchant(yield, 1200, &item, player)
                    }
                },
            }
        case "treasure", "treasuremany", "treasurenone":
            // the box of what was found in a lair
            player := game.Model.GetHumanPlayer()
            if player == nil {
                return false
            }
            var found Treasure
            switch name {
                case "treasure": found.Treasures = []TreasureItem{&TreasureGold{Amount: 120}}
                case "treasuremany":
                    item := artifact.MakeRandomArtifact(game.Cache)
                    found.Treasures = []TreasureItem{&TreasureGold{Amount: 120}, &TreasureMana{Amount: 80}, &TreasureMagicalItem{Artifact: &item}, &TreasureSpellbook{Magic: data.LifeMagic, Count: 1}}
            }
            event = &GameEventInvokeRoutine{
                Routine: func(yield coroutine.YieldFunc) {
                    game.doTreasurePopup(yield, player, found)
                },
            }
        case "levelup", "outpost", "outpostnew":
            // a hero that has made a level, the window of an outpost
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
                    if name == "levelup" {
                        for _, hero := range player.HeroPool {
                            if len(hero.GetAbilities()) < 3 {
                                continue
                            }
                            hero.AddExperience(200)
                            game.showHeroLevelUpPopup(yield, hero)
                            break
                        }
                        return
                    }
                    game.showOutpost(yield, city, nil, player, name == "outpostnew")
                },
            }
        case "summon", "summonitem", "globalcast", "banish", "enemycity":
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
                    switch name {
                        case "summon": game.doSummon(yield, summon.MakeSummonUnit(game.Cache, units.WarBear, player.Wizard.Base, true))
                        case "summonitem": game.doSummon(yield, summon.MakeSummonArtifact(game.Cache, player.Wizard.Base, true))
                        case "globalcast": game.doCastGlobalEnchantment(yield, player, data.EnchantmentNatureAwareness, func(){})
                        case "banish":
                            for _, other := range game.Model.Players {
                                if other != player {
                                    game.doBanish(yield, player, other)
                                    break
                                }
                            }
                        case "enemycity": game.doEnemyCityView(yield, city, player, player)
                    }
                },
            }
        case "diplomacy", "diplomacymenu", "diplomacypropose", "diplomacywar", "diplomacygreeting":
            // development: the talk with the first computer wizard, or what it says at the end of a
            // turn; the menu and the proposal come after the greeting, which the capture clicks away
            human := game.Model.GetHumanPlayer()
            var other *playerlib.Player
            for _, player := range game.Model.Players {
                if player != human && !player.IsNeutral() {
                    other = player
                    break
                }
            }
            if human == nil || other == nil {
                return false
            }
            game.Model.MakeWizardContact(human, other)
            event = &GameEventInvokeRoutine{
                Routine: func(yield coroutine.YieldFunc) {
                    switch name {
                        case "diplomacy": game.showDiplomacy(yield, human, other, nil)
                        case "diplomacymenu":
                            diplomacy.CaptureAdvance = 1
                            game.showDiplomacy(yield, human, other, nil)
                        case "diplomacypropose":
                            diplomacy.CaptureAdvance = 1
                            game.showDiplomacy(yield, human, other, &relations.Message{From: other, Action: relations.ActionProposePact, OfferGold: 100})
                        case "diplomacywar":
                            game.showDiplomacy(yield, human, other, &relations.Message{From: other, Action: relations.ActionWarDeclared})
                        case "diplomacygreeting":
                            game.showDiplomacy(yield, human, other, &relations.Message{From: other, Action: relations.ActionGreeting + int(other.Personality)})
                    }
                },
            }
        case "vault", "vaultitem":
            // the screen of the items, with heroes; vaultitem with an item in the hand
            player := game.Model.GetHumanPlayer()
            if player == nil || len(player.Cities) == 0 {
                return false
            }
            var city *citylib.City
            for _, check := range player.Cities {
                city = check
                break
            }
            player.Gold = 1234
            player.Mana = 567
            added := 0
            for _, hero := range player.HeroPool {
                if added < 3 && player.AddHero(hero, city.X, city.Y, city.Plane) {
                    hero.SetStatus(herolib.StatusEmployed)
                    added += 1
                }
            }
            if name == "vaultitem" {
                item := artifact.MakeRandomArtifact(game.Cache)
                event = &GameEventVault{CreatedArtifact: &item, Player: player}
            } else {
                event = &GameEventVault{Player: player}
            }
        case "score", "halloffame", "lose", "resign":
            // the end of a game (conquest.go)
            human := game.Model.GetHumanPlayer()
            event = &GameEventInvokeRoutine{
                Routine: func(yield coroutine.YieldFunc) {
                    switch name {
                        case "score":
                            for _, other := range game.Model.Players {
                                if other != human && !other.IsNeutral() {
                                    human.DefeatedWizards = append(human.DefeatedWizards, other.GetBanner())
                                }
                            }
                            human.Fame = 37
                            score := game.endScore(nil)
                            score.Turn = 180
                            score.Townsfolk = 57
                            game.showScreen(yield, -1, func() (coroutine.AcceptYieldFunc, func (*ebiten.Image)) {
                                return halloffame.ScoreScreen(game.Cache, score)
                            })
                        case "halloffame":
                            entries := []halloffame.Entry{
                                {Name: human.Wizard.Name, Score: 6120, Race: data.RaceHighMen},
                                {Name: "Merlin", Score: 3405, Race: data.RaceHighElf},
                                {Name: "Kali", Score: 1270, Race: data.RaceDarkElf},
                                {Name: "Lo Pan", Score: 96, Race: data.RaceKlackon},
                            }
                            game.showScreen(yield, -1, func() (coroutine.AcceptYieldFunc, func (*ebiten.Image)) {
                                return halloffame.HallOfFameScreenOf(game.Cache, entries)
                            })
                        case "lose":
                            game.showScreen(yield, -1, func() (coroutine.AcceptYieldFunc, func (*ebiten.Image)) {
                                return mastery.LoseScreen(game.Cache, human.Wizard.Base)
                            })
                        case "resign":
                            human.CastSpellOfReturn()
                            game.askResign(yield, human)
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
