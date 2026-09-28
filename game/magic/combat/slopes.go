package combat

// The slopes of plateaus (plateau.go). Not in the original.
//
// Nothing is raised on the screen: slopes are pictures laid over the ground, like the marks of
// spells. Along an edge of a raised cell where the next cell is lower lies a piece of slope. On the
// edges toward the viewer (SE and SW) it hangs down below the edge, over the lower ground, so the
// plateau looks taller than a tile would let it (slopeFaceDepth); on the far edges (NE and NW) it
// rises a little above the edge, as seen from above (slopeFarDepth). A slope is shaded light at its
// top edge, bending over to a dark crease near its foot (slopeProfile).
//
// Where exactly two edges of a cell that meet at a corner drop, the cell is cut in half from corner
// to corner and the slope hangs from that line (diagonal N, E, S, W; user, 2026-09-28: "diagonal
// shapes would go a long way"). Along a plateau whose outline runs across the grid, which on the
// screen is a staircase of tiles, every step is cut the same way, so the steps make one straight
// line. It is only the look: the whole cell is still raised. Where two edges of a cell drop, its corner is cut in one of several shapes, or left
// square, so the corners of plateaus do not all look alike. Where the cell beyond a corner is lower
// but the two edges beside it are not, an inner corner fills the notch.
//
// The game makes a first set of the pictures (SlopePicture). Pictures of the replacement folder go
// before them:
//
//   environment/Slopes/edge NE_0.png ... edge NW_0.png
//   environment/Slopes/inner N_0.png ... inner W_0.png
//   environment/Slopes/outer N 1_0.png, outer N 2_0.png, ...   from 1 on, as many as there are
//   environment/Slopes/diagonal N_0.png ... diagonal W_0.png
//
// Every picture is slopeCanvasWidth by slopeCanvasHeight, the ground picture of the tile lying
// slopeCanvasLeft right of and slopeCanvasTop below its top left corner, so a face can hang below
// the tile. Any colors; see-through pixels darken or lighten the ground under them.
//
// All slopes are drawn after all the ground, so the face of one cell lies over the ground of the
// next.

import (
    "fmt"
    "image"
    "image/color"
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/mod"
    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
)

// the picture of a piece, and where the ground picture of the tile lies in it
const slopeCanvasWidth = 46
const slopeCanvasHeight = 40
const slopeCanvasLeft = 8
const slopeCanvasTop = 4

// the ground picture of a tile is 30 by 16, but tiles lie 32 apart: the diamond of a cell is 32
// wide, its left corner one pixel left of the picture. slopes follow the diamond, so the slopes of
// cells side by side meet without a gap
const slopeTileHeight = 16
const slopeDiamondCenterX = 15
const slopeDiamondHalfWidth = CellStepX
const slopeDiamondHalfHeight = CellStepY

// the slope toward the viewer (SE and SW edges) reaches this far below the edge, the one away from
// the viewer (NE and NW) this far above it, in art pixels. the taller, the higher a plateau looks
const slopeFaceDepth = 6
const slopeFarDepth = 3
// how dark each slope is at its foot, 0 to 1. SE is turned away from the light more than SW
const slopeDarkSE = 0.55
const slopeDarkSW = 0.42
const slopeDarkFar = 0.3
// the shading of a slope runs from light at the top edge to its darkest at the crease, this far
// down the slope (0 to 1), and fades into the ground below it
const slopeCrease = 0.8
const slopeTopShade = 0.1
// a line of light on the top edge, how light
const slopeLipLight = 0.18
// the notch of an inner corner, in cells from the corner
const slopeInnerReach = 0.35

// how many of the corners where two edges drop stay square, 0 to 1. the rest take a cut
const slopeSquareCorners = 0.25

// the cuts of a corner the game has, in cells from the corner
type slopeCut struct {
    // a straight cut across the corner, this far along both edges
    Bevel float64
    // a round cut of this radius
    Round float64
    // a square bite of this size
    Bite float64
}

var slopeCuts = []slopeCut{
    {Bevel: 0.3},
    {Bevel: 0.5},
    {Round: 0.4},
    {Bite: 0.25},
}

const slopesSet = "Slopes"

var slopeEdgeNames = []string{"edge NE", "edge SE", "edge SW", "edge NW"}
var slopeInnerNames = []string{"inner N", "inner E", "inner S", "inner W"}
var slopeCornerNames = []string{"N", "E", "S", "W"}
var slopeDiagonalNames = []string{"diagonal N", "diagonal E", "diagonal S", "diagonal W"}

