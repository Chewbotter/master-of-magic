package mod

// Markers of the world map that are not pictures of the game: the flag on the tiles of a path a
// stack will only walk in a later turn.
//
// The flag is made here, of colors of the game's palette. Its cloth has the colors 215 to 217,
// the green that figures carry as the color of their wizard, so the game can give it the color of
// any wizard the way it does with figures. A picture `markers/flag.png` in the replacement folder
// takes its place; it can have any size and is put with its middle on the middle of the tile.

import (
    "fmt"
    "image"
    "image/color"
    "image/png"
    "os"
    "path/filepath"
)

const markersFolder = "markers"
const FlagFile = "flag.png"

// the colors of the palette the flag is made of
const flagOutline = 1
const flagPole = 13
const flagClothLight = 215
const flagCloth = 216
const flagClothDark = 217

// o outline, p pole, 1 2 3 cloth from light to dark, anything else see-through
var flagRows = []string{
    "ooo.........",
    "opoooo......",
    "op1122oo....",
    "op112223oo..",
    "op11222233o.",
    "op112223oo..",
    "op1122oo....",
    "opoooo......",
    "opo.........",
    "opo.........",
    "opo.........",
    "ooo.........",
}

// the flag as the game makes it
func DefaultFlag(palette color.Palette) *image.Paletted {
    picture := image.NewPaletted(image.Rect(0, 0, len(flagRows[0]), len(flagRows)), palette)

    for y, row := range flagRows {
        for x, pixel := range row {
            index := 0
            switch pixel {
                case 'o': index = flagOutline
                case 'p': index = flagPole
                case '1': index = flagClothLight
                case '2': index = flagCloth
                case '3': index = flagClothDark
            }
            picture.SetColorIndex(x, y, uint8(index))
        }
    }

    return picture
}

// the flag, the one of the replacement folder if there is one
func MarkerFlag(palette color.Palette) *image.Paletted {
    if folder != "" {
        replacement := readFrame([]string{filepath.Join(folder, markersFolder, FlagFile)}, palette)
        if replacement != nil {
            return replacement
        }
    }

    return DefaultFlag(palette)
}

// The tabs in the corners of the spellbooks that turn the pages, as they look under the mouse.
// The game makes that look from the tab: the same picture, lighter by TabHoverLight. A picture
// `markers/<name> hover.png` in the replacement folder takes its place, as it is (any colors,
// see-through pixels stay so). It is put with its lower left corner on the lower left corner of
// the tab, so a picture that is higher than the tab reaches up.
const TabCastLeft = "tab cast left"
const TabCastRight = "tab cast right"
const TabResearchLeft = "tab research left"
const TabResearchRight = "tab research right"

const TabHoverLight = 0.2

// the archive and entry of the picture of a tab
type TabPicture struct {
    Name string
    Archive string
    Entry int
}

var TabPictures = []TabPicture{
    {Name: TabCastLeft, Archive: "spells.lbx", Entry: 1},
    {Name: TabCastRight, Archive: "spells.lbx", Entry: 2},
    {Name: TabResearchLeft, Archive: "scroll.lbx", Entry: 7},
    {Name: TabResearchRight, Archive: "scroll.lbx", Entry: 8},
}

func TabHoverName(name string) string {
    return name + " hover"
}

// a picture of the markers folder as it is, nil when there is none
func ReadMarker(name string) image.Image {
    if folder == "" || name == "" {
        return nil
    }

    path := filepath.Join(folder, markersFolder, name + ".png")
    // the files of the folder are looked up anew
    delete(folderLists, filepath.Dir(path))
    if !hasFile(path) {
        return nil
    }

    file, err := os.Open(path)
    if err != nil {
        return nil
    }
    picture, err := png.Decode(file)
    file.Close()
    if err != nil {
        reportOnce(fmt.Sprintf("Replacement picture %v can not be read: %v", path, err))
        return nil
    }

    reportOnce(fmt.Sprintf("Replacement picture %v", path))
    return picture
}

// how light a pixel is, 0 to 255
func lightOf(pixel color.Color) float64 {
    red, green, blue, _ := pixel.RGBA()
    return (float64(red >> 8) * 299 + float64(green >> 8) * 587 + float64(blue >> 8) * 114) / 1000
}

// a pixel belongs to the face of a tab when it is at least this light, as a part of the lightest
// pixel of the tab. the folds around the face are darker
const tabFaceLight = 0.72

// the face of a tab: the light triangle of paper inside of its dark folds. true for the pixels
// of it, row by row. found from the lightest pixel of the tab, going on to the light pixels next
// to it, above, below, left and right: the folds are lines of dark pixels all around the face, and
// what is light outside of them (the rim of the page) is not reached
func TabFace(tab image.Image) []bool {
    bounds := tab.Bounds()
    width := bounds.Dx()
    height := bounds.Dy()
    face := make([]bool, width * height)

    solid := func(x int, y int) bool {
        if x < 0 || y < 0 || x >= width || y >= height {
            return false
        }
        _, _, _, alpha := tab.At(bounds.Min.X + x, bounds.Min.Y + y).RGBA()
        return alpha != 0
    }

    // the lightest pixel that has the tab on all four sides of it
    lightest := -1.0
    start := image.Pt(-1, -1)
    for y := 0; y < height; y++ {
        for x := 0; x < width; x++ {
            if !solid(x, y) || !solid(x - 1, y) || !solid(x + 1, y) || !solid(x, y - 1) || !solid(x, y + 1) {
                continue
            }
            light := lightOf(tab.At(bounds.Min.X + x, bounds.Min.Y + y))
            if light > lightest {
                lightest = light
                start = image.Pt(x, y)
            }
        }
    }

    if start.X < 0 {
        return face
    }

    from := lightest * tabFaceLight
    todo := []image.Point{start}
    face[start.Y * width + start.X] = true
    for len(todo) > 0 {
        at := todo[len(todo) - 1]
        todo = todo[:len(todo) - 1]

        for _, step := range []image.Point{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
            next := at.Add(step)
            if !solid(next.X, next.Y) || face[next.Y * width + next.X] {
                continue
            }
            if lightOf(tab.At(bounds.Min.X + next.X, bounds.Min.Y + next.Y)) < from {
                continue
            }
            face[next.Y * width + next.X] = true
            todo = append(todo, next)
        }
    }

    return face
}

// the look of a tab under the mouse as the game makes it: its face is lighter, its folds are as they are
func MakeTabHover(tab image.Image) image.Image {
    bounds := tab.Bounds()
    out := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
    face := TabFace(tab)

    lighter := func(value uint32) uint8 {
        return uint8(min(255, float64(value >> 8) * (1 + TabHoverLight)))
    }

    for y := 0; y < bounds.Dy(); y++ {
        for x := 0; x < bounds.Dx(); x++ {
            red, green, blue, alpha := tab.At(bounds.Min.X + x, bounds.Min.Y + y).RGBA()
            if alpha == 0 {
                continue
            }

            if face[y * bounds.Dx() + x] {
                out.Set(x, y, color.NRGBA{R: lighter(red), G: lighter(green), B: lighter(blue), A: 255})
            } else {
                out.Set(x, y, color.NRGBA{R: uint8(red >> 8), G: uint8(green >> 8), B: uint8(blue >> 8), A: 255})
            }
        }
    }

    return out
}
