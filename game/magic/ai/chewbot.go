package ai

// Chewbot, the AI of the setting "Enemy AI" (display/enemyai.go): the original game's AI, ported,
// then changed from there. It takes over from the clone's AI one part at a time; what it does
// not do yet, the clone's AI does (ChewbotAI.AIBehavior).
//
// Done here: what the cities of a computer player build and buy, and what neutral cities build;
// where the armies, settlers, engineers, spirits and ships of a computer wizard go (chewbotmove.go
// and the files beside it, rules in docs/mod/ai-overland.md).
// Facts from the ReMoM project's reconstruction (MoM/src/AIBUILD.c: Player_All_Colony_Autobuild,
// Player_Colony_Autobuild_CP, Player_Colony_Autobuild_NP, AI_Player_City_Buy_Production;
// MoM/src/NEXTTURN.c: City_Apply_Production; MoM/src/INITGAME.c and NewGame.c: the objective of a
// wizard; MoX/src/MOM_DAT.c: the weight tables; MoX/src/random.c: Get_Weighted_Choice). The code
// is ours. The rules in words: docs/mod/ai-cities.md. The combat AI of Chewbot is in
// combat/aichewbot.go.

import (
    "image"
    "log"
    "math/rand/v2"
    "slices"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    buildinglib "github.com/kazzmir/master-of-magic/game/magic/building"
    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/lib/set"
)

// false: the clone's cities whatever the setting says
var ChewbotCities = true

// development: every choice of a city in the log
var ChewbotCityLog = false

// The original's own mistakes that change how it plays, kept on purpose (see combat/aichewbot.go)
const (
    // Get_Weighted_Choice halves every weight when their sum reaches 512, and again until it is
    // below: small weights (the buildings next to an expensive unit) drop to 0
    quirkWeightHalving = true
    // Player_Colony_Autobuild_CP counts the cities on Trade Goods by looking at the city that picks,
    // once for every city of the player: a city that is on Trade Goods finds all of them on it and
    // never picks it again at once, any other finds none and picks it 40 in 100 times or more
    quirkTradeGoodsCountsCity = true
)

// the objectives of a computer wizard (the original's AI_OBJ_...): they slant what its cities build
type chewObjective int
const (
    chewPragmatist chewObjective = iota
    chewMilitarist
    chewTheurgist
    chewPerfectionist
    chewExpansionist
)

func (objective chewObjective) String() string {
    switch objective {
        case chewMilitarist: return "Militarist"
        case chewTheurgist: return "Theurgist"
        case chewPerfectionist: return "Perfectionist"
        case chewExpansionist: return "Expansionist"
    }
    return "Pragmatist"
}

// ai_build_base_weights_table, by the category of a building: projects, gold, religion, research,
// military, production, food, navy
var chewBuildBase = []int{10, 15, 30, 10, 10, 15, 15, 10, 10, 10}

// ai_build_objective_weights_table [category][objective]; row 8 is combat units, row 9 builders
var chewBuildObjective = [10][5]int{
    {0, 0, 0, 5, 0},
    {0, 0, 0, 5, 0},
    {0, 0, 10, 0, 0},
    {0, 0, 10, 0, 0},
    {0, 10, 0, 0, 2},
    {0, 0, 0, 5, 0},
    {0, 0, 0, 5, 0},
    {0, 5, 0, 0, 5},
    {0, 5, 0, 0, 3},
    {0, 0, 0, 0, 10},
}

const (
    chewRowCombatUnits = 8
    chewRowBuilders = 9
)

// TBL_AI_Realm_OBJ: the chances of the objectives by the realm of most books (nature, sorcery,
// chaos, life, death)
var chewRealmObjective = map[data.MagicType][5]int{
    data.NatureMagic: {0, 2, 2, 4, 2},
    data.SorceryMagic: {0, 1, 4, 2, 3},
    data.ChaosMagic: {0, 3, 2, 2, 3},
    data.LifeMagic: {0, 2, 3, 4, 1},
    data.DeathMagic: {0, 4, 1, 1, 4},
}

