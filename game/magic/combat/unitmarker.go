package combat

// The mark over the unit whose turn it is, shown while Tab is held. Not in the original.
//
// The fork showed a column of light over the tile, drawn with lines of the screen. It is a double
// chevron of art pixels now that points down at the unit and goes up and down a little (user,
// 2026-09-28).

import (
    "math"

    "github.com/hajimehoshi/ebiten/v2"
)

// development: as if Tab was held
var HoldTab bool

// false: the column of light of the fork
const UnitChevron = true

// art pixels. a chevron is this wide, its arms this thick, the second one this far under the first
const chevronWidth = 11
const chevronThick = 2
const chevronStep = 4
// its tip is this far over the middle of the tile of the unit, over the heads of its figures
const chevronAbove = 24
// it goes this far up and down, and once up and down takes this many seconds
const chevronBounce = 1.5
const chevronBounceTime = 0.9

// of 255: the chevron and its edge
var chevronFill = [4]byte{255, 255, 255, 255}
var chevronEdge = [4]byte{0, 0, 0, 255}

// the pixels of the chevron, 4 numbers a pixel, with its size
func makeChevronPixels() ([]byte, int, int) {
    half := chevronWidth / 2
    // room for the edge around it
    width := chevronWidth + 2
    height := half + chevronThick + chevronStep + 2

    filled := make([]bool, width * height)
    inside := func(x int, y int) bool {
        return x >= 0 && y >= 0 && x < width && y < height && filled[y * width + x]
    }

    for _, top := range []int{0, chevronStep} {
        for x := range chevronWidth {
            // the arms go down to the tip in the middle
            distance := x - half
            if distance < 0 {
                distance = -distance
            }
            for thick := range chevronThick {
                y := top + half - distance + thick
                filled[(y + 1) * width + x + 1] = true
            }
        }
    }

    pixels := make([]byte, 4 * width * height)
    for y := range height {
        for x := range width {
            at := (y * width + x) * 4
            if inside(x, y) {
                copy(pixels[at:], chevronFill[:])
                continue
            }

            // the edge: next to the chevron, corners too
            edge := false
            for dy := -1; dy <= 1 && !edge; dy++ {
                for dx := -1; dx <= 1; dx++ {
                    if inside(x + dx, y + dy) {
                        edge = true
                        break
                    }
                }
            }
            if edge {
                copy(pixels[at:], chevronEdge[:])
            }
        }
    }

    return pixels, width, height
}

func (combat *CombatScreen) chevronPicture() *ebiten.Image {
    if combat.chevron == nil {
        pixels, width, height := makeChevronPixels()
        combat.chevron = ebiten.NewImage(width, height)
        combat.chevron.WritePixels(pixels)
    }
    return combat.chevron
}

// how far up the chevron is now, in whole art pixels
func chevronLift(counter uint64, ticksPerSecond int) int {
    seconds := float64(counter) / float64(max(1, ticksPerSecond))
    wave := (math.Sin(seconds / chevronBounceTime * 2 * math.Pi) + 1) / 2
    return int(math.Round(wave * chevronBounce))
}

// draws the mark over a unit. part of the field
func (combat *CombatScreen) drawUnitChevron(screen *ebiten.Image, unit *ArmyUnit) {
    if unit == nil || unit.Moving {
        return
    }

    picture := combat.chevronPicture()
    placeX, placeY := spellPlace(unit.X, unit.Y)

    x := placeX - picture.Bounds().Dx() / 2
    // the tip is the lowest row but the edge
    y := placeY - chevronAbove - (picture.Bounds().Dy() - 1) - chevronLift(combat.Counter, ebiten.TPS())

    combat.drawOnField(screen, picture, x, y)
}
