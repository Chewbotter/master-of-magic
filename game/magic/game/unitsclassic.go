package game

// Units on the world map as the original game has them. Rules in words: docs/mod/units.md.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/NEXTTURN.c: Create_Unit,
// Heal_All_Units, Heal_Unit; CITYCALC.c: Evict_Weakest_Unit; SETTLE.c: Evict_Unit,
// Army_Do_Settle; UNITSTK.c: Next_Turn_Process_Purify, Army_Do_Meld, Unit_Action_Special_Meld;
// Outpost.c: Map_Square_Survey). The code is ours.

import (
    "image"
    "log"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/maplib"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// Army_Do_Settle: a settler that is undead can not settle
func canSettle(unit units.StackUnit) bool {
    if !unit.HasAbility(data.AbilityCreateOutpost) {
        return false
    }
    return !(units.ClassicUnits && unit.IsUndead())
}

// the first active unit of a stack that can settle, nil when none
func stackSettler(stack *playerlib.UnitStack) units.StackUnit {
    for _, unit := range stack.ActiveUnits() {
        if canSettle(unit) {
            return unit
        }
    }
    return nil
}

// Create_Unit: a unit built in a city with no better weapon gets magic weapons with Alchemy
func classicAlchemyWeapon(player *playerlib.Player, unit units.StackUnit) {
    if !units.ClassicUnits || unit.GetRace() == data.RaceFantastic {
        return
    }
    if unit.GetWeaponBonus() == data.WeaponNone && player.Wizard.RetortEnabled(data.RetortAlchemy) {
        unit.SetWeaponBonus(data.WeaponMagic)
    }
}

// Heal_All_Units: every unit that is hurt heals the whole of its hits / divisor, the rest as a
// chance; a healer in the stack heals every other unit a fifth more, once; undead and death never
func classicHealStack(stack *playerlib.UnitStack, divisor int) {
    healer := false
    for _, unit := range stack.Units() {
        if unit.HasAbility(data.AbilityHealer) {
            healer = true
        }
    }
    for _, unit := range stack.Units() {
        if unit.IsUndead() || unit.GetRealm() == data.DeathMagic || unit.GetDamage() <= 0 {
            continue
        }
        total := unit.GetMaxHealth()
        heal := units.ClassicHealAmount(total, divisor)
        if healer {
            heal += total / 5
        }
        unit.AdjustHealth(min(heal, unit.GetDamage()))
    }
}

// the level of a unit as Evict_Weakest_Unit counts it
func classicUnitLevel(unit units.StackUnit) int {
    if unit.IsHero() {
        return int(unit.GetHeroExperienceLevel())
    }
    return int(unit.GetExperienceLevel())
}

// Evict_Weakest_Unit: while more than 9 units stand on the square, the weakest (gold and mana
// upkeep and level, the last of equals) goes to the first free square around, row by row from the
// upper left; a neutral one, or one with nowhere to go, is lost
func (game *Game) classicEvictWeakest(x int, y int, plane data.Plane) {
    for {
        stack, player := game.Model.FindStack(x, y, plane)
        if stack == nil || len(stack.Units()) <= data.MaxUnitsInStack {
            return
        }

        var weakest units.StackUnit
        least := 0
        for _, unit := range stack.Units() {
            score := player.UnitGoldUpkeep(unit) + player.UnitManaUpkeep(unit) + classicUnitLevel(unit)
            if weakest == nil || score <= least {
                weakest = unit
                least = score
            }
        }

        if player.IsNeutral() {
            player.RemoveUnit(weakest)
            continue
        }

        place, ok := game.classicEvictPlace(stack, player, weakest)
        if !ok {
            log.Printf("Unit %v of %v lost: no room around %v,%v", weakest.GetName(), player.Wizard.Name, x, y)
            player.RemoveUnit(weakest)
            continue
        }

        stack.RemoveUnit(weakest)
        weakest.SetX(place.X)
        weakest.SetY(place.Y)
        player.AddUnit(weakest)
    }
}

// Evict_Unit: the first square around, row by row from the upper left, without a lair or tower,
// without a city of the owner, without units of another, with room, that the unit can stand on
// (the original takes the row of the edge again at the top of the map; here a square off the map
// is left out)
func (game *Game) classicEvictPlace(stack *playerlib.UnitStack, player *playerlib.Player, unit units.StackUnit) (image.Point, bool) {
    mapUse := game.GetMap(stack.Plane())
    plane := stack.Plane()
    // Unit_Space_At_Square: water for a unit that flies, swims, sails or walks the wind itself, or
    // onto a square where a wind walker stands
    water := unit.IsFlying() || unit.IsSwimmer() || unit.IsSailing() || unit.HasAbility(data.AbilityWindWalking)

    for dy := -1; dy <= 1; dy++ {
        for dx := -1; dx <= 1; dx++ {
            if dx == 0 && dy == 0 {
                continue
            }
            cx := mapUse.WrapX(stack.X() + dx)
            cy := stack.Y() + dy
            if cy < 0 || cy >= mapUse.Height() {
                continue
            }
            if mapUse.GetEncounter(cx, cy) != nil || mapUse.HasOpenTower(cx, cy) {
                continue
            }
            if player.FindCity(cx, cy, plane) != nil {
                continue
            }
            other, owner := game.Model.FindStack(cx, cy, plane)
            if other != nil && (owner != player || len(other.Units()) >= data.MaxUnitsInStack) {
                continue
            }
            isWater := mapUse.GetTile(cx, cy).Tile.IsWater()
            carried := false
            if other != nil {
                for _, there := range other.Units() {
                    if there.HasAbility(data.AbilityWindWalking) {
                        carried = true
                    }
                }
            }
            if isWater && !water && !carried {
                continue
            }
            if !isWater && unit.IsSailing() {
                continue
            }
            return image.Pt(cx, cy), true
        }
    }
    return image.Point{}, false
}

// Next_Turn_Process_Purify: every purifying unit counts its turns from 0; the square is clean when
// the counts of the units there summed before this turn's step reach 4 (one unit 5 turns, two or
// three 3, four or more 2)
func (model *GameModel) classicPurify(player *playerlib.Player) {
    for _, stack := range player.Stacks {
        var workers []*units.OverworldUnit
        for _, unit := range stack.Units() {
            if unit.GetBusy() != units.BusyStatusPurify {
                continue
            }
            if !unit.HasAbility(data.AbilityPurify) {
                unit.SetBusy(units.BusyStatusNone)
                continue
            }
            if worker, ok := unit.(*units.OverworldUnit); ok {
                workers = append(workers, worker)
            }
        }
        if len(workers) == 0 {
            continue
        }

        progress := 0
        for _, worker := range workers {
            progress += worker.PurifyTurns
            worker.PurifyTurns += 1
        }

        mapUse := model.GetMap(stack.Plane())
        if progress >= 4 || !mapUse.HasCorruption(stack.X(), stack.Y()) {
            mapUse.RemoveCorruption(stack.X(), stack.Y())
            for _, worker := range workers {
                worker.SetBusy(units.BusyStatusNone)
            }
        }
    }
}

// Army_Do_Meld: the first Guardian Spirit of the stack, else the last unit that melds
func classicMelder(stackUnits []units.StackUnit) units.StackUnit {
    var last units.StackUnit
    for _, unit := range stackUnits {
        if !unit.HasAbility(data.AbilityMeld) {
            continue
        }
        if raw := unit.GetRawUnit(); raw.Equals(units.GuardianSpirit) {
            return unit
        }
        last = unit
    }
    return last
}

// Unit_Action_Special_Meld: not on a warped node; the wizard's own node only to put a Guardian
// Spirit on it where there is none
func classicCanMeld(stackUnits []units.StackUnit, player *playerlib.Player, node *maplib.ExtraMagicNode) bool {
    if node == nil || node.Warped {
        return false
    }
    melder := classicMelder(stackUnits)
    if melder == nil {
        return false
    }
    if node.MeldingWizard == player {
        raw := melder.GetRawUnit()
        return !node.GuardianSpiritMeld && raw.Equals(units.GuardianSpirit)
    }
    return true
}
