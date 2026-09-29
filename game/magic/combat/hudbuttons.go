package combat

// The right column of the buttons of the combat bar with the modern controls (user, 2026-09-29):
// AUTO, STAY, END in the place of WAIT, AUTO, DONE.
//
//   WAIT is gone: a click on a unit picks it (unitorder.go), so nothing has to be waited for.
//   STAY is what DONE was: the turn of the selected unit is over.
//   END is new: the turn of every unit of the player that still has its turn is over.
//
// The original has no pictures for STAY and END, the words of its buttons are painted on them.
// They are made here from the picture of DONE: its face without the word (the pixels of the word
// take the colors of the face left of it, row by row), and the word in the letters of the
// buttons, which are the smallest font of the game with a light edge right of and below them.
// The colors were read from the pictures of the buttons (compix.lbx 3 and 28).

import (
    "image"
    "image/color"
    "log"

    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/lib/font"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: the buttons of the original also with the modern controls
const ModernButtons = true

// the picture the made buttons are made of, and the one of it when the buttons can not be used
const madeButtonIndex = 3
const madeButtonDisabledIndex = 28

// the word of the picture is inside of these columns and rows, its light edge too
const (
    madeButtonWordLeft = 4
    madeButtonWordRight = 21
    madeButtonWordTop = 1
    madeButtonWordBottom = 6
    // the columns of the face the colors are taken from, left of the word
    madeButtonFaceLeft = 1
    madeButtonFaceColumns = 3
    // the word has its middle in this column and starts in this row
    madeButtonMiddle = 13
    madeButtonTextY = 1
)

// palette indexes: the letters, their soft edge, the light edge right of and below them, and the
// letters of a button that can not be used
const (
    madeButtonLetters = 187
    madeButtonSoft = 182
    madeButtonLight = 177
    madeButtonLettersDisabled = 183
)

const madeButtonFontIndex = 0

type madeButton struct {
    // not pressed, pressed
    Frames []*ebiten.Image
    Disabled *ebiten.Image
}

func modernButtons() bool {
    return ModernButtons && display.ModernControls()
}

// the picture without its word. from is the first column and row of the button in the picture: a
// pressed button is a pixel further right and down
func blankButton(picture *image.Paletted, from image.Point) *image.NRGBA {
    bounds := picture.Bounds()
    out := image.NewNRGBA(bounds)

    for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
        for x := bounds.Min.X; x < bounds.Max.X; x++ {
            useX := x
            column := x - bounds.Min.X - from.X
            row := y - bounds.Min.Y - from.Y
            if column >= madeButtonWordLeft && column <= madeButtonWordRight && row >= madeButtonWordTop && row <= madeButtonWordBottom {
                useX = bounds.Min.X + from.X + madeButtonFaceLeft + column % madeButtonFaceColumns
            }
            out.Set(x, y, picture.At(useX, y))
        }
    }

    return out
}

// the pictures of a button with this word, nil if they can not be made
func (combat *CombatScreen) makeWordButton(word string) *madeButton {
    made, ok := combat.madeButtons[word]
    if ok {
        return made
    }
    if combat.madeButtons == nil {
        combat.madeButtons = make(map[string]*madeButton)
    }
    // also when it fails, so it is tried once
    combat.madeButtons[word] = nil

    hud := combat.Fonts.Hud
    if hud == nil || len(hud.LbxFonts) <= madeButtonFontIndex || len(hud.Palette) < 256 {
        return nil
    }

    lbxFile, err := combat.Cache.GetLbxFile("compix.lbx")
    if err != nil {
        log.Printf("Unable to read compix.lbx for the button %v: %v", word, err)
        return nil
    }
    frames, err := lbxFile.ReadImages(madeButtonIndex)
    if err != nil || len(frames) < 2 {
        return nil
    }
    disabled, err := lbxFile.ReadImages(madeButtonDisabledIndex)
    if err != nil || len(disabled) < 1 {
        return nil
    }

    letters := func(index int) *font.StyledFont {
        colors := []color.Color{hud.Palette[madeButtonSoft], hud.Palette[index]}
        return font.MakeStyledFont(hud.LbxFonts[madeButtonFontIndex], colors, font.ShadowDown, hud.Palette[madeButtonLight])
    }

    paint := func(picture *image.Paletted, from image.Point, styled *font.StyledFont) *ebiten.Image {
        out := ebiten.NewImageFromImage(blankButton(picture, from))
        styled.Print(out, from.X + madeButtonMiddle, from.Y + madeButtonTextY, font.FontOptions{Scale: 1, Justify: font.FontJustifyCenter}, word)
        return out
    }

    made = &madeButton{
        Frames: []*ebiten.Image{
            paint(frames[0], image.Pt(0, 0), letters(madeButtonLetters)),
            paint(frames[1], image.Pt(1, 1), letters(madeButtonLetters)),
        },
        Disabled: paint(disabled[0], image.Pt(0, 0), letters(madeButtonLettersDisabled)),
    }
    combat.madeButtons[word] = made
    return made
}

// the button END: the turn of all units of the player that still have it is over
func (combat *CombatScreen) endPlayerTurn() {
    model := combat.Model
    selected := model.SelectedUnit
    if selected == nil || model.IsAIControlled(selected) {
        return
    }

    for _, unit := range model.GetArmy(selected).GetUnits() {
        if unit.Team == selected.Team && unit.LastTurn < model.CurrentTurn {
            unit.LastTurn = model.CurrentTurn
        }
    }

    model.NextUnit()
}
