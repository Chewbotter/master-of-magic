package game

// The game's side of the original's diplomacy rules (relations/): what the rules need to know of the
// world, and the diplomacy of a turn. The rules in words: docs/mod/diplomacy.md.

import (
    "fmt"
    "image"
    "log"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/relations"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    "github.com/kazzmir/master-of-magic/game/magic/combat"
    "github.com/kazzmir/master-of-magic/game/magic/diplomacy"
    musiclib "github.com/kazzmir/master-of-magic/game/magic/music"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    uilib "github.com/kazzmir/master-of-magic/game/magic/ui"
    "github.com/kazzmir/master-of-magic/lib/coroutine"

    "github.com/hajimehoshi/ebiten/v2"
)

// what a computer wizard says to the human at the end of a turn, shown when the human's turn comes
type GameEventDiplomacyMessage struct {
    Message relations.Message
}

// the original's five levels, 0 intro to 4 impossible; the fork's Extreme counts as Impossible
func originalDifficulty(difficulty data.DifficultySetting) int {
    switch difficulty {
        case data.DifficultyIntro: return 0
        case data.DifficultyEasy: return 1
        case data.DifficultyAverage: return 2
        case data.DifficultyHard: return 3
    }
    return 4
}

// the rules of diplomacy in the world as it is now
func (model *GameModel) RelationRules() *relations.Rules {
    return &relations.Rules{
        Players: model.Players,
        Turn: int(model.TurnNumber),
        Difficulty: originalDifficulty(model.GetDifficulty()),
        LandSize: model.Settings.LandSize,
        FortressLandCosts: model.fortressLandCosts,
        UnitNearCity: model.unitNearCity,
        CancelCityEnchantments: model.cancelCityEnchantments,
        ArmyStrength: armyStrength,
        Tell: func(message relations.Message) {
            model.DiplomacyMessages = append(model.DiplomacyMessages, message)
        },
    }
}

// the production cost of the units of an invader and of the owner on the landmass and plane of the
// owner's fortress (Invader_Army_Strength_Comparison)
func (model *GameModel) fortressLandCosts(invader *playerlib.Player, owner *playerlib.Player) (int, int, bool) {
    fortress := owner.FindFortressCity()
    if fortress == nil {
        return 0, 0, false
    }
    mapObject := model.GetMap(fortress.Plane)
    if mapObject == nil {
        return 0, 0, false
    }
    continent := mapObject.Map.FindContinent(fortress.X, fortress.Y)
    cost := func(player *playerlib.Player) int {
        total := 0
        for _, stack := range player.Stacks {
            if stack.Plane() != fortress.Plane || !continent.Contains(image.Pt(mapObject.WrapX(stack.X()), stack.Y())) {
                continue
            }
            for _, unit := range stack.Units() {
                total += unit.GetProductionCost()
            }
        }
        return total
    }
    return cost(invader), cost(owner), true
}

// a unit of a wizard, no settler or engineer, within 2 squares of a city of another (DIPL_GetOffMyLawn)
func (model *GameModel) unitNearCity(cityOwner *playerlib.Player, unitOwner *playerlib.Player) (string, bool) {
    for _, stack := range unitOwner.Stacks {
        fighting := false
        for _, unit := range stack.Units() {
            if !unit.HasAbility(data.AbilityCreateOutpost) && !unit.HasAbility(data.AbilityConstruction) {
                fighting = true
            }
        }
        if !fighting {
            continue
        }
        for _, city := range cityOwner.Cities {
            if city.Plane != stack.Plane() {
                continue
            }
            mapObject := model.GetMap(city.Plane)
            dx := mapObject.XDistance(city.X, stack.X())
            if dx < 0 {
                dx = -dx
            }
            dy := city.Y - stack.Y()
            if dy < 0 {
                dy = -dy
            }
            if max(dx, dy) < 3 {
                return city.Name, true
            }
        }
    }
    return "", false
}

// every city enchantment each of two wizards has on the other's cities ends (Cancel_Players_City_Enchantments)
func (model *GameModel) cancelCityEnchantments(a *playerlib.Player, b *playerlib.Player) {
    for _, pair := range [][2]*playerlib.Player{{a, b}, {b, a}} {
        caster, owner := pair[0], pair[1]
        for _, city := range owner.Cities {
            for _, enchantment := range city.Enchantments.Values() {
                if enchantment.Owner == caster.GetBanner() {
                    city.RemoveEnchantments(enchantment.Enchantment)
                }
            }
        }
    }
}

// the astrologer's army strength: gold upkeep and twice the mana upkeep of all units, in fifths
func armyStrength(player *playerlib.Player) int {
    total := 0
    for _, stack := range player.Stacks {
        for _, unit := range stack.Units() {
            total += unit.GetUpkeepGold() + 2 * unit.GetUpkeepMana()
        }
    }
    return total / 5
}

