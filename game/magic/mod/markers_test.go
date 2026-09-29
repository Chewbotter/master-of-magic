package mod

import (
    "image"
    "image/color"
    "testing"
)

// a tab as the game has them: a light rim, a dark fold, the light face, dark folds at its right
// and below, and light pixels of the page outside of the folds
func TestTabFace(test *testing.T) {
    rows := []string{
        "....rdfd",
        "...rdffd",
        "..rdfffd",
        ".rdffffd",
        "rdfffffd",
        "dddddddd",
        "L.......",
    }

    shades := map[rune]uint8{'r': 104, 'd': 72, 'f': 124, 'L': 124}

    tab := image.NewNRGBA(image.Rect(0, 0, len(rows[0]), len(rows)))
    for y, row := range rows {
        for x, pixel := range row {
            if pixel == '.' {
                continue
            }
            shade := shades[pixel]
            tab.Set(x, y, color.NRGBA{R: shade, G: shade, B: shade, A: 255})
        }
    }

    face := TabFace(tab)
    for y, row := range rows {
        for x, pixel := range row {
            is := face[y * len(rows[0]) + x]
            if is != (pixel == 'f') {
                test.Errorf("pixel %v, %v (%c): a part of the face %v", x, y, pixel, is)
            }
        }
    }

    hover := MakeTabHover(tab)
    fold := hover.At(0, 5)
    if fold != tab.At(0, 5) {
        test.Errorf("a fold is %v under the mouse, it is %v", fold, tab.At(0, 5))
    }
    if lightOf(hover.At(4, 3)) <= lightOf(tab.At(4, 3)) {
        test.Errorf("the face is not lighter under the mouse")
    }
}
