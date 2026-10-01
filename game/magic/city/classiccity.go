package city

// The rules of cities as the original game has them. Rules in words: docs/mod/cities.md.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/CITYCALC.c:
// City_Food_Terrain, City_Food_Production, City_Minimum_Farmers, City_Maximum_Size,
// City_Growth_Rate, City_Rebel_Count, City_Production_Production, City_Gold_Production,
// City_Research_Production, All_Outpost_Population_Growth; NEXTTURN.c: Apply_City_Changes;
// Terrain.c; RACETYPE.c; Spells130.c: Apply_Consecration; TERRSTAT.LBX entry 1 for the unrest
// of races). The code is ours.

import (
    rand "github.com/kazzmir/master-of-magic/lib/chance"
    "slices"

    buildinglib "github.com/kazzmir/master-of-magic/game/magic/building"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/maplib"
    "github.com/kazzmir/master-of-magic/game/magic/terrain"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    "github.com/kazzmir/master-of-magic/lib/fraction"
)

// false: upstream's rules of cities
var ClassicCities = true

// City_Rebel_Count: Great Wasting and Armageddon of another wizard SET the pacification to -1 and
// -2, which wipes out temples and garrison (kept, the original's)
const quirkWastingSetsPacification = true

// Apply_Consecration: the corruption is cleared on a block of 4 by 4 squares from 2 up and left
// of the city, not the city's area (kept, the original's)
const quirkConsecrationBlock = true

// the people a person is made of in the growth of the original (Pop_10s: units of 10 people)
const growthUnit = 10

// the races in the original's order (TERRSTAT, RACETYPE)
var classicRaceOrder = []data.Race{
    data.RaceBarbarian, data.RaceBeastmen, data.RaceDarkElf, data.RaceDraconian, data.RaceDwarf,
    data.RaceGnoll, data.RaceHalfling, data.RaceHighElf, data.RaceHighMen, data.RaceKlackon,
    data.RaceLizard, data.RaceNomad, data.RaceOrc, data.RaceTroll,
}

// TBL_Unrest: tens of percent, row the race of the capital, column the race of the city
var classicRaceUnrest = [14][14]int{
    {0, 1, 1, 1, 1, 1, 1, 1, 1, 2, 1, 0, 0, 1},
    {1, 0, 2, 2, 2, 0, 1, 2, 1, 2, 1, 1, 1, 2},
    {1, 2, 0, 2, 3, 2, 2, 4, 2, 2, 2, 2, 2, 3},
    {1, 2, 2, 0, 2, 1, 1, 1, 1, 2, 1, 1, 1, 2},
    {1, 2, 3, 2, 0, 1, 0, 3, 0, 2, 1, 0, 3, 4},
    {1, 0, 2, 1, 1, 0, 0, 1, 1, 2, 1, 1, 0, 0},
    {1, 1, 2, 1, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0},
    {1, 2, 4, 1, 3, 1, 0, 0, 0, 2, 1, 0, 2, 3},
    {1, 1, 2, 1, 0, 1, 0, 0, 0, 2, 1, 0, 0, 1},
    {2, 2, 2, 2, 2, 2, 2, 2, 2, -2, 2, 2, 2, 2},
    {1, 1, 2, 1, 1, 1, 0, 1, 1, 2, 0, 1, 1, 1},
    {0, 1, 2, 1, 0, 1, 0, 0, 0, 2, 1, 0, 0, 1},
    {0, 1, 2, 1, 3, 0, 0, 2, 0, 2, 1, 0, 0, 0},
    {1, 2, 3, 2, 4, 0, 0, 3, 1, 2, 1, 1, 0, 0},
}

// RACETYPE.c: growth in units of 10 people, and the outpost growth rate
var classicRaceGrowth = map[data.Race][2]int{
    data.RaceBarbarian: {2, 15}, data.RaceBeastmen: {0, 5}, data.RaceDarkElf: {-2, 2},
    data.RaceDraconian: {-1, 5}, data.RaceDwarf: {-2, 7}, data.RaceGnoll: {-1, 5},
    data.RaceHalfling: {0, 15}, data.RaceHighElf: {-2, 5}, data.RaceHighMen: {0, 10},
    data.RaceKlackon: {-1, 5}, data.RaceLizard: {1, 10}, data.RaceNomad: {-1, 10},
    data.RaceOrc: {0, 10}, data.RaceTroll: {-2, 3},
}

