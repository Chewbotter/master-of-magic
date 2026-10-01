package game

// Development: the human's seat played by Chewbot in a run without a window (-sim-stand-in). The
// player stays the human for every rule of the game and of the computer wizards (no difficulty
// bonuses, the AI's rules that single out the human, the human's upkeep, diplomacy toward the
// human), but its turns are the computer players' turns (Update), and wherever the game would open
// a window for the human or wait for its hand it does what it does for a computer player or nothing:
//
//   windows that only show something (notices, scrolls, the book of research, summons, events,
//   new buildings, level ups, the movements of the camera and of the selected stack): left out,
//   with what they do to the game (the end of a global enchantment's casting) done
//   choices (heroes and mercenaries for hire, the merchant, items for the vault, treasure, the
//   square of a spell, razing a town): the AI's answer
//   battles: fought out on the battlefield with Chewbot leading both armies, as the human on
//   auto (StandInTactical; false: the original's quick resolution, as Strategic Combat Only)
//
// The diplomacy the computer wizards address to the human is made by the rules and never answered:
// a human who does not talk.

import (
    "github.com/kazzmir/master-of-magic/lib/coroutine"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    herolib "github.com/kazzmir/master-of-magic/game/magic/hero"
)

// the stand-in's battles are fought out on the battlefield (false: quick resolution)
var StandInTactical = true

// a person sits at the screen for this player: the human, unless the stand-in plays its seat
func (game *Game) atScreen(player interface{ IsHuman() bool }) bool {
    return player.IsHuman() && !game.StandInHuman
}

// an event of the queue as the stand-in meets it; true when it is dealt with
func (game *Game) standInEvent(yield coroutine.YieldFunc, event GameEvent) bool {
    switch event.(type) {
        case *GameEventMagicView, *GameEventDiplomacy, *GameEventDiplomacyMessage, *GameEventDefaultItemEditor,
             *GameEventSurveyor, *GameEventCartographer, *GameEventAstrologer, *GameEventHistorian,
             *GameEventApprenticeUI, *GameEventArmyView, *GameEventShowBanish, *GameEventNotice,
             *GameEventCastSpellBook, *GameEventCityListView, *GameEventNewOutpost, *GameEventShowRandomEvent,
             *GameEventScroll, *GameEventLearnedSpell, *GameEventResearchSpell, *GameEventNewBuilding,
             *GameEventCityName, *GameEventSummonUnit, *GameEventSummonArtifact, *GameEventSummonHero,
             *GameEventGameMenu, *GameEventHeroLevelUp, *GameEventMoveCamera, *GameEventMoveUnit,
             *GameEventBuildRoad, *GameEventRunUI:
            // a window or the human's hand: nobody is there
            return true
        case *GameEventCastGlobalEnchantment:
            // the picture of the casting is left out, what follows it is not
            if after := event.(*GameEventCastGlobalEnchantment).After; after != nil {
                after()
            }
            return true
        case *GameEventHireHero:
            hire := event.(*GameEventHireHero)
            if !hire.Player.IsHuman() || hire.Player.AIBehavior == nil {
                return false
            }
            cost := 0
            if ClassicHeroes {
                cost = hire.Cost
            }
            hire.Player.AIBehavior.HandleHireHero(hire.Player, hire.Hero, cost, true, data.PlanePoint{})
            if hire.Hero.Status == herolib.StatusEmployed {
                game.ResolveStackAt(hire.Hero.GetX(), hire.Hero.GetY(), hire.Hero.GetPlane())
            }
            return true
        case *GameEventHireMercenaries:
            hire := event.(*GameEventHireMercenaries)
            if !hire.Player.IsHuman() || hire.Player.AIBehavior == nil {
                return false
            }
            hire.Player.AIBehavior.HandleHireMercenaries(hire.Player, hire.Units, hire.Cost)
            if len(hire.Units) > 0 && hire.Player.FindStackByUnit(hire.Units[0]) != nil {
                game.ResolveStackAt(hire.Units[0].GetX(), hire.Units[0].GetY(), hire.Units[0].GetPlane())
            }
            return true
        case *GameEventMerchant:
            merchant := event.(*GameEventMerchant)
            if !merchant.Player.IsHuman() || merchant.Player.AIBehavior == nil {
                return false
            }
            merchant.Player.AIBehavior.HandleMerchantItem(merchant.Player, merchant.Artifact, merchant.Cost)
            return true
        case *GameEventVault:
            vault := event.(*GameEventVault)
            if !vault.Player.IsHuman() || vault.Player.AIBehavior == nil {
                return false
            }
            // an item for the vault goes where a computer player's goes
            vault.Player.AIBehavior.HandleMerchantItem(vault.Player, vault.CreatedArtifact, 0)
            return true
        case *GameEventTreasure:
            treasure := event.(*GameEventTreasure)
            if !treasure.Player.IsHuman() {
                return false
            }
            game.ApplyTreasure(yield, treasure.Player, treasure.Treasure)
            return true
        case *GameEventSelectLocationForSpell:
            selectLocation := event.(*GameEventSelectLocationForSpell)
            if !selectLocation.Player.IsHuman() {
                return false
            }
            if selectLocation.Spell.Name == "Spell of Return" {
                game.aiSpellOfReturn(selectLocation, func(x int, y int) {
                    selectLocation.SelectedFunc(yield, x, y)
                })
            } else {
                game.aiSelectLocation(yield, selectLocation)
            }
            return true
    }
    return false
}
