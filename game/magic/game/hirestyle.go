package game

// The windows of a hero or mercenaries that ask to be hired and of the merchant, and the box a
// name is typed in, matched to the original game.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/HIRE.c: Hire_Hero_Popup,
// Hire_Hero_Popup_Draw, Hire_Merc_Popup_Draw, Merchant_Popup, Merchant_Popup_Draw;
// MoM/src/CityScr.c: Change_Home_City_Name_Popup; MoM/src/Outpost.c: Change_City_Name_Popup;
// MoX/src/Fields.c: Draw_Field; MoX/src/Input.c: Draw_Input_Box_Popup). The code is ours.
//
//   HERO, MERCENARIES: the banner (hire 0) at 0,0 and its words in the middle at 128,6; the
//     window of the unit at 25,17 (upstream 31,16), the box of the buttons 213,133 from it, the
//     buttons 221,143 and 221,162 from it
//   the words of the banner: "Hero for Hire: N gold", "Hero Summoned", "Champion Summoned",
//     "A prisoner asks to join your army", "Mercenary for Hire: N gold",
//     "N Mercenaries for Hire: N gold"; the first button is "Hire" for a hero that asks for
//     gold, else "Accept"
//   MERCHANT: the picture (hire 2) at 5,15; the words from 62,24 in lines of 182, with two spaces
//     after "sale.", a shadow below and right in 253 of the palette of the picture; the item at
//     19,80; the buttons (hire 1) at 257,136 and 257,155
//   ALL TEXT: font 4 in its color set 4 with the soft edge 163; no shadow but for the merchant
//   THE WORD OF A BUTTON: its middle on the middle of the button, a pixel right and down while
//     the button is down
//   THE BOX OF A NAME (backgrnd 33): of the first city at 60,30, of an outpost at 85,100, of a
//     hero at 54,72; the title 10,6 from the box ("Name Your Hero" in the middle, 75,6); the name
//     16,21 from it (a hero 16,20), at most 75 wide and 12 letters (a hero 120 and 13); soft edge
//     103 (a hero 163); the cursor is a block under the letter that comes next, which fills up
//     from below row by row and empties again, in the colors 97 to 103 (a hero 176 to 185)
//
// OURS: the title "Set Name" of the city screen, which the original does not have; it is at the
// place of the outpost's.

