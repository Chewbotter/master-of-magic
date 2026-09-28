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

    start := leafStrips(width, 0)
    if len(start) != width {
        test.Fatalf("%v strips at the start", len(start))
    }
    for index, strip := range start {
        if strip.Column != index || strip.Source != index || strip.Back || strip.Lift != 0 || strip.Shade != 1 || strip.Rim {
            test.Fatalf("strip %v at the start: %+v", index, strip)
        }
    }

    end := leafStrips(width, 1)
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

// in between the leaf is lifted, not wider than a page, and has no two strips in one column
func TestLeafTurns(test *testing.T) {
    width := 130

    highest := 0
    for step := 1; step < 40; step++ {
        turned := float64(step) / 40
        strips := leafStrips(width, turned)
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
    early := leafStrips(width, 0.1)
    for _, strip := range early {
        if strip.Column < 0 {
            test.Errorf("on the left early in the turn: %+v", strip)
            break
        }
    }
    late := leafStrips(width, 0.9)
    for _, strip := range late {
        if strip.Column >= 0 {
            test.Errorf("on the right late in the turn: %+v", strip)
            break
        }
    }
}
