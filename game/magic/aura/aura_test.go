package combat

import (
    "image"
    "image/color"
    "testing"
)

func TestAura(test *testing.T) {
    palette := make(color.Palette, 256)
    for index := range palette {
        palette[index] = color.RGBA{R: uint8(index), G: uint8(index), B: uint8(index), A: 0xff}
    }
    palette[0] = color.RGBA{}

    // a figure of one pixel in the middle, a shadow color right of it, a pixel in a corner
    picture := image.NewPaletted(image.Rect(0, 0, 5, 5), palette)
    picture.SetColorIndex(2, 2, 10)
    picture.SetColorIndex(3, 2, 230)
    picture.SetColorIndex(0, 0, 10)

    got := make(map[image.Point]bool)
    for _, pixel := range auraPixels(picture) {
        got[pixel] = true
    }

    want := []image.Point{{1, 2}, {3, 2}, {2, 1}, {2, 3}, {1, 0}, {0, 1}}
    if len(got) != len(want) {
        test.Errorf("outline has %v pixels, want %v: %v", len(got), len(want), got)
    }
    for _, pixel := range want {
        if !got[pixel] {
            test.Errorf("no outline at %v", pixel)
        }
    }

    // the numbers of the original: the step grows by one a pixel, so 0 0 1 3 6 2 7 5, then 4 4 ...
    picks := auraPicks(18)
    wantPicks := []int{0, 0, 1, 3, 6, 2, 7, 5, 4, 4, 5, 7, 2, 6, 3, 1, 0, 0}
    for index, pick := range wantPicks {
        if picks[index] != pick {
            test.Errorf("pick %v is %v, want %v", index, picks[index], pick)
        }
    }

    // a stage later every list of the original starts one further
    out := withAura(func(original *image.Paletted) image.Image { return original }, auraLife, 1)(picture)
    r, _, _, _ := out.At(1, 2).RGBA()
    if r == 0 {
        test.Errorf("the outline has no color")
    }
    _, _, _, alpha := out.At(4, 4).RGBA()
    if alpha != 0 {
        test.Errorf("a pixel away from the figure is not empty")
    }
}
