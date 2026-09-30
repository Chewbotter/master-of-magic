package maplib

// The worlds of the original game: 60 by 40 squares whatever the Land Size, which says how much of
// it is land (terrain/classic.go), with the original's nodes and towers, and its lairs. Facts from
// the ReMoM project's reconstruction (MoM/src/MAPGEN.c: Generate_Lairs). The code is ours.
//
//   lairs: 25 strong and 32 weak, each on a plane by chance, 3 to 56 across and 3 to 36 down, on
//     land, not next to another lair, node or tower
//
// What is still the fork's: the insides of nodes, lairs and towers (their guardians and treasure),
// minerals, rivers, the cities (placed in main.go), roads.

import (
    "image"
    "math/rand/v2"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/terrain"
)

// false: the fork's worlds, as large as the Land Size
var ClassicMaps = true

const (
    classicStrongLairs = 25
    classicWeakLairs = 32
    // tries for one lair before it is left out
    classicLairTries = 1000
)

// both worlds of the original's kind: the maps of Arcanus and Myrror
func MakeClassicMaps(terrainData *terrain.TerrainData, landSize int, magicSetting data.MagicSetting, difficulty data.DifficultySetting, cityProvider CityProvider) (*Map, *Map) {
    arcanus, myrror, towers := terrain.GenerateClassicWorld(landSize, terrainData)
    return MakeMapFromTerrain(arcanus, terrainData, magicSetting, difficulty, data.PlaneArcanus, cityProvider, towers),
        MakeMapFromTerrain(myrror, terrainData, magicSetting, difficulty, data.PlaneMyrror, cityProvider, towers)
}

// Generate_Lairs for one plane: every one of the original's lairs is on this plane one time in 2
func placeClassicLairs(map_ *terrain.Map, terrainData *terrain.TerrainData, extraMap map[image.Point]map[ExtraKind]ExtraTile, difficulty data.DifficultySetting, plane data.Plane) {
    free := func(x int, y int) bool {
        tile := terrainData.Tiles[map_.Terrain[x][y]].Tile
        if !tile.IsLand() || tile.IsMagic() {
            return false
        }
        for dx := -1; dx <= 1; dx++ {
            for dy := -1; dy <= 1; dy++ {
                point := image.Pt(map_.WrapX(x + dx), y + dy)
                if _, has := extraMap[point][ExtraKindEncounter]; has {
                    return false
                }
                if point.Y >= 0 && point.Y < map_.Rows() && terrainData.Tiles[map_.Terrain[point.X][point.Y]].Tile.IsMagic() {
                    return false
                }
            }
        }
        return true
    }
    place := func(weak bool) {
        for range classicLairTries {
            x := 2 + rand.N(54) + 1
            y := 2 + rand.N(34) + 1
            if x >= map_.Columns() || y >= map_.Rows() || !free(x, y) {
                continue
            }
            point := image.Pt(x, y)
            if extraMap[point] == nil {
                extraMap[point] = make(map[ExtraKind]ExtraTile)
            }
            extraMap[point][ExtraKindEncounter] = makeEncounter(randomEncounterType(), difficulty, weak, plane)
            return
        }
    }
    for range classicStrongLairs {
        if rand.N(2) == 0 {
            place(false)
        }
    }
    for range classicWeakLairs {
        if rand.N(2) == 0 {
            place(true)
        }
    }
}
