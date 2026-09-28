package combat

import (
    "testing"
)

// the chevron is the same on both sides, its tip is in the middle of the lowest row of its own
// color, and it has an edge all around
func TestChevronPixels(test *testing.T) {
    pixels, width, height := makeChevronPixels()

    kind := func(x int, y int) int {
        at := (y * width + x) * 4
        switch {
            case pixels[at + 3] == 0: return 0
            case pixels[at] == chevronFill[0]: return 2
        }
        return 1
    }

    for y := range height {
        for x := range width {
            if kind(x, y) != kind(width - 1 - x, y) {
                test.Fatalf("not the same on both sides at %v,%v", x, y)
            }
            // nothing of the chevron on the rim of the picture: the edge has room
            if kind(x, y) == 2 && (x == 0 || y == 0 || x == width - 1 || y == height - 1) {
                test.Errorf("the chevron touches the rim at %v,%v", x, y)
            }
        }
    }

    lowest := height - 2
    for x := range width {
        expected := 1
        if x == width / 2 {
            expected = 2
        }
        if x < width / 2 - 1 || x > width / 2 + 1 {
            expected = kind(x, lowest)
        }
        if kind(x, lowest) != expected {
            test.Errorf("lowest row at %v is %v", x, kind(x, lowest))
        }
    }
}

// it goes up and down by whole pixels and no further than it should
func TestChevronLift(test *testing.T) {
    seen := make(map[int]bool)
    for counter := range uint64(240) {
        lift := chevronLift(counter, 60)
        if lift < 0 || float64(lift) > chevronBounce + 0.5 {
            test.Fatalf("lift %v at %v", lift, counter)
        }
        seen[lift] = true
    }
    if len(seen) < 2 {
        test.Errorf("the chevron does not move")
    }
}
