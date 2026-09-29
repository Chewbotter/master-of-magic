package combat

// The coast (user, 2026-09-29). Not in the original, where a battle beside the sea is fought on
// land all around.
//
// A battle on land that has the sea beside it on the world map has a coast on that side of the
// field: a wide beach, then water to the end of the ground. The armies stand on land: the beach
// starts behind the places of the army on that side.
//
// A COAST IS A FEATURE, NOT A BIOME: it lies on whatever the landscape of the battle is. It is the
// first of the features that water and the nearness of towns are to be (see CLAUDE.md).
//
// The field is the world map turned by an eighth: its north is the upper right of the screen. The
// grid has it as the roads of the original do (Build_Road_Network): east is +cgx, west -cgx, north
// -cgy, south +cgy. The defenders stand in the west, the attackers in the east.
//
// THE SHAPE, KEPT SIMPLE (user, 2026-09-29: "make the generation only create coastlines that can
// easily follow the rules, instead of accounting for complex exceptions"). Everything is decided
// cell by cell. Along the coast every column of cells has, from the land out:
//   land                  as without a coast
//   one TRANSITION        the tile of the landscape, with sand over the part toward the beach
//   the BEACH             tiles of sand, 3 to 5 of them
//   one TRANSITION        a tile of water, with sand over the part toward the beach
//   water                 to the end of the ground
// From one column to the next the beach starts at the same cell or one farther out or in, and
// the same for the water, never both at once. So a transition always has the transitions of the
// columns beside it next to it or across a corner.
//
// The edge of the sand runs through the transitions and never leaves them: through the middle of
// a cell, to the middle of the side it shares with the next, or to the corner where the next one
// lies across it. So it is straight along the coast and runs at a slant where the coast moves, as
// the cut corners of the plateaus do (slopes.go). Pixel by pixel it is ragged: a slow wave along
// the coast and a scatter of single pixels (sandIn).
//
// TRANSITIONS COUNT AS WHAT IS UNDERNEATH (user, 2026-09-29: "only full beach tiles should count
// as their own unique rough terrain"): as the landscape, or as water.
//
// RULES. The beach is rough ground: a step onto it costs twice what grass costs (moveHalvesRough).
// Water is for the units that fight on the sea (game.go leaves out the others of a battle on the
// ocean): what flies, swims or sails. All others can not enter it.
//
// PICTURES. The beach is the sand of the desert, the water the water of the battles on the ocean.
// The folder Shore of the replacement folder can take their places and add to them:
//
//   environment/Shore/sand 1_0.png ... sand 4_0.png     in place of the game's, sand 5 on are added
//   environment/Shore/water 1_0.png ... water 4_0.png   the same. the game's have 5 frames

import (
    "math"
    "math/rand/v2"

    "github.com/kazzmir/master-of-magic/game/magic/data"
)

// turns it off
var CoastGround = true

type CoastSide int
const (
    CoastNone CoastSide = iota
    CoastEast
    CoastNorth
    CoastWest
    CoastSouth
    // one of the four by chance, picked when the ground is made
    CoastAny
)

func (side CoastSide) String() string {
    switch side {
        case CoastEast: return "east"
        case CoastNorth: return "north"
        case CoastWest: return "west"
        case CoastSouth: return "south"
        case CoastAny: return "any"
    }
    return "none"
}

// the first cells the sand can reach lie this many cells behind the places of the armies
// (deploy.go): near enough to be seen without moving the view, with room to stand behind the last
// row. armies of more than 12 units take more places, the beach lies farther out then
const coastBehindArmy = 2

// the lines of the coast wander: they lie up to this many cells farther out than their nearest
const coastWander = 2
// one step along the coast in this many moves a line
const coastWanderChance = 3
// how many cells of beach lie between the two transitions
const coastBeachMin = 3
const coastBeachMax = 5
// the beach is narrow in places: a stretch starts at one column in this many, has this many cells
// of beach and keeps them for this many columns. the beach gets there and back a cell a column
const coastNarrowChance = 14
const coastNarrowLengthMin = 3
const coastNarrowLengthMax = 6
const coastNarrowMin = 1
const coastNarrowMax = 2
// plateaus end this many cells before the beach, so their edge is not drawn along it
const coastFlatBefore = 1

// how ragged the edge of the sand is, in cells: a slow wave along the coast, coastWaveLength cells
// long, and a scatter pixel by pixel
const coastWave = 0.2
const coastWaveLength = 2.5
const coastScatter = 0.15

// the pictures of the beach and of the water as the ground has them, in place of the numbers of the
// archive of the landscape, and how many the game has of each
const groundSandFirst = 60
const groundWaterFirst = 70
const groundSandCount = 4
const groundWaterCount = 4

const coastSet = "Shore"
const coastSandName = "sand"
const coastWaterName = "water"

// the archive and the first picture of the sand and of the water
func sandPictures(plane data.Plane) (string, int) {
    return terrainSetLbx(CombatLandscapeDesert, plane), groundGrassFirst
}