// the diplomacy of a turn, after every player had its turn
func (game *Game) doDiplomacyTurn() {
    rules := game.Model.RelationRules()
    // the start of the Spell of Mastery: every other wizard's relation to the caster drops by 50
    // (Cast_Spell_Overland_Do); after it, half that every turn (relations.EndOfTurn)
    if game.masteryNoticed == nil {
        game.masteryNoticed = make(map[*playerlib.Player]bool)
    }
    for _, player := range game.Model.Players {
        casting := player.CastingSpell.Name == "Spell of Mastery"
        if casting && !game.masteryNoticed[player] {
            rules.MasteryCast(player, 1)
        }
        game.masteryNoticed[player] = casting
    }
    game.Stats.noteContacts(game.Model.Players, game.Model.TurnNumber)
    rules.EndOfTurn(game.Model.HasEnchantment(data.EnchantmentTimeStop))
    for _, message := range game.Model.DiplomacyMessages {
        if game.Stats != nil {
            game.Stats.Messages[message.Action] += 1
        }
        log.Printf("Year=%v diplomacy: %v says %v to the human (strength %v, gold %v, spell %v)", game.Model.TurnNumber, message.From.Wizard.Name, message.Action, message.Strength, message.OfferGold, message.OfferSpell)
    }
    game.queueDiplomacyMessages()
}

// the screens of what the wizards said this turn (Resolve_Delayed_Diplomacy_Orders), one after the
// other when the human's turn comes; none in runs without a window
func (game *Game) queueDiplomacyMessages() {
    if game.captureSkipping || game.headless || game.Events == nil {
        return
    }
    for _, message := range game.Model.DiplomacyMessages {
        if DebugNoGreetings && message.Action >= relations.ActionGreeting && message.Action <= relations.ActionGreeting + int(playerlib.PersonalityPeaceful) {
            continue
        }
        message.CitySize = game.citySizeNamed(message.City)
        select {
            case game.Events <- &GameEventDiplomacyMessage{Message: message}:
            default:
        }
    }
}

// the size of the human's city of this name, as the messages name it
func (game *Game) citySizeNamed(name string) string {
    if name == "" {
        return ""
    }
    for _, player := range game.Model.Players {
        for _, city := range player.Cities {
            if city.Name == name {
                return city.GetSize().String()
            }
        }
    }
    return ""
}

// the screen of diplomacy: the talk the human opens (message nil), or what a wizard says
func (game *Game) showDiplomacy(yield coroutine.YieldFunc, human *playerlib.Player, other *playerlib.Player, message *relations.Message) {
    songs := 0
    setup := diplomacy.Setup{
        Cache: game.Cache,
        Human: human,
        Other: other,
        Rules: game.Model.RelationRules(),
        Message: message,
        Year: 1400 + int(game.Model.TurnNumber / 12),
        PlaySong: func(song musiclib.Song) {
            if songs > 0 {
                game.Music.PopSong()
                songs -= 1
            }
            game.Music.PushSong(song)
            songs += 1
        },
    }
    logic, draw := diplomacy.ShowDiplomacyScreen(setup)

    game.PushDrawer(func (screen *ebiten.Image){
        draw(screen)
    })
    defer game.PopDrawer()
    defer func() {
        for songs > 0 {
            game.Music.PopSong()
            songs -= 1
        }
    }()

    logic(yield)

    yield()
    game.RefreshUI()
}

func (game *Game) showDiplomacyMessage(yield coroutine.YieldFunc, message relations.Message) {
    human := game.Model.GetHumanPlayer()
    if human == nil || message.From == nil || message.From.Defeated {
        return
    }
    game.showDiplomacy(yield, human, message.From, &message)
}

// Combat.c: the human's stack steps onto a stack or city of a wizard it has a pact or alliance with.
// "You have a treaty with X.  Do you still wish to attack?" Yes: the treaty is broken and the battle
// goes on; no: the stack stays where it was
func (game *Game) confirmAttackOnPartner(yield coroutine.YieldFunc, player *playerlib.Player, info playerlib.CityStackInfo, stack *playerlib.UnitStack) bool {
    var defender *playerlib.Player
    if otherStack := info.FindStack(stack.X(), stack.Y(), stack.Plane()); otherStack != nil {
        defender = game.GetStackOwner(otherStack)
    } else if city := info.FindCity(stack.X(), stack.Y(), stack.Plane()); city != nil {
        defender = game.GetCityOwner(city)
    }
    if defender == nil || defender == player || defender.IsNeutral() {
        return true
    }
    relation, ok := defender.GetDiplomaticRelation(player)
    if !ok || (relation.Treaty != data.TreatyPact && relation.Treaty != data.TreatyAlliance) {
        return true
    }

    attack := false
    quit := false
    ui := &uilib.UI{
        Cache: game.Cache,
        Draw: func(ui *uilib.UI, screen *ebiten.Image){
            ui.StandardDraw(screen)
        },
    }
    ui.SetElementsFromArray(nil)
    group := uilib.MakeGroup()
    yes := func(){
        attack = true
        quit = true
    }
    no := func(){
        quit = true
    }
    group.AddElements(uilib.MakeConfirmDialogWithLayer(group, game.Cache, &game.ImageCache, 1, fmt.Sprintf("You have a treaty with %v.  Do you still wish to attack?", defender.Wizard.Name), true, yes, no))
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

    if attack {
        game.Model.RelationRules().HumanAttacksPartner(player, defender)
    }
    return attack
}

