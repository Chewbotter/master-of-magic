package combat

// The ground of a battlefield, made the way the original makes it. Facts read from ReMoM
// (Generate_Combat_Map and the functions it calls: Scatter_Terrain_Patches, Build_Road_Network,
// Remove_Rough_Terrain, Merge_Dirt_Patches, Set_Terrain_Tile_Types, Combat_Screen_Map_Draw for the
// roads). The code is ours.
//
// Every cell first gets a kind of ground: grass, rough (raised ground) or dirt. Rough and dirt come
// in patches that wander from a random cell. Then every cell gets a picture from its kind and the
// kinds of its neighbors: grass next to dirt shows the edge of the dirt, rough shows which of its
// four sides go on as rough. Roads lead from the middle of the field out of it toward the roads of
// the world map around the battle.
//
// Our field is larger than the original's grid, and a border of ground lies around it
// (fieldedge.go). The ground is made for all of it, in cells of the original's grid, so it goes on
// across the edge of the field. The original's grid gets the original's number of patches, the rest
// as many for its size.
//
// Not made, as in the original: rivers. The original has the code for them, but never gives it a
// river to follow.

type TerrainGroup int

const (
    TerrainGrass TerrainGroup = iota
    TerrainRough
    TerrainDirt
)

// what the world map around a battle tells the ground of the battle. the zero value is open grass
// land without roads
type ZoneGround struct {
    // a forest or a node of nature: grass with many trees
    Forest bool
    // hills: grass with much rough ground
    Hills bool
    // roads on the world map around the battle, in rows of 3 from the north west. 4 is the tile of
    // the battle itself: without a road there, the battlefield has none
    Roads [9]bool
    EnchantedRoads bool
}

// the directions a step goes, as the original's step_delta_cgx and step_delta_cgy: 1 is +y, 2 is -x,
// 3 is -y, 4 is +x of the grid. 0 stays
var stepCellX = [5]int{0, 0, -1, 0, 1}
var stepCellY = [5]int{0, 1, 0, -1, 0}

// patches of rough ground, and how long each wanders: 3 to 10 steps
const roughSpan = 8
const roughBase = 3
// patches of dirt, 1 to 4 steps each
const dirtPatches = 8
const dirtSpan = 4
const dirtBase = 0
// passes that turn rough next to dirt into grass, and that fill grass between dirt
const roughRemovePasses = 8
const dirtMergePasses = 10

// the pictures of a terrain set, by number
const groundGrassFirst = 0
const groundDirtFirst = 4
const groundRoughFirst = 32

// the pieces of road, cmbtcity 69 on: 6 directions in two sets of 7, and the same for enchanted roads
const roadLbx = "cmbtcity.lbx"
const roadFirstPicture = 69
const roadSetPictures = 7
const roadEnchantedPictures = 14

// bits of a road: which of its neighbors it runs to. with bit 0 the first set of pictures, else the
// second
const (
    roadFirstSet = 1
    roadAcross = 2
    roadAlong = 4
    roadUpLeft = 8
    roadRightUp = 16
    roadDownRight = 32
    roadLeftDown = 64
)

// the ground of a battlefield and the border around it, in cells of the original's grid
type BattleGround struct {
    MinX int
    MinY int
    Width int
    Height int
    Group []TerrainGroup
    // the picture of the terrain set of every cell
    Picture []int
    // the bits of the road of every cell, 0 without one
    Roads []int
    // the trees that stand in every cell, see movecost.go
    Trees []int
    // the height of every cell, 0 or 1, nil without plateaus. see plateau.go
    Heights []int
    EnchantedRoads bool
}

func (ground *BattleGround) contains(cgx int, cgy int) bool {
    return cgx >= ground.MinX && cgy >= ground.MinY && cgx < ground.MinX + ground.Width && cgy < ground.MinY + ground.Height
}

func (ground *BattleGround) index(cgx int, cgy int) int {
    return (cgy - ground.MinY) * ground.Width + (cgx - ground.MinX)
}

// the kind of ground of a cell. beyond the ground it is grass, as the original has it beyond its grid
func (ground *BattleGround) GroupAt(cgx int, cgy int) TerrainGroup {
    if !ground.contains(cgx, cgy) {
        return TerrainGrass
    }
    return ground.Group[ground.index(cgx, cgy)]
}

func (ground *BattleGround) setGroup(cgx int, cgy int, group TerrainGroup) {
    if ground.contains(cgx, cgy) {
        ground.Group[ground.index(cgx, cgy)] = group
    }
}

func (ground *BattleGround) RoadAt(cgx int, cgy int) int {
    if !ground.contains(cgx, cgy) {
        return 0
    }
    return ground.Roads[ground.index(cgx, cgy)]
}