import (
    "fmt"
    "image"
    "image/color"

    "github.com/kazzmir/master-of-magic/lib/font"
    "github.com/kazzmir/master-of-magic/lib/lbx"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    fontslib "github.com/kazzmir/master-of-magic/game/magic/fonts"
    herolib "github.com/kazzmir/master-of-magic/game/magic/hero"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: the windows of upstream
var OriginalHireText = true

// false: the box of a name of upstream
var OriginalNameInput = true

const (
    hireFont = 4
    hireColorSet = 4
    hireEdge = 163

    hireTitleX = 128
    hireTitleY = 6
    hireWindowX = 25
    hireWindowY = 17
    hireButtonBoxX = hireWindowX + 213
    hireButtonBoxY = hireWindowY + 133
    hireButtonX = hireWindowX + 221
    hireButtonY = hireWindowY + 143
    hireRejectY = hireWindowY + 162

    merchantX = 5
    merchantY = 15
    merchantTextX = 62
    merchantTextY = 24
    merchantTextWidth = 182
    merchantShadow = 253
    merchantItemX = 19
    merchantItemY = 80
    merchantButtonX = 257
    merchantBuyY = 136
    merchantRejectY = 155

    // a button that is down has its word this much right and down
    buttonDownShift = 1
)

// why a hero asks
type hireKind int
const (
    hireForGold hireKind = iota
    hireSummoned
    hirePrisoner
    hireChampion
)

func hireKindOf(hero *herolib.Hero, cost int, atFortress bool) hireKind {
    switch {
        case cost > 0: return hireForGold
        case !atFortress: return hirePrisoner
        case hero.IsChampion(): return hireChampion
    }
    return hireSummoned
}

func (kind hireKind) Title(cost int) string {
    switch kind {
        case hireSummoned: return "Hero Summoned"
        case hireChampion: return "Champion Summoned"
        case hirePrisoner: return "A prisoner asks to join your army"
    }
    return fmt.Sprintf("Hero for Hire: %v gold", cost)
}

func (kind hireKind) Button() string {
    if kind == hireForGold {
        return "Hire"
    }
    return "Accept"
}

func mercenariesTitle(count int, cost int) string {
    if count > 1 {
        return fmt.Sprintf("%v Mercenaries for Hire: %v gold", count, cost)
    }
    return fmt.Sprintf("Mercenary for Hire: %v gold", cost)
}

func merchantMessage(item string, cost int) string {
    return fmt.Sprintf("A merchant arrives and offers a magic %v for sale.  The price is only %v gold pieces.", item, cost)
}

// the text of the windows of hiring, nil when it can not be made
func hireText2(cache *lbx.LbxCache) *font.StyledFont {
    if !OriginalHireText {
        return nil
    }

    original := fontslib.LoadOriginal(cache, "hire.lbx", 0)
    if original == nil {
        return nil
    }
    return original.Style(hireFont, original.Set(hireColorSet, hireEdge), font.ShadowNone, -1)
}

// the words of the merchant, nil when they can not be made
func merchantText(cache *lbx.LbxCache) *fontslib.Shadowed {
    if !OriginalHireText {
        return nil
    }

    original := fontslib.LoadOriginal(cache, "hire.lbx", 2)
    if original == nil {
        return nil
    }

    text := original.Style(hireFont, original.Set(hireColorSet, hireEdge), font.ShadowDown, merchantShadow)
    if text == nil {
        return nil
    }
    return &fontslib.Shadowed{Text: text, LineStep: text.LineHeight()}
}

// the word of a button as the original puts it
func printButtonWord(screen *ebiten.Image, text *font.StyledFont, button image.Rectangle, down bool, alpha float32, word string) {
    x := button.Min.X + (button.Dx() - 1) / 2
    y := button.Min.Y + (button.Dy() - 1) / 2 - (text.Height() - 1) / 2
    if down {
        x += buttonDownShift
        y += buttonDownShift
    }

    var options ebiten.DrawImageOptions
    options.ColorScale.ScaleAlpha(alpha)
    text.Print(screen, x, y, font.FontOptions{Options: &options, Scale: scale.ScaleAmount, Justify: font.FontJustifyCenter}, word)
}

// the box a name is typed in
type nameInput struct {
    X int
    Y int
    Title string
    // the title has its middle on the middle of the box, else it starts at the left
    TitleMiddle bool
    TextDown int
    Width int
    Letters int
    Edge int
    Cursor []int
}

const (
    nameInputLbx = "backgrnd.lbx"
    nameInputIndex = 33
    nameTitleX = 10
    nameTitleMiddleX = 75
    nameTitleY = 6
    nameTextX = 16
    // game ticks a step of the cursor lasts
    nameCursorTicks = 3
)

var cityCursorColors = []int{97, 97, 98, 98, 99, 99, 100, 100, 101, 101, 102, 102, 103, 103}
var heroCursorColors = []int{176, 177, 178, 179, 180, 181, 182, 183, 184, 185, 176, 177, 178, 179}

// the box for a title of upstream's callers
func nameInputOf(title string, x int, y int) (nameInput, bool) {
    if !OriginalNameInput {
        return nameInput{}, false
    }

    city := nameInput{X: x, Y: y, Title: title, TextDown: 21, Width: 75, Letters: 12, Edge: 103, Cursor: cityCursorColors}

    switch title {
        case "New Starting City":
            city.X, city.Y, city.Title = 60, 30, "Name Starting City"
        case "New Outpost", "Set Name":
            city.X, city.Y = 85, 100
        case "Hero Name":
            return nameInput{X: 54, Y: 72, Title: "Name Your Hero", TitleMiddle: true, TextDown: 20, Width: 120, Letters: 13, Edge: hireEdge, Cursor: heroCursorColors}, true
    }

    return city, true
}

type nameInputStyle struct {
    Text *font.StyledFont
    Cursor []color.Color
}

func (input nameInput) style(cache *lbx.LbxCache) *nameInputStyle {
    original := fontslib.LoadOriginal(cache, nameInputLbx, nameInputIndex)
    if original == nil {
        return nil
    }

    text := original.Style(hireFont, original.Set(hireColorSet, input.Edge), font.ShadowNone, -1)
    if text == nil {
        return nil
    }
    return &nameInputStyle{Text: text, Cursor: original.Colors(input.Cursor...)}
}

// a name cut to what the box takes
func (input nameInput) fit(style *nameInputStyle, name string) string {
    for len(name) > 0 && (len(name) > input.Letters || style.Text.Width(name) > input.Width) {
        name = name[:len(name) - 1]
    }
    return name
}

var nameCursorPixel *ebiten.Image

// the rows of the cursor that show at a step, from the top of the letters, with the number of the color of each
func nameCursorRows(step int, height int) map[int]int {
    rows := make(map[int]int)
    if step <= 0 {
        return rows
    }

    if step <= height {
        // fills up from below
        for drawn := 0; drawn < step; drawn++ {
            rows[height - step + 1 + drawn] = 1 + drawn
        }
        return rows
    }

    // empties from above
    left := height - (step - height)
    for drawn := 0; left - drawn > 0; drawn++ {
        rows[left - drawn - 1] = height - drawn
    }
    return rows
}

func (input nameInput) draw(screen *ebiten.Image, style *nameInputStyle, background *ebiten.Image, name string, counter uint64) {
    var options ebiten.DrawImageOptions
    options.GeoM.Translate(float64(input.X), float64(input.Y))
    scale.DrawScaled(screen, background, &options)

    var plain ebiten.DrawImageOptions
    print := font.FontOptions{Options: &plain, Scale: scale.ScaleAmount}

    if input.TitleMiddle {
        middle := print
        middle.Justify = font.FontJustifyCenter
        style.Text.Print(screen, input.X + nameTitleMiddleX, input.Y + nameTitleY, middle, input.Title)
    } else {
        style.Text.Print(screen, input.X + nameTitleX, input.Y + nameTitleY, print, input.Title)
    }

    textX := input.X + nameTextX
    textY := input.Y + input.TextDown

    // the cursor, under the letters
    if nameCursorPixel == nil {
        nameCursorPixel = ebiten.NewImage(1, 1)
        nameCursorPixel.Fill(color.White)
    }

    height := style.Text.Height() - 1
    step := int(counter / nameCursorTicks) % (2 * (height + 1))
    cursorX := textX + style.Text.Width(name)
    cursorWidth := style.Text.Width(" ") + 2
    for row, number := range nameCursorRows(step, height) {
        if number < 0 || number >= len(style.Cursor) || style.Cursor[number] == nil {
            continue
        }

        var rowOptions ebiten.DrawImageOptions
        rowOptions.GeoM.Scale(float64(cursorWidth), 1)
        rowOptions.GeoM.Translate(float64(cursorX), float64(textY + row))
        rowOptions.ColorScale.ScaleWithColor(style.Cursor[number])
        scale.DrawScaled(screen, nameCursorPixel, &rowOptions)
    }

    style.Text.Print(screen, textX, textY, print, name)
}
