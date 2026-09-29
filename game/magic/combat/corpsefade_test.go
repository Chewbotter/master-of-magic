package combat

import (
    "image"
    "testing"
)

// a tile keeps its newest corpses, the oldest of a tile that has too many fade
func TestCorpsesToFade(test *testing.T) {
    here := image.Pt(10, 10)
    there := image.Pt(11, 10)

    // 5 on one tile, 2 on another, in the order they fell
    tiles := []image.Point{here, there, here, here, there, here, here}
    fading := make([]bool, len(tiles))

    if got := corpsesToFade(tiles, fading, 6); len(got) != 0 {
        test.Errorf("a tile that keeps 6 has 5: %v fade", got)
    }

    got := corpsesToFade(tiles, fading, 3)
    // the two oldest of the 5
    if len(got) != 2 || got[0] != 0 || got[1] != 2 {
        test.Errorf("a tile that keeps 3 has 5: %v fade, should be 0 and 2", got)
    }

    // the ones that fade already do not count, and do not start again
    fading[0] = true
    fading[2] = true
    if got := corpsesToFade(tiles, fading, 3); len(got) != 0 {
        test.Errorf("two fade already: %v more", got)
    }

    // one more falls
    tiles = append(tiles, here)
    fading = append(fading, false)
    got = corpsesToFade(tiles, fading, 3)
    if len(got) != 1 || got[0] != 3 {
        test.Errorf("one more falls: %v fade, should be 3", got)
    }
}

// the corpse of a unit that is one figure stays
func TestCorpseLasts(test *testing.T) {
    if !corpseLasts(1) {
        test.Errorf("the corpse of a unit of one figure fades")
    }
    for _, figures := range []int{2, 4, 6, 8} {
        if corpseLasts(figures) {
            test.Errorf("the corpse of a figure of a unit of %v never fades", figures)
        }
    }
}

// a corpse that fades shows less and less and is gone
func TestCorpseShows(test *testing.T) {
    body := corpse{}
    if corpseShows(&body, 1000) != 1 {
        test.Errorf("a corpse that does not fade shows %v", corpseShows(&body, 1000))
    }

    body.Fading = true
    body.FadeStart = 100
    if corpseShows(&body, 100) != 1 {
        test.Errorf("as it starts to fade it shows %v", corpseShows(&body, 100))
    }

    last := 1.0
    for counter := uint64(100); counter < 100 + 60 * 4; counter += 10 {
        shows := corpseShows(&body, counter)
        if shows > last {
            test.Errorf("it shows more at %v", counter)
        }
        last = shows
    }
    if last != 0 {
        test.Errorf("after its time it shows %v", last)
    }
}
