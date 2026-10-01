package game

// The cost of a step on the world map as the original game has it. Rules in words:
// docs/mod/movement.md.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/UnitMove.c:
// Army_Movement_Modes, Init_MovePathMap; MainScr.c: Update_MovePathMap, Eval_Move_Path__WIP,
// Army_Moves2, Army_Boatriders; MAPGEN.c: Movement_Mode_Cost_Maps; the table of TERRSTAT.LBX,
// dumped in its doc/___LBX). The code is ours.

import (
    "slices"
    "sync"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/maplib"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/terrain"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    "github.com/kazzmir/master-of-magic/lib/fraction"
)

// TERRSTAT gives mountaineers 3 moves on grassland, and a stack with one mountaineer takes the
// mountaineers' costs whole (kept, the original's)
const quirkMountaineerGrass = true

// a stack that flies moves free on every road, the normal roads of Arcanus too (kept, the
// original's)
const quirkFliersFreeOnRoads = true

// half moves; -1 can not be entered
const noEntry = -1

// the columns of TERRSTAT in half moves: walking, forester, mountaineer, swimming
type terrainCosts struct {
    Walk, Forester, Mountaineer, Swim int
}

var classicTerrainCosts = map[terrain.TerrainType]terrainCosts{
    terrain.Grass: {2, 2, 6, 2},
    terrain.Forest: {4, 2, 4, 4},
    terrain.Mountain: {8, 8, 2, 8},
    terrain.Hill: {6, 6, 2, 6},
    terrain.Desert: {2, 2, 2, 2},
    terrain.Swamp: {6, 6, 6, 2},
    terrain.Volcano: {6, 6, 6, 2},
    terrain.Tundra: {4, 4, 4, 2},
    terrain.SorceryNode: {2, 2, 2, 2},
    terrain.NatureNode: {4, 2, 4, 4},
    terrain.ChaosNode: {8, 8, 2, 8},
    terrain.River: {4, 4, 4, 2},
}

// the movement modes of a stack (Army_Movement_Modes)
type moveModes struct {
    Forester, Mountaineer, Swimming, Flying, Pathfinding bool
}

func unitSwims(unit units.StackUnit) bool {
    return unit.IsSwimmer() || unit.HasAbility(data.AbilityNonCorporeal) || unit.HasEnchantment(data.UnitEnchantmentWaterWalking) || unit.HasEnchantment(data.UnitEnchantmentWraithForm)
}

// the modes of the stacks a path search is running for: worked out once a search, not once a
// step (the searches of late games spent most of their time here); a search that overlaps another
// of the same stack only works them out again
var searchModes sync.Map

func stackModes(stack *playerlib.UnitStack) moveModes {
    if found, ok := searchModes.Load(stack); ok {
        return found.(moveModes)
    }
    return computeStackModes(stack)
}

// the modes of a path search's stack, kept until the returned function is called
func keepStackModes(stack *playerlib.UnitStack) func() {
    searchModes.Store(stack, computeStackModes(stack))
    return func() {
        searchModes.Delete(stack)
    }
}

func computeStackModes(stack *playerlib.UnitStack) moveModes {
    var modes moveModes
    active := stack.ActiveUnits()
    fliers, swimmers, nonCorporeal := 0, 0, 0
    for _, unit := range active {
        // one forester, one mountaineer gives the whole stack the mode
        if unit.HasAbility(data.AbilityForester) {
            modes.Forester = true
        }
        if unit.HasAbility(data.AbilityMountaineer) {
            modes.Mountaineer = true
        }
        if unit.HasEnchantment(data.UnitEnchantmentPathFinding) {
            modes.Pathfinding = true
        }
        if unit.IsFlying() {
            fliers += 1
        } else if unitSwims(unit) {
            swimmers += 1
        }
        if unit.HasAbility(data.AbilityNonCorporeal) || unit.HasEnchantment(data.UnitEnchantmentWraithForm) {
            nonCorporeal += 1
        }
    }
    count := len(active)
    modes.Flying = count > 0 && (fliers == count || stack.AllFlyers())
    modes.Swimming = count > 0 && swimmers > 0 && fliers + swimmers >= count
    // every unit non-corporeal: every square half a move, as with Pathfinding
    if count > 0 && nonCorporeal == count {
        modes.Pathfinding = true
        modes.Swimming = true
    }
    if stack.HasPathfinding() {
        modes.Pathfinding = true
    }
    return modes
}