func slopeOuterName(corner int, number int) string {
    return fmt.Sprintf("outer %v %v", slopeCornerNames[corner], number)
}

// the pieces of slope the game has, by name, for the export
func SlopePictures() map[string]image.Image {
    out := make(map[string]image.Image)
    for side, name := range slopeEdgeNames {
        out[name] = slopeEdgePicture(side)
    }
    for corner, name := range slopeInnerNames {
        out[name] = slopeInnerPicture(corner)
    }
    for corner, name := range slopeDiagonalNames {
        out[name] = slopeDiagonalPicture(corner)
    }
    for corner := range slopeCornerNames {
        for number, cut := range slopeCuts {
            out[slopeOuterName(corner, number + 1)] = slopeOuterPicture(corner, cut)
        }
    }
    return out
}

// the middle of a pixel of a picture of slope, in the pixels of the tile's ground picture
func slopeTilePoint(x int, y int) (float64, float64) {
    return float64(x - slopeCanvasLeft) + 0.5, float64(y - slopeCanvasTop) + 0.5
}

// a point of a picture of slope in the cells of the grid: u runs from the NW edge (0) to the SE edge
// (1), v from the NE edge (0) to the SW edge (1). inside is false beyond the diamond
func slopeCell(x int, y int) (float64, float64, bool) {
    tileX, tileY := slopeTilePoint(x, y)
    a := (tileX - slopeDiamondCenterX) / slopeDiamondHalfWidth
    b := (tileY - slopeDiamondHalfHeight) / slopeDiamondHalfHeight
    u := (a + b + 1) / 2
    v := (b - a + 1) / 2
    return u, v, math.Abs(a) + math.Abs(b) <= 1
}

// how far a point is from the edges NE, SE, SW, NW of its cell, in cells
func slopeEdgeDistances(u float64, v float64) [4]float64 {
    return [4]float64{v, 1 - u, 1 - v, u}
}

func slopeShade(dark bool, amount float64) color.NRGBA {
    value := uint8(255)
    if dark {
        value = 0
    }
    return color.NRGBA{R: value, G: value, B: value, A: uint8(math.Round(math.Min(1, math.Max(0, amount)) * 255))}
}

func newSlopeCanvas() *image.NRGBA {
    return image.NewNRGBA(image.Rect(0, 0, slopeCanvasWidth, slopeCanvasHeight))
}

// how dark a slope is at a point, 0 at its top edge to 1 at its foot: light at the top, rounding
// over to the darkest at the crease, then fading into the ground
func slopeProfile(t float64) float64 {
    if t <= 0 || t >= 1 {
        return 0
    }
    if t <= slopeCrease {
        s := t / slopeCrease
        // smooth, so the slope looks bent, not flat
        s = s * s * (3 - 2 * s)
        return slopeTopShade + (1 - slopeTopShade) * s
    }
    return (1 - t) / (1 - slopeCrease)
}

// the y of an edge of the diamond at a column, and whether the column lies under it. the columns of
// an edge are half open, so the edges of cells side by side do not overlap
func slopeEdgeY(side int, tileX float64) (float64, bool) {
    left := float64(slopeDiamondCenterX - slopeDiamondHalfWidth)
    right := float64(slopeDiamondCenterX + slopeDiamondHalfWidth)
    middle := float64(slopeDiamondCenterX)
    rise := float64(slopeDiamondHalfHeight) / float64(slopeDiamondHalfWidth)
    switch side {
        case 0: return slopeDiamondHalfHeight - (right - tileX) * rise, tileX >= middle && tileX < right
        case 1: return slopeDiamondHalfHeight + (right - tileX) * rise, tileX >= middle && tileX < right
        case 2: return slopeDiamondHalfHeight + (tileX - left) * rise, tileX >= left && tileX < middle
    }
    return slopeDiamondHalfHeight - (tileX - left) * rise, tileX >= left && tileX < middle
}

