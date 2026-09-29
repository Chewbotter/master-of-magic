package combat

// Rivers (user, 2026-09-29). Not in the original: its pictures of a river are tiles of grass with a
// band of water one tile wide, and the code that laid them is never run.
//
// A RIVER IS A FEATURE, as the coast is (coast.go): it lies on whatever the landscape of the battle
// is. It is SHALLOW WATER OVER THE GROUND: the tiles are the ones of the landscape, and the water is
// a tint that is laid over them, so the bed of the river is the ground of the biome and the river
// looks as if it can be waded through. It can: every unit can enter it.
//
// THE SHAPE is the one of the coast, decided cell by cell. Across the river every column of cells
// has:
//   land
//   one BANK         the ground, with water over the part toward the river. counts as the ground
//   the RIVER        2 to 3 cells, water over all of them
//   one BANK         as the first
//   land
// From one column to the next the river moves by one cell to a side or changes its width by one,
// never both, so the banks of columns side by side always touch. The edge of the water runs
// through the banks as the edge of the sand does through the transitions of a coast (edgeIn).
//
// WHERE. Between the armies, which have to cross it (RiverAcross), or beside them on the north or
// the south of the field (RiverBeside). Between the armies it keeps to the cells between their
// front rows and wanders farther only beyond the columns they stand in.
//
// RULES. A cell of the river is rough ground: a step into it costs twice what grass costs. Flying
// units pay as everywhere. A bank costs what the ground costs.
//
// INTO THE SEA. A river that runs toward a coast runs into it: over the beach and the sand of
// the cells the sea starts in. The sea comes in to meet it, the beach is narrow there (makeCoast),
// and over its last cells the water of the river turns into the water of the sea, in steps.
//
// ON TUNDRA THE RIVER IS FROZEN, as the sea is (mod/shore.go): ice over the ground, without
// glints and splashes. The rules are the ones of water.
//
// NO ROADS (user, 2026-09-29: "avoid having rivers and roads in the same battle scene", "favor
// rivers over roads if both are feasible"): a battlefield with a river has no roads. A town has
// roads and houses where the river would run, so a battle of a town has no river.
//
// GLINTS. The replacement folder can lay small pictures over the water, which the user draws as
// white pixels that move:
//
//   environment/River/glint 1_0.png, glint 1_1.png, ... glint 2_0.png, ...
//
// From 1 on without a gap, any size, as many frames as wanted. They are scattered over the cells
// of the river, the middle of the picture on its place, each at a time of its own.

import (
    "image"
    "image/color"
    "math"
    "math/rand/v2"

    "github.com/kazzmir/master-of-magic/game/magic/mod"

    "github.com/hajimehoshi/ebiten/v2"
)

// turns them off
var RiverGround = true

type RiverCourse int
const (
    RiverNone RiverCourse = iota
    // between the armies
    RiverAcross
    // beside the armies, on the north or the south of the field
    RiverBeside
    // one of the two by chance, picked when the ground is made
    RiverAny
)

func (course RiverCourse) String() string {
    switch course {
        case RiverAcross: return "across"
        case RiverBeside: return "beside"
        case RiverAny: return "any"
    }
    return "none"
}

// how many cells of water lie between the two banks
const riverWidthMin = 2
const riverWidthMax = 3
// beyond the columns of the armies the river wanders up to this many cells from where it runs
// between them. beside the armies it wanders this many cells
const riverWander = 5
// a river beside the armies has its near bank this many cells beside their places at the nearest
const riverBesideArmy = 2
// along the river: one column in riverMoveChance moves the river, toward the side it is heading
// for, which changes at one column in riverTurnChance. one move in riverWidthChance changes the
// width in place of it
const riverMoveChance = 2
const riverTurnChance = 5
const riverWidthChance = 4
// plateaus end this many cells before the banks
const riverFlatBefore = 1