// the picture of the ground of a tile of our field or its border
func (ground *BattleGround) PictureOfTile(x int, y int) int {
    cgx, cgy := TileToCell(x, y)
    if !ground.contains(cgx, cgy) {
        return groundGrassFirst
    }
    return ground.Picture[ground.index(cgx, cgy)]
}

func (ground *BattleGround) RoadOfTile(x int, y int) int {
    cgx, cgy := TileToCell(x, y)
    return ground.RoadAt(cgx, cgy)
}

func (ground *BattleGround) addTree(cgx int, cgy int) {
    if ground != nil && ground.contains(cgx, cgy) {
        ground.Trees[ground.index(cgx, cgy)] += 1
    }
}

func (ground *BattleGround) TreesAt(cgx int, cgy int) int {
    if !ground.contains(cgx, cgy) {
        return 0
    }
    return ground.Trees[ground.index(cgx, cgy)]
}

// trees and rocks stand on grass without a road only
func (ground *BattleGround) sceneryAllowed(cgx int, cgy int) bool {
    if ground == nil {
        return true
    }
    if !ground.contains(cgx, cgy) {
        return false
    }
    return ground.GroupAt(cgx, cgy) == TerrainGrass && ground.RoadAt(cgx, cgy) == 0
}

func insideOriginalGrid(cgx int, cgy int) bool {
    return cgx >= 0 && cgy >= 0 && cgx < OriginalGridWidth && cgy < OriginalGridHeight
}

// how many patches of rough ground the original makes on its grid
func roughPatches(landscape CombatLandscape, ground ZoneGround) int {
    switch {
        case ground.Forest: return 5
        case ground.Hills: return 20
    }
    switch landscape {
        case CombatLandscapeDesert: return 10
        case CombatLandscapeMountain: return 30
        case CombatLandscapeTundra: return 10
    }
    return 5
}

func makeBattleGround(width int, height int, landscape CombatLandscape, zone ZoneType) *BattleGround {
    if landscape == CombatLandscapeWater {
        return nil
    }

    // the cells of the field and its border
    minX, _ := TileToCell(0, -BattlefieldBorder)
    _, minY := TileToCell(width - 1 + BattlefieldBorder, 0)
    ground := &BattleGround{
        MinX: minX,
        MinY: minY,
        Width: height + BattlefieldBorder * 2,
        Height: width + BattlefieldBorder * 2,
        EnchantedRoads: zone.Ground.EnchantedRoads,
    }
    cells := ground.Width * ground.Height
    ground.Group = make([]TerrainGroup, cells)
    ground.Picture = make([]int, cells)
    ground.Roads = make([]int, cells)
    ground.Trees = make([]int, cells)

    // the original's grid gets its number of patches, the rest of the ground as many for its size
    beyond := max(0, cells / (OriginalGridWidth * OriginalGridHeight) - 1)
    rough := roughPatches(landscape, zone.Ground)
    ground.scatterPatches(TerrainRough, rough, roughSpan, roughBase, true)
    ground.scatterPatches(TerrainRough, rough * beyond, roughSpan, roughBase, false)
    if PlateauGround {
        // the rough ground becomes plateaus and small hills, see plateau.go
        ground.makePlateaus(zone, rough * (beyond + 1))
        // the original's rough pictures as short mounds, see plateau.go
        ground.makeMounds(rough * (beyond + 1))
    }
    ground.scatterPatches(TerrainDirt, dirtPatches, dirtSpan, dirtBase, true)
    ground.scatterPatches(TerrainDirt, dirtPatches * beyond, dirtSpan, dirtBase, false)

    ground.buildRoads(zone)
    ground.removeRough(zone)
    ground.mergeDirt()
    ground.choosePictures()

    return ground
}

// patches that wander from a random cell, Scatter_Terrain_Patches. inside: the patches start on the
// original's grid, else beyond it
func (ground *BattleGround) scatterPatches(group TerrainGroup, count int, span int, base int, inside bool) {
    for range count {
        var cgx, cgy int
        for {
            cgx = ground.MinX + roll(ground.Width) - 1
            cgy = ground.MinY + roll(ground.Height) - 1
            if insideOriginalGrid(cgx, cgy) == inside {
                break
            }
        }

        start := roll(4)
        cgx += stepCellX[start]
        cgy += stepCellY[start]

        size := base + roll(span)
        for range size {
            step := roll(4)
            cgx = min(max(cgx + stepCellX[step], ground.MinX), ground.MinX + ground.Width - 1)
            cgy = min(max(cgy + stepCellY[step], ground.MinY), ground.MinY + ground.Height - 1)
            ground.setGroup(cgx, cgy, group)
        }
    }
}

// a town takes the cells of the original's town. an outpost does not, its roads start in the middle
func isTown(zone ZoneType) bool {
    return zone.City != nil && !zone.City.Outpost
}

