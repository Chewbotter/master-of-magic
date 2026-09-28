package combat

// The slopes of plateaus (plateau.go). Not in the original.
//
// Nothing is raised on the screen: the slopes are shading laid over the ground, one picture for the
// whole field and its border, made when the battle is first drawn.
//
// First the outline of the raised ground is made, in the cells of the grid (plateauOutline). It
// follows the raised cells, with their corners rounded where that is possible:
//   - a cell at a corner of a plateau, where two edges that meet drop, is cut along an arc that
//     runs into the edges of the cells beside it
//   - where such cells stand in a row, as a staircase of tiles on the screen, each is cut in half
//     from corner to corner, so the steps make one straight line
//   - a cell at the tip of a plateau and a hill of one cell have all their free corners rounded
//   - where the outline turns inward, the corner of the low cell is filled in along an arc
// The outline is only the look: a cell that is raised is raised as a whole for moving.
//
// Then every art pixel is shaded by where it lies to the outline (makeSlopeShading): the top a
// little lighter, the slope toward the viewer below the outline, the slope away from the viewer
// above it and shorter, as seen from above. A slope runs from light at the top over a soft crease
// into the ground (slopeProfile).
//
// User, 2026-09-28: after pieces per tile with hard corners and strong shading, "less intense
// shadowing and highlights, a smoother transition and rounded corners when possible".

import (
    "image"
    "image/color"
    "math"

    "github.com/hajimehoshi/ebiten/v2"
)

// the slope toward the viewer reaches this far below the outline, the one away from the viewer this
// far above it, in art pixels. the taller, the higher a plateau looks
const slopeFaceDepth = 7
const slopeFarDepth = 4
// how dark the slopes are at their darkest, 0 to 1: the ones toward the lower right, the lower
// left, and away from the viewer
const slopeDarkSE = 0.36
const slopeDarkSW = 0.26
const slopeDarkFar = 0.18
// the darkest part of a slope lies this far down it, 0 to 1. the slope gets darker down to there
// and fades into the ground from there on
const slopeCrease = 0.5
// how dark a slope is right at its top, 0 to 1 of its darkest
const slopeTopShade = 0.15
// how much lighter the top of a plateau is, and the line along its outline on top of that
const slopeTopLight = 0.05
const slopeRimLight = 0.06
// the direction of the outline is taken from this many pixels to each side
const slopeFacingReach = 4

// the corners of the outline, in cells. a corner of a plateau is cut along an arc of one of the
// first, a tip or a single hill along one of the second, an inward corner is filled along the third
var slopeCornerRadii = []float64{1, 0.75}
var slopeTipRadii = []float64{0.5, 0.4, 0.3}
const slopeFillRadius = 0.5
// a cell cut in half from corner to corner
const slopeCutDiagonal = -1

// the outline of the raised ground
type plateauOutline struct {
    ground *BattleGround
    // for every cell its corners N, E, S, W: for a raised cell what is cut away, for a low cell what
    // is filled in. 0 nothing, slopeCutDiagonal, or the radius of the arc
    corners [][4]float64
}

// the two edges beside a corner, as indexes of slopeSides: N lies between NW and NE
func slopeCornerEdges(corner int) (int, int) {
    return (corner + 3) % 4, corner
}

// a number for a corner of a cell, the same every time
func slopeCornerHash(cgx int, cgy int, corner int) uint32 {
    value := uint32(cgx + 1000) * 73856093 ^ uint32(cgy + 1000) * 19349663 ^ uint32(corner + 1) * 83492791
    value ^= value >> 13
    value *= 2654435761
    value ^= value >> 16
    return value
}

// the edges NE, SE, SW, NW of a raised cell beyond which the ground is lower
func (ground *BattleGround) dropEdges(cgx int, cgy int) ([4]bool, int) {
    var drops [4]bool
    count := 0
    height := ground.HeightAt(cgx, cgy)
    for side, offset := range slopeSides {
        if height > 0 && ground.HeightAt(cgx + offset.X, cgy + offset.Y) < height {
            drops[side] = true
            count += 1
        }
    }
    return drops, count
}

// true if exactly the two edges beside a corner of a cell drop
func (ground *BattleGround) isCornerCell(cgx int, cgy int, corner int) bool {
    drops, count := ground.dropEdges(cgx, cgy)
    before, after := slopeCornerEdges(corner)
    return count == 2 && drops[before] && drops[after]
}

// how far along its edges a cut reaches from its corner, in cells
func slopeCutExtent(cut float64) float64 {
    if cut == slopeCutDiagonal {
        return 1
    }
    return cut
}

