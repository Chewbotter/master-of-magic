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
//   around the plots of crops: a side of a plot as a whole, with a gap here and there
//   along the roads: short rows beside them
//   at the houses: on the two near sides of a house
// Never where the armies start, never across a road, and never twice on one side.
//
// Fences are for the look: they cost nothing and block nothing.

import (
    "fmt"
    "math/rand/v2"

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

// a plot of crops: its first cell, and the cells of a side of it
type farmPlot struct {
    X int
    Y int
    Size int
}

// of the sides of the plots this share has a fence, and of the cells of such a side this share
// has none, as a gap
const fencePlotSideShare = 0.5
const fenceGapShare = 0.15
// rows of fences along the roads: one for this many cells of road, this many cells long
const fenceRoadCells = 9
const fenceRoadMin = 2
const fenceRoadMax = 4
// of the houses this share has fences
const fenceHouseShare = 0.6

// the point of the picture of a fence that is put on the middle of its side: the middle of its
// width, this far above its bottom edge
const fenceAnchorBelow = 6

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

func (fences *fenceSet) add(place fencePlace) bool {
    x1, y1, x2, y2 := place.cells()
    if !fences.open(x1, y1) || !fences.open(x2, y2) || fences.Places[place] {
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
        for side := range 4 {
            if rand.Float64() >= fencePlotSideShare {
                continue
            }
            for along := range plot.Size {
                if rand.Float64() < fenceGapShare {
                    continue
                }
                switch side {
                    case 0: fences.add(cellSide(plot.X + along, plot.Y, 0))
                    case 1: fences.add(cellSide(plot.X + plot.Size - 1, plot.Y + along, 1))
                    case 2: fences.add(cellSide(plot.X + along, plot.Y + plot.Size - 1, 2))
                    case 3: fences.add(cellSide(plot.X, plot.Y + along, 3))
                }
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

    // at the houses: the two near sides
    for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
        for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
            if ground.builtAt(cgx, cgy) && rand.Float64() < fenceHouseShare {
                fences.add(cellSide(cgx, cgy, 1))
                fences.add(cellSide(cgx, cgy, 2))
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
