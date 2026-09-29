package game

import (
    "image"
    "testing"
)

// the camera goes to the tile before a lair, and stays where it is when the stack stands next to
// the lair
func TestTileBefore(test *testing.T) {
    path := worldPath{image.Pt(5, 5), image.Pt(6, 5), image.Pt(7, 6), image.Pt(8, 6)}

    if x, y, found := tileBefore(path, 5, 5, 8, 6); !found || x != 7 || y != 6 {
        test.Fatalf("before the lair: %v, %v, %v", x, y, found)
    }
    // the path can start with the tile the stack stands on, or with the one after it
    if _, _, found := tileBefore(path, 5, 5, 6, 5); found {
        test.Fatalf("the stack stands next to the lair")
    }
    if _, _, found := tileBefore(path[1:], 5, 5, 6, 5); found {
        test.Fatalf("the stack stands next to the lair, the path starts after it")
    }
    if _, _, found := tileBefore(path, 5, 5, 20, 20); found {
        test.Fatalf("a tile that is not on the path")
    }
}
