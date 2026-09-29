package game

// What the world map around a battle tells the ground of its battlefield: forest, hills and the
// roads around it (see combat/terrain.go). The original reads the same in Build_Battlefield (ReMoM).

import (
    "image"

    "github.com/kazzmir/master-of-magic/game/magic/combat"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/maplib"
    "github.com/kazzmir/master-of-magic/game/magic/mod"
    "github.com/kazzmir/master-of-magic/game/magic/terrain"
)

// true if one of the 8 tiles around a tile of the world map is of a kind
func (game *Game) hasNeighbor(mapObject *maplib.Map, x int, y int, kind terrain.TerrainType) bool {
    for dy := -1; dy <= 1; dy++ {
        for dx := -1; dx <= 1; dx++ {
            if dx == 0 && dy == 0 || y + dy < 0 || y + dy >= mapObject.Height() {
                continue
            }
            if mapObject.GetTile(mapObject.WrapX(x + dx), y + dy).Tile.TerrainType() == kind {
                return true
            }
        }
    }
    return false
}

// the side of a tile of land the sea lies on, see combat/coast.go. of the three tiles of every
// side the most of them sea decide, the tile in the middle of a side counting double. a lake is
// sea for this: a battle beside one has a coast (user, 2026-09-29: "skip lakes and just use coast")
func (game *Game) coastSide(mapObject *maplib.Map, x int, y int) combat.CoastSide {
    sea := func(dx int, dy int) int {
        if y + dy < 0 || y + dy >= mapObject.Height() {
            return 0
        }
        switch mapObject.GetTile(mapObject.WrapX(x + dx), y + dy).Tile.TerrainType() {
            case terrain.Ocean, terrain.Shore, terrain.Lake: return 1
        }
        return 0
    }

    if !mapObject.GetTile(x, y).Tile.IsLand() {
        return combat.CoastNone
    }

    sides := []struct {
        Side combat.CoastSide
        Sea int
    }{
        {Side: combat.CoastEast, Sea: sea(1, -1) + sea(1, 0) * 2 + sea(1, 1)},
        {Side: combat.CoastNorth, Sea: sea(-1, -1) + sea(0, -1) * 2 + sea(1, -1)},
        {Side: combat.CoastWest, Sea: sea(-1, -1) + sea(-1, 0) * 2 + sea(-1, 1)},
        {Side: combat.CoastSouth, Sea: sea(-1, 1) + sea(0, 1) * 2 + sea(1, 1)},
    }

    out := combat.CoastNone
    most := 0
    for _, side := range sides {
        if side.Sea > most {
            out = side.Side
            most = side.Sea
        }
    }
    return out
}

// how many tiles a tile is from the nearest town that has it among the tiles it works, 1 or 2, and
// the race of that town. 0 if there is none. an outpost has no fields. see combat/farmland.go
func (game *Game) nearTown(mapObject *maplib.Map, x int, y int, plane data.Plane) (int, data.Race) {
    near := combat.FarmlandNone
    var race data.Race

    for dy := -2; dy <= 2; dy++ {
        for dx := -2; dx <= 2; dx++ {
            away := max(dx, -dx, dy, -dy)
            // a town works the tiles within 2 of it but for the corners
            if away == 0 || dx * dx + dy * dy == 8 || y + dy < 0 || y + dy >= mapObject.Height() {
                continue
            }
            if near != combat.FarmlandNone && away >= near {
                continue
            }

            for _, player := range game.Model.Players {
                city := player.FindCity(mapObject.WrapX(x + dx), y + dy, plane)
                if city != nil && !city.Outpost {
                    near = away
                    race = city.Race
                }
            }
        }
    }

    return near, race
}

func (game *Game) combatGround(x int, y int, plane data.Plane) combat.ZoneGround {
    mapObject := game.GetMap(plane)

    var out combat.ZoneGround

    switch mapObject.GetTile(x, y).Tile.TerrainType() {
        // the original counts a node of nature as forest
        case terrain.Forest, terrain.NatureNode: out.SetBiome(mod.BiomeForest)
        case terrain.Hill: out.SetBiome(mod.BiomeHills)
        case terrain.Swamp: out.SetBiome(mod.BiomeSwamp)
        case terrain.Volcano: out.SetBiome(mod.BiomeVolcano)
        case terrain.Mountain:
            if game.hasNeighbor(mapObject, x, y, terrain.Tundra) {
                out.SetBiome(mod.BiomeSnowyMountain)
            }
    }

    out.Coast = game.coastSide(mapObject, x, y)

    // the fields of a town. the battle decides what it shows of them
    if mapObject.GetTile(x, y).Tile.IsLand() {
        out.Farmland, out.FarmRace = game.nearTown(mapObject, x, y, plane)
    }

    // a town counts as a road, as it does for the roads of the world map
    hasRoad := func(roadX int, roadY int) bool {
        return mapObject.ContainsRoad(roadX, roadY) || mapObject.CityProvider.ContainsCity(roadX, roadY, plane)
    }

    for dy := -1; dy <= 1; dy++ {
        for dx := -1; dx <= 1; dx++ {
            roadY := y + dy
            if roadY < 0 || roadY >= mapObject.Height() {
                continue
            }
            out.Roads[(dy + 1) * 3 + dx + 1] = hasRoad(mapObject.WrapX(x + dx), roadY)
        }
    }

    road, ok := mapObject.ExtraMap[image.Pt(x, y)][maplib.ExtraKindRoad].(*maplib.ExtraRoad)
    if ok {
        out.EnchantedRoads = road.Enchanted
    }

    return out
}
