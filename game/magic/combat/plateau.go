package combat

// Hills as plateaus (user request, 2026-09-28, not in the original). The original's rough ground
// wanders in lines one cell wide and its pictures show ridges, which on grass land often reads as
// snaking mounds. Here the same patches become raised ground of one height: small hills of one cell,
// and patches grown and smoothed into plateaus that cover larger areas. The top of a plateau is the
// usual ground, drawn a little lighter; where it drops to the ground around it, a slope lies along
// the edge. Only going up or down a slope costs more (movecost.go), the top is ground as any other.
//
// Nothing is raised on the screen: a slope is a picture over the ground of the raised cell, like the
// marks of spells. The game makes a first set of them (slopePicture); pictures of the replacement
// folder go before them:
//
//   environment/Slopes/<piece>_0.png     edge NE, edge SE, edge SW, edge NW, inner N, inner E,
//                                        inner S, inner W
//
// An edge lies along that edge of the tile where the next cell is lower. An inner corner is where
// the two edges next to a corner are raised and the cell beyond the corner is not. They can have any
// size and colors, and see-through pixels darken or lighten the ground under them. Their top left
// corner lies where the top left corner of the ground picture does.
//
// PlateauGround false brings back the original's rough ground.

import (
    "image"
    "image/color"
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/mod"
    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
)

var PlateauGround = true

// every cell of a patch of rough raises the cells next to it too, then the edges are smoothed: a
// cell with this many raised of its 8 neighbors is raised, one with this few or less is not
const plateauSmoothPasses = 2
const plateauFillNeighbors = 5
const plateauDropNeighbors = 2
// small hills of a single cell, for every patch of rough ground this many
const plateauSingleHillsPerPatch = 0.5

// the top of a plateau is drawn this much brighter
const plateauTopBrightness = 1.15

// the first set of slopes: how far into the tile the band of an edge reaches, 0 to 1 of the way to
// the middle, and how dark it is at the edge. edges toward the viewer are the slopes seen from the
// front; the far edges only show a light rim
const slopeBandWidth = 0.9
const slopeInnerWidth = 0.7
const slopeDarkSE = 0.65
const slopeDarkSW = 0.5
const slopeLightFar = 0.2

const slopesSet = "Slopes"

// the pieces of slope, in the order of slopeSides
var slopeEdgeNames = []string{"edge NE", "edge SE", "edge SW", "edge NW"}
var slopeInnerNames = []string{"inner N", "inner E", "inner S", "inner W"}

func SlopePieceNames() []string {
    return append(append([]string{}, slopeEdgeNames...), slopeInnerNames...)
}

func (ground *BattleGround) HeightAt(cgx int, cgy int) int {
    if ground == nil || !ground.contains(cgx, cgy) || ground.Heights == nil {
        return 0
    }
    return ground.Heights[ground.index(cgx, cgy)]
}

func (ground *BattleGround) setHeight(cgx int, cgy int, height int) {
    if ground.contains(cgx, cgy) {
        ground.Heights[ground.index(cgx, cgy)] = height
    }
}

func (ground *BattleGround) raisedNeighbors(cgx int, cgy int) int {
    count := 0
    for dy := -1; dy <= 1; dy++ {
        for dx := -1; dx <= 1; dx++ {
            if (dx != 0 || dy != 0) && ground.HeightAt(cgx + dx, cgy + dy) > 0 {
                count += 1
            }
        }
    }
    return count
}

// the cells a plateau stays out of: the town, the lair or node, as rough ground does
func plateauKeepsOut(zone ZoneType, cgx int, cgy int) bool {
    if zone.City == nil && zone.Encounter == ZoneNone {
        return false
    }
    return cgx >= townCellMinX && cgx <= townCellMaxX && cgy >= townCellMinY && cgy <= townCellMaxY
}

// turns the rough ground the original's patches made into plateaus and small hills
func (ground *BattleGround) makePlateaus(zone ZoneType, patches int) {
    ground.Heights = make([]int, ground.Width * ground.Height)

    lastX := ground.MinX + ground.Width
    lastY := ground.MinY + ground.Height

    // the patches, and the cells next to them
    for cgy := ground.MinY; cgy < lastY; cgy++ {
        for cgx := ground.MinX; cgx < lastX; cgx++ {
            if ground.GroupAt(cgx, cgy) != TerrainRough {
                continue
            }
            ground.setGroup(cgx, cgy, TerrainGrass)
            ground.setHeight(cgx, cgy, 1)
            for step := 1; step <= 4; step++ {
                ground.setHeight(cgx + stepCellX[step], cgy + stepCellY[step], 1)
            }
        }
    }

    // smooth edges: fill the bays, drop the spurs
    for range plateauSmoothPasses {
        next := make([]int, len(ground.Heights))
        for cgy := ground.MinY; cgy < lastY; cgy++ {
            for cgx := ground.MinX; cgx < lastX; cgx++ {
                index := ground.index(cgx, cgy)
                neighbors := ground.raisedNeighbors(cgx, cgy)
                next[index] = ground.Heights[index]
                if neighbors >= plateauFillNeighbors {
                    next[index] = 1
                } else if neighbors <= plateauDropNeighbors {
                    next[index] = 0
                }
            }
        }
        ground.Heights = next
    }

    // small hills of one cell, on their own
    singles := int(math.Round(float64(patches) * plateauSingleHillsPerPatch))
    for range singles {
        for range sceneryMaxTries {
            cgx := ground.MinX + roll(ground.Width) - 1
            cgy := ground.MinY + roll(ground.Height) - 1
            if ground.HeightAt(cgx, cgy) == 0 && ground.raisedNeighbors(cgx, cgy) == 0 {
                ground.setHeight(cgx, cgy, 1)
                break
            }
        }
    }

    for cgy := ground.MinY; cgy < lastY; cgy++ {
        for cgx := ground.MinX; cgx < lastX; cgx++ {
            if plateauKeepsOut(zone, cgx, cgy) {
                ground.setHeight(cgx, cgy, 0)
            }
        }
    }
}

