package combat

import (
    "testing"
)

// one white pixel in the middle: light around it as far as the reach, in steps, none further
func TestHaloOfOnePixel(test *testing.T) {
    size := 3
    source := make([]byte, 4 * size * size)
    middle := (1 * size + 1) * 4
    source[middle] = 255
    source[middle + 1] = 128
    source[middle + 2] = 0
    source[middle + 3] = 255

    reach := 4
    bands := 3
    halo, width, height := makeHalo(source, size, size, reach, bands)
    if width != size + reach * 2 || height != size + reach * 2 {
        test.Fatalf("halo is %v by %v", width, height)
    }

    alphaAt := func(x int, y int) int {
        return int(halo[(y * width + x) * 4 + 3])
    }

    centerX := 1 + reach
    centerY := 1 + reach
    if alphaAt(centerX, centerY) == 0 {
        test.Errorf("no light where the pixel is")
    }
    if alphaAt(centerX + reach, centerY) == 0 {
        test.Errorf("no light %v from the pixel", reach)
    }
    if alphaAt(centerX + reach + 1, centerY) != 0 {
        test.Errorf("light further than the reach")
    }
    if alphaAt(0, 0) != 0 {
        test.Errorf("light in the corner")
    }

    // only as many kinds of brightness as there are steps
    kinds := make(map[int]bool)
    for index := 3; index < len(halo); index += 4 {
        if halo[index] != 0 {
            kinds[int(halo[index])] = true
        }
    }
    if len(kinds) == 0 || len(kinds) > bands {
        test.Errorf("%v kinds of brightness, %v steps", len(kinds), bands)
    }

    // the color of the pixel: red, half as much green, no blue
    at := (centerY * width + centerX) * 4
    if halo[at] == 0 || halo[at + 2] != 0 || halo[at + 1] * 2 > halo[at] + 2 || halo[at + 1] * 2 + 2 < halo[at] {
        test.Errorf("color %v %v %v", halo[at], halo[at + 1], halo[at + 2])
    }
}

// black pixels give no light
func TestHaloOfBlack(test *testing.T) {
    size := 3
    source := make([]byte, 4 * size * size)
    for index := 3; index < len(source); index += 4 {
        source[index] = 255
    }

    halo, _, _ := makeHalo(source, size, size, 4, 3)
    for index, value := range halo {
        if value != 0 {
            test.Fatalf("light at %v", index)
        }
    }
}