// the water: its color, and how much of it lies over the ground near the banks and in the middle,
// 0 to 1, and where it runs into the sea, in the color of the sea
type riverLook struct {
    Color color.NRGBA
    Shallow float64
    Deep float64
    Sea color.NRGBA
    Mouth float64
}

var riverWaterLook = riverLook{
    Color: color.NRGBA{R: 38, G: 92, B: 160, A: 255}, Shallow: 0.38, Deep: 0.52,
    Sea: color.NRGBA{R: 24, G: 56, B: 120, A: 255}, Mouth: 0.85,
}
// a river of the tundra is frozen, as its sea is (user, 2026-09-29)
var riverIceLook = riverLook{
    Color: color.NRGBA{R: 160, G: 200, B: 232, A: 255}, Shallow: 0.5, Deep: 0.66,
    Sea: color.NRGBA{R: 186, G: 212, B: 232, A: 255}, Mouth: 0.9,
}

// the middle of a river starts this many cells from the edge of the water
const riverDeepFrom = 0.55
// the water turns into the water of the sea over this many cells before it, in this many steps
const riverMouthLength = 3.0
const riverMouthSteps = 3
// the wet ground along the water: how many cells wide, and how dark
const riverWetWidth = 0.14
const riverWetDark = 0.3

// glints for every cell of water
const riverGlints = 0.5
// a frame of a glint is shown this many times as long as a frame of a prop (user, 2026-09-29:
// "slow the glint motion by half")
const riverGlintSlow = 2
const riverSet = mod.FeatureRiver
const riverGlintName = "glint"

// what a cell is of the river
type riverPart int
const (
    riverLand riverPart = iota
    riverNearBank
    riverFarBank
    riverWater
)

// the course a battle gets: none in a town, one of the two for any
func riverCourse(zone ZoneType) RiverCourse {
    course := zone.Ground.River
    if !RiverGround || zone.City != nil {
        return RiverNone
    }
    if course == RiverAny {
        return []RiverCourse{RiverAcross, RiverBeside}[rand.N(2)]
    }
    return course
}

// the river as it runs along a column: where its near bank lies and how wide it is
type riverWalk struct {
    Start int
    Width int
    Heading int
}

// the next column. the near bank is not nearer than low, the far bank not farther than high
func (walk riverWalk) next(low int, high int) riverWalk {
    out := walk
    if rand.N(riverTurnChance) == 0 {
        out.Heading = -out.Heading
    }
    if rand.N(riverMoveChance) != 0 {
        return out
    }

    if rand.N(riverWidthChance) == 0 {
        out.Width = riverWidthMin + rand.N(riverWidthMax - riverWidthMin + 1)
        // by one cell
        out.Width = min(max(out.Width, walk.Width - 1), walk.Width + 1)
    } else {
        out.Start += out.Heading
    }

    if out.Start < low || out.Start + out.Width + 1 > high {
        // against its limit: it stays and turns
        out = walk
        out.Heading = -walk.Heading
    }
    return out
}

