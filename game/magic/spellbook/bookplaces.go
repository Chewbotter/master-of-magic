package spellbook

// Where things are in the spellbook for casting, in art pixels. Measured on the picture of the
// book (spells.lbx 0) and taken from where MakeSpellBookCastUI draws.

import (
    "image"
    "image/color"

    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
)

// where the picture of the book is drawn on the screen
const castBookX = 10
const castBookY = 10

// the two pages in the picture of the book. the column between them is the spine
var castBookLeftPage = image.Rect(12, 3, 142, 153)
var castBookRightPage = image.Rect(143, 3, 273, 153)

// the cost of a spell ends here, from the left of the content of a page: where the lines of the
// page end (the picture of them is 122 wide). the original prints it ending on 121 (ReMoM
// Spellbook.c SmlBook_Compose). upstream had 124, which is the edge of the page (user,
// 2026-09-29: "a buffer before the edge of the page")
const castCostRight = 122
// on a right page a pixel less: the edge of the page is closer there (user, 2026-09-29: "looks
// right on the left page, and like it should be scooted in one more pixel on the right page")
const castCostRightPage = 121
// the lines of the page end this many pixels before the cost. the original clears them from 4
// before it (ReMoM Spellbook.c SmlBook_Compose); upstream ended them where it took the cost to
// start, which was in the cost once that was moved in
const castCostGap = 3

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

// the ribbon with the X that closes the book, in the picture of the book
var castBookRibbon = image.Rect(159, 156, 178, 186)
// pressed it is this many art pixels lower and this dark, 0 to 1
const ribbonPressedDown = 1
const ribbonPressedShade = 0.75
// how much white lies over it while the mouse is over it, 0 to 1
const ribbonOverLight = 0.3

// true for the colors of the ribbon and its X: reds, and the yellow of the X. what is around the
// ribbon in the picture of the book is brown
func ribbonColor(red byte, green byte, blue byte) bool {
    isRed := red > 90 && int(green) * 100 < int(red) * 45 && int(blue) * 100 < int(red) * 45
    isYellow := red > 180 && green > 100 && blue < 110
    return isRed || isYellow
}

// the ribbon cut out of its part of the picture of the book, and the same in white
func ribbonShapes(part *ebiten.Image) (*ebiten.Image, *ebiten.Image) {
    bounds := part.Bounds()
    pixels := make([]byte, 4 * bounds.Dx() * bounds.Dy())
    part.ReadPixels(pixels)

    white := make([]byte, len(pixels))
    for index := 0; index < len(pixels); index += 4 {
        if pixels[index + 3] == 0 || !ribbonColor(pixels[index], pixels[index + 1], pixels[index + 2]) {
            pixels[index] = 0
            pixels[index + 1] = 0
            pixels[index + 2] = 0
            pixels[index + 3] = 0
            continue
        }
        white[index] = 255
        white[index + 1] = 255
        white[index + 2] = 255
        white[index + 3] = 255
    }

    ribbon := ebiten.NewImage(bounds.Dx(), bounds.Dy())
    ribbon.WritePixels(pixels)
    shape := ebiten.NewImage(bounds.Dx(), bounds.Dy())
    shape.WritePixels(white)
    return ribbon, shape
}

// the pictures of the realms that pick which spells the book shows: in a row over the book. the
// fork had them on the upper rim of the book
const realmFilterY = 2
// the middle of the row, over the spine of the book
const realmFilterMiddle = castBookX + 142
// art pixels between two of them, with room for the border of the one that is picked
const realmFilterGap = 5
// art pixels the border is wide
const realmFilterBorder = 1
// how much of the dark field under a picture shows, 0 to 1
const realmFilterField = 0.5

// the four sides of a border around a field that is this wide, inside of the rectangle
func borderRects(around image.Rectangle, wide int) []image.Rectangle {
    return []image.Rectangle{
        image.Rect(around.Min.X, around.Min.Y, around.Max.X, around.Min.Y + wide),
        image.Rect(around.Min.X, around.Max.Y - wide, around.Max.X, around.Max.Y),
        image.Rect(around.Min.X, around.Min.Y + wide, around.Min.X + wide, around.Max.Y - wide),
        image.Rect(around.Max.X - wide, around.Min.Y + wide, around.Max.X, around.Max.Y - wide),
    }
}

// fills a rectangle of art pixels with a color. pixel is a picture of one white pixel
func fillArtRect(screen *ebiten.Image, pixel *ebiten.Image, area image.Rectangle, tint color.RGBA, alpha float32) {
    var options ebiten.DrawImageOptions
    options.GeoM.Scale(float64(area.Dx()), float64(area.Dy()))
    options.GeoM.Translate(float64(area.Min.X), float64(area.Min.Y))
    options.ColorScale.ScaleWithColor(tint)
    options.ColorScale.ScaleAlpha(alpha)
    scale.DrawScaled(screen, pixel, &options)
}

// the ticks a turn takes: the ones of the leaf the game makes, or the ones the old turn has
func castTurnTicks(old uint64) uint64 {
    if ProceduralPageTurn {
        return PageTurnTicks
    }
    return old
}
