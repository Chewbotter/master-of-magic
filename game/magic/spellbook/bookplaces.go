package spellbook

// Where things are in the spellbook for casting, in art pixels. Measured on the picture of the
// book (spells.lbx 0) and taken from where MakeSpellBookCastUI draws.

import (
    "image"
)

// where the picture of the book is drawn on the screen
const castBookX = 10
const castBookY = 10

// the two pages in the picture of the book. the column between them is the spine
var castBookLeftPage = image.Rect(12, 3, 142, 153)
var castBookRightPage = image.Rect(143, 3, 273, 153)

// where the letters of a page start, in the picture of the book
var castBookLeftContent = image.Pt(15, 5)
var castBookRightContent = image.Pt(149, 5)

// the spellbook of research: its picture (scroll.lbx 6) fills the screen
var researchBookLeftPage = image.Rect(14, 7, 158, 178)
var researchBookRightPage = image.Rect(161, 7, 305, 178)
var researchBookLeftContent = image.Pt(0, 0)
var researchBookRightContent = image.Pt(148, 0)

func researchLeaf() pageLeaf {
    return pageLeaf{
        RightX: researchBookRightPage.Min.X,
        LeftX: researchBookLeftPage.Max.X - 1,
        Top: researchBookRightPage.Min.Y,
        Width: researchBookRightPage.Dx(),
        Height: researchBookRightPage.Dy(),
    }
}

// the ticks a turn takes: the ones of the leaf the game makes, or the ones the old turn has
func castTurnTicks(old uint64) uint64 {
    if ProceduralPageTurn {
        return PageTurnTicks
    }
    return old
}