// the piece along an edge: toward the viewer the slope hangs below the edge, away from the viewer
// it rises above it, shorter, as seen from above
func slopeEdgePicture(side int) image.Image {
    out := newSlopeCanvas()
    near := side == 1 || side == 2

    depth := float64(slopeFarDepth)
    dark := slopeDarkFar
    switch side {
        case 1:
            depth, dark = slopeFaceDepth, slopeDarkSE
        case 2:
            depth, dark = slopeFaceDepth, slopeDarkSW
    }

    for y := range slopeCanvasHeight {
        for x := range slopeCanvasWidth {
            tileX, tileY := slopeTilePoint(x, y)
            edgeY, under := slopeEdgeY(side, tileX)
            if !under {
                continue
            }

            // how far beyond the edge, outward from the cell: down for the near edges, up for the far
            beyond := tileY - edgeY
            if !near {
                beyond = edgeY - tileY
            }

            switch {
                case beyond > 0:
                    if shade := dark * slopeProfile(beyond / depth); shade > 0 {
                        out.SetNRGBA(x, y, slopeShade(true, shade))
                    }
                case beyond > -1:
                    // the top edge itself
                    out.SetNRGBA(x, y, slopeShade(false, slopeLipLight))
            }
        }
    }

    return out
}

// how a corner looks where it drops: N away from the viewer, S toward it
func slopeCornerShade(corner int) float64 {
    switch corner {
        case 0: return slopeDarkFar
        case 1: return (slopeDarkSE + slopeDarkFar) / 2
        case 2: return (slopeDarkSE + slopeDarkSW) / 2
    }
    return (slopeDarkSW + slopeDarkFar) / 2
}

// the two edges beside a corner, as indexes of slopeEdgeDistances: N lies between NW and NE
func slopeCornerEdges(corner int) (int, int) {
    return (corner + 3) % 4, corner
}

// the notch of an inner corner: a short slope bending around it
func slopeInnerPicture(corner int) image.Image {
    out := newSlopeCanvas()
    before, after := slopeCornerEdges(corner)
    dark := slopeCornerShade(corner)

    for y := range slopeCanvasHeight {
        for x := range slopeCanvasWidth {
            u, v, inside := slopeCell(x, y)
            if !inside {
                continue
            }
            distances := slopeEdgeDistances(u, v)
            reach := math.Max(distances[before], distances[after])
            if reach < slopeInnerReach {
                // from the top edge of the notch down to the corner
                if shade := dark * slopeProfile(1 - reach / slopeInnerReach); shade > 0 {
                    out.SetNRGBA(x, y, slopeShade(true, shade))
                }
            }
        }
    }
    return out
}

// a corner cut away where two edges drop: the slope runs across the cut, from its edge down to the
// corner
func slopeOuterPicture(corner int, cut slopeCut) image.Image {
    out := newSlopeCanvas()
    before, after := slopeCornerEdges(corner)
    dark := slopeCornerShade(corner)

    for y := range slopeCanvasHeight {
        for x := range slopeCanvasWidth {
            u, v, inside := slopeCell(x, y)
            if !inside {
                continue
            }
            distances := slopeEdgeDistances(u, v)
            s, t := distances[before], distances[after]

            // how far into the cut, 0 at its edge to 1 at the corner. below 0 is not cut
            into := -1.0
            switch {
                case cut.Bevel > 0:
                    into = 1 - (s + t) / cut.Bevel
                case cut.Round > 0:
                    r := cut.Round
                    if s < r && t < r {
                        into = (math.Hypot(r - s, r - t) - r) / (r * (math.Sqrt2 - 1))
                    }
                case cut.Bite > 0:
                    into = 1 - math.Max(s, t) / cut.Bite
            }

            switch {
                case into > 0:
                    // the slope beyond the cut goes on down to the corner, where the edges' slopes start
                    if shade := dark * slopeProfile(math.Min(into, 1) * slopeCrease); shade > 0 {
                        out.SetNRGBA(x, y, slopeShade(true, shade))
                    }
                case into > -0.08:
                    out.SetNRGBA(x, y, slopeShade(false, slopeLipLight))
            }
        }
    }
    return out
}

