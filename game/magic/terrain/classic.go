package terrain

// The land of the original game: a world of 60 by 40 squares, the Land Size setting says how much
// of it is land. Facts from the ReMoM project's reconstruction (MoM/src/MAPGEN.c:
// Generate_Landmasses, Translate_Heightmap_To_Base_Terrain_Types, Add_Tundra,
// Generate_Climate_Terrain_Types; MoX/src/MOM_DEF.h: WORLD_WIDTH, WORLD_HEIGHT). The code is ours.
//
//   landmasses: walkers from up to 8 spread out places (a 5 by 5 grid of sections of 16 by 11, each
//     used once while the 8 last), each 3 to 5 arms from the place, an arm 5 to 14 squares long
//     (small), 10 to 19 (medium), 20 to 29 (large); every square walked on is one higher; the
//     walk turns by chance and goes straight less the longer it did; it keeps 2 squares from the
//     sides and 4 from the top and bottom; it ends when 360 (small), 480 (medium), 720 (large)
//     squares are land
//   heights: 6 and more mountain, 4 and 5 hills, 2 and 3 forest, 1 forest one time in 4, else
//     grassland
//   tundra: the top and bottom rows all (sea too), the rows next to them in runs of 1 to 4 squares,
//     one time in 4 per column; in rows 2 to 7 from either pole grassland and forest become tundra
//     the more often the nearer the pole
//   deserts: 8 patches, from a place in the middle rows (9 to 25 across 3 to 55) 5 arms of 5 to 10
//     random steps that turn any land into desert (the place only when it is forest); a step goes
//     around the edges of the world, also over the poles, which can put desert on a pole
//   swamps: 8 patches, from a place in rows 11 to 30, 5 arms of 3 to 5 steps that turn forest into
//     swamp
//
// The landmass numbers the original keeps while it walks (Build_Landmass) change no terrain and are
// not kept here: the game finds its landmasses by itself.

import (
    "image"
    "math/rand/v2"

    "github.com/kazzmir/master-of-magic/game/magic/data"
)

const (
    ClassicWidth = 60
    ClassicHeight = 40
)

// the original's Random(n): 1 to n
var classicRoll = func(n int) int {
    if n <= 0 {
        return 1
    }
    return rand.N(n) + 1
}

// the four ways a walker goes: down, left, up, right (dir_chg_*_tbl)
var classicStepX = []int{0, -1, 0, 1, 0}
var classicStepY = []int{1, 0, -1, 0, 0}

// the land squares the walk makes, by Land Size (0 small, 1 medium, 2 large)
func classicLandNeeded(landSize int) int {
    switch landSize {
        case 0: return 360
        case 2: return 720
    }
    return 480
}

// how long an arm of a landmass is
func classicArmLength(landSize int) int {
    switch landSize {
        case 0: return 5 + classicRoll(10)
        case 2: return 20 + classicRoll(10)
    }
    return 10 + classicRoll(10)
}

// Generate_Landmasses: the heights of the squares, [x][y]; 0 is sea
func classicHeights(landSize int) [][]int {
    heights := make([][]int, ClassicWidth)
    for x := range heights {
        heights[x] = make([]int, ClassicHeight)
    }

    const sectionWidth = 16
    const sectionHeight = 11
    var sections [5][5]bool
    sectionsLeft := 8

    needed := classicLandNeeded(landSize)
    made := 0
    for made <= needed {
        lastWay := -1
        straight := 1
        baseX, baseY := 0, 0
        for {
            baseX = 6 + classicRoll(46)
            baseY = 6 + classicRoll(26)
            if sectionsLeft <= 0 {
                break
            }
            if !sections[baseY / sectionHeight][baseX / sectionWidth] {
                sections[baseY / sectionHeight][baseX / sectionWidth] = true
                sectionsLeft -= 1
                break
            }
        }

        arms := 2 + classicRoll(3)
        for arm := 0; arm < arms; arm++ {
            length := classicArmLength(landSize)
            x := baseX + classicStepX[arm]
            y := baseY + classicStepY[arm]
            for steps := 0; steps < length && made <= needed; {
                if heights[x][y] == 0 {
                    made += 1
                }
                heights[x][y] += 1

                for {
                    way := classicRoll(4) - 1
                    if way == lastWay {
                        // the longer it went straight, the less it goes on so
                        if classicRoll(straight * 2) != 1 {
                            continue
                        }
                        straight += 1
                    } else {
                        straight = 1
                    }
                    lastWay = way
                    nextX := x + classicStepX[way]
                    nextY := y + classicStepY[way]
                    if nextX < 2 || nextY < 4 || nextX >= ClassicWidth - 2 || nextY >= ClassicHeight - 4 {
                        lastWay = -1
                        continue
                    }
                    x, y = nextX, nextY
                    steps += 1
                    break
                }
            }
        }
    }
    return heights
}