func makePlateauOutline(ground *BattleGround) *plateauOutline {
    outline := &plateauOutline{
        ground: ground,
        corners: make([][4]float64, ground.Width * ground.Height),
    }

    lastX := ground.MinX + ground.Width
    lastY := ground.MinY + ground.Height

    // what is cut from the raised cells
    for cgy := ground.MinY; cgy < lastY; cgy++ {
        for cgx := ground.MinX; cgx < lastX; cgx++ {
            drops, count := ground.dropEdges(cgx, cgy)
            if count < 2 {
                continue
            }

            for corner := range slopeCorners {
                before, after := slopeCornerEdges(corner)
                if !drops[before] || !drops[after] {
                    continue
                }

                hash := slopeCornerHash(cgx, cgy, corner)
                cut := slopeTipRadii[hash % uint32(len(slopeTipRadii))]
                if count == 2 {
                    // a corner of a plateau. in a row of them, the cells to its sides are the same
                    cut = slopeCornerRadii[hash % uint32(len(slopeCornerRadii))]
                    for _, side := range []int{(corner + 1) % 4, (corner + 3) % 4} {
                        if ground.isCornerCell(cgx + slopeCorners[side].X, cgy + slopeCorners[side].Y, corner) {
                            cut = slopeCutDiagonal
                        }
                    }
                }
                outline.corners[ground.index(cgx, cgy)][corner] = cut
            }
        }
    }

    cutsOf := func(cgx int, cgy int) [4]float64 {
        if !ground.contains(cgx, cgy) {
            return [4]float64{}
        }
        return outline.corners[ground.index(cgx, cgy)]
    }

    // what is filled in where the outline turns inward: a low cell with raised cells beyond both
    // edges beside a corner. the arc runs into the edges of those two cells, so it can only be as
    // large as what their own cuts leave of the edges
    for cgy := ground.MinY; cgy < lastY; cgy++ {
        for cgx := ground.MinX; cgx < lastX; cgx++ {
            if ground.HeightAt(cgx, cgy) > 0 {
                continue
            }

            for corner := range slopeCorners {
                before, after := slopeCornerEdges(corner)
                beforeX, beforeY := cgx + slopeSides[before].X, cgy + slopeSides[before].Y
                afterX, afterY := cgx + slopeSides[after].X, cgy + slopeSides[after].Y
                if ground.HeightAt(beforeX, beforeY) == 0 || ground.HeightAt(afterX, afterY) == 0 {
                    continue
                }

                // the corner is the corner after this one of the cell beyond the edge before, and
                // the corner before this one of the cell beyond the edge after. their edges toward
                // this cell end at the corner across from this one
                cutsBefore := cutsOf(beforeX, beforeY)
                cutsAfter := cutsOf(afterX, afterY)
                if cutsBefore[(corner + 1) % 4] != 0 || cutsAfter[(corner + 3) % 4] != 0 {
                    continue
                }

                across := (corner + 2) % 4
                radius := math.Min(slopeFillRadius, 1 - slopeCutExtent(cutsBefore[across]))
                radius = math.Min(radius, 1 - slopeCutExtent(cutsAfter[across]))
                if radius > 0.05 {
                    outline.corners[ground.index(cgx, cgy)][corner] = radius
                }
            }
        }
    }

    return outline
}

// true if a point is on top of a plateau. in cells: the whole numbers are the cell, the rest runs
// from its NW edge (0) to its SE edge (1) and from its NE edge (0) to its SW edge (1)
func (outline *plateauOutline) topAt(cellX float64, cellY float64) bool {
    cgx := int(math.Floor(cellX))
    cgy := int(math.Floor(cellY))
    if !outline.ground.contains(cgx, cgy) {
        return false
    }

    raised := outline.ground.HeightAt(cgx, cgy) > 0
    u := cellX - float64(cgx)
    v := cellY - float64(cgy)
    // how far from the edges NE, SE, SW, NW
    distances := [4]float64{v, 1 - u, 1 - v, u}

    for corner, radius := range outline.corners[outline.ground.index(cgx, cgy)] {
        if radius == 0 {
            continue
        }
        before, after := slopeCornerEdges(corner)
        s, t := distances[before], distances[after]

        inCorner := false
        if radius == slopeCutDiagonal {
            inCorner = s + t < 1
        } else {
            inCorner = s < radius && t < radius && math.Hypot(radius - s, radius - t) > radius
        }
        if inCorner {
            // cut from a raised cell, filled into a low one
            return !raised
        }
    }

    return raised
}

// the middle of the diamond of a cell on the original's screen
func cellCenterScreen(cgx int, cgy int) (int, int) {
    return (cgx - cgy) * CellStepX + cellOriginX + cellMiddleX, (cgx + cgy) * CellStepY + cellOriginY + cellMiddleY
}

