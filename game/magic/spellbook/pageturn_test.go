package spellbook

import (
    "testing"
)

func TestPageComesUp(test *testing.T) {
    if pageComesUp(0) != 0 || pageComesUp(0.5) != 0 {
        test.Errorf("in the first half of the turn: %v %v", pageComesUp(0), pageComesUp(0.5))
    }
    if pageComesUp(1) != 1 {
        test.Errorf("at the end: %v", pageComesUp(1))
    }

    if pageTurnPart(12, 24) != 0.5 || pageTurnPart(30, 24) != 1 || pageTurnPart(3, 0) != 1 {
        test.Errorf("how far a turn has come")
    }
}

// before the turn the leaf is the right page as it lies, after it the left page as it lies
func TestLeafLiesFlat(test *testing.T) {
    width := 130

    start := leafStrips(width, 0, false)
    if len(start) != width {
        test.Fatalf("%v strips at the start", len(start))
    }
    for index, strip := range start {
        if strip.Column != index || strip.Source != index || strip.Back || strip.Lift != 0 || strip.Shade != 1 || strip.Rim {
            test.Fatalf("strip %v at the start: %+v", index, strip)
        }
    }

    end := leafStrips(width, 1, false)
    if len(end) != width {
        test.Fatalf("%v strips at the end", len(end))
    }
    for index, strip := range end {
        // from the rim of the left page to the spine
        column := index - width
        source := width - 1 - index
        if strip.Column != column || strip.Source != source || !strip.Back || strip.Lift != 0 || strip.Shade != 1 || strip.Rim {
            test.Fatalf("strip %v at the end: %+v, should be in column %v from %v", index, strip, column, source)
        }
    }
}

// a leaf that is turned back is the one that is turned forward in a mirror
func TestLeafMirrored(test *testing.T) {
    width := 130

    // before it is turned back it lies on the left, after it on the right
    start := leafStrips(width, 0, true)
    end := leafStrips(width, 1, false)
    if len(start) != width || len(end) != width {
        test.Fatalf("%v and %v strips", len(start), len(end))
    }
    for index := range start {
        if start[index] != end[index] {
            test.Fatalf("strip %v: %+v before it is turned back, %+v after it was turned forward", index, start[index], end[index])
        }
    }

    done := leafStrips(width, 1, true)
    flat := leafStrips(width, 0, false)
    for index := range done {
        if done[index] != flat[index] {
            test.Fatalf("strip %v: %+v after it was turned back, %+v before it is turned forward", index, done[index], flat[index])
        }
    }

    for step := 1; step < 20; step++ {
        turned := float64(step) / 20
        forward := leafStrips(width, turned, false)
        backward := leafStrips(width, turned, true)
        if len(forward) != len(backward) {
            test.Fatalf("%v and %v strips at %v", len(forward), len(backward), turned)
        }

        // the same strips, from the other end
        for index, strip := range forward {
            other := backward[len(backward) - 1 - index]
            if other.Column != -1 - strip.Column || other.Source != strip.Source || other.Back == strip.Back || other.Lift != strip.Lift {
                test.Fatalf("at %v strip %+v is %+v in the mirror", turned, strip, other)
            }
        }
    }
}

// in between the leaf is lifted, not wider than a page, and has no two strips in one column
func TestLeafTurns(test *testing.T) {
    width := 130

    highest := 0
    for step := 1; step < 40; step++ {
        turned := float64(step) / 40
        strips := leafStrips(width, turned, step % 2 == 0)
        if len(strips) == 0 {
            test.Fatalf("no leaf at %v", turned)
        }

        seen := make(map[int]bool)
        rims := 0
        for _, strip := range strips {
            if seen[strip.Column] {
                test.Fatalf("two strips in column %v at %v", strip.Column, turned)
            }
            seen[strip.Column] = true

            if strip.Column < -width || strip.Column >= width {
                test.Errorf("column %v at %v", strip.Column, turned)
            }
            if strip.Source < 0 || strip.Source >= width {
                test.Errorf("from column %v at %v", strip.Source, turned)
            }
            if strip.Lift < 0 {
                test.Errorf("lift %v at %v", strip.Lift, turned)
            }
            if strip.Shade <= 0 || strip.Shade > 1 {
                test.Errorf("shade %v at %v", strip.Shade, turned)
            }
            if strip.Rim {
                rims += 1
            }
            highest = max(highest, strip.Lift)
        }

        if rims != 1 {
            test.Errorf("%v rims at %v", rims, turned)
        }
    }

    if highest < 20 {
        test.Errorf("the leaf is lifted by %v at most", highest)
    }

    // early in the turn the leaf is still on the right, late on the left
    early := leafStrips(width, 0.1, false)
    for _, strip := range early {
        if strip.Column < 0 {
            test.Errorf("on the left early in the turn: %+v", strip)
            break
        }
    }
    late := leafStrips(width, 0.9, false)
    for _, strip := range late {
        if strip.Column >= 0 {
            test.Errorf("on the right late in the turn: %+v", strip)
            break
        }
    }
}

// the ribbon is red with a yellow X, what is around it in the picture of the book is brown
func TestRibbonColor(test *testing.T) {
    for _, each := range [][3]byte{{255, 0, 0}, {200, 20, 20}, {130, 10, 10}, {250, 170, 40}} {
        if !ribbonColor(each[0], each[1], each[2]) {
            test.Errorf("%v is a color of the ribbon", each)
        }
    }
    for _, each := range [][3]byte{{130, 90, 45}, {90, 60, 30}, {60, 40, 20}, {0, 0, 0}, {200, 200, 200}} {
        if ribbonColor(each[0], each[1], each[2]) {
            test.Errorf("%v is no color of the ribbon", each)
        }
    }
}
