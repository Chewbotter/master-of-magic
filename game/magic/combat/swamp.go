package combat

// The pools of a swamp (user, 2026-09-29): "Let's give swamp pools of water throughout the level,
// which are only a couple tiles in area. Tint these brackish green."
//
// A pool is shallow water over the ground of the swamp, as a river is (river.go): a tint that is
// laid over the tiles, with a ragged edge and wet ground along it. Its shape is a number of
// rounds that run into each other, so no two pools are alike.
//
// Pools are of different sizes, from a couple of cells to some 20. They lie all over the ground,
// apart from each other, off the roads, up to the places the armies start in. Not
// in a town or at a lair, not on a coast or in a river. Trees, rocks and
// large pieces keep off them.
//
// RULES, the ones of the water of a river: a cell whose middle is in the water is rough ground, a
// step into it costs twice what grass costs, and units that walk into it splash. A cell a pool
// only reaches into costs what the ground costs.

import (
    "cmp"
    "image"
    "image/color"
    "math"
    "math/rand/v2"
    "slices"

    "github.com/kazzmir/master-of-magic/game/magic/mod"

    "github.com/hajimehoshi/ebiten/v2"
)

// turns them off
var SwampPools = true

// one pool for this many cells of the ground
const swampPoolCells = 45
// places that are tried for a pool
const swampTries = 12
// how far from a round before it a round of a pool lies, as a share of the two radiuses together
const swampRoundAwayMin = 0.45
const swampRoundAwayMax = 0.75

// pools are of different sizes (user, 2026-09-29: "pools of varying size, some of which are bigger
// than this"): of so many pools, by chance, with so many rounds of such a size in cells
type poolSize struct {
    Chance int
    RoundsMin int
    RoundsMax int
    RadiusMin float64
    RadiusMax float64
}

var swampSizes = []poolSize{
    // a couple of cells
    {Chance: 6, RoundsMin: 2, RoundsMax: 3, RadiusMin: 0.55, RadiusMax: 0.9},
    // 5 to 10 cells
    {Chance: 3, RoundsMin: 3, RoundsMax: 5, RadiusMin: 0.8, RadiusMax: 1.3},
    // 12 to 25 cells
    {Chance: 1, RoundsMin: 5, RoundsMax: 8, RadiusMin: 1.1, RadiusMax: 1.7},
}

func randomPoolSize() poolSize {
    all := 0
    for _, size := range swampSizes {
        all += size.Chance
    }
    pick := rand.N(all)
    for _, size := range swampSizes {
        if pick < size.Chance {
            return size
        }
        pick -= size.Chance
    }
    return swampSizes[0]
}
// what a pool reaches beyond its rounds, in cells: the scatter of its edge and the wet ground
const swampBeyond = coastScatter + riverWetWidth

// the water: brackish green. how much of it lies over the ground near the edge and in the middle,
// which starts this many cells from the edge
var swampColor = color.NRGBA{R: 96, G: 108, B: 34, A: 255}
const swampShallow = 0.6
const swampDeep = 0.8
const swampDeepFrom = 0.4

type poolRound struct {
    X float64
    Y float64
    Radius float64
}

// what a cell is of the pools
type poolPart uint8
const (
    poolNone poolPart = iota
    // a pool reaches into the cell
    poolEdge
    // the middle of the cell is in the water
    poolWater
)

// the pools of a ground
type swampPools struct {
    Rounds []poolRound
    // what every cell is of them, and the rounds that reach into it
    Parts []poolPart
    Near map[int][]int
    Seed uint32
}

// how far a point is from a round, in cells: above 0 inside of it
func (round poolRound) depth(cellX float64, cellY float64) float64 {
    return round.Radius - math.Hypot(cellX - round.X, cellY - round.Y)
}

