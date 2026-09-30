package main

// Minerals, roads and rivers of the original's worlds, made after all cities are placed as the
// original does (ReMoM MoM/src/MAPGEN.c: Generate_Terrain_Specials, Desert_Terrain_Special,
// Hills_Terrain_Special, Mountain_Terrain_Special, Generate_Roads, Init_New_Game, Generate_River;
// MoX/src/special.c: Path_Wrap, Range). The rules in words: docs/mod/worlds.md. The code is ours.
//
//   minerals: one try in every cell of a grid, 4 squares on Arcanus and 3 on Myrror, at the corner
//     of the cell plus 1 to twice the cell across and down; not on a square with a mineral, lair,
//     tower, node or city; grassland always and forest one time in 2 becomes mountain (3 in 7),
//     hills (3 in 7) or swamp (1 in 7); then forest has wild game, swamp nightshade, and mountain,
//     hills and desert a mineral by the tables below (Myrror only rolls their upper part)
//   roads: every two neutral cities of a plane on one landmass within a Range of 10 (the larger
//     distance and half the smaller, not around the world) get a straight road between them unless
//     it crosses the sea; every city has a road on its square; roads on Myrror are enchanted

import (
    "image"
    "log"
    "math/rand/v2"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/maplib"
    "github.com/kazzmir/master-of-magic/game/magic/terrain"
    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    gamelib "github.com/kazzmir/master-of-magic/game/magic/game"
)

const (
    // Generate_Terrain_Specials: the test that keeps minerals off the edges joins its parts with
    // "and", so it never keeps any off; kept
    quirkSpecialsAtEdges = true
    // Generate_Roads: roads only between cities within this Range
    classicRoadRange = 11
)

// the original's Random(n) for the specials: 1 to n
var classicSpecialRoll = func(n int) int {
    return rand.N(max(n, 1)) + 1
}

// a table of minerals: a roll up to Top (Arcanus) picks the first entry it is at most; Myrror rolls
// From + 1 to Top
type classicSpecialTable struct {
    Top int
    MyrrorFrom int
    Entries []classicSpecialEntry
}

type classicSpecialEntry struct {
    Upto int
    Bonus data.BonusType
}

func (table classicSpecialTable) pick(plane data.Plane) data.BonusType {
    var roll int
    if plane == data.PlaneMyrror {
        roll = table.MyrrorFrom + classicSpecialRoll(table.Top - table.MyrrorFrom)
    } else {
        roll = classicSpecialRoll(table.Top)
    }
    for _, entry := range table.Entries {
        if roll <= entry.Upto {
            return entry.Bonus
        }
    }
    return data.BonusNone
}

// Mountain_Terrain_Special: Arcanus 1 to 18; Myrror 11 to 20
// (Arcanus: iron 4, silver 3, coal 5, gold 3, mithril 3 of 18; Myrror: iron, silver, coal 1,
// gold 2, mithril 3, adamantium 2 of 10)
var classicMountainSpecials = classicSpecialTable{Top: 28, MyrrorFrom: 18, Entries: []classicSpecialEntry{
    {4, data.BonusIronOre}, {7, data.BonusSilverOre}, {12, data.BonusCoal}, {15, data.BonusGoldOre}, {18, data.BonusMithrilOre},
    {19, data.BonusIronOre}, {20, data.BonusSilverOre}, {21, data.BonusCoal}, {23, data.BonusGoldOre}, {26, data.BonusMithrilOre}, {28, data.BonusAdamantiumOre},
}}