// the tax rate as the original's steps 0 to 6 (half a gold each), and its unrest
func classicTaxStep(rate fraction.Fraction) int {
    return int(rate.ToFloat() * 2 + 0.5)
}

var classicTaxUnrest = []int{0, 10, 20, 30, 45, 60, 75}

func classicRaceIndex(race data.Race) int {
    for i, check := range classicRaceOrder {
        if check == race {
            return i
        }
    }
    return -1
}

func (city *City) isNeutral() bool {
    owner, ok := city.ownerDifficulty()
    return ok && owner.Neutral
}

// City_Food_Terrain: the food of the land, shared squares half, Gaia's Blessing half again,
// rounded down once
func (city *City) classicTerrainFood() int {
    food := fraction.Zero()
    for _, tile := range city.GetCatchmentArea() {
        value := tile.FoodBonus()
        if tile.IsShared {
            value = value.Divide(fraction.FromInt(2))
        }
        food = food.Add(value)
    }
    if city.HasEnchantment(data.CityEnchantmentGaiasBlessing) {
        food = food.Multiply(fraction.Make(3, 2))
    }
    return int(food.ToFloat())
}

// City_Food_WildGame: 2 a square with wild game, 1 when shared
func (city *City) classicWildGame() int {
    food := 0
    for _, tile := range city.GetCatchmentArea() {
        if tile.GetBonus() == data.BonusWildGame {
            if tile.IsShared {
                food += 1
            } else {
                food += 2
            }
        }
    }
    return food
}

// City_Maximum_Size (the cap of 25 is in the growth)
func (city *City) classicMaxSize() int {
    size := city.classicTerrainFood()
    if city.HasEnchantment(data.CityEnchantmentFamine) {
        size /= 2
    }
    if city.Buildings.Contains(buildinglib.BuildingGranary) {
        size += 2
    }
    if city.Buildings.Contains(buildinglib.BuildingFarmersMarket) {
        size += 3
    }
    return size + city.classicWildGame()
}

// the food of the farmers before the buildings: 2 each, 3 for Halflings or with an Animists'
// Guild, Foresters' Guild 2, halved by Famine, over the food of the land only half counts
func (city *City) classicFarmedFood(farmers int) int {
    perFarmer := 2
    if city.Race == data.RaceHalfling || city.Buildings.Contains(buildinglib.BuildingAnimistsGuild) {
        perFarmer = 3
    }
    food := farmers * perFarmer
    if city.Buildings.Contains(buildinglib.BuildingForestersGuild) {
        food += 2
    }
    if city.HasEnchantment(data.CityEnchantmentFamine) {
        food /= 2
    }
    terrain := city.classicTerrainFood()
    if food > terrain {
        food = terrain + (food - terrain) / 2
    }
    return food
}

// City_Food_Production
func (city *City) classicFood(farmers int) int {
    if city.Citizens() == 0 {
        return 0
    }
    food := city.classicFarmedFood(min(farmers, city.Citizens()))
    if city.Buildings.Contains(buildinglib.BuildingGranary) {
        food += 2
    }
    if city.Buildings.Contains(buildinglib.BuildingFarmersMarket) {
        food += 3
    }
    return food + city.classicWildGame()
}

// City_Minimum_Farmers
func (city *City) classicMinimumFarmers() int {
    need := city.Citizens() - city.classicWildGame()
    if city.Buildings.Contains(buildinglib.BuildingGranary) {
        need -= 2
    }
    if city.Buildings.Contains(buildinglib.BuildingFarmersMarket) {
        need -= 3
    }
    if need <= 0 {
        return 0
    }
    for farmers := 1; farmers < city.Citizens(); farmers++ {
        if city.classicFarmedFood(farmers) >= need {
            return farmers
        }
    }
    return city.Citizens()
}