// the kinds of land the original's first passes know
type classicKind int

const (
    classicOcean classicKind = iota
    classicGrass
    classicForest
    classicHills
    classicMountain
    classicTundra
    classicDesert
    classicSwamp
)

// Translate_Heightmap_To_Base_Terrain_Types, Add_Tundra, Generate_Climate_Terrain_Types
func classicKinds(landSize int) [][]classicKind {
    heights := classicHeights(landSize)
    kinds := make([][]classicKind, ClassicWidth)
    for x := range kinds {
        kinds[x] = make([]classicKind, ClassicHeight)
        for y := range kinds[x] {
            switch height := heights[x][y]; {
                case height == 0: kinds[x][y] = classicOcean
                case height >= 6: kinds[x][y] = classicMountain
                case height >= 4: kinds[x][y] = classicHills
                case height >= 2: kinds[x][y] = classicForest
                case classicRoll(4) == 1: kinds[x][y] = classicForest
                default: kinds[x][y] = classicGrass
            }
        }
    }

    // Add_Tundra
    last := ClassicHeight - 1
    for x := 0; x < ClassicWidth; x++ {
        kinds[x][0] = classicTundra
        kinds[x][last] = classicTundra
        for _, row := range []int{1, last - 1} {
            if classicRoll(4) == 1 {
                count := classicRoll(4)
                for run := 0; run < count && x + run < ClassicWidth; run++ {
                    kinds[x + run][row] = classicTundra
                }
            }
        }
    }

    // the poles
    for row := 2; row < 8; row++ {
        for x := 0; x < ClassicWidth; x++ {
            for _, y := range []int{row, last - row} {
                if (kinds[x][y] == classicGrass || kinds[x][y] == classicForest) && 2 + classicRoll(8) >= row {
                    kinds[x][y] = classicTundra
                }
            }
        }
    }

    wrap := func(x int, y int) (int, int) {
        return (x + ClassicWidth) % ClassicWidth, (y + ClassicHeight) % ClassicHeight
    }

    // deserts
    for range 8 {
        baseX := 2 + classicRoll(ClassicWidth - 6)
        baseY := 8 + classicRoll(ClassicHeight - 16)
        if kinds[baseX][baseY] == classicForest {
            kinds[baseX][baseY] = classicDesert
        }
        for arm := 0; arm < 5; arm++ {
            x, y := baseX + classicStepX[arm], baseY + classicStepY[arm]
            steps := 4 + classicRoll(6)
            for range steps {
                way := classicRoll(4) - 1
                x, y = wrap(x + classicStepX[way], y + classicStepY[way])
                if kinds[x][y] != classicOcean {
                    kinds[x][y] = classicDesert
                }
            }
        }
    }

    // swamps
    for range 8 {
        baseX := 1 + classicRoll(ClassicWidth - 4)
        baseY := 10 + classicRoll(ClassicHeight - 20)
        if kinds[baseX][baseY] == classicForest {
            kinds[baseX][baseY] = classicSwamp
        }
        for arm := 0; arm < 5; arm++ {
            x, y := baseX + classicStepX[arm], baseY + classicStepY[arm]
            steps := 2 + classicRoll(3)
            for range steps {
                way := classicRoll(4) - 1
                x, y = wrap(x + classicStepX[way], y + classicStepY[way])
                if kinds[x][y] == classicForest {
                    kinds[x][y] = classicSwamp
                }
            }
        }
    }

    return kinds
}

// the kinds of node
const (
    classicSorcery = iota
    classicNature
    classicChaos
)

// a node of the original's worlds: its place, kind (ClassicSorcery, ClassicNature, ClassicChaos),
// power and aura (as many squares as its power, the node's own first)
type ClassicNode struct {
    X, Y int
    Kind int
    Power int
    Aura []image.Point
}

const (
    ClassicSorcery = classicSorcery
    ClassicNature = classicNature
    ClassicChaos = classicChaos
)

// Make_Aura: the squares of a node's aura: the node, up to 8 around it, then up to 2 away
func classicAura(power int, x int, y int) []image.Point {
    aura := []image.Point{image.Pt(x, y)}
    has := func(point image.Point) bool {
        for _, other := range aura {
            if other == point {
                return true
            }
        }
        return false
    }
    inside := func(point image.Point) bool {
        return point.X > 0 && point.Y > 0 && point.X < ClassicWidth - 1 && point.Y < ClassicHeight - 1
    }
    for count := 1; count < power && count < 9; count++ {
        for {
            point := image.Pt(x + classicStepX[classicRoll(4) - 1], y + classicStepY[classicRoll(4) - 1])
            if !has(point) && inside(point) {
                aura = append(aura, point)
                break
            }
        }
    }
    for count := 9; count < power; count++ {
        for {
            point := image.Pt(x + classicStepX[classicRoll(4) - 1] * classicRoll(2), y + classicStepY[classicRoll(4) - 1] * classicRoll(2))
            if !has(point) && inside(point) {
                aura = append(aura, point)
                break
            }
        }
    }
    return aura
}

