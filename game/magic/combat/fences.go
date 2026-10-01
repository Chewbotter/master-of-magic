package combat

// The fences of farmland (user, 2026-09-29): "There are also 12 fence pieces, 6 for each
// direction, called prop_fence. Unlike the normal prop objects, these should be placed around crop
// fields in the correct orientation or along roads or houses."
//
// A fence stands ALONG A SIDE OF A CELL, one picture for a side. The sides of a cell run two ways
// on the screen: its NE and SW sides down to the right, its NW and SE sides up to the right. A
// picture is used for the sides that run as it does (mod/fences.go).
//
// WHERE, by chance:
//   along the plots of crops: a side of a plot as a whole, with a gap here and there, on no more
//   than fencePlotSides of the sides of a plot
//   along the roads: short rows beside them
// Never where the armies start, never across a road, and never twice on one side.
//
// NO BOXES (user, 2026-09-29: "prevent fences from having too many adjacent pieces (such as making
// a closed box or three edges of a closed box) since this looks unintentional"): no cell has
// fences on more than fenceCellSides of its four sides.
//
// NOT AT A HOUSE (user: "fences appear over the top of houses"): a fence on one of the two far
// sides of the cell of a house would be drawn over the house, which is drawn from the far corner
// of its cell, and one on a near side runs into its picture, which is wider than its cell. The
// four sides of the cell of a house have none. Houses have no fences of their own (first they had them on
// their two near sides; user: "no special rule about fences being near houses").
//
// Fences are for the look: they cost nothing and block nothing.

import (
    "fmt"
    rand "github.com/kazzmir/master-of-magic/lib/chance"

    "github.com/kazzmir/master-of-magic/game/magic/mod"

    "github.com/hajimehoshi/ebiten/v2"
)

// turns them off
var Fences = true

// the sides of a cell a fence is kept under: its NE side, toward the cell above it in the grid,
// and its NW side, toward the cell to its left. the SW side of a cell is the NE side of the cell
// below it, its SE side the NW side of the cell to its right
type fenceSide int
const (
    fenceNE fenceSide = iota
    fenceNW
)

type fencePlace struct {
    Cgx int
    Cgy int
    Side fenceSide
}

// a plot of crops: its first cell, and its cells across and down the grid
type farmPlot struct {
    X int
    Y int
    Width int
    Height int
}

// of the sides of the plots this share has a fence, and of the cells of such a side this share
// has none, as a gap
const fencePlotSideShare = 0.5
const fenceGapShare = 0.15
// rows of fences along the roads: one for this many cells of road, this many cells long
const fenceRoadCells = 9
const fenceRoadMin = 2
const fenceRoadMax = 4
// the most sides of a plot that have a fence, and the most sides of any cell
const fencePlotSides = 2
const fenceCellSides = 2

// the point of the picture of a fence that is put on the middle of its side: the middle of its
// width, this far above its bottom edge
const fenceAnchorBelow = 6

// a side of a cell of the rim of a plot: of its north east, south east, south west or north west
// side, the cell that far along it
func (plot farmPlot) side(side int, along int) fencePlace {
    switch side {
        case 0: return cellSide(plot.X + along, plot.Y, 0)
        case 1: return cellSide(plot.X + plot.Width - 1, plot.Y + along, 1)
        case 2: return cellSide(plot.X + along, plot.Y + plot.Height - 1, 2)
    }
    return cellSide(plot.X, plot.Y + along, 3)
}

// the fences of a ground, by the side they stand on
type fenceSet struct {
    Places map[fencePlace]bool
    ground *BattleGround
    large bool
}

// the sides of a cell as the sides that are kept: north east, south east, south west, north west
func cellSide(cgx int, cgy int, side int) fencePlace {
    switch side {
        case 0: return fencePlace{Cgx: cgx, Cgy: cgy, Side: fenceNE}
        case 1: return fencePlace{Cgx: cgx + 1, Cgy: cgy, Side: fenceNW}
        case 2: return fencePlace{Cgx: cgx, Cgy: cgy + 1, Side: fenceNE}
    }
    return fencePlace{Cgx: cgx, Cgy: cgy, Side: fenceNW}
}

// the two cells a side lies between
func (place fencePlace) cells() (int, int, int, int) {
    if place.Side == fenceNE {
        return place.Cgx, place.Cgy, place.Cgx, place.Cgy - 1
    }
    return place.Cgx, place.Cgy, place.Cgx - 1, place.Cgy
}

// the middle of a side on the original's screen
func (place fencePlace) screen() (int, int) {
    x, y := cellCenterScreen(place.Cgx, place.Cgy)
    if place.Side == fenceNE {
        return x + CellStepX / 2, y - CellStepY / 2
    }
    return x - CellStepX / 2, y - CellStepY / 2
}

// true if a fence can stand in a cell: on land of the ground, not where an army starts
func (fences *fenceSet) open(cgx int, cgy int) bool {
    ground := fences.ground
    if !ground.contains(cgx, cgy) || armyStartsOf(cgx, cgy, fences.large) {
        return false
    }
    group := ground.GroupAt(cgx, cgy)
    return group != TerrainSand && group != TerrainWater && ground.coastAt(cgx, cgy) == coastLand && ground.riverAt(cgx, cgy) == riverLand
}