// Hills_Terrain_Special (Arcanus: iron 6, silver 4, coal 3, gold 4, mithril 1 of 18; Myrror: iron,
// silver, coal 1, gold 4, mithril 2, adamantium 1 of 10)
var classicHillsSpecials = classicSpecialTable{Top: 28, MyrrorFrom: 18, Entries: []classicSpecialEntry{
    {6, data.BonusIronOre}, {10, data.BonusSilverOre}, {13, data.BonusCoal}, {17, data.BonusGoldOre}, {18, data.BonusMithrilOre},
    {19, data.BonusIronOre}, {20, data.BonusSilverOre}, {21, data.BonusCoal}, {25, data.BonusGoldOre}, {27, data.BonusMithrilOre}, {28, data.BonusAdamantiumOre},
}}

// Desert_Terrain_Special (Arcanus: gems 6, quork 3 of 9; Myrror: gems 1, quork 3, crysx 1 of 5)
var classicDesertSpecials = classicSpecialTable{Top: 14, MyrrorFrom: 9, Entries: []classicSpecialEntry{
    {6, data.BonusGem}, {9, data.BonusQuorkCrystal},
    {10, data.BonusGem}, {13, data.BonusQuorkCrystal}, {14, data.BonusCrysxCrystal},
}}

// the tables are written as one roll over both planes: Arcanus rolls 1 to MyrrorFrom, Myrror
// MyrrorFrom + 1 to Top
func (table classicSpecialTable) pickFor(plane data.Plane) data.BonusType {
    if plane == data.PlaneMyrror {
        return table.pick(plane)
    }
    return classicSpecialTable{Top: table.MyrrorFrom, Entries: table.Entries}.pick(plane)
}

func classicCitiesOn(game *gamelib.Game, plane data.Plane) []*citylib.City {
    var out []*citylib.City
    for _, city := range game.Model.AllCities() {
        if city.Plane == plane {
            out = append(out, city)
        }
    }
    return out
}

// Generate_Terrain_Specials for one plane
func classicSpecials(game *gamelib.Game, plane data.Plane) {
    mapObject := game.GetMap(plane)
    width, height := mapObject.Width(), mapObject.Height()
    stride := 4
    if plane == data.PlaneMyrror {
        stride = 3
    }
    cities := make(map[image.Point]bool)
    for _, city := range classicCitiesOn(game, plane) {
        cities[image.Pt(city.X, city.Y)] = true
    }

    changed := false
    for anchorY := 0; anchorY < height; anchorY += stride {
        for anchorX := 0; anchorX < width; anchorX += stride {
            y := anchorY + classicSpecialRoll(stride * 2)
            // past the right edge the original runs into the next row, past the bottom into the
            // other plane: here x goes around the world and a square below the map is none (MY CALL)
            x := mapObject.WrapX(anchorX + classicSpecialRoll(stride * 2))
            if y >= height {
                continue
            }
            if !quirkSpecialsAtEdges && (y <= 2 || y >= height - 2 || x <= 2 || x >= width - 2) {
                continue
            }
            point := image.Pt(x, y)
            extras := mapObject.ExtraMap[point]
            if _, has := extras[maplib.ExtraKindBonus]; has {
                continue
            }
            if _, has := extras[maplib.ExtraKindEncounter]; has {
                continue
            }
            if cities[point] {
                continue
            }

            kind := mapObject.GetTile(x, y).Tile.TerrainType()
            convert := kind == terrain.Grass
            if kind == terrain.Forest {
                convert = classicSpecialRoll(2) == 1
            }
            if convert {
                roll := classicSpecialRoll(7)
                switch {
                    case roll <= 3:
                        kind = terrain.Mountain
                        mapObject.Map.Terrain[x][y] = terrain.TileMountain1.Index(plane)
                    case roll <= 6:
                        kind = terrain.Hill
                        mapObject.Map.Terrain[x][y] = terrain.TileHills1.Index(plane)
                    default:
                        kind = terrain.Swamp
                        mapObject.Map.Terrain[x][y] = terrain.TileSwamp1.Index(plane)
                }
                changed = true
            }

            bonus := data.BonusNone
            switch kind {
                case terrain.Forest: bonus = data.BonusWildGame
                case terrain.Swamp: bonus = data.BonusNightshade
                case terrain.Mountain: bonus = classicMountainSpecials.pickFor(plane)
                case terrain.Hill: bonus = classicHillsSpecials.pickFor(plane)
                case terrain.Desert: bonus = classicDesertSpecials.pickFor(plane)
            }
            if bonus != data.BonusNone {
                if extras == nil {
                    extras = make(map[maplib.ExtraKind]maplib.ExtraTile)
                    mapObject.ExtraMap[point] = extras
                }
                mapObject.SetBonus(x, y, bonus)
            }
        }
    }
    if changed {
        mapObject.Map.ResolveTiles(mapObject.Data, plane)
    }
}

