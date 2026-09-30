package ai

// The orders of the overland AI of Chewbot at sea: ships, ferry points, landings. Facts from the
// ReMoM project's reconstruction (MoM/src/AIMOVE.c: AI_Move_Out_Boats, AI_Stack_Set_Boats_Goto,
// AI_Stacks_Ferry_Add_Location, AI_Stacks_Wartime_Ocean_Movement_And_Cleanup,
// AI_Stacks_Ocean_Landmass_Orders, AI_Enemy_Unit_In_Range, Map_Square_Area_Has_Opponent). The code
// is ours. The rules in words: docs/mod/ai-overland.md.

import (
    "image"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

const (
    // the original's list of places ships are asked to
    chewMaxFerries = 15
    // AI_Enemy_Unit_In_Range: settlers do not land where enemies are this near
    chewSettlerLandingRange = 5
)

// a sea square next to a stage point where ships are wanted, and the room left there
type chewFerry struct {
    X int
    Y int
    WP int
    Valid bool
    Spaces int
}

// AI_Stacks_Ferry_Add_Location: a ferry point, once; false when the list is full
func (turn *chewTurn) addFerry(x int, y int, wp int) bool {
    for _, ferry := range turn.Ferries {
        if ferry.X == x && ferry.Y == y && ferry.WP == wp {
            return true
        }
    }
    if len(turn.Ferries) >= chewMaxFerries {
        return false
    }
    turn.Ferries = append(turn.Ferries, chewFerry{X: x, Y: y, WP: wp, Valid: true, Spaces: chewMaxStack})
    return true
}

// the stacks of the wizard with a ship
func (turn *chewTurn) boatStacks(wp int) []*playerlib.UnitStack {
    var out []*playerlib.UnitStack
    for _, stack := range turn.World.Self.Stacks {
        if chewPlaneIndex(stack.Plane()) != wp {
            continue
        }
        for _, unit := range stack.Units() {
            if chewIsBoat(unit) {
                out = append(out, stack)
                break
            }
        }
    }
    return out
}

// AI_Move_Out_Boats: ships on land (in a city) go out to a sea square next to it (the last one by
// rows)
func (turn *chewTurn) moveOutBoats() {
    world := turn.World
    for wp := range 2 {
        for _, stack := range turn.boatStacks(wp) {
            if turn.Skip[stack] || world.LandmassAt(stack.X(), stack.Y(), wp) == 0 {
                continue
            }
            found := false
            var sea image.Point
            for dy := -1; dy <= 1; dy++ {
                for dx := -1; dx <= 1; dx++ {
                    if dx == 0 && dy == 0 {
                        continue
                    }
                    x, y := world.WrapX(stack.X() + dx), stack.Y() + dy
                    if world.inside(y) && chewIsSea(world.Maps[wp].GetTile(x, y)) {
                        found = true
                        sea = image.Pt(x, y)
                    }
                }
            }
            if !found {
                continue
            }
            for _, unit := range stack.Units() {
                if chewIsBoat(unit) {
                    turn.order(unit, sea.X, sea.Y, chewPlaneOf(wp))
                }
            }
        }
    }
}

// units of another player on a landmass within a range of a square (AI_Enemy_Unit_In_Range, meant
// for enemies; the original looks at the wizard's own units there)
func (world *chewWorld) enemyUnitNear(x int, y int, wp int, reach int, landmass int) bool {
    for _, player := range world.Players {
        if player == world.Self {
            continue
        }
        for _, stack := range player.Stacks {
            if chewPlaneIndex(stack.Plane()) != wp || len(stack.Units()) == 0 {
                continue
            }
            if world.Distance(x, y, stack.X(), stack.Y()) > reach {
                continue
            }
            if landmass >= 0 && world.LandmassAt(stack.X(), stack.Y(), wp) != landmass {
                continue
            }
            return true
        }
    }
    return false
}

// AI_Stacks_Wartime_Ocean_Movement_And_Cleanup: with a war landmass, units at sea that fly or swim
// and have no order go to its stage point (or, with none, the one of the contested or unsettled
// landmass nearest to the fortress). The original then kills stacks at sea with no ship and no
// unit that can be there; the game has none such
func (turn *chewTurn) wartimeOceanMovement(wp int) {
    overland := turn.Overland
    world := turn.World
    war := overland.WarTarget[wp]
    if war == 0 {
        return
    }
    stageX, stageY := overland.StageX[wp][war], overland.StageY[wp][war]
    if stageX == 0 && stageY == 0 {
        fortressX, fortressY := 0, 0
        if fortress := chewFortress(world.Self); fortress != nil {
            fortressX, fortressY = fortress.X, fortress.Y
        }
        best := 1000
        for landmass := range overland.LandType[wp] {
            kind := overland.LandType[wp][landmass]
            if kind != chewLandContested && kind != chewLandNoOwnCity {
                continue
            }
            distance := world.Distance(fortressX, fortressY, overland.StageX[wp][landmass], overland.StageY[wp][landmass])
            if distance < best {
                best = distance
                stageX, stageY = overland.StageX[wp][landmass], overland.StageY[wp][landmass]
            }
        }
    }
    if stageX == 0 && stageY == 0 {
        return
    }

    for _, stack := range world.Self.Stacks {
        if chewPlaneIndex(stack.Plane()) != wp || world.LandmassAt(stack.X(), stack.Y(), wp) != 0 || turn.Skip[stack] {
            continue
        }
        for _, unit := range stack.Units() {
            if _, has := overland.Orders[chewKey(unit)]; has {
                continue
            }
            if chewIsMelder(unit) || chewIsBoat(unit) {
                continue
            }
            if chewSeafaring(unit) || unit.HasAbility(data.AbilityNonCorporeal) {
                turn.order(unit, stageX, stageY, chewPlaneOf(wp))
            }
        }
    }
}

// the room of a ship, and the place of the ship itself
func chewTransportRoom(unit units.StackUnit) int {
    if !chewIsBoat(unit) {
        return 1
    }
    return int(unit.GetAbilityValue(data.AbilityTransport)) + 1
}

// AI_Stacks_Ocean_Landmass_Orders: the stacks at sea. Ships alone go to the nearest ferry point
// that has room; a ship with others on board and no order lands them on a free square next to it
// that is on the war landmass or on a landmass with no city of the wizard or an ally (settlers: on
// the landmass for settlers, where no enemy is near), else sails the settlers to the landmass for
// settlers or the army to the coast of the war landmass
func (turn *chewTurn) oceanLandmassOrders(wp int) {
    world := turn.World
    overland := turn.Overland
    plane := chewPlaneOf(wp)

    for index := range turn.Ferries {
        ferry := &turn.Ferries[index]
        ferry.Spaces = chewMaxStack
    }
    for key, order := range overland.Orders {
        if order.Kind != chewOrderGoto || chewPlaneIndex(order.Plane) != wp {
            continue
        }
        for index := range turn.Ferries {
            ferry := &turn.Ferries[index]
            if ferry.WP == wp && ferry.X == order.X && ferry.Y == order.Y {
                ferry.Spaces -= chewTransportRoom(key)
            }
        }
    }
    for index := range turn.Ferries {
        if turn.Ferries[index].Spaces <= 0 {
            turn.Ferries[index].Valid = false
        }
    }

    pass := turn.initPass(wp, 0)
    pass.doMeld()

    for _, stack := range pass.Stacks {
        hasBoat, onlyBoats, hasSettler, hasGoto := false, true, false, false
        if unitStack := world.Self.FindStack(stack.X, stack.Y, plane); unitStack != nil {
            for _, unit := range unitStack.Units() {
                if chewIsBoat(unit) {
                    hasBoat = true
                } else {
                    onlyBoats = false
                }
                if chewIsSettler(unit) {
                    hasSettler = true
                }
                if order, has := overland.Orders[chewKey(unit)]; has && order.Kind == chewOrderGoto {
                    hasGoto = true
                }
            }
        }
        if !hasBoat {
            continue
        }

        if onlyBoats {
            best := 10000
            chosen := -1
            for index, ferry := range turn.Ferries {
                if !ferry.Valid || ferry.WP != wp {
                    continue
                }
                distance := world.Distance(stack.X, stack.Y, ferry.X, ferry.Y)
                if distance < best {
                    best = distance
                    chosen = index
                }
            }
            if chosen < 0 {
                continue
            }
            ferry := &turn.Ferries[chosen]
            room := ferry.Spaces
            for slot, unit := range stack.Slots {
                if room <= 0 || unit == nil {
                    continue
                }
                room -= chewTransportRoom(unit)
                pass.order(stack, slot, ferry.X, ferry.Y)
            }
            ferry.Spaces = room
            if room <= 0 {
                ferry.Valid = false
            }
            continue
        }

        if hasGoto {
            continue
        }

        // a free land square next to the ship (the last one by rows)
        landX, landY, landmass := 0, 0, 0
        allowed := false
        for dy := -1; dy <= 1; dy++ {
            for dx := -1; dx <= 1; dx++ {
                x, y := world.WrapX(stack.X + dx), stack.Y + dy
                if !world.inside(y) {
                    continue
                }
                if world.LandmassAt(x, y, wp) != 0 && world.EvalAt(x, y, wp) == 0 {
                    landX, landY, landmass = x, y, world.LandmassAt(x, y, wp)
                    allowed = true
                }
            }
        }

        if allowed && !hasSettler {
            allowed = false
            war := overland.WarTarget[wp]
            for dy := -1; dy <= 1; dy++ {
                for dx := -1; dx <= 1; dx++ {
                    x, y := world.WrapX(stack.X + dx), stack.Y + dy
                    if world.inside(y) && war != 0 && world.LandmassAt(x, y, wp) == war {
                        landX, landY, landmass = x, y, war
                        allowed = true
                    }
                }
            }
            if !allowed {
                friendly := false
                for _, city := range world.Cities {
                    if chewPlaneIndex(city.City.Plane) != wp || world.LandmassAt(city.City.X, city.City.Y, wp) != landmass {
                        continue
                    }
                    if city.Owner == world.Self || chewTreaty(world.Self, city.Owner) == data.TreatyAlliance {
                        friendly = true
                    }
                }
                allowed = !friendly
            }
        } else if allowed {
            target := overland.SettlerLandmass[wp]
            for dy := -1; dy <= 1; dy++ {
                for dx := -1; dx <= 1; dx++ {
                    x, y := world.WrapX(stack.X + dx), stack.Y + dy
                    if world.inside(y) && target != 0 && world.LandmassAt(x, y, wp) == target {
                        landX, landY, landmass = x, y, target
                    }
                }
            }
            enemies := world.enemyUnitNear(stack.X, stack.Y, wp, chewSettlerLandingRange, landmass)
            if (enemies || overland.LandType[wp][landmass] != chewLandNoOwnCity) && target != landmass {
                allowed = false
            }
        }

        if allowed {
            for slot, unit := range stack.Slots {
                if unit == nil {
                    continue
                }
                if chewIsBoat(unit) {
                    // the ship waits for the next load
                    stack.Slots[slot] = nil
                } else {
                    pass.order(stack, slot, landX, landY)
                }
            }
            continue
        }

        if hasSettler && overland.SettlerLandmass[wp] != 0 {
            seaX, seaY := 0, 0
            for dy := -1; dy <= 1; dy++ {
                for dx := -1; dx <= 1; dx++ {
                    x, y := world.WrapX(overland.SettlerX[wp] + dx), overland.SettlerY[wp] + dy
                    if world.inside(y) && world.LandmassAt(x, y, wp) == 0 {
                        seaX, seaY = x, y
                    }
                }
            }
            if seaX != 0 || seaY != 0 {
                for slot := range stack.Slots {
                    pass.order(stack, slot, seaX, seaY)
                }
            }
            continue
        }

        if war := overland.WarTarget[wp]; war != 0 {
            best := 1000
            var dock image.Point
            for _, square := range world.DockSquares[wp][war] {
                distance := world.Distance(square.X, square.Y, stack.X, stack.Y)
                if distance < best && world.EvalAt(square.X, square.Y, wp) == 0 {
                    best = distance
                    dock = square
                }
            }
            if best < 1000 {
                seaX, seaY := stack.X, stack.Y
                for dy := -1; dy <= 1; dy++ {
                    for dx := -1; dx <= 1; dx++ {
                        x, y := world.WrapX(dock.X + dx), dock.Y + dy
                        if world.inside(y) && world.LandmassAt(x, y, wp) == 0 {
                            seaX, seaY = x, y
                        }
                    }
                }
                for slot := range stack.Slots {
                    pass.order(stack, slot, seaX, seaY)
                }
            }
        }
    }
}
