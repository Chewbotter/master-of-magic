package city

// What it costs to buy what a city produces: the gold, and the production that is bought. The
// formula of the city screen (cityview), so the computer players pay the same: the production
// still missing, times 4 when nothing is built yet, times 3 below a third, else times 2.

import (
    buildinglib "github.com/kazzmir/master-of-magic/game/magic/building"
)

// the gold and the production of buying. 0, 0 when there is nothing to buy
func (city *City) BuyCost() (int, float32) {
    cost := 0
    if !city.ProducingUnit.IsNone() {
        cost = city.UnitProductionCost(&city.ProducingUnit)
    } else if city.ProducingBuilding != buildinglib.BuildingNone && city.ProducingBuilding != buildinglib.BuildingTradeGoods && city.ProducingBuilding != buildinglib.BuildingHousing {
        cost = city.BuildingInfo.ProductionCost(city.ProducingBuilding)
    }

    remaining := float32(cost) - city.Production
    if cost <= 0 || remaining <= 0 {
        return 0, 0
    }

    // City_Can_Buy_Product: not what will be done in under 2 turns anyway
    if ClassicCities && city.ProducingTurnsLeft() < 2 {
        return 0, 0
    }

    modifier := float32(2)
    switch {
        case city.Production == 0: modifier = 4
        case city.Production / float32(cost) < 1.0 / 3: modifier = 3
    }

    return int(remaining * modifier), remaining
}