// numbers of the original
const (
    // Player_All_Colony_Autobuild: a city on Housing or Trade Goods picks anew one turn in this many
    chewRepickChance = 6
    // the city's defenders are enough: a roll of 1 to this is not above their number
    chewDefendersOnFortressLand = 30
    chewDefendersElsewhere = 15
    // Player_Colony_Autobuild_NP: 5 and a roll of 1 to 15
    chewNeutralDefendersBase = 5
    chewNeutralDefendersRoll = 15
    // after this turn the human's continent is no different
    chewFortressLandTurns = 200
    // settlers and engineers are wanted more in the first turns
    chewEarlyTurns = 50
    // Get_Weighted_Choice
    chewWeightLimit = 512
)

type ChewbotAI struct {
    // the clone's AI, for everything Chewbot does not do yet
    playerlib.AIBehavior

    Neutral bool
    Objective chewObjective
    objectiveSet bool

    // what every city was building at the last turn, to see that it has finished it
    lastBuilding map[*citylib.City]buildinglib.Building
    // cities that finished a unit: they pick anew (the original's bt_AUTOBUILD)
    finished map[*citylib.City]bool

    // the overland AI's state from turn to turn (chewbotcontinents.go)
    overland *chewOverland
    services playerlib.AIServices
}

// Chewbot around the clone's AI of a computer wizard or of the neutral player
func MakeChewbotAI(clone playerlib.AIBehavior, neutral bool) *ChewbotAI {
    return &ChewbotAI{
        AIBehavior: clone,
        Neutral: neutral,
        lastBuilding: make(map[*citylib.City]buildinglib.Building),
        finished: make(map[*citylib.City]bool),
    }
}

func chewbotCitiesActive() bool {
    return ChewbotCities && display.ChewbotAI()
}

func chewCityLog(format string, args ...any) {
    if ChewbotCityLog {
        log.Printf("chewbot city: " + format, args...)
    }
}

// City_Apply_Production: a city of a computer player picks anew after it has finished a unit
func (ai *ChewbotAI) ProducedUnit(city *citylib.City, player *playerlib.Player) {
    if !chewbotCitiesActive() {
        ai.AIBehavior.ProducedUnit(city, player)
        return
    }
    ai.finished[city] = true
}

func (ai *ChewbotAI) Update(self *playerlib.Player, services playerlib.AIServices) []playerlib.AIDecision {
    ai.services = services
    decisions := ai.AIBehavior.Update(self, services)
    cities := chewbotCitiesActive()
    // the neutral player's stacks move as the clone moves them
    moves := !ai.Neutral && chewbotMovesActive()
    if !cities && !moves {
        return decisions
    }

    // the clone's production and moves are Chewbot's now
    var out []playerlib.AIDecision
    for _, decision := range decisions {
        if _, isProduce := decision.(*playerlib.AIProduceDecision); isProduce && cities {
            continue
        }
        if moves && chewIsMoveDecision(decision) {
            continue
        }
        out = append(out, decision)
    }

    if cities {
        out = append(out, ai.cityDecisions(self, services)...)
    }
    if moves {
        out = append(out, ai.moveDecisions(self, services)...)
    }
    return out
}

// the objective of a wizard, picked once as the original does at the start of a game: by the realm
// of its most books and its retorts
func (ai *ChewbotAI) objective(self *playerlib.Player) chewObjective {
    if ai.objectiveSet {
        return ai.Objective
    }
    ai.objectiveSet = true

    weights := chewRealmObjective[self.Wizard.MostBooks()]
    wizard := &self.Wizard
    if wizard.RetortEnabled(data.RetortWarlord) {
        weights[chewMilitarist] += 2
        weights[chewExpansionist] += 3
    }
    if wizard.RetortEnabled(data.RetortChaosMastery) || wizard.RetortEnabled(data.RetortNatureMastery) {
        weights[chewTheurgist] += 1
    }
    if wizard.RetortEnabled(data.RetortAlchemy) {
        weights[chewPerfectionist] += 2
    }
    if wizard.RetortEnabled(data.RetortArchmage) {
        weights[chewTheurgist] += 4
    }
    if wizard.RetortEnabled(data.RetortMyrran) {
        weights[chewExpansionist] += 4
    }
    if wizard.RetortEnabled(data.RetortConjurer) {
        weights[chewMilitarist] += 3
    }

    ai.Objective = chewObjective(chewWeightedChoice(weights[:]))
    chewCityLog("%v is a %v", self.Wizard.Name, ai.Objective)
    return ai.Objective
}

