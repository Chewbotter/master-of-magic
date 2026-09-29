package combat

// Large pieces of ground: pictures that cover 2 by 2 tiles, for what does not fit on one. A larger
// cluster of rocks, a bare patch of dirt, a hill with a larger footprint (user, 2026-09-28). Not in
// the original.
//
// A battlefield has a few of them, far fewer than tiles (largeShare), and EVERY PICTURE ONCE AT
// MOST (user, 2026-09-28: "a hard limit of one per map ... one of each individual large image, not
// 1 large image total"): with three pictures there are up to three large pieces, each a different
// one. They lie on the field itself, not on the ground around it, where they would hardly be seen.
// They lie on open grass only: all four cells plain grass of one height, without a road, outside
// of the town and the lair, and no two on the same cell. Trees and rocks keep off them. They are
// for the look only and cost what the ground costs.
//
// A large piece is drawn OVER the ground of its four tiles, which is drawn as always. So its
// picture can cover them whole, or be see-through around what it shows.
//
// The pictures: the game makes the first two itself, of the pictures of the landscape: a cluster of
// its rocks (MakeRockCluster) and a bare patch of its dirt (MakeDirtPatch). The replacement folder
// can take their places and add more (mod/environment.go):
//
//   environment/Grass/large 1_0.png, large 2_0.png   in place of the game's
//   environment/Grass/large 3_0.png, ...             more, without a gap
//
// A picture is largeWidth by largeHeight, the four tiles lying in it as they do on the screen: the
// top one 16 from the left, the left and right ones 8 down, the bottom one 16 down. A picture of
// another size stands with the middle of its bottom edge on the bottom corner of the four tiles.
// More frames (_1, _2, ...) make it move, as the ground does.

import (
    "image"
    "image/color"
    "math"
    "math/rand/v2"

    "github.com/kazzmir/master-of-magic/game/magic/mod"
    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
)

// turns them off
var LargeGround = true

// how much of the field large pieces cover at most, 0 to 1. no more pieces than there are pictures
const largeShare = 0.04
// places that are tried for every piece before it is given up
const largeTries = 30

// the name of the pictures, and how many the game makes itself
const largeName = "large"
const largeGameCount = 2

// the rocks of the cluster the game makes: where each stands in the four tiles, by the point the
// game puts on the place of a rock, and the room above the four tiles for the rocks at the top
var rockClusterPlaces = []image.Point{{31, 9}, {21, 13}, {41, 14}, {12, 18}, {30, 18}, {50, 19}, {23, 23}, {39, 24}, {31, 28}}
const rockClusterAbove = 6

// the picture of a large piece, and where the pictures of its four tiles lie in it: top, left,
// right, bottom
const largeWidth = 62
const largeHeight = 32
const largeTileWidth = 30
const largeTileHeight = 16
var largeTilePlaces = [4]image.Point{{16, 0}, {0, 8}, {32, 8}, {16, 16}}

// the patch of dirt the game makes: its reach from the middle, 1 being the edge of the four tiles,
// how much its outline waves, and how wide its edge is scattered into the grass
const dirtPatchReach = 0.85
const dirtPatchWave = 0.1
const dirtPatchEdge = 0.3

type LargePiece struct {
    // the top one of its four cells: the others are one further along each axis, and both
    Cgx int
    Cgy int
    // which of the pictures, from 1
    Number int
}

// the four cells of a large piece
func largeCells(cgx int, cgy int) [4]image.Point {
    return [4]image.Point{{cgx, cgy}, {cgx, cgy + 1}, {cgx + 1, cgy}, {cgx + 1, cgy + 1}}
}

// true if a cell can lie under a large piece
func (ground *BattleGround) largeAllowed(zone ZoneType, cgx int, cgy int) bool {
    if !ground.contains(cgx, cgy) || ground.Covered[ground.index(cgx, cgy)] {
        return false
    }
    if ground.GroupAt(cgx, cgy) != TerrainGrass || ground.RoadAt(cgx, cgy) != 0 || ground.HeightAt(cgx, cgy) != 0 || ground.shoreAt(cgx, cgy) || ground.riverAt(cgx, cgy) != riverLand {
        return false
    }
    if plateauKeepsOut(zone, cgx, cgy) {
        return false
    }

    // plain grass: no dirt next to it, which would show its edge on the tile
    for dy := -1; dy <= 1; dy++ {
        for dx := -1; dx <= 1; dx++ {
            if ground.GroupAt(cgx + dx, cgy + dy) == TerrainDirt {
                return false
            }
        }
    }
    return true
}

