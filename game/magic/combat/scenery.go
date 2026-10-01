package combat

// What stands on the battlefield, matched to the original game: trees, rocks, the houses of a city,
// its walls, and the order everything on the field is drawn in.
//
// The original keeps every thing that stands on the field as a position on its screen plus the point
// of the picture that is put there. We keep the same: positions here are in art pixels of the
// original's 320 by 200 screen, and the camera moves them to where that screen is shown.
// (ReMoM: MoM/src/Combat.c Generate_Combat_Map, Scatter_Tree_Scenery, Scatter_Rock_Scenery,
// Spawn_Tree_Entities, Spawn_Rock_Entities, Spawn_Structure_Entities, Spawn_Stone_Wall_Entities,
// Spawn_Dark_Wall_Entities, Spawn_Fire_Wall_Entities, Combat_Grid_Entity_Create,
// Set_Entity_Draw_Order, Set_Movement_Cost_Maps. MoM/src/LOADER.c for the pictures. The code here is ours.)

import (
    rand "github.com/kazzmir/master-of-magic/lib/chance"

    "github.com/kazzmir/master-of-magic/game/magic/data"

    "github.com/hajimehoshi/ebiten/v2"
)

type SceneryKind int
const (
    SceneryTree SceneryKind = iota
    SceneryRock
    SceneryHouse
    // the fortress of a city, or an outpost
    SceneryStructure
)

type SceneryPiece struct {
    Kind SceneryKind
    // where it stands on the original's screen
    ScreenX int
    ScreenY int
    Lbx string
    Index int
}

// the pictures of a use
type sceneryPool struct {
    Lbx string
    First int
    Count int
}

func makeSceneryPool(lbx string, first int, count int) sceneryPool {
    return sceneryPool{Lbx: lbx, First: first, Count: count}
}

// a piece with one of the pictures, by chance
func (pool sceneryPool) piece(kind SceneryKind, screenX int, screenY int) SceneryPiece {
    return SceneryPiece{Kind: kind, ScreenX: screenX, ScreenY: screenY, Lbx: pool.Lbx, Index: pool.First + rand.N(pool.Count)}
}

// the original's view of the battlefield: scenery only stands where its screen shows the field
const sceneryScreenWidth = 320
const sceneryScreenHeight = 180

// the pictures of a terrain set: five trees, then five rocks
const sceneryTreeIndex = 48
const sceneryRockIndex = 53
const sceneryPictures = 5

// the point of the picture that is put on the position: the middle of its width, this far above
// its bottom edge. for the pictures of the game that is 8, 13 of a tree and 6, 12 of a rock, as the
// original has them
const treeAnchorBelow = 5
const rockAnchorBelow = 1
// houses: the middle of the picture, this far above its bottom
const houseAnchorBelow = 14
// houses stand this far below the top corner of their cell
const houseShiftY = 2
// trees and rocks are moved left by this from where their place in the cell is
const sceneryShiftX = 5

// trees come in patches. a tree stands up to this many cells from the middle of its patch
const treePatchReach = 3
// places within a cell, in steps of 2 across and 1 down along each side of the diamond
const treeSubcellMax = 6
const rockSubcellMin = 4
const rockSubcellMax = 5
const treePatchesExtra = 4
const treesPerPatch = 10
// the original gives up on a battlefield after this many places were turned down
const sceneryMaxTries = 500

const houseLbx = "cmbtcity.lbx"
const houseIndex = 2
const housePictures = 5
const fortressIndex = 17
const outpostIndex = 18
// a city of this many citizens fills every cell of the town with a house
const housesFullTown = 12

// the cells of a town, in the original's grid
const townCellMinX = 5
const townCellMaxX = 8
const townCellMinY = 10
const townCellMaxY = 13

// our grid is larger than the original's and the camera can show more than the original's screen.
// beyond that screen scenery continues as dense as on it, over the field and the ground around it
// (fieldedge.go). this is how many times the screen fits in that area, less the screen itself
func sceneryBeyondScreen(width int, height int) int {
    tiles := (width + BattlefieldBorder * 2) * (height + BattlefieldBorder * 2)
    // a tile is a diamond, half of its 32 by 16 box
    area := tiles * CellStepX * CellStepY * 2
    screen := sceneryScreenWidth * sceneryScreenHeight
    return max(0, area / screen - 1)
}

// 1 to n, the original's random numbers
func roll(n int) int {
    if n <= 0 {
        return 0
    }
    return rand.N(n) + 1
}