// the half of a cell beyond the line between the two corners beside a corner, cut away: the slope
// hangs from the line, and what lies beyond the slope is drawn as low as the ground around
func slopeDiagonalPicture(corner int) image.Image {
    out := newSlopeCanvas()
    middleX := float64(slopeDiamondCenterX)
    middleY := float64(slopeDiamondHalfHeight)
    left := float64(slopeDiamondCenterX - slopeDiamondHalfWidth)
    right := float64(slopeDiamondCenterX + slopeDiamondHalfWidth)
    // the top of a plateau is drawn lighter; taking that back leaves the ground as low as around it
    unlift := 1 - 1 / plateauTopBrightness

    for y := range slopeCanvasHeight {
        for x := range slopeCanvasWidth {
            tileX, tileY := slopeTilePoint(x, y)
            _, _, inside := slopeCell(x, y)

            // how far beyond the line, in the way the slope goes, and how deep that slope is
            var beyond, depth, dark float64
            switch corner {
                case 0:
                    if tileX < left || tileX >= right {
                        continue
                    }
                    beyond, depth, dark = middleY - tileY, slopeFarDepth, slopeDarkFar
                case 2:
                    if tileX < left || tileX >= right {
                        continue
                    }
                    beyond, depth, dark = tileY - middleY, slopeFaceDepth, (slopeDarkSE + slopeDarkSW) / 2
                case 1:
                    if tileY < 0 || tileY >= slopeTileHeight {
                        continue
                    }
                    beyond, depth, dark = tileX - middleX, slopeFaceDepth, (slopeDarkSE + slopeDarkFar) / 2
                case 3:
                    if tileY < 0 || tileY >= slopeTileHeight {
                        continue
                    }
                    beyond, depth, dark = middleX - tileX, slopeFaceDepth, (slopeDarkSW + slopeDarkFar) / 2
            }

            switch {
                case beyond > 0:
                    // the slope, then low ground to the end of the cell
                    shade := dark * slopeProfile(beyond / depth)
                    if beyond >= depth * slopeCrease && inside {
                        shade = math.Max(shade, unlift)
                    }
                    if shade > 0 {
                        out.SetNRGBA(x, y, slopeShade(true, shade))
                    }
                case beyond > -1 && inside:
                    out.SetNRGBA(x, y, slopeShade(false, slopeLipLight))
            }
        }
    }
    return out
}

// the pieces of slope of a raised cell
type slopePieceSet struct {
    Edges [4]*ebiten.Image
    Inner [4]*ebiten.Image
    Diagonal [4]*ebiten.Image
    // the cuts of each corner
    Outer [4][]*ebiten.Image
}

// the pictures of the slopes: the ones of the replacement folder, or the first set
func (combat *CombatScreen) slopePictures() *slopePieceSet {
    if combat.slopeCache != nil {
        return combat.slopeCache
    }

    set := &slopePieceSet{}
    read := func(name string, made func() image.Image) *ebiten.Image {
        source := mod.ReadEnvironmentExtra(slopesSet, name)
        if source == nil {
            source = made()
        }
        return ebiten.NewImageFromImage(source)
    }

    for side, name := range slopeEdgeNames {
        set.Edges[side] = read(name, func() image.Image { return slopeEdgePicture(side) })
    }
    for corner, name := range slopeInnerNames {
        set.Inner[corner] = read(name, func() image.Image { return slopeInnerPicture(corner) })
    }
    for corner, name := range slopeDiagonalNames {
        set.Diagonal[corner] = read(name, func() image.Image { return slopeDiagonalPicture(corner) })
    }
    for corner := range slopeCornerNames {
        // the cuts of the replacement folder, from 1 on, as many as there are
        for number := 1; ; number++ {
            source := mod.ReadEnvironmentExtra(slopesSet, slopeOuterName(corner, number))
            if source == nil {
                break
            }
            set.Outer[corner] = append(set.Outer[corner], ebiten.NewImageFromImage(source))
        }
        if len(set.Outer[corner]) == 0 {
            for _, cut := range slopeCuts {
                set.Outer[corner] = append(set.Outer[corner], ebiten.NewImageFromImage(slopeOuterPicture(corner, cut)))
            }
        }
    }

    combat.slopeCache = set
    return set
}

// a number for a corner of a cell, the same every time
func slopeCornerHash(cgx int, cgy int, corner int) uint32 {
    value := uint32(cgx + 1000) * 73856093 ^ uint32(cgy + 1000) * 19349663 ^ uint32(corner + 1) * 83492791
    value ^= value >> 13
    value *= 2654435761
    value ^= value >> 16
    return value
}

