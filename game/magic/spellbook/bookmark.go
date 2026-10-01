package spellbook

// The bookmarks of the book for casting: ribbons at the right edge of the book, one for every
// kind of spell the book has pages of, in the order of the book. A click on one turns the book
// to the first page of its kind. Not in the original (user, 2026-09-29).
//
// A ribbon comes out from under the pages, and every ribbon from a page further down than the
// one above it, whose edge is a pixel further right in the picture of the book. All are as long
// (user, 2026-09-29: "I like the tabs being one row of pixels back for each, but keep each about
// the same horizontal length"; at first the lower ones were further in the book and shorter).
// The ribbon of the kind the book is open at is pulled out a little. The pictures are made by
// game/magic/mod/bookmarks.go and can be repainted.

import (
    "image"

    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: no bookmarks
const Bookmarks = true

// the kinds of spells in the order of the book, as computeHalfPages has them, with the names of
// their pictures
var bookmarkSections = []struct{
    Section Section
    Kind string
}{
    {SectionSummoning, BookmarkSummoning},
    {SectionSpecial, BookmarkSpecial},
    {SectionCitySpell, BookmarkCity},
    {SectionEnchantment, BookmarkEnchantment},
    {SectionUnitSpell, BookmarkUnit},
    {SectionCombatSpell, BookmarkCombat},
}

// places in the picture of the book, in art pixels
const (
    // the edge of the top page: the first ribbon comes out here, the next ones a pixel further
    // right each, at the edges of the pages under it
    bookmarkPageEdge = 272
    bookmarkEdgeStep = 1
    // the top of the first ribbon and from one to the next. between the golden corners of the cover
    bookmarkTop = 38
    bookmarkDown = 18
    // the ribbon of the kind the book is open at is this much further out
    bookmarkPulled = 2
    // a book with fewer kinds of spells has no bookmarks
    bookmarkKindsNeeded = 2
)
// how much lighter a ribbon is under the mouse
const bookmarkLight = pageTabLight

var bookmarkPictures = make(map[string]*ebiten.Image)

// the pictures of the replacement folder are read when the book opens
func forgetBookmarks() {
    for kind, picture := range bookmarkPictures {
        picture.Deallocate()
        delete(bookmarkPictures, kind)
    }
}

func bookmarkPicture(kind string) *ebiten.Image {
    picture, ok := bookmarkPictures[kind]
    if !ok {
        picture = ebiten.NewImageFromImage(Bookmark(kind))
        bookmarkPictures[kind] = picture
    }
    return picture
}

// the first page of a kind of spells in the pages of the book, -1 if it has none
func bookmarkPage(pages []Page, section Section) int {
    for index, page := range pages {
        if page.Title == section.Name() {
            return index
        }
    }
    return -1
}

// true if the pages have enough kinds of spells for bookmarks
func bookmarksShown(pages []Page) bool {
    if !Bookmarks {
        return false
    }

    kinds := 0
    for _, entry := range bookmarkSections {
        if bookmarkPage(pages, entry.Section) >= 0 {
            kinds += 1
        }
    }
    return kinds >= bookmarkKindsNeeded
}

// the kind of spells the book is open at: the one of its right page, else of its left
func bookmarkOpenAt(pages []Page, current int) int {
    for _, page := range []int{current + 1, current} {
        if page < 0 || page >= len(pages) {
            continue
        }
        for slot, entry := range bookmarkSections {
            if pages[page].Title == entry.Section.Name() {
                return slot
            }
        }
    }
    return -1
}

// where the ribbon of a slot is, on the screen in art pixels
func bookmarkShown(slot int, pulled bool, width int, height int) image.Rectangle {
    left := castBookX + bookmarkPageEdge + slot * bookmarkEdgeStep
    top := castBookY + bookmarkTop + slot * bookmarkDown
    if pulled {
        // more of it is out of the book: it is longer by that, from the pages on
        width += bookmarkPulled
    }

    return image.Rect(left, top, left + width, top + height)
}

func drawBookmark(screen *ebiten.Image, slot int, pulled bool, over bool, alpha float32) {
    picture := bookmarkPicture(bookmarkSections[slot].Kind)
    bounds := picture.Bounds()
    shown := bookmarkShown(slot, false, bounds.Dx(), bounds.Dy())

    var options ebiten.DrawImageOptions
    if over {
        options.ColorScale.Scale(1 + bookmarkLight, 1 + bookmarkLight, 1 + bookmarkLight, 1)
    }
    options.ColorScale.ScaleAlpha(alpha)

    if pulled {
        // its first column again for what has come out of the book
        first := picture.SubImage(image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Min.X + 1, bounds.Max.Y)).(*ebiten.Image)
        for column := range bookmarkPulled {
            more := options
            more.GeoM.Translate(float64(shown.Min.X + column), float64(shown.Min.Y))
            scale.DrawScaled(screen, first, &more)
        }
        options.GeoM.Translate(bookmarkPulled, 0)
    }

    options.GeoM.Translate(float64(shown.Min.X), float64(shown.Min.Y))
    scale.DrawScaled(screen, picture, &options)
}

// all bookmarks of the book. over is the slot under the mouse, -1 for none
func drawBookmarks(screen *ebiten.Image, pages []Page, current int, over int, alpha float32) {
    if !bookmarksShown(pages) {
        return
    }

    open := bookmarkOpenAt(pages, current)
    for slot, entry := range bookmarkSections {
        if bookmarkPage(pages, entry.Section) < 0 {
            continue
        }
        drawBookmark(screen, slot, slot == open, slot == over, alpha)
    }
}