// where a place within a cell is on the original's screen
func cellScreen(cgx int, cgy int, subX int, subY int) (int, int) {
    x := (cgx - cgy) * CellStepX + cellOriginX + (subX - subY) * 2
    y := (cgx + cgy) * CellStepY + cellOriginY + subX + subY
    return x, y
}

func floorDivide(value int, by int) (int, int) {
    quotient := value / by
    remainder := value % by
    if remainder < 0 {
        quotient -= 1
        remainder += by
    }
    return quotient, remainder
}

// the larger, the later a thing at this position of the original's screen is drawn: by the row of
// cells it stands in, then by the cell, then by where in the cell
func DrawOrder(screenX int, screenY int) int {
    half := (screenX - cellOriginX) / 2
    cellX, insideX := floorDivide(half + screenY - cellOriginY, CellStepX)
    cellY, insideY := floorDivide(screenY - cellOriginY - half, CellStepX)

    return ((cellY * 1000 + cellX) * CellStepX + insideY) * CellStepX + insideX
}

func terrainSetLbx(landscape CombatLandscape, plane data.Plane) string {
    arcanus := plane == data.PlaneArcanus
    switch landscape {
        case CombatLandscapeDesert:
            if arcanus {
                return "cmbdesrt.lbx"
            }
            return "cmbdesrc.lbx"
        case CombatLandscapeMountain:
            if arcanus {
                return "cmbmount.lbx"
            }
            return "cmbmounc.lbx"
        case CombatLandscapeTundra:
            if arcanus {
                return "cmbtundr.lbx"
            }
            return "cmbtundc.lbx"
    }

    if arcanus {
        return "cmbgrass.lbx"
    }
    return "cmbgrasc.lbx"
}

func treeCount(landscape CombatLandscape, ground ZoneGround) int {
    if ground.Forest {
        return 30 + roll(30)
    }
    switch landscape {
        case CombatLandscapeDesert: return roll(10)
        case CombatLandscapeMountain: return roll(40)
        case CombatLandscapeTundra: return roll(20)
    }
    return roll(20)
}

// on the original's screen
func rockCount(landscape CombatLandscape, ground ZoneGround) int {
    if ground.Forest {
        return roll(8)
    }
    switch landscape {
        case CombatLandscapeDesert: return roll(8) - 1
        case CombatLandscapeMountain: return roll(12)
        case CombatLandscapeTundra: return roll(12) + 4
    }
    return roll(5) - 1
}

// the cells trees and rocks keep out of
func sceneryCellTaken(zone ZoneType, cgx int, cgy int, rock bool) bool {
    city := zone.City != nil
    structure := city || (zone.Encounter != ZoneNone && zone.Encounter != ZoneNatureNode)
    openField := !city && (zone.Encounter == ZoneNone || zone.Encounter == ZoneTower)

    if !openField && cgx >= townCellMinX && cgx < townCellMaxX && cgy > townCellMinY && cgy <= townCellMaxY {
        return true
    }

    if structure && cgx >= townCellMinX && cgx < townCellMaxX && cgy >= townCellMinY && cgy < townCellMaxY {
        return true
    }

    // the near sides of the town. rocks keep out of them on every battlefield
    if city || rock {
        if city && cgx == townCellMaxX && cgy == townCellMinY {
            return true
        }
        if cgx == townCellMaxX && cgy > townCellMinY && cgy <= townCellMaxY {
            return true
        }
        if cgy == townCellMaxY && cgx >= townCellMinX && cgx <= townCellMaxX {
            return true
        }
    }

    return false
}

type sceneryArea struct {
    // the cells places are picked from
    MinX, MaxX int
    MinY, MaxY int
    Accept func(screenX int, screenY int) bool
}

func onOriginalScreen(screenX int, screenY int) bool {
    return screenX > 0 && screenX < sceneryScreenWidth && screenY > 0 && screenY < sceneryScreenHeight
}

// the area the original places scenery in: its grid, where its screen shows it
func originalArea() sceneryArea {
    return sceneryArea{
        MinX: 0, MaxX: OriginalGridWidth - 1,
        MinY: 0, MaxY: OriginalGridHeight - 1,
        Accept: onOriginalScreen,
    }
}

