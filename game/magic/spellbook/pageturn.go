package spellbook

// The turn of a page of the spellbook.
//
// The original has 4 pictures of the leaf that turns. The fork draws the letters of the page on
// the leaf, bent to fit it. Where the leaf does not cover the page it comes down on, along its
// outer rim and under its curl, that page is seen: the old one, until the turn was over and the
// new one was there at once. Most of all its last row was seen to come up out of nothing.
//
// While the leaf comes down the page under it goes over into the new one.

// how far a turn has come, 0 to 1. ticks since it started, and the ticks it takes
func pageTurnPart(ticks uint64, all uint64) float64 {
    if all == 0 {
        return 1
    }
    return min(1, float64(ticks) / float64(all))
}

// the leaf comes down in the second half of the turn
const pageComesUpFrom = 0.5

// how much of the new page shows on the side the leaf comes down on, 0 to 1
func pageComesUp(turned float64) float32 {
    if turned <= pageComesUpFrom {
        return 0
    }

    part := (turned - pageComesUpFrom) / (1 - pageComesUpFrom)
    // slowly at first and at the end
    return float32(part * part * (3 - 2 * part))
}