func waterPictures(plane data.Plane) (string, int) {
    lbx, first, _ := groundPictures(CombatLandscapeWater, plane)
    return lbx, first
}

// the columns and rows of the grid the places of the armies take: the first and the last of each
func armyPlaces(large bool) (int, int, int, int) {
    rows := deployRows
    columns := deployRowWidth
    if large {
        rows = len(deployDefenderRows)
        columns = len(deployColumns)
    }

    firstX, lastX := deployDefenderRows[0], deployAttackerRows[0]
    for _, row := range deployDefenderRows[:rows] {
        firstX = min(firstX, row)
    }
    for _, row := range deployAttackerRows[:rows] {
        lastX = max(lastX, row)
    }

    firstY, lastY := deployColumns[0], deployColumns[0]
    for _, column := range deployColumns[:columns] {
        firstY = min(firstY, column)
        lastY = max(lastY, column)
    }

    return firstX, lastX, firstY, lastY
}

// the coast of a battlefield
type coastLines struct {
    Side CoastSide
    // the column or row of the grid of the first cells the sand can reach
    First int
    // along the coast, from the cell FirstAlong on: the depth of the transition where the beach
    // starts, and of the one where the water starts
    FirstAlong int
    Starts []int
    Waters []int
    // for the wave of the edge
    Seed uint32
}

// what a cell is of the coast
type coastPart int
const (
    // land as without a coast
    coastLand coastPart = iota
    // transitions: sand over what is underneath, the landscape or the water
    coastLandEdge
    coastWaterEdge
    coastBeach
    coastWater
)

// how far a point lies out toward the coast and where along it, in cells. a cell of depth 0 has the
// points of depth 0 to 1
func (coast *coastLines) depth(cellX float64, cellY float64) (float64, float64) {
    switch coast.Side {
        case CoastEast: return cellX - float64(coast.First), cellY
        case CoastWest: return float64(coast.First + 1) - cellX, cellY
        case CoastNorth: return float64(coast.First + 1) - cellY, cellX
        case CoastSouth: return cellY - float64(coast.First), cellX
    }
    return -1000, 0
}

// the same for a cell: its depth, and its place along the coast
func (coast *coastLines) cellDepth(cgx int, cgy int) (int, int) {
    depth, along := coast.depth(float64(cgx) + 0.5, float64(cgy) + 0.5)
    return int(math.Floor(depth)), int(math.Floor(along))
}

// one of the values along the coast. beyond its ends the value of the end
func (coast *coastLines) at(values []int, along int) int {
    return values[min(max(along - coast.FirstAlong, 0), len(values) - 1)]
}

func (coast *coastLines) partOf(cgx int, cgy int) coastPart {
    depth, along := coast.cellDepth(cgx, cgy)
    start := coast.at(coast.Starts, along)
    water := coast.at(coast.Waters, along)

    switch {
        case depth < start: return coastLand
        case depth == start: return coastLandEdge
        case depth < water: return coastBeach
        case depth == water: return coastWaterEdge
    }
    return coastWater
}

// a number for a place, -1 to 1, the same every time
func coastNoise(a int, b int, seed uint32) float64 {
    value := uint32(a + 10000) * 73856093 ^ uint32(b + 10000) * 19349663 ^ seed * 83492791
    value ^= value >> 13
    value *= 2654435761
    value ^= value >> 16
    return float64(value % 2001) / 1000 - 1
}

// where the edge of the sand runs through a transition at a place along the coast, 0 to 1 of the
// depth of the cell: through its middle, and straight from there to where the edge of the next
// column starts, which is the middle of the shared side or the corner. values are the depths of the
// transitions along the coast
func (coast *coastLines) edgeIn(values []int, along float64) float64 {
    column := int(math.Floor(along))
    // from the middle of the column to its sides, -0.5 to 0.5
    side := along - float64(column) - 0.5

    next := column + 1
    if side < 0 {
        next = column - 1
    }
    step := float64(coast.at(values, next) - coast.at(values, column))
    edge := 0.5 + step * math.Abs(side)

    // the wave, which is none where the edge meets the side of the cell
    at := along / coastWaveLength
    before := int(math.Floor(at))
    part := slopeSmooth(at - float64(before))
    wave := coastNoise(before, 0, coast.Seed) * (1 - part) + coastNoise(before + 1, 0, coast.Seed) * part
    return edge + wave * coastWave * 2 * math.Min(edge, 1 - edge)
}

// true if there is sand at a point of a transition, which is the pixel pixelX, pixelY of the field
func (coast *coastLines) sandIn(part coastPart, cellX float64, cellY float64, pixelX int, pixelY int) bool {
    depth, along := coast.depth(cellX, cellY)
    inside := depth - math.Floor(depth) + coastNoise(pixelX, pixelY, coast.Seed) * coastScatter

    switch part {
        // the sand lies toward the beach: farther out than the edge, or nearer than it
        case coastLandEdge: return inside >= coast.edgeIn(coast.Starts, along)
        case coastWaterEdge: return inside < coast.edgeIn(coast.Waters, along)
    }
    return part == coastBeach
}