// the cells a round reaches into
func (round poolRound) cells() []image.Point {
    var out []image.Point
    reach := round.Radius + swampBeyond
    for cgy := int(math.Floor(round.Y - reach)); cgy <= int(math.Floor(round.Y + reach)); cgy++ {
        for cgx := int(math.Floor(round.X - reach)); cgx <= int(math.Floor(round.X + reach)); cgx++ {
            // the point of the cell that is nearest to the middle of the round
            nearX := min(max(round.X, float64(cgx)), float64(cgx + 1))
            nearY := min(max(round.Y, float64(cgy)), float64(cgy + 1))
            if math.Hypot(nearX - round.X, nearY - round.Y) < reach {
                out = append(out, image.Pt(cgx, cgy))
            }
        }
    }
    return out
}

// lays the pools of a swamp over the ground, after the coast, the river and the farmland
func (ground *BattleGround) makePools(zone ZoneType) {
    if !SwampPools || zone.Ground.Biome != mod.BiomeSwamp {
        return
    }

    pools := &swampPools{
        Parts: make([]poolPart, ground.Width * ground.Height),
        Near: make(map[int][]int),
        Seed: rand.Uint32(),
    }
    ground.Pools = pools

    placed := make(map[int]bool)

    free := func(cgx int, cgy int) bool {
        if !ground.contains(cgx, cgy) {
            return false
        }
        index := ground.index(cgx, cgy)
        group := ground.Group[index]
        if group == TerrainSand || group == TerrainWater || group == TerrainCrop || ground.Roads[index] != 0 {
            return false
        }
        if ground.coastAt(cgx, cgy) != coastLand || ground.riverAt(cgx, cgy) != riverLand {
            return false
        }
        // up to the places the armies of the battle start in
        return !armyStartsOf(cgx, cgy, zone.Ground.LargeArmy) && !sceneryCellTaken(zone, cgx, cgy, true)
    }

    // the sizes of the pools, the large ones first, which find room less easily
    count := ground.Width * ground.Height / swampPoolCells
    sizes := make([]poolSize, count)
    for index := range sizes {
        sizes[index] = randomPoolSize()
    }
    slices.SortStableFunc(sizes, func(a poolSize, b poolSize) int {
        return cmp.Compare(b.RadiusMax, a.RadiusMax)
    })

    for try := range count * swampTries {
        size := sizes[try / swampTries]
        if placed[try / swampTries] {
            continue
        }
        rounds := []poolRound{{
            X: float64(ground.MinX + rand.N(ground.Width)) + randomPart(0.3, 0.7),
            Y: float64(ground.MinY + rand.N(ground.Height)) + randomPart(0.3, 0.7),
            Radius: randomPart(size.RadiusMin, size.RadiusMax),
        }}
        for range size.RoundsMin - 1 + rand.N(size.RoundsMax - size.RoundsMin + 1) {
            // beside one of the rounds that are there, so a large pool has arms
            from := rounds[rand.N(len(rounds))]
            radius := randomPart(size.RadiusMin, size.RadiusMax)
            angle := rand.Float64() * 2 * math.Pi
            away := randomPart(swampRoundAwayMin, swampRoundAwayMax) * (from.Radius + radius)
            rounds = append(rounds, poolRound{
                X: from.X + math.Cos(angle) * away,
                Y: from.Y + math.Sin(angle) * away,
                Radius: radius,
            })
        }

        // all of its cells are free, and no other pool lies in them or next to them
        fits := true
        for _, round := range rounds {
            for _, cell := range round.cells() {
                if !free(cell.X, cell.Y) {
                    fits = false
                }
                for dy := -1; dy <= 1; dy++ {
                    for dx := -1; dx <= 1; dx++ {
                        if ground.poolAt(cell.X + dx, cell.Y + dy) != poolNone {
                            fits = false
                        }
                    }
                }
            }
        }
        if !fits {
            continue
        }

        for _, round := range rounds {
            number := len(pools.Rounds)
            pools.Rounds = append(pools.Rounds, round)
            for _, cell := range round.cells() {
                index := ground.index(cell.X, cell.Y)
                pools.Near[index] = append(pools.Near[index], number)
                pools.Parts[index] = max(pools.Parts[index], poolEdge)
                if round.depth(float64(cell.X) + 0.5, float64(cell.Y) + 0.5) >= 0 {
                    pools.Parts[index] = poolWater
                }
                // mounds have no place under the water
                if ground.Group[index] == TerrainRough {
                    ground.Group[index] = TerrainGrass
                }
            }
        }
        placed[try / swampTries] = true
    }

    // the ground is flat under a pool and beside it, as it is at a river
    if ground.Heights != nil {
        flatten := func(reach int) {
            for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
                for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
                    if ground.poolAt(cgx, cgy) == poolNone {
                        continue
                    }
                    for dy := -reach; dy <= reach; dy++ {
                        for dx := -reach; dx <= reach; dx++ {
                            if ground.contains(cgx + dx, cgy + dy) {
                                ground.Heights[ground.index(cgx + dx, cgy + dy)] = 0
                            }
                        }
                    }
                }
            }
        }
        for range 2 {
            flatten(1)
            // which raises a cell here and there, so once more after it
            ground.breakLongRuns(zone)
        }
        flatten(0)
    }
}

