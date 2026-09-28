package combat

import (
    "testing"
)

// a block of 3 by 3 in a picture of 5 by 5: the rim is the side that faces the light, and the
// middle of the block is never a part of it
func TestRimPixels(test *testing.T) {
    size := 5
    source := make([]byte, 4 * size * size)
    for y := 1; y <= 3; y++ {
        for x := 1; x <= 3; x++ {
            source[(y * size + x) * 4 + 3] = 255
        }
    }

    lit := func(pixels []byte, x int, y int) bool {
        return pixels[(y * size + x) * 4 + 3] != 0
    }

    // the light to the right: the right column of the block
    right := makeRimPixels(source, size, size, 0)
    for y := range size {
        for x := range size {
            expected := x == 3 && y >= 1 && y <= 3
            if lit(right, x, y) != expected {
                test.Errorf("light from the right, %v,%v: %v", x, y, lit(right, x, y))
            }
        }
    }

    // from above: the upper row
    above := makeRimPixels(source, size, size, 6)
    for y := range size {
        for x := range size {
            expected := y == 1 && x >= 1 && x <= 3
            if lit(above, x, y) != expected {
                test.Errorf("light from above, %v,%v: %v", x, y, lit(above, x, y))
            }
        }
    }

    // from the upper right: the upper row and the right column
    corner := makeRimPixels(source, size, size, 7)
    for y := range size {
        for x := range size {
            expected := x >= 1 && x <= 3 && y >= 1 && y <= 3 && (y == 1 || x == 3)
            if lit(corner, x, y) != expected {
                test.Errorf("light from the upper right, %v,%v: %v", x, y, lit(corner, x, y))
            }
        }
    }

    for side := range rimSides {
        rim := makeRimPixels(source, size, size, side)
        if lit(rim, 2, 2) {
            test.Errorf("side %v: the middle of the figure is lit", side)
        }
        for y := range size {
            for x := range size {
                if lit(rim, x, y) && source[(y * size + x) * 4 + 3] == 0 {
                    test.Errorf("side %v: light at %v,%v, where the figure is not", side, x, y)
                }
            }
        }
    }
}

// a rim comes up fast and goes out a little slower, and neither at once
func TestRimStep(test *testing.T) {
    now := 0.0
    first := rimStep(now, 0.65)
    if first <= 0 || first >= 0.65 {
        test.Errorf("after one tick of coming up: %v", first)
    }

    for range 60 {
        now = rimStep(now, 0.65)
    }
    if now < 0.64 {
        test.Errorf("after a second of coming up: %v", now)
    }

    // the light is gone at once
    after := rimStep(now, 0)
    if after >= now || after < now * 0.5 {
        test.Errorf("one tick after the light is gone: %v of %v", after, now)
    }

    ticks := 0
    for now >= rimGone && ticks < 600 {
        now = rimStep(now, 0)
        ticks += 1
    }
    // gone within half a second, not within a tenth
    if ticks < 6 || ticks > 30 {
        test.Errorf("the rim is gone after %v ticks", ticks)
    }
}

func TestRimSide(test *testing.T) {
    for _, each := range []struct{X, Y float64; Side int}{
        {10, 0, 0}, {10, 10, 1}, {0, 10, 2}, {-10, 10, 3}, {-10, 0, 4}, {-10, -10, 5}, {0, -10, 6}, {10, -10, 7},
        {10, 1, 0}, {10, -1, 0},
    } {
        if side := rimSide(each.X, each.Y); side != each.Side {
            test.Errorf("toward %v,%v: side %v, should be %v", each.X, each.Y, side, each.Side)
        }
    }
}

// all of it near the light, less toward the rim of the light, none beyond it
func TestRimLightAt(test *testing.T) {
    orange := [3]float32{1, 0.7, 0.4}
    lights := []spellLightSource{{X: 100, Y: 100, Radius: 60, Strength: 1, Rim: 0.9, RimColor: orange}}

    near, side, tint := rimLightAt(lights, 80, 100)
    if near != 0.9 || side != 0 || tint != orange {
        test.Errorf("20 to the left of the light: %v from side %v in %v", near, side, tint)
    }

    far, _, _ := rimLightAt(lights, 45, 100)
    if far <= 0 || far >= near {
        test.Errorf("55 to the left of the light: %v", far)
    }

    none, _, _ := rimLightAt(lights, 30, 100)
    if none != 0 {
        test.Errorf("70 to the left of the light: %v", none)
    }

    // up and down the screen the light goes half as far
    none, _, _ = rimLightAt(lights, 100, 135)
    if none != 0 {
        test.Errorf("35 under the light: %v", none)
    }

    off, _, _ := rimLightAt([]spellLightSource{{X: 100, Y: 100, Radius: 60, Strength: 1, Rim: 0}}, 80, 100)
    if off != 0 {
        test.Errorf("a spell without rim light: %v", off)
    }
}
