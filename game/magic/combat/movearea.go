package combat

// Where the unit whose turn it is can go, shown on the ground. Not in the original (user,
// 2026-09-27).
//
// The tiles are the ones the game lets the unit walk to (CombatModel.CanMoveTo), so what is shown
// is what a click does. They are drawn from the shape of the original's outline of a cell, on a
// picture of art pixels that is lined up with the ground (as the shadows are, see shadows.go), so
// neighbors neither overlap nor leave gaps.
//
// Several looks to try, MoveAreaKey goes through them in a battle.
//
// A cell under the cursor the unit can not go to or attack gets a black outline, and the cursor
// stays the plain one, in place of the blue outline and the red X.

import (
    "image"
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/lib/fraction"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/inpututil"
)

type MoveAreaLook int
const (
    MoveAreaOff MoveAreaLook = iota
    // the ground of the area a little lighter
    MoveAreaFill
    // a line around the area
    MoveAreaEdge
    MoveAreaFillAndEdge
    // a dot in the middle of every tile
    MoveAreaDots
    moveAreaLooks
)

func (look MoveAreaLook) String() string {
    switch look {
        case MoveAreaFill: return "fill"
        case MoveAreaEdge: return "edge"
        case MoveAreaFillAndEdge: return "fill and edge"
        case MoveAreaDots: return "dots"
    }
    return "off"
}

// the look a battle starts with
var MoveArea = MoveAreaFillAndEdge
// goes through the looks, to compare them
var MoveAreaKey = ebiten.KeyF6

// the color of the area, parts of 1
const moveAreaRed = 1.0
const moveAreaGreen = 1.0
const moveAreaBlue = 1.0
// how much of it shows, 0 to 1
const moveAreaFillStrength = 0.1
const moveAreaEdgeStrength = 0.4
const moveAreaDotStrength = 0.4
// art pixels, the dot is this wide and high
const moveAreaDotWidth = 2
const moveAreaDotHeight = 2

// the outline of a cell the unit can not reach: black, this much of it shows
const OutOfReachOutline = true
const outOfReachStrength = 0.5

// as far as a unit that teleports goes, see CombatModel.CanMoveTo
const moveAreaTeleportReach = 10

// the tiles a unit can go to, kept until the unit or the battle around it has changed
type moveAreaTiles struct {
    Unit *ArmyUnit
    X, Y int
    Moves fraction.Fraction
    Units int
    Tiles map[image.Point]bool
}

// the shapes the area is made of, from the original's outline of a cell
type moveAreaShapes struct {
    Fill *ebiten.Image
    // the four sides of the outline, by the tile that lies beyond them
    Sides [4]*ebiten.Image
    Dot *ebiten.Image
}

// the tile beyond each side: upper left, upper right, lower left, lower right on the screen
var moveAreaNeighbors = [4]image.Point{
    image.Pt(0, -1),
    image.Pt(1, 0),
    image.Pt(-1, 0),
    image.Pt(0, 1),
}

func (combat *CombatScreen) updateMoveAreaKey() {
    if inpututil.IsKeyJustPressed(MoveAreaKey) {
        MoveArea = (MoveArea + 1) % moveAreaLooks
        combat.CastMessage("Move area: " + MoveArea.String())
    }
}

// the unit whose area shows, or nil
func (combat *CombatScreen) moveAreaUnit() *ArmyUnit {
    unit := combat.Model.SelectedUnit
    if MoveArea == MoveAreaOff || unit == nil || combat.together {
        return nil
    }
    if combat.Model.IsAIControlled(unit) || unit.Moving || unit.GetHealth() <= 0 {
        return nil
    }
    if combat.IsSelectingSpell() || combat.ExtraControl {
        return nil
    }
    if unit.MovesLeft.LessThanEqual(fraction.FromInt(0)) {
        return nil
    }
    return unit
}