// the smaller of two costs, where a square that can not be entered stays so (Copy_Memory_Less_Than
// compares signed bytes)
func lessCost(base int, other int) int {
    if base == noEntry {
        return base
    }
    if other != noEntry && other < base {
        return other
    }
    return base
}

// Init_MovePathMap: the cost in half moves of entering a square, noEntry when it can not be
func classicStepCost(stack *playerlib.UnitStack, mapUse *maplib.Map, tile maplib.FullTile, road bool, enchantedRoad bool) int {
    modes := stackModes(stack)

    if modes.Flying {
        if road && quirkFliersFreeOnRoads {
            return 0
        }
        if enchantedRoad {
            return 0
        }
        return 2
    }

    water := !tile.Tile.IsLand()
    if road {
        // the land modes on a road: half a move on Arcanus, nothing on Myrror and enchanted roads
        if enchantedRoad || mapUse.Plane == data.PlaneMyrror {
            return 0
        }
        return 1
    }

    if modes.Pathfinding {
        if water && !modes.Swimming {
            return noEntry
        }
        return 1
    }

    costs, ok := classicTerrainCosts[tile.Tile.TerrainType()]
    if water || !ok {
        costs = terrainCosts{noEntry, noEntry, noEntry, 2}
    }
    mountaineer := costs.Mountaineer
    if !quirkMountaineerGrass && tile.Tile.TerrainType() == terrain.Grass {
        mountaineer = costs.Walk
    }

    switch {
        case modes.Forester:
            cost := costs.Forester
            if modes.Swimming {
                cost = lessCost(cost, costs.Swim)
            }
            if modes.Mountaineer {
                cost = lessCost(cost, mountaineer)
            }
            return cost
        case modes.Mountaineer:
            cost := mountaineer
            if modes.Swimming {
                cost = lessCost(cost, costs.Swim)
            }
            return cost
        case modes.Swimming:
            return costs.Swim
    }
    return costs.Walk
}

// the units of a stack that need a seat on a ship (Army_Boatriders): all but fliers and swimmers;
// none with a wind walker
func boatRiders(stack *playerlib.UnitStack) int {
    riders := 0
    for _, unit := range stack.Units() {
        if unit.HasAbility(data.AbilityWindWalking) || unit.HasEnchantment(data.UnitEnchantmentWindWalking) {
            return 0
        }
        if !unit.IsFlying() && !unitSwims(unit) && !unit.IsSailing() {
            riders += 1
        }
    }
    return riders
}

// the seats of the ships of a stack
func shipSeats(stack *playerlib.UnitStack) int {
    seats := 0
    for _, unit := range stack.Units() {
        seats += int(unit.GetAbilityValue(data.AbilityTransport))
    }
    return seats
}

// the stack limit and the seats of ships, for a step onto a square with another stack of the
// player's: false when it may not
func classicRoomAt(stack *playerlib.UnitStack, other *playerlib.UnitStack, water bool) bool {
    return classicRoomAmong(stack, []*playerlib.UnitStack{other}, water)
}

