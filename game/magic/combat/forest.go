package combat

// The trees of a forest (user, 2026-09-29): "greatly increase the number of trees. If possible,
// they should cluster more, leaving clearings (especially near the center of the battlefield),
// instead of being distributed uniformly."
//
// The original scatters 31 to 60 trees over its screen in patches of 5 (scatterTrees), which is a
// meadow with trees. Here a forest is WOODS AND CLEARINGS: a number that changes slowly from place
// to place says how dense the wood is at a cell. Below forestClearing there is a clearing, above it
// trees, the more the higher it is, so a wood is thin at its rim and dense inside. Around the
// middle of the field, where the armies start and meet, lies a clearing of its own: no trees in its
// middle, more and more toward its rim. The cells the armies start in have no trees at all.
//
// The rules are the ones of all trees: every tree in a cell makes the step into it cost half a
// move more, up to twice the cost of open ground (movecost.go). So the woods are slow and the
// clearings are where the armies move fast.

import (
    "math"
    rand "github.com/kazzmir/master-of-magic/lib/chance"
)

// false scatters the trees of a forest as the original does
var ForestWoods = true

// woods and clearings are about this many cells across
const forestPatch = 5.0
// of the number of a place, -1 to 1: below this there is a clearing. lower means more wood
const forestClearing = -0.12
// above the clearing the number has to rise by this much for a tree more in a cell
const forestTreeStep = 0.16
// the most trees of a cell
const forestTreesMost = 3
// a cell of the wood has no trees one time in this many, so it is not even
const forestGapChance = 5

// the clearing of the armies: its middle, and how far it reaches along the two sides of the grid.
// within forestOpen of that reach there are no trees, from there to its rim more and more
const forestMiddleX = 11.0
const forestMiddleY = 11.5
const forestReachX = 6.5
const forestReachY = 5.0
const forestOpen = 0.5

// a number for a place that changes slowly from place to place, -1 to 1, mostly near 0. size is
// how many cells lie between two places whose numbers have nothing to do with each other
func patchNoise(cgx int, cgy int, size float64, seed uint32) float64 {
    x := float64(cgx) / size
    y := float64(cgy) / size
    beforeX := int(math.Floor(x))
    beforeY := int(math.Floor(y))
    partX := slopeSmooth(x - float64(beforeX))
    partY := slopeSmooth(y - float64(beforeY))

    top := coastNoise(beforeX, beforeY, seed) * (1 - partX) + coastNoise(beforeX + 1, beforeY, seed) * partX
    bottom := coastNoise(beforeX, beforeY + 1, seed) * (1 - partX) + coastNoise(beforeX + 1, beforeY + 1, seed) * partX
    return top * (1 - partY) + bottom * partY
}

// how much of the wood of a place the clearing of the armies leaves: 0 in it, 1 beyond its rim
func forestBeyondArmies(cgx int, cgy int) float64 {
    x := (float64(cgx) + 0.5 - forestMiddleX) / forestReachX
    y := (float64(cgy) + 0.5 - forestMiddleY) / forestReachY
    away := math.Hypot(x, y)
    return min(max((away - forestOpen) / (1 - forestOpen), 0), 1)
}

// how many trees a cell of a forest has
func forestTrees(cgx int, cgy int, seed uint32) int {
    dense := (patchNoise(cgx, cgy, forestPatch, seed) - forestClearing) * forestBeyondArmies(cgx, cgy)
    if dense <= 0 {
        return 0
    }
    return min(1 + int(dense / forestTreeStep), forestTreesMost)
}

// the trees of a forest, over the field and the ground around it
func makeWoods(pool sceneryPool, zone ZoneType, ground *BattleGround) []SceneryPiece {
    var out []SceneryPiece
    if ground == nil {
        return out
    }

    seed := rand.Uint32()
    for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
        for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
            trees := forestTrees(cgx, cgy, seed)
            if trees == 0 || rand.N(forestGapChance) == 0 {
                continue
            }
            // no trees where the armies start, of whatever size they are
            if armyStarts(cgx, cgy) || sceneryCellTaken(zone, cgx, cgy, false) || !ground.sceneryAllowed(cgx, cgy) {
                continue
            }

            for range trees {
                screenX, screenY := cellScreen(cgx, cgy, roll(treeSubcellMax), roll(treeSubcellMax))
                out = append(out, pool.piece(SceneryTree, screenX - sceneryShiftX, screenY))
                // a tree makes its cell harder to go through, see movecost.go
                ground.addTree(cgx, cgy)
            }
        }
    }

    return out
}