// the rest of our field
func beyondArea(width int, height int) sceneryArea {
    cornerX1, cornerY1 := TileToCell(-BattlefieldBorder, -BattlefieldBorder)
    cornerX2, cornerY2 := TileToCell(width - 1 + BattlefieldBorder, height - 1 + BattlefieldBorder)

    return sceneryArea{
        MinX: min(cornerX1, cornerX2), MaxX: max(cornerX1, cornerX2),
        MinY: min(cornerY1, cornerY2), MaxY: max(cornerY1, cornerY2),
        Accept: func(screenX int, screenY int) bool {
            return !onOriginalScreen(screenX, screenY)
        },
    }
}

func (area sceneryArea) randomCell() (int, int) {
    return area.MinX + rand.N(area.MaxX - area.MinX + 1), area.MinY + rand.N(area.MaxY - area.MinY + 1)
}

func (area sceneryArea) contains(cgx int, cgy int) bool {
    return cgx >= area.MinX && cgx <= area.MaxX && cgy >= area.MinY && cgy <= area.MaxY
}

func scatterTrees(count int, pool sceneryPool, zone ZoneType, area sceneryArea, ground *BattleGround) []SceneryPiece {
    var out []SceneryPiece

    if count <= 0 {
        return out
    }

    patches := count / treesPerPatch + roll(treePatchesExtra)
    if count < patches {
        return out
    }
    patchSize := max(1, count / patches)

    tries := 0
    for range patches {
        patchX, patchY := area.randomCell()
        for picks := 0; !ground.sceneryAllowed(patchX, patchY); picks++ {
            if picks >= sceneryMaxTries {
                return out
            }
            patchX, patchY = area.randomCell()
        }

        placed := 0
        for placed < patchSize {
            cgx := patchX + roll(treePatchReach * 2 + 1) - treePatchReach - 1
            cgy := patchY + roll(treePatchReach * 2 + 1) - treePatchReach - 1
            screenX, screenY := cellScreen(cgx, cgy, roll(treeSubcellMax), roll(treeSubcellMax))

            if area.contains(cgx, cgy) && !sceneryCellTaken(zone, cgx, cgy, false) && ground.sceneryAllowed(cgx, cgy) && area.Accept(screenX, screenY) {
                out = append(out, pool.piece(SceneryTree, screenX - sceneryShiftX, screenY))
                // a tree makes its cell harder to go through, see movecost.go
                ground.addTree(cgx, cgy)
                placed += 1
            } else {
                tries += 1
                if tries >= sceneryMaxTries {
                    // the original places nothing more on this battlefield
                    return out
                }
            }
        }
    }

    return out
}

func scatterRocks(count int, pool sceneryPool, zone ZoneType, area sceneryArea, ground *BattleGround) []SceneryPiece {
    var out []SceneryPiece

    tries := 0
    for len(out) < count && tries < sceneryMaxTries {
        cgx, cgy := area.randomCell()
        subX := rockSubcellMin + rand.N(rockSubcellMax - rockSubcellMin + 1)
        subY := rockSubcellMin + rand.N(rockSubcellMax - rockSubcellMin + 1)
        screenX, screenY := cellScreen(cgx, cgy, subX, subY)

        if sceneryCellTaken(zone, cgx, cgy, true) || !ground.sceneryAllowed(cgx, cgy) || !area.Accept(screenX, screenY) {
            tries += 1
            continue
        }

        out = append(out, pool.piece(SceneryRock, screenX - sceneryShiftX, screenY))
    }

    return out
}

// the pictures of houses come in three styles of five
func houseStyle(race data.Race) int {
    switch race.HouseType() {
        case data.HouseTypeHut: return 1
        case data.HouseTypeTree: return 2
    }
    return 0
}

