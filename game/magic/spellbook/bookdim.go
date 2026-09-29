package spellbook

// The dark behind the spellbook for casting.
//
// Upstream drew it at full strength from the first frame to the last, so it popped in and out
// while the book faded. It now has the alpha of the book. The book only draws on the picture of
// the game, 320 art pixels wide: a screen that shows more to the left and right of it, as the
// battlefield in widescreen does, asks how dark it is and draws the dark there itself.

// how dark it is behind the book when the book is all there, 0 to 1
const CastBookDark = 0.5

var castBookDark float32

// called by the book every frame it is drawn. gives how dark it is behind it
func noteCastBookDark(alpha float32) float32 {
    castBookDark = CastBookDark * alpha
    return castBookDark
}

// how dark it is behind the book that was drawn last, 0 when none was drawn since this was
// asked. for a screen that draws once a frame
func TakeCastBookDark() float32 {
    dark := castBookDark
    castBookDark = 0
    return dark
}
