package spellbook

// The bookmarks of the spellbook for casting: ribbons that hang out of the pages at the right
// edge of the book, one for every kind of spell, with a symbol of the kind. Not in the original
// (user, 2026-09-29). See bookmark.go.
//
// The pictures are made here, as the ribbon with the X under the book is painted (spells.lbx 0):
// its three reds, the darker ones where the cloth comes out of the book and before its forked end,
// with a column of both colors in turn (dither) between two reds, and the symbol in the gold of
// the X with its orange on the edges that look up and left.

import (
    "image"
    "image/color"
)

const BookmarkWidth = 24
const BookmarkHeight = 11

// the kinds, in the order of the book
const (
    BookmarkSummoning = "summoning"
    BookmarkSpecial = "special"
    BookmarkCity = "city"
    BookmarkEnchantment = "enchantment"
    BookmarkUnit = "unit"
    BookmarkCombat = "combat"
)

var BookmarkKinds = []string{BookmarkSummoning, BookmarkSpecial, BookmarkCity, BookmarkEnchantment, BookmarkUnit, BookmarkCombat}

// how deep the fork of the end of a ribbon is cut, in its middle row
const bookmarkFork = 3
// where the symbol is on the ribbon
const bookmarkSymbolX = 11
const bookmarkSymbolY = 2

// the colors of the ribbon with the X, read from its picture
var bookmarkOutline = color.NRGBA{R: 8, G: 4, B: 4, A: 255}
var bookmarkCloth = color.NRGBA{R: 212, G: 0, B: 0, A: 255}
var bookmarkShade = color.NRGBA{R: 180, G: 0, B: 0, A: 255}
var bookmarkDark = color.NRGBA{R: 152, G: 0, B: 0, A: 255}
var bookmarkSymbol = color.NRGBA{R: 236, G: 164, B: 36, A: 255}
var bookmarkSymbolEdge = color.NRGBA{R: 252, G: 120, B: 16, A: 255}

// the reds along the ribbon, from where it comes out of the book: a letter per column.
// d dark, s shade, c cloth; D and S are a column of two of them in turn, dark with shade and
// shade with cloth
const bookmarkColumns = "dDsSccccccccccccccccSsSc"
// the lowest row of the cloth is a red darker than its column
const bookmarkLowRow = BookmarkHeight - 2

// 7 by 7, X is the symbol
var bookmarkSymbols = map[string][]string{
    // a star
    BookmarkSummoning: {
        "...X...",
        "...X...",
        "XXXXXXX",
        ".XXXXX.",
        "..XXX..",
        ".XX.XX.",
        "XX...XX",
    },
    // a spark
    BookmarkSpecial: {
        "...X...",
        "...X...",
        "..XXX..",
        "XXXXXXX",
        "..XXX..",
        "...X...",
        "...X...",
    },
    // a gate between towers
    BookmarkCity: {
        "X.X.X.X",
        "XXXXXXX",
        ".XXXXX.",
        ".XX.XX.",
        ".X...X.",
        ".X...X.",
        "XX...XX",
    },
    // a moon
    BookmarkEnchantment: {
        "..XXX..",
        ".XXX...",
        "XXX....",
        "XXX....",
        "XXX...X",
        ".XXXXX.",
        "..XXX..",
    },
    // a shield
    BookmarkUnit: {
        "XXXXXXX",
        "XXX.XXX",
        "XXX.XXX",
        "X.....X",
        ".XX.XX.",
        "..X.X..",
        "...X...",
    },
    // a sword
    BookmarkCombat: {
        ".....XX",
        "....XXX",
        "...XXX.",
        "X.XXX..",
        ".XXX...",
        "XXX....",
        "X..X...",
    },
}

// the red of the cloth at a place
func bookmarkRed(x int, y int) color.NRGBA {
    reds := []color.NRGBA{bookmarkDark, bookmarkShade, bookmarkCloth}

    letter := byte('c')
    if x >= 0 && x < len(bookmarkColumns) {
        letter = bookmarkColumns[x]
    }

    shade := 2
    switch letter {
        case 'd': shade = 0
        case 's': shade = 1
        case 'D': shade = (x + y) % 2
        case 'S': shade = 1 + (x + y) % 2
    }

    if y == bookmarkLowRow {
        shade = max(0, shade - 1)
    }

    return reds[shade]
}

// the bookmark of a kind as the game makes it
func DefaultBookmark(kind string) *image.NRGBA {
    out := image.NewNRGBA(image.Rect(0, 0, BookmarkWidth, BookmarkHeight))

    // the last column of the cloth in a row: the end is forked
    end := func(y int) int {
        middle := BookmarkHeight / 2
        away := y - middle
        if away < 0 {
            away = -away
        }
        return BookmarkWidth - 2 - max(0, bookmarkFork - away)
    }

    cloth := func(x int, y int) bool {
        return y >= 1 && y <= BookmarkHeight - 2 && x >= 0 && x <= end(y)
    }

    for y := range BookmarkHeight {
        for x := range BookmarkWidth {
            if cloth(x, y) {
                out.SetNRGBA(x, y, bookmarkRed(x, y))
            } else if cloth(x, y - 1) || cloth(x, y + 1) || cloth(x - 1, y) || cloth(x + 1, y) {
                out.SetNRGBA(x, y, bookmarkOutline)
            }
        }
    }

    rows := bookmarkSymbols[kind]
    part := func(x int, y int) bool {
        return y >= 0 && y < len(rows) && x >= 0 && x < len(rows[y]) && rows[y][x] == 'X'
    }

    for y := range rows {
        for x := range rows[y] {
            if !part(x, y) || !cloth(bookmarkSymbolX + x, bookmarkSymbolY + y) {
                continue
            }
            shade := bookmarkSymbol
            // the edges that look up and left, as on the X
            if !part(x - 1, y) && !part(x, y - 1) {
                shade = bookmarkSymbolEdge
            }
            out.SetNRGBA(bookmarkSymbolX + x, bookmarkSymbolY + y, shade)
        }
    }

    return out
}

// the bookmark of a kind
func Bookmark(kind string) image.Image {
    return DefaultBookmark(kind)
}
