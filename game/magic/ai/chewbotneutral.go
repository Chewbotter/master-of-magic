package ai

// The neutral player's turn of the original: farmers of its cities, where its stacks outside the
// cities go, raiders that leave its cities, monsters that come out of lairs, garrisons that are too
// large (ReMoM MoM/src/AIDUDES.c: AI_Next_Turn, NPC_Farmers; MoM/src/AIDATA.c: NPC_Destinations,
// Build_NPC_Stacks, Make_Raiders, Make_Monsters, NPC_Excess_Garrison; MoM/src/AIMOVE.c:
// Adjacent_Free_Square, Make_Monster_List). What its cities build is chewbot.go's
// (Player_Colony_Autobuild_NP). The rules in words: docs/mod/ai-neutral.md. The code is ours.

import (
    "cmp"
    "log"
    "slices"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/game/magic/maplib"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    buildinglib "github.com/kazzmir/master-of-magic/game/magic/building"
    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

// false: the neutral player is the clone's (units made in its cities from nothing, all farmers)
var ChewbotNeutral = true

// dev: log what the neutral player does
var ChewbotNeutralLog = false

func chewbotNeutralActive() bool {
    return ChewbotNeutral && display.ChewbotAI()
}

func chewNeutralLog(format string, args ...any) {
    if ChewbotNeutralLog {
        log.Printf("chewbot neutral: " + format, args...)
    }
}

const (
    // Make_Raiders: Barbarian Spearmen and Swordsmen never raid (a comparison of unit numbers that
    // meant to leave out the units before them)
    quirkNoBarbarianRaiders = true

    // Make_Raiders: the accumulator starts raiders at this
    chewRaidersAt = 30
    // Make_Raiders: a failed try adds this to the monsters' accumulator
    chewRaidersFailed = 15
    // Make_Monsters: the monsters' accumulator starts them at 50 less 5 a level; not before this turn
    chewMonstersAt = 50
    chewMonstersTurn = 50
    // Make_Monster_List: nothing below this budget, at most this many tries
    chewMonsterLeast = 25
    chewMonsterTries = 2000
    // tries of a city or lair
    chewNeutralTries = 1000
    // Build_NPC_Stacks: at most this many stacks; NPC_Destinations: at most this many cities looked at
    chewNeutralStacksMost = 80
    chewNeutralTargetsMost = 30
    // a stack
    chewStackMost = 9
)

// what the neutral player keeps from turn to turn (NOT SAVED)
type chewNeutral struct {
    // Make_Raiders' accumulator (the original keeps it in casting_cost_original)
    Raiders int
    // Make_Monsters' accumulator (average_unit_cost)
    Monsters int
    // where the units outside the cities are going (the original's us_GOTO and dst_wx, dst_wy)
    Orders map[chewUnitKey]data.PlanePoint
}

// the neutral player's turn, in the original's order; the production of its cities comes first
// (cityDecisions)
func (ai *ChewbotAI) neutralDecisions(self *playerlib.Player, services playerlib.AIServices) []playerlib.AIDecision {
    if ai.neutral == nil {
        ai.neutral = &chewNeutral{Orders: make(map[chewUnitKey]data.PlanePoint)}
    }
    world := makeChewWorld(self, services)
    level := citylib.OriginalDifficulty(services.GetDifficulty())

    var out []playerlib.AIDecision
    out = append(out, chewNeutralFarmers(self, services)...)
    out = append(out, ai.neutralDestinations(self, world, level)...)
    out = append(out, ai.makeRaiders(self, world, level)...)
    out = append(out, ai.makeMonsters(self, world, level)...)
    chewExcessGarrison(self, services)
    return out
}

// the units on a square, of any player
func chewUnitsAt(services playerlib.AIServices, x int, y int, plane data.Plane) []units.StackUnit {
    stack, _ := services.FindStack(x, y, plane)
    if stack == nil {
        return nil
    }
    return stack.Units()
}

// NPC_Farmers: farmers for the least farmers of the city and one food for every unit on its square,
// 2 food a farmer (3 for Halflings or with an Animists' Guild), at most all its people. (The
// original adds a number of farmers to a number of units, its slip, kept: it is what its neutral
// cities do.) The game keeps at least the farmers that feed the city (City.ResetCitizens).
func chewNeutralFarmers(self *playerlib.Player, services playerlib.AIServices) []playerlib.AIDecision {
    var out []playerlib.AIDecision
    for _, city := range chewSortedCities(self) {
        least := city.ComputeSubsistenceFarmers()
        need := least + len(chewUnitsAt(services, city.X, city.Y, city.Plane))
        perFarmer := 2
        if city.Race == data.RaceHalfling || city.Buildings.Contains(buildinglib.BuildingAnimistsGuild) {
            perFarmer = 3
        }
        farmers := min((need + perFarmer - 1) / perFarmer, city.Citizens())
        out = append(out, &playerlib.AIUpdateCityDecision{City: city, Farmers: farmers, Workers: city.Citizens() - farmers})
    }
    return out
}

// the cities of a player in a fixed order: by plane, row and column (the original goes by the
// number of the city, which the game does not keep)
func chewSortedCities(player *playerlib.Player) []*citylib.City {
    var out []*citylib.City
    for _, city := range player.Cities {
        out = append(out, city)
    }
    chewSortCityList(out)
    return out
}

func chewSortCityList(cities []*citylib.City) {
    slices.SortFunc(cities, func(a *citylib.City, b *citylib.City) int {
        return cmp.Or(cmp.Compare(a.Plane, b.Plane), cmp.Compare(a.Y, b.Y), cmp.Compare(a.X, b.X))
    })
}

// the cities of the wizards (not the neutral player's) in that order, with their owners
func chewWizardCities(world *chewWorld) []chewCity {
    var out []chewCity
    for _, city := range world.Cities {
        if !city.Owner.IsNeutral() {
            out = append(out, city)
        }
    }
    slices.SortFunc(out, func(a chewCity, b chewCity) int {
        return cmp.Or(cmp.Compare(a.City.Plane, b.City.Plane), cmp.Compare(a.City.Y, b.City.Y), cmp.Compare(a.City.X, b.City.X))
    })
    return out
}

// NPC_Destinations: the worth of a city as a target: 10 less the units in it less its distance,
// 5 more for a city of the human at hard and 5 more at impossible
func (ai *ChewbotAI) neutralTargetValue(world *chewWorld, level int, x int, y int, plane data.Plane, stackX int, stackY int) int {
    value := 10 - len(chewUnitsAt(world.Services, x, y, plane)) - world.Distance(x, y, stackX, stackY)
    _, owner := world.Services.FindCity(x, y, plane)
    if owner != nil && owner.IsHuman() {
        if level > 2 {
            value += 5
        }
        if level > 3 {
            value += 5
        }
    }
    return value
}

// NPC_Destinations: every stack of the neutral player outside its cities goes to a city of a
// wizard; one next to a city of a wizard that is worth more than its target turns to it; a stack
// with no city to go to is disbanded. Then every stack with a destination walks toward it
// (AI_Execute_Orders).
func (ai *ChewbotAI) neutralDestinations(self *playerlib.Player, world *chewWorld, level int) []playerlib.AIDecision {
    orders := ai.neutral.Orders
    // the orders of units that are gone or have arrived end
    alive := make(map[chewUnitKey]bool)
    for _, stack := range self.Stacks {
        for _, unit := range stack.Units() {
            alive[chewKey(unit)] = true
            if order, has := orders[chewKey(unit)]; has && order.X == stack.X() && order.Y == stack.Y() && order.Plane == stack.Plane() {
                delete(orders, chewKey(unit))
            }
        }
    }
    for key := range orders {
        if !alive[key] {
            delete(orders, key)
        }
    }

    wizardCities := chewWizardCities(world)
    setOrder := func(stack *playerlib.UnitStack, x int, y int) {
        for _, unit := range stack.Units() {
            orders[chewKey(unit)] = data.PlanePoint{X: x, Y: y, Plane: stack.Plane()}
        }
    }

    var decisions []playerlib.AIDecision
    count := 0
    for _, stack := range slices.Clone(self.Stacks) {
        if stack.IsEmpty() {
            continue
        }
        if city, _ := world.Services.FindCity(stack.X(), stack.Y(), stack.Plane()); city != nil {
            continue
        }
        if count >= chewNeutralStacksMost {
            break
        }
        count += 1

        wp := chewPlaneIndex(stack.Plane())
        landmass := world.LandmassAt(stack.X(), stack.Y(), wp)
        order, going := orders[chewKey(stack.Units()[0])]

        // next to a city of a wizard while on its way: that city if it is worth more
        if going {
            for _, next := range wizardCities {
                city := next.City
                if city.Plane != stack.Plane() || world.Distance(city.X, city.Y, stack.X(), stack.Y()) >= 2 {
                    continue
                }
                if city.X != order.X || city.Y != order.Y {
                    nextValue := ai.neutralTargetValue(world, level, city.X, city.Y, city.Plane, stack.X(), stack.Y())
                    currentValue := ai.neutralTargetValue(world, level, order.X, order.Y, order.Plane, stack.X(), stack.Y())
                    if nextValue > currentValue {
                        chewNeutralLog("%v at %v,%v turns to %v", stack.Units()[0].GetName(), stack.X(), stack.Y(), city.Name)
                        setOrder(stack, city.X, city.Y)
                    }
                }
                break
            }
        }

        if !going {
            // Build_NPC_Stacks marks a stack of fliers or swimmers as bound to its landmass and a
            // stack with walkers as free, the wrong way round; meant (and done here): walkers only
            // go to cities on their landmass
            landOnly := stack.AnyLandWalkers()
            best := -1000
            var target *citylib.City
            examined := 0
            for _, next := range wizardCities {
                if examined >= chewNeutralTargetsMost {
                    break
                }
                city := next.City
                if city.Plane != stack.Plane() {
                    continue
                }
                if landOnly && world.LandmassAt(city.X, city.Y, wp) != landmass {
                    continue
                }
                value := ai.neutralTargetValue(world, level, city.X, city.Y, city.Plane, stack.X(), stack.Y())
                if value > best {
                    best = value
                    target = city
                }
                examined += 1
            }
            if target != nil {
                chewNeutralLog("%v at %v,%v goes to %v", stack.Units()[0].GetName(), stack.X(), stack.Y(), target.Name)
                setOrder(stack, target.X, target.Y)
            } else {
                chewNeutralLog("%v units at %v,%v have nowhere to go and are disbanded", stack.Size(), stack.X(), stack.Y())
                for _, unit := range stack.Units() {
                    self.RemoveUnit(unit)
                }
                continue
            }
        }

        order = orders[chewKey(stack.Units()[0])]
        if stack.OutOfMoves() {
            continue
        }
        path, _ := world.Services.FindPath(stack.X(), stack.Y(), order.X, order.Y, self, stack, self.GetFog(stack.Plane()))
        if len(path) > 0 {
            decisions = append(decisions, &playerlib.AIMoveStackDecision{Stack: stack, Path: path})
        }
    }
    return decisions
}

// Adjacent_Free_Square: the first square around a place, row by row from the upper left, that is
// land with no unit, lair or city on it
func chewAdjacentFreeSquare(world *chewWorld, x int, y int, plane data.Plane) (int, int, bool) {
    mapObject := world.Maps[chewPlaneIndex(plane)]
    for dy := -1; dy <= 1; dy++ {
        for dx := -1; dx <= 1; dx++ {
            if dx == 0 && dy == 0 {
                continue
            }
            squareX := world.WrapX(x + dx)
            squareY := y + dy
            if !world.inside(squareY) || !mapObject.GetTile(squareX, squareY).Tile.IsLand() {
                continue
            }
            if len(chewUnitsAt(world.Services, squareX, squareY, plane)) > 0 {
                continue
            }
            if mapObject.GetEncounter(squareX, squareY) != nil {
                continue
            }
            if city, _ := world.Services.FindCity(squareX, squareY, plane); city != nil {
                continue
            }
            return squareX, squareY, true
        }
    }
    return 0, 0, false
}

// a city of a wizard on the landmass
func chewWizardCityOn(world *chewWorld, plane data.Plane, landmass int) bool {
    wp := chewPlaneIndex(plane)
    for _, city := range world.Cities {
        if !city.Owner.IsNeutral() && city.City.Plane == plane && world.LandmassAt(city.City.X, city.City.Y, wp) == landmass {
            return true
        }
    }
    return false
}

// the fortress of a computer wizard on the landmass
func chewComputerFortressOn(world *chewWorld, plane data.Plane, landmass int) bool {
    wp := chewPlaneIndex(plane)
    for _, player := range world.Players {
        if player.IsHuman() || player.IsNeutral() {
            continue
        }
        if fortress := chewFortress(player); fortress != nil && fortress.Plane == plane && world.LandmassAt(fortress.X, fortress.Y, wp) == landmass {
            return true
        }
    }
    return false
}

// Make_Raiders: the accumulator grows by 1 to level + 1 a turn; at 30 a neutral city with a city of a
// wizard on its landmass, a free square next to it and a garrison sends (garrison x level / 6) units
// (two thirds with a computer wizard's fortress on the landmass, half on Myrror before turn 200, at
// least 1) of the kinds of its garrison, and a third as many units of its garrison go
func (ai *ChewbotAI) makeRaiders(self *playerlib.Player, world *chewWorld, level int) []playerlib.AIDecision {
    cities := chewSortedCities(self)
    if len(cities) == 0 {
        return nil
    }
    ai.neutral.Raiders += chewRandom(level + 1)
    if ai.neutral.Raiders < chewRaidersAt {
        return nil
    }
    ai.neutral.Raiders = 0

    // why tries fail, for the log: no wizard's city on the landmass, no free square, no garrison
    var failed [3]int
    for range chewNeutralTries {
        city := cities[chewRoll(len(cities))]
        landmass := world.LandmassAt(city.X, city.Y, chewPlaneIndex(city.Plane))
        if !chewWizardCityOn(world, city.Plane, landmass) {
            failed[0] += 1
            continue
        }
        spawnX, spawnY, free := chewAdjacentFreeSquare(world, city.X, city.Y, city.Plane)
        if !free {
            failed[1] += 1
            continue
        }
        garrison := chewUnitsAt(world.Services, city.X, city.Y, city.Plane)
        if len(garrison) == 0 {
            failed[2] += 1
            continue
        }

        count := len(garrison) * level / 6
        if chewComputerFortressOn(world, city.Plane, landmass) {
            count = count * 2 / 3
        }
        if city.Plane == data.PlaneMyrror && world.Turn < 200 {
            count /= 2
        }
        count = max(count, 1)

        var out []playerlib.AIDecision
        for range count {
            unit := garrison[chewRoll(len(garrison))].GetRawUnit()
            if quirkNoBarbarianRaiders && (unit.Equals(units.BarbarianSpearmen) || unit.Equals(units.BarbarianSwordsmen)) {
                continue
            }
            if chewIsTransport(&unit) || chewIsBuilder(&unit) {
                continue
            }
            out = append(out, &playerlib.AICreateUnitDecision{Unit: unit, X: spawnX, Y: spawnY, Plane: city.Plane})
        }
        for index := range len(out) / 3 {
            self.RemoveUnit(garrison[index])
        }
        chewNeutralLog("raiders: %v units out of %v (%v, garrison %v %v) to %v,%v", len(out), city.Name, city.Race, len(garrison), garrison[0].GetRawUnit().Name, spawnX, spawnY)
        return out
    }

    chewNeutralLog("raiders: none (tries without a wizard's city on the landmass %v, without a free square %v, without a garrison %v)", failed[0], failed[1], failed[2])
    ai.neutral.Monsters += chewRaidersFailed
    return nil
}

// a lair, tower or node with its place
type chewLair struct {
    Encounter *maplib.ExtraEncounter
    X int
    Y int
    Plane data.Plane
}

// every lair, tower and node of both planes in a fixed order
func chewLairs(world *chewWorld) []chewLair {
    var out []chewLair
    for wp := range 2 {
        for _, point := range chewEncounterPoints(world.Maps[wp]) {
            out = append(out, chewLair{Encounter: world.Maps[wp].GetEncounter(point.X, point.Y), X: point.X, Y: point.Y, Plane: chewPlaneOf(wp)})
        }
    }
    return out
}

// a lair whose guardians can rampage: it has them and the first is not of life
func chewRampageLair(lair chewLair) bool {
    return lair.Encounter != nil && len(lair.Encounter.Units) > 0 && lair.Encounter.Units[0].Realm != data.LifeMagic
}

// the realm of the monsters of a lair: the one of its first guardian, death for a guardian of a race
func chewLairRealm(lair chewLair) data.MagicType {
    switch realm := lair.Encounter.Units[0].Realm; realm {
        case data.ChaosMagic, data.DeathMagic, data.NatureMagic, data.SorceryMagic, data.ArcaneMagic: return realm
    }
    return data.DeathMagic
}

// Make_Monster_List: creatures of the realm, each costing no more than the budget left, until
// less than 25 is left, 2000 tries or a full stack
func chewMonsterList(budget int, realm data.MagicType) []units.Unit {
    var pool []units.Unit
    for _, unit := range units.AllUnits {
        if unit.CastingCost > 0 {
            pool = append(pool, unit)
        }
    }
    var out []units.Unit
    budget = max(budget, 0)
    for tries := 0; budget >= chewMonsterLeast && tries < chewMonsterTries && len(pool) > 0; tries++ {
        unit := pool[chewRoll(len(pool))]
        if unit.Realm == realm && unit.CastingCost <= budget && !chewIsTransport(&unit) && len(out) < chewStackMost {
            budget -= unit.CastingCost
            out = append(out, unit)
        }
    }
    return out
}

// Make_Monsters: the accumulator grows by 1 to level + 1 a turn; at 50 less 5 a level, from turn 50
// on, a lair (tower, node) with guardians not of life on a landmass with a city of a wizard sends
// creatures of the realm of its guardians to a free square next to it, for a budget of
// (1 to level + 1, twice) x turn / 5, half on a landmass with a computer wizard's fortress and
// without the human's
func (ai *ChewbotAI) makeMonsters(self *playerlib.Player, world *chewWorld, level int) []playerlib.AIDecision {
    lairs := chewLairs(world)
    if !slices.ContainsFunc(lairs, chewRampageLair) {
        return nil
    }
    ai.neutral.Monsters += chewRandom(level + 1)
    if ai.neutral.Monsters < chewMonstersAt - level * 5 {
        return nil
    }
    ai.neutral.Monsters = 0
    if world.Turn < chewMonstersTurn {
        return nil
    }

    for range chewNeutralTries {
        lair := lairs[chewRoll(len(lairs))]
        if !chewRampageLair(lair) {
            continue
        }
        wp := chewPlaneIndex(lair.Plane)
        landmass := world.LandmassAt(lair.X, lair.Y, wp)
        if !chewWizardCityOn(world, lair.Plane, landmass) {
            continue
        }

        computerHome := false
        human := chewHuman(self, world.Services)
        humanFortress := (*citylib.City)(nil)
        if human != nil {
            humanFortress = chewFortress(human)
        }
        if humanFortress == nil || humanFortress.Plane != lair.Plane || world.LandmassAt(humanFortress.X, humanFortress.Y, wp) != landmass {
            computerHome = chewComputerFortressOn(world, lair.Plane, landmass)
        }

        spawnX, spawnY, free := chewAdjacentFreeSquare(world, lair.X, lair.Y, lair.Plane)
        if !free {
            return nil
        }
        budget := (chewRandom(level + 1) + chewRandom(level + 1)) * world.Turn / 5
        if computerHome {
            budget /= 2
        }
        realm := chewLairRealm(lair)
        var out []playerlib.AIDecision
        for _, unit := range chewMonsterList(budget, realm) {
            out = append(out, &playerlib.AICreateUnitDecision{Unit: unit, X: spawnX, Y: spawnY, Plane: lair.Plane})
        }
        chewNeutralLog("monsters: %v of %v (budget %v) from %v,%v to %v,%v", len(out), realm, budget, lair.X, lair.Y, spawnX, spawnY)
        return out
    }
    return nil
}

// the cost of a unit as the original's table has it: production, or casting for a creature
func chewUnitCost(unit units.StackUnit) int {
    raw := unit.GetRawUnit()
    if raw.ProductionCost > 0 {
        return raw.ProductionCost
    }
    return raw.CastingCost
}

// NPC_Excess_Garrison: a neutral city with more units than its people (2 more with a Granary, 2 more
// with a Farmers' Market) loses its cheapest unit, one a turn
func chewExcessGarrison(self *playerlib.Player, services playerlib.AIServices) {
    for _, city := range chewSortedCities(self) {
        garrison := chewUnitsAt(services, city.X, city.Y, city.Plane)
        most := city.Citizens()
        if city.Buildings.Contains(buildinglib.BuildingGranary) {
            most += 2
        }
        if city.Buildings.Contains(buildinglib.BuildingFarmersMarket) {
            most += 2
        }
        if len(garrison) <= most {
            continue
        }
        var cheapest units.StackUnit
        lowest := 1000
        for _, unit := range garrison {
            if cost := chewUnitCost(unit); cost < lowest {
                lowest = cost
                cheapest = unit
            }
        }
        if cheapest != nil {
            chewNeutralLog("%v has %v units for %v, %v goes", city.Name, len(garrison), most, cheapest.GetName())
            self.RemoveUnit(cheapest)
        }
    }
}