// Get_Weighted_Choice: an index by its weight; the first when all are 0
func chewWeightedChoice(weights []int) int {
    weights = slices.Clone(weights)
    total := 0
    for index := 0; index < len(weights); index++ {
        total += weights[index]
        if quirkWeightHalving && total >= chewWeightLimit {
            for other := range weights {
                weights[other] >>= 1
            }
            total = 0
            index = -1
        }
    }
    if total <= 0 {
        return 0
    }

    roll := 1 + rand.IntN(total)
    roll -= weights[0]
    choice := 0
    for roll > 0 && choice < len(weights) - 1 {
        choice += 1
        roll -= weights[choice]
    }
    return choice
}

// one thing a city can produce
type chewProduct struct {
    Building buildinglib.Building
    Unit units.Unit
}

func (product chewProduct) isUnit() bool {
    return !product.Unit.IsNone()
}

// Calculate_Product_Array: the buildings and units the city can produce
func chewProducts(city *citylib.City) []chewProduct {
    var out []chewProduct
    buildings := city.ComputePossibleBuildings(false).Values()
    slices.Sort(buildings)
    for _, building := range buildings {
        out = append(out, chewProduct{Building: building, Unit: units.UnitNone})
    }
    for _, unit := range city.ComputePossibleUnits() {
        out = append(out, chewProduct{Building: buildinglib.BuildingNone, Unit: unit})
    }
    return out
}

func chewIsBuilder(unit *units.Unit) bool {
    return unit.HasAbility(data.AbilityCreateOutpost)
}

func chewIsEngineer(unit *units.Unit) bool {
    return unit.HasAbility(data.AbilityConstruction)
}

func chewIsTransport(unit *units.Unit) bool {
    return unit.HasAbility(data.AbilityTransport)
}

// the weight of a unit: its cost in tens, squared
func chewUnitWeight(unit *units.Unit) int {
    tens := unit.ProductionCost / 10
    return tens * tens
}

func (ai *ChewbotAI) cityDecisions(self *playerlib.Player, services playerlib.AIServices) []playerlib.AIDecision {
    var decisions []playerlib.AIDecision

    for _, city := range self.Cities {
        if city.Outpost {
            continue
        }

        // it has finished what it built
        last, known := ai.lastBuilding[city]
        finished := ai.finished[city] || (known && last != buildinglib.BuildingNone && city.Buildings.Contains(last))
        delete(ai.finished, city)

        nothing := city.ProducingBuilding == buildinglib.BuildingNone && city.ProducingUnit.IsNone()
        onFiller := city.ProducingUnit.IsNone() && (city.ProducingBuilding == buildinglib.BuildingHousing || city.ProducingBuilding == buildinglib.BuildingTradeGoods)

        pick := finished || nothing || (onFiller && rand.IntN(chewRepickChance) == 0)

        if pick {
            var product chewProduct
            if ai.Neutral {
                product = ai.pickNeutral(self, city)
            } else {
                product = ai.pickWizard(self, services, city)
            }
            decisions = append(decisions, &playerlib.AIProduceDecision{City: city, Building: product.Building, Unit: product.Unit})
            ai.lastBuilding[city] = product.Building
            chewCityLog("%v in %v builds %v%v", self.Wizard.Name, city.Name, services.GetBuildingInfos().Name(product.Building), product.Unit.Name)
            continue
        }

        ai.lastBuilding[city] = city.ProducingBuilding

        // AI_Player_City_Buy_Production: not for the neutral player nor Housing or Trade Goods
        if !ai.Neutral && !onFiller && ai.wantsToBuy(self, city) {
            decisions = append(decisions, &playerlib.AIBuyProductionDecision{City: city})
            chewCityLog("%v buys the production of %v", self.Wizard.Name, city.Name)
        }
    }

    return decisions
}

