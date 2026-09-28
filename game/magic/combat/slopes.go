package combat

// The slopes of plateaus (plateau.go). Not in the original.
//
// Nothing is raised on the screen: slopes are pictures laid over the ground, like the marks of
// spells. Along an edge of a raised cell where the next cell is lower lies a piece of slope. On the
// edges toward the viewer (SE and SW) its face hangs down below the edge, over the lower ground, so
// the plateau looks taller than a tile would let it (slopeFaceDepth); the far edges (NE and NW) show
// a light rim. Where two edges of a cell drop, its corner is cut in one of several shapes, or left
// square, so the corners of plateaus do not all look alike. Where the cell beyond a corner is lower
// but the two edges beside it are not, an inner corner fills the notch.
//
// The game makes a first set of the pictures (SlopePicture). Pictures of the replacement folder go
// before them:
//
//   environment/Slopes/edge NE_0.png ... edge NW_0.png
//   environment/Slopes/inner N_0.png ... inner W_0.png
//   environment/Slopes/outer N 1_0.png, outer N 2_0.png, ...   from 1 on, as many as there are
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

// the ground picture of a tile and the points of its diamond, in its own pixels
const slopeTileWidth = 30
const slopeTileHeight = 16

// how far the face of a slope toward the viewer hangs below the edge, in art pixels. the taller,
// the higher a plateau looks
const slopeFaceDepth = 12
// how dark the faces are at their darkest, 0 to 1. SE is turned away from the light more than SW
const slopeDarkSE = 0.62
const slopeDarkSW = 0.45
// the last part of a face, 0 to 1 of its depth, fades into the ground
const slopeFaceFoot = 0.2
// a thin light line on the edge above a face, in cells from the edge, and how light
const slopeLipNear = 0.07
const slopeLipLightNear = 0.3
// the light rim of the far edges
const slopeLipFar = 0.3
const slopeLightFar = 0.22
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
    for corner := range slopeCornerNames {
        for number, cut := range slopeCuts {
            out[slopeOuterName(corner, number + 1)] = slopeOuterPicture(corner, cut)
        }
    }
    return out
}

