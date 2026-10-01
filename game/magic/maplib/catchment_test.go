package maplib

import (
    "image"
    "math/rand/v2"
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/terrain"
)

// the rule of a shared square written out: another city within the area of 21 squares of it
func sharedByRule(mapObject *Map, cities map[image.Point]bool, x int, y int, tileX int, tileY int) bool {
    for dx := -2; dx <= 2; dx++ {
        for dy := -2; dy <= 2; dy++ {
            if (dx == 2 || dx == -2) && (dy == 2 || dy == -2) {
                continue
            }
            otherX := mapObject.WrapX(tileX + dx)
            otherY := tileY + dy
            if otherX == x && otherY == y {
                continue
            }
            if cities[image.Pt(otherX, otherY)] {
                return true
            }
        }
    }
    return false
}

// the area found with the cities around looked up once is the one of the rule, at the edges of a
// world that goes around too
func TestCatchmentSharedSquares(test *testing.T) {
    provider := TestCityProvider{Cities: make(map[image.Point]bool)}
    terrainData := terrain.MakeTerrainData([]image.Image{nil}, []terrain.TerrainTile{terrain.TerrainTile{TileIndex: 0, Tile: terrain.TileLand}})
    mapObject := Map{
        Data: terrainData,
        Plane: data.PlaneArcanus,
        Map: terrain.MakeMap(20, 16),
        CityProvider: &provider,
        ExtraMap: make(map[image.Point]map[ExtraKind]ExtraTile),
    }
    width, height := mapObject.Width(), mapObject.Height()
    random := rand.New(rand.NewPCG(1, 2))
    for round := range 200 {
        provider.Cities = make(map[image.Point]bool)
        for range 1 + random.IntN(8) {
            provider.Cities[image.Pt(random.IntN(width), random.IntN(height))] = true
        }
        x, y := random.IntN(width), random.IntN(height)
        if round % 4 == 0 {
            x = []int{0, 1, width - 2, width - 1}[round / 4 % 4]
        }
        for point, tile := range mapObject.GetCatchmentArea(x, y) {
            want := sharedByRule(&mapObject, provider.Cities, x, y, point.X, point.Y)
            if tile.IsShared != want {
                test.Fatalf("round %v: city at %v,%v, square %v shared %v, the rule says %v (cities %v)", round, x, y, point, tile.IsShared, want, provider.Cities)
            }
        }
    }
}
