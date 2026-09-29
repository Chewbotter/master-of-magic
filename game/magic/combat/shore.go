package combat

// The sand of the transitions of a coast (coast.go): the cells a line of the coast runs through
// have the tile of what is underneath, the landscape or the water, and the sand is drawn over the
// part of them that lies on the beach. One picture for the whole field and its border, made when
// the battle is first drawn, as the shading of the plateaus is (slopes.go).
//
// The sand of a pixel is the pixel a tile of sand would have there, so the beach goes on from the
// tiles that are sand as a whole without a seam. There is no blending: a pixel is sand or it is
// not, which is what makes the edge look as the art does.

import (
    "image"
    "image/color"
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/mod"

    "github.com/hajimehoshi/ebiten/v2"
)

// the ground picture of a tile, and where the middle of its diamond lies in it
const shoreTileWidth = 30
const shoreTileHeight = 16
const shoreTileMiddleX = 15
const shoreTileMiddleY = 8

// the color of a picture at a place, and whether there is any. a place beyond the picture or on a
// see-through pixel of it takes the nearest pixel toward the middle of the tile that has one
func shoreColor(picture image.Image, x int, y int) (color.NRGBA, bool) {
    bounds := picture.Bounds()
    x = min(max(x, 0), min(bounds.Dx(), shoreTileWidth) - 1)
    y = min(max(y, 0), min(bounds.Dy(), shoreTileHeight) - 1)

    for range shoreTileMiddleX {
        red, green, blue, alpha := picture.At(bounds.Min.X + x, bounds.Min.Y + y).RGBA()
        if alpha >= 0x8000 {
            return color.NRGBA{R: uint8(red * 0xff / alpha), G: uint8(green * 0xff / alpha), B: uint8(blue * 0xff / alpha), A: 0xff}, true
        }
        switch {
            case x < shoreTileMiddleX - 1: x += 1
            case x > shoreTileMiddleX: x -= 1
        }
        switch {
            case y < shoreTileMiddleY - 1: y += 1
            case y > shoreTileMiddleY: y -= 1
        }
    }
    return color.NRGBA{}, false
}

// the sand over the transitions of the coast of a battlefield, and where its top left corner lies
// on the original's screen. sands are the pictures of the tiles of sand. nil without a coast
func makeShoreSand(ground *BattleGround, sands []image.Image) (*image.NRGBA, int, int) {
    if ground == nil || ground.Lines == nil || len(sands) == 0 {
        return nil, 0, 0
    }

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
    top -= CellStepY
    bottom += CellStepY

    width := right - left
    height := bottom - top
    out := image.NewNRGBA(image.Rect(0, 0, width, height))
    any := false

    for y := range height {
        for x := range width {
            cellX, cellY := screenToCell(float64(left + x) + 0.5, float64(top + y) + 0.5)
            cgx, cgy := int(math.Floor(cellX)), int(math.Floor(cellY))
            if !ground.shoreAt(cgx, cgy) || !ground.Lines.sandIn(ground.coastAt(cgx, cgy), cellX, cellY, left + x, top + y) {
                continue
            }

            tileX, tileY := CellToTile(cgx, cgy)
            brightness := float64(borderBrightness(borderDepth(float64(tileX), float64(tileY))))
            if brightness <= 0 {
                continue
            }

            // the pixel of a tile of sand at this place of the cell
            centerX, centerY := cellCenterScreen(cgx, cgy)
            sand := sands[slopeCornerHash(cgx, cgy, 0) % uint32(len(sands))]
            pixel, ok := shoreColor(sand, left + x - (centerX - shoreTileMiddleX), top + y - (centerY - shoreTileMiddleY))
            if !ok {
                continue
            }

            pixel.R = uint8(float64(pixel.R) * brightness)
            pixel.G = uint8(float64(pixel.G) * brightness)
            pixel.B = uint8(float64(pixel.B) * brightness)
            out.SetNRGBA(x, y, pixel)
            any = true
        }
    }

    if !any {
        return nil, 0, 0
    }
    return out, left, top
}

// the pictures of the tiles of sand: of the folder of the coast, or the game's
func (combat *CombatScreen) sandTiles() []image.Image {
    ground := combat.Model.Ground
    var out []image.Image

    archive, err := combat.Cache.GetLbxFile(ground.SandLbx)
    for number := 1; number <= groundSandCount + ground.Extras[groundSandFirst]; number++ {
        added := mod.ReadExtra(ground.SandSet, coastSandName, number)
        if len(added) > 0 {
            out = append(out, added[0])
            continue
        }
        if number > groundSandCount || err != nil {
            continue
        }

        index := ground.SandFirst + number - 1
        frames, err := archive.ReadImages(index)
        if err != nil || len(frames) == 0 {
            continue
        }
        // the sand of the replacement folder, if it has any
        frames = mod.Replace(ground.SandLbx, index, frames)
        out = append(out, frames[0])
    }

    return out
}

// the sand of the transitions of the coast, over the ground of their tiles. roads that run there
// are drawn again, over the sand
func (combat *CombatScreen) drawShore(screen *ebiten.Image) {
    ground := combat.Model.Ground
    if ground == nil || ground.Lines == nil {
        return
    }

    if !combat.shoreMade {
        combat.shoreMade = true
        sand, x, y := makeShoreSand(ground, combat.sandTiles())
        if sand != nil {
            combat.shoreSand = ebiten.NewImageFromImage(sand)
            combat.shoreX = x
            combat.shoreY = y
        }
    }
    if combat.shoreSand == nil {
        return
    }

    combat.drawOnField(screen, combat.shoreSand, combat.shoreX, combat.shoreY)

    matrix := combat.GetCameraMatrix()
    var options ebiten.DrawImageOptions
    for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
        for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
            if !ground.shoreAt(cgx, cgy) || ground.RoadAt(cgx, cgy) == 0 {
                continue
            }
            x, y := CellToTile(cgx, cgy)
            brightness := borderBrightness(borderDepth(float64(x), float64(y)))

            tx, ty := matrix.Apply(float64(x), float64(y))
            options.GeoM.Reset()
            options.GeoM.Translate(TerrainOffsetX, TerrainOffsetY)
            options.GeoM.Scale(combat.CameraScale, combat.CameraScale)
            options.GeoM.Translate(tx, ty)
            options.ColorScale.Reset()
            options.ColorScale.Scale(brightness, brightness, brightness, 1)
            combat.drawRoad(screen, x, y, &options)
        }
    }
}
