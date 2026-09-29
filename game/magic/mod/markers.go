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

// the look of a tab under the mouse as the game makes it
func MakeTabHover(tab image.Image) image.Image {
    bounds := tab.Bounds()
    out := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))

    lighter := func(value uint32) uint8 {
        return uint8(min(255, float64(value >> 8) * (1 + TabHoverLight)))
    }

    for y := 0; y < bounds.Dy(); y++ {
        for x := 0; x < bounds.Dx(); x++ {
            red, green, blue, alpha := tab.At(bounds.Min.X + x, bounds.Min.Y + y).RGBA()
            if alpha != 0 {
                out.Set(x, y, color.NRGBA{R: lighter(red), G: lighter(green), B: lighter(blue), A: 255})
            }
        }
    }

    return out
}