// the neighbors across the edges NE, SE, SW, NW of a cell, and beyond its corners N, E, S, W
var slopeSides = [4]image.Point{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}
var slopeCorners = [4]image.Point{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}}

// which pieces of slope lie on a raised cell: indexes into SlopePieceNames
func (ground *BattleGround) slopePieces(cgx int, cgy int) []int {
    height := ground.HeightAt(cgx, cgy)
    if height == 0 {
        return nil
    }

    var out []int
    lower := func(offset image.Point) bool {
        return ground.HeightAt(cgx + offset.X, cgy + offset.Y) < height
    }

    for side, offset := range slopeSides {
        if lower(offset) {
            out = append(out, side)
        }
    }
    // the corner N lies between the edges NW and NE, E between NE and SE, and so on
    for corner, offset := range slopeCorners {
        before := slopeSides[(corner + 3) % 4]
        after := slopeSides[corner]
        if lower(offset) && !lower(before) && !lower(after) {
            out = append(out, len(slopeEdgeNames) + corner)
        }
    }
    return out
}

// true if a step between two cells goes up or down a slope
func (ground *BattleGround) isSlope(fromX int, fromY int, toX int, toY int) bool {
    if ground == nil || ground.Heights == nil {
        return false
    }
    return ground.HeightAt(fromX, fromY) != ground.HeightAt(toX, toY)
}

// the first set of slopes, see the top of this file. a picture as large as the ground picture, with
// the band of the piece in black or white of some opacity
func SlopePicture(piece int) image.Image {
    const width = 30
    const height = 16
    out := image.NewNRGBA(image.Rect(0, 0, width, height))

    centerX := float64(width) / 2
    centerY := float64(height) / 2

    for y := range height {
        for x := range width {
            // where the pixel lies in the diamond: a - b runs from the SW edge (-1) to the NE edge (1),
            // a + b from the NW edge (-1) to the SE edge (1)
            a := (float64(x) + 0.5 - centerX) / centerX
            b := (float64(y) + 0.5 - centerY) / centerY
            if math.Abs(a) + math.Abs(b) > 1 {
                continue
            }

            // how close to the edges, 0 at the edge
            toNE := 1 - (a - b)
            toSE := 1 - (a + b)
            toSW := 1 + (a - b)
            toNW := 1 + (a + b)

            var shade float64
            var light bool

            // 1 at the edge down to 0 at reach, the middle of the tile being 1 away
            band := func(distance float64, reach float64) float64 {
                if distance >= reach {
                    return 0
                }
                return 1 - distance / reach
            }

            switch piece {
                case 0: shade, light = band(toNE, slopeBandWidth / 2) * slopeLightFar, true
                case 1: shade = band(toSE, slopeBandWidth) * slopeDarkSE
                case 2: shade = band(toSW, slopeBandWidth) * slopeDarkSW
                case 3: shade, light = band(toNW, slopeBandWidth / 2) * slopeLightFar, true
                // inner corners: near both edges of the corner
                case 4: shade, light = band(math.Max(toNW, toNE), slopeInnerWidth / 2) * slopeLightFar, true
                case 5: shade = band(math.Max(toNE, toSE), slopeInnerWidth) * slopeDarkSE / 2
                case 6: shade = band(math.Max(toSE, toSW), slopeInnerWidth) * slopeDarkSE
                case 7: shade = band(math.Max(toSW, toNW), slopeInnerWidth) * slopeDarkSW
            }

            if shade <= 0 {
                continue
            }
            value := uint8(0)
            if light {
                value = 255
            }
            out.SetNRGBA(x, y, color.NRGBA{R: value, G: value, B: value, A: uint8(math.Round(shade * 255))})
        }
    }

    return out
}

// the pictures of the slopes: the ones of the replacement folder, or the first set
func (combat *CombatScreen) slopePictures() []*ebiten.Image {
    if combat.slopeCache != nil {
        return combat.slopeCache
    }

    for piece, name := range SlopePieceNames() {
        source := mod.ReadEnvironmentExtra(slopesSet, name)
        if source == nil {
            source = SlopePicture(piece)
        }
        combat.slopeCache = append(combat.slopeCache, ebiten.NewImageFromImage(source))
    }
    return combat.slopeCache
}

// the slopes on a tile, drawn with the options of its ground picture
func (combat *CombatScreen) drawSlopes(screen *ebiten.Image, x int, y int, options *ebiten.DrawImageOptions) {
    ground := combat.Model.Ground
    if ground == nil || ground.Heights == nil {
        return
    }

    cgx, cgy := TileToCell(x, y)
    pieces := ground.slopePieces(cgx, cgy)
    if len(pieces) == 0 {
        return
    }

    pictures := combat.slopePictures()
    for _, piece := range pieces {
        scale.DrawScaled(screen, pictures[piece], options)
    }
}

// how much brighter the ground of a tile is drawn
func (combat *CombatScreen) groundBrightness(x int, y int) float32 {
    if combat.Model.Ground == nil {
        return 1
    }
    cgx, cgy := TileToCell(x, y)
    if combat.Model.Ground.HeightAt(cgx, cgy) > 0 {
        return plateauTopBrightness
    }
    return 1
}
