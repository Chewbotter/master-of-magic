package game

// Heroes, mercenaries, merchants and fame as the original game has them. Rules in words:
// docs/mod/heroes.md.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/NEXTTURN.c:
// Determine_Offer, Pick_Random_Hero; CITYCALC.c: Generate_Mercenaries; HIRE.c: Hire_Hero_Popup;
// Combat.c: End_Of_Combat). The code is ours.

import (
    "math/rand/v2"

    "github.com/kazzmir/master-of-magic/game/magic/artifact"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    herolib "github.com/kazzmir/master-of-magic/game/magic/hero"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// false: upstream's offers of heroes, mercenaries and items, and its fame of battles
var ClassicHeroes = true

// Determine_Offer: the chances of a computer wizard's offers are figured on the HUMAN's fame (kept,
// the original's)
const quirkOffersOnHumanFame = true

// Generate_Mercenaries: both tests of the level are "above 90", so mercenaries are regular or
// elite, never veteran (kept, the original's)
const quirkMercenariesNoVeteran = true

// Hire_Hero_Popup, Generate_Mercenaries: the gold is checked against the full price and then the
// price halved for Charismatic (kept, the original's)
const quirkCharismaticAfterCheck = true

// Pick_Random_Hero: a hero's fame is strictly below the wizard's, so with no fame at all no hero
// comes (kept, as the code reads)
const quirkHeroFameStrict = true

// Random(100) <= chance
func classicOfferRoll(chance int) bool {
    return rand.N(100) + 1 <= chance
}

// the fame the chances of an offer are figured on
func (game *Game) classicOfferFame(player *playerlib.Player) int {
    human := game.Model.GetHumanPlayer()
    if quirkOffersOnHumanFame && human != nil && !player.IsHuman() {
        return human.GetFame()
    }
    return player.GetFame()
}

// Determine_Offer: the chance of an offer, Famous twice, at most 10, computer wizards 10 more
func (game *Game) classicOfferChance(player *playerlib.Player, base int, per int) int {
    chance := base + game.classicOfferFame(player) / per
    if player.Wizard.RetortEnabled(data.RetortFamous) {
        chance *= 2
    }
    chance = min(chance, 10)
    if !player.IsHuman() {
        chance += 10
    }
    return chance
}

// Pick_Random_Hero: up to 10 draws among all 34 heroes; the Priestess and the Paladin need life
// books, the Black Knight and the Necromancer death books; the fame the hero asks for is below the
// wizard's
func classicPickHero(player *playerlib.Player) *herolib.Hero {
    var all []*herolib.Hero
    for _, hero := range player.HeroPool {
        if hero.HeroType != herolib.HeroTorin {
            all = append(all, hero)
        }
    }
    if len(all) == 0 {
        return nil
    }
    fame := player.GetFame()
    for range 10 {
        hero := all[rand.N(len(all))]
        if hero.Status != herolib.StatusAvailable {
            continue
        }
        switch hero.HeroType {
            case herolib.HeroElana, herolib.HeroRoland:
                if !player.HasLifeBooks() {
                    continue
                }
            case herolib.HeroMortu, herolib.HeroRavashack:
                if !player.HasDeathBooks() {
                    continue
                }
        }
        if quirkHeroFameStrict && hero.GetRequiredFame() >= fame {
            continue
        }
        if !quirkHeroFameStrict && hero.GetRequiredFame() > fame {
            continue
        }
        return hero
    }
    return nil
}

// Determine_Offer: a hero comes by the chance, less for every two heroes the wizard has
func (game *Game) classicHireHero(player *playerlib.Player) {
    if len(player.AliveHeroes()) >= 6 {
        return
    }
    chance := game.classicOfferChance(player, 3, 25) / ((len(player.AliveHeroes()) + 1) / 2 + 1)
    if !classicOfferRoll(chance) {
        return
    }
    hero := classicPickHero(player)
    if hero == nil {
        return
    }

    fee := hero.GetHireFee()
    if player.IsHuman() && fee > player.Gold {
        // a hero turned away for want of gold gains a level
        hero.GainLevel(units.ExperienceChampionHero)
        return
    }
    if player.Wizard.RetortEnabled(data.RetortCharismatic) && quirkCharismaticAfterCheck {
        fee /= 2
    }
    select {
        case game.Events <- &GameEventHireHero{Cost: fee, Hero: hero, Player: player}:
        default:
    }
}

// Generate_Mercenaries: from turn 30, 20 draws of any unit but creatures, settlers, ships, heroes,
// the Nomads' Magicians and the Draconians' engineers, of a plane the wizard is on (a tower of
// another's counts for both planes)
func (game *Game) classicHireMercenaries(player *playerlib.Player) {
    if game.Model.TurnNumber < 30 {
        return
    }
    fortress := player.FindFortressCity()
    if fortress == nil {
        return
    }
    if !classicOfferRoll(game.classicOfferChance(player, 1, 20)) {
        return
    }

    fame := player.GetFame()
    level, experience := 1, 20
    if rand.N(100) + 1 + fame > 90 {
        level, experience = 3, 120
    } else if !quirkMercenariesNoVeteran && rand.N(100) + 1 + fame > 60 {
        level, experience = 2, 60
    }

    arcanus, myrror := game.classicMercenaryPlanes(player)

    var kind *units.Unit
    for range 20 {
        candidate := units.AllUnits[rand.N(len(units.AllUnits))]
        if candidate.Race == data.RaceFantastic || candidate.Race == data.RaceHero || candidate.IsSettlers() || candidate.Sailing {
            continue
        }
        if candidate.Race == data.RaceNomad && candidate.Name == "Magicians" {
            continue
        }
        if candidate.Race == data.RaceDraconian && candidate.HasAbility(data.AbilityConstruction) {
            continue
        }
        myrran := false
        for _, race := range data.MyrranRaces() {
            if candidate.Race == race {
                myrran = true
            }
        }
        if (myrran && !myrror) || (!myrran && !arcanus) {
            continue
        }
        kind = &candidate
        break
    }
    if kind == nil {
        return
    }

    count := 1
    roll := rand.N(100) + 1 + fame
    switch {
        case roll > 90: count = 3
        case roll > 60: count = 2
    }
    cost := kind.ProductionCost * (3 + level) / 2 * count
    if player.Gold < cost {
        return
    }
    if player.Wizard.RetortEnabled(data.RetortCharismatic) {
        cost /= 2
    }

    var hired []*units.OverworldUnit
    for range count {
        unit := units.MakeOverworldUnitFromUnit(*kind, fortress.X, fortress.Y, fortress.Plane, player.Wizard.Banner, player.MakeExperienceInfo(), player.MakeUnitEnchantmentProvider())
        unit.Experience = experience
        hired = append(hired, unit)
    }
    select {
        case game.Events <- &GameEventHireMercenaries{Cost: cost, Units: hired, Player: player}:
        default:
    }
}

// the planes a wizard's mercenaries may come from: where it has units or cities; any tower that is
// not its own opens both (the original's, which opens both nearly always)
func (game *Game) classicMercenaryPlanes(player *playerlib.Player) (bool, bool) {
    arcanus, myrror := false, false
    for unit := range player.Units() {
        if unit.GetPlane() == data.PlaneArcanus {
            arcanus = true
        } else {
            myrror = true
        }
    }
    for _, city := range player.Cities {
        if city.Plane == data.PlaneArcanus {
            arcanus = true
        } else {
            myrror = true
        }
    }
    if game.classicTowerOfAnother(player) {
        return true, true
    }
    return arcanus, myrror
}

// true when a tower of wizardry is not the player's. MY CALL: always (the fork keeps no owner of a
// tower; in the original a player holds all six only in a game nearly won)
func (game *Game) classicTowerOfAnother(player *playerlib.Player) bool {
    return true
}

// Determine_Offer: only the human is offered items; a random one, at three times its cost, gone
// when the gold does not reach it, half for Charismatic after that
func (game *Game) classicMerchant(player *playerlib.Player) {
    if !player.IsHuman() {
        return
    }
    if !classicOfferRoll(game.classicOfferChance(player, 2, 25)) {
        return
    }
    item, ok := artifact.MakeClassicRandomItem(game.Cache, 0, 0)
    if !ok {
        return
    }
    price := item.Cost * 3
    if player.Gold < price {
        return
    }
    if player.Wizard.RetortEnabled(data.RetortCharismatic) {
        price /= 2
    }
    select {
        case game.Events <- &GameEventMerchant{Cost: price, Artifact: &item, Player: player}:
        default:
    }
}

// End_Of_Combat: the fame of a battle. the winner +1 for 4 dead enemy units or more; the loser -1
// only when it is the human with more than 20; the winner +1 once when a dead enemy unit costs 600
// or more; the loser less (level + 1) / 2 for each of its dead heroes
func classicBattleFame(winner *playerlib.Player, loser *playerlib.Player, loserStack *playerlib.UnitStack, defeatedUnits int) (int, int) {
    winnerFame, loserFame := 0, 0
    if defeatedUnits >= 4 && !winner.IsNeutral() {
        winnerFame += 1
        if loser.IsHuman() && loser.GetFame() > 20 {
            loserFame -= 1
        }
    }
    rareFoe := false
    for _, unit := range loserStack.Units() {
        if unit.GetHealth() > 0 {
            continue
        }
        if unit.GetRawUnit().CastingCost >= 600 && !rareFoe && !winner.IsNeutral() {
            winnerFame += 1
            rareFoe = true
        }
        if hero, ok := unit.(*herolib.Hero); ok {
            loserFame -= (int(hero.GetHeroExperienceLevel()) + 1) / 2
        }
    }
    return winnerFame, loserFame
}
