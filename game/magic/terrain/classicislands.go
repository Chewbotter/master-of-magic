package terrain

// Extend_Islands and the rivers of the original's worlds (ReMoM MoM/src/MAPGEN.c: Extend_Islands,
// Init_New_Game, Generate_River; the rules in words: docs/mod/worlds.md). The code is ours.

import (
    "image"

    "github.com/kazzmir/master-of-magic/game/magic/data"
)

// Extend_Islands: the original means to grow the land around a node or tower that is alone in the
// sea in any direction, but works out the neighbor wrong, so only east, south and southwest ever
// grow; kept
const quirkIslandsGrowSouthEast = true

// Extend_Islands for one plane: a node or tower with the sea on all four sides, two times in three,
// tries 2 to 9 neighbors by chance: east and south become grassland, southwest too unless the
// squares south and west of it are both sea (the other neighbors stay)
func classicExtendIslands(kinds [][]classicKind, nodes []classicNode, towers []image.Point) {
    special := make(map[image.Point]bool)
    for _, node := range nodes {
        special[image.Pt(node.X, node.Y)] = true
    }
    for _, tower := range towers {
        special[tower] = true
    }
    sea := func(x int, y int) bool {
        if y < 0 || y >= ClassicHeight {
            return true
        }
        x = (x + ClassicWidth) % ClassicWidth
        return kinds[x][y] == classicOcean && !special[image.Pt(x, y)]
    }

    for y := 1; y < ClassicHeight - 1; y++ {
        for x := 1; x < ClassicWidth - 1; x++ {
            if !special[image.Pt(x, y)] {
                continue
            }
            if !sea(x + 1, y) || !sea(x - 1, y) || !sea(x, y + 1) || !sea(x, y - 1) {
                continue
            }
            if classicRoll(3) <= 1 {
                continue
            }
            attempts := 1 + classicRoll(8)
            for range attempts {
                dx := classicRoll(3) - 2
                dy := classicRoll(3) - 2
                if dx == 0 && dy == 0 {
                    continue
                }
                targetX, targetY := x + dx, y + dy
                if special[image.Pt(targetX, targetY)] {
                    continue
                }
                grow := true
                if quirkIslandsGrowSouthEast {
                    switch dy * 3 + dx + 1 {
                        case 2, 4:
                        case 3: grow = !(sea(targetX, targetY + 1) && sea(targetX - 1, targetY))
                        default: grow = false
                    }
                }
                if grow {
                    kinds[targetX][targetY] = classicGrass
                }
            }
        }
    }
}

// Generate_River: squares a river may not go on (a mineral, a lair, a city: given by the game)
type RiverBlocked func(x int, y int) bool

// the original's four ways: south, west, north, east, and the opposite of each (upstream)
var classicRiverStepX = []int{0, -1, 0, 1}
var classicRiverStepY = []int{1, 0, -1, 0}
var classicRiverUpstream = []int{2, 3, 0, 1}

const (
    // Generate_River: attempts of a walk, and the longest and shortest river
    classicRiverAttempts = 30
    classicRiverLongest = 28
    classicRiverShortest = 4
    // Init_New_Game: tries of one river
    ClassicRiverTries = 2000
)

func (map_ *Map) classicTerrain(x int, y int, data *TerrainData) TerrainType {
    return data.Tiles[map_.Terrain[x][y]].Tile.TerrainType()
}

func classicIsSea(kind TerrainType) bool {
    switch kind {
        case Ocean, Shore, Lake: return true
    }
    return false
}

// Generate_River: one try at a river on the map. It starts 5 to 56 across and 5 to 36 down away from
// the sea (all 8 around), not on mountain, hills, a node, a river or a blocked square; it goes one
// way, half the time turning (never back); mountain, hills, desert, nodes and blocked squares stop a
// step; it ends next to the sea or on another river, after 30 steps at most, 4 to 28 squares long.
// The original then refuses a mouth on a shore without a picture of a river's mouth; here a river
// whose squares or their neighbors find no picture is refused (as the fork's rivers are)
func (map_ *Map) ClassicRiver(data *TerrainData, plane data.Plane, blocked RiverBlocked) bool {
    width, height := map_.Columns(), map_.Rows()
    inside := func(y int) bool {
        return y >= 0 && y < height
    }
    wrap := func(x int) int {
        return ((x % width) + width) % width
    }
    stops := func(x int, y int) bool {
        switch map_.classicTerrain(x, y, data) {
            case Mountain, Hill, Desert, SorceryNode, NatureNode, ChaosNode: return true
        }
        return blocked(x, y)
    }
    seaAt := func(x int, y int) bool {
        return inside(y) && classicIsSea(map_.classicTerrain(wrap(x), y, data))
    }

    startX := 4 + classicRoll(52)
    startY := 4 + classicRoll(32)
    if startX >= width || startY >= height - 1 || blocked(startX, startY) {
        return false
    }
    for dx := -1; dx <= 1; dx++ {
        for dy := -1; dy <= 1; dy++ {
            if seaAt(startX + dx, startY + dy) {
                return false
            }
        }
    }
    switch map_.classicTerrain(startX, startY, data) {
        case Mountain, Hill, SorceryNode, NatureNode, ChaosNode, River: return false
    }

    path := []image.Point{image.Pt(startX, startY)}
    downstream := classicRoll(4) - 1
    finished := false
    for attempt := 0; !finished; attempt++ {
        if attempt >= classicRiverAttempts {
            return false
        }
        way := downstream
        if classicRoll(2) != 2 && attempt > 0 {
            way = classicRoll(4) - 1
            for way == classicRiverUpstream[downstream] {
                way = classicRoll(4) - 1
            }
        }
        last := path[len(path) - 1]
        x := wrap(last.X + classicRiverStepX[way])
        y := last.Y + classicRiverStepY[way]
        if !inside(y) || stops(x, y) {
            continue
        }
        path = append(path, image.Pt(x, y))
        if map_.classicTerrain(x, y, data) == River || seaAt(x, y) || seaAt(x + 1, y) || seaAt(x - 1, y) || seaAt(x, y + 1) || seaAt(x, y - 1) {
            finished = true
        }
        if len(path) > classicRiverLongest {
            return false
        }
    }
    if len(path) < classicRiverShortest {
        return false
    }

    // the pictures: every square of the river and its neighbors must find one
    saved := make(map[image.Point]int)
    for _, point := range path {
        if _, has := saved[point]; !has {
            saved[point] = map_.Terrain[point.X][point.Y]
        }
        map_.Terrain[point.X][point.Y] = TileRiver0001.Index(plane)
    }
    undo := func() {
        for point, index := range saved {
            map_.Terrain[point.X][point.Y] = index
        }
    }
    check := make(map[image.Point]bool)
    for _, point := range path {
        check[point] = true
        for way := range 4 {
            y := point.Y + classicRiverStepY[way]
            if inside(y) {
                check[image.Pt(wrap(point.X + classicRiverStepX[way]), y)] = true
            }
        }
    }
    trial := map_.Copy()
    for point := range check {
        if _, err := trial.ResolveTile(point.X, point.Y, data, plane); err != nil {
            undo()
            return false
        }
    }
    return true
}
