package game

// What becomes of a wizard that loses its fortress or its last city, and the end of the game, as
// the original game has it. Rules in words: docs/mod/conquest.md.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/City_ovr55.c:
// Change_City_Ownership; CONQUEST.c: Resolve_Wizard_Conquest, Banish_Wizard, Conquest_Spells,
// CP_Is_Dead, Conquest_Animation; SCORE.c: End_Of_Game_Score; SPLMASTR.c: Spell_Of_Mastery).
// The code is ours. Upstream banished a wizard whenever its fortress fell, with nothing more, and
// a wizard without cities was marked defeated and played on; no win or loss by conquest.
//
// A CITY OF A WIZARD IS TAKEN (not by the neutral player's loss): when it was the fortress, or it
// was the wizard's last city (outposts count, the city taken counts), the wizard is conquered:
//   2 or more cities: BANISHED. It starts the Spell of Return at once, whatever it cast. A
//     computer wizard whose casting skill (with half the skill of its heroes on the square) is
//     under 40, or that has no mana, is DEFEATED instead. Otherwise: its fortress is gone, the
//     winner +5 fame, the loser's mana halved and the half given to the winner, every other
//     wizard likes the winner 20 less; the human is asked whether to resign (yes: defeated); a
//     computer wizard gives the winner up to 2 of its spells (not when the human was banished,
//     the original's mistake).
//   its last city: DEFEATED. The winner +5 fame, half of the loser's gold and half of its mana
//     (the loser keeps its own); the loser's city enchantments everywhere are gone; in its
//     cities its creatures and heroes are dismissed and its other units become the neutral
//     player's, and so do the cities (not the human's); its nodes are free; all its other units
//     are dismissed (heroes lose their items); its global enchantments end; treaties end and every
//     other wizard likes the winner 40 less; the winner gets up to 2 of its spells.
//   the human defeated: the lose animation and the score. The last computer wizard without
//     fortress and cities: the win animation and the score. A banished wizard with cities keeps
//     the game going.
// A defeated wizard plays no more turns. A banished wizard that casts the Spell of Return has no
// power income; a computer wizard pays the spell with its casting skill every turn, mana or not.
//
// MY CALLS, see docs/mod/improvements.md: the scene of a conquest shows when the human is the
// winner or the loser (as upstream; the reconstruction's scene returns at once unless the
// neutral player takes part, which it doubts); the wizards the human conquers are counted for
// the score (the reconstruction sets that in the scene too); a wizard is back from banishment
// when its Spell of Return is cast (the reconstruction sets nothing back, which its notes doubt).

