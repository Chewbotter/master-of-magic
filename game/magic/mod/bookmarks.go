package mod

// The bookmarks of the spellbook for casting: ribbons that hang out of the pages at the right
// edge of the book, one for every kind of spell, with a symbol of the kind. Not in the original
// (user, 2026-09-29). See game/magic/spellbook/bookmark.go.
//
// The pictures are made here, a first pass the user repaints: a red ribbon as the one with the X
// under the book, with a forked end, and a symbol in gold. A picture
// `markers/bookmark <kind>.png` in the replacement folder takes the place of one, as it is. It
// should have the size of the one the game makes: the left part of a ribbon is under the pages,
// more of it the further down its kind is in the book, so what should always show is in the
// right part (from column BookmarkAlwaysShown on).

import (
    "image"
    "image/color"
)

const BookmarkWidth = 28
const BookmarkHeight = 11
// the columns from this one on show for every bookmark
const BookmarkAlwaysShown = 15

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

func BookmarkName(kind string) string {
    return "bookmark " + kind
}

// how deep the fork of the end of a ribbon is cut, in its middle row
const bookmarkFork = 3
// where the symbol is on the ribbon
// a column right of where the pages end on the lowest ribbon
const bookmarkSymbolX = 16
const bookmarkSymbolY = 2

var bookmarkOutline = color.NRGBA{R: 40, G: 8, B: 8, A: 255}
var bookmarkLight = color.NRGBA{R: 228, G: 52, B: 44, A: 255}
var bookmarkCloth = color.NRGBA{R: 192, G: 24, B: 24, A: 255}
var bookmarkDark = color.NRGBA{R: 136, G: 12, B: 12, A: 255}
var bookmarkSymbol = color.NRGBA{R: 252, G: 204, B: 68, A: 255}
var bookmarkSymbolLight = color.NRGBA{R: 255, G: 240, B: 140, A: 255}
var bookmarkSymbolShadow = color.NRGBA{R: 92, G: 8, B: 8, A: 255}

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
                shade := bookmarkCloth
                if y == 1 {
                    shade = bookmarkLight
                } else if y >= BookmarkHeight - 3 {
                    shade = bookmarkDark
                }
                out.SetNRGBA(x, y, shade)
            } else if cloth(x, y - 1) || cloth(x, y + 1) || cloth(x - 1, y) || cloth(x + 1, y) {
                out.SetNRGBA(x, y, bookmarkOutline)
            }
        }
    }

    rows := bookmarkSymbols[kind]
    // its shadow right of and below it, then the symbol, its top row lighter
    for pass := range 2 {
        for y, row := range rows {
            for x, pixel := range row {
                if pixel != 'X' {
                    continue
                }
                atX := bookmarkSymbolX + x
                atY := bookmarkSymbolY + y
                if pass == 0 {
                    if cloth(atX + 1, atY + 1) {
                        out.SetNRGBA(atX + 1, atY + 1, bookmarkSymbolShadow)
                    }
                } else if cloth(atX, atY) {
                    shade := bookmarkSymbol
                    if y == 0 || (y > 0 && rows[y - 1][x] != 'X') {
                        shade = bookmarkSymbolLight
                    }
                    out.SetNRGBA(atX, atY, shade)
                }
            }
        }
    }

    return out
}

// the bookmark of a kind, the one of the replacement folder if there is one
func Bookmark(kind string) image.Image {
    replacement := ReadMarker(BookmarkName(kind))
    if replacement != nil {
        return replacement
    }

    return DefaultBookmark(kind)
}