func makeHouses(zone ZoneType) []SceneryPiece {
    var out []SceneryPiece

    city := zone.City
    if city == nil {
        return out
    }

    lairX, lairY := cellScreen(lairCellX, lairCellY, 0, 0)

    if city.Outpost {
        return append(out, SceneryPiece{Kind: SceneryStructure, ScreenX: lairX, ScreenY: lairY, Lbx: houseLbx, Index: outpostIndex})
    }

    fortress := city.HasFortress()
    walled := city.HasWall()
    style := houseStyle(city.Race)
    housePool := makeSceneryPool(houseLbx, houseIndex + style * housePictures, housePictures)

    if fortress {
        out = append(out, SceneryPiece{Kind: SceneryStructure, ScreenX: lairX, ScreenY: lairY, Lbx: houseLbx, Index: fortressIndex})
    }

    townSize := townCellMaxX - townCellMinX + 1

    free := func(column int, row int) bool {
        corner := (column == 0 || column == townSize - 1) && (row == 0 || row == townSize - 1)
        if walled && corner {
            return false
        }
        if fortress && townCellMinX + column == lairCellX && townCellMinY + row == lairCellY {
            return false
        }
        return true
    }

    add := func(column int, row int) {
        screenX, screenY := cellScreen(townCellMinX + column, townCellMinY + row, 0, 0)
        out = append(out, housePool.piece(SceneryHouse, screenX, screenY + houseShiftY))
    }

    houses := city.Citizens()
    full := housesFullTown
    if fortress {
        full -= 1
    }

    if houses >= full {
        // row by row until the town is full
        count := 0
        for row := range townSize {
            for column := range townSize {
                if count < full && free(column, row) {
                    add(column, row)
                    count += 1
                }
            }
        }
        return out
    }

    taken := make(map[int]bool)
    for count := 0; count < houses; {
        column := rand.N(townSize)
        row := rand.N(townSize)
        if !free(column, row) || taken[row * townSize + column] {
            continue
        }
        taken[row * townSize + column] = true
        add(column, row)
        count += 1
    }

    return out
}

// everything that stands on a battlefield from its start to its end
func makeScenery(width int, height int, landscape CombatLandscape, plane data.Plane, zone ZoneType, ground *BattleGround) []SceneryPiece {
    out := makeHouses(zone)

    // water has neither trees nor rocks
    if landscape == CombatLandscapeWater {
        return out
    }

    lbx := terrainSetLbx(landscape, plane)
    treePool := makeSceneryPool(lbx, sceneryTreeIndex, sceneryPictures)
    rockPool := makeSceneryPool(lbx, sceneryRockIndex, sceneryPictures)

    trees := treeCount(landscape, zone.Ground)
    out = append(out, scatterTrees(trees, treePool, zone, originalArea(), ground)...)
    out = append(out, scatterTrees(trees * sceneryBeyondScreen(width, height), treePool, zone, beyondArea(width, height), ground)...)

    rocks := rockCount(landscape, zone.Ground)
    out = append(out, scatterRocks(rocks, rockPool, zone, originalArea(), ground)...)
    out = append(out, scatterRocks(rocks * sceneryBeyondScreen(width, height), rockPool, zone, beyondArea(width, height), ground)...)

    return out
}

// the point of its picture that is put on the position of a piece
func (piece *SceneryPiece) anchor(width int, height int) (int, int) {
    switch piece.Kind {
        case SceneryTree: return width / 2, height - treeAnchorBelow
        case SceneryRock: return width / 2, height - rockAnchorBelow
        case SceneryHouse: return width / 2, height - houseAnchorBelow
        case SceneryStructure: return structureAnchorX, height - structureAnchorBelow
    }
    return 0, 0
}

// the walls of a town are 12 pieces, one per cell of its edge. walls of fire and of darkness have
// two more, the near sides of the two corners at the left and right
type wallPiece struct {
    // the number of the picture within the pictures of its wall
    Piece int
    CellX int
    CellY int
    // from the top corner of the cell to the position of the piece
    ShiftX int
    ShiftY int
    // the point of the picture that is put there. below 0: half the width of the picture
    AnchorX int
    // for walls of stone, and for walls of fire and of darkness
    AnchorY int
    MagicAnchorY int
    // only walls of fire and of darkness have it
    MagicOnly bool
    // on the side of the town toward the bottom of the screen: drawn over what stands in the cell
    Near bool
}

const wallHalfWidth = -1
// the corner piece while a wall rises
const wallRiseCornerAnchorX = 15

