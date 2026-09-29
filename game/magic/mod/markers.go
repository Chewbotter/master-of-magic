package mod

// Markers of the world map that are not pictures of the game: the flag on the tiles of a path a
// stack will only walk in a later turn.
//
// The flag is made here, of colors of the game's palette. Its cloth has the colors 215 to 217,
// the green that figures carry as the color of their wizard, so the game can give it the color of
// any wizard the way it does with figures. A picture `markers/flag.png` in the replacement folder
// takes its place; it can have any size and is put with its middle on the middle of the tile.

import (
    "image"
    "image/color"
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