// how many sides of a cell have a fence
func (fences *fenceSet) sidesOf(cgx int, cgy int) int {
    count := 0
    for side := range 4 {
        if fences.Places[cellSide(cgx, cgy, side)] {
            count += 1
        }
    }
    return count
}

func (fences *fenceSet) add(place fencePlace) bool {
    x1, y1, x2, y2 := place.cells()
    if !fences.open(x1, y1) || !fences.open(x2, y2) || fences.Places[place] {
        return false
    }
    // no boxes
    if fences.sidesOf(x1, y1) >= fenceCellSides || fences.sidesOf(x2, y2) >= fenceCellSides {
        return false
    }
    // not at a house: behind it a fence would be drawn over it, and its picture is wider than
    // its cell, so a fence on a near side would run into it
    if fences.ground.builtAt(x1, y1) || fences.ground.builtAt(x2, y2) {
        return false
    }
    // not across a road
    if fences.ground.RoadAt(x1, y1) != 0 && fences.ground.RoadAt(x2, y2) != 0 {
        return false
    }
    fences.Places[place] = true
    return true
}

// where the fences of a ground stand
func (ground *BattleGround) fencePlaces(zone ZoneType) []fencePlace {
    fences := fenceSet{Places: make(map[fencePlace]bool), ground: ground, large: zone.Ground.LargeArmy}

    // around the plots
    for _, plot := range ground.Plots {
        sides := 0
        for _, side := range rand.Perm(4) {
            if sides >= fencePlotSides || rand.Float64() >= fencePlotSideShare {
                continue
            }
            sides += 1
            // the cells of the side
            length := plot.Width
            if side == 1 || side == 3 {
                length = plot.Height
            }
            for along := range length {
                if rand.Float64() < fenceGapShare {
                    continue
                }
                fences.add(plot.side(side, along))
            }
        }
    }

    // along the roads: the sides of cells of road toward cells without one
    var beside []fencePlace
    roads := 0
    for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
        for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
            if ground.RoadAt(cgx, cgy) == 0 {
                continue
            }
            roads += 1
            for side := range 4 {
                place := cellSide(cgx, cgy, side)
                x1, y1, x2, y2 := place.cells()
                if ground.RoadAt(x1, y1) == 0 || ground.RoadAt(x2, y2) == 0 {
                    beside = append(beside, place)
                }
            }
        }
    }
    for range roads / fenceRoadCells {
        if len(beside) == 0 {
            break
        }
        place := beside[rand.N(len(beside))]
        // on along the side, as long as there is road beside it
        for range fenceRoadMin + rand.N(fenceRoadMax - fenceRoadMin + 1) {
            x1, y1, x2, y2 := place.cells()
            if ground.RoadAt(x1, y1) == 0 && ground.RoadAt(x2, y2) == 0 {
                break
            }
            fences.add(place)
            if place.Side == fenceNE {
                place.Cgx += 1
            } else {
                place.Cgy += 1
            }
        }
    }

    var out []fencePlace
    for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
        for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
            for _, side := range []fenceSide{fenceNE, fenceNW} {
                place := fencePlace{Cgx: cgx, Cgy: cgy, Side: side}
                if fences.Places[place] {
                    out = append(out, place)
                }
            }
        }
    }
    return out
}

// the fences of the farmland of a battlefield, of the pictures of its folder. after its houses
func makeFences(zone ZoneType, ground *BattleGround) []SceneryPiece {
    var out []SceneryPiece
    if !Fences || ground == nil || ground.Farmland == FarmlandNone {
        return out
    }

    up, down := mod.Fences(ground.FarmSet)
    for _, place := range ground.fencePlaces(zone) {
        // the NE side of a cell runs down to the right, its NW side up to the right
        pictures := down
        if place.Side == fenceNW {
            pictures = up
        }
        if len(pictures) == 0 {
            continue
        }

        x, y := place.screen()
        out = append(out, SceneryPiece{
            Kind: SceneryFence,
            ScreenX: x,
            ScreenY: y,
            Set: ground.FarmSet,
            Name: mod.FenceFile(0),
            Number: pictures[rand.N(len(pictures))],
        })
    }
    return out
}

// the picture of a fence
func (combat *CombatScreen) fencePictures(piece *SceneryPiece) []*ebiten.Image {
    key := fmt.Sprintf("%v/fence/%v", piece.Set, piece.Number)
    if pictures, ok := combat.addedCache[key]; ok {
        return pictures
    }

    var pictures []*ebiten.Image
    source := mod.ReadNamed(piece.Set, mod.FenceFile(piece.Number))
    if source != nil {
        pictures = append(pictures, ebiten.NewImageFromImage(source))
    }

    if combat.addedCache == nil {
        combat.addedCache = make(map[string][]*ebiten.Image)
    }
    combat.addedCache[key] = pictures
    return pictures
}
