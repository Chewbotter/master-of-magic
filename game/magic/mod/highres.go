package mod

// AN EXPERIMENT (user, 2026-09-29): pictures of the ground that are as large as 2 by 2 tiles.
//
// A folder of a biome can have a twin whose name ends in _highres, with the names of the folder
// and pictures of the ground that are twice as wide and as high:
//
//   environment/Forest/grass 1_0.png           30 by 16
//   environment/Forest_highres/grass 1_0.png   60 by 32
//
// First they were drawn at half their size, in the place of one tile with pixels half as large.
// The user: "this doesn't look good. Let's try the same tile set but scaled to be the original
// pixel density - in other words, each tile is a 2x2 so we keep the pixel density but see less
// repetition."
//
// So a picture of a twin lies over 2 by 2 tiles, pixel for pixel. It is cut in four QUARTERS, each
// with the shape of a tile, which are the frames 0 to 3 of its entry: the top one, the right one,
// the left one, the bottom one of the four tiles on the screen. A tile shows the quarter of where
// it lies among the four (combat/terraindraw.go).
//
// Only grass, dirt and the edges of dirt. Raised ground, trees, rocks and large pieces of a twin
// are not used: they would be twice as large.

import (
    "fmt"
    "image"
    "path/filepath"
    "strings"
)

// turns it off: the twins are not looked at
var HighRes = true

const HighResSuffix = "_highres"
// the quarters of a picture
const Quarters = 4

// the twin of a folder of the environment
func HighResFolder(set string) string {
    if set == "" {
        return ""
    }
    return set + HighResSuffix
}

// the name of the picture of the ground of an entry without its number, as "grass" and "dirt edge
// SE". nothing for what is no picture of a landscape
func GroundName(archive string, entry int) string {
    picture, ok := environmentEntries[entryKey(archive, entry)]
    if !ok || EnvironmentSet(archive) == "" {
        return ""
    }
    name := picture.Name
    cut := strings.LastIndex(name, " ")
    if cut > 0 && strings.Trim(name[cut + 1:], "0123456789") == "" {
        name = name[:cut]
    }
    return name
}

// the file of the picture of an entry in the twin of the folder of the biome that is fought in.
// nothing without a biome, and for what is not grass or dirt
func quartersPath(archive string, entry int) string {
    if !HighRes || folder == "" || biome == "" {
        return ""
    }
    name := GroundName(archive, entry)
    if !strings.HasPrefix(name, "grass") && !strings.HasPrefix(name, "dirt") {
        return ""
    }
    picture := environmentEntries[entryKey(archive, entry)]
    return filepath.Join(folder, environmentFolder, HighResFolder(BiomeFolder(biome, picture.Set)), EnvironmentFrameFile(picture.Name, 0))
}

// true if the frames of an entry are the quarters of a picture of 2 by 2 tiles
func Quartered(archive string, entry int) bool {
    path := quartersPath(archive, entry)
    return path != "" && hasFile(path)
}

// where in a picture of 2 by 2 tiles the picture of one of its tiles starts. across and down say
// which of the tiles along the two sides of the grid, 0 or 1 each
func quarterCorner(size image.Point, tile image.Point, across int, down int) image.Point {
    return image.Pt((size.X - tile.X) / 2 + (across - down) * (size.X - tile.X) / 2, (across + down) * (size.Y - tile.Y) / 2)
}

// the number of the frame of a quarter
func QuarterFrame(across int, down int) int {
    return down * 2 + across
}

// the quarters of the picture of an entry, in the shape of the picture of a tile. nil if there is
// none, or if it is not twice as large as the tile
func readQuarters(archive string, entry int, shape *image.Paletted) []*image.Paletted {
    path := quartersPath(archive, entry)
    if path == "" || !hasFile(path) {
        return nil
    }
    whole := readFrame([]string{path}, shape.Palette)
    if whole == nil {
        return nil
    }

    size := whole.Bounds().Size()
    tile := shape.Bounds().Size()
    if size.X != tile.X * 2 || size.Y != tile.Y * 2 {
        reportOnce(fmt.Sprintf("Picture %v is %v by %v, it has to be %v by %v. The plain picture is used", path, size.X, size.Y, tile.X * 2, tile.Y * 2))
        return nil
    }

    // the color of the picture at a place. where it is see-through, the one of the nearest place
    // toward its middle that is not: the shape of a tile is a little wider at the edge of the
    // picture than the picture is
    colorAt := func(x int, y int) uint8 {
        for range size.X {
            index := whole.ColorIndexAt(whole.Bounds().Min.X + x, whole.Bounds().Min.Y + y)
            if index != clearColor {
                return index
            }
            switch {
                case x < size.X / 2 - 1: x += 1
                case x > size.X / 2: x -= 1
            }
            switch {
                case y < size.Y / 2 - 1: y += 1
                case y > size.Y / 2: y -= 1
            }
        }
        return clearColor
    }

    out := make([]*image.Paletted, Quarters)
    for down := range 2 {
        for across := range 2 {
            corner := quarterCorner(size, tile, across, down)
            quarter := image.NewPaletted(image.Rect(0, 0, tile.X, tile.Y), shape.Palette)
            for y := range tile.Y {
                for x := range tile.X {
                    if shape.ColorIndexAt(shape.Bounds().Min.X + x, shape.Bounds().Min.Y + y) != clearColor {
                        quarter.SetColorIndex(x, y, colorAt(corner.X + x, corner.Y + y))
                    }
                }
            }
            out[QuarterFrame(across, down)] = quarter
        }
    }
    return out
}