func (combat *CombatScreen) moveAreaOf(unit *ArmyUnit) map[image.Point]bool {
    model := combat.Model
    units := len(model.AttackingArmy.units) + len(model.DefendingArmy.units) + len(model.MagicVortexes)

    kept := combat.moveArea
    if kept != nil && kept.Unit == unit && kept.X == unit.X && kept.Y == unit.Y && kept.Moves.Equals(unit.MovesLeft) && kept.Units == units {
        return kept.Tiles
    }

    // a step takes 1 at least and can be made as long as anything is left
    reach := int(math.Ceil(unit.MovesLeft.ToFloat()))
    if unit.CanTeleport() {
        reach = moveAreaTeleportReach
    }

    tiles := make(map[image.Point]bool)
    for y := unit.Y - reach; y <= unit.Y + reach; y++ {
        for x := unit.X - reach; x <= unit.X + reach; x++ {
            if !model.IsInsideMap(x, y) || (x == unit.X && y == unit.Y) {
                continue
            }
            if model.GetUnit(x, y) != nil || model.ContainsMagicVortex(x, y) {
                continue
            }
            if model.CanMoveTo(unit, x, y, false) {
                tiles[image.Pt(x, y)] = true
            }
        }
    }

    combat.moveArea = &moveAreaTiles{Unit: unit, X: unit.X, Y: unit.Y, Moves: unit.MovesLeft, Units: units, Tiles: tiles}
    return tiles
}

func (combat *CombatScreen) moveAreaShapes() *moveAreaShapes {
    if combat.moveShapes != nil {
        return combat.moveShapes
    }

    images, err := combat.ImageCache.GetImages(cellOutlineLbx, cellOutlineScannedIndex)
    if err != nil || len(images) == 0 {
        return nil
    }

    source := images[0]
    width := source.Bounds().Dx()
    height := source.Bounds().Dy()
    pixels := make([]byte, 4 * width * height)
    source.ReadPixels(pixels)

    solid := func(x int, y int) bool {
        return pixels[(y * width + x) * 4 + 3] != 0
    }

    white := []byte{255, 255, 255, 255}
    fill := make([]byte, len(pixels))
    var sides [4][]byte
    for index := range sides {
        sides[index] = make([]byte, len(pixels))
    }

    for y := range height {
        first := -1
        last := -1
        for x := range width {
            if !solid(x, y) {
                continue
            }
            if first == -1 {
                first = x
            }
            last = x

            side := 0
            if x * 2 >= width {
                side += 1
            }
            if y * 2 >= height {
                side += 2
            }
            copy(sides[side][(y * width + x) * 4:], white)
        }

        for x := first; first != -1 && x <= last; x++ {
            copy(fill[(y * width + x) * 4:], white)
        }
    }

    shapes := &moveAreaShapes{}
    shapes.Fill = ebiten.NewImage(width, height)
    shapes.Fill.WritePixels(fill)
    for index := range sides {
        shapes.Sides[index] = ebiten.NewImage(width, height)
        shapes.Sides[index].WritePixels(sides[index])
    }

    dot := make([]byte, len(pixels))
    for y := (height - moveAreaDotHeight) / 2; y < (height + moveAreaDotHeight) / 2; y++ {
        for x := (width - moveAreaDotWidth) / 2; x < (width + moveAreaDotWidth) / 2; x++ {
            copy(dot[(y * width + x) * 4:], white)
        }
    }
    shapes.Dot = ebiten.NewImage(width, height)
    shapes.Dot.WritePixels(dot)

    combat.moveShapes = shapes
    return shapes
}