import (
    "fmt"
    "log"
    rand "github.com/kazzmir/master-of-magic/lib/chance"

    buildinglib "github.com/kazzmir/master-of-magic/game/magic/building"
    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/halloffame"
    herolib "github.com/kazzmir/master-of-magic/game/magic/hero"
    "github.com/kazzmir/master-of-magic/game/magic/mastery"
    "github.com/kazzmir/master-of-magic/game/magic/maplib"
    musiclib "github.com/kazzmir/master-of-magic/game/magic/music"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    uilib "github.com/kazzmir/master-of-magic/game/magic/ui"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    "github.com/kazzmir/master-of-magic/lib/coroutine"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: upstream's banishment (always, nothing more) and no end of the game
var ClassicConquest = true

// development: a headless run goes on past the end of the game (-sim-play-on)
var SimPlayOn = false

const (
    // Banish_Wizard: a computer wizard with less can not come back
    conquestReturnSkill = 40
    conquestFame = 5
    conquestSpellsMost = 2
    conquestReserveMost = 30000
)

// what Banish_Wizard answers
const (
    banishStays = 0
    banishResigns = 1
    banishCanNotReturn = 99
)

// what is known of the loser before the city changes hands (Change_City_Ownership counts then)
type conquestBefore struct {
    // its cities, at most 2, the city taken with them
    Cities int
    Fortress bool
}

func conquestBeforeCapture(defender *playerlib.Player, city *citylib.City) conquestBefore {
    return conquestBefore{
        Cities: min(len(defender.Cities), 2),
        Fortress: city.Buildings.Contains(buildinglib.BuildingFortress) && !defender.Banished,
    }
}

// Change_City_Ownership: after a city of a wizard was taken or razed
func (game *Game) conquestAfterCapture(yield coroutine.YieldFunc, loser *playerlib.Player, winner *playerlib.Player, city *citylib.City, before conquestBefore) {
    if units.ClassicMovement && winner.FindCity(city.X, city.Y, city.Plane) != nil {
        // a city taken explores 3 around it
        winner.LiftFogSquare(city.X, city.Y, 3, city.Plane)
    }
    if loser.IsNeutral() || loser.Defeated {
        return
    }
    if !before.Fortress && before.Cities >= 2 {
        return
    }

    // Resolve_Wizard_Conquest
    resign := banishStays
    if before.Cities > 1 {
        resign = game.banishWizard(yield, loser, winner, city)
    }
    if before.Cities >= 2 && resign == banishStays {
        return
    }
    game.defeatWizard(yield, loser, winner, resign == banishResigns)
}

// the casting skill of a wizard with half that of its heroes on the square (Banish_Wizard)
func conquestSkill(player *playerlib.Player, city *citylib.City) int {
    heroes := 0
    for _, unit := range player.GetUnits(city.X, city.Y, city.Plane) {
        heroes += int(unit.GetCastingSkill()) / 2
    }
    return player.ComputeCastingSkill() + heroes
}

// the headless and skipping runs show nothing and ask nothing
func (game *Game) conquestQuiet() bool {
    return game.headless || game.captureSkipping
}

// the human counts the wizards it conquered for the score, each once
func noteConquered(winner *playerlib.Player, loser *playerlib.Player) {
    if winner == nil || !winner.IsHuman() {
        return
    }
    for _, banner := range winner.DefeatedWizards {
        if banner == loser.GetBanner() {
            return
        }
    }
    winner.DefeatedWizards = append(winner.DefeatedWizards, loser.GetBanner())
}

// Banish_Wizard
func (game *Game) banishWizard(yield coroutine.YieldFunc, loser *playerlib.Player, winner *playerlib.Player, city *citylib.City) int {
    // first, for everyone: whatever it cast is lost
    spellOfReturn := loser.KnownSpells.FindByName("Spell of Return")
    if !spellOfReturn.Valid() && game.Cache != nil {
        allSpells := game.AllSpells()
        spellOfReturn = allSpells.FindByName("Spell of Return")
    }
    if spellOfReturn.Valid() {
        loser.CastingSpell = spellOfReturn
        loser.CastingSpellProgress = 0
        loser.CastingSpellTarget = nil
    }

    if !loser.IsHuman() {
        if conquestSkill(loser, city) < conquestReturnSkill || loser.Mana < 1 {
            return banishCanNotReturn
        }
    }

    loser.Banished = true
    winner.Fame += conquestFame
    loser.Mana /= 2
    winner.Mana = min(winner.Mana + loser.Mana, conquestReserveMost)

    noteConquered(winner, loser)
    game.Stats.noteConquest(fmt.Sprintf("turn %v: %v banishes %v", game.Model.TurnNumber, winner.Wizard.Name, loser.Wizard.Name))
    winner.DidBanish(loser)
    game.conquestScene(yield, loser, winner)

    // every other wizard minds a banishment
    game.Model.RelationRules().Banished(winner, loser)

    if loser.IsHuman() {
        if game.askResign(yield, loser) {
            return banishResigns
        }
        return banishStays
    }

    game.conquestSpells(winner, loser)
    return banishStays
}

// the turns the Spell of Return still needs at the wizard's casting skill
func turnsToReturn(player *playerlib.Player) int {
    skill := player.ComputeCastingSkill()
    if skill <= 0 {
        return 0
    }
    remaining := max(0, player.ComputeEffectiveSpellCost(player.CastingSpell, true) - player.CastingSpellProgress)
    return remaining / skill
}

// Banish_Wizard: "Do you wish to resign?". true for yes
func (game *Game) askResign(yield coroutine.YieldFunc, human *playerlib.Player) bool {
    if game.conquestQuiet() {
        return false
    }

    message := fmt.Sprintf("There are at least %v turns remaining until you may return. Do you wish to resign?", turnsToReturn(human))

    resign := false
    quit := false
    yes := func(){
        resign = true
        quit = true
    }
    no := func(){
        quit = true
    }

    ui := &uilib.UI{
        Cache: game.Cache,
        Draw: func(ui *uilib.UI, screen *ebiten.Image){
            ui.StandardDraw(screen)
        },
    }
    ui.SetElementsFromArray(nil)
    group := uilib.MakeGroup()
    group.AddElements(uilib.MakeConfirmDialog(group, game.Cache, &game.ImageCache, message, true, yes, no))
    ui.AddGroup(group)

    oldDrawer := game.LastDrawer()
    game.PushDrawer(func(screen *ebiten.Image){
        oldDrawer(screen)
        ui.Draw(ui, screen)
    })
    defer game.PopDrawer()

    yield()
    for !quit {
        game.Counter += 1
        ui.StandardUpdate()
        yield()
    }
    yield()

    return resign
}

// Conquest_Animation: the scene of a conquest, when the human took part (MY CALL, see the top)
func (game *Game) conquestScene(yield coroutine.YieldFunc, loser *playerlib.Player, winner *playerlib.Player) {
    if game.conquestQuiet() || !(winner.IsHuman() || loser.IsHuman()) {
        return
    }
    game.doBanish(yield, winner, loser)
}

// a unit leaves the game: a hero's items are destroyed
func dismissUnit(player *playerlib.Player, unit units.StackUnit) {
    if hero, ok := unit.(*herolib.Hero); ok {
        for i := range hero.Equipment {
            hero.Equipment[i] = nil
        }
    }
    player.RemoveUnit(unit)
}

// Resolve_Wizard_Conquest: the full defeat
func (game *Game) defeatWizard(yield coroutine.YieldFunc, loser *playerlib.Player, winner *playerlib.Player, resigned bool) {
    loser.Defeated = true
    loser.Banished = true
    winner.Fame += conquestFame

    // its city enchantments, on its own cities and on others'
    for _, city := range game.Model.AllCities() {
        city.RemoveAllEnchantmentsByOwner(loser.GetBanner())
    }

    neutral := game.Model.GetNeutralPlayer()
    for _, city := range loser.GetCities() {
        for _, unit := range loser.GetUnits(city.X, city.Y, city.Plane) {
            if unit.GetRace() == data.RaceFantastic || unit.IsHero() || neutral == nil {
                dismissUnit(loser, unit)
            } else {
                loser.RemoveUnit(unit)
                neutral.AddUnit(neutral.UpdateUnit(unit))
            }
        }
        // a defeated human keeps its cities (the original's, which it doubts)
        if !loser.IsHuman() && neutral != nil {
            ChangeCityOwner(city, loser, neutral, ChangeCityKeepEnchantments)
        }
    }

    // its nodes are free
    for _, mapUse := range []*maplib.Map{game.Model.ArcanusMap, game.Model.MyrrorMap} {
        if mapUse == nil {
            continue
        }
        for _, node := range mapUse.GetMeldedNodes(loser) {
            node.MeldingWizard = nil
            node.GuardianSpiritMeld = false
            node.Warped = false
            node.WarpedOwner = nil
        }
    }

    winner.Gold = min(winner.Gold + loser.Gold / 2, conquestReserveMost)
    winner.Mana = min(winner.Mana + loser.Mana / 2, conquestReserveMost)

    var outside []units.StackUnit
    for unit := range loser.Units() {
        outside = append(outside, unit)
    }
    for _, unit := range outside {
        dismissUnit(loser, unit)
    }

    loser.GlobalEnchantments.Clear()

    noteConquered(winner, loser)
    game.Stats.noteConquest(fmt.Sprintf("turn %v: %v defeats %v", game.Model.TurnNumber, winner.Wizard.Name, loser.Wizard.Name))
    winner.DidDefeat(loser)
    if !resigned {
        game.conquestScene(yield, loser, winner)
    }

    // treaties end, every other wizard minds a conquest
    game.Model.RelationRules().Conquered(winner, loser)

    game.conquestSpells(winner, loser)

    if loser.IsHuman() {
        game.endGame(yield, nil, false)
    } else if game.computerWizardsGone() {
        game.endGame(yield, nil, true)
    }
}

// CP_Is_Dead: no computer wizard has its fortress or a city
func (game *Game) computerWizardsGone() bool {
    for _, player := range game.Model.Players {
        if player.IsHuman() || player.IsNeutral() {
            continue
        }
        if !player.Banished || len(player.Cities) > 0 {
            return false
        }
    }
    return true
}

// Conquest_Spells: up to 2 spells the loser knew and the winner did not, of realms the winner has
// books of (arcane always)
func (game *Game) conquestSpells(winner *playerlib.Player, loser *playerlib.Player) {
    if winner.IsNeutral() {
        return
    }

    var choices []spellbook.Spell
    for _, spell := range loser.KnownSpells.Spells {
        if winner.KnownSpells.Contains(spell) {
            continue
        }
        if spell.Magic != data.ArcaneMagic && winner.Wizard.MagicLevel(spell.Magic) == 0 {
            continue
        }
        choices = append(choices, spell)
    }

    for range conquestSpellsMost {
        if len(choices) == 0 {
            return
        }
        pick := rand.N(len(choices))
        spell := choices[pick]
        choices = append(choices[:pick], choices[pick + 1:]...)

        winner.LearnSpell(spell)
        log.Printf("%v finds %v in the ruins of %v", winner.Wizard.Name, spell.Name, loser.Wizard.Name)

        if winner.IsHuman() && !game.conquestQuiet() {
            select {
                case game.Events <- &GameEventNotice{Message: fmt.Sprintf("You found a %v spell in the ruins.", spell.Name)}:
                default:
            }
        }
    }
}

// the score of the human (End_Of_Game_Score). caster: who cast the Spell of Mastery, or nil
func (game *Game) endScore(caster *playerlib.Player) halloffame.Score {
    human := game.Model.GetHumanPlayer()

    townsfolk := 0
    for _, city := range human.CitiesInOrder() {
        townsfolk += city.Citizens()
    }

    var banished []data.WizardBase
    counted := make(map[data.BannerType]bool)
    for _, banner := range human.DefeatedWizards {
        counted[banner] = true
    }
    for _, player := range game.Model.Players {
        if player == human || player.IsNeutral() {
            continue
        }
        // a win by the Spell of Mastery counts every wizard whose fortress stands
        if counted[player.GetBanner()] || (caster == human && !player.Banished) {
            banished = append(banished, player.Wizard.Base)
        }
    }

    return halloffame.Score{
        Name: human.Wizard.Name,
        Race: human.Wizard.Race,
        Spells: len(human.KnownSpells.Spells),
        Townsfolk: townsfolk,
        Banished: banished,
        Fame: human.GetFame(),
        Turn: int(game.Model.TurnNumber),
        FortressStands: !human.Banished,
        Mastery: caster == human,
        OtherMastery: caster != nil && caster != human,
        Difficulty: game.Model.Settings.Difficulty,
    }
}

// the end of the game: the win or lose animation (none when the win screens were shown by the
// Spell of Mastery), the score, the Hall of Fame, then the start screen
func (game *Game) endGame(yield coroutine.YieldFunc, caster *playerlib.Player, won bool) {
    human := game.Model.GetHumanPlayer()
    score := game.endScore(caster)
    log.Printf("Game over at turn %v: the human %v, score %v", game.Model.TurnNumber, map[bool]string{true: "won", false: "lost"}[won], score.Final())

    if game.Stats != nil {
        game.Stats.lock.Lock()
        game.Stats.GameOver = fmt.Sprintf("turn %v, the human %v", game.Model.TurnNumber, map[bool]string{true: "won", false: "lost"}[won])
        game.Stats.lock.Unlock()
    }

    if game.headless && SimPlayOn && caster == nil {
        return
    }

    if !game.conquestQuiet() {
        if caster == nil {
            if won {
                game.showScreen(yield, musiclib.SongYouWin, func() (coroutine.AcceptYieldFunc, func (*ebiten.Image)) {
                    return mastery.SpellOfMasteryEndScreen(game.Cache, human.Wizard.Base)
                })
            } else {
                game.showScreen(yield, musiclib.SongYouLose, func() (coroutine.AcceptYieldFunc, func (*ebiten.Image)) {
                    return mastery.LoseScreen(game.Cache, human.Wizard.Base)
                })
            }
        }

        game.showScreen(yield, -1, func() (coroutine.AcceptYieldFunc, func (*ebiten.Image)) {
            return halloffame.ScoreScreen(game.Cache, score)
        })
        halloffame.Insert(halloffame.Entry{Name: score.Name, Score: score.Final(), Race: score.Race})
        game.showScreen(yield, -1, func() (coroutine.AcceptYieldFunc, func (*ebiten.Image)) {
            return halloffame.HallOfFameScreen(game.Cache)
        })
    }

    game.State = GameStateQuit

    // what is left to show is of a game that is over
    draining := true
    for draining {
        select {
            case <-game.Events:
            default:
                draining = false
        }
    }
}

// a screen of its own until it ends; song -1 is none
func (game *Game) showScreen(yield coroutine.YieldFunc, song musiclib.Song, screen func() (coroutine.AcceptYieldFunc, func (*ebiten.Image))) {
    logic, draw := screen()
    if song >= 0 {
        game.Music.PushSong(song)
        defer game.Music.PopSong()
    }
    game.PushDrawer(draw)
    defer game.PopDrawer()
    logic(yield)
}

// a banished wizard that casts the Spell of Return (Players_Apply_Magic_Power, CITYCALC.c)
func classicReturning(player *playerlib.Player) bool {
    return ClassicConquest && player.Banished && player.CastingSpell.Name == "Spell of Return"
}

// Spell_Of_Mastery: the caster wins. the human's win shows upstream's vortex and win screens, a
// computer wizard's the lose animation (the original's own screen of it, SPELLOSE.LBX, is not
// made); then the score
func (game *Game) doSpellOfMasteryEnd(yield coroutine.YieldFunc, caster *playerlib.Player) {
    human := game.Model.GetHumanPlayer()
    if caster == human && !game.conquestQuiet() {
        var losers []data.WizardBase
        for _, other := range game.Model.Players {
            if other != caster && !other.Defeated && !other.IsNeutral() {
                losers = append(losers, other.Wizard.Base)
            }
        }
        game.showScreen(yield, musiclib.SongSpellOfMastery, func() (coroutine.AcceptYieldFunc, func (*ebiten.Image)) {
            return mastery.LabVortexScreen(game.Cache, caster.Wizard.Base, losers)
        })
        game.showScreen(yield, musiclib.SongYouWin, func() (coroutine.AcceptYieldFunc, func (*ebiten.Image)) {
            return mastery.SpellOfMasteryEndScreen(game.Cache, caster.Wizard.Base)
        })
    } else if !game.conquestQuiet() {
        game.showScreen(yield, musiclib.SongYouLose, func() (coroutine.AcceptYieldFunc, func (*ebiten.Image)) {
            return mastery.LoseScreen(game.Cache, human.Wizard.Base)
        })
    }
    game.endGame(yield, caster, caster == human)
}