// City_Growth_Rate, in people
func (city *City) classicGrowth() int {
    people := city.Citizens()
    if people == 0 {
        return 0
    }
    most := city.classicMaxSize()
    // a city exactly at its maximum does not grow, and does not starve either (the original's)
    if people == most {
        return 0
    }
    if most == 0 {
        return -people * growthUnit
    }
    if food := city.FoodProductionRate(); food < people {
        return (food - people) * 5 * growthUnit
    }

    rate := (most - people + 1) / 2 + classicRaceGrowth[city.Race][0]
    if city.Buildings.Contains(buildinglib.BuildingGranary) {
        rate += 2
    }
    if city.Buildings.Contains(buildinglib.BuildingFarmersMarket) {
        rate += 3
    }
    if people >= MAX_CITY_CITIZENS {
        return 0
    }
    if city.HasEnchantment(data.CityEnchantmentStreamOfLife) {
        rate *= 2
    }
    if city.CityServices != nil && city.CityServices.PopulationBoomActive(city) {
        rate *= 2
    }

    percent := 0
    if city.HasEnchantment(data.CityEnchantmentDarkRituals) {
        percent -= 25
    }
    if city.ProducingBuilding == buildinglib.BuildingHousing {
        if people == 1 {
            percent += 50
        } else {
            percent += (people - city.Rebels - city.Farmers) * 100 / people
        }
        if city.Buildings.Contains(buildinglib.BuildingSawmill) {
            percent += 10
        }
        if city.Buildings.Contains(buildinglib.BuildingBuildersHall) {
            percent += 15
        }
    }
    rate = rate * (percent + 100) / 100

    if rate <= 0 {
        return 0
    }
    return max(0, city.difficultyGrowth(rate)) * growthUnit
}

// Apply_City_Changes: the growth goes into the tens of people of the city; a whole person more at
// 1000 (what is over is lost), one less below 0; one a turn at most; a city of 1 does not starve
// away (it is set to 50 people)
func (city *City) classicApplyGrowth(growth int) {
    people := city.Citizens()
    rest := city.Population - people * 1000 + growth

    most := MAX_CITY_CITIZENS
    if city.isNeutral() {
        most = 8
    }

    if rest >= 1000 {
        if people < most {
            people += 1
            rest = 0
        } else {
            rest = min(rest, 990)
        }
    } else if rest < 0 {
        if people <= 1 {
            rest = 50
        } else {
            people -= 1
            rest += 1000
        }
    }

    city.Population = people * 1000 + max(0, rest)
}