// how much a curse on a city of a wizard angers it (Change_Relations_For_Bad_City_Spell)
var cityCurseReactions = map[string]int{
    "Earthquake": -9,
    "Corruption": -25,
    "Raise Volcano": -50,
    "Chaos Rift": -5,
    "Call the Void": -20,
    "Evil Presence": -5,
    "Famine": -5,
    "Cursed Lands": -5,
    "Pestilence": -8,
}

// a curse on a city: its owner's relation to the caster drops
func (game *Game) cityCurseReaction(caster *playerlib.Player, spell string, city *citylib.City) {
    if city == nil {
        return
    }
    _, owner := game.Model.FindCity(city.X, city.Y, city.Plane)
    value, ok := cityCurseReactions[spell]
    if owner == nil || owner == caster || !ok {
        return
    }
    game.Model.RelationRules().ChangeRelations(value, caster, owner, relations.ActionCityCurse, city.Name, spell)
}

// a curse on a square near a city (Raise Volcano, Corruption): the owner of a city within 2 squares
func (game *Game) curseReactionAt(caster *playerlib.Player, spell string, x int, y int, plane data.Plane) {
    mapObject := game.Model.GetMap(plane)
    for _, player := range game.Model.Players {
        for _, city := range player.Cities {
            if city.Plane != plane || player == caster {
                continue
            }
            dx := mapObject.XDistance(city.X, x)
            if dx < 0 {
                dx = -dx
            }
            dy := city.Y - y
            if dy < 0 {
                dy = -dy
            }
            if max(dx, dy) <= 2 {
                game.cityCurseReaction(caster, spell, city)
                return
            }
        }
    }
}

// a spell that harms the units of a stack on the world map: its owner minds a little
func (game *Game) stackSpellReaction(caster *playerlib.Player, owner *playerlib.Player) {
    if owner == nil || owner == caster {
        return
    }
    game.Model.RelationRules().ChangeRelations(-3, caster, owner, relations.ActionBattle, "", "")
}

// a battle changes relations (Combat, End_Of_Combat, Update_Defender_Hostility): a city taken, -40
// and war; a city that held, -60 for a fortress, else -20; the units the defender lost; and the
// defender's hostility to the attacker
func (game *Game) combatRelations(attacker *playerlib.Player, defender *playerlib.Player, city *citylib.City, state combat.CombatState, killed []units.StackUnit) {
    if attacker == nil || defender == nil {
        return
    }
    rules := game.Model.RelationRules()
    attackerWon := state == combat.CombatStateAttackerWin || state == combat.CombatStateDefenderFlee
    if city != nil {
        if attackerWon {
            rules.ChangeRelations(-40, attacker, defender, relations.ActionCityAttacked, city.Name, "")
            if !attacker.IsNeutral() && !defender.IsNeutral() {
                rules.DeclareWar(attacker, defender)
            }
        } else {
            value := -20
            if city.HasFortress() {
                value = -60
            }
            rules.ChangeRelations(value, attacker, defender, relations.ActionCityAttacked, city.Name, "")
        }
    }
    if !defender.IsHuman() && !defender.IsNeutral() && !attacker.IsNeutral() {
        rules.ChangeRelations(-battleGrievance(killed), attacker, defender, relations.ActionBattle, "", "")
    }
    rules.WasAttacked(defender, attacker)
}

// the units of a stack that count in a battle's grievance: heroes 20, others 1 to 20, settlers and
// engineers 1 to 10 more (End_Of_Combat)
func battleGrievance(lost []units.StackUnit) int {
    total := 0
    for _, unit := range lost {
        if unit.IsHero() {
            total += 20
        } else {
            total += relations.Roll(20)
        }
        if unit.HasAbility(data.AbilityCreateOutpost) || unit.HasAbility(data.AbilityConstruction) {
            total += relations.Roll(10)
        }
    }
    return total
}