// where the river runs along the ground, before the coast is made. the course was picked when
// the ground was made
func (ground *BattleGround) planRiver(zone ZoneType) {
    if ground.River == RiverNone {
        return
    }

    _, _, firstY, lastY := armyPlaces(zone.Ground.LargeArmy)
    stream := &coastLines{Seed: rand.Uint32()}

    // the limits of the near and the far bank along the river, by the cells from the middle
    var limits func(along int) (int, int)
    middle := (firstY + lastY) / 2

    switch ground.River {
        case RiverAcross:
            // across the grid from west to east, between the front rows of the armies
            stream.Side = CoastEast
            stream.First = deployDefenderRows[0] + 1
            between := deployAttackerRows[0] - 1 - stream.First
            limits = func(along int) (int, int) {
                beyond := min(max(0, firstY - along, along - lastY), riverWander)
                return -beyond, between + beyond
            }
        default:
            // on the side the sea is not on
            north := rand.N(2) == 0
            switch ground.Coast {
                case CoastNorth: north = false
                case CoastSouth: north = true
            }
            if north {
                stream.Side = CoastNorth
                stream.First = firstY - riverBesideArmy
            } else {
                stream.Side = CoastSouth
                stream.First = lastY + riverBesideArmy
            }
            limits = func(along int) (int, int) {
                return 0, riverWander + riverWidthMax + 1
            }
    }

    stream.FirstAlong = min(ground.MinX, ground.MinY)
    last := max(ground.MinX + ground.Width, ground.MinY + ground.Height)
    stream.Starts = make([]int, last - stream.FirstAlong)
    stream.Waters = make([]int, last - stream.FirstAlong)

    // from the middle of the field to both of its ends, where the limits are the same or wider
    // from column to column
    low, high := limits(middle)
    first := riverWalk{Width: riverWidthMin + rand.N(riverWidthMax - riverWidthMin + 1), Heading: 1 - rand.N(2) * 2}
    first.Width = min(first.Width, high - low - 1)
    first.Start = low + rand.N(high - low - first.Width)

    set := func(along int, walk riverWalk) {
        stream.Starts[along - stream.FirstAlong] = walk.Start
        stream.Waters[along - stream.FirstAlong] = walk.Start + walk.Width + 1
    }
    set(middle, first)
    walk := first
    for along := middle + 1; along < last; along++ {
        walk = walk.next(limits(along))
        set(along, walk)
    }
    walk = first
    walk.Heading = -walk.Heading
    for along := middle - 1; along >= stream.FirstAlong; along-- {
        walk = walk.next(limits(along))
        set(along, walk)
    }

    ground.Stream = stream
}

// the cells of the grid a river takes across itself at a place along it, with its banks: the
// first and the last
func (stream *coastLines) span(along int) (int, int) {
    near := stream.at(stream.Starts, along)
    far := stream.at(stream.Waters, along)
    switch stream.Side {
        case CoastWest, CoastNorth: return stream.First - far, stream.First - near
    }
    return stream.First + near, stream.First + far
}

// true if a coast runs across the grid as the columns of x do
func (coast *coastLines) acrossX() bool {
    return coast.Side == CoastEast || coast.Side == CoastWest
}

// where along a coast the river of the ground runs into the sea: the first and the last cell.
// false if it does not, which is without a river and with one that runs along the coast
func (ground *BattleGround) riverMouth(coast *coastLines) (int, int, bool) {
    stream := ground.Stream
    if stream == nil || stream.acrossX() == coast.acrossX() {
        return 0, 0, false
    }

    // from where the beach can start to where the water starts at the farthest
    out := 1
    if coast.Side == CoastWest || coast.Side == CoastNorth {
        out = -1
    }
    first, last := stream.span(coast.First)
    for depth := 1; depth <= coastWander + coastBeachMax + 1; depth++ {
        low, high := stream.span(coast.First + out * depth)
        first = min(first, low)
        last = max(last, high)
    }
    return first, last, true
}

// lays the river over the ground, after the coast
func (ground *BattleGround) makeRiver(zone ZoneType) {
    stream := ground.Stream
    if stream == nil {
        return
    }
    ground.Banks = make([]riverPart, ground.Width * ground.Height)

    for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
        for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
            index := ground.index(cgx, cgy)
            // the river ends in the sea: it runs on over the sand of the cells where the water
            // starts
            if ground.coastAt(cgx, cgy) == coastWater {
                continue
            }

            part := riverLand
            switch stream.partOf(cgx, cgy) {
                case coastLandEdge: part = riverNearBank
                case coastBeach: part = riverWater
                case coastWaterEdge: part = riverFarBank
            }
            ground.Banks[index] = part

            // mounds have no place under the water
            if part != riverLand && ground.Group[index] == TerrainRough {
                ground.Group[index] = TerrainGrass
            }
        }
    }

    // flat from a little before the banks on
    if ground.Heights != nil {
        near := func(cgx int, cgy int, reach int) bool {
            for dy := -reach; dy <= reach; dy++ {
                for dx := -reach; dx <= reach; dx++ {
                    if ground.riverAt(cgx + dx, cgy + dy) != riverLand {
                        return true
                    }
                }
            }
            return false
        }
        flatten := func(reach int) {
            for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
                for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
                    if near(cgx, cgy, reach) {
                        ground.Heights[ground.index(cgx, cgy)] = 0
                    }
                }
            }
        }

        for range 2 {
            flatten(riverFlatBefore)
            // which raises a cell here and there, so once more after it
            ground.breakLongRuns(zone)
        }
        flatten(0)
    }
}