// City_Rebel_Count
func (city *City) classicRebels() int {
    people := city.Citizens()
    if people == 0 {
        return 0
    }

    percent := 0
    capital := classicRaceIndex(city.ReignProvider.GetRulingRace())
    own := classicRaceIndex(city.Race)
    if capital >= 0 && own >= 0 {
        percent += classicRaceUnrest[capital][own] * 10
    }
    step := classicTaxStep(city.ReignProvider.GetTaxRate())
    if step >= 0 && step < len(classicTaxUnrest) {
        percent += classicTaxUnrest[step]
    }
    if city.HasEnchantment(data.CityEnchantmentFamine) {
        percent += 25
    }
    rebels := people * percent / 100

    pacify := 0
    if !city.HasEnchantment(data.CityEnchantmentEvilPresence) || city.ReignProvider.HasDeathBooks() {
        religious := 0
        for _, building := range []buildinglib.Building{buildinglib.BuildingShrine, buildinglib.BuildingTemple, buildinglib.BuildingParthenon, buildinglib.BuildingCathedral} {
            if city.Buildings.Contains(building) {
                religious += 1
            }
        }
        if city.ReignProvider.HasDivinePower() || city.ReignProvider.HasInfernalPower() {
            religious += religious / 2
        }
        pacify += religious
    }
    if city.Buildings.Contains(buildinglib.BuildingOracle) {
        pacify += 2
    }
    if city.Buildings.Contains(buildinglib.BuildingAnimistsGuild) {
        pacify += 1
    }

    if city.CityServices != nil {
        for banner, enchantments := range city.CityServices.GetAllGlobalEnchantments() {
            if banner == city.ReignProvider.GetBanner() {
                continue
            }
            if enchantments.Contains(data.EnchantmentGreatWasting) {
                if quirkWastingSetsPacification {
                    pacify = -1
                } else {
                    pacify -= 1
                }
            }
            if enchantments.Contains(data.EnchantmentArmageddon) {
                if quirkWastingSetsPacification {
                    pacify = -2
                } else {
                    pacify -= 2
                }
            }
        }
    }

    if city.HasEnchantment(data.CityEnchantmentDarkRituals) {
        pacify -= 1
    }
    if city.HasEnchantment(data.CityEnchantmentPestilence) {
        pacify -= 2
    }
    if city.HasEnchantment(data.CityEnchantmentCursedLands) {
        pacify -= 1
    }
    if city.ReignProvider.GetGlobalEnchantments().Contains(data.EnchantmentJustCause) {
        pacify += 1
    }
    // MY CALL: the city's own Gaia's Blessing (the reconstruction reads a global slot, see
    // docs/mod/improvements.md)
    if city.HasEnchantment(data.CityEnchantmentGaiasBlessing) {
        pacify += 2
    }

    if pacify < rebels {
        garrison := 0
        for _, unit := range city.ReignProvider.GetUnits(city.X, city.Y, city.Plane) {
            if unit.GetRace() != data.RaceFantastic {
                garrison += 1
            }
        }
        pacify += garrison / 2
    }

    rebels = max(0, rebels - pacify)
    // MY CALL, as above: the city's own Stream of Life
    if city.HasEnchantment(data.CityEnchantmentStreamOfLife) {
        rebels = 0
    }
    return min(rebels, people)
}

// the farmers at least the minimum, the rebels from the workers first and then the farmers,
// even below the minimum (City_Screen_Draw_Population_Row)
func (city *City) classicSetCitizens() {
    people := city.Citizens()
    rebels := city.classicRebels()
    city.Farmers = max(city.Farmers, city.classicMinimumFarmers())
    city.Farmers = max(0, min(city.Farmers, people - rebels))
    city.Rebels = rebels
    city.Workers = max(0, people - city.Farmers - rebels)
}

// City_Production_Production, before the difficulty
func (city *City) classicProduction() int {
    perWorker := 4
    if city.Race == data.RaceDwarf || city.Race == data.RaceKlackon {
        perWorker = 6
    }
    base := (city.Workers * perWorker + city.Farmers + 1) / 2

    percent := 100
    gaia := city.HasEnchantment(data.CityEnchantmentGaiasBlessing)
    for _, tile := range city.GetCatchmentArea() {
        value := classicSquareProduction(tile, gaia)
        if tile.IsShared {
            value /= 2
        }
        percent += value
    }
    for building, more := range map[buildinglib.Building]int{
        buildinglib.BuildingForestersGuild: 25, buildinglib.BuildingSawmill: 25,
        buildinglib.BuildingMinersGuild: 50, buildinglib.BuildingMechaniciansGuild: 50,
    } {
        if city.Buildings.Contains(building) {
            percent += more
        }
    }
    if city.HasEnchantment(data.CityEnchantmentInspirations) {
        percent += 100
    }

    production := base * percent / 100
    if city.HasEnchantment(data.CityEnchantmentCursedLands) {
        production /= 2
    }
    return production
}

// Square_Gold_Income summed: 2 silver, 3 gold, 5 gems; Dwarves twice; Miners' Guild half again;
// shared squares half; each step rounded down
func (city *City) classicMineralGold() int {
    gold := 0
    for _, tile := range city.GetCatchmentArea() {
        value := tile.GetBonus().GoldBonus()
        if city.Race == data.RaceDwarf {
            value *= 2
        }
        if city.Buildings.Contains(buildinglib.BuildingMinersGuild) {
            value = value * 3 / 2
        }
        if tile.IsShared {
            value /= 2
        }
        gold += value
    }
    return gold
}