// draws the area on the ground. part of the field
func (combat *CombatScreen) drawMoveArea(screen *ebiten.Image) {
    unit := combat.moveAreaUnit()
    if unit == nil {
        return
    }

    shapes := combat.moveAreaShapes()
    if shapes == nil {
        return
    }

    tiles := combat.moveAreaOf(unit)
    if len(tiles) == 0 {
        return
    }

    // the picture of art pixels, lined up with the ground as the one of the shadows
    pixel := math.Max(1, math.Round(combat.CameraScale * scale.ScaleAmount))
    matrix := combat.GetCameraMatrix()
    fieldX, fieldY := matrix.Apply(0, 0)
    startX := math.Mod(math.Round(fieldX * scale.ScaleAmount), pixel)
    startY := math.Mod(math.Round(fieldY * scale.ScaleAmount), pixel)
    if startX > 0 {
        startX -= pixel
    }
    if startY > 0 {
        startY -= pixel
    }

    bounds := screen.Bounds()
    width := int(math.Ceil(float64(bounds.Dx()) / pixel)) + 2
    height := int(math.Ceil(float64(bounds.Dy()) / pixel)) + 2
    if combat.moveLayer == nil || combat.moveLayer.Bounds().Dx() != width || combat.moveLayer.Bounds().Dy() != height {
        combat.moveLayer = ebiten.NewImage(width, height)
    }
    layer := combat.moveLayer

    place := func(x int, y int) (float64, float64) {
        tx, ty := matrix.Apply(float64(x), float64(y))
        atX := (tx * scale.ScaleAmount + TerrainOffsetX * pixel - startX) / pixel
        atY := (ty * scale.ScaleAmount + TerrainOffsetY * pixel - startY) / pixel
        return math.Round(atX), math.Round(atY)
    }

    // the unit stands in its area
    inArea := func(x int, y int) bool {
        return tiles[image.Pt(x, y)] || (x == unit.X && y == unit.Y)
    }

    // the shapes are solid, so a pixel that two tiles share is not twice as strong
    pass := func(strength float64, shapesOf func(x int, y int) []*ebiten.Image) {
        layer.Clear()
        var options ebiten.DrawImageOptions

        draw := func(x int, y int) {
            atX, atY := place(x, y)
            for _, shape := range shapesOf(x, y) {
                options.GeoM.Reset()
                options.GeoM.Translate(atX, atY)
                layer.DrawImage(shape, &options)
            }
        }

        for point := range tiles {
            draw(point.X, point.Y)
        }
        draw(unit.X, unit.Y)

        var onto ebiten.DrawImageOptions
        onto.GeoM.Scale(pixel, pixel)
        onto.GeoM.Translate(startX, startY)
        onto.ColorScale.Scale(moveAreaRed, moveAreaGreen, moveAreaBlue, 1)
        onto.ColorScale.ScaleAlpha(float32(strength))
        screen.DrawImage(layer, &onto)
    }

    if MoveArea == MoveAreaFill || MoveArea == MoveAreaFillAndEdge {
        pass(moveAreaFillStrength, func(x int, y int) []*ebiten.Image {
            return []*ebiten.Image{shapes.Fill}
        })
    }

    if MoveArea == MoveAreaEdge || MoveArea == MoveAreaFillAndEdge {
        pass(moveAreaEdgeStrength, func(x int, y int) []*ebiten.Image {
            var out []*ebiten.Image
            for side, neighbor := range moveAreaNeighbors {
                if !inArea(x + neighbor.X, y + neighbor.Y) {
                    out = append(out, shapes.Sides[side])
                }
            }
            return out
        })
    }

    if MoveArea == MoveAreaDots {
        pass(moveAreaDotStrength, func(x int, y int) []*ebiten.Image {
            if x == unit.X && y == unit.Y {
                return nil
            }
            return []*ebiten.Image{shapes.Dot}
        })
    }
}

// the outline of the cell under the cursor
func (combat *CombatScreen) drawCursorOutline(screen *ebiten.Image) {
    if !OutOfReachOutline || !combat.outOfReach {
        combat.drawCellOutline(screen, combat.MouseTileX, combat.MouseTileY, false)
        return
    }

    x, y := combat.MouseTileX, combat.MouseTileY
    if !combat.Model.IsInsideMap(x, y) {
        return
    }

    images, err := combat.ImageCache.GetImages(cellOutlineLbx, cellOutlineScannedIndex)
    if err != nil || len(images) == 0 {
        return
    }

    matrix := combat.GetCameraMatrix()
    tx, ty := matrix.Apply(float64(x), float64(y))

    var options ebiten.DrawImageOptions
    options.GeoM.Translate(TerrainOffsetX, TerrainOffsetY)
    options.GeoM.Scale(combat.CameraScale, combat.CameraScale)
    options.GeoM.Translate(tx, ty)
    options.ColorScale.Scale(0, 0, 0, 1)
    options.ColorScale.ScaleAlpha(outOfReachStrength)
    scale.DrawScaled(screen, images[0], &options)
}
