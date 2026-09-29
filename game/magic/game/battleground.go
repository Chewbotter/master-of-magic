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