// what a cell is of the pools of the ground. none without pools
func (ground *BattleGround) poolAt(cgx int, cgy int) poolPart {
    if ground == nil || ground.Pools == nil || !ground.contains(cgx, cgy) {
        return poolNone
    }
    return ground.Pools.Parts[ground.index(cgx, cgy)]
}

// how far a point of a cell lies inside of the water of a pool, in cells from its edge. below 0 it
// is that far from the water
func (ground *BattleGround) poolDepth(cgx int, cgy int, cellX float64, cellY float64) float64 {
    depth := -1000.0
    for _, number := range ground.Pools.Near[ground.index(cgx, cgy)] {
        depth = math.Max(depth, ground.Pools.Rounds[number].depth(cellX, cellY))
    }
    return depth
}

// the water of the pools of a battlefield, to be laid over its ground, and where its top left
// corner lies on the original's screen. nil without pools
func makePoolWater(ground *BattleGround) (*image.NRGBA, int, int) {
    if ground == nil || ground.Pools == nil || len(ground.Pools.Rounds) == 0 {
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

    out := image.NewNRGBA(image.Rect(0, 0, right - left, bottom - top))

    for y := range bottom - top {
        for x := range right - left {
            cellX, cellY := screenToCell(float64(left + x) + 0.5, float64(top + y) + 0.5)
            cgx, cgy := int(math.Floor(cellX)), int(math.Floor(cellY))
            if ground.poolAt(cgx, cgy) == poolNone {
                continue
            }

            tileX, tileY := CellToTile(cgx, cgy)
            brightness := float64(borderBrightness(borderDepth(float64(tileX), float64(tileY))))
            if brightness <= 0 {
                continue
            }

            // a pixel has water of one of two depths over it, or is wet ground, or neither
            depth := ground.poolDepth(cgx, cgy, cellX, cellY) + coastNoise(left + x, top + y, ground.Pools.Seed) * coastScatter
            pixel := swampColor
            switch {
                case depth >= swampDeepFrom: pixel.A = uint8(math.Round(swampDeep * 255))
                case depth >= 0: pixel.A = uint8(math.Round(swampShallow * 255))
                case depth >= -riverWetWidth: pixel = color.NRGBA{A: uint8(math.Round(riverWetDark * 255))}
                default: continue
            }

            pixel.R = uint8(float64(pixel.R) * brightness)
            pixel.G = uint8(float64(pixel.G) * brightness)
            pixel.B = uint8(float64(pixel.B) * brightness)
            out.SetNRGBA(x, y, pixel)
        }
    }

    return out, left, top
}

// the water of the pools, over the ground
func (combat *CombatScreen) drawPools(screen *ebiten.Image) {
    ground := combat.Model.Ground
    if ground == nil || ground.Pools == nil {
        return
    }

    if !combat.poolsMade {
        combat.poolsMade = true
        water, x, y := makePoolWater(ground)
        if water != nil {
            combat.poolWater = ebiten.NewImageFromImage(water)
            combat.poolX = x
            combat.poolY = y
        }
    }
    if combat.poolWater == nil {
        return
    }

    combat.drawOnField(screen, combat.poolWater, combat.poolX, combat.poolY)
}