// a road that starts at a cell and wanders to the edge of the ground, Build_Road_Network. heading is
// the step it keeps to, back the one it never takes
type roadWalk struct {
    StartX int
    StartY int
    Heading int
    Back int
    // one in this many steps goes a random way
    Wander int
    // after a step that was chosen the next one goes the same way. with this only after a turn
    KeepAfterTurn bool
}

func (ground *BattleGround) walkRoad(walk roadWalk, done func(cgx int, cgy int) bool) {
    cgx, cgy := walk.StartX, walk.StartY
    ground.markRoad(cgx, cgy)

    previous := walk.Heading
    steps := 0
    for !done(cgx, cgy) {
        var step int
        if steps < 2 {
            step = previous
            steps += 1
        } else {
            for {
                if roll(walk.Wander) == 1 {
                    step = roll(4) - 1
                } else {
                    step = walk.Heading
                }
                if step != walk.Back {
                    break
                }
            }
            if !walk.KeepAfterTurn || step != previous {
                steps = 1
            }
        }
        previous = step

        nextX := cgx + stepCellX[step + 1]
        nextY := cgy + stepCellY[step + 1]
        if ground.contains(nextX, nextY) {
            cgx, cgy = nextX, nextY
            ground.markRoad(cgx, cgy)
        }
    }
}

func (ground *BattleGround) markRoad(cgx int, cgy int) {
    if ground.contains(cgx, cgy) {
        ground.Roads[ground.index(cgx, cgy)] = roadFirstSet
    }
}

func (ground *BattleGround) buildRoads(zone ZoneType) {
    roads := zone.Ground.Roads
    if !roads[4] {
        return
    }

    town := isTown(zone)
    lastX := ground.MinX + ground.Width - 1
    lastY := ground.MinY + ground.Height - 1

    start := func(townX int, townY int) (int, int) {
        if town {
            return townX, townY
        }
        return 10, 10
    }

    // steps: 0 is +y, 1 is -x, 2 is -y, 3 is +x of the grid
    if roads[5] || roads[2] {
        x, y := start(townCellMaxX - 1, townCellMaxY - 1)
        ground.walkRoad(roadWalk{StartX: x, StartY: y, Heading: 3, Back: 1, Wander: 4}, func(cgx int, cgy int) bool { return cgx >= lastX })
    }
    if roads[1] || roads[0] {
        x, y := start(7, 11)
        ground.walkRoad(roadWalk{StartX: x, StartY: y, Heading: 2, Back: 0, Wander: 3, KeepAfterTurn: true}, func(cgx int, cgy int) bool { return cgy <= ground.MinY })
    }
    if roads[3] || roads[6] {
        x, y := start(6, 12)
        ground.walkRoad(roadWalk{StartX: x, StartY: y, Heading: 1, Back: 3, Wander: 4}, func(cgx int, cgy int) bool { return cgx <= ground.MinX })
    }
    if roads[7] || roads[8] {
        x, y := start(7, 12)
        ground.walkRoad(roadWalk{StartX: x, StartY: y, Heading: 0, Back: 2, Wander: 4}, func(cgx int, cgy int) bool { return cgy >= lastY })
    }

    // which neighbors each road runs to. beyond the ground the road goes on
    road := func(cgx int, cgy int) bool {
        return !ground.contains(cgx, cgy) || ground.RoadAt(cgx, cgy) != 0
    }
    for cgy := ground.MinY; cgy <= lastY; cgy++ {
        for cgx := ground.MinX; cgx <= lastX; cgx++ {
            index := ground.index(cgx, cgy)
            if ground.Roads[index] == 0 {
                continue
            }
            up, down := road(cgx, cgy - 1), road(cgx, cgy + 1)
            left, right := road(cgx - 1, cgy), road(cgx + 1, cgy)
            if left && right {
                ground.Roads[index] += roadAcross
            }
            if up && down {
                ground.Roads[index] += roadAlong
            }
            if up && left {
                ground.Roads[index] += roadUpLeft
            }
            if right && up {
                ground.Roads[index] += roadRightUp
            }
            if down && right {
                ground.Roads[index] += roadDownRight
            }
            if left && down {
                ground.Roads[index] += roadLeftDown
            }
        }
    }

    // half of the roads show the other set of pictures
    for index, bits := range ground.Roads {
        if bits != 0 && roll(2) == 1 {
            ground.Roads[index] = bits &^ roadFirstSet
        }
    }
}