// puts the large pieces on a field of width by height tiles. pictures is how many there are: every
// one of them is used once at most
func (ground *BattleGround) placeLarge(width int, height int, zone ZoneType, pictures int) {
    ground.Covered = make([]bool, ground.Width * ground.Height)
    if !LargeGround || pictures <= 0 {
        return
    }

    onField := func(cgx int, cgy int) bool {
        x, y := CellToTile(cgx, cgy)
        return x >= 0 && y >= 0 && x < width && y < height
    }

    count := int(math.Round(float64(width * height) * largeShare / 4))
    // the pictures in an order by chance, each once
    for _, picture := range rand.Perm(pictures) {
        if len(ground.Large) >= count {
            break
        }

        for range largeTries {
            cgx, cgy := TileToCell(rand.N(width), rand.N(height))

            free := true
            for _, cell := range largeCells(cgx, cgy) {
                free = free && onField(cell.X, cell.Y) && ground.largeAllowed(zone, cell.X, cell.Y)
            }
            if !free {
                continue
            }

            for _, cell := range largeCells(cgx, cgy) {
                ground.Covered[ground.index(cell.X, cell.Y)] = true
            }
            ground.Large = append(ground.Large, LargePiece{Cgx: cgx, Cgy: cgy, Number: picture + 1})
            break
        }
    }
}

func (ground *BattleGround) coveredAt(cgx int, cgy int) bool {
    return ground.Covered != nil && ground.contains(cgx, cgy) && ground.Covered[ground.index(cgx, cgy)]
}

// a number for a pixel, 0 to 1, the same every time
func largeNoise(x int, y int) float64 {
    value := uint32(x + 100) * 73856093 ^ uint32(y + 100) * 19349663
    value ^= value >> 13
    value *= 2654435761
    value ^= value >> 16
    return float64(value % 1000) / 1000
}

// The large piece the game makes: a bare patch of dirt. tiles gives the pictures of the ground of
// a landscape by their number in its archive; the patch is cut from its four pictures of dirt, laid
// as the four tiles lie, with an outline that waves and an edge scattered into the grass. Around
// the patch the picture is see-through. nil if the pictures are not there
func MakeDirtPatch(tiles func(index int) *image.Paletted) *image.Paletted {
    var out *image.Paletted

    for place, corner := range largeTilePlaces {
        tile := tiles(groundDirtFirst + place)
        if tile == nil {
            return nil
        }
        if out == nil {
            palette := make(color.Palette, len(tile.Palette))
            copy(palette, tile.Palette)
            out = image.NewPaletted(image.Rect(0, 0, largeWidth, largeHeight), palette)
        }

        bounds := tile.Bounds()
        for y := 0; y < bounds.Dy() && y < largeTileHeight; y++ {
            for x := 0; x < bounds.Dx() && x < largeTileWidth; x++ {
                value := tile.ColorIndexAt(bounds.Min.X + x, bounds.Min.Y + y)
                if value == 0 {
                    continue
                }

                // where the pixel lies in the four tiles, along the two axes of the grid: 0 in the
                // middle, 1 at the edges
                pixelX := corner.X + x
                pixelY := corner.Y + y
                a := (float64(pixelX) + 0.5 - largeWidth / 2) / (largeWidth / 2)
                b := (float64(pixelY) + 0.5 - largeHeight / 2) / (largeHeight / 2)
                u, v := a + b, b - a

                reach := dirtPatchReach + dirtPatchWave * math.Sin(3 * math.Atan2(v, u) + 1)
                inside := (reach - math.Hypot(u, v)) / dirtPatchEdge + 0.5
                if largeNoise(pixelX, pixelY) < inside {
                    out.SetColorIndex(pixelX, pixelY, value)
                }
            }
        }
    }

    return out
}

// The first large piece the game makes: a cluster of rocks. pictures gives the pictures of a
// landscape by their number in its archive; the cluster is its five rocks, set out over the four
// tiles. Around the rocks the picture is see-through. nil if the pictures are not there
func MakeRockCluster(pictures func(index int) *image.Paletted) *image.Paletted {
    var out *image.Paletted

    for place, point := range rockClusterPlaces {
        rock := pictures(sceneryRockIndex + place * 2 % sceneryPictures)
        if rock == nil {
            return nil
        }
        if out == nil {
            palette := make(color.Palette, len(rock.Palette))
            copy(palette, rock.Palette)
            out = image.NewPaletted(image.Rect(0, 0, largeWidth, largeHeight + rockClusterAbove), palette)
        }

        bounds := rock.Bounds()
        left := point.X - bounds.Dx() / 2
        top := point.Y + rockClusterAbove - (bounds.Dy() - rockAnchorBelow)
        for y := range bounds.Dy() {
            for x := range bounds.Dx() {
                value := rock.ColorIndexAt(bounds.Min.X + x, bounds.Min.Y + y)
                if value != 0 && image.Pt(left + x, top + y).In(out.Bounds()) {
                    out.SetColorIndex(left + x, top + y, value)
                }
            }
        }
    }

    return out
}

