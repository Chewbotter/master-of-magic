package ai

// The overland AI of Chewbot: where the armies, settlers, engineers, spirits, priests and ships of a
// computer wizard go. The original's AI_Set_Unit_Orders, ported (the ReMoM project's
// reconstruction, MoM/src/AIMOVE.c; the turn in MoM/src/AIDUDES.c AI_Next_Turn). The code is
// ours. The rules in words: docs/mod/ai-overland.md.
//
// The original gives every unit a status (ready, going to a square, attacking, building a road,
// purifying) and moves the units of every status in its own phase. Here the orders are kept in
// chewOverland.Orders, and every turn become the game's decisions: a path for every group of units
// of a stack that go to the same square, and the decisions to settle, meld, purify, build a road
// and cross to the other plane. The clone's orders to move are dropped.

import (
    "image"
    "log"
    "sync"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/game/magic/maplib"
    "github.com/kazzmir/master-of-magic/game/magic/pathfinding"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

// false: the clone moves the armies whatever the setting says
var ChewbotMoves = true

// development: the orders of every turn in the log
var ChewbotMoveLog = false

// development: when not nil, the units given an order, by wizard and by the step of the original
// that gave it (a run without a window counts them for its summary)
var ChewbotOrderCounts map[string]map[string]int
var chewOrderCountsLock sync.Mutex

func chewCountOrder(wizard string, why string, count int) {
    chewOrderCountsLock.Lock()
    defer chewOrderCountsLock.Unlock()
    if ChewbotOrderCounts == nil {
        return
    }
    if ChewbotOrderCounts[wizard] == nil {
        ChewbotOrderCounts[wizard] = make(map[string]int)
    }
    ChewbotOrderCounts[wizard][why] += count
}

// The original's own mistakes in its overland orders that change how it plays, kept on purpose
const (
    // Map_Square_Area_Has_Opponent looks for the wizard's own units instead of an opponent's, and
    // always finds the settler itself: after turn 100 a settler settles one time in 5 only, enemies
    // near or not
    quirkSettleOpponentCheck = true
    // AI_Choose_War_Landmass and AI_Evaluate_Hostility look for a wizard that casts Fire Elemental
    // overland (meant: the Spell of Mastery), which never happens: a wizard casting the Spell of
    // Mastery is no reason to pick a war landmass
    quirkWarOnMastery = true
)

// the city a wizard's army sees as an opportunity has fewer defenders than this
const chewOpportunityDefenders = 4

func chewbotMovesActive() bool {
    return ChewbotMoves && display.ChewbotAI()
}

func chewMoveLog(format string, args ...any) {
    if ChewbotMoveLog {
        log.Printf("chewbot move: " + format, args...)
    }
}

type chewUnitKey = units.StackUnit

func chewKey(unit units.StackUnit) chewUnitKey {
    return unit
}

// the orders that last (the original's us_GOTO and us_Move)
const (
    chewOrderGoto = iota
    chewOrderAttack
)

type chewOrder struct {
    Kind int
    X int
    Y int
    Plane data.Plane
    // the step of the original that gave it, for the log
    Why string
}

// one turn of the overland AI of a wizard
type chewTurn struct {
    AI *ChewbotAI
    Overland *chewOverland
    World *chewWorld
    // stacks that cross to the other plane in this turn and get no other order
    Skip map[*playerlib.UnitStack]bool
    Shifts []*playerlib.UnitStack
    Settles []units.StackUnit
    Melds []units.StackUnit
    Purifies []units.StackUnit
    Roads map[chewUnitKey]image.Point
    Ferries []chewFerry
    // _ai_expedition_size_threshold
    ExpeditionSize int
    // the step of the original that gives orders now, for the log
    Why string
}

// AI_Stacks_Order_Attack_Target_Or_Goto_Destination: attack when an enemy stack off a site stands
// there, else go there
func (turn *chewTurn) order(unit units.StackUnit, x int, y int, plane data.Plane) {
    if unit == nil {
        return
    }
    kind := chewOrderGoto
    value := turn.World.EvalAt(x, y, chewPlaneIndex(plane))
    if value & chewEvalSite == 0 && value & chewEvalStrength != 0 {
        kind = chewOrderAttack
    }
    turn.Overland.Orders[chewKey(unit)] = &chewOrder{Kind: kind, X: turn.World.WrapX(x), Y: y, Plane: plane, Why: turn.Why}
    chewCountOrder(turn.World.Self.Wizard.Name, turn.Why, 1)
}

// on its way, building a road or purifying: the original's units that are left out of the plans
func (turn *chewTurn) busy(unit units.StackUnit) bool {
    if order, has := turn.Overland.Orders[chewKey(unit)]; has && order.Kind == chewOrderGoto {
        return true
    }
    switch unit.GetBusy() {
        case units.BusyStatusBuildRoad, units.BusyStatusPurify: return true
    }
    // on its way along a road it builds
    return len(unit.GetBuildRoadPath()) > 0
}

// a path for a group of units
func (turn *chewTurn) path(x int, y int, toX int, toY int, plane data.Plane, group []units.StackUnit) (pathfinding.Path, bool) {
    self := turn.World.Self
    stack := playerlib.MakeUnitStackFromUnits(group)
    return turn.World.Services.FindPath(x, y, toX, toY, self, stack, self.GetFog(plane))
}

// the orders of units that are gone end, and so do the ones of units that have arrived
func (overland *chewOverland) pruneOrders(self *playerlib.Player) {
    alive := make(map[chewUnitKey]bool)
    for _, stack := range self.Stacks {
        for _, unit := range stack.Units() {
            alive[chewKey(unit)] = true
            if order, has := overland.Orders[chewKey(unit)]; has && order.Kind == chewOrderGoto {
                if stack.X() == order.X && stack.Y() == order.Y && stack.Plane() == order.Plane {
                    delete(overland.Orders, chewKey(unit))
                }
            }
        }
    }
    for key := range overland.Orders {
        if !alive[key] {
            delete(overland.Orders, key)
        }
    }
    for key := range overland.MoveFailed {
        if !alive[key] {
            delete(overland.MoveFailed, key)
        }
    }
}

// the overland orders of a computer wizard for this turn, as decisions of the game
func (ai *ChewbotAI) moveDecisions(self *playerlib.Player, services playerlib.AIServices) []playerlib.AIDecision {
    if ai.overland == nil {
        ai.overland = makeChewOverland()
    }
    overland := ai.overland

    world := makeChewWorld(self, services)
    overland.fit(world)
    overland.noteTreaties(world)
    overland.pruneOrders(self)
    overland.evaluateHostility(world)
    world.makeEvaluationMap(overland.hostilityOf)
    overland.chooseWarLandmass(world)
    overland.evaluateContinents(world)

    turn := &chewTurn{
        AI: ai,
        Overland: overland,
        World: world,
        Skip: make(map[*playerlib.UnitStack]bool),
        Roads: make(map[chewUnitKey]image.Point),
        ExpeditionSize: min(2 + world.Turn / 30, chewMaxStack),
    }
    turn.setUnitOrders()
    return turn.decisions()
}

// AI_Set_Unit_Orders
func (turn *chewTurn) setUnitOrders() {
    world := turn.World
    overland := turn.Overland

    // the original disbands units here when food or gold runs short (AI_Disband_To_Balance_Budget):
    // left to the game's own rules for now
    turn.shiftOffHomePlane()
    turn.Why = "boats"
    turn.moveOutBoats()
    turn.Why = "opportunity"
    turn.findOpportunityCityTarget()

    for wp := range 2 {
        for landmass := 1; landmass <= world.LandmassCount(wp); landmass++ {
            pass := turn.initPass(wp, landmass)
            turn.Why = "move out"
            pass.moveOutNonMilitaryGarrisoned()
            turn.Why = "survey"
            pass.surveyExpeditionForces()
            turn.Why = "meld"
            pass.doMeld()
            turn.Why = "settle"
            pass.doSettle()
            turn.Why = "purify"
            pass.doPurify()
            turn.Why = "road"
            pass.doRoadBuild()
            pass.buildTargetList()
            turn.Why = "target"
            pass.roamersTargetOrDeploy()

            kind := overland.LandType[wp][landmass]
            if kind >= chewLandLeaveable || kind == chewLandOwn || kind == chewLandIgnored || kind == chewLandNoOwnCity {
                turn.Why = "war landmass"
                pass.orderToWarLandmass()
            }
            if kind >= chewLandLeaveable || kind == chewLandOwn {
                turn.Why = "relocate"
                pass.relocateRoamers()
            }
            turn.Why = "stage"
            pass.stageExpeditionForces()
            kind = overland.LandType[wp][landmass]
            if kind == chewLandOwn || kind == chewLandContested || kind >= chewLandLeaveable {
                turn.Why = "garrison"
                pass.garrisonSites()
            }
        }
        turn.Why = "wartime sea"
        turn.wartimeOceanMovement(wp)
        turn.Why = "sea"
        turn.oceanLandmassOrders(wp)
    }
}

// AI_Shift_Off_Home_Plane: with no Planar Seal, a stack of the wizard on a tower of the plane of
// its fortress, or in a city of its own with an Astral Gate, crosses to the other plane
func (turn *chewTurn) shiftOffHomePlane() {
    world := turn.World
    fortress := chewFortress(world.Self)
    if fortress == nil || !world.planesOpen() {
        return
    }
    home := fortress.Plane
    for _, stack := range world.Self.Stacks {
        if stack.Plane() != home || len(stack.Units()) == 0 {
            continue
        }
        wp := chewPlaneIndex(home)
        shift := world.Maps[wp].HasOpenTower(stack.X(), stack.Y())
        if city := world.Self.FindCity(stack.X(), stack.Y(), home); city != nil && city.HasEnchantment(data.CityEnchantmentAstralGate) {
            shift = true
        }
        if shift {
            turn.Shifts = append(turn.Shifts, stack)
            turn.Skip[stack] = true
        }
    }
}

// AI_Find_Opportunity_City_Target: units on their way next to a city of another player with fewer
// than 4 units in it drop their orders (and so may attack it)
func (turn *chewTurn) findOpportunityCityTarget() {
    world := turn.World
    for _, stack := range world.Self.Stacks {
        going := false
        for _, unit := range stack.Units() {
            if order, has := turn.Overland.Orders[chewKey(unit)]; has && order.Kind == chewOrderGoto {
                going = true
            }
        }
        if !going {
            continue
        }
        wp := chewPlaneIndex(stack.Plane())
        found := false
        for _, city := range world.Cities {
            if city.Owner == world.Self || chewPlaneIndex(city.City.Plane) != wp {
                continue
            }
            if world.Distance(stack.X(), stack.Y(), city.City.X, city.City.Y) > 1 {
                continue
            }
            defenders := 0
            for _, player := range world.Players {
                if defending := player.FindStack(city.City.X, city.City.Y, city.City.Plane); defending != nil {
                    defenders += len(defending.Units())
                }
            }
            if defenders < chewOpportunityDefenders {
                found = true
            }
        }
        if found {
            for _, unit := range stack.Units() {
                delete(turn.Overland.Orders, chewKey(unit))
            }
        }
    }
}

// the orders as the game's decisions
func (turn *chewTurn) decisions() []playerlib.AIDecision {
    world := turn.World
    self := world.Self
    overland := turn.Overland
    overland.MoveFailed = make(map[chewUnitKey]bool)

    var decisions []playerlib.AIDecision
    name := self.Wizard.Name
    for _, stack := range turn.Shifts {
        decisions = append(decisions, &playerlib.AIPlaneShiftDecision{Stack: stack})
        chewCountOrder(name, "plane shift", len(stack.Units()))
    }

    settles := chewSetOf(turn.Settles)
    melds := chewSetOf(turn.Melds)
    purifies := chewSetOf(turn.Purifies)

    // every stack first stays where it is: a path of an earlier turn is not walked on
    for _, stack := range self.Stacks {
        if !turn.Skip[stack] {
            decisions = append(decisions, &playerlib.AIMoveStackDecision{Stack: stack})
        }
    }

    for _, stack := range append([]*playerlib.UnitStack(nil), self.Stacks...) {
        if turn.Skip[stack] {
            continue
        }

        // the units that go somewhere, by where
        var destinations []image.Point
        groups := make(map[image.Point][]units.StackUnit)
        for _, unit := range stack.Units() {
            order, has := overland.Orders[chewKey(unit)]
            if !has || order.Plane != stack.Plane() || (order.X == stack.X() && order.Y == stack.Y()) {
                continue
            }
            point := image.Pt(order.X, order.Y)
            if _, seen := groups[point]; !seen {
                destinations = append(destinations, point)
            }
            groups[point] = append(groups[point], unit)
        }
        for _, point := range destinations {
            group := groups[point]
            path, ok := turn.path(stack.X(), stack.Y(), point.X, point.Y, stack.Plane(), group)
            if !ok || len(path) == 0 {
                for _, unit := range group {
                    delete(overland.Orders, chewKey(unit))
                    overland.MoveFailed[chewKey(unit)] = true
                }
                chewMoveLog("%v: no path from %v,%v to %v for %v units", self.Wizard.Name, stack.X(), stack.Y(), point, len(group))
                chewCountOrder(name, "no path", len(group))
                continue
            }
            why := overland.Orders[chewKey(group[0])].Why
            chewMoveLog("%v: %v units from %v,%v to %v (%v steps, %v)", self.Wizard.Name, len(group), stack.X(), stack.Y(), point, len(path), why)
            decisions = append(decisions, &playerlib.AIMoveStackDecision{Stack: stack, Path: path, Units: group})
        }

        settled, melded := false, false
        var purifiers []units.StackUnit
        for _, unit := range stack.Units() {
            key := chewKey(unit)
            if settles[key] && !settled {
                settled = true
                chewMoveLog("%v: settles at %v,%v", self.Wizard.Name, stack.X(), stack.Y())
                decisions = append(decisions, &playerlib.AIBuildOutpostDecision{Stack: stack})
                chewCountOrder(name, "found a city", 1)
            }
            if melds[key] && !melded {
                melded = true
                decisions = append(decisions, &playerlib.AIMeldNodeDecision{Stack: stack})
                chewCountOrder(name, "meld", 1)
            }
            if purifies[key] {
                purifiers = append(purifiers, unit)
            }
            if road, has := turn.Roads[key]; has {
                chewMoveLog("%v: road from %v,%v to %v", self.Wizard.Name, stack.X(), stack.Y(), road)
                decisions = append(decisions, &playerlib.AIBuildRoadDecision{Stack: stack, X: road.X, Y: road.Y, Units: []units.StackUnit{unit}})
                chewCountOrder(name, "start a road", 1)
            } else if roadPath := unit.GetBuildRoadPath(); len(roadPath) > 0 && unit.GetBusy() != units.BusyStatusBuildRoad {
                // a road on its way: the game finished a square of it, the engineer goes on to the next
                end := roadPath[len(roadPath) - 1]
                decisions = append(decisions, &playerlib.AIBuildRoadDecision{Stack: stack, X: end.X, Y: end.Y, Units: []units.StackUnit{unit}})
            }
        }
        if len(purifiers) > 0 {
            decisions = append(decisions, &playerlib.AIPurifyDecision{Stack: stack, Units: purifiers})
            chewCountOrder(name, "purify", len(purifiers))
        }
    }
    return decisions
}

func chewSetOf(list []units.StackUnit) map[chewUnitKey]bool {
    out := make(map[chewUnitKey]bool)
    for _, unit := range list {
        out[chewKey(unit)] = true
    }
    return out
}

// the decisions of the clone that move stacks: Chewbot gives its own
func chewIsMoveDecision(decision playerlib.AIDecision) bool {
    switch decision.(type) {
        case *playerlib.AIMoveStackDecision, *playerlib.AIBuildOutpostDecision, *playerlib.AIBuildRoadDecision,
             *playerlib.AIMeldNodeDecision, *playerlib.AIPlaneShiftDecision:
            return true
    }
    return false
}

// a stack walks on along its path to where Chewbot sent it
func (ai *ChewbotAI) MovedStack(stack *playerlib.UnitStack, path pathfinding.Path) pathfinding.Path {
    if ai.Neutral || !chewbotMovesActive() {
        return ai.AIBehavior.MovedStack(stack, path)
    }
    return path
}

// a move that could not be made: the original's Move_Failed, read by the next choice of a target
func (ai *ChewbotAI) InvalidMove(stack *playerlib.UnitStack) {
    if ai.Neutral || !chewbotMovesActive() || ai.overland == nil {
        ai.AIBehavior.InvalidMove(stack)
        return
    }
    for _, unit := range stack.Units() {
        ai.overland.MoveFailed[chewKey(unit)] = true
    }
}

// a stack of Chewbot fights the guardians of a place only when it was sent there
func (ai *ChewbotAI) ConfirmEncounter(stack *playerlib.UnitStack, encounter *maplib.ExtraEncounter) bool {
    if ai.Neutral || !chewbotMovesActive() || ai.overland == nil {
        return ai.AIBehavior.ConfirmEncounter(stack, encounter)
    }
    for _, unit := range stack.Units() {
        order, has := ai.overland.Orders[chewKey(unit)]
        if !has || order.Plane != stack.Plane() {
            continue
        }
        if ai.encounterAt(order.X, order.Y, encounter) {
            return true
        }
    }
    return false
}

func (ai *ChewbotAI) encounterAt(x int, y int, encounter *maplib.ExtraEncounter) bool {
    if ai.services == nil {
        return false
    }
    for _, plane := range []data.Plane{data.PlaneArcanus, data.PlaneMyrror} {
        if ai.services.GetMap(plane).GetEncounter(x, y) == encounter {
            return true
        }
    }
    return false
}

// Update_Defender_Hostility
func (ai *ChewbotAI) WasAttacked(self *playerlib.Player, attacker *playerlib.Player) {
    if ai.Neutral {
        return
    }
    if ai.overland == nil {
        ai.overland = makeChewOverland()
    }
    ai.overland.wasAttacked(self, attacker)
}