// lays the coast over the ground: beach and water, flat, without roads into the water
func (ground *BattleGround) makeCoast(zone ZoneType) {
    side := zone.Ground.Coast
    if !CoastGround || side == CoastNone {
        return
    }
    if side == CoastAny {
        side = []CoastSide{CoastEast, CoastNorth, CoastWest, CoastSouth}[rand.N(4)]
    }

    firstX, lastX, firstY, lastY := armyPlaces(zone.Ground.LargeArmy)
    coast := &coastLines{Side: side, Seed: rand.Uint32()}
    switch side {
        case CoastEast: coast.First = lastX + coastBehindArmy
        case CoastWest: coast.First = firstX - coastBehindArmy
        case CoastNorth: coast.First = firstY - coastBehindArmy
        case CoastSouth: coast.First = lastY + coastBehindArmy
    }

    // along the coast: where the beach starts and how wide it is. one of the two changes at a time,
    // by one cell, so the transitions of columns side by side always touch
    coast.FirstAlong = min(ground.MinX, ground.MinY)
    last := max(ground.MinX + ground.Width, ground.MinY + ground.Height)
    coast.Starts = make([]int, last - coast.FirstAlong)
    coast.Waters = make([]int, last - coast.FirstAlong)

    start := rand.N(coastWander + 1)
    width := coastBeachMin + rand.N(coastBeachMax - coastBeachMin + 1)
    narrowLeft := 0
    narrowWidth := 0
    for along := range coast.Starts {
        if narrowLeft == 0 && rand.N(coastNarrowChance) == 0 {
            narrowLeft = coastNarrowLengthMin + rand.N(coastNarrowLengthMax - coastNarrowLengthMin + 1)
            narrowWidth = coastNarrowMin + rand.N(coastNarrowMax - coastNarrowMin + 1)
        }

        switch {
            case narrowLeft > 0:
                // a narrow stretch: the water comes nearer, the start stays. its length counts
                // from where the beach is as narrow as it gets, so it is never a single notch
                if width > narrowWidth {
                    width -= 1
                } else {
                    narrowLeft -= 1
                }
            case width < coastBeachMin:
                // back to the width of the beach after it
                width += 1
            case rand.N(coastWanderChance) == 0:
                if rand.N(2) == 0 {
                    start = min(max(start + rand.N(3) - 1, 0), coastWander)
                } else {
                    width = min(max(width + rand.N(3) - 1, coastBeachMin), coastBeachMax)
                }
        }
        coast.Starts[along] = start
        coast.Waters[along] = start + width + 1
    }

    ground.Coast = side
    ground.Lines = coast
    ground.Shore = make([]coastPart, ground.Width * ground.Height)

    flat := func(cgx int, cgy int) bool {
        for dy := -coastFlatBefore; dy <= coastFlatBefore; dy++ {
            for dx := -coastFlatBefore; dx <= coastFlatBefore; dx++ {
                if coast.partOf(cgx + dx, cgy + dy) != coastLand {
                    return true
                }
            }
        }
        return false
    }

    for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
        for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
            index := ground.index(cgx, cgy)
            part := coast.partOf(cgx, cgy)
            ground.Shore[index] = part

            switch part {
                case coastLandEdge:
                    // what the landscape has there. mounds have no place under the sand
                    if ground.Group[index] == TerrainRough {
                        ground.Group[index] = TerrainGrass
                    }
                case coastBeach:
                    ground.Group[index] = TerrainSand
                case coastWaterEdge, coastWater:
                    ground.Group[index] = TerrainWater
                    // a road ends at the water
                    ground.Roads[index] = 0
            }
        }
    }

    // flat from a little before the beach on
    if ground.Heights != nil {
        for range 2 {
            for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
                for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
                    if flat(cgx, cgy) {
                        ground.Heights[ground.index(cgx, cgy)] = 0
                    }
                }
            }
            // which raises a cell here and there, so once more after it
            ground.breakLongRuns(zone)
        }
        for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
            for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
                if coast.partOf(cgx, cgy) != coastLand {
                    ground.Heights[ground.index(cgx, cgy)] = 0
                }
            }
        }
    }
}

// what a cell is of the coast. land without one
func (ground *BattleGround) coastAt(cgx int, cgy int) coastPart {
    if ground == nil || ground.Shore == nil || !ground.contains(cgx, cgy) {
        return coastLand
    }
    return ground.Shore[ground.index(cgx, cgy)]
}

// true if a cell is a transition of the coast
func (ground *BattleGround) shoreAt(cgx int, cgy int) bool {
    part := ground.coastAt(cgx, cgy)
    return part == coastLandEdge || part == coastWaterEdge
}

// true if a unit can go into water
func (unit *ArmyUnit) CanEnterWater() bool {
    return unit.IsFlying() || unit.IsSwimmer() || unit.Unit.IsSailing()
}

// true if the tile is water of a coast
func (model *CombatModel) IsWaterTile(x int, y int) bool {
    tile := model.GetTile(x, y)
    return tile != nil && tile.Ground == TerrainWater
}
