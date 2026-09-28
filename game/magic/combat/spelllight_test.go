package combat

import (
    "image/color"
    "math"
    "testing"
)

// a light is all there in its middle, not there at its rim, and has only its steps
func TestLightPixels(test *testing.T) {
    pixels, width, height := makeLightPixels(40)
    if width != 81 || height != 41 {
        test.Fatalf("a light of 40 is %v by %v", width, height)
    }

    at := func(x int, y int) int {
        return int(pixels[(y * width + x) * 4 + 3])
    }

    if at(40, 20) != 255 {
        test.Errorf("the middle shows %v", at(40, 20))
    }
    if at(0, 20) != 0 || at(40, 0) != 0 || at(0, 0) != 0 {
        test.Errorf("the rim shows %v %v %v", at(0, 20), at(40, 0), at(0, 0))
    }

    kinds := make(map[int]bool)
    for index := 3; index < len(pixels); index += 4 {
        kinds[int(pixels[index])] = true
    }
    if len(kinds) > lightBands + 1 {
        test.Errorf("%v kinds of brightness, %v steps", len(kinds), lightBands)
    }
}

// the shadow of a unit lies away from the light, and as always far from it or without one
func TestShadowNear(test *testing.T) {
    field := MakeBattlefieldMatrix()
    unitX, unitY := field.Apply(10, 10)

    lean, length, part := shadowNear(nil, 1, 10, 10)
    if lean != shadowLean || length != shadowLength || part != 0 {
        test.Errorf("without a light: %v %v %v", lean, length, part)
    }

    // the light to the left of the unit: the shadow goes to the right
    left := []spellLightSource{{X: unitX - 20, Y: unitY, Radius: 60, Strength: 1, Reach: 100}}
    lean, length, part = shadowNear(left, 1, 10, 10)
    if part <= 0 || lean <= 0 || math.Abs(length) > spellShadowThin + 0.2 {
        test.Errorf("light to the left: lean %v length %v part %v", lean, length, part)
    }

    // the light to the right: to the left
    right := []spellLightSource{{X: unitX + 20, Y: unitY, Radius: 60, Strength: 1, Reach: 100}}
    lean, _, _ = shadowNear(right, 1, 10, 10)
    if lean >= 0 {
        test.Errorf("light to the right: lean %v", lean)
    }

    // the light below the unit on the screen: the shadow goes up
    below := []spellLightSource{{X: unitX, Y: unitY + 15, Radius: 60, Strength: 1, Reach: 100}}
    _, length, _ = shadowNear(below, 1, 10, 10)
    if length >= 0 {
        test.Errorf("light below: length %v", length)
    }

    // far from the light
    far := []spellLightSource{{X: unitX - 300, Y: unitY, Radius: 60, Strength: 1, Reach: 100}}
    lean, length, part = shadowNear(far, 1, 10, 10)
    if lean != shadowLean || length != shadowLength || part != 0 {
        test.Errorf("far from the light: %v %v %v", lean, length, part)
    }

    // the field is not dark yet
    lean, length, _ = shadowNear(left, 0, 10, 10)
    if lean != shadowLean || length != shadowLength {
        test.Errorf("before the field is dark: %v %v", lean, length)
    }
}

// a corpse takes the color on by its part and keeps its brightest part
func TestCorpseTint(test *testing.T) {
    red, green, blue := corpseTintParts(color.RGBA{R: 200, G: 100, B: 50, A: 255}, 0.5)
    if red != 1 || math.Abs(green - 0.75) > 0.001 || math.Abs(blue - 0.625) > 0.001 {
        test.Errorf("half of it: %v %v %v", red, green, blue)
    }

    red, green, blue = corpseTintParts(color.RGBA{R: 200, G: 100, B: 50, A: 255}, 0)
    if red != 1 || green != 1 || blue != 1 {
        test.Errorf("none of it: %v %v %v", red, green, blue)
    }
}