// City_Gold_Production, before the difficulty and the upkeep of buildings
func (city *City) classicGold() int {
    people := city.Citizens()
    trade := city.CatchmentProvider.GetGoldBonus(city.X, city.Y)
    if city.CityServices != nil {
        for _, other := range city.CityServices.FindRoadConnectedCities(city) {
            if other.Race == city.Race {
                trade += other.Citizens() / 2
            } else {
                trade += other.Citizens()
            }
        }
    }
    if city.Race == data.RaceNomad {
        trade += 50
    }
    trade = min(trade, 3 * people)

    for building, more := range map[buildinglib.Building]int{
        buildinglib.BuildingMerchantsGuild: 100, buildinglib.BuildingBank: 50, buildinglib.BuildingMarketplace: 50,
    } {
        if city.Buildings.Contains(building) {
            trade += more
        }
    }
    if city.HasEnchantment(data.CityEnchantmentProsperity) {
        trade += 100
    }

    taxes := (people - city.Rebels) * classicTaxStep(city.ReignProvider.GetTaxRate()) / 2
    gold := (taxes + city.classicMineralGold()) * (trade + 100) / 100
    if city.ProducingBuilding == buildinglib.BuildingTradeGoods {
        gold += int(city.WorkProductionRate()) / 2
    }
    return min(gold, 255)
}

// City_Research_Production
func (city *City) classicResearch() int {
    research := 0
    for building, more := range map[buildinglib.Building]int{
        buildinglib.BuildingLibrary: 2, buildinglib.BuildingSagesGuild: 3,
        buildinglib.BuildingUniversity: 5, buildinglib.BuildingWizardsGuild: 8,
    } {
        if city.Buildings.Contains(building) {
            research += more
        }
    }
    return research
}

// City_Current_Product_Cost: iron 5 (10 with a Miners' Guild), coal 10 (20); Dwarves twice;
// shared squares half; at most 50 percent
func (city *City) classicUnitCost(unit *units.Unit) int {
    if !unit.ProductionCostReduction {
        return unit.ProductionCost
    }
    reduction := 0
    for _, tile := range city.GetCatchmentArea() {
        value := tile.GetBonus().UnitReductionBonus()
        if city.Buildings.Contains(buildinglib.BuildingMinersGuild) {
            value *= 2
        }
        if city.Race == data.RaceDwarf {
            value *= 2
        }
        if tile.IsShared {
            value /= 2
        }
        reduction += value
    }
    reduction = min(reduction, 50)
    return unit.ProductionCost * (100 - reduction) / 100
}

// All_Outpost_Population_Growth: one roll to grow 1 to 3 houses, one to lose 1 or 2
func (city *City) classicGrowOutpost() CityEvent {
    grow := city.classicTerrainFood() + classicRaceGrowth[city.Race][1]
    if city.HasEnchantment(data.CityEnchantmentGaiasBlessing) {
        grow += 20
    }
    if city.HasEnchantment(data.CityEnchantmentStreamOfLife) {
        grow += 10
    }
    // MY CALL, the meaning of the original's (it reads the wrong square): each mineral of the area,
    // 5 for iron and silver, 10 for the others
    for _, tile := range city.GetCatchmentArea() {
        switch tile.GetBonus() {
            case data.BonusNone, data.BonusWildGame:
            case data.BonusIronOre, data.BonusSilverOre: grow += 5
            default: grow += 10
        }
    }
    grow = int(city.difficultyOutpostGrowth(float64(grow)))

    shrink := 5
    if city.HasEnchantment(data.CityEnchantmentEvilPresence) {
        shrink += 5
    }
    for _, curse := range []data.CityEnchantment{data.CityEnchantmentPestilence, data.CityEnchantmentFamine, data.CityEnchantmentChaosRift} {
        if city.HasEnchantment(curse) {
            shrink += 10
        }
    }

    // Random(100) is 1 to 100
    if rand.N(100) + 1 <= grow {
        city.Population += 100 * (rand.N(3) + 1)
    }
    if rand.N(100) + 1 <= shrink {
        city.Population -= 100 * (rand.N(2) + 1)
    }

    if city.Population < 100 {
        return &CityEventOutpostDestroyed{}
    } else if city.Population >= 1000 {
        city.Outpost = false
        return &CityEventOutpostHamlet{}
    }
    return nil
}

