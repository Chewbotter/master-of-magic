package maplib

// The worlds of the original game: 60 by 40 squares whatever the Land Size, which says how much of
// it is land (terrain/classic.go), with the original's nodes and towers, and its lairs. Facts from
// the ReMoM project's reconstruction (MoM/src/MAPGEN.c: Generate_Lairs). The code is ours.
//
//   lairs: 25 strong and 32 weak, each on a plane by chance, 3 to 56 across and 3 to 36 down, on
//     land, not next to another lair, node or tower
//   guardians and the budget of treasure of lairs, towers and nodes: classiclairs.go
//
// Cities, minerals, roads and rivers come later, in the main package (classiccities.go,
// classicspecials.go). What is still the fork's: the treasure itself, rolled on conquest from the
// budget (game/treasure.go).

import (
    "image"
    "math/rand/v2"
    "slices"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/terrain"
    "github.com/kazzmir/master-of-magic/lib/set"
)

// false: the fork's worlds, as large as the Land Size
var ClassicMaps = true

const (
    classicStrongLairs = 25
    classicWeakLairs = 32
    // tries for one lair before it is left out
    classicLairTries = 1000
)

// both worlds of the original's kind: the maps of Arcanus and Myrror, with the original's towers,
// nodes and lairs and their guardians (classiclairs.go)
func MakeClassicMaps(terrainData *terrain.TerrainData, landSize int, magicSetting data.MagicSetting, difficulty data.DifficultySetting, cityProvider CityProvider) (*Map, *Map) {
    arcanus, myrror, towers, nodes := terrain.GenerateClassicWorld(landSize, terrainData)
    maps := [2]*Map{
        MakeMapFromTerrain(arcanus, terrainData, magicSetting, difficulty, data.PlaneArcanus, cityProvider, towers),
        MakeMapFromTerrain(myrror, terrainData, magicSetting, difficulty, data.PlaneMyrror, cityProvider, towers),
    }
    planes := [2]data.Plane{data.PlaneArcanus, data.PlaneMyrror}

    // a tower has one set of guardians and treasure, the same on both planes
    for _, tower := range towers {
        encounter := classicTowerEncounter(difficulty)
        for _, mapObject := range maps {
            copied := *encounter
            copied.Units = slices.Clone(encounter.Units)
            copied.ExploredBy = set.MakeSet[Wizard]()
            mapObject.ExtraMap[tower][ExtraKindEncounter] = &copied
        }
    }

    for wp, mapObject := range maps {
        for _, node := range nodes[wp] {
            point := image.Pt(node.X, node.Y)
            magicNode, encounter := classicNodeEncounter(node, magicSetting, difficulty, planes[wp])
            mapObject.ExtraMap[point][ExtraKindMagicNode] = magicNode
            mapObject.ExtraMap[point][ExtraKindEncounter] = encounter
        }
    }

    placeClassicLairs(maps, difficulty)
    return maps[0], maps[1]
}

// Generate_Lairs: 25 strong lairs, then 32 weak ones, each on a plane by chance
func placeClassicLairs(maps [2]*Map, difficulty data.DifficultySetting) {
    planes := [2]data.Plane{data.PlaneArcanus, data.PlaneMyrror}
    free := func(mapObject *Map, x int, y int) bool {
        map_ := mapObject.Map
        tile := mapObject.Data.Tiles[map_.Terrain[x][y]].Tile
        if !tile.IsLand() || tile.IsMagic() {
            return false
        }
        for dx := -1; dx <= 1; dx++ {
            for dy := -1; dy <= 1; dy++ {
                point := image.Pt(map_.WrapX(x + dx), y + dy)
                if _, has := mapObject.ExtraMap[point][ExtraKindEncounter]; has {
                    return false
                }
                if point.Y >= 0 && point.Y < map_.Rows() && mapObject.Data.Tiles[map_.Terrain[point.X][point.Y]].Tile.IsMagic() {
                    return false
                }
            }
        }
        return true
    }
    place := func(weak bool) {
        wp := rand.N(2)
        mapObject := maps[wp]
        for range classicLairTries {
            x := 2 + rand.N(54) + 1
            y := 2 + rand.N(34) + 1
            if x >= mapObject.Map.Columns() || y >= mapObject.Map.Rows() || !free(mapObject, x, y) {
                continue
            }
            mapObject.ExtraMap[image.Pt(x, y)][ExtraKindEncounter] = classicLairEncounter(weak, difficulty, planes[wp])
            return
        }
    }
    for range classicStrongLairs {
        place(false)
    }
    for range classicWeakLairs {
        place(true)
    }
}
