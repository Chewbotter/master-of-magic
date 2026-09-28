package combat

// Hills as plateaus (user request, 2026-09-28, not in the original). The original's rough ground
// wanders in lines one cell wide and its pictures show ridges, which on grass land often reads as
// snaking mounds. Here the same patches become raised ground of one height: small hills of one cell,
// and patches grown and smoothed into plateaus that cover larger areas. The top of a plateau is the
// usual ground, drawn a little lighter; slopes lie along its outline, which is rounded (slopes.go).
// Only going up or down a slope costs more (movecost.go), the top is ground as any other.
//
// The pictures of the original's rough ground still show, as mounds of one to three cells that
// never run on in lines (makeMounds). They are for the look only and cost what the ground costs.
//
// PlateauGround false brings back the original's rough ground.

import (
    "image"
    "math"
)

var PlateauGround = true

// every cell of a patch of rough raises the cells next to it too, then the edges are smoothed: a
// cell with this many raised of its 8 neighbors is raised, one with this few or less is not
const plateauSmoothPasses = 2
const plateauFillNeighbors = 5
const plateauDropNeighbors = 2
// small hills of a single cell, for every patch of rough ground this many
const plateauSingleHillsPerPatch = 0.5

// mounds of the original's rough pictures, for every patch of rough ground this many, each 1 to
// moundSpan steps long
const moundsPerPatch = 0.6
const moundSpan = 2

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

// short patches of the original's rough ground: a cell and at most moundSpan steps from it, so they
// never run on as lines. the original's rules for rough still apply to them (removeRough)
func (ground *BattleGround) makeMounds(patches int) {
    count := int(math.Round(float64(patches) * moundsPerPatch))
    for range count {
        cgx := ground.MinX + roll(ground.Width) - 1
        cgy := ground.MinY + roll(ground.Height) - 1
        ground.setGroup(cgx, cgy, TerrainRough)
        for range roll(moundSpan + 1) - 1 {
            step := roll(4)
            cgx += stepCellX[step]
            cgy += stepCellY[step]
            ground.setGroup(cgx, cgy, TerrainRough)
        }
    }
}

// the neighbors across the edges NE, SE, SW, NW of a cell, and beyond its corners N, E, S, W
var slopeSides = [4]image.Point{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}
var slopeCorners = [4]image.Point{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}}

// true if a step between two cells goes up or down a slope
func (ground *BattleGround) isSlope(fromX int, fromY int, toX int, toY int) bool {
    if ground == nil || ground.Heights == nil {
        return false
    }
    return ground.HeightAt(fromX, fromY) != ground.HeightAt(toX, toY)
}
