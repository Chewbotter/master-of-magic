package ai

// The orders of the overland AI of Chewbot for the units on one landmass. Facts from the ReMoM
// project's reconstruction (MoM/src/AIMOVE.c: AI_Stacks_Init_Build_Target_Order,
// AI_Stacks_Target_Nearest_Hostile_Stack, AI_Stacks_Move_Out_NonMilitary_Garrisoned,
// AI_Stacks_Survey_Expedition_Forces(_Stack), AI_Stacks_Do_Meld, AI_Stacks_Do_Settle,
// AI_Stacks_Do_Purify, AI_Stacks_Do_RoadBuild, AI_Build_Target_List, AI_Add_Target,
// AI_Stacks_Assign_Target, AI_Stacks_Roamers_Target_Or_Deploy, AI_Stacks_Order_To_War_Landmass,
// AI_Stacks_Relocate_Roamers, AI_Stacks_Setup_Ferry, AI_Stacks_Stage_Expedition_Forces,
// AI_Stacks_Garrison_Sites, AI_Tower_Target_Worthwhile, AI_Find_Tower_To_Settle_Elsewhere,
// AI_Stacks_Reorder_Settle_Elsewhere, Adjacent_Free_Square, Next_Nearest_Ferry_Square,
// Adjacent_Land_Square; MoM/src/CITYCALC.c: Compute_Base_Values_For_Map_Square). The code is ours.
// The rules in words: docs/mod/ai-overland.md.