// the same against all the stacks of the player's on the square: the units of a square are one
// group in the original, and a computer wizard's army can stand there in several stacks within a
// turn (its groups go their own ways), which were merged into one of more than 9 at the end of it
func classicRoomAmong(stack *playerlib.UnitStack, others []*playerlib.UnitStack, water bool) bool {
    there := 0
    riders := boatRiders(stack)
    seats := shipSeats(stack)
    for _, other := range others {
        if other == nil || other == stack {
            continue
        }
        there += other.Size()
        riders += boatRiders(other)
        seats += shipSeats(other)
    }
    if there == 0 {
        return true
    }
    moving := len(stack.ActiveUnits())
    if there + moving > data.MaxUnitsInStack {
        return false
    }
    if water {
        // Eval_Move_Path: riders there and riders coming against the seats of both
        return riders <= seats
    }
    return true
}

// every stack of the owner of a stack on its square
func (model *GameModel) stacksBeside(other *playerlib.UnitStack, x int, y int) []*playerlib.UnitStack {
    for _, player := range model.Players {
        all := player.FindAllStacks(x, y, other.Plane())
        if slices.Contains(all, other) {
            return all
        }
    }
    return []*playerlib.UnitStack{other}
}

// the cost of a step as a fraction of moves and whether it can be made, from classicStepCost
func (model *GameModel) classicTerrainCost(pathStack playerlib.PathStack, destX int, destY int, mapUse *maplib.Map, tileTo maplib.FullTile, hasCity bool, getStack func(int, int) (playerlib.PathStack, bool)) (fraction.Fraction, bool) {
    stack, ok := pathStack.(*playerlib.UnitStack)
    if !ok {
        return fraction.FromInt(1), true
    }

    // Move_Path_Find: the top and bottom rows are never reached (a map of a row or two, as in
    // tests, has no poles)
    if mapUse.Height() > 2 && (destY <= 0 || destY >= mapUse.Height() - 1) {
        return fraction.Zero(), false
    }

    if found, ok := getStack(destX, destY); ok {
        if other, ok := found.(*playerlib.UnitStack); ok && !classicRoomAmong(stack, model.stacksBeside(other, destX, destY), !tileTo.Tile.IsLand()) {
            return fraction.Zero(), false
        }
    }

    road := hasCity
    enchanted := false
    if value, ok := tileTo.Extras[maplib.ExtraKindRoad]; ok {
        road = true
        enchanted = value.(*maplib.ExtraRoad).Enchanted
    }

    // ships sail at a move a square
    if !tileTo.Tile.IsLand() && stack.HasSailingUnits(true) {
        return fraction.FromInt(1), true
    }
    // land walkers boarding a ship
    if !tileTo.Tile.IsLand() && stack.AnyLandWalkers() && !stackModes(stack).Flying {
        return fraction.FromInt(1), true
    }

    cost := classicStepCost(stack, mapUse, tileTo, road, enchanted)
    if cost == noEntry {
        return fraction.Zero(), false
    }
    return fraction.Make(cost, 2), true
}

// Eval_Move_Path: a go-to of the human stops before a lair, an enemy stack or a city not its own
func (game *Game) classicGoToStops(player *playerlib.Player, stack *playerlib.UnitStack, entityInfo playerlib.CityStackInfo, mapUse *maplib.Map, x int, y int) bool {
    x = mapUse.WrapX(x)
    if encounter := mapUse.GetEncounter(x, y); encounter != nil {
        return true
    }
    if entityInfo.ContainsEnemy(x, y, stack.Plane(), player) {
        return true
    }
    if city := entityInfo.FindCity(x, y, stack.Plane()); city != nil && player.FindCity(x, y, stack.Plane()) == nil {
        return true
    }
    return false
}

// Turns_To_Build_Road (Terrain.c): the turns of one builder
func classicRoadTurns(tile maplib.FullTile) int {
    switch tile.Tile.TerrainType() {
        case terrain.Grass: return 3
        case terrain.Desert: return 4
        case terrain.Forest, terrain.Hill, terrain.Tundra: return 6
        case terrain.Mountain, terrain.Swamp, terrain.Volcano: return 8
        case terrain.SorceryNode: return 4
        case terrain.NatureNode, terrain.ChaosNode, terrain.River: return 5
    }
    return 3
}