// Range (classiccities.go), not around the world: the original's slip in Generate_Roads, kept, so
// cities across the edge of the map get no road

// Path_Wrap: the squares of a straight line from one square to another, the shorter way around the
// world, without the first and with the last
func classicLine(x1 int, y1 int, x2 int, y2 int, width int) []image.Point {
    dx := x2 - x1
    if dx > width / 2 {
        dx -= width
    }
    if dx < -width / 2 {
        dx += width
    }
    dy := y2 - y1
    stepX, stepY := 1, 1
    if dx < 0 {
        stepX = -1
        dx = -dx
    }
    if dy < 0 {
        stepY = -1
        dy = -dy
    }
    // the original's fraction of 16 bits, starting at one half
    steps := max(dx, dy)
    var out []image.Point
    x, y := x1, y1
    carry := 0x8000
    add := 0
    if steps > 0 {
        add = (min(dx, dy) << 16) / steps
    }
    for range steps {
        carry += add
        stepMinor := carry >= 0x10000
        carry &= 0xFFFF
        switch {
            case dx == dy:
                x += stepX
                y += stepY
            case dx > dy:
                x += stepX
                if stepMinor {
                    y += stepY
                }
            default:
                y += stepY
                if stepMinor {
                    x += stepX
                }
        }
        out = append(out, image.Pt(((x % width) + width) % width, y))
    }
    return out
}

// Generate_Roads for one plane
func classicRoads(game *gamelib.Game, plane data.Plane) {
    mapObject := game.GetMap(plane)
    width := mapObject.Width()
    enchanted := plane == data.PlaneMyrror

    landmass := make(map[image.Point]int)
    for index, continent := range mapObject.Map.FindContinents() {
        for _, point := range continent.Values() {
            landmass[point] = index + 1
        }
    }

    var neutral []*citylib.City
    for _, player := range game.Model.Players {
        if !player.IsNeutral() {
            continue
        }
        for _, city := range player.Cities {
            if city.Plane == plane {
                neutral = append(neutral, city)
            }
        }
    }

    for first := 0; first < len(neutral); first++ {
        for second := first + 1; second < len(neutral); second++ {
            from, to := neutral[first], neutral[second]
            if landmass[image.Pt(from.X, from.Y)] != landmass[image.Pt(to.X, to.Y)] {
                continue
            }
            if classicRange(from.X, from.Y, to.X, to.Y) >= classicRoadRange {
                continue
            }
            line := classicLine(from.X, from.Y, to.X, to.Y, width)
            inner := line[:max(len(line) - 1, 0)]
            blocked := false
            for _, point := range inner {
                if !mapObject.GetTile(point.X, point.Y).Tile.IsLand() {
                    blocked = true
                    break
                }
            }
            if blocked {
                continue
            }
            for _, point := range inner {
                mapObject.SetRoad(point.X, point.Y, enchanted)
            }
        }
    }

    // a road under every city (on Myrror enchanted: the original writes the flag of the enchanted
    // road beside the square by a slip, but moves on every road of Myrror as on an enchanted one)
    for _, city := range classicCitiesOn(game, plane) {
        mapObject.SetRoad(city.X, city.Y, enchanted)
    }
}

