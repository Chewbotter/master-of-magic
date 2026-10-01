package banish

// The words over the picture of a wizard that is banished, matched to the original game.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/CONQUEST.c;
// MoX/src/Fonts.c: Load_Palette). The code is ours.
//
//   the screen has a palette of its own, the fourth of fonts.lbx (entry 5), and with it color
//     sets of the fonts of its own
//   font 5 in its color set 5 with a heavy outline (all around, two deep below and right) in 1,
//     the middle at 160,6 (upstream 160,10)
//   "<winner> banishes <loser>" when the loser has cities left, else "<winner> defeats <loser>"

import (
    "strings"
    "image/color"
    "log"

    "github.com/kazzmir/master-of-magic/lib/font"
    "github.com/kazzmir/master-of-magic/lib/lbx"
    fontslib "github.com/kazzmir/master-of-magic/game/magic/fonts"
)

// false: the words of upstream
var OriginalBanishText = true

const (
    banishTextMiddle = 160
    banishTextY = 6
    banishOutline = 1
    banishFont = 5
    banishColorSet = 5
    // the entry of fonts.lbx with the palette of this screen
    banishPaletteEntry = 5
)

// nil when the text can not be made
func banishText(cache *lbx.LbxCache) *font.StyledFont {
    if !OriginalBanishText {
        return nil
    }

    game := fontslib.LoadOriginal(cache, "", 0)
    if game == nil {
        return nil
    }

    fontsLbx, err := cache.GetLbxFile("fonts.lbx")
    if err != nil {
        return nil
    }

    blocks, err := font.ReadColorBlocks(fontsLbx, banishPaletteEntry)
    raw, err2 := fontsLbx.RawData(banishPaletteEntry)
    if err != nil || err2 != nil || len(raw) < 768 {
        log.Printf("Banish: unable to read the palette of the screen: %v %v", err, err2)
        return nil
    }

    // a palette of the data has 6 bits a color
    var palette color.Palette
    for index := range 256 {
        palette = append(palette, color.RGBA{R: raw[index * 3] << 2, G: raw[index * 3 + 1] << 2, B: raw[index * 3 + 2] << 2, A: 255})
    }

    own := &fontslib.Original{Fonts: game.Fonts, Blocks: blocks, Palette: palette}
    return own.Style(banishFont, own.Set(banishColorSet, -1), font.ShadowOutlineHeavy, banishOutline)
}

// Conquest_Animation: "Raiders destroy X's fortress." when raiders take the fortress of a wizard
// with cities left, "Raiders destroy X." when it was the last (on with the original's conquest)
var ClassicRaidersText = true

func possessive(name string) string {
    if strings.HasSuffix(name, "s") || strings.HasSuffix(name, "S") {
        return name + "'"
    }
    return name + "'s"
}

func raidersWords(loser string, citiesLeft int) string {
    if citiesLeft > 0 {
        return "Raiders destroy " + possessive(loser) + " fortress."
    }
    return "Raiders destroy " + loser + "."
}

func banishWords(winner string, loser string, citiesLeft int) string {
    if citiesLeft > 0 {
        return winner + " banishes " + loser
    }
    return winner + " defeats " + loser
}
