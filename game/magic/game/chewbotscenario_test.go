package game

// Scenario tests of Chewbot's orders on the world map (ai/chewbotmove.go and the files beside it):
// a small world drawn in text, the real game model, map and paths, Chewbot's rolls fixed at their
// lowest (every chance the original rolls for happens), and a check of the orders of one turn.
// The rules in words: docs/mod/ai-overland.md.

import (
    "fmt"
    "image"
    "strings"
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/ai"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/maplib"
    "github.com/kazzmir/master-of-magic/game/magic/pathfinding"
    "github.com/kazzmir/master-of-magic/game/magic/setup"
    "github.com/kazzmir/master-of-magic/game/magic/terrain"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    "github.com/kazzmir/master-of-magic/lib/set"
    buildinglib "github.com/kazzmir/master-of-magic/game/magic/building"
    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    herolib "github.com/kazzmir/master-of-magic/game/magic/hero"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

// the clone's AI of a scenario: gives nothing, so every order is Chewbot's
type chewStubAI struct {
    playerlib.AIBehavior
}

func (stub *chewStubAI) Update(*playerlib.Player, playerlib.AIServices) []playerlib.AIDecision {
    return nil
}

func (stub *chewStubAI) DidDiscoverEnemy(*playerlib.Player) {
}

type chewScenario struct {
    Test *testing.T
    Model *GameModel
    Width int
    Height int
}

const (
    chewTileLand = 0
    chewTileSea = 1
)

func makeChewMap(plane data.Plane, rows []string, width int, height int, model *GameModel) *maplib.Map {
    terrainData := terrain.MakeTerrainData([]image.Image{nil, nil}, []terrain.TerrainTile{
        terrain.TerrainTile{TileIndex: chewTileLand, Tile: terrain.TileLand},
        terrain.TerrainTile{TileIndex: chewTileSea, Tile: terrain.TileOcean},
    })
    mapObject := &maplib.Map{
        Map: terrain.MakeMap(height, width),
        Data: terrainData,
        Plane: plane,
        CityProvider: model,
        ExtraMap: make(map[image.Point]map[maplib.ExtraKind]maplib.ExtraTile),
    }
    for x := range width {
        for y := range height {
            mapObject.ExtraMap[image.Pt(x, y)] = make(map[maplib.ExtraKind]maplib.ExtraTile)
            mapObject.Map.Terrain[x][y] = chewTileSea
            if rows != nil && rows[y][x] != '~' {
                mapObject.Map.Terrain[x][y] = chewTileLand
            }
        }
    }
    return mapObject
}

// a world from rows of text: '~' is sea, anything else land (Arcanus); Myrror is all sea
func makeChewScenario(test *testing.T, turn uint64, rows ...string) *chewScenario {
    height := len(rows)
    width := len(rows[0])
    model := &GameModel{TurnNumber: turn, Events: make(chan GameEvent, 1000)}
    model.ArcanusMap = makeChewMap(data.PlaneArcanus, rows, width, height, model)
    model.MyrrorMap = makeChewMap(data.PlaneMyrror, nil, width, height, model)

    test.Cleanup(ai.ChewbotFixedRolls(0))
    oldCities := ai.ChewbotCities
    ai.ChewbotCities = false
    // the orders are tested here: no spells, no glue (their players have no economy)
    oldSpells, oldGlue := ai.ChewbotSpells, ai.ChewbotTurnGlue
    ai.ChewbotSpells, ai.ChewbotTurnGlue = false, false
    test.Cleanup(func() {
        ai.ChewbotCities = oldCities
        ai.ChewbotSpells, ai.ChewbotTurnGlue = oldSpells, oldGlue
    })
    return &chewScenario{Test: test, Model: model, Width: width, Height: height}
}

// a computer wizard played by Chewbot
func (scenario *chewScenario) wizard(name string, banner data.BannerType) *playerlib.Player {
    player := playerlib.MakePlayer(setup.WizardCustom{Name: name, Banner: banner}, false, scenario.Width, scenario.Height, make(map[herolib.HeroType]string), scenario.Model)
    player.AIBehavior = ai.MakeChewbotAI(&chewStubAI{}, banner == data.BannerBrown)
    scenario.Model.Players = append(scenario.Model.Players, player)
    return player
}

func (scenario *chewScenario) city(player *playerlib.Player, x int, y int, citizens int, fortress bool) *citylib.City {
    city := &citylib.City{
        Name: fmt.Sprintf("%v %v,%v", player.Wizard.Name, x, y),
        X: x,
        Y: y,
        Plane: data.PlaneArcanus,
        Race: data.RaceHighMen,
        Population: citizens * 1000,
        Buildings: set.MakeSet[buildinglib.Building](),
    }
    if fortress {
        city.Buildings.Insert(buildinglib.BuildingFortress)
    }
    player.AddCity(city)
    return city
}

// units of a player on a square of Arcanus
func (scenario *chewScenario) units(player *playerlib.Player, x int, y int, kinds ...units.Unit) []units.StackUnit {
    var out []units.StackUnit
    for _, kind := range kinds {
        out = append(out, player.AddUnit(units.MakeOverworldUnit(kind, x, y, data.PlaneArcanus)))
    }
    return out
}

// the player has seen every square
func (scenario *chewScenario) reveal(player *playerlib.Player) {
    for _, plane := range []data.Plane{data.PlaneArcanus, data.PlaneMyrror} {
        fog := player.GetFog(plane)
        for x := range fog {
            for y := range fog[x] {
                fog[x][y] = data.FogTypeVisible
            }
        }
    }
}

func (scenario *chewScenario) extra(x int, y int, kind maplib.ExtraKind, tile maplib.ExtraTile) {
    scenario.Model.ArcanusMap.ExtraMap[image.Pt(x, y)][kind] = tile
}

// the orders of one turn of a player
func (scenario *chewScenario) turn(player *playerlib.Player) []playerlib.AIDecision {
    return player.AIBehavior.Update(player, scenario.Model)
}

// where a unit is sent: the end of the path of the decision that moves it
func chewDestinationOf(decisions []playerlib.AIDecision, unit units.StackUnit) (image.Point, bool) {
    for _, decision := range decisions {
        move, ok := decision.(*playerlib.AIMoveStackDecision)
        if !ok || len(move.Path) == 0 {
            continue
        }
        moved := move.Units
        if len(moved) == 0 {
            moved = move.Stack.Units()
        }
        for _, other := range moved {
            if other == unit {
                return chewPathEnd(move.Path), true
            }
        }
    }
    return image.Point{}, false
}

func chewPathEnd(path pathfinding.Path) image.Point {
    return path[len(path) - 1]
}

func chewDescribe(decisions []playerlib.AIDecision) string {
    var parts []string
    for _, decision := range decisions {
        switch move := decision.(type) {
            case *playerlib.AIMoveStackDecision:
                if len(move.Path) == 0 {
                    continue
                }
                parts = append(parts, fmt.Sprintf("move %v units from %v,%v to %v", max(len(move.Units), len(move.Stack.Units())), move.Stack.X(), move.Stack.Y(), chewPathEnd(move.Path)))
            default:
                parts = append(parts, fmt.Sprintf("%T", decision))
        }
    }
    return strings.Join(parts, "; ")
}

func chewHas[T playerlib.AIDecision](decisions []playerlib.AIDecision, stack *playerlib.UnitStack) bool {
    for _, decision := range decisions {
        if typed, ok := decision.(T); ok {
            switch value := any(typed).(type) {
                case *playerlib.AIBuildOutpostDecision: if value.Stack == stack { return true }
                case *playerlib.AIMeldNodeDecision: if value.Stack == stack { return true }
                case *playerlib.AIPurifyDecision: if value.Stack == stack { return true }
                case *playerlib.AIPlaneShiftDecision: if value.Stack == stack { return true }
                case *playerlib.AIBuildRoadDecision: if value.Stack == stack { return true }
            }
        }
    }
    return false
}

// the original gathers an army at the stage point of its landmass first: the first turn sends it
// there, the stack is put there, and the orders of the next turn are what the army does from there
func (scenario *chewScenario) gatherAndTurn(player *playerlib.Player, army []units.StackUnit, goal image.Point) []playerlib.AIDecision {
    first := scenario.turn(player)
    stage, ok := chewDestinationOf(first, army[0])
    if ok && stage == goal {
        // no expedition is gathered (too few units for one): the army goes at once
        return first
    }
    if !ok {
        scenario.Test.Fatalf("the army should first gather at the stage point; orders: %v", chewDescribe(first))
    }
    stack := player.FindStackByUnit(army[0])
    stack.SetX(stage.X)
    stack.SetY(stage.Y)
    return scenario.turn(player)
}

// one island of land in a sea, x 2 to 17 and y 2 to 9
var chewIsland = []string{
    "~~~~~~~~~~~~~~~~~~~~",
    "~~~~~~~~~~~~~~~~~~~~",
    "~~................~~",
    "~~................~~",
    "~~................~~",
    "~~................~~",
    "~~................~~",
    "~~................~~",
    "~~................~~",
    "~~................~~",
    "~~~~~~~~~~~~~~~~~~~~",
    "~~~~~~~~~~~~~~~~~~~~",
}

// a settler where a city may be built builds one
func TestChewbotSettlerFoundsCity(test *testing.T) {
    scenario := makeChewScenario(test, 50, chewIsland...)
    wizard := scenario.wizard("Merlin", data.BannerRed)
    scenario.city(wizard, 4, 5, 4, true)
    settler := scenario.units(wizard, 12, 5, units.HighMenSettlers)[0]

    decisions := scenario.turn(wizard)
    if !chewHas[*playerlib.AIBuildOutpostDecision](decisions, wizard.FindStackByUnit(settler)) {
        test.Errorf("the settler should found a city at 12,5; orders: %v", chewDescribe(decisions))
    }
}

// a settler next to a city goes to a square where a city may be built
func TestChewbotSettlerWalksToSite(test *testing.T) {
    scenario := makeChewScenario(test, 50, chewIsland...)
    wizard := scenario.wizard("Merlin", data.BannerRed)
    scenario.city(wizard, 4, 5, 4, true)
    settler := scenario.units(wizard, 5, 5, units.HighMenSettlers)[0]

    decisions := scenario.turn(wizard)
    to, ok := chewDestinationOf(decisions, settler)
    if !ok {
        test.Fatalf("the settler should walk to a site; orders: %v", chewDescribe(decisions))
    }
    if !scenario.Model.IsSettlableLocation(to.X, to.Y, data.PlaneArcanus) {
        test.Errorf("the settler walks to %v, where no city may be built", to)
    }
}

// an army attacks a weak neutral city of its island
func TestChewbotArmyAttacksNeutralCity(test *testing.T) {
    scenario := makeChewScenario(test, 50, chewIsland...)
    wizard := scenario.wizard("Merlin", data.BannerRed)
    raiders := scenario.wizard("Raiders", data.BannerBrown)
    scenario.city(wizard, 4, 5, 4, true)
    scenario.city(raiders, 15, 7, 2, false)
    scenario.units(raiders, 15, 7, units.HighMenSpearmen)
    army := scenario.units(wizard, 8, 5, units.HighMenSwordsmen, units.HighMenSwordsmen, units.HighMenSwordsmen)

    decisions := scenario.gatherAndTurn(wizard, army, image.Pt(15, 7))
    to, ok := chewDestinationOf(decisions, army[0])
    if !ok || to != image.Pt(15, 7) {
        test.Errorf("the army should go for the neutral city at 15,7; orders: %v", chewDescribe(decisions))
    }
}

// an army does not go for a target it is not stronger than three quarters of
func TestChewbotArmyLeavesStrongCity(test *testing.T) {
    scenario := makeChewScenario(test, 50, chewIsland...)
    wizard := scenario.wizard("Merlin", data.BannerRed)
    raiders := scenario.wizard("Raiders", data.BannerBrown)
    scenario.city(wizard, 4, 5, 4, true)
    scenario.city(raiders, 15, 7, 2, false)
    scenario.units(raiders, 15, 7, units.GreatDrake, units.GreatDrake, units.GreatDrake)
    army := scenario.units(wizard, 8, 5, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen)

    decisions := scenario.gatherAndTurn(wizard, army, image.Pt(15, 7))
    if to, ok := chewDestinationOf(decisions, army[0]); ok && to == image.Pt(15, 7) {
        test.Errorf("three spearmen should not attack three great drakes; orders: %v", chewDescribe(decisions))
    }
}

// a wizard goes for the cities of another only at war (hostility 3, which a declaration of war sets
// at once; without one, hostility comes from the rules' evaluation, from turn 100 on)
func TestChewbotWizardCitiesOnlyAtWar(test *testing.T) {
    for _, war := range []bool{false, true} {
        scenario := makeChewScenario(test, 50, chewIsland...)
        wizard := scenario.wizard("Merlin", data.BannerRed)
        other := scenario.wizard("Jafar", data.BannerGreen)
        scenario.city(wizard, 4, 5, 4, true)
        scenario.city(other, 15, 7, 2, false)
        scenario.units(other, 15, 7, units.HighMenSpearmen)
        wizard.AwarePlayer(other)
        other.AwarePlayer(wizard)
        if war {
            scenario.Model.RelationRules().DeclareWar(wizard, other)
        }
        army := scenario.units(wizard, 8, 5, units.HighMenSwordsmen, units.HighMenSwordsmen, units.HighMenSwordsmen)

        decisions := scenario.gatherAndTurn(wizard, army, image.Pt(15, 7))
        to, ok := chewDestinationOf(decisions, army[0])
        attacks := ok && to == image.Pt(15, 7)
        if !war && attacks {
            test.Errorf("without war the army should not attack a wizard's city; orders: %v", chewDescribe(decisions))
        }
        if war && !attacks {
            test.Errorf("at war the army should attack the city; orders: %v", chewDescribe(decisions))
        }
    }
}

// a lair of hell hounds in the far corner: the island has a target, so spare units are not all
// gathered for an expedition (on a landmass with nothing to attack the original gathers every
// spare unit at the stage point, see TestChewbotNothingToAttackGathers)
func (scenario *chewScenario) lair(x int, y int) {
    scenario.extra(x, y, maplib.ExtraKindEncounter, &maplib.ExtraEncounter{Type: maplib.EncounterTypeCave, Units: []units.Unit{units.HellHounds, units.HellHounds}})
}

// on a landmass with nothing to attack, every spare unit goes to the stage point
func TestChewbotNothingToAttackGathers(test *testing.T) {
    scenario := makeChewScenario(test, 50, chewIsland...)
    wizard := scenario.wizard("Merlin", data.BannerRed)
    scenario.city(wizard, 4, 5, 4, true)
    one := scenario.units(wizard, 9, 8, units.HighMenSpearmen)[0]
    two := scenario.units(wizard, 14, 3, units.HighMenSpearmen)[0]

    decisions := scenario.turn(wizard)
    first, ok1 := chewDestinationOf(decisions, one)
    second, ok2 := chewDestinationOf(decisions, two)
    if !ok1 || !ok2 || first != second {
        test.Errorf("both units should go to the same stage point; orders: %v", chewDescribe(decisions))
    }
}

// a lone unit goes to a city of the wizard that wants a larger garrison
func TestChewbotLoneUnitGarrisonsCity(test *testing.T) {
    scenario := makeChewScenario(test, 50, chewIsland...)
    wizard := scenario.wizard("Merlin", data.BannerRed)
    scenario.lair(17, 9)
    scenario.city(wizard, 4, 5, 4, true)
    scenario.city(wizard, 14, 5, 6, false)
    unit := scenario.units(wizard, 9, 8, units.HighMenSpearmen)[0]

    decisions := scenario.turn(wizard)
    to, ok := chewDestinationOf(decisions, unit)
    if !ok || (to != image.Pt(4, 5) && to != image.Pt(14, 5)) {
        test.Errorf("the spearmen should go into a city; orders: %v", chewDescribe(decisions))
    }
}

// a spirit goes to a free node of its island and melds with it
func TestChewbotSpiritMelds(test *testing.T) {
    scenario := makeChewScenario(test, 50, chewIsland...)
    wizard := scenario.wizard("Merlin", data.BannerRed)
    scenario.city(wizard, 4, 5, 4, true)
    scenario.extra(12, 7, maplib.ExtraKindMagicNode, &maplib.ExtraMagicNode{Kind: maplib.MagicNodeNature, Zone: []image.Point{image.Pt(12, 7)}})
    walking := scenario.units(wizard, 8, 3, units.MagicSpirit)[0]
    onNode := scenario.units(wizard, 12, 7, units.MagicSpirit)[0]

    decisions := scenario.turn(wizard)
    if to, ok := chewDestinationOf(decisions, walking); !ok || to != image.Pt(12, 7) {
        test.Errorf("the spirit should go to the node at 12,7; orders: %v", chewDescribe(decisions))
    }
    if !chewHas[*playerlib.AIMeldNodeDecision](decisions, wizard.FindStackByUnit(onNode)) {
        test.Errorf("the spirit on the node should meld; orders: %v", chewDescribe(decisions))
    }
}

// a priest purifies corrupted land near a city
func TestChewbotPriestPurifies(test *testing.T) {
    scenario := makeChewScenario(test, 50, chewIsland...)
    wizard := scenario.wizard("Merlin", data.BannerRed)
    scenario.lair(17, 9)
    scenario.city(wizard, 4, 5, 4, true)
    scenario.Model.ArcanusMap.SetCorruption(6, 6)
    walking := scenario.units(wizard, 12, 3, units.OrcShamans)[0]

    decisions := scenario.turn(wizard)
    if to, ok := chewDestinationOf(decisions, walking); !ok || to != image.Pt(6, 6) {
        test.Errorf("the shamans should go to the corrupted land at 6,6; orders: %v", chewDescribe(decisions))
    }

    scenario = makeChewScenario(test, 50, chewIsland...)
    wizard = scenario.wizard("Merlin", data.BannerRed)
    scenario.lair(17, 9)
    scenario.city(wizard, 4, 5, 4, true)
    scenario.Model.ArcanusMap.SetCorruption(6, 6)
    onIt := scenario.units(wizard, 6, 6, units.OrcShamans)[0]
    decisions = scenario.turn(wizard)
    if !chewHas[*playerlib.AIPurifyDecision](decisions, wizard.FindStackByUnit(onIt)) {
        test.Errorf("the shamans on the corrupted land should purify it; orders: %v", chewDescribe(decisions))
    }
}

// an engineer next to a city builds a road to the nearest city it has no road to
func TestChewbotEngineerBuildsRoad(test *testing.T) {
    scenario := makeChewScenario(test, 50, chewIsland...)
    wizard := scenario.wizard("Merlin", data.BannerRed)
    scenario.city(wizard, 4, 5, 4, true)
    scenario.city(wizard, 12, 5, 4, false)
    scenario.reveal(wizard)
    engineer := scenario.units(wizard, 5, 5, units.OrcEngineers)[0]

    decisions := scenario.turn(wizard)
    found := false
    for _, decision := range decisions {
        if road, ok := decision.(*playerlib.AIBuildRoadDecision); ok && road.X == 12 && road.Y == 5 && len(road.Units) == 1 && road.Units[0] == engineer {
            found = true
        }
    }
    if !found {
        test.Errorf("the engineers should build a road to the city at 12,5; orders: %v", chewDescribe(decisions))
    }
}

// a stack on a tower of the plane of the fortress crosses to the other plane
func TestChewbotStackOnTowerCrosses(test *testing.T) {
    scenario := makeChewScenario(test, 50, chewIsland...)
    wizard := scenario.wizard("Merlin", data.BannerRed)
    scenario.city(wizard, 4, 5, 4, true)
    scenario.Model.ArcanusMap.SetPlaneTower(12, 7)
    unit := scenario.units(wizard, 12, 7, units.HighMenSpearmen)[0]

    decisions := scenario.turn(wizard)
    if !chewHas[*playerlib.AIPlaneShiftDecision](decisions, wizard.FindStackByUnit(unit)) {
        test.Errorf("the stack on the tower should cross to Myrror; orders: %v", chewDescribe(decisions))
    }
}

// two islands: the wizard's in the west, another with a neutral city in the east
var chewTwoIslands = []string{
    "~~~~~~~~~~~~~~~~~~~~~~~~",
    "~~~~~~~~~~~~~~~~~~~~~~~~",
    "~~.......~~~~~~.......~~",
    "~~.......~~~~~~.......~~",
    "~~.......~~~~~~.......~~",
    "~~.......~~~~~~.......~~",
    "~~.......~~~~~~.......~~",
    "~~~~~~~~~~~~~~~~~~~~~~~~",
    "~~~~~~~~~~~~~~~~~~~~~~~~",
}

// a ship in a city on the coast goes out to sea
func TestChewbotShipLeavesPort(test *testing.T) {
    scenario := makeChewScenario(test, 50, chewTwoIslands...)
    wizard := scenario.wizard("Merlin", data.BannerRed)
    scenario.city(wizard, 8, 4, 4, true)
    ship := scenario.units(wizard, 8, 4, units.Trireme)[0]

    decisions := scenario.turn(wizard)
    to, ok := chewDestinationOf(decisions, ship)
    if !ok || scenario.Model.ArcanusMap.GetTile(to.X, to.Y).Tile.IsLand() {
        test.Errorf("the ship should go out to sea; orders: %v", chewDescribe(decisions))
    }
}

// a ship with an army on board next to the war landmass lands the army and stays
func TestChewbotShipLandsArmy(test *testing.T) {
    scenario := makeChewScenario(test, 50, chewTwoIslands...)
    wizard := scenario.wizard("Merlin", data.BannerRed)
    raiders := scenario.wizard("Raiders", data.BannerBrown)
    scenario.city(wizard, 4, 4, 4, true)
    scenario.city(raiders, 19, 4, 2, false)
    scenario.units(raiders, 19, 4, units.HighMenSpearmen)
    ship := scenario.units(wizard, 14, 4, units.Trireme)[0]
    army := scenario.units(wizard, 14, 4, units.HighMenSwordsmen, units.HighMenSwordsmen)

    decisions := scenario.turn(wizard)
    to, ok := chewDestinationOf(decisions, army[0])
    if !ok || !scenario.Model.ArcanusMap.GetTile(to.X, to.Y).Tile.IsLand() || to.X < 15 {
        test.Errorf("the army should land on the eastern island; orders: %v", chewDescribe(decisions))
    }
    if _, moves := chewDestinationOf(decisions, ship); moves {
        test.Errorf("the ship should wait where it is; orders: %v", chewDescribe(decisions))
    }
}

// Chewbot's own rule: a city with more units than it wants sends the rest on an expedition (the
// original's rule, beyond 5, sends none of these)
func TestChewbotCitySendsSpareUnits(test *testing.T) {
    // the drafting rule alone: the expedition is the original's size (7 at turn 150)
    oldSize := ai.ChewbotExpeditionByEmpire
    ai.ChewbotExpeditionByEmpire = false
    defer func() {
        ai.ChewbotExpeditionByEmpire = oldSize
    }()
    for _, beyondNeed := range []bool{true, false} {
        old := ai.ChewbotDraftBeyondNeed
        ai.ChewbotDraftBeyondNeed = beyondNeed
        // turn 150: the expedition is 7 units, and the original's fortress gives none any more
        scenario := makeChewScenario(test, 150, chewIsland...)
        wizard := scenario.wizard("Merlin", data.BannerRed)
        scenario.lair(17, 9)
        scenario.city(wizard, 4, 5, 4, true)
        scenario.units(wizard, 4, 5, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen)
        scenario.city(wizard, 12, 5, 3, false)
        garrison := scenario.units(wizard, 12, 5, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen)

        decisions := scenario.turn(wizard)
        ai.ChewbotDraftBeyondNeed = old
        leaving := 0
        for _, unit := range garrison {
            if _, ok := chewDestinationOf(decisions, unit); ok {
                leaving += 1
            }
        }
        // a city of 3 citizens on its own island wants 2 + 3 / 4 = 2; the other 7 are an expedition.
        // the original's rule finds 4 beyond 5, too few for one
        if beyondNeed && leaving != 7 {
            test.Errorf("7 of the 9 spearmen should leave the city of 3 citizens, %v do; orders: %v", leaving, chewDescribe(decisions))
        }
        if !beyondNeed && leaving != 0 {
            test.Errorf("with the original's rule no spearmen should leave, %v do; orders: %v", leaving, chewDescribe(decisions))
        }
    }
}

// Chewbot's own size of expeditions: a small empire sends a small one (the original's size at turn
// 150 is 7, more than this wizard can spare)
func TestChewbotExpeditionBySizeOfEmpire(test *testing.T) {
    for _, byEmpire := range []bool{true, false} {
        old := ai.ChewbotExpeditionByEmpire
        ai.ChewbotExpeditionByEmpire = byEmpire
        scenario := makeChewScenario(test, 150, chewIsland...)
        wizard := scenario.wizard("Merlin", data.BannerRed)
        scenario.lair(17, 9)
        scenario.city(wizard, 4, 5, 4, true)
        scenario.units(wizard, 4, 5, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen)
        scenario.city(wizard, 12, 5, 3, false)
        garrison := scenario.units(wizard, 12, 5, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen)

        decisions := scenario.turn(wizard)
        ai.ChewbotExpeditionByEmpire = old
        leaving := 0
        for _, unit := range garrison {
            if _, ok := chewDestinationOf(decisions, unit); ok {
                leaving += 1
            }
        }
        // 14 fighting units: an expedition of 3; the city of 3 citizens wants 2 and has 3 to spare
        if byEmpire && leaving != 3 {
            test.Errorf("3 spearmen should leave for an expedition of 3, %v do; orders: %v", leaving, chewDescribe(decisions))
        }
        if !byEmpire && leaving != 0 {
            test.Errorf("with the original's size (7) no expedition should leave, %v do; orders: %v", leaving, chewDescribe(decisions))
        }
    }
}

// a stack of the game whose units are all gone (a lost battle) does not stop Chewbot's turn
func TestChewbotEmptyStack(test *testing.T) {
    scenario := makeChewScenario(test, 150, chewIsland...)
    wizard := scenario.wizard("Merlin", data.BannerRed)
    scenario.city(wizard, 4, 5, 4, true)
    scenario.units(wizard, 8, 5, units.HighMenSpearmen, units.HighMenSpearmen)
    wizard.Stacks = append(wizard.Stacks, playerlib.MakeUnitStack())
    scenario.turn(wizard)
}