// the distance of the original (Delta_XY_With_Wrap): the larger of across, around the world, and down
func classicDistance(x1 int, y1 int, x2 int, y2 int) int {
    dx := x1 - x2
    if dx < 0 {
        dx = -dx
    }
    dx = min(dx, ClassicWidth - dx)
    dy := y1 - y2
    if dy < 0 {
        dy = -dy
    }
    return max(dx, dy)
}

// Generate_Nodes, Set_Node_Type, Rebalance_Node_Types: 16 nodes on Arcanus, 14 on Myrror, spread
// over the world by regions (Arcanus 12 by 10 squares with room for 24 by 20, Myrror 12 by 20 with
// room for 24 by 40), 3 squares from the sides and poles (2 from the north), not in the sea but one
// time in 40 (Myrror 25), 3 or more apart, auras (their power: Arcanus 5 to 10 squares, Myrror 10
// to 20) not on each other. The kind by chance (sorcery 1 to 25, chaos and nature 1 to 15: chaos
// when it is the highest, else sorcery when above nature); then sorcery nodes beyond 9 (Myrror 4)
// make up to 6 chaos and 6 nature nodes (Myrror 3). The original checks Myrror's nodes against
// the land of Arcanus (a mistake it marks); here each plane its own
func classicNodes(kinds [][]classicKind, plane data.Plane) []ClassicNode {
    var nodes []ClassicNode
    count := 16
    if plane == data.PlaneMyrror {
        count = 14
    }
    for index := 0; index < count; index++ {
        // the original tries for ever; a region with no room gives up here
        for tries := 0; tries < classicNodeTries; tries++ {
            var x, y, power int
            if plane == data.PlaneMyrror {
                // the original's regions start 4 before the first (a mistake it marks), so they wrap
                // Go divides and takes the remainder as C does (toward 0)
                region := index - 4
                baseX := region % 5 * 12
                baseY := region / 5 * 20
                x = baseX + classicRoll(24) - 1
                y = baseY + classicRoll(40) - 1
                power = 9 + classicRoll(11)
            } else {
                x = index % 5 * 12 + classicRoll(24) - 1
                y = index / 5 * 10 + classicRoll(20) - 1
                power = 4 + classicRoll(6)
            }
            x = (x + ClassicWidth) % ClassicWidth
            y = (y + ClassicHeight) % ClassicHeight
            if x < 3 || y < 2 || x >= ClassicWidth - 3 || y >= ClassicHeight - 3 {
                continue
            }
            chance := 40
            if plane == data.PlaneMyrror {
                chance = 25
            }
            if kinds[x][y] == classicOcean && classicRoll(chance) != 1 {
                continue
            }
            near := false
            for _, other := range nodes {
                if classicDistance(x, y, other.X, other.Y) < 3 {
                    near = true
                }
            }
            if near {
                continue
            }
            aura := classicAura(power, x, y)
            overlap := false
            for _, other := range nodes {
                for _, point := range other.Aura {
                    for _, mine := range aura {
                        if point == mine {
                            overlap = true
                        }
                    }
                }
            }
            if overlap {
                continue
            }
            sorcery, chaos, nature := classicRoll(25), classicRoll(15), classicRoll(15)
            kind := classicNature
            switch {
                case chaos > sorcery && chaos > nature: kind = classicChaos
                case sorcery > nature: kind = classicSorcery
            }
            nodes = append(nodes, ClassicNode{X: x, Y: y, Kind: kind, Power: power, Aura: aura})
            break
        }
    }

    counts := make(map[int]int)
    for _, node := range nodes {
        counts[node.Kind] += 1
    }
    most, least := 9, 6
    if plane == data.PlaneMyrror {
        most, least = 4, 3
    }
    // the original does not count the excess down, so any excess makes up both
    if counts[classicSorcery] > most {
        for _, want := range []int{classicChaos, classicNature} {
            for counts[want] < least {
                converted := false
                for _, index := range rand.Perm(len(nodes)) {
                    if nodes[index].Kind == classicSorcery {
                        nodes[index].Kind = want
                        converted = true
                        break
                    }
                }
                if !converted {
                    break
                }
                counts[want] += 1
            }
        }
    }
    return nodes
}

const classicNodeTries = 20000