// Apply_Consecration: every turn the curses of the city end, and the corruption around it
func (city *City) classicConsecration(mapObject *maplib.Map) {
    city.RemoveEnchantments(data.CityEnchantmentChaosRift, data.CityEnchantmentEvilPresence, data.CityEnchantmentCursedLands, data.CityEnchantmentFamine, data.CityEnchantmentPestilence)
    if !quirkConsecrationBlock {
        for point := range city.GetCatchmentArea() {
            mapObject.RemoveCorruption(point.X, point.Y)
        }
        return
    }
    for dx := -2; dx < 2; dx++ {
        for dy := -2; dy < 2; dy++ {
            y := city.Y + dy
            if y < 0 || y >= mapObject.Height() {
                continue
            }
            mapObject.RemoveCorruption(mapObject.WrapX(city.X + dx), y)
        }
    }
}

// City_Building_Has_Requirement: the building of the city that needs this one, BuildingNone when
// none does (then it can be sold)
func (city *City) RequiredBy(building buildinglib.Building) buildinglib.Building {
    for _, other := range city.Buildings.Values() {
        for _, need := range city.BuildingInfo.Dependencies(other) {
            if need == building {
                return other
            }
        }
    }
    return buildinglib.BuildingNone
}

// City_Building_Is_Currently_Required: the name of what the city produces when it needs this
// building, else ""
func (city *City) ProductionNeeds(building buildinglib.Building) string {
    if !city.ProducingUnit.IsNone() {
        for _, need := range city.ProducingUnit.RequiredBuildings {
            if need == building {
                return city.ProducingUnit.Name
            }
        }
        return ""
    }
    if city.ProducingBuilding != buildinglib.BuildingNone {
        for _, need := range city.BuildingInfo.Dependencies(city.ProducingBuilding) {
            if need == building {
                return city.BuildingInfo.Name(city.ProducingBuilding)
            }
        }
    }
    return ""
}

// the index of a race in the original's order, -1 when none (events)
func ClassicRaceIndexOf(race data.Race) int {
    return classicRaceIndex(race)
}

// the unrest of a city's race under the capital's race, in tens of percent (TBL_Unrest)
func ClassicRaceUnrest(capital int, own int) int {
    return classicRaceUnrest[capital][own]
}

// the number of a unit in the original's table: units1 entries 0 to 119, units2 from 120
func classicUnitType(unit units.Unit) int {
    if unit.LbxFile == "units2.lbx" {
        return 120 + unit.Index
    }
    return unit.Index
}

// Calculate_Product_Array: the units in the table's order, at most 12; when there are 12, the
// list made again without Spearmen and Swordsmen
func (city *City) classicPossibleUnits() []units.Unit {
    all := city.AllPossibleUnits()
    slices.SortStableFunc(all, func(a units.Unit, b units.Unit) int {
        return classicUnitType(a) - classicUnitType(b)
    })
    take := func(skipBasic bool) []units.Unit {
        var out []units.Unit
        for _, unit := range all {
            if len(out) >= 12 {
                break
            }
            if skipBasic && (unit.Name == "Spearmen" || unit.Name == "Swordsmen") {
                continue
            }
            out = append(out, unit)
        }
        return out
    }
    out := take(false)
    if len(out) >= 12 {
        out = take(true)
    }
    return out
}

// Square_Production_Bonus: the plain inner desert (the first of its four pictures) gives nothing;
// a nature node 6 with Gaia's Blessing
const quirkPlainDesertBarren = true

func classicSquareProduction(tile maplib.FullTile, gaia bool) int {
    if tile.Corrupted() {
        return 0
    }
    switch tile.Tile.TerrainType() {
        case terrain.Desert:
            if quirkPlainDesertBarren && tile.Tile.Index(data.PlaneArcanus) == terrain.IndexDesert1 {
                return 0
            }
        case terrain.NatureNode:
            if gaia {
                return 6
            }
    }
    return tile.ProductionBonus(gaia)
}
