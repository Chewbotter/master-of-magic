package game

// The text of the screen of the items (the vault), matched to the original game.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/ItemScrn.c:
// Item_Screen_Draw). The code is ours.
//
//   the window (armylist 5) at 17,2
//   gold and mana: the numbers font 1 (font 0 when one of them is above 19999) in its color set
//     0, ending on 183 and 217 at y 166; "GP" and "MP" font 0 from 186 and 220
//   a hero: the name font 1, letters 179, 36 right of the portrait and a pixel over its top;
//     under it, 7 lower, "Same Location" with letters 157 or "Item Teleport" in the color set 2
//   ALL with the soft edge 225 of the palette of the window and a shadow below and right in 0
//     (black)

import (
    "fmt"
    "image/color"

    "github.com/kazzmir/master-of-magic/lib/font"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    fontslib "github.com/kazzmir/master-of-magic/game/magic/fonts"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: the text of upstream
var OriginalVaultText = true

const (
    vaultEdge = 225
    vaultGoldRight = 183
    vaultManaRight = 217
    vaultGoldWordX = 186
    vaultManaWordX = 220
    vaultReservesY = 166
    // above this the numbers are in the narrow font
    vaultNarrowFrom = 19999
    vaultNameColor = 179
    vaultSameColor = 157
    vaultTeleportSet = 2
    vaultNameX = 36
    vaultNameY = -1
    vaultPlaceY = 6
)

type vaultStyle struct {
    Number *font.StyledFont
    Narrow *font.StyledFont
    Name *font.StyledFont
    Same *font.StyledFont
    Teleport *font.StyledFont
}

// nil when the text can not be made
func (game *Game) vaultStyle() *vaultStyle {
    if !OriginalVaultText {
        return nil
    }

    original := fontslib.LoadOriginal(game.Cache, "armylist.lbx", 5)
    if original == nil {
        return nil
    }

    black := color.RGBA{A: 255}
    make := func(fontIndex int, colors []color.Color) *font.StyledFont {
        if colors == nil {
            return nil
        }
        return font.MakeStyledFont(original.Fonts[fontIndex], colors, font.ShadowDown, black)
    }

    style := &vaultStyle{
        Number: make(1, original.Set(0, vaultEdge)),
        Narrow: make(0, original.Set(0, vaultEdge)),
        Name: make(1, original.Flat(vaultEdge, vaultNameColor)),
        Same: make(1, original.Flat(vaultEdge, vaultSameColor)),
        Teleport: make(1, original.Set(vaultTeleportSet, vaultEdge)),
    }
    if style.Number == nil || style.Narrow == nil || style.Name == nil || style.Same == nil || style.Teleport == nil {
        return nil
    }
    return style
}

func (style *vaultStyle) drawReserves(screen *ebiten.Image, gold int, mana int) {
    number := style.Number
    if gold > vaultNarrowFrom || mana > vaultNarrowFrom {
        number = style.Narrow
    }

    var options ebiten.DrawImageOptions
    left := font.FontOptions{Options: &options, Scale: scale.ScaleAmount}
    right := left
    right.Justify = font.FontJustifyRight

    number.Print(screen, vaultGoldRight, vaultReservesY, right, fmt.Sprintf("%v", gold))
    number.Print(screen, vaultManaRight, vaultReservesY, right, fmt.Sprintf("%v", mana))
    style.Narrow.Print(screen, vaultGoldWordX, vaultReservesY, left, "GP")
    style.Narrow.Print(screen, vaultManaWordX, vaultReservesY, left, "MP")
}

// the name of a hero whose portrait is at x, y, and where the item that is held would go
func (style *vaultStyle) drawHero(screen *ebiten.Image, x int, y int, name string, holding bool, same bool) {
    var options ebiten.DrawImageOptions
    print := font.FontOptions{Options: &options, Scale: scale.ScaleAmount}

    style.Name.Print(screen, x + vaultNameX, y + vaultNameY, print, name)
    if !holding {
        return
    }

    if same {
        style.Same.Print(screen, x + vaultNameX, y + vaultPlaceY, print, "Same Location")
    } else {
        style.Teleport.Print(screen, x + vaultNameX, y + vaultPlaceY, print, "Item Teleport")
    }
}