// what a cell is of the river. land without one
func (ground *BattleGround) riverAt(cgx int, cgy int) riverPart {
    if ground == nil || ground.Banks == nil || !ground.contains(cgx, cgy) {
        return riverLand
    }
    return ground.Banks[ground.index(cgx, cgy)]
}

// true if the tile is shallow water as a whole: of a river, or of a pool of a swamp (swamp.go)
func (model *CombatModel) IsRiverTile(x int, y int) bool {
    cgx, cgy := TileToCell(x, y)
    return model.Ground.riverAt(cgx, cgy) == riverWater || model.Ground.poolAt(cgx, cgy) == poolWater
}

// how far a point lies inside of the water, in cells from its nearer edge. below 0 it is that far
// from the water. the point is the pixel pixelX, pixelY of the field, which makes the edge ragged
func (stream *coastLines) waterDepth(cellX float64, cellY float64, pixelX int, pixelY int) float64 {
    depth, along := stream.depth(cellX, cellY)
    depth += coastNoise(pixelX, pixelY, stream.Seed) * coastScatter

    column := int(math.Floor(along))
    near := float64(stream.at(stream.Starts, column)) + stream.edgeIn(stream.Starts, along)
    far := float64(stream.at(stream.Waters, column)) + stream.edgeIn(stream.Waters, along)
    return math.Min(depth - near, far - depth)
}

// the water of the river of a battlefield, to be laid over its ground, and where its top left
// corner lies on the original's screen. nil without a river
func makeRiverWater(ground *BattleGround) (*image.NRGBA, int, int) {
    if ground == nil || ground.Stream == nil {
        return nil, 0, 0
    }

    // the cells of the ground lie in a diamond on the screen, between the middles of its four
    // corner cells
    lastX := ground.MinX + ground.Width - 1
    lastY := ground.MinY + ground.Height - 1
    left, _ := cellCenterScreen(ground.MinX, lastY)
    right, _ := cellCenterScreen(lastX, ground.MinY)
    _, top := cellCenterScreen(ground.MinX, ground.MinY)
    _, bottom := cellCenterScreen(lastX, lastY)
    left -= CellStepX
    right += CellStepX
    top -= CellStepY
    bottom += CellStepY

    width := right - left
    height := bottom - top
    out := image.NewNRGBA(image.Rect(0, 0, width, height))
    any := false

    look := riverWaterLook
    if ground.Frozen {
        look = riverIceLook
    }

    for y := range height {
        for x := range width {
            cellX, cellY := screenToCell(float64(left + x) + 0.5, float64(top + y) + 0.5)
            cgx, cgy := int(math.Floor(cellX)), int(math.Floor(cellY))
            if ground.riverAt(cgx, cgy) == riverLand {
                continue
            }
            // where the sea starts the river lies on the sand, the sea takes it from there
            if ground.coastAt(cgx, cgy) == coastWaterEdge && !ground.Lines.sandIn(coastWaterEdge, cellX, cellY, left + x, top + y) {
                continue
            }

            tileX, tileY := CellToTile(cgx, cgy)
            brightness := float64(borderBrightness(borderDepth(float64(tileX), float64(tileY))))
            if brightness <= 0 {
                continue
            }

            // a pixel has water of one of two depths over it, or is wet ground, or neither. there
            // is nothing in between, as in the art
            depth := ground.Stream.waterDepth(cellX, cellY, left + x, top + y)
            if ground.riverAt(cgx, cgy) == riverWater {
                // the ragged edge is for the banks: a cell of the river is water as a whole
                depth = math.Max(depth, 0)
            }
            pixel := look.Color
            switch {
                case depth >= riverDeepFrom: pixel = ground.mouthPixel(look, look.Deep, cellX, cellY, left + x, top + y)
                case depth >= 0: pixel = ground.mouthPixel(look, look.Shallow, cellX, cellY, left + x, top + y)
                case depth >= -riverWetWidth: pixel = color.NRGBA{A: uint8(math.Round(riverWetDark * 255))}
                default: continue
            }

            pixel.R = uint8(float64(pixel.R) * brightness)
            pixel.G = uint8(float64(pixel.G) * brightness)
            pixel.B = uint8(float64(pixel.B) * brightness)
            out.SetNRGBA(x, y, pixel)
            any = true
        }
    }

    if !any {
        return nil, 0, 0
    }
    return out, left, top
}