import (
    "image"
    "slices"
    "sync"

    "github.com/kazzmir/master-of-magic/game/magic/maplib"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

// numbers of the original
const (
    chewMaxStack = data.MaxUnitsInStack
    // an army of this many that found nothing to attack goes to the coast toward the war landmass
    chewLargeStack = 8
    // a garrison of more than this gives its best units to an expedition
    chewGarrisonKeeps = 5
    // before this turn the fortress gives its units too
    chewFortressGivesUntil = 100
    // garrisons: 9 in the fortress, 2 and a unit for 3 citizens (4 where the wizard feels safe or
    // the race is strong), a node 8 (4 where the wizard does not feel safe)
    chewGarrisonBase = 2
    chewGarrisonPerCitizens = 3
    chewGarrisonPerCitizensSafe = 4
    chewNodeGarrison = 8
    chewNodeGarrisonUnsafe = 4
    // Do_Settle: after this turn a settler settles one time in 5 (see quirkSettleOpponentCheck)
    chewSettleFreelyUntil = 100
    // city values and target values
    chewFortressValue = 500
    chewWarFactor = 10
    chewLairValue = 50
    chewNodeOwnedValue = 50
    chewNodeGuardedValue = 25
    chewNodePowerValue = 10
    chewTowerUnownedValue = 150
    chewTowerHostileValue = 10
    chewTowerOwnValue = 1
    // AI_Tower_Target_Worthwhile: a landmass larger than this is worth it
    chewTowerLandmassSize = 10
)

// the kinds of the stacks of the wizard on a landmass (AISTK_...)
const (
    chewStackUnknown = iota
    chewStackRoamer
    chewStackGarrison
    chewStackFortress
)

// the units of the wizard on one square; an empty slot is a unit that is busy (on its way, building
// a road, purifying) or was given an order in this turn
type chewAIStack struct {
    X int
    Y int
    Slots []units.StackUnit
    Type int
}

func (stack *chewAIStack) Count() int {
    return len(stack.Slots)
}

// a unit an expedition may take from its stack
type chewDraft struct {
    Stack *chewAIStack
    Slot int
    Unit units.StackUnit
    Value int
}

type chewTarget struct {
    X int
    Y int
    Strength int
    Value int
}

// the work on one landmass of one plane in one turn
type chewPass struct {
    AI *ChewbotAI
    Overland *chewOverland
    World *chewWorld
    WP int
    Landmass int
    Stacks []*chewAIStack
    // units at the stage point, and units that have it as their destination
    Staged int
    Enroute int
    Drafted []chewDraft
    Targets []chewTarget
    Turn *chewTurn
}

func (pass *chewPass) plane() data.Plane {
    return chewPlaneOf(pass.WP)
}

func (pass *chewPass) stage() (int, int) {
    return pass.Overland.StageX[pass.WP][pass.Landmass], pass.Overland.StageY[pass.WP][pass.Landmass]
}

func (pass *chewPass) landType() int {
    return pass.Overland.LandType[pass.WP][pass.Landmass]
}

// AI_Stacks_Order_Attack_Target_Or_Goto_Destination: the unit goes to a square, to attack when an
// enemy stack stands there off a site; its slot is emptied
func (pass *chewPass) order(stack *chewAIStack, slot int, x int, y int) {
    if stack == nil || slot < 0 || slot >= len(stack.Slots) || stack.Slots[slot] == nil {
        return
    }
    unit := stack.Slots[slot]
    pass.Turn.order(unit, x, y, pass.plane())
    stack.Slots[slot] = nil
}

// AI_Stacks_Init_Build_Target_Order: the units of the wizard on the landmass by square; a square
// with something on the evaluation map (a site of any kind) holds a garrison, any other a roamer;
// a roamer of one unit is unknown; a roamer that was attacking goes on to the nearest hostile stack
func (turn *chewTurn) initPass(wp int, landmass int) *chewPass {
    pass := &chewPass{
        AI: turn.AI,
        Overland: turn.Overland,
        World: turn.World,
        WP: wp,
        Landmass: landmass,
        Turn: turn,
    }
    world := turn.World
    stageX, stageY := pass.stage()
    fortress := chewFortress(world.Self)

    bySquare := make(map[image.Point]*chewAIStack)
    for _, unitStack := range world.Self.Stacks {
        if chewPlaneIndex(unitStack.Plane()) != wp || turn.Skip[unitStack] {
            continue
        }
        x, y := unitStack.X(), unitStack.Y()
        if x == stageX && y == stageY {
            pass.Staged += len(unitStack.Units())
        }
        if world.LandmassAt(x, y, wp) != landmass {
            continue
        }

        stack, ok := bySquare[image.Pt(x, y)]
        if !ok {
            stack = &chewAIStack{X: x, Y: y, Type: chewStackRoamer}
            if world.EvalAt(x, y, wp) != 0 {
                stack.Type = chewStackGarrison
                if fortress != nil && fortress.X == x && fortress.Y == y && chewPlaneIndex(fortress.Plane) == wp {
                    stack.Type = chewStackFortress
                }
            }
            bySquare[image.Pt(x, y)] = stack
            pass.Stacks = append(pass.Stacks, stack)
        }
        for _, unit := range unitStack.Units() {
            if turn.busy(unit) {
                stack.Slots = append(stack.Slots, nil)
            } else {
                stack.Slots = append(stack.Slots, unit)
            }
            if order, has := turn.Overland.Orders[chewKey(unit)]; has && order.X == stageX && order.Y == stageY {
                pass.Enroute += 1
            }
        }
    }

    turn.Why = "chase"
    for _, stack := range pass.Stacks {
        if stack.Type != chewStackRoamer {
            continue
        }
        first := stack.Slots[0]
        attacking := false
        if first != nil {
            if order, has := turn.Overland.Orders[chewKey(first)]; has && order.Kind == chewOrderAttack {
                attacking = true
            }
        }
        if !attacking {
            // the original tests count + 1 < 3
            if stack.Count() + 1 < 3 {
                stack.Type = chewStackUnknown
            }
            continue
        }
        if x, y, ok := pass.nearestHostileStack(stack); ok {
            for slot := range stack.Slots {
                pass.order(stack, slot, x, y)
            }
        } else {
            for _, unit := range stack.Slots {
                if unit != nil {
                    delete(turn.Overland.Orders, chewKey(unit))
                }
            }
            stack.Type = chewStackUnknown
        }
    }
    return pass
}

// AI_Stacks_Target_Nearest_Hostile_Stack: the nearest square of the landmass with hostile units
// that is no site
func (pass *chewPass) nearestHostileStack(stack *chewAIStack) (int, int, bool) {
    best := 1000
    var found image.Point
    for _, square := range pass.World.LandSquares[pass.WP][pass.Landmass] {
        value := pass.World.EvalAt(square.X, square.Y, pass.WP)
        if value & chewEvalStrength == 0 || value & chewEvalNonHostile != 0 || value & chewEvalSite != 0 {
            continue
        }
        distance := pass.World.Distance(stack.X, stack.Y, square.X, square.Y)
        if distance < best {
            best = distance
            found = square
        }
    }
    return found.X, found.Y, best < 1000
}

func chewIsSettler(unit units.StackUnit) bool {
    return unit.HasAbility(data.AbilityCreateOutpost)
}

func chewBuildsRoads(unit units.StackUnit) bool {
    return unit.HasAbility(data.AbilityConstruction)
}

func chewIsMelder(unit units.StackUnit) bool {
    return unit.HasAbility(data.AbilityMeld)
}

func chewIsBoat(unit units.StackUnit) bool {
    return unit.HasAbility(data.AbilityTransport)
}

// Unit_Has_AirTravel or Unit_Has_WaterTravel
func chewSeafaring(unit units.StackUnit) bool {
    return unit.IsFlying() || unit.IsSwimmer() || unit.IsSailing()
}

// Adjacent_Free_Square: a land square around a square with no unit, intact lair or city on it,
// the first by rows
func (world *chewWorld) adjacentFreeSquare(x int, y int, wp int) (int, int, bool) {
    taken := func(tx int, ty int) bool {
        tx = world.WrapX(tx)
        plane := chewPlaneOf(wp)
        for _, player := range world.Players {
            if stack := player.FindStack(tx, ty, plane); stack != nil && len(stack.Units()) > 0 {
                return true
            }
        }
        if world.Maps[wp].GetEncounter(tx, ty) != nil {
            return true
        }
        _, isCity := world.CityAt(tx, ty, wp)
        return isCity
    }
    for dy := -1; dy <= 1; dy++ {
        for dx := -1; dx <= 1; dx++ {
            if dx == 0 && dy == 0 {
                continue
            }
            tx, ty := x + dx, y + dy
            if !world.inside(ty) || world.LandmassAt(tx, ty, wp) == 0 || taken(tx, ty) {
                continue
            }
            return world.WrapX(tx), ty, true
        }
    }
    return 0, 0, false
}

// AI_Stacks_Move_Out_NonMilitary_Garrisoned: settlers and engineers in a garrison step out to a
// free square next to it; so does a spirit that is not on a node
func (pass *chewPass) moveOutNonMilitaryGarrisoned() {
    for _, stack := range pass.Stacks {
        if stack.Type != chewStackGarrison && stack.Type != chewStackFortress {
            continue
        }
        for slot, unit := range stack.Slots {
            if unit == nil {
                continue
            }
            out := chewIsSettler(unit) || chewBuildsRoads(unit)
            if !out && chewIsMelder(unit) {
                out = !pass.World.Maps[pass.WP].HasMagicNode(stack.X, stack.Y)
            }
            if !out {
                continue
            }
            if x, y, ok := pass.World.adjacentFreeSquare(stack.X, stack.Y, pass.WP); ok {
                pass.order(stack, slot, x, y)
            }
        }
    }
}

// AI_Stacks_Survey_Expedition_Forces: the units an expedition may take, the 9 strongest: all
// fighting units of unknown stacks and of roamers that are not busy, the best beyond 5 of a
// garrison, and beyond 5 of the fortress before turn 100
func (pass *chewPass) surveyExpeditionForces() {
    pass.Drafted = nil
    for _, stack := range pass.Stacks {
        switch stack.Type {
            case chewStackUnknown:
                pass.surveyStack(stack, stack.Count())
            case chewStackRoamer:
                if stack.Slots[0] != nil {
                    pass.surveyStack(stack, stack.Count())
                }
            case chewStackGarrison:
                if stack.Count() > chewGarrisonKeeps {
                    pass.surveyStack(stack, stack.Count() - chewGarrisonKeeps)
                }
            case chewStackFortress:
                if pass.World.Turn < chewFortressGivesUntil && stack.Count() > chewGarrisonKeeps {
                    pass.surveyStack(stack, stack.Count() - chewGarrisonKeeps)
                }
        }
    }
    slices.SortStableFunc(pass.Drafted, func(a chewDraft, b chewDraft) int {
        return b.Value - a.Value
    })
}

func (pass *chewPass) surveyStack(stack *chewAIStack, excess int) {
    var candidates []chewDraft
    for slot, unit := range stack.Slots {
        if unit == nil || chewBuildsRoads(unit) || chewIsSettler(unit) || chewIsMelder(unit) || chewIsBoat(unit) {
            continue
        }
        candidates = append(candidates, chewDraft{Stack: stack, Slot: slot, Unit: unit, Value: chewUnitStrength(unit) / 10})
    }
    slices.SortStableFunc(candidates, func(a chewDraft, b chewDraft) int {
        return b.Value - a.Value
    })
    if excess < len(candidates) {
        candidates = candidates[:excess]
    }

    for _, candidate := range candidates {
        if len(pass.Drafted) < chewMaxStack {
            pass.Drafted = append(pass.Drafted, candidate)
            continue
        }
        // the list is full: the candidate takes the place of the weakest when it is stronger
        weakest := 0
        for index, drafted := range pass.Drafted {
            if drafted.Value <= pass.Drafted[weakest].Value {
                weakest = index
            }
        }
        if candidate.Value > pass.Drafted[weakest].Value {
            pass.Drafted[weakest] = candidate
        }
    }
}

// the player that holds a node, nil when nobody does
func (world *chewWorld) nodeOwner(x int, y int, wp int) *playerlib.Player {
    node := world.Maps[wp].GetMagicNode(x, y)
    if node == nil || node.MeldingWizard == nil {
        return nil
    }
    for _, player := range world.Players {
        if node.MeldingWizard == player || node.MeldingWizard.GetBanner() == player.GetBanner() {
            return player
        }
    }
    return nil
}

// the power of a node: the squares of its aura
func (world *chewWorld) nodePower(x int, y int, wp int) int {
    node := world.Maps[wp].GetMagicNode(x, y)
    if node == nil {
        return 0
    }
    return len(node.Zone)
}

// AI_Stacks_Do_Meld: a spirit goes to the nearest node of the plane that is not the wizard's, has
// nothing on it, and lies on a landmass of the wizard's own or one it waits to leave, or holds a
// garrison of the wizard; on it, it melds
func (pass *chewPass) doMeld() {
    world := pass.World
    self := world.Self
    for _, stack := range pass.Stacks {
        for slot, unit := range stack.Slots {
            if unit == nil || !chewIsMelder(unit) {
                continue
            }
            best := 1000
            var target image.Point
            for _, node := range chewNodePoints(world.Maps[pass.WP]) {
                if world.nodeOwner(node.X, node.Y, pass.WP) == self || world.EvalAt(node.X, node.Y, pass.WP) != chewEvalSite {
                    continue
                }
                nodeLandmass := world.LandmassAt(node.X, node.Y, pass.WP)
                kind := pass.Overland.LandType[pass.WP][nodeLandmass]
                garrisoned := false
                if own := self.FindStack(node.X, node.Y, pass.plane()); own != nil && len(own.Units()) > 0 {
                    garrisoned = true
                }
                if kind != chewLandOwn && kind < chewLandLeaveable && !garrisoned {
                    continue
                }
                distance := world.Distance(node.X, node.Y, stack.X, stack.Y)
                if distance < best {
                    best = distance
                    target = node
                }
            }
            if best == 1000 {
                continue
            }
            if best == 0 {
                pass.Turn.Melds = append(pass.Turn.Melds, unit)
                stack.Slots[slot] = nil
            } else {
                pass.order(stack, slot, target.X, target.Y)
            }
        }
    }
}

// Map_Square_Survey: a city may be built here (the game's own rule, and not on a tower)
func (world *chewWorld) canSettle(x int, y int, wp int) bool {
    if world.Maps[wp].HasOpenTower(x, y) {
        return false
    }
    return world.Services.IsSettlableLocation(x, y, chewPlaneOf(wp))
}

// the values of city sites of one map, the same for every wizard
type chewSiteTable struct {
    Map *maplib.Map
    Cities int
    Turn int
    Values map[image.Point]int
}

var chewSiteTables [2]chewSiteTable
var chewSiteLock sync.Mutex

// the value of a site, kept from turn to turn while the plane has as many cities (a city founded or
// lost changes what the squares around it are worth) and for at most chewSiteKeepTurns turns
func (overland *chewOverland) siteValue(world *chewWorld, x int, y int, wp int) int {
    cities := 0
    for _, city := range world.Cities {
        if chewPlaneIndex(city.City.Plane) == wp {
            cities += 1
        }
    }
    chewSiteLock.Lock()
    defer chewSiteLock.Unlock()
    table := &chewSiteTables[wp]
    if table.Values == nil || table.Map != world.Maps[wp] || table.Cities != cities || world.Turn - table.Turn >= chewSiteKeepTurns || world.Turn < table.Turn {
        *table = chewSiteTable{Map: world.Maps[wp], Cities: cities, Turn: world.Turn, Values: make(map[image.Point]int)}
    }
    point := image.Pt(x, y)
    if value, ok := table.Values[point]; ok {
        return value
    }
    value := world.siteValue(x, y, wp)
    table.Values[point] = value
    return value
}

// how long the values of sites are kept at most
const chewSiteKeepTurns = 20

// Compute_Base_Values_For_Map_Square, weighed as AI_Stacks_Do_Settle does: the maximum population
// times 10, the production bonus, the gold bonus of the square, the reduction of the cost of
// units, gold of minerals times 3, magic of minerals times 5, nightshade 10, mithril 20,
// adamantium 50, the coast 50
func (world *chewWorld) siteValue(x int, y int, wp int) int {
    mapObject := world.Maps[wp]
    population := world.Services.ComputeMaximumPopulation(x, y, chewPlaneOf(wp))
    production, reduction, gold, magic := 0, 0, 0, 0
    nightshade, mithril, adamantium := 0, 0, 0
    for _, tile := range mapObject.GetCatchmentArea(x, y) {
        production += tile.ProductionBonus(false)
        bonus := tile.GetBonus()
        reduction += bonus.UnitReductionBonus()
        gold += bonus.GoldBonus()
        magic += bonus.PowerBonus()
        switch bonus {
            case data.BonusNightshade: nightshade = 1
            case data.BonusMithrilOre: mithril = 1
            case data.BonusAdamantiumOre: adamantium = 1
        }
    }
    reduction = min(reduction, 50)
    center := mapObject.GetTile(x, y)
    shore := 0
    if mapObject.OnShore(x, y) {
        shore = 1
    }
    return population * 10 + production + center.GoldBonus(mapObject) + reduction + gold * 3 + magic * 5 + nightshade * 10 + mithril * 20 + adamantium * 50 + shore * 50
}

// AI_Stacks_Do_Settle: no settler moves while a garrison of the landmass is empty (never, the
// original counts stacks). A settler settles where it stands when it may and a roll of 1..5 is not
// above the objective's chance (Expansionist 4, Theurgist 3, Militarist and Pragmatist 2,
// Perfectionist 1), before turn 100 or else one time in 5 (quirkSettleOpponentCheck); else it goes
// to a tower of the wizard's own to cross to the other plane, else to the best square of the
// landmass by value over distance, else by sea to the landmass for settlers
func (pass *chewPass) doSettle() {
    world := pass.World
    chance := 1
    switch pass.AI.objective(world.Self) {
        case chewExpansionist: chance = 4
        case chewMilitarist, chewPragmatist: chance = 2
        case chewTheurgist: chance = 3
    }

    for _, stack := range pass.Stacks {
        for slot, unit := range stack.Slots {
            if unit == nil || !chewIsSettler(unit) {
                continue
            }
            // the original always finds an opponent near (see the quirk); meant: an enemy within 3
            safe := false
            if !quirkSettleOpponentCheck {
                safe = !world.enemyUnitNear(stack.X, stack.Y, pass.WP, 3, -1)
            }
            if (safe || world.Turn < chewSettleFreelyUntil || chewRandom(5) == 1) && world.canSettle(stack.X, stack.Y, pass.WP) && chewRandom(5) <= chance {
                pass.Turn.Settles = append(pass.Turn.Settles, unit)
                stack.Slots[slot] = nil
                continue
            }
            if x, y, ok := pass.towerToSettleElsewhere(stack); ok {
                pass.order(stack, slot, x, y)
                continue
            }

            best := -1
            var target image.Point
            for _, square := range world.LandSquares[pass.WP][pass.Landmass] {
                if !world.canSettle(square.X, square.Y, pass.WP) || world.EvalAt(square.X, square.Y, pass.WP) & chewEvalStrength != 0 {
                    continue
                }
                distance := chewRange(stack.X, stack.Y, square.X, square.Y, world.Width)
                if distance == 0 {
                    continue
                }
                value := pass.Overland.siteValue(world, square.X, square.Y, pass.WP) / distance
                if value > best {
                    best = value
                    target = square
                }
            }
            if best < 0 {
                pass.settleElsewhere(stack, slot)
            } else {
                pass.order(stack, slot, target.X, target.Y)
            }
        }
    }
}

// Range of the original: the larger distance and half the smaller
func chewRange(x1 int, y1 int, x2 int, y2 int, width int) int {
    across := x2 - x1
    if across < 0 {
        across = -across
    }
    if width - across < across {
        across = width - across
    }
    along := y2 - y1
    if along < 0 {
        along = -along
    }
    if across > along {
        return across + along / 2
    }
    return along + across / 2
}

// no Planar Seal anywhere
func (world *chewWorld) planesOpen() bool {
    for _, player := range world.Players {
        if player.GlobalEnchantments != nil && player.GlobalEnchantments.Contains(data.EnchantmentPlanarSeal) {
            return false
        }
    }
    return true
}

// the owner of an open tower: the player whose units stand on it (the game keeps no owner)
func (world *chewWorld) towerOwner(x int, y int, wp int) *playerlib.Player {
    for _, player := range world.Players {
        if stack := player.FindStack(x, y, chewPlaneOf(wp)); stack != nil && len(stack.Units()) > 0 {
            return player
        }
    }
    return nil
}

// AI_Find_Tower_To_Settle_Elsewhere: a settler on the home plane goes to the nearest tower of the
// wizard's own on its landmass, unless the landmass on the other side is contested or left alone
func (pass *chewPass) towerToSettleElsewhere(stack *chewAIStack) (int, int, bool) {
    world := pass.World
    fortress := chewFortress(world.Self)
    if !world.planesOpen() || fortress == nil || chewPlaneIndex(fortress.Plane) != pass.WP {
        return 0, 0, false
    }
    mapObject := world.Maps[pass.WP]
    if mapObject.HasOpenTower(stack.X, stack.Y) {
        return 0, 0, false
    }
    best := 1000
    var found image.Point
    for _, tower := range mapObject.GetOpenTowerLocations() {
        if world.towerOwner(tower.X, tower.Y, pass.WP) != world.Self || world.LandmassAt(tower.X, tower.Y, pass.WP) != pass.Landmass {
            continue
        }
        distance := world.Distance(stack.X, stack.Y, tower.X, tower.Y)
        if distance < best {
            best = distance
            found = tower
        }
    }
    if best == 1000 {
        return 0, 0, false
    }
    other := 1 - pass.WP
    kind := pass.Overland.LandType[other][world.LandmassAt(found.X, found.Y, other)]
    if kind == chewLandContested || kind == chewLandIgnored {
        return 0, 0, false
    }
    return found.X, found.Y, true
}

// AI_Stacks_Reorder_Settle_Elsewhere: a settler that finds no square on its landmass goes to the
// landmass for settlers: straight there when it can fly or swim; else, away from the sea, to a free
// square at the nearest coast; at the sea, onto a ship next to it, or it asks for a ship there
func (pass *chewPass) settleElsewhere(stack *chewAIStack, slot int) {
    world := pass.World
    unit := stack.Slots[slot]
    if pass.Overland.SettlerLandmass[pass.WP] == 0 {
        return
    }
    if chewSeafaring(unit) {
        pass.order(stack, slot, pass.Overland.SettlerX[pass.WP], pass.Overland.SettlerY[pass.WP])
        return
    }

    var sea image.Point
    atSea := false
    for dy := -1; dy <= 1; dy++ {
        for dx := -1; dx <= 1; dx++ {
            x, y := world.WrapX(stack.X + dx), stack.Y + dy
            if y <= 0 || !world.inside(y) {
                continue
            }
            if chewIsSea(world.Maps[pass.WP].GetTile(x, y)) {
                atSea = true
                sea = image.Pt(x, y)
            }
        }
    }

    if !atSea {
        dock := world.nextNearestFerrySquare(stack.X, stack.Y, pass.WP)
        if x, y, ok := world.adjacentLandSquare(dock.X, dock.Y, pass.WP); ok {
            pass.order(stack, slot, x, y)
        }
        return
    }

    for _, boat := range pass.Turn.boatStacks(pass.WP) {
        if chewAbs(world.WrapX(boat.X() - stack.X + world.Width / 2) - world.Width / 2) < 2 && chewAbs(boat.Y() - stack.Y) < 2 {
            pass.order(stack, slot, boat.X(), boat.Y())
            return
        }
    }
    if pass.Turn.addFerry(sea.X, sea.Y, pass.WP) {
        stack.Slots[slot] = nil
    }
}

func chewAbs(value int) int {
    if value < 0 {
        return -value
    }
    return value
}

// Next_Nearest_Ferry_Square: the coast square of the landmass nearest to a square that has a free
// land square around it (itself too); the square itself when there is none
func (world *chewWorld) nextNearestFerrySquare(x int, y int, wp int) image.Point {
    landmass := world.LandmassAt(x, y, wp)
    best := 2000
    found := image.Pt(x, y)
    for _, dock := range world.DockSquares[wp][landmass] {
        free := false
        for dy := -1; dy <= 1 && !free; dy++ {
            for dx := -1; dx <= 1 && !free; dx++ {
                if world.LandmassAt(dock.X + dx, dock.Y + dy, wp) != 0 && world.EvalAt(dock.X + dx, dock.Y + dy, wp) == 0 {
                    free = true
                }
            }
        }
        if !free {
            continue
        }
        distance := world.Distance(dock.X, dock.Y, x, y)
        if distance < best {
            best = distance
            found = dock
        }
    }
    return found
}

// Adjacent_Land_Square: the first free land square around a square, by rows, the square too
func (world *chewWorld) adjacentLandSquare(x int, y int, wp int) (int, int, bool) {
    for dy := -1; dy <= 1; dy++ {
        ty := y + dy
        if ty <= 0 || !world.inside(ty) {
            continue
        }
        for dx := -1; dx <= 1; dx++ {
            tx := world.WrapX(x + dx)
            if world.LandmassAt(tx, ty, wp) != 0 && world.EvalAt(tx, ty, wp) == 0 {
                return tx, ty, true
            }
        }
    }
    return 0, 0, false
}

// AI_Stacks_Do_Purify: when a corrupted square of the landmass with no enemy on it lies within 2
// of a site with nothing on it, the priests of unknown stacks go there and purify it
func (pass *chewPass) doPurify() {
    world := pass.World
    mapObject := world.Maps[pass.WP]
    found := false
    var target image.Point
    for _, square := range world.LandSquares[pass.WP][pass.Landmass] {
        if found {
            break
        }
        if !mapObject.HasCorruption(square.X, square.Y) || world.EvalAt(square.X, square.Y, pass.WP) & chewEvalStrength != 0 {
            continue
        }
        for dy := -2; dy <= 2; dy++ {
            for dx := -2; dx <= 2; dx++ {
                if world.inside(square.Y + dy) && world.EvalAt(square.X + dx, square.Y + dy, pass.WP) == chewEvalSite {
                    found = true
                    target = square
                }
            }
        }
    }
    if !found {
        return
    }

    for _, stack := range pass.Stacks {
        if stack.Type != chewStackUnknown {
            continue
        }
        for slot, unit := range stack.Slots {
            if unit == nil || !unit.HasAbility(data.AbilityPurify) {
                continue
            }
            if mapObject.HasCorruption(stack.X, stack.Y) {
                pass.Turn.Purifies = append(pass.Turn.Purifies, unit)
                stack.Slots[slot] = nil
            } else {
                pass.order(stack, slot, target.X, target.Y)
            }
        }
    }
}

// AI_Stacks_Do_RoadBuild: an engineer goes to the nearest city of its landmass, anyone's; next to
// it, it builds a road from where it stands to the nearest other city of that landmass the city has
// no road to (the original also looks at cities of other landmasses for engineers that can not
// fly, which they can not reach). The game builds roads only over land the wizard has seen, so a
// city the road can not reach that way is passed over; the original sees all of the map
func (pass *chewPass) doRoadBuild() {
    world := pass.World
    mapObject := world.Maps[pass.WP]
    cityPoints := make(map[image.Point]bool)
    for _, city := range world.Cities {
        if chewPlaneIndex(city.City.Plane) == pass.WP {
            cityPoints[image.Pt(city.City.X, city.City.Y)] = true
        }
    }

    for _, stack := range pass.Stacks {
        for slot, unit := range stack.Slots {
            if unit == nil || !chewBuildsRoads(unit) {
                continue
            }
            best := 1000
            var nearest *chewCity
            for index := range world.Cities {
                city := &world.Cities[index]
                if chewPlaneIndex(city.City.Plane) != pass.WP || world.LandmassAt(city.City.X, city.City.Y, pass.WP) != pass.Landmass {
                    continue
                }
                distance := world.Distance(stack.X, stack.Y, city.City.X, city.City.Y)
                if distance < best {
                    best = distance
                    nearest = city
                }
            }
            if nearest == nil {
                continue
            }
            if best > 1 {
                pass.order(stack, slot, nearest.City.X, nearest.City.Y)
                continue
            }

            best = 1000
            var roadTo *chewCity
            for index := range world.Cities {
                city := &world.Cities[index]
                if city == nearest || chewPlaneIndex(city.City.Plane) != pass.WP || world.LandmassAt(city.City.X, city.City.Y, pass.WP) != pass.Landmass {
                    continue
                }
                if citiesRoadConnected(mapObject, cityPoints, nearest.City.X, nearest.City.Y, city.City.X, city.City.Y) {
                    continue
                }
                distance := world.Distance(nearest.City.X, nearest.City.Y, city.City.X, city.City.Y)
                if distance >= best || !world.roadReachable(stack.X, stack.Y, city.City.X, city.City.Y, pass.WP) {
                    continue
                }
                if distance < best {
                    best = distance
                    roadTo = city
                }
            }
            if roadTo != nil {
                pass.Turn.Roads[chewKey(unit)] = image.Pt(roadTo.City.X, roadTo.City.Y)
                stack.Slots[slot] = nil
            }
        }
    }
}

// the game's road can be built from a square to another: over land the wizard has explored, with
// no unit, city or place with guardians of anyone else on the way (the target may have them);
// game/road.go FindRoadPath
func (world *chewWorld) roadReachable(fromX int, fromY int, toX int, toY int, wp int) bool {
    plane := chewPlaneOf(wp)
    fog := world.Self.GetFog(plane)
    mapObject := world.Maps[wp]
    blocked := func(x int, y int) bool {
        if x == toX && y == toY {
            return false
        }
        if mapObject.GetEncounter(x, y) != nil {
            return true
        }
        for _, player := range world.Players {
            if player == world.Self {
                continue
            }
            if stack := player.FindStack(x, y, plane); stack != nil && len(stack.Units()) > 0 {
                return true
            }
            if player.FindCity(x, y, plane) != nil {
                return true
            }
        }
        return false
    }
    seen := make(map[image.Point]bool)
    search := []image.Point{image.Pt(world.WrapX(fromX), fromY)}
    seen[search[0]] = true
    for len(search) > 0 {
        point := search[0]
        search = search[1:]
        if point.X == toX && point.Y == toY {
            return true
        }
        for dy := -1; dy <= 1; dy++ {
            for dx := -1; dx <= 1; dx++ {
                next := image.Pt(world.WrapX(point.X + dx), point.Y + dy)
                if seen[next] || !world.inside(next.Y) {
                    continue
                }
                seen[next] = true
                if world.LandmassAt(next.X, next.Y, wp) == 0 || fog.GetFog(next.X, next.Y) == data.FogTypeUnexplored || blocked(next.X, next.Y) {
                    continue
                }
                search = append(search, next)
            }
        }
    }
    return false
}

func (pass *chewPass) addTarget(x int, y int, strength int, value int) {
    pass.Targets = append(pass.Targets, chewTarget{X: x, Y: y, Strength: strength, Value: value})
}

// AI_Tower_Target_Worthwhile: the landmass on the other side of the tower is not the wizard's own
// nor left alone and holds a neutral city or one of a wizard it is hostile to (2 or more); or it
// is no landmass without the wizard's cities and has more than 10 squares
func (pass *chewPass) towerWorthwhile(x int, y int) bool {
    world := pass.World
    other := 1 - pass.WP
    landmass := world.LandmassAt(x, y, other)
    kind := pass.Overland.LandType[other][landmass]
    if kind != chewLandOwn && kind != chewLandIgnored {
        for _, city := range world.Cities {
            if chewPlaneIndex(city.City.Plane) != other || world.LandmassAt(city.City.X, city.City.Y, other) != landmass || city.Owner == world.Self {
                continue
            }
            if city.Owner.IsNeutral() || pass.Overland.hostilityOf(city.Owner) >= 2 {
                return true
            }
        }
    }
    if kind != chewLandNoOwnCity && landmass != 0 && len(world.LandSquares[other][landmass]) > chewTowerLandmassSize {
        return true
    }
    return false
}

// AI_Build_Target_List: what the armies of the landmass may attack, with its strength and value
func (pass *chewPass) buildTargetList() {
    world := pass.World
    self := world.Self
    overland := pass.Overland
    pass.Targets = nil

    atWar := func(other *playerlib.Player) bool {
        return !other.IsNeutral() && chewTreaty(self, other) == data.TreatyWar
    }
    strengthAt := func(x int, y int, owner *playerlib.Player) int {
        strength := world.EvalAt(x, y, pass.WP) & chewEvalStrength
        if owner != nil && owner.IsHuman() {
            // the human's places count a quarter stronger
            strength = strength * 5 / 4
        }
        return strength
    }

    // fortresses of wizards it is hostile to (3 or more)
    for _, other := range world.Players {
        if other == self || other.IsNeutral() {
            continue
        }
        fortress := chewFortress(other)
        if fortress == nil || chewPlaneIndex(fortress.Plane) != pass.WP {
            continue
        }
        base := 1
        if atWar(other) {
            base = chewWarFactor
        }
        if world.LandmassAt(fortress.X, fortress.Y, pass.WP) == pass.Landmass && overland.hostilityOf(other) >= 3 {
            pass.addTarget(fortress.X, fortress.Y, strengthAt(fortress.X, fortress.Y, other), base * chewFortressValue)
        }
    }

    // cities: neutral ones of the landmass, and the ones of wizards it is hostile to (3 or more)
    for _, city := range world.Cities {
        if chewPlaneIndex(city.City.Plane) != pass.WP || city.Owner == self {
            continue
        }
        base := 1
        if atWar(city.Owner) {
            base = chewWarFactor
        }
        if world.LandmassAt(city.City.X, city.City.Y, pass.WP) != pass.Landmass {
            continue
        }
        if !city.Owner.IsNeutral() && overland.hostilityOf(city.Owner) < 3 {
            continue
        }
        value := ((city.City.Citizens() + city.City.Buildings.Size()) * 10 + 100) * base
        pass.addTarget(city.City.X, city.City.Y, strengthAt(city.City.X, city.City.Y, city.Owner), value)
    }

    // hostile stacks off the sites: their full strength while the wizard's fortress is on the
    // landmass, else a third
    fortress := chewFortress(self)
    homeLand := fortress != nil && chewPlaneIndex(fortress.Plane) == pass.WP && world.LandmassAt(fortress.X, fortress.Y, pass.WP) == pass.Landmass
    for _, square := range world.LandSquares[pass.WP][pass.Landmass] {
        value := world.EvalAt(square.X, square.Y, pass.WP)
        if value == 0 || value & chewEvalNonHostile != 0 || value & chewEvalSite != 0 || value & chewEvalStrength == 0 {
            continue
        }
        strength := value & chewEvalStrength
        if homeLand {
            pass.addTarget(square.X, square.Y, strength, strength)
        } else {
            pass.addTarget(square.X, square.Y, strength, strength / 3)
        }
    }

    // nodes: of a wizard it is hostile to (2 or more), or nobody's with guardians
    mapObject := world.Maps[pass.WP]
    for _, node := range chewNodePoints(mapObject) {
        owner := world.nodeOwner(node.X, node.Y, pass.WP)
        if owner == self || (owner != nil && overland.hostilityOf(owner) < 2) {
            continue
        }
        if world.LandmassAt(node.X, node.Y, pass.WP) != pass.Landmass {
            continue
        }
        strength := world.EvalAt(node.X, node.Y, pass.WP) & chewEvalStrength
        power := world.nodePower(node.X, node.Y, pass.WP)
        if owner != nil {
            pass.addTarget(node.X, node.Y, strength, power * chewNodePowerValue + chewNodeOwnedValue)
        } else if strength != 0 {
            pass.addTarget(node.X, node.Y, strength, power * chewNodePowerValue + chewNodeGuardedValue)
        }
    }

    // lairs, nodes and towers with guardians
    for _, lair := range chewEncounterPoints(mapObject) {
        if world.LandmassAt(lair.X, lair.Y, pass.WP) != pass.Landmass {
            continue
        }
        pass.addTarget(lair.X, lair.Y, world.EvalAt(lair.X, lair.Y, pass.WP) & chewEvalStrength, chewLairValue)
    }

    // towers, on the plane of the wizard's fortress, where it is worth crossing
    if fortress != nil && chewPlaneIndex(fortress.Plane) == pass.WP {
        for _, tower := range chewTowerPoints(world, pass.WP) {
            if world.LandmassAt(tower.X, tower.Y, pass.WP) != pass.Landmass || !pass.towerWorthwhile(tower.X, tower.Y) {
                continue
            }
            owner := world.towerOwner(tower.X, tower.Y, pass.WP)
            strength := world.EvalAt(tower.X, tower.Y, pass.WP) & chewEvalStrength
            switch {
                case owner == self:
                    pass.addTarget(tower.X, tower.Y, 0, chewTowerOwnValue)
                case owner != nil && overland.hostilityOf(owner) < 2:
                case owner != nil && overland.hostilityOf(owner) >= 3:
                    pass.addTarget(tower.X, tower.Y, strength, chewTowerHostileValue)
                case owner == nil:
                    pass.addTarget(tower.X, tower.Y, strength, chewTowerUnownedValue)
            }
        }
    }

    if len(pass.Targets) == 0 {
        overland.LandType[pass.WP][pass.Landmass] = chewLandNoTargets
    }
}

// AI_Stacks_Assign_Target: the target of the highest value over 4 times the distance (at least 1)
// that the stack is stronger than three quarters of; when the last move of the stack failed, only
// one it can find a path to. The target is taken
func (pass *chewPass) assignTarget(stack *chewAIStack) (int, int, bool) {
    strength := 0
    var group []units.StackUnit
    moveFailed := false
    for _, unit := range stack.Slots {
        if unit == nil {
            continue
        }
        group = append(group, unit)
        // the original reads the flag of the last unit only
        moveFailed = pass.Overland.MoveFailed[chewKey(unit)]
        strength += chewUnitStrength(unit) / 10
    }

    best := 0
    chosen := -1
    for index, target := range pass.Targets {
        if target.Value <= 0 || target.Strength * 3 / 4 >= strength {
            continue
        }
        distance := max(1, pass.World.Distance(target.X, target.Y, stack.X, stack.Y))
        weighted := max(1, target.Value / (distance * 4))
        if weighted <= best {
            continue
        }
        if moveFailed && len(group) > 0 {
            if _, ok := pass.Turn.path(stack.X, stack.Y, target.X, target.Y, pass.plane(), group); !ok {
                continue
            }
        }
        best = weighted
        chosen = index
    }
    if chosen < 0 {
        return 0, 0, false
    }
    pass.Targets[chosen].Value = 0
    return pass.Targets[chosen].X, pass.Targets[chosen].Y, true
}

// AI_Stacks_Roamers_Target_Or_Deploy: every roamer with no busy unit goes for a target. When there
// is an army of 8 or more, none found a target, and there is a war landmass elsewhere, the landmass
// waits to leave: its stage point is the free coast square nearest to the war landmass's coast
func (pass *chewPass) roamersTargetOrDeploy() {
    large := false
    noTarget := true
    for _, stack := range pass.Stacks {
        if stack.Type != chewStackRoamer {
            continue
        }
        if stack.Count() >= chewLargeStack {
            large = true
        }
        if slices.Contains(stack.Slots, nil) {
            continue
        }
        if ChewbotMoveLog {
            strength := 0
            for _, unit := range stack.Slots {
                strength += chewUnitStrength(unit) / 10
            }
            weakest := -1
            for _, target := range pass.Targets {
                if target.Value > 0 && (weakest < 0 || target.Strength < weakest) {
                    weakest = target.Strength
                }
            }
            chewMoveLog("%v: roamer of %v units at %v,%v, strength %v; %v targets on landmass %v, weakest %v (needs more than three quarters)", pass.World.Self.Wizard.Name, stack.Count(), stack.X, stack.Y, strength, len(pass.Targets), pass.Landmass, weakest)
        }
        if x, y, ok := pass.assignTarget(stack); ok {
            chewMoveLog("%v: %v units at %v,%v go for the target at %v,%v (eval %x)", pass.World.Self.Wizard.Name, stack.Count(), stack.X, stack.Y, x, y, pass.World.EvalAt(x, y, pass.WP))
            noTarget = false
            for slot := range stack.Slots {
                pass.order(stack, slot, x, y)
            }
        }
    }

    war := pass.Overland.WarTarget[pass.WP]
    if !large || !noTarget || war == 0 || war == pass.Landmass {
        return
    }
    warX, warY, _ := pass.World.dockCenter(pass.WP, war)
    if square, ok := pass.World.nearestFree(pass.World.DockSquares[pass.WP][pass.Landmass], pass.WP, warX, warY, 0); ok {
        pass.Overland.LandType[pass.WP][pass.Landmass] = chewLandLeaveable
        pass.Overland.StageX[pass.WP][pass.Landmass] = square.X
        pass.Overland.StageY[pass.WP][pass.Landmass] = square.Y
    }
}

// AI_Stacks_Order_To_War_Landmass: the units that fly or swim in roamers and unknown stacks go to
// the stage point of the war landmass (spirits not)
func (pass *chewPass) orderToWarLandmass() {
    war := pass.Overland.WarTarget[pass.WP]
    if war == 0 || war == pass.Landmass {
        return
    }
    x, y := pass.Overland.StageX[pass.WP][war], pass.Overland.StageY[pass.WP][war]
    if x == 0 && y == 0 {
        return
    }
    for _, stack := range pass.Stacks {
        if stack.Type != chewStackRoamer && stack.Type != chewStackUnknown {
            continue
        }
        for slot, unit := range stack.Slots {
            if unit == nil || !(unit.IsFlying() || unit.IsSwimmer() || unit.IsSailing()) || chewIsMelder(unit) {
                continue
            }
            pass.order(stack, slot, x, y)
        }
    }
}

// AI_Stacks_Relocate_Roamers: roamers fill the stack at the stage point up to 9 (none go while no
// stack stands there) and ask for a ship next to the stage point
func (pass *chewPass) relocateRoamers() {
    stageX, stageY := pass.stage()
    room := 0
    for _, stack := range pass.Stacks {
        if stack.X == stageX && stack.Y == stageY {
            room = chewMaxStack - stack.Count()
        }
    }
    for _, stack := range pass.Stacks {
        if stack.Type != chewStackRoamer || stack.Slots[0] == nil {
            continue
        }
        pass.setupFerry(stack, stageX, stageY, room)
        room -= stack.Count()
    }
}

// AI_Stacks_Setup_Ferry: the sea square next to the stage point is a ferry point; units go to the
// stage point while there is room; when own units are at sea next to the stack, as many as their
// ships have room for go onto them
func (pass *chewPass) setupFerry(stack *chewAIStack, stageX int, stageY int, room int) {
    world := pass.World
    for dy := -1; dy <= 1; dy++ {
        for dx := -1; dx <= 1; dx++ {
            x, y := world.WrapX(stageX + dx), stageY + dy
            if world.inside(y) && world.LandmassAt(x, y, pass.WP) == 0 {
                pass.Turn.addFerry(x, y, pass.WP)
            }
        }
    }

    if stack.X != stageX || stack.Y != stageY {
        for slot := range stack.Slots {
            if room <= 0 {
                break
            }
            pass.order(stack, slot, stageX, stageY)
            room -= 1
        }
    }

    found := false
    var boatSquare image.Point
    capacity := 0
    free := chewMaxStack
    for _, unitStack := range world.Self.Stacks {
        if chewPlaneIndex(unitStack.Plane()) != pass.WP || world.LandmassAt(unitStack.X(), unitStack.Y(), pass.WP) != 0 {
            continue
        }
        if chewAbs(world.WrapX(unitStack.X() - stack.X + world.Width / 2) - world.Width / 2) > 1 || chewAbs(unitStack.Y() - stack.Y) > 1 {
            continue
        }
        for _, unit := range unitStack.Units() {
            found = true
            boatSquare = image.Pt(unitStack.X(), unitStack.Y())
            if chewIsBoat(unit) {
                // the ship holds where it is
                pass.Turn.order(unit, unitStack.X(), unitStack.Y(), pass.plane())
                capacity += int(unit.GetAbilityValue(data.AbilityTransport))
            } else if !chewSeafaring(unit) {
                capacity -= 1
            }
            free -= 1
        }
    }
    if !found {
        return
    }
    for slot := range stack.Slots {
        if slot >= free || capacity <= 0 {
            continue
        }
        unit := stack.Slots[slot]
        pass.order(stack, slot, boatSquare.X, boatSquare.Y)
        // the original takes a place for every unit but the ones that both fly and swim
        if unit != nil && !(unit.IsFlying() && (unit.IsSwimmer() || unit.IsSailing())) {
            capacity -= 1
        }
    }
}

// AI_Stacks_Stage_Expedition_Forces: with 9 at the stage point, one time in 20 the landmass is
// looked at again. While fewer than the size of an expedition (2, and 1 more every 30 turns, up to
// 9) wait or are on their way, and enough could come or the landmass is not held, the strongest
// units the survey found go to the stage point
func (pass *chewPass) stageExpeditionForces() {
    if pass.Staged == chewMaxStack && chewRandom(20) == 1 {
        pass.Overland.reevaluateContinent(pass.World, pass.WP, pass.Landmass)
    }
    threshold := pass.Turn.ExpeditionSize
    if ChewbotMoveLog && pass.World.Turn % 10 == 0 {
        garrisoned, loose := 0, 0
        for _, stack := range pass.Stacks {
            if stack.Type == chewStackGarrison || stack.Type == chewStackFortress {
                garrisoned += stack.Count()
            } else {
                loose += stack.Count()
            }
        }
        if garrisoned + loose > 0 {
            chewMoveLog("%v: landmass %v type %v: %v in garrisons, %v outside; staged %v, on the way %v, drafted %v, expedition size %v, targets %v", pass.World.Self.Wizard.Name, pass.Landmass, pass.landType(), garrisoned, loose, pass.Staged, pass.Enroute, len(pass.Drafted), threshold, len(pass.Targets))
        }
    }
    if pass.Staged + pass.Enroute >= threshold {
        return
    }
    kind := pass.landType()
    if kind < chewLandLeaveable && kind != chewLandNoOwnCity && pass.Staged + pass.Enroute + len(pass.Drafted) < threshold {
        return
    }
    stageX, stageY := pass.stage()
    if stageX == 0 && stageY == 0 {
        return
    }
    gap := chewMaxStack - pass.Enroute
    for index := 0; index < gap && index < len(pass.Drafted); index++ {
        draft := pass.Drafted[index]
        if draft.Stack.Slots[draft.Slot] == draft.Unit {
            pass.order(draft.Stack, draft.Slot, stageX, stageY)
        } else {
            // it was given another order in this turn; the original gives it this one after it
            pass.Turn.order(draft.Unit, stageX, stageY, pass.plane())
        }
    }
}

// AI_Stacks_Garrison_Sites: the garrisons the cities of the landmass want (the fortress 9, others
// 2 and a unit for 3 citizens, for 4 where the wizard feels safe or the race is strong; at most 9)
// and the free nodes (8, where it does not feel safe 4 for a node that has a garrison); every
// unknown stack and roamer away from the stage point sends units to the site with the smallest
// distance less what it wants, a city before any node
func (pass *chewPass) garrisonSites() {
    world := pass.World
    kind := pass.landType()
    safe := kind == chewLandOwn || kind >= chewLandLeaveable
    fortress := chewFortress(world.Self)

    type site struct {
        X int
        Y int
        Want int
    }
    stackAt := func(x int, y int) *chewAIStack {
        for _, stack := range pass.Stacks {
            if stack.X == x && stack.Y == y {
                return stack
            }
        }
        return nil
    }

    var cities []site
    for _, city := range world.Cities {
        if city.Owner != world.Self || chewPlaneIndex(city.City.Plane) != pass.WP || world.LandmassAt(city.City.X, city.City.Y, pass.WP) != pass.Landmass {
            continue
        }
        strong := city.City.Race == data.RaceDwarf || city.City.Race == data.RaceTroll || city.City.Race == data.RaceDraconian
        want := chewGarrisonBase + city.City.Citizens() / chewGarrisonPerCitizens
        if safe || strong {
            want = chewGarrisonBase + city.City.Citizens() / chewGarrisonPerCitizensSafe
        }
        if fortress != nil && city.City == fortress {
            want = chewMaxStack
        }
        want = min(want, chewMaxStack)
        if stack := stackAt(city.City.X, city.City.Y); stack != nil {
            want -= stack.Count()
        }
        if want > 0 {
            cities = append(cities, site{X: city.City.X, Y: city.City.Y, Want: want})
        }
    }

    var nodes []site
    for _, node := range chewNodePoints(world.Maps[pass.WP]) {
        if world.EvalAt(node.X, node.Y, pass.WP) != chewEvalSite || world.LandmassAt(node.X, node.Y, pass.WP) != pass.Landmass {
            continue
        }
        want := chewNodeGarrison
        if stack := stackAt(node.X, node.Y); stack != nil {
            // the original: 8 where it feels safe, 4 where it does not (meant the other way round)
            if safe {
                want = chewNodeGarrison - stack.Count()
            } else {
                want = chewNodeGarrisonUnsafe - stack.Count()
            }
        }
        if want > 0 {
            nodes = append(nodes, site{X: node.X, Y: node.Y, Want: want})
        }
    }

    stageX, stageY := pass.stage()
    for _, stack := range pass.Stacks {
        if stack.Type != chewStackUnknown && stack.Type != chewStackRoamer {
            continue
        }
        if stack.X == stageX && stack.Y == stageY {
            continue
        }
        best := 1000
        var target *site
        for index := range cities {
            if cities[index].Want <= 0 {
                continue
            }
            value := world.Distance(stack.X, stack.Y, cities[index].X, cities[index].Y) - cities[index].Want
            if value < best {
                best = value
                target = &cities[index]
            }
        }
        if best == 1000 {
            for index := range nodes {
                if nodes[index].Want <= 0 {
                    continue
                }
                value := world.Distance(stack.X, stack.Y, nodes[index].X, nodes[index].Y) - nodes[index].Want
                if value < best {
                    best = value
                    target = &nodes[index]
                }
            }
        }
        if target == nil {
            continue
        }
        send := target.Want
        for slot, unit := range stack.Slots {
            // the original counts the slots it looked at, not the units it sent
            if slot >= send {
                break
            }
            if unit == nil || chewBuildsRoads(unit) || chewIsSettler(unit) || chewIsMelder(unit) {
                continue
            }
            pass.order(stack, slot, target.X, target.Y)
            target.Want -= 1
        }
    }
}