// AI_Player_City_Buy_Production: gold above a multiple of the price: 1 to 4 times for a building,
// 0 to 3 for settlers, 3 to 7 for other units. the price is the fork's, the one of the city screen
func (ai *ChewbotAI) wantsToBuy(self *playerlib.Player, city *citylib.City) bool {
    price, remaining := city.BuyCost()
    if remaining <= 0 || price <= 0 {
        return false
    }

    times := 1 + rand.IntN(4)
    if !city.ProducingUnit.IsNone() {
        if city.ProducingUnit.IsSettlers() {
            times = rand.IntN(4)
        } else {
            times = 3 + rand.IntN(5)
        }
    }

    return self.Gold > times * price
}

// the units of the player on the land of a place, by what they can do
type chewLandUnits struct {
    Settlers bool
    Engineers bool
    Transports int
}

// the tiles of the continent of a place
func chewContinent(services playerlib.AIServices, x int, y int, plane data.Plane) *set.Set[image.Point] {
    mapObject := services.GetMap(plane)
    if mapObject == nil {
        return set.NewSet[image.Point]()
    }
    return mapObject.Map.FindContinent(x, y)
}

// Player_Colony_Autobuild_CP
func (ai *ChewbotAI) pickWizard(self *playerlib.Player, services playerlib.AIServices, city *citylib.City) chewProduct {
    objective := ai.objective(self)
    turn := int(services.GetTurnNumber())
    buildingInfos := services.GetBuildingInfos()

    // Trade Goods is picked more while few cities make it: 40 and 10 for every 8 cities, less 10
    // for every city on it (see quirkTradeGoodsCountsCity)
    cities, onTradeGoods := 0, 0
    for _, other := range self.Cities {
        if other.Outpost {
            continue
        }
        cities += 1
        counted := other
        if quirkTradeGoodsCountsCity {
            counted = city
        }
        if counted.ProducingBuilding == buildinglib.BuildingTradeGoods && counted.ProducingUnit.IsNone() {
            onTradeGoods += 1
        }
    }
    tradeGoodsChance := 40 + (cities / 8 - onTradeGoods) * 10

    continent := chewContinent(services, city.X, city.Y, city.Plane)

    // the land of the human's fortress, for the first 200 turns
    onHumanLand := false
    if turn <= chewFortressLandTurns {
        for _, other := range services.GetEnemies(self) {
            if !other.IsHuman() {
                continue
            }
            fortress := other.FindFortressCity()
            if fortress != nil && fortress.Plane == city.Plane && continent.Contains(image.Pt(fortress.X, fortress.Y)) {
                onHumanLand = true
            }
        }
    }

    // settlers and engineers: one at a time, on the land or in production
    land := chewLandUnits{}
    for _, stack := range self.Stacks {
        if stack.Plane() != city.Plane || !continent.Contains(image.Pt(stack.X(), stack.Y())) {
            continue
        }
        for _, unit := range stack.Units() {
            raw := unit.GetRawUnit()
            land.Settlers = land.Settlers || chewIsBuilder(&raw)
            land.Engineers = land.Engineers || chewIsEngineer(&raw)
            if chewIsTransport(&raw) {
                land.Transports += 1
            }
        }
    }
    for _, other := range self.Cities {
        if other == city || other.ProducingUnit.IsNone() {
            continue
        }
        land.Settlers = land.Settlers || chewIsBuilder(&other.ProducingUnit)
        land.Engineers = land.Engineers || chewIsEngineer(&other.ProducingUnit)
    }
    needSettlers := !land.Settlers
    needEngineers := !land.Engineers

    // are the defenders of the city enough?
    defenders := 0
    if stack := self.FindStack(city.X, city.Y, city.Plane); stack != nil {
        defenders = len(stack.Units())
    }

    needUnits := false
    if onHumanLand {
        needUnits = 1 + rand.IntN(chewDefendersOnFortressLand) > defenders
    } else {
        needUnits = 1 + rand.IntN(chewDefendersElsewhere) > defenders

        // away from the human, the buildings of an army come first, one for every defender
        for index, building := range []buildinglib.Building{buildinglib.BuildingBarracks, buildinglib.BuildingBuildersHall, buildinglib.BuildingShrine, buildinglib.BuildingSmithy, buildinglib.BuildingGranary} {
            if defenders > index && !city.Buildings.Contains(building) && city.ComputePossibleBuildings(true).Contains(building) {
                return chewProduct{Building: building, Unit: units.UnitNone}
            }
        }
    }

    if 1 + rand.IntN(100) <= tradeGoodsChance {
        return chewProduct{Building: buildinglib.BuildingTradeGoods, Unit: units.UnitNone}
    }

    products := chewProducts(city)
    weights := make([]int, len(products))

    for index, product := range products {
        if product.isUnit() {
            weights[index] = chewUnitWeight(&product.Unit)
        } else {
            category := buildingInfos.GetCategory(product.Building)
            weights[index] = chewCategoryBase(category) + chewCategoryObjective(category, objective)
        }
    }

    // Housing counts twice in a small city. the original looks at the population of another
    // city here, the one after the last of the list: an internal mistake, the city itself is used
    if city.Citizens() < 5 {
        for index, product := range products {
            if product.Building == buildinglib.BuildingHousing {
                weights[index] *= 2
            }
        }
    }

    objectiveScale := func(weight int, row int) int {
        return weight * (chewBuildObjective[row][objective] + 10) / 10
    }

    for index, product := range products {
        if !product.isUnit() {
            if needUnits {
                weights[index] = 0
            }
            continue
        }

        unit := &product.Unit
        switch {
            case chewIsBuilder(unit):
                if needSettlers && city.Citizens() > 2 {
                    if turn < chewEarlyTurns {
                        weights[index] = weights[index] * 3 / 2
                    }
                    weights[index] = objectiveScale(weights[index], chewRowBuilders)
                } else {
                    weights[index] = 0
                }
            case chewIsEngineer(unit):
                if needEngineers {
                    if turn < chewEarlyTurns {
                        weights[index] = weights[index] * 3 / 2
                    }
                    weights[index] = objectiveScale(weights[index], chewRowBuilders)
                } else {
                    weights[index] = 0
                }
            case chewIsTransport(unit):
                if turn / 40 < land.Transports || land.Transports > 8 {
                    weights[index] = 0
                } else {
                    weights[index] = weights[index] * (turn / 40 - land.Transports)
                }
            default:
                if needUnits {
                    weights[index] = objectiveScale(weights[index], chewRowCombatUnits)
                } else {
                    weights[index] = 0
                }
        }
    }

    if len(products) == 0 {
        return chewProduct{Building: buildinglib.BuildingTradeGoods, Unit: units.UnitNone}
    }
    return products[chewWeightedChoice(weights)]
}

