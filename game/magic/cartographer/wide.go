package cartographer

// Widescreen: the game draws the cartographer's map in the middle of the wide screen, so the cursor
// has to be moved into the map's own coordinates. Set while the cartographer is open, nil otherwise.
var CursorTransform func(x int, y int) (int, int)

func transformCursor(x int, y int) (int, int) {
    if CursorTransform != nil {
        return CursorTransform(x, y)
    }
    return x, y
}
