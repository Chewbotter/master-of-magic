package ai

// What the overland AI of Chewbot thinks of every landmass, of every other wizard, and which
// landmass it makes war on. Facts from the ReMoM project's reconstruction (MoM/src/AIMOVE.c:
// AI_Evaluate_Continents, AI_Reevaluate_Continent, AI_Choose_War_Landmass). Hostility is kept by
// the rules of diplomacy (relations/). The code is ours. The rules in words: docs/mod/ai-overland.md.

import (
    "image"
    "math/rand/v2"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

// what a landmass is to the wizard (e_LANDMASS_TYPE)
const (
    chewLandUnevaluated = iota
    // its cities and more than ten times the cost of units of all others on it
    chewLandOwn
    // its cities, and others have units there
    chewLandContested
    chewLandNoOwnCity
    // left alone: an ally has a city there, or there is no free square for a stage point
    chewLandIgnored
    // an army of 8 found nothing to attack and waits at a coast to go to the war landmass
    chewLandLeaveable
    // nothing to attack
    chewLandNoTargets
)

// numbers of the original
const (
    // AI_Evaluate_Continents: every 25 + 1..10 + 1..10 turns
    chewContinentsEvery = 25
    chewContinentsRoll = 10
    // a landmass is its own when the cost of its units is more than this times the others'
    chewOwnLandFactor = 10
)

// the state of the overland AI that lasts from turn to turn (the original's _ai_continents and
// the arrays beside it); not saved
type chewOverland struct {
    // per plane, per landmass: its type and stage point
    LandType [2][]int
    StageX [2][]int
    StageY [2][]int
    // the landmass the wizard makes war on, per plane; 0 is none
    WarTarget [2]int
    // the landmass and square settlers go to by sea, per plane; landmass 0 is none
    SettlerLandmass [2]int
    SettlerX [2]int
    SettlerY [2]int
    ContinentsCountdown int

    // the wizard whose state this is
    Self *playerlib.Player

    Orders map[chewUnitKey]*chewOrder
    // the move of a unit failed in the last turn: it wants a path to a new target
    MoveFailed map[chewUnitKey]bool
}

func makeChewOverland() *chewOverland {
    return &chewOverland{
        Orders: make(map[chewUnitKey]*chewOrder),
        MoveFailed: make(map[chewUnitKey]bool),
    }
}

// every roll of Chewbot: 0 to n - 1. Tests put a function here that gives the same numbers every
// time (ChewbotFixedRolls)
var chewRoll = func(n int) int {
    return rand.IntN(n)
}

// development: every roll of Chewbot gives this, so a test gets the same orders every time; the
// lowest (0) means every chance the original rolls for happens. Returns the function that undoes it
func ChewbotFixedRolls(value int) func() {
    old := chewRoll
    chewRoll = func(n int) int {
        return min(max(value, 0), max(n - 1, 0))
    }
    return func() {
        chewRoll = old
    }
}

// Random(n) of the original: 1 to n
func chewRandom(n int) int {
    if n <= 0 {
        return 1
    }
    return 1 + chewRoll(n)
}

// the arrays of a plane as long as it has landmasses (a change of land and sea keeps what it can)
func (overland *chewOverland) fit(world *chewWorld) {
    for wp := range 2 {
        count := world.LandmassCount(wp) + 1
        for len(overland.LandType[wp]) < count {
            overland.LandType[wp] = append(overland.LandType[wp], chewLandUnevaluated)
            overland.StageX[wp] = append(overland.StageX[wp], 0)
            overland.StageY[wp] = append(overland.StageY[wp], 0)
        }
        overland.LandType[wp] = overland.LandType[wp][:count]
        overland.StageX[wp] = overland.StageX[wp][:count]
        overland.StageY[wp] = overland.StageY[wp][:count]
        if overland.WarTarget[wp] >= count {
            overland.WarTarget[wp] = 0
        }
        if overland.SettlerLandmass[wp] >= count {
            overland.SettlerLandmass[wp] = 0
        }
    }
}

// the wizard's hostility to another (kept by the rules of diplomacy, relations/): to the neutral
// player RaiderHostility, to a wizard it has not met 0
func (overland *chewOverland) hostilityOf(player *playerlib.Player) int {
    if overland.Self == nil || player == nil {
        return 0
    }
    if player.IsNeutral() {
        return overland.Self.RaiderHostility
    }
    if relation, ok := overland.Self.PlayerRelations[player]; ok {
        return relation.Hostility
    }
    return 0
}

func chewTreaty(self *playerlib.Player, other *playerlib.Player) data.TreatyType {
    if relation, ok := self.PlayerRelations[other]; ok {
        return relation.Treaty
    }
    return data.TreatyNone
}

func chewCastingMastery(player *playerlib.Player) bool {
    return player.CastingSpell.Name == "Spell of Mastery"
}

// the unit cost of the units of the wizard and of all others on every landmass of a plane,
// settlers not counted
func (world *chewWorld) unitCosts(wp int) ([]int, []int) {
    own := make([]int, world.LandmassCount(wp) + 1)
    others := make([]int, world.LandmassCount(wp) + 1)
    for _, player := range world.Players {
        for _, stack := range player.Stacks {
            if chewPlaneIndex(stack.Plane()) != wp {
                continue
            }
            landmass := world.LandmassAt(stack.X(), stack.Y(), wp)
            for _, unit := range stack.Units() {
                if unit.HasAbility(data.AbilityCreateOutpost) {
                    continue
                }
                if player == world.Self {
                    own[landmass] += unit.GetProductionCost()
                } else {
                    others[landmass] += unit.GetProductionCost()
                }
            }
        }
    }
    return own, others
}

// the middle of the wizard's cities on every landmass of a plane, and how many there are
func (world *chewWorld) cityCenters(wp int) ([]int, []int, []int) {
    sumX := make([]int, world.LandmassCount(wp) + 1)
    sumY := make([]int, world.LandmassCount(wp) + 1)
    count := make([]int, world.LandmassCount(wp) + 1)
    for _, city := range world.Cities {
        if city.Owner != world.Self || chewPlaneIndex(city.City.Plane) != wp {
            continue
        }
        landmass := world.LandmassAt(city.City.X, city.City.Y, wp)
        sumX[landmass] += city.City.X
        sumY[landmass] += city.City.Y
        count[landmass] += 1
    }
    for landmass := range count {
        if count[landmass] > 0 {
            sumX[landmass] /= count[landmass]
            sumY[landmass] /= count[landmass]
        }
    }
    return sumX, sumY, count
}

// the type of a landmass by the wizard's cities and the units on it, and an ally's city
func (world *chewWorld) classify(wp int, landmass int, ownCities int, ownCost int, otherCost int) int {
    kind := chewLandNoOwnCity
    if ownCities > 0 {
        if ownCost > otherCost * chewOwnLandFactor {
            kind = chewLandOwn
        } else {
            kind = chewLandContested
        }
    }
    if kind == chewLandNoOwnCity {
        for _, city := range world.Cities {
            if city.Owner == world.Self || city.Owner.IsNeutral() || chewPlaneIndex(city.City.Plane) != wp {
                continue
            }
            if world.LandmassAt(city.City.X, city.City.Y, wp) == landmass && chewTreaty(world.Self, city.Owner) == data.TreatyAlliance {
                kind = chewLandIgnored
            }
        }
    }
    return kind
}

// the free square (nothing on the evaluation map) of a list nearest to a point, with a jitter of
// 1..jitter added to every distance (0 for none); false when there is none
func (world *chewWorld) nearestFree(squares []image.Point, wp int, x int, y int, jitter int) (image.Point, bool) {
    best := 1000
    var found image.Point
    for _, square := range squares {
        distance := world.Distance(square.X, square.Y, x, y)
        if jitter > 0 {
            distance += chewRandom(jitter)
        }
        if distance < best && world.EvalAt(square.X, square.Y, wp) == 0 {
            best = distance
            found = square
        }
    }
    return found, best < 1000
}

// the middle of the dock squares of a landmass
func (world *chewWorld) dockCenter(wp int, landmass int) (int, int, bool) {
    docks := world.DockSquares[wp][landmass]
    if landmass <= 0 || len(docks) == 0 {
        return 0, 0, false
    }
    sumX, sumY := 0, 0
    for _, dock := range docks {
        sumX += dock.X
        sumY += dock.Y
    }
    return sumX / len(docks), sumY / len(docks), true
}

// AI_Evaluate_Continents: every 25 to 45 turns, at once when the stage point of a landmass without
// the wizard's cities is taken. Orders to squares that are empty now are dropped; every landmass
// gets its type and stage point; the landmass settlers go to by sea is picked
func (overland *chewOverland) evaluateContinents(world *chewWorld) {
    force := false
    for wp := range 2 {
        for landmass := range overland.LandType[wp] {
            if overland.LandType[wp][landmass] == chewLandNoOwnCity && world.EvalAt(overland.StageX[wp][landmass], overland.StageY[wp][landmass], wp) != 0 {
                force = true
            }
        }
    }
    if force {
        overland.ContinentsCountdown = 0
    }
    overland.ContinentsCountdown -= 1
    if overland.ContinentsCountdown > 0 {
        return
    }
    overland.ContinentsCountdown = chewContinentsEvery + chewRandom(chewContinentsRoll) + chewRandom(chewContinentsRoll)

    // orders to a square where nothing is any more end (the original: all but engineers)
    for key, order := range overland.Orders {
        if order.Kind != chewOrderGoto && order.Kind != chewOrderAttack {
            continue
        }
        if world.EvalAt(order.X, order.Y, chewPlaneIndex(order.Plane)) == 0 {
            if order.Why == "target" {
                chewMoveLog("%v: the target at %v,%v is empty, the order ends", world.Self.Wizard.Name, order.X, order.Y)
            }
            delete(overland.Orders, key)
        }
    }

    for wp := range 2 {
        centerX, centerY, cities := world.cityCenters(wp)
        own, others := world.unitCosts(wp)
        for landmass := 1; landmass <= world.LandmassCount(wp); landmass++ {
            if cities[landmass] > 0 {
                overland.StageX[wp][landmass] = centerX[landmass]
                overland.StageY[wp][landmass] = centerY[landmass]
            }
            if overland.LandType[wp][landmass] == chewLandNoTargets && others[landmass] == 0 && cities[landmass] == 0 {
                continue
            }
            overland.LandType[wp][landmass] = world.classify(wp, landmass, cities[landmass], own[landmass], others[landmass])
        }
        overland.LandType[wp][0] = chewLandIgnored

        // the landmass for settlers by sea: no city of the wizard, no unit of another, a free
        // square nearest to the fortress with a jitter of 1..20
        fortressX, fortressY := 0, 0
        if fortress := chewFortress(world.Self); fortress != nil {
            fortressX, fortressY = fortress.X, fortress.Y
        }
        best := 1000
        overland.SettlerLandmass[wp] = 0
        for landmass := 1; landmass <= world.LandmassCount(wp); landmass++ {
            if overland.LandType[wp][landmass] != chewLandNoOwnCity || others[landmass] != 0 {
                continue
            }
            for _, square := range world.LandSquares[wp][landmass] {
                if world.EvalAt(square.X, square.Y, wp) != 0 {
                    continue
                }
                distance := chewRandom(20) + world.Distance(square.X, square.Y, fortressX, fortressY)
                if distance < best {
                    best = distance
                    overland.SettlerLandmass[wp] = landmass
                    overland.SettlerX[wp] = square.X
                    overland.SettlerY[wp] = square.Y
                }
            }
        }

        for landmass := 1; landmass <= world.LandmassCount(wp); landmass++ {
            switch overland.LandType[wp][landmass] {
                case chewLandOwn, chewLandContested, chewLandIgnored:
                    // the stage point is the middle of its cities, or the free square nearest to it
                    x, y := overland.StageX[wp][landmass], overland.StageY[wp][landmass]
                    if world.EvalAt(x, y, wp) == 0 && world.LandmassAt(x, y, wp) == landmass {
                        continue
                    }
                    square, ok := world.nearestFree(world.LandSquares[wp][landmass], wp, x, y, 0)
                    overland.setStage(wp, landmass, square, ok)
                case chewLandNoOwnCity:
                    // a free coast square near the old stage point
                    square, ok := world.nearestFree(world.DockSquares[wp][landmass], wp, overland.StageX[wp][landmass], overland.StageY[wp][landmass], 20)
                    overland.setStage(wp, landmass, square, ok)
            }
        }

        // with a war landmass, the stage point of a landmass that is the wizard's own is the free
        // coast square nearest to the middle of the war landmass's coast
        if overland.WarTarget[wp] != 0 {
            warX, warY, ok := world.dockCenter(wp, overland.WarTarget[wp])
            if ok {
                for landmass := 1; landmass <= world.LandmassCount(wp); landmass++ {
                    if overland.LandType[wp][landmass] != chewLandOwn {
                        continue
                    }
                    if square, found := world.nearestFree(world.DockSquares[wp][landmass], wp, warX, warY, 0); found {
                        overland.StageX[wp][landmass] = square.X
                        overland.StageY[wp][landmass] = square.Y
                    }
                }
            }
        }
    }
}

// a stage point, or none: then the landmass is left alone
func (overland *chewOverland) setStage(wp int, landmass int, square image.Point, ok bool) {
    if ok {
        overland.StageX[wp][landmass] = square.X
        overland.StageY[wp][landmass] = square.Y
    } else {
        overland.StageX[wp][landmass] = 0
        overland.StageY[wp][landmass] = 0
        overland.LandType[wp][landmass] = chewLandIgnored
    }
}

// AI_Reevaluate_Continent: when 9 units wait at the stage point, one time in 20: all units of the
// landmass drop their orders, the landmass gets its type again, a stage point by the middle of its
// cities (with a jitter of 1..5), and a landmass of its own the coast square toward the war landmass
func (overland *chewOverland) reevaluateContinent(world *chewWorld, wp int, landmass int) {
    for _, stack := range world.Self.Stacks {
        if chewPlaneIndex(stack.Plane()) != wp || world.LandmassAt(stack.X(), stack.Y(), wp) != landmass {
            continue
        }
        for _, unit := range stack.Units() {
            if !unit.HasAbility(data.AbilityConstruction) {
                delete(overland.Orders, chewKey(unit))
            }
        }
    }

    centerX, centerY, cities := world.cityCenters(wp)
    own, others := world.unitCosts(wp)
    if cities[landmass] > 0 {
        overland.StageX[wp][landmass] = centerX[landmass]
        overland.StageY[wp][landmass] = centerY[landmass]
    }
    overland.LandType[wp][landmass] = world.classify(wp, landmass, cities[landmass], own[landmass], others[landmass])

    switch overland.LandType[wp][landmass] {
        case chewLandOwn, chewLandContested, chewLandIgnored:
            square, ok := world.nearestFree(world.LandSquares[wp][landmass], wp, overland.StageX[wp][landmass], overland.StageY[wp][landmass], 5)
            overland.setStage(wp, landmass, square, ok)
    }

    if overland.LandType[wp][landmass] == chewLandOwn {
        if warX, warY, ok := world.dockCenter(wp, overland.WarTarget[wp]); ok {
            if square, found := world.nearestFree(world.DockSquares[wp][landmass], wp, warX, warY, 5); found {
                overland.StageX[wp][landmass] = square.X
                overland.StageY[wp][landmass] = square.Y
            }
        }
    }
}

// AI_Choose_War_Landmass: per plane, a landmass is picked again when there is none, when the
// wizard's army waits to leave, or when the one it has holds no city to fight for any more. The
// weights: cities of the first wizard it is at war with (10 each) and neutral cities (1 each), by
// the distance of the landmass's coast from the middle of the wizard's cities; with no such city,
// the size of every landmass that is not its own
func (overland *chewOverland) chooseWarLandmass(world *chewWorld) {
    var firstHostile *playerlib.Player
    for _, other := range world.Players {
        if other == world.Self || other.IsNeutral() {
            continue
        }
        // the original also takes a wizard that casts Fire Elemental overland (meant: the Spell
        // of Mastery), which never happens; see quirkWarOnMastery
        if chewTreaty(world.Self, other) == data.TreatyWar || (!quirkWarOnMastery && chewCastingMastery(other)) {
            firstHostile = other
            break
        }
    }

    for wp := range 2 {
        reevaluate := false
        current := overland.WarTarget[wp]
        if current == 0 {
            reevaluate = true
        } else {
            switch overland.LandType[wp][current] {
                case chewLandNoOwnCity, chewLandIgnored:
                    counts := make(map[*playerlib.Player]int)
                    for _, city := range world.Cities {
                        if city.Owner != world.Self && chewPlaneIndex(city.City.Plane) == wp && world.LandmassAt(city.City.X, city.City.Y, wp) == current {
                            counts[city.Owner] += 1
                        }
                    }
                    if firstHostile != nil {
                        reevaluate = counts[firstHostile] == 0
                    } else {
                        reevaluate = true
                        for owner, count := range counts {
                            if count == 0 {
                                continue
                            }
                            if owner.IsNeutral() || overland.hostilityOf(owner) >= 2 {
                                reevaluate = false
                            }
                        }
                    }
                case chewLandLeaveable:
                    reevaluate = true
            }
        }
        if !reevaluate {
            continue
        }

        count := world.LandmassCount(wp) + 1
        values := make([]int, count)
        weights := make([]int, count)
        empireX, empireY, ownCities := 0, 0, 0
        for _, city := range world.Cities {
            if chewPlaneIndex(city.City.Plane) != wp {
                continue
            }
            landmass := world.LandmassAt(city.City.X, city.City.Y, wp)
            switch {
                case city.Owner == world.Self:
                    empireX += city.City.X
                    empireY += city.City.Y
                    ownCities += 1
                case firstHostile == nil && city.Owner.IsNeutral():
                    values[landmass] += 1
                case firstHostile != nil && city.Owner == firstHostile:
                    values[landmass] += 10
                case firstHostile != nil && city.Owner.IsNeutral():
                    values[landmass] += 1
            }
        }
        if ownCities == 0 {
            if fortress := chewFortress(world.Self); fortress != nil {
                empireX, empireY = fortress.X, fortress.Y
            }
        } else {
            empireX /= ownCities
            empireY /= ownCities
        }

        for landmass := 1; landmass < count; landmass++ {
            nearest := 1000
            for _, dock := range world.DockSquares[wp][landmass] {
                if world.EvalAt(dock.X, dock.Y, wp) == 0 {
                    nearest = min(nearest, world.Distance(dock.X, dock.Y, empireX, empireY))
                }
            }
            nearest = max(nearest, 1)
            if firstHostile == nil {
                weights[landmass] = values[landmass] * 1000 / nearest
                if values[landmass] > 0 && weights[landmass] < 10 {
                    weights[landmass] = 10
                }
            } else if values[landmass] > 0 {
                weights[landmass] = max(10, (2000 - values[landmass] * 10) / nearest)
            }
        }

        if chewSum(weights) == 0 {
            // the original leaves out the landmass of a fortress that is gone (banished); the game
            // keeps no place of a fortress that is gone, so every landmass counts
            for landmass := 1; landmass < count; landmass++ {
                kind := overland.LandType[wp][landmass]
                if kind == chewLandOwn || kind == chewLandNoTargets {
                    continue
                }
                weights[landmass] = len(world.LandSquares[wp][landmass])
                if weights[landmass] < 10 {
                    weights[landmass] = 0
                }
            }
        }

        if chewSum(weights) == 0 {
            overland.WarTarget[wp] = 0
        } else {
            overland.WarTarget[wp] = chewWeightedChoice(weights)
        }
        chewMoveLog("%v: war landmass on plane %v is %v", world.Self.Wizard.Name, wp, overland.WarTarget[wp])
    }
}

func chewSum(values []int) int {
    total := 0
    for _, value := range values {
        total += value
    }
    return total
}