// a pixel of the water of a river, of which so much lies over the ground: near the sea more of
// it, and in the color of the sea, in steps
func (ground *BattleGround) mouthPixel(look riverLook, share float64, cellX float64, cellY float64, pixelX int, pixelY int) color.NRGBA {
    pixel := look.Color
    coast := ground.Lines

    if coast != nil && ground.Stream.acrossX() != coast.acrossX() {
        depth, along := coast.depth(cellX, cellY)
        column := int(math.Floor(along))
        sea := float64(coast.at(coast.Waters, column)) + coast.edgeIn(coast.Waters, along)

        // 0 far from the sea, 1 at it
        near := 1 - (sea - depth) / riverMouthLength + coastNoise(pixelX, pixelY, ground.Stream.Seed + 1) * 0.5 / riverMouthSteps
        step := min(max(math.Floor(near * riverMouthSteps + 0.5), 0), riverMouthSteps) / riverMouthSteps

        mix := func(from uint8, to uint8) uint8 {
            return uint8(math.Round(float64(from) + (float64(to) - float64(from)) * step))
        }
        pixel = color.NRGBA{R: mix(look.Color.R, look.Sea.R), G: mix(look.Color.G, look.Sea.G), B: mix(look.Color.B, look.Sea.B), A: 255}
        share += (look.Mouth - share) * step
    }

    pixel.A = uint8(math.Round(share * 255))
    return pixel
}

// the glints on the water of a river, of the pictures of the replacement folder. ice has none
func makeGlints(ground *BattleGround) []SceneryPiece {
    var out []SceneryPiece
    if ground == nil || ground.Banks == nil || ground.Frozen {
        return out
    }

    // the game has none of its own
    pool := makeSceneryPool(mod.BiomeFolder(riverSet, ground.BaseSet), riverGlintName, "", 0, 0)
    if pool.Count + pool.Extras <= 0 {
        return out
    }

    for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
        for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
            if ground.riverAt(cgx, cgy) != riverWater || rand.Float64() >= riverGlints {
                continue
            }
            screenX, screenY := cellScreen(cgx, cgy, roll(propSubcellMax), roll(propSubcellMax))
            out = append(out, pool.piece(SceneryFlat, screenX, screenY))
        }
    }

    return out
}

// the water of the river, over the ground and the sand of a coast
func (combat *CombatScreen) drawRiver(screen *ebiten.Image) {
    ground := combat.Model.Ground
    if ground == nil || ground.Stream == nil {
        return
    }

    if !combat.riverMade {
        combat.riverMade = true
        water, x, y := makeRiverWater(ground)
        if water != nil {
            combat.riverWater = ebiten.NewImageFromImage(water)
            combat.riverX = x
            combat.riverY = y
        }
    }
    if combat.riverWater == nil {
        return
    }

    combat.drawOnField(screen, combat.riverWater, combat.riverX, combat.riverY)
}
