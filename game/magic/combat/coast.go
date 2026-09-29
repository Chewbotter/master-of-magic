package combat

// The coast (user, 2026-09-29). Not in the original, where a battle beside the sea is fought on
// land all around.
//
// A battle on land that has the sea beside it on the world map has a coast on that side of the
// field: a strip of beach, then water to the end of the ground. The armies stand on land: the beach
// starts behind the places of the army on that side.
//
// A COAST IS A FEATURE, NOT A BIOME: it lies on whatever the landscape of the battle is. It is the
// first of the features that water and the nearness of towns are to be (see CLAUDE.md).
//
// The field is the world map turned by an eighth: its north is the upper right of the screen. The
// grid has it as the roads of the original do (Build_Road_Network): east is +cgx, west -cgx, north
// -cgy, south +cgy. The defenders stand in the west, the attackers in the east.
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
)

func (side CoastSide) String() string {
    switch side {
        case CoastEast: return "east"
        case CoastNorth: return "north"
        case CoastWest: return "west"
        case CoastSouth: return "south"
    }
    return "none"
}

// the beach starts this many cells behind the places of the armies at the nearest (deploy.go): near
// enough to be seen without moving the view, with room to stand behind the last row. armies of more
// than 12 units take more places, the beach lies farther out then
const coastBehindArmy = 3

// the line of the coast wanders: it lies up to this many cells farther out than its nearest
const coastWander = 2
// one step along the coast in this many moves the line
const coastWanderChance = 3
// how many cells wide the beach is
const coastBeachMin = 2
const coastBeachMax = 3
// plateaus end this many cells before the beach, so their edge is not drawn along it
const coastFlatBefore = 1

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

// how far a cell lies out toward the coast, from where the beach starts at the nearest: 0 the first
// column or row of the beach, below 0 inland. and where it lies along the coast
func (side CoastSide) depth(cgx int, cgy int, large bool) (int, int) {
    firstX, lastX, firstY, lastY := armyPlaces(large)
    switch side {
        case CoastEast: return cgx - (lastX + coastBehindArmy), cgy
        case CoastWest: return firstX - coastBehindArmy - cgx, cgy
        case CoastNorth: return firstY - coastBehindArmy - cgy, cgx
        case CoastSouth: return cgy - (lastY + coastBehindArmy), cgx
    }
    return -1000, 0
}

// lays the coast over the ground: beach and water, flat, without roads into the water
func (ground *BattleGround) makeCoast(zone ZoneType) {
    side := zone.Ground.Coast
    if !CoastGround || side == CoastNone {
        return
    }
    ground.Coast = side

    // along the coast: how far out the beach starts, and how wide it is
    first := min(ground.MinX, ground.MinY)
    last := max(ground.MinX + ground.Width, ground.MinY + ground.Height)
    starts := make([]int, last - first)
    widths := make([]int, last - first)

    start := rand.N(coastWander + 1)
    width := coastBeachMin + rand.N(coastBeachMax - coastBeachMin + 1)
    for along := range starts {
        if rand.N(coastWanderChance) == 0 {
            start = min(max(start + rand.N(3) - 1, 0), coastWander)
        }
        if rand.N(coastWanderChance) == 0 {
            width = coastBeachMin + rand.N(coastBeachMax - coastBeachMin + 1)
        }
        starts[along] = start
        widths[along] = width
    }

    for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
        for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
            depth, along := side.depth(cgx, cgy, zone.Ground.LargeArmy)
            out := depth - starts[along - first]
            index := ground.index(cgx, cgy)

            if out >= -coastFlatBefore && ground.Heights != nil {
                ground.Heights[index] = 0
            }
            if out < 0 {
                continue
            }

            ground.Group[index] = TerrainSand
            if out >= widths[along - first] {
                ground.Group[index] = TerrainWater
                // a road ends at the water
                ground.Roads[index] = 0
            }
        }
    }

    if ground.Heights != nil {
        ground.breakLongRuns(zone)
    }
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
