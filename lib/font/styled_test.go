package font

import (
    "image"
    "testing"
)

// the shadow copies per mode, as the original draws them
func TestShadowOffsets(test *testing.T) {
    counts := map[ShadowMode]int{
        ShadowNone: 0,
        ShadowDown: 3,
        ShadowUp: 3,
        ShadowHeavy: 6,
        ShadowOutline: 8,
        ShadowOutlineHeavy: 13,
    }

    for mode, expected := range counts {
        if len(mode.Offsets()) != expected {
            test.Errorf("mode %v: %v offsets, expected %v", mode, len(mode.Offsets()), expected)
        }
    }

    // chiseled text: only below and right
    for _, offset := range ShadowDown.Offsets() {
        if offset.X < 0 || offset.Y < 0 {
            test.Errorf("shadow down has an offset up or left: %v", offset)
        }
    }

    // a full outline covers all eight neighbors
    seen := map[image.Point]bool{}
    for _, offset := range ShadowOutline.Offsets() {
        seen[offset] = true
    }
    for x := -1; x <= 1; x++ {
        for y := -1; y <= 1; y++ {
            if (x != 0 || y != 0) && !seen[image.Pt(x, y)] {
                test.Errorf("outline misses the neighbor %v,%v", x, y)
            }
        }
    }
}
