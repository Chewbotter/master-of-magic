package mod

import (
    "image"
    "image/color"
    "testing"
)

func testPalette() color.Palette {
    return color.Palette{
        color.RGBA{},
        color.RGBA{R: 10, G: 10, B: 10, A: 255},
        color.RGBA{R: 200, G: 0, B: 0, A: 255},
        color.RGBA{R: 0, G: 188, B: 0, A: 255},
    }
}

// a picture in indexed color with the palette of the game keeps the numbers of its colors
func TestIndexedPictureIsTakenAsItIs(test *testing.T) {
    palette := testPalette()
    source := image.NewPaletted(image.Rect(0, 0, 2, 1), palette)
    source.SetColorIndex(0, 0, 3)
    source.SetColorIndex(1, 0, 0)

    out := toPaletted(source, palette, "test")
    if out.ColorIndexAt(0, 0) != 3 || out.ColorIndexAt(1, 0) != 0 {
        test.Fatalf("got %v and %v", out.ColorIndexAt(0, 0), out.ColorIndexAt(1, 0))
    }
}

// a picture in full color is matched to the palette, see-through stays see-through
func TestFullColorPictureIsMatched(test *testing.T) {
    palette := testPalette()
    source := image.NewNRGBA(image.Rect(0, 0, 3, 1))
    source.Set(0, 0, color.NRGBA{R: 0, G: 188, B: 0, A: 255})
    source.Set(1, 0, color.NRGBA{R: 190, G: 20, B: 10, A: 255})
    source.Set(2, 0, color.NRGBA{R: 255, G: 255, B: 255, A: 0})

    out := toPaletted(source, palette, "test")
    if out.ColorIndexAt(0, 0) != 3 {
        test.Fatalf("the same color: got %v", out.ColorIndexAt(0, 0))
    }
    if out.ColorIndexAt(1, 0) != 2 {
        test.Fatalf("the closest color: got %v", out.ColorIndexAt(1, 0))
    }
    if out.ColorIndexAt(2, 0) != 0 {
        test.Fatalf("see-through: got %v", out.ColorIndexAt(2, 0))
    }
}

// black that is not see-through must not become the see-through color
func TestSolidBlackStaysSolid(test *testing.T) {
    palette := testPalette()
    source := image.NewNRGBA(image.Rect(0, 0, 1, 1))
    source.Set(0, 0, color.NRGBA{R: 0, G: 0, B: 0, A: 255})

    out := toPaletted(source, palette, "test")
    if out.ColorIndexAt(0, 0) == 0 {
        test.Fatalf("solid black became see-through")
    }
}