// which parts of slope a raised cell has: the edges that drop, the corners where both edges beside
// them drop, and the inner corners. indexes into slopeSides and slopeCorners. where exactly two
// edges drop and they meet at a corner, that corner is cut in half instead: see slopeDiagonal
func (ground *BattleGround) slopeParts(cgx int, cgy int) ([]int, []int, []int) {
    height := ground.HeightAt(cgx, cgy)
    if height == 0 {
        return nil, nil, nil
    }

    lower := func(offset image.Point) bool {
        return ground.HeightAt(cgx + offset.X, cgy + offset.Y) < height
    }

    var edges, outer, inner []int
    for side, offset := range slopeSides {
        if lower(offset) {
            edges = append(edges, side)
        }
    }
    for corner, offset := range slopeCorners {
        before, after := slopeCornerEdges(corner)
        dropBefore := lower(slopeSides[before])
        dropAfter := lower(slopeSides[after])
        switch {
            case dropBefore && dropAfter: outer = append(outer, corner)
            case !dropBefore && !dropAfter && lower(offset): inner = append(inner, corner)
        }
    }
    return edges, outer, inner
}

// the corner a cell is cut in half at, or -1: exactly two of its edges drop, and they meet there
func slopeDiagonal(edges []int) int {
    if len(edges) != 2 {
        return -1
    }
    for corner := range slopeCornerNames {
        before, after := slopeCornerEdges(corner)
        if (edges[0] == before && edges[1] == after) || (edges[0] == after && edges[1] == before) {
            return corner
        }
    }
    return -1
}

// the pictures of slope on a raised cell
func (combat *CombatScreen) slopesOf(cgx int, cgy int) []*ebiten.Image {
    edges, outer, inner := combat.Model.Ground.slopeParts(cgx, cgy)
    if len(edges) == 0 && len(inner) == 0 {
        return nil
    }

    set := combat.slopePictures()
    var out []*ebiten.Image

    // cut in half: the diagonal takes the place of both edges and the cut of their corner
    if diagonal := slopeDiagonal(edges); diagonal >= 0 {
        out = append(out, set.Diagonal[diagonal])
        for _, corner := range inner {
            out = append(out, set.Inner[corner])
        }
        return out
    }

    for _, side := range edges {
        out = append(out, set.Edges[side])
    }
    for _, corner := range outer {
        cuts := set.Outer[corner]
        hash := slopeCornerHash(cgx, cgy, corner)
        // some corners stay square, the others take one of the cuts
        if len(cuts) > 0 && float64(hash % 1000) >= slopeSquareCorners * 1000 {
            out = append(out, cuts[(hash / 1000) % uint32(len(cuts))])
        }
    }
    for _, corner := range inner {
        out = append(out, set.Inner[corner])
    }
    return out
}

// all slopes of the field and its border, after all of the ground
func (combat *CombatScreen) drawSlopes(screen *ebiten.Image) {
    ground := combat.Model.Ground
    if ground == nil || ground.Heights == nil {
        return
    }

    matrix := combat.GetCameraMatrix()
    width := float64(screen.Bounds().Dx()) / scale.ScaleAmount
    height := float64(screen.Bounds().Dy()) / scale.ScaleAmount
    marginX := CellStepX * borderCullMargin * combat.CameraScale
    marginY := (CellStepY * borderCullMargin + slopeCanvasHeight) * combat.CameraScale

    var options ebiten.DrawImageOptions

    // row by row down the screen, so a face lies over what is farther away
    for row := -BattlefieldWidth - BattlefieldBorder * 2; row < BattlefieldHeight + BattlefieldBorder * 2; row++ {
        for x := -BattlefieldBorder; x < BattlefieldWidth + BattlefieldBorder; x++ {
            y := row + x
            if y < -BattlefieldBorder || y >= BattlefieldHeight + BattlefieldBorder {
                continue
            }

            tx, ty := matrix.Apply(float64(x), float64(y))
            if tx < -marginX || ty < -marginY || tx > width + marginX || ty > height + marginY {
                continue
            }

            cgx, cgy := TileToCell(x, y)
            pieces := combat.slopesOf(cgx, cgy)
            if len(pieces) == 0 {
                continue
            }

            brightness := borderBrightness(borderDepth(float64(x), float64(y)))
            if brightness <= 0 {
                continue
            }

            options.GeoM.Reset()
            options.GeoM.Translate(TerrainOffsetX - slopeCanvasLeft, TerrainOffsetY - slopeCanvasTop)
            options.GeoM.Scale(combat.CameraScale, combat.CameraScale)
            options.GeoM.Translate(tx, ty)
            options.ColorScale.Reset()
            options.ColorScale.Scale(brightness, brightness, brightness, brightness)
            for _, piece := range pieces {
                scale.DrawScaled(screen, piece, &options)
            }
        }
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