// Remove_Rough_Terrain: no rough next to dirt, on a road, or in the town or lair
func (ground *BattleGround) removeRough(zone ZoneType) {
    for range roughRemovePasses {
        for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
            for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
                if ground.GroupAt(cgx, cgy) != TerrainRough {
                    continue
                }
                nearDirt := false
                for dy := -1; dy <= 1; dy++ {
                    for dx := -1; dx <= 1; dx++ {
                        if (dx != 0 || dy != 0) && ground.GroupAt(cgx + dx, cgy + dy) == TerrainDirt {
                            nearDirt = true
                        }
                    }
                }
                if nearDirt || ground.RoadAt(cgx, cgy) != 0 {
                    ground.setGroup(cgx, cgy, TerrainGrass)
                }
            }
        }
    }

    if zone.City != nil || zone.Encounter != ZoneNone {
        for cgy := townCellMinY; cgy <= townCellMaxY; cgy++ {
            for cgx := townCellMinX; cgx <= townCellMaxX; cgx++ {
                if ground.GroupAt(cgx, cgy) == TerrainRough {
                    ground.setGroup(cgx, cgy, TerrainGrass)
                }
            }
        }
    }
}

// Merge_Dirt_Patches: grass between two cells of dirt becomes dirt
func (ground *BattleGround) mergeDirt() {
    for range dirtMergePasses {
        for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
            for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
                if ground.GroupAt(cgx, cgy) != TerrainGrass {
                    continue
                }
                across := ground.GroupAt(cgx - 1, cgy) == TerrainDirt && ground.GroupAt(cgx + 1, cgy) == TerrainDirt
                along := ground.GroupAt(cgx, cgy - 1) == TerrainDirt && ground.GroupAt(cgx, cgy + 1) == TerrainDirt
                if across || along {
                    ground.setGroup(cgx, cgy, TerrainDirt)
                }
            }
        }
    }
}

// the picture of rough ground by which of its sides go on as rough: up 8, right 4, down 2, left 1
var roughPictures = [16]int{15, 6, 0, 9, 2, 3, 7, 8, 5, 14, 1, 11, 12, 13, 10, 4}

// Set_Terrain_Tile_Types
func (ground *BattleGround) choosePictures() {
    for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
        for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
            ground.Picture[ground.index(cgx, cgy)] = ground.pictureOf(cgx, cgy)
        }
    }
}

func (ground *BattleGround) pictureOf(cgx int, cgy int) int {
    is := func(dx int, dy int, group TerrainGroup) bool {
        return ground.GroupAt(cgx + dx, cgy + dy) == group
    }

    switch ground.GroupAt(cgx, cgy) {
        case TerrainDirt:
            return groundDirtFirst - 1 + roll(4)

        case TerrainRough:
            sides := 0
            if is(0, -1, TerrainRough) {
                sides += 8
            }
            if is(1, 0, TerrainRough) {
                sides += 4
            }
            if is(0, 1, TerrainRough) {
                sides += 2
            }
            if is(-1, 0, TerrainRough) {
                sides += 1
            }
            return groundRoughFirst + roughPictures[sides]
    }

    // grass: where dirt lies next to it, the edge of the dirt. two pictures of each
    up := is(0, -1, TerrainDirt)
    down := is(0, 1, TerrainDirt)
    left := is(-1, 0, TerrainDirt)
    right := is(1, 0, TerrainDirt)

    edge := func(first int) int {
        return first - 1 + roll(2)
    }

    switch {
        case right && up: return edge(28)
        case right && down: return edge(12)
        case right: return edge(30)
        case up && left: return edge(22)
        case up: return edge(24)
        case left && down: return edge(16)
        case left: return edge(18)
        case down: return edge(10)
        case is(1, 1, TerrainDirt): return edge(8)
        case is(1, -1, TerrainDirt): return edge(26)
        case is(-1, -1, TerrainDirt): return edge(20)
        case is(-1, 1, TerrainDirt): return edge(14)
    }

    return groundGrassFirst - 1 + roll(4)
}

// the kind of ground and the road of every tile of the field, and its picture
func (ground *BattleGround) applyTo(tiles [][]Tile) {
    if ground == nil {
        return
    }
    for y := range tiles {
        for x := range tiles[y] {
            cgx, cgy := TileToCell(x, y)
            tiles[y][x].Index = ground.PictureOfTile(x, y)
            tiles[y][x].Ground = ground.GroupAt(cgx, cgy)
            tiles[y][x].Road = ground.RoadAt(cgx, cgy) != 0
        }
    }
}

// the pieces of road of a cell, as cmbtcity entries
func (ground *BattleGround) roadPictures(bits int) []int {
    var out []int
    if bits == 0 {
        return out
    }

    first := roadFirstPicture
    if ground.EnchantedRoads {
        first += roadEnchantedPictures
    }
    if bits & roadFirstSet == 0 {
        first += roadSetPictures
    }

    for piece, bit := range []int{roadAcross, roadAlong, roadUpLeft, roadRightUp, roadDownRight, roadLeftDown} {
        if bits & bit != 0 {
            out = append(out, first + piece)
        }
    }
    return out
}