// the large pieces the game makes, in the order of their numbers
func MakeLargePieces(pictures func(index int) *image.Paletted) []*image.Paletted {
    return []*image.Paletted{MakeRockCluster(pictures), MakeDirtPatch(pictures)}
}

// the frames of a large piece: of the replacement folder, or one the game makes
func (combat *CombatScreen) largePictures(number int) []*ebiten.Image {
    ground := combat.Model.Ground
    pictures := combat.addedPictures(ground.Set, largeName, number)
    if len(pictures) > 0 {
        return pictures
    }
    // in a biome, the ones that take the place of the game's in the folder of its landscape
    if number <= largeGameCount && ground.BaseSet != ground.Set {
        pictures = combat.addedPictures(ground.BaseSet, largeName, number)
        if len(pictures) > 0 {
            return pictures
        }
    }

    if combat.largeMade == nil {
        combat.largeMade = make([][]*ebiten.Image, largeGameCount)

        lbx := terrainSetLbx(combat.Landscape, combat.Model.Plane)
        archive, err := combat.Cache.GetLbxFile(lbx)
        if err == nil {
            made := MakeLargePieces(func(index int) *image.Paletted {
                frames, err := archive.ReadImages(index)
                if err != nil || len(frames) == 0 {
                    return nil
                }
                // the pictures of the replacement folder, if it has any
                frames = mod.Replace(lbx, index, frames)
                return frames[0]
            })
            for index, picture := range made {
                if picture != nil && index < len(combat.largeMade) {
                    combat.largeMade[index] = []*ebiten.Image{ebiten.NewImageFromImage(picture)}
                }
            }
        }
    }

    if number < 1 || number > len(combat.largeMade) {
        return nil
    }
    return combat.largeMade[number - 1]
}

// the large pieces, over the ground of their tiles
func (combat *CombatScreen) drawLargePieces(screen *ebiten.Image, animationIndex uint64) {
    ground := combat.Model.Ground
    if ground == nil || len(ground.Large) == 0 {
        return
    }

    matrix := combat.GetCameraMatrix()
    var options ebiten.DrawImageOptions

    for _, piece := range ground.Large {
        tileX, tileY := CellToTile(piece.Cgx, piece.Cgy)
        brightness := borderBrightness(borderDepth(float64(tileX), float64(tileY)))
        if brightness <= 0 {
            continue
        }

        pictures := combat.largePictures(piece.Number)
        if len(pictures) == 0 {
            continue
        }
        picture := pictures[animationIndex % uint64(len(pictures))]

        // the ground picture of the top tile lies 16 from the left of the large one. a picture of
        // another size: the middle of its bottom edge on the bottom corner of the four tiles
        width := picture.Bounds().Dx()
        height := picture.Bounds().Dy()
        left := float64(TerrainOffsetX - largeTilePlaces[0].X) + float64(largeWidth - width) / 2
        top := float64(TerrainOffsetY + largeHeight - height)

        tx, ty := matrix.Apply(float64(tileX), float64(tileY))
        options.GeoM.Reset()
        options.GeoM.Translate(math.Floor(left), top)
        options.GeoM.Scale(combat.CameraScale, combat.CameraScale)
        options.GeoM.Translate(tx, ty)
        options.ColorScale.Reset()
        options.ColorScale.Scale(brightness, brightness, brightness, 1)
        scale.DrawScaled(screen, picture, &options)
    }
}

// the four tiles of a large piece as plain grass, laid as they lie on the screen: what a picture of
// a large piece is painted over. the export writes it next to the pictures
func MakeLargeTemplate(tiles func(index int) *image.Paletted) *image.Paletted {
    var out *image.Paletted

    for place, corner := range largeTilePlaces {
        tile := tiles(groundGrassFirst + place)
        if tile == nil {
            return nil
        }
        if out == nil {
            palette := make(color.Palette, len(tile.Palette))
            copy(palette, tile.Palette)
            out = image.NewPaletted(image.Rect(0, 0, largeWidth, largeHeight), palette)
        }

        bounds := tile.Bounds()
        for y := 0; y < bounds.Dy() && y < largeTileHeight; y++ {
            for x := 0; x < bounds.Dx() && x < largeTileWidth; x++ {
                value := tile.ColorIndexAt(bounds.Min.X + x, bounds.Min.Y + y)
                if value != 0 {
                    out.SetColorIndex(corner.X + x, corner.Y + y, value)
                }
            }
        }
    }

    return out
}

// the name of the pictures of large pieces, for the export
func LargeName() string {
    return largeName
}