// a point of a picture of slope in the cells of the grid: u runs from the NW edge (0) to the SE edge
// (1), v from the NE edge (0) to the SW edge (1). inside is false beyond the tile
func slopeCell(x int, y int) (float64, float64, bool) {
    tileX := float64(x - slopeCanvasLeft) + 0.5
    tileY := float64(y - slopeCanvasTop) + 0.5
    a := (tileX - slopeTileWidth / 2) / (slopeTileWidth / 2)
    b := (tileY - slopeTileHeight / 2) / (slopeTileHeight / 2)
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

// the piece along an edge: a face hanging below it toward the viewer, a light rim away from it
func slopeEdgePicture(side int) image.Image {
    out := newSlopeCanvas()

    for y := range slopeCanvasHeight {
        for x := range slopeCanvasWidth {
            u, v, inside := slopeCell(x, y)
            distance := slopeEdgeDistances(u, v)[side]

            if inside {
                switch side {
                    case 0, 3:
                        if distance < slopeLipFar {
                            out.SetNRGBA(x, y, slopeShade(false, slopeLightFar * (1 - distance / slopeLipFar)))
                        }
                    case 1, 2:
                        if distance < slopeLipNear {
                            out.SetNRGBA(x, y, slopeShade(false, slopeLipLightNear))
                        }
                }
                continue
            }

            if side != 1 && side != 2 {
                continue
            }

            // the face: straight below the edge, down to slopeFaceDepth
            tileX := float64(x - slopeCanvasLeft) + 0.5
            tileY := float64(y - slopeCanvasTop) + 0.5
            var edgeY float64
            switch side {
                case 1:
                    if tileX < slopeTileWidth / 2 || tileX > slopeTileWidth {
                        continue
                    }
                    edgeY = slopeTileHeight / 2 + (slopeTileWidth - tileX) * slopeTileHeight / slopeTileWidth
                case 2:
                    if tileX < 0 || tileX > slopeTileWidth / 2 {
                        continue
                    }
                    edgeY = slopeTileHeight / 2 + tileX * slopeTileHeight / slopeTileWidth
            }

            depth := (tileY - edgeY) / slopeFaceDepth
            if depth <= 0 || depth > 1 {
                continue
            }

            dark := slopeDarkSE
            if side == 2 {
                dark = slopeDarkSW
            }
            // a little darker toward the foot, and fading into the ground at the end
            amount := dark * (0.8 + 0.2 * depth)
            if depth > 1 - slopeFaceFoot {
                amount *= (1 - depth) / slopeFaceFoot
            }
            out.SetNRGBA(x, y, slopeShade(true, amount))
        }
    }

    return out
}

// how a corner looks where it drops: N away from the viewer, S toward it
func slopeCornerShade(corner int) (bool, float64) {
    switch corner {
        // the top corner: cut away it is lower ground, not lifted
        case 0: return true, 1 - 1 / plateauTopBrightness
        case 1: return true, slopeDarkSE * 0.8
        case 2: return true, (slopeDarkSE + slopeDarkSW) / 2
    }
    return true, slopeDarkSW * 0.8
}

// the two edges beside a corner, as indexes of slopeEdgeDistances: N lies between NW and NE
func slopeCornerEdges(corner int) (int, int) {
    return (corner + 3) % 4, corner
}

// the notch of an inner corner
func slopeInnerPicture(corner int) image.Image {
    out := newSlopeCanvas()
    before, after := slopeCornerEdges(corner)
    dark, amount := slopeCornerShade(corner)
    if corner == 0 {
        dark, amount = false, slopeLightFar
    }

    for y := range slopeCanvasHeight {
        for x := range slopeCanvasWidth {
            u, v, inside := slopeCell(x, y)
            if !inside {
                continue
            }
            distances := slopeEdgeDistances(u, v)
            reach := math.Max(distances[before], distances[after])
            if reach < slopeInnerReach {
                out.SetNRGBA(x, y, slopeShade(dark, amount * (1 - reach / slopeInnerReach)))
            }
        }
    }
    return out
}

// a corner cut away where two edges drop
func slopeOuterPicture(corner int, cut slopeCut) image.Image {
    out := newSlopeCanvas()
    before, after := slopeCornerEdges(corner)
    dark, amount := slopeCornerShade(corner)

    for y := range slopeCanvasHeight {
        for x := range slopeCanvasWidth {
            u, v, inside := slopeCell(x, y)
            if !inside {
                continue
            }
            distances := slopeEdgeDistances(u, v)
            s, t := distances[before], distances[after]

            cutAway := false
            switch {
                case cut.Bevel > 0:
                    cutAway = s + t < cut.Bevel
                case cut.Round > 0:
                    r := cut.Round
                    cutAway = s < r && t < r && (r - s) * (r - s) + (r - t) * (r - t) > r * r
                case cut.Bite > 0:
                    cutAway = s < cut.Bite && t < cut.Bite
            }
            if cutAway {
                out.SetNRGBA(x, y, slopeShade(dark, amount))
            }
        }
    }
    return out
}

// the pieces of slope of a raised cell
type slopePieceSet struct {
    Edges [4]*ebiten.Image
    Inner [4]*ebiten.Image
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
// them drop, and the inner corners. indexes into slopeSides and slopeCorners
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

// the pictures of slope on a raised cell
func (combat *CombatScreen) slopesOf(cgx int, cgy int) []*ebiten.Image {
    edges, outer, inner := combat.Model.Ground.slopeParts(cgx, cgy)
    if len(edges) == 0 && len(inner) == 0 {
        return nil
    }

    set := combat.slopePictures()
    var out []*ebiten.Image
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