var wallPieces = []wallPiece{
    // the far corner
    {Piece: 0, CellX: 5, CellY: 10, ShiftX: 1, ShiftY: 1, AnchorX: wallHalfWidth, AnchorY: 18, MagicAnchorY: 19},
    // the far left side
    {Piece: 1, CellX: 5, CellY: 11, ShiftX: 1, ShiftY: 1, AnchorX: 17, AnchorY: 17, MagicAnchorY: 18},
    {Piece: 2, CellX: 5, CellY: 12, ShiftX: 1, ShiftY: 1, AnchorX: 17, AnchorY: 17, MagicAnchorY: 18},
    {Piece: 3, CellX: 5, CellY: 13, ShiftX: 1, ShiftY: 1, AnchorX: 17, AnchorY: 17, MagicAnchorY: 18},
    // the far right side
    {Piece: 4, CellX: 6, CellY: 10, ShiftX: 1, ShiftY: 0, AnchorX: 17, AnchorY: 17, MagicAnchorY: 17},
    {Piece: 5, CellX: 7, CellY: 10, ShiftX: 1, ShiftY: 0, AnchorX: 17, AnchorY: 17, MagicAnchorY: 17},
    {Piece: 6, CellX: 8, CellY: 10, ShiftX: 1, ShiftY: 0, AnchorX: 17, AnchorY: 17, MagicAnchorY: 17},
    // the near left side
    {Piece: 7, CellX: 6, CellY: 13, ShiftX: 0, ShiftY: 14, AnchorX: 16, AnchorY: 30, MagicAnchorY: 29, Near: true},
    {Piece: 8, CellX: 7, CellY: 13, ShiftX: 0, ShiftY: 14, AnchorX: 16, AnchorY: 30, MagicAnchorY: 29, Near: true},
    {Piece: 9, CellX: 8, CellY: 13, ShiftX: 0, ShiftY: 14, AnchorX: 16, AnchorY: 30, MagicAnchorY: 29, Near: true},
    // the near right side, the second is the gate
    {Piece: 10, CellX: 8, CellY: 11, ShiftX: 0, ShiftY: 14, AnchorX: 16, AnchorY: 30, MagicAnchorY: 29, Near: true},
    {Piece: 11, CellX: 8, CellY: 12, ShiftX: 0, ShiftY: 14, AnchorX: 16, AnchorY: 30, MagicAnchorY: 29, Near: true},
    // the near sides of the left and right corners
    {Piece: 12, CellX: 5, CellY: 13, ShiftX: 0, ShiftY: 14, AnchorX: 16, MagicAnchorY: 30, MagicOnly: true, Near: true},
    {Piece: 13, CellX: 8, CellY: 10, ShiftX: 0, ShiftY: 14, AnchorX: 16, MagicAnchorY: 30, MagicOnly: true, Near: true},
}

// the ground of a town is one picture: its roads, or the clouds of a flying fortress. the original
// draws it this far up and left of the ground picture of the nearest cell of the town
const townGroundLbx = "cmbtcity.lbx"
const townRoadsIndex = 0
const townCloudsIndex = 113
const townGroundShift = 48
// the ground under an outpost
const outpostGroundIndex = 1

const wallLbx = "citywall.lbx"
const wallRiseLbx = "wallrise.lbx"
const wallStonePictures = 12
const wallFireIndex = 36
const wallDarknessIndex = 50
// the frames of a wall of stone: standing, broken
const wallFrameStanding = 0
const wallFrameBroken = 1
// walls of fire and darkness step through this many frames, one step every second redraw of the original
const wallMagicFrames = 4
const wallMagicTicksPerFrame = 2
// our ticks a frame of a rising wall shows
const wallRiseTicksPerFrame = 8

// things at one position are drawn in this order, first to last: the original draws the one it made
// last first, and makes trees, rocks, figures, houses, then the walls. of the walls it draws the far
// pieces fire, darkness, stone and the near pieces stone, darkness, fire
const (
    layerWallFarFire = iota
    layerWallFarDarkness
    layerWallFarStone
    layerWallNearStone
    layerWallNearDarkness
    layerWallNearFire
    layerStructure
    layerFigure
    layerRock
    layerTree
)

// the structure in the middle of a battlefield that is drawn in order with the units. the sorcery
// and chaos nodes are part of the ground
const sorceryNodeIndex = 66

func (top TileTop) standsOnField() bool {
    if top.Drawer != nil || top.Index == -1 || top.Index == sorceryNodeIndex {
        return false
    }
    return top.Alignment == TileAlignStructure || top.Alignment == TileAlignNatureNode
}

// the pictures of walls of stone differ by plane
func wallStoneIndex(plane data.Plane) int {
    if plane == data.PlaneMyrror {
        return wallStonePictures
    }
    return 0
}

// the blocked cell of a battlefield in a city: the fortress
func fortressTile(zone ZoneType) (int, int, bool) {
    if zone.City == nil || zone.City.Outpost || !zone.City.HasFortress() {
        return 0, 0, false
    }
    x, y := LairTile()
    return x, y, true
}

// maps the original's screen to where the camera shows it
func originalScreenMatrix(camera ebiten.GeoM) ebiten.GeoM {
    matrix := MakeBattlefieldMatrix()
    matrix.Invert()
    matrix.Concat(camera)
    return matrix
}
