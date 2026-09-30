package ai

// Chewbot's turn glue: what the original's computer wizard does around its orders every turn: its
// gold and mana reserves, its tax and farmers, units it cannot pay for, settlers and engineers it
// has too many of, and units in Stasis that will not come out (ReMoM MoM/src/AIDUDES.c:
// AI_Update_Gold_And_Mana_Reserves, AI_Update_Gold_Income_And_Food_Income,
// AI_Kill_Excess_Settlers_And_Engineers, AI_Hopeless_Stasis, AI_Landmass_Values_And_Strengths;
// MoM/src/AIMOVE.c: AI_Disband_To_Balance_Budget; MoX/src/MOM_DAT.c: difficulty_modifiers_table).
// The rules in words: docs/mod/ai-turn.md. The code is ours.

import (
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    "github.com/kazzmir/master-of-magic/lib/fraction"
    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

// false: the clone's economy whatever the setting says
var ChewbotTurnGlue = true

func chewbotGlueActive() bool {
    return ChewbotTurnGlue && display.ChewbotAI()
}

const (
    // MAX_GOLD_RESERVE, MAX_MANA_RESERVE
    chewMaxReserve = 30000
    // the tax rates of the original: 0 to 3 gold a citizen in halves; the AI keeps to 4 (2 gold)
    chewTaxMost = 4
    // AI_Hopeless_Stasis: a unit in Stasis with less resistance dies
    chewStasisHopeless = 7
)

// the tax rates of the original by number
func chewTaxRate(index int) fraction.Fraction {
    return fraction.Make(index, 2)
}

func chewTaxIndex(rate fraction.Fraction) int {
    for index := 0; index <= 6; index++ {
        if rate.Equals(chewTaxRate(index)) {
            return index
        }
    }
    return 2
}

// what the glue keeps from turn to turn
type chewGlue struct {
    TaxIndex int
    TaxSet bool
}

// the clone's new turn resets the tax to 1 gold and turns mana into gold; with Chewbot's glue its
// gold, mana and tax stay as the glue left them (the clone's farmers are kept)
func (ai *ChewbotAI) NewTurn(self *playerlib.Player) {
    gold, mana := self.Gold, self.Mana
    ai.AIBehavior.NewTurn(self)
    if ai.Neutral || !chewbotGlueActive() {
        return
    }
    self.Gold, self.Mana = gold, mana
    if ai.glue != nil && ai.glue.TaxSet {
        self.UpdateTaxRate(chewTaxRate(ai.glue.TaxIndex))
    }
}

// the glue of a turn, before the orders of the units (the original: reserves and income after the
// spells, the budget first thing of the orders); the excess settlers go after the orders are set
// (excessBuildersAfterOrders)
func (ai *ChewbotAI) turnGlue(self *playerlib.Player, services playerlib.AIServices) {
    if ai.glue == nil {
        ai.glue = &chewGlue{TaxIndex: chewTaxIndex(self.TaxRate), TaxSet: true}
    }
    ai.updateReserves(self, services)
    ai.updateIncome(self, services)
    world := makeChewWorld(self, services)
    spells := ai.spellWorld(self, services)
    ai.disbandToBudget(self, services, world, spells)
}

// AI_Next_Turn: the excess settlers and engineers go after AI_Set_Unit_Orders and before the orders
// are carried out; the orders of the units that go are dropped
func (ai *ChewbotAI) excessBuildersAfterOrders(self *playerlib.Player, services playerlib.AIServices, decisions []playerlib.AIDecision) []playerlib.AIDecision {
    gone := ai.killExcessBuilders(self, makeChewWorld(self, services), int(services.GetTurnNumber()))
    if len(gone) == 0 {
        return decisions
    }
    removed := make(map[units.StackUnit]bool)
    for _, unit := range gone {
        removed[unit] = true
    }
    keep := func(units_ []units.StackUnit) ([]units.StackUnit, bool) {
        if len(units_) == 0 {
            return units_, true
        }
        var out []units.StackUnit
        for _, unit := range units_ {
            if !removed[unit] {
                out = append(out, unit)
            }
        }
        return out, len(out) > 0
    }
    var out []playerlib.AIDecision
    for _, decision := range decisions {
        switch decision := decision.(type) {
            case *playerlib.AIMoveStackDecision:
                if decision.Stack == nil || decision.Stack.IsEmpty() {
                    continue
                }
                left, ok := keep(decision.Units)
                if !ok {
                    continue
                }
                decision.Units = left
            case *playerlib.AIBuildRoadDecision:
                if decision.Stack == nil || decision.Stack.IsEmpty() {
                    continue
                }
                left, ok := keep(decision.Units)
                if !ok {
                    continue
                }
                decision.Units = left
            case *playerlib.AIBuildOutpostDecision:
                if decision.Stack == nil || decision.Stack.IsEmpty() {
                    continue
                }
        }
        out = append(out, decision)
    }
    return out
}

// AI_Update_Gold_And_Mana_Reserves: a wizard without Alchemy that has no mana takes a quarter of its
// gold as mana and keeps half; one time in 10 it moves a quarter of the larger into the smaller
// and keeps half of it (gold into mana when mana is below a fifth of the gold and there is more than
// 100 gold; mana into gold when gold is below a quarter of the mana and there is more than 100 mana;
// else before turn 100 mana into gold when there is more than 50). With Alchemy the two are 90 gold
// to 10 mana, from turn 150 on 30 to 70. Casting Spell of Return all gold goes to mana (half without
// Alchemy)
func (ai *ChewbotAI) updateReserves(self *playerlib.Player, services playerlib.AIServices) {
    alchemy := self.Wizard.RetortEnabled(data.RetortAlchemy)
    set := func(gold int, mana int) {
        self.Gold = min(gold, chewMaxReserve)
        self.Mana = min(mana, chewMaxReserve)
    }
    if self.CastingSpell.Name == "Spell of Return" {
        // the original sets the mana to it (meant: adds)
        if alchemy {
            set(0, self.Mana + self.Gold)
        } else {
            set(0, self.Mana + self.Gold / 2)
        }
        return
    }
    if alchemy {
        total := self.Gold + self.Mana
        if services.GetTurnNumber() >= 150 {
            mana := min(total * 7 / 10, chewMaxReserve)
            set(total - mana, mana)
        } else {
            gold := min(total * 9 / 10, chewMaxReserve)
            set(gold, total - gold)
        }
        return
    }
    if self.Mana == 0 {
        set(self.Gold / 2, self.Gold / 4)
    }
    if chewRoll(10) != 0 {
        return
    }
    gold, mana := self.Gold, self.Mana
    switch {
        case mana < gold / 5:
            if gold > 100 {
                set(gold - gold / 2, mana + gold / 4)
            }
        case gold < mana / 4:
            if mana > 100 {
                set(gold + mana / 4, mana - mana / 2)
            }
        default:
            if services.GetTurnNumber() < 100 && mana > 50 {
                set(gold + mana / 4, mana - mana / 2)
            }
    }
}

// AI_Update_Gold_Income_And_Food_Income: the tax up one step while the gold income is below 75 for
// every 50 turns (to 3 gold), down one step when it is above 200 (to half a gold), never above 2
// gold; with too little food a farmer more in its cities by chance until the food is made up
// (counting 2 food a farmer), 50 tries
func (ai *ChewbotAI) updateIncome(self *playerlib.Player, services playerlib.AIServices) {
    glue := ai.glue
    turn := int(services.GetTurnNumber())
    setTax := func(index int) {
        glue.TaxIndex = index
        self.UpdateTaxRate(chewTaxRate(index))
    }
    if turn / 50 * 75 > self.GoldPerTurn() && glue.TaxIndex < 6 {
        setTax(glue.TaxIndex + 1)
    }
    if self.GoldPerTurn() > 200 && glue.TaxIndex > 1 {
        setTax(glue.TaxIndex - 1)
    }
    if glue.TaxIndex > chewTaxMost {
        setTax(chewTaxMost)
    }

    food := self.FoodPerTurn()
    if food >= 0 {
        return
    }
    missing := -food
    var own []*citylib.City
    for _, city := range self.Cities {
        own = append(own, city)
    }
    chewSortCities(own)
    if len(own) == 0 {
        return
    }
    for tries := 0; tries < 50 && missing > 0; tries++ {
        city := own[chewRoll(len(own))]
        if city.Farmers < city.Citizens() && city.ConvertWorkerToFarmer() {
            city.UpdateUnrest()
            missing -= 2
        }
    }
}

// the power of a node as the original counts it for a landmass: its power times (magic + 1) times 2,
// magic 0 weak, 1 normal, 2 powerful
func chewNodeValueFactor(services playerlib.AIServices) int {
    magic := 1
    if provider, ok := services.(interface{ GetMagicSetting() data.MagicSetting }); ok {
        switch provider.GetMagicSetting() {
            case data.MagicSettingWeak: magic = 0
            case data.MagicSettingPowerful: magic = 2
        }
    }
    return (magic + 1) * 2
}

// AI_Landmass_Values_And_Strengths sets (does not add) the value of a landmass for every city, so
// the last city in the original's order counts (kept; the order here: plane, row, column)
const quirkLandmassLastCity = true

// the cost of a unit in the original's table: production, or casting for a creature
func chewTableCost(unit units.Unit) int {
    if unit.ProductionCost > 0 {
        return unit.ProductionCost
    }
    return unit.CastingCost
}

// AI_Landmass_Values_And_Strengths: per landmass, the wizard's strength (its stacks, a tenth of their
// value) less a tenth of what it has there (cities, nodes) less the strength of the others (their
// stacks outside cities and the guardians of lairs, count times cost / 10); above 0 less 25, not
// below 0. So 0 is a landmass where it is not clearly ahead or where it is behind by nothing
func (ai *ChewbotAI) landmassRatios(world *chewWorld, spells *chewSpellWorld, services playerlib.AIServices) [2]map[int]int {
    var own, enemy, value [2]map[int]int
    for wp := range 2 {
        own[wp] = make(map[int]int)
        enemy[wp] = make(map[int]int)
        value[wp] = make(map[int]int)
    }
    for _, stack := range spells.OwnStacks {
        wp := chewPlaneIndex(stack.Plane)
        own[wp][world.LandmassAt(stack.X, stack.Y, wp)] += stack.Value / 10
    }
    for _, stack := range spells.EnemyStacks {
        wp := chewPlaneIndex(stack.Plane)
        enemy[wp][world.LandmassAt(stack.X, stack.Y, wp)] += stack.Value / 10
    }
    for wp := range 2 {
        mapObject := services.GetMap(chewPlaneOf(wp))
        for _, point := range mapObject.GetEncounterLocations() {
            encounter := mapObject.GetEncounter(point.X, point.Y)
            if encounter == nil {
                continue
            }
            for _, unit := range encounter.Units {
                enemy[wp][world.LandmassAt(point.X, point.Y, wp)] += chewTableCost(unit) / 10
            }
        }
    }
    // the cities, every player's: the original sets the landmass's value to the wizard's value of the
    // city (0 for another player's), so the last city of a landmass counts; then the nodes add theirs
    var cities []*citylib.City
    for _, city := range world.Cities {
        cities = append(cities, city.City)
    }
    chewSortCityList(cities)
    for _, city := range cities {
        wp := chewPlaneIndex(city.Plane)
        landmass := world.LandmassAt(city.X, city.Y, wp)
        if quirkLandmassLastCity {
            value[wp][landmass] = spells.OwnValue[city]
        } else {
            value[wp][landmass] += spells.OwnValue[city]
        }
    }
    for wp := range 2 {
        mapObject := services.GetMap(chewPlaneOf(wp))
        for _, point := range mapObject.GetMagicNodeLocations() {
            node := mapObject.GetMagicNode(point.X, point.Y)
            if node != nil && chewNodeOwner(node, spells.Self) {
                value[wp][world.LandmassAt(point.X, point.Y, wp)] += len(node.Zone) * chewNodeValueFactor(services)
            }
        }
    }

    var ratios [2]map[int]int
    for wp := range 2 {
        ratios[wp] = make(map[int]int)
        landmasses := make(map[int]bool)
        for landmass := range own[wp] {
            landmasses[landmass] = true
        }
        for landmass := range enemy[wp] {
            landmasses[landmass] = true
        }
        for landmass := range value[wp] {
            landmasses[landmass] = true
        }
        for landmass := range landmasses {
            ratio := own[wp][landmass] - value[wp][landmass] / 10 - enemy[wp][landmass]
            if ratio > 0 {
                ratio = max(ratio - 25, 0)
            }
            ratios[wp][landmass] = ratio
        }
    }
    return ratios
}

// a unit of the wizard with where it stands
type chewOwnUnit struct {
    Unit units.StackUnit
    Landmass int
    Plane int
}

// AI_Disband_To_Balance_Budget: while the gold or food income is below 0 (200 at most), the weakest
// normal unit on land goes (a tenth of its strength, twice on a landmass where the wizard is not
// clearly ahead), making up a food and its gold upkeep times the maintenance of the difficulty (at
// least 1); while the mana income is below 0 the same with the summoned units and their mana
func (ai *ChewbotAI) disbandToBudget(self *playerlib.Player, services playerlib.AIServices, world *chewWorld, spells *chewSpellWorld) {
    power := services.ComputePower(self)
    goldDeficit := max(-self.GoldPerTurn(), 0)
    foodDeficit := max(-self.FoodPerTurn(), 0)
    manaDeficit := max(-self.ManaPerTurn(power, services), 0)
    if goldDeficit == 0 && foodDeficit == 0 && manaDeficit == 0 {
        return
    }
    ratios := ai.landmassRatios(world, spells, services)
    // difficulty_modifiers_table, maintenance (percent)
    maintenance := citylib.DifficultyTable[chewDifficulty(services)].Maintenance

    ownUnits := func(summoned bool) []chewOwnUnit {
        var out []chewOwnUnit
        for _, stack := range self.Stacks {
            wp := chewPlaneIndex(stack.Plane())
            landmass := world.LandmassAt(stack.X(), stack.Y(), wp)
            if landmass == 0 {
                continue
            }
            for _, unit := range stack.Units() {
                if (unit.GetRace() == data.RaceFantastic) == summoned {
                    out = append(out, chewOwnUnit{Unit: unit, Landmass: landmass, Plane: wp})
                }
            }
        }
        return out
    }
    weakest := func(list []chewOwnUnit) (units.StackUnit, bool) {
        var pick units.StackUnit
        lowest := 10000
        for _, candidate := range list {
            value := chewUnitStrength(candidate.Unit) / 10
            if ratios[candidate.Plane][candidate.Landmass] == 0 {
                value *= 2
            }
            if value < lowest {
                lowest = value
                pick = candidate.Unit
            }
        }
        return pick, pick != nil
    }
    remove := func(list []chewOwnUnit, unit units.StackUnit) []chewOwnUnit {
        var out []chewOwnUnit
        for _, candidate := range list {
            if candidate.Unit != unit {
                out = append(out, candidate)
            }
        }
        return out
    }

    normal := ownUnits(false)
    for tries := 0; tries < 200 && (foodDeficit > 0 || goldDeficit > 0); tries++ {
        unit, ok := weakest(normal)
        if !ok {
            break
        }
        foodDeficit -= 1
        goldDeficit -= max(unit.GetUpkeepGold() * maintenance / 100, 1)
        normal = remove(normal, unit)
        chewSpellLog(self, "disbands %v for gold and food", unit.GetName())
        self.RemoveUnit(unit)
    }
    summoned := ownUnits(true)
    for tries := 0; tries < 200 && manaDeficit > 0; tries++ {
        unit, ok := weakest(summoned)
        if !ok {
            break
        }
        manaDeficit -= max(unit.GetUpkeepMana() * maintenance / 100, 1)
        summoned = remove(summoned, unit)
        chewSpellLog(self, "disbands %v for mana", unit.GetName())
        self.RemoveUnit(unit)
    }
}

// AI_Kill_Excess_Settlers_And_Engineers: one settler a landmass, the others go; after turn 200 one
// engineer a landmass too
func (ai *ChewbotAI) killExcessBuilders(self *playerlib.Player, world *chewWorld, turn int) []units.StackUnit {
    settlers := make(map[[2]int]int)
    engineers := make(map[[2]int]int)
    var gone []units.StackUnit
    for _, stack := range self.Stacks {
        wp := chewPlaneIndex(stack.Plane())
        landmass := world.LandmassAt(stack.X(), stack.Y(), wp)
        if landmass == 0 {
            continue
        }
        key := [2]int{wp, landmass}
        for _, unit := range stack.Units() {
            if unit.HasAbility(data.AbilityCreateOutpost) {
                settlers[key] += 1
                if settlers[key] > 1 {
                    gone = append(gone, unit)
                }
            }
            if unit.HasAbility(data.AbilityConstruction) && turn > 200 {
                engineers[key] += 1
                if engineers[key] > 1 {
                    gone = append(gone, unit)
                }
            }
        }
    }
    for _, unit := range gone {
        chewSpellLog(self, "disbands %v, one too many on its landmass", unit.GetName())
        self.RemoveUnit(unit)
    }
    return gone
}

// AI_Hopeless_Stasis: a unit of a computer player in Stasis with a resistance below 7 dies
func chewHopelessStasis(self *playerlib.Player) {
    var gone []units.StackUnit
    for _, stack := range self.Stacks {
        for _, unit := range stack.Units() {
            if unit.GetBusy() == units.BusyStatusStasis && unit.GetResistance() < chewStasisHopeless {
                gone = append(gone, unit)
            }
        }
    }
    for _, unit := range gone {
        self.RemoveUnit(unit)
    }
}