// Player_Colony_Autobuild_NP: a neutral city builds Barracks first, then garrison or buildings
func (ai *ChewbotAI) pickNeutral(self *playerlib.Player, city *citylib.City) chewProduct {
    if !city.Buildings.Contains(buildinglib.BuildingBarracks) && city.ComputePossibleBuildings(true).Contains(buildinglib.BuildingBarracks) {
        return chewProduct{Building: buildinglib.BuildingBarracks, Unit: units.UnitNone}
    }

    defenders := 0
    if stack := self.FindStack(city.X, city.Y, city.Plane); stack != nil {
        defenders = len(stack.Units())
    }
    needUnits := chewNeutralDefendersBase + 1 + rand.IntN(chewNeutralDefendersRoll) > defenders

    products := chewProducts(city)
    weights := make([]int, len(products))
    for index, product := range products {
        if product.isUnit() {
            unit := &product.Unit
            if chewIsBuilder(unit) || chewIsEngineer(unit) || chewIsTransport(unit) || !needUnits {
                continue
            }
            weights[index] = chewUnitWeight(unit)
        } else {
            if needUnits || product.Building == buildinglib.BuildingShipwrightsGuild || product.Building == buildinglib.BuildingMaritimeGuild {
                continue
            }
            weights[index] = chewCategoryBase(city.BuildingInfo.GetCategory(product.Building))
        }
    }

    if len(products) == 0 {
        return chewProduct{Building: buildinglib.BuildingTradeGoods, Unit: units.UnitNone}
    }
    return products[chewWeightedChoice(weights)]
}

func chewCategoryBase(category int) int {
    if category < 0 || category >= len(chewBuildBase) {
        return 0
    }
    return chewBuildBase[category]
}

func chewCategoryObjective(category int, objective chewObjective) int {
    if category < 0 || category >= len(chewBuildObjective) {
        return 0
    }
    return chewBuildObjective[category][objective]
}