// minerals, roads and rivers of both planes, after all cities
func classicSpecialsAndRoads(game *gamelib.Game) {
    for _, plane := range []data.Plane{data.PlaneArcanus, data.PlaneMyrror} {
        classicSpecials(game, plane)
    }
    for _, plane := range []data.Plane{data.PlaneArcanus, data.PlaneMyrror} {
        classicRoads(game, plane)
    }
    classicRivers(game)
    for _, plane := range []data.Plane{data.PlaneArcanus, data.PlaneMyrror} {
        classicShuffle(game.GetMap(plane), plane)
    }
}

// Shuffle_Terrains: the plain pictures of grassland (4), forest (3), desert (4) and swamp (3) become
// one of their look-alikes by chance; plain tundra stays two times in three and becomes the second
// picture one time in three (the original's switch misses the third; kept)
func classicShuffle(mapObject *maplib.Map, plane data.Plane) {
    variants := map[int][]int{}
    add := func(first terrain.Tile, others ...terrain.Tile) {
        indexes := []int{first.Index(plane)}
        for _, other := range others {
            indexes = append(indexes, other.Index(plane))
        }
        variants[first.Index(plane)] = indexes
    }
    add(terrain.TileGrasslands1, terrain.TileGrasslands4, terrain.TileGrasslands2, terrain.TileGrasslands3)
    add(terrain.TileForest1, terrain.TileForest2, terrain.TileForest3)
    add(terrain.TileAllDesert1, terrain.TileAllDesert2, terrain.TileAllDesert3, terrain.TileAllDesert4)
    add(terrain.TileSwamp1, terrain.TileSwamp2, terrain.TileSwamp3)
    tundra := terrain.TileAllTundra1.Index(plane)
    variants[tundra] = []int{tundra, tundra, terrain.TileAllTundra2.Index(plane)}

    for x := 0; x < mapObject.Width(); x++ {
        for y := 0; y < mapObject.Height(); y++ {
            if choices, has := variants[mapObject.Map.Terrain[x][y]]; has {
                mapObject.Map.Terrain[x][y] = choices[classicSpecialRoll(len(choices)) - 1]
            }
        }
    }
}

// Init_New_Game: 10 rivers a plane, in turn, each tried up to 2000 times (terrain.ClassicRiver). A
// river does not go on a mineral (the original's rule) nor on a lair, tower or city (MY CALL: the
// original lets it, and the square becomes river under them)
func classicRivers(game *gamelib.Game) {
    planes := []data.Plane{data.PlaneArcanus, data.PlaneMyrror}
    blocked := make(map[data.Plane]terrain.RiverBlocked)
    for _, plane := range planes {
        mapObject := game.GetMap(plane)
        cities := make(map[image.Point]bool)
        for _, city := range classicCitiesOn(game, plane) {
            cities[image.Pt(city.X, city.Y)] = true
        }
        blocked[plane] = func(x int, y int) bool {
            extras := mapObject.ExtraMap[image.Pt(x, y)]
            if _, has := extras[maplib.ExtraKindBonus]; has {
                return true
            }
            if _, has := extras[maplib.ExtraKindEncounter]; has {
                return true
            }
            return cities[image.Pt(x, y)]
        }
    }
    made := make(map[data.Plane]int)
    for range terrain.ClassicRivers {
        for _, plane := range planes {
            mapObject := game.GetMap(plane)
            for range terrain.ClassicRiverTries {
                if mapObject.Map.ClassicRiver(mapObject.Data, plane, blocked[plane]) {
                    made[plane] += 1
                    break
                }
            }
        }
    }
    log.Printf("rivers: %v on Arcanus, %v on Myrror of %v each", made[data.PlaneArcanus], made[data.PlaneMyrror], terrain.ClassicRivers)
    for _, plane := range planes {
        mapObject := game.GetMap(plane)
        mapObject.Map.ResolveTiles(mapObject.Data, plane)
    }
}
