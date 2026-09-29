package font

// The color sets of the fonts. The original keeps 16 sets of 16 palette indexes with every
// palette of fonts.lbx, after the 768 bytes of the palette itself, and picks one by its number
// when it sets the style of a text (ReMoM: MoX/src/Fonts.c Load_Palette, Set_Font_Style: the
// pixel value k of a letter gets color k of the set; 0 is the soft edge, which the original
// often sets by itself afterwards).

import (
    "fmt"

    "github.com/kazzmir/master-of-magic/lib/lbx"
)

// the entry of fonts.lbx with the palette of the game
const GamePaletteEntry = 2

const colorBlocksStart = 768
const ColorBlockCount = 16
const ColorBlockSize = 16

// the color sets of a palette of fonts.lbx, as palette indexes
func ReadColorBlocks(fontsLbx *lbx.LbxFile, entry int) ([][]int, error) {
    data, err := fontsLbx.RawData(entry)
    if err != nil {
        return nil, err
    }

    if len(data) < colorBlocksStart + ColorBlockCount * ColorBlockSize {
        return nil, fmt.Errorf("entry %v of fonts.lbx has no color sets: %v bytes", entry, len(data))
    }

    var blocks [][]int
    for block := range ColorBlockCount {
        var colors []int
        for index := range ColorBlockSize {
            colors = append(colors, int(data[colorBlocksStart + block * ColorBlockSize + index]))
        }
        blocks = append(blocks, colors)
    }

    return blocks, nil
}
