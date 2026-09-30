package game

// The box that tells what was found in a lair, matched to the original game.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/Lair.c:
// Lair_Treasure_Popup, Add_Comma_Or_And, An; MoX/src/GENDRAW.c: Notify1, Notify1_Draw, Notify_Load;
// MoM/src/ItemMake.c: Get_Item_Type_Name). The code is ours.
//
//   the box is AS WIDE AS ITS TEXT: the text is broken into lines of 175 at most, the widest line
//     is the width; the picture of the box (resource 56) is cut off 12 after the text, there the
//     box of the picture (resource 57) follows over a black field, with the chest (reload 20) 6
//     right and 8 down in it
//   the whole has its middle at x 190: the box starts at 190 - (96 + width) / 2; y 60
//   the text: font 4 in its color set 4 with soft edge 21, 10 right and 8 down in the box; under
//     it the same all in 17 a pixel lower and a pixel lower and right
//   the words: "Inside you find " and what was found with ", " between and " and " before the
//     last, then a full stop: "absolutely nothing", "N gold", "N mana crystals", "a prisoner",
//     "a Web spell", "1 Life spell book" ("2 Life spell books"), "a retort of Alchemy", and an
//     item by its KIND ("a Sword", "an Amulet"), not by its name
//
// OURS: the kind of an item is the first word of its name where the name is "<kind> of ...".

import (
    "fmt"
    "image"
    "image/color"
    "strings"

    "github.com/kazzmir/master-of-magic/lib/font"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    fontslib "github.com/kazzmir/master-of-magic/game/magic/fonts"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: the box of upstream
var OriginalTreasureBox = true

const (
    treasureLbx = "resource.lbx"
    treasureBoxIndex = 56
    treasurePictureBoxIndex = 57
    treasureMiddleX = 190
    treasureY = 60
    treasureTextWidth = 175
    treasureTextX = 10
    treasureTextY = 8
    treasureEdge = 21
    treasureShadow = 17
    // from the end of the text to the box of the picture
    treasureTextAfter = 12
    // what the box of the picture and the rims add to the width of the text
    treasureAround = 96
    treasureChestX = 6
    treasureChestY = 8
    treasurePictureWidth = 51
    treasurePictureHeight = 56
)

func treasureItemWords(item TreasureItem) string {
    switch found := item.(type) {
        case *TreasureGold: return fmt.Sprintf("%v gold", found.Amount)
        case *TreasureMana: return fmt.Sprintf("%v mana crystals", found.Amount)
        case *TreasurePrisonerHero: return "a prisoner"
        case *TreasureSpell: return fmt.Sprintf("%v %v spell", articleFor(found.Spell.Name), found.Spell.Name)
        case *TreasureSpellbook:
            count := max(found.Count, 1)
            if count > 1 {
                return fmt.Sprintf("%v %v spell books", count, found.Magic)
            }
            return fmt.Sprintf("%v %v spell book", count, found.Magic)
        case *TreasureRetort: return fmt.Sprintf("a retort of %v", found.Retort)
        case *TreasureMagicalItem:
            kind := found.Artifact.Type.Name()
            before, _, isNamedByKind := strings.Cut(found.Artifact.Name, " of ")
            if isNamedByKind && !strings.Contains(before, " ") {
                kind = before
            }
            if kind == "" || kind == "Misc" {
                kind = "Item"
            }
            return articleFor(kind) + " " + kind
    }
    return item.String()
}

// the words of the original for what was found
func treasureMessage(treasure Treasure) string {
    if len(treasure.Treasures) == 0 {
        return "Inside you find absolutely nothing."
    }

    out := "Inside you find "
    for index, item := range treasure.Treasures {
        switch {
            case index == 0:
            case index == len(treasure.Treasures) - 1: out += " and "
            default: out += ", "
        }
        out += treasureItemWords(item)
    }
    return out + "."
}

// draws the box. false when it can not be made and nothing was drawn
func (game *Game) drawTreasureOriginal(screen *ebiten.Image, treasure Treasure, alpha float32) bool {
    if !OriginalTreasureBox {
        return false
    }

    original := fontslib.LoadOriginal(game.Cache, treasureLbx, treasureBoxIndex)
    if original == nil {
        return false
    }
    text := original.Shadowed(4, 4, treasureEdge, treasureShadow)
    box, err1 := game.ImageCache.GetImage(treasureLbx, treasureBoxIndex, 0)
    pictureBox, err2 := game.ImageCache.GetImage(treasureLbx, treasurePictureBoxIndex, 0)
    chest, err3 := game.ImageCache.GetImage("reload.lbx", 20, 0)
    if text == nil || err1 != nil || err2 != nil || err3 != nil {
        return false
    }

    lines := text.Wrap(treasureMessage(treasure), treasureTextWidth)
    width := text.Widest(lines)
    boxX := treasureMiddleX - (treasureAround + width) / 2
    pictureX := boxX + width + treasureTextAfter

    var options ebiten.DrawImageOptions
    options.ColorScale.ScaleAlpha(alpha)
    options.GeoM.Translate(float64(boxX), float64(treasureY))
    cut := image.Rect(0, 0, min(width + treasureTextAfter, box.Bounds().Dx()), box.Bounds().Dy()).Add(box.Bounds().Min)
    scale.DrawScaled(screen, box.SubImage(cut).(*ebiten.Image), &options)

    if nameCursorPixel == nil {
        nameCursorPixel = ebiten.NewImage(1, 1)
        nameCursorPixel.Fill(color.White)
    }
    var black ebiten.DrawImageOptions
    black.GeoM.Scale(treasurePictureWidth, treasurePictureHeight)
    black.GeoM.Translate(float64(pictureX), float64(treasureY))
    black.ColorScale.Scale(0, 0, 0, 1)
    black.ColorScale.ScaleAlpha(alpha)
    scale.DrawScaled(screen, nameCursorPixel, &black)

    options.GeoM.Reset()
    options.GeoM.Translate(float64(pictureX), float64(treasureY))
    scale.DrawScaled(screen, pictureBox, &options)

    options.GeoM.Reset()
    options.GeoM.Translate(float64(pictureX + treasureChestX), float64(treasureY + treasureChestY))
    scale.DrawScaled(screen, chest, &options)

    text.PrintLines(screen, func(int) int { return boxX + treasureTextX }, treasureY + treasureTextY, font.FontJustifyLeft, alpha, lines)
    return true
}