// the fork's picture of a kind of land; the kinds of grass and forest by chance as the fork has them
// (the original picks its variants later, Shuffle_Terrains)
func classicTile(kind classicKind, plane data.Plane) int {
    switch kind {
        case classicGrass:
            tiles := []Tile{TileGrasslands1, TileGrasslands2, TileGrasslands3, TileGrasslands4}
            return tiles[rand.N(len(tiles))].Index(plane)
        case classicForest:
            tiles := []Tile{TileForest1, TileForest2, TileForest3}
            return tiles[rand.N(len(tiles))].Index(plane)
        case classicHills: return TileHills1.Index(plane)
        case classicMountain: return TileMountain1.Index(plane)
        case classicTundra: return TileTundra.Index(plane)
        case classicDesert: return TileAllDesert1.Index(plane)
        case classicSwamp: return TileSwamp1.Index(plane)
    }
    return TileOcean.Index(plane)
}

// the land and nodes of the original game on a map of 60 by 40, then the fork's rivers and the pictures that
// fit their neighbors
func classicMap(kinds [][]classicKind, nodes []ClassicNode, data *TerrainData, plane data.Plane) *Map {
    map_ := MakeMap(ClassicHeight, ClassicWidth)
    land := 0
    for x := 0; x < ClassicWidth; x++ {
        for y := 0; y < ClassicHeight; y++ {
            map_.Terrain[x][y] = classicTile(kinds[x][y], plane)
            if kinds[x][y] != classicOcean && y > 0 && y < ClassicHeight - 1 {
                land += 1
            }
        }
    }
    nodeTiles := map[int]Tile{classicSorcery: TileSorceryLake, classicNature: TileNatureForest, classicChaos: TileChaosVolcano}
    for _, node := range nodes {
        map_.Terrain[node.X][node.Y] = nodeTiles[node.Kind].Index(plane)
    }
    // the rivers come after the cities, minerals and roads, as in the original (ClassicRiver,
    // called by the game)
    map_.ResolveTiles(data, plane)
    return map_
}

// Generate_Towers: 6 towers of wizardry, 3 to 56 across and 3 to 36 down, where either plane has
// land (the sea one time in 40), 4 or more from every node, apart by a distance that starts at 10
// and shrinks by one every 50 tries that fail (the original's count of tries starts with what was
// left on the stack, so it shrinks at once: from 9); the square becomes grassland on both planes
func classicTowers(arcanus [][]classicKind, myrror [][]classicKind, nodes []ClassicNode) []image.Point {
    var towers []image.Point
    distance := 10
    tries := 500
    for len(towers) < ClassicTowers {
        tries += 1
        if tries > 500 {
            tries = 450
            distance -= 1
        }
        x := 2 + classicRoll(54)
        y := 2 + classicRoll(34)
        if arcanus[x][y] == classicOcean && myrror[x][y] == classicOcean && classicRoll(40) > 1 {
            continue
        }
        ok := true
        for _, tower := range towers {
            if classicDistance(x, y, tower.X, tower.Y) < distance {
                ok = false
            }
        }
        for _, node := range nodes {
            if classicDistance(x, y, node.X, node.Y) < 4 {
                ok = false
            }
        }
        if !ok {
            continue
        }
        towers = append(towers, image.Pt(x, y))
        arcanus[x][y] = classicGrass
        myrror[x][y] = classicGrass
    }
    return towers
}

const ClassicTowers = 6

// the worlds of the original game (Init_New_Game, as far as it is ported): the land of both planes,
// their nodes, the towers, the islands grown around them (Extend_Islands). The towers and the nodes
// of both planes are returned for the game to make their guardians
func GenerateClassicWorld(landSize int, terrainData *TerrainData) (*Map, *Map, []image.Point, [2][]ClassicNode) {
    arcanusKinds := classicKinds(landSize)
    myrrorKinds := classicKinds(landSize)
    arcanusNodes := classicNodes(arcanusKinds, data.PlaneArcanus)
    myrrorNodes := classicNodes(myrrorKinds, data.PlaneMyrror)
    towers := classicTowers(arcanusKinds, myrrorKinds, append(append([]ClassicNode{}, arcanusNodes...), myrrorNodes...))
    classicExtendIslands(arcanusKinds, arcanusNodes, towers)
    classicExtendIslands(myrrorKinds, myrrorNodes, towers)
    arcanus := classicMap(arcanusKinds, arcanusNodes, terrainData, data.PlaneArcanus)
    myrror := classicMap(myrrorKinds, myrrorNodes, terrainData, data.PlaneMyrror)
    return arcanus, myrror, towers, [2][]ClassicNode{arcanusNodes, myrrorNodes}
}

// NUM_RIVERS
const ClassicRivers = 10