// a point of the original's screen in cells, as topAt takes it
func screenToCell(x float64, y float64) (float64, float64) {
    a := (x - cellOriginX - cellMiddleX) / CellStepX
    b := (y - cellOriginY - cellMiddleY) / CellStepY
    return (a + b + 1) / 2, (b - a + 1) / 2
}

func slopeSmooth(value float64) float64 {
    return value * value * (3 - 2 * value)
}

// how dark a slope is at a point, 0 at its top to 1 at its foot, 0 to 1 of its darkest
func slopeProfile(t float64) float64 {
    if t <= 0 || t >= 1 {
        return 0
    }
    if t < slopeCrease {
        return slopeTopShade + (1 - slopeTopShade) * slopeSmooth(t / slopeCrease)
    }
    return slopeSmooth((1 - t) / (1 - slopeCrease))
}

// the shading of the plateaus of a battlefield, and where its top left corner lies on the
// original's screen. nil without plateaus
func makeSlopeShading(ground *BattleGround) (*image.NRGBA, int, int) {
    if ground == nil || ground.Heights == nil {
        return nil, 0, 0
    }

    raised := false
    for _, height := range ground.Heights {
        raised = raised || height > 0
    }
    if !raised {
        return nil, 0, 0
    }

    outline := makePlateauOutline(ground)

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
    top -= CellStepY + slopeFarDepth
    bottom += CellStepY + slopeFaceDepth

    width := right - left
    height := bottom - top

    // what is on top, and how bright the ground is drawn there (fieldedge.go)
    isTop := make([]bool, width * height)
    bright := make([]float32, width * height)
    for y := range height {
        for x := range width {
            cellX, cellY := screenToCell(float64(left + x) + 0.5, float64(top + y) + 0.5)
            cgx, cgy := int(math.Floor(cellX)), int(math.Floor(cellY))
            if !ground.contains(cgx, cgy) {
                continue
            }
            tileX, tileY := CellToTile(cgx, cgy)
            bright[y * width + x] = borderBrightness(borderDepth(float64(tileX), float64(tileY)))
            isTop[y * width + x] = outline.topAt(cellX, cellY)
        }
    }

    topAt := func(x int, y int) bool {
        return x >= 0 && y >= 0 && x < width && y < height && isTop[y * width + x]
    }

    // which way the outline faces at a point of it: 1 to the lower right, -1 to the lower left
    facing := func(x int, y int) float64 {
        sum := 0
        for reach := 1; reach <= slopeFacingReach; reach++ {
            if topAt(x - reach, y) {
                sum += 1
            }
            if topAt(x + reach, y) {
                sum -= 1
            }
        }
        return float64(sum) / slopeFacingReach
    }

    out := image.NewNRGBA(image.Rect(0, 0, width, height))
    for y := range height {
        for x := range width {
            brightness := float64(bright[y * width + x])
            if brightness <= 0 {
                continue
            }

            if topAt(x, y) {
                light := slopeTopLight
                if !topAt(x, y + 1) || !topAt(x, y - 1) {
                    light += slopeRimLight
                }
                out.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: uint8(math.Round(light * brightness * 255))})
                continue
            }

            shade := 0.0
            // the slope toward the viewer hangs from the outline above
            for reach := 1; reach <= slopeFaceDepth; reach++ {
                if topAt(x, y - reach) {
                    turn := facing(x, y - reach)
                    dark := (slopeDarkSE + slopeDarkSW) / 2 + turn * (slopeDarkSE - slopeDarkSW) / 2
                    shade = dark * slopeProfile((float64(reach) - 0.5) / slopeFaceDepth)
                    break
                }
            }
            // the slope away from the viewer rises to the outline below
            for reach := 1; reach <= slopeFarDepth; reach++ {
                if topAt(x, y + reach) {
                    shade = math.Max(shade, slopeDarkFar * slopeProfile((float64(reach) - 0.5) / slopeFarDepth))
                    break
                }
            }

            if shade > 0 {
                out.SetNRGBA(x, y, color.NRGBA{A: uint8(math.Round(shade * 255))})
            }
        }
    }

    return out, left, top
}

// the shading of the plateaus over all of the ground
func (combat *CombatScreen) drawSlopes(screen *ebiten.Image) {
    if !combat.slopesMade {
        combat.slopesMade = true
        shading, x, y := makeSlopeShading(combat.Model.Ground)
        if shading != nil {
            combat.slopeShading = ebiten.NewImageFromImage(shading)
            combat.slopeX = x
            combat.slopeY = y
        }
    }

    if combat.slopeShading != nil {
        combat.drawOnField(screen, combat.slopeShading, combat.slopeX, combat.slopeY)
    }
}
