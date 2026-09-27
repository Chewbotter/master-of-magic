package display

import (
    "math"
    "testing"
)

// window sizes here are real pixels
func TestDrawScaleFitsTheWindow(test *testing.T) {
    cases := []struct {
        width int
        height int
        expected float64
    }{
        // the user's 1776 x 1000 window at 150% display scaling
        {2664, 1500, 7},
        {1776, 1000, 5},
        {1600, 1000, 5},
        {1422, 800, 4},
        {640, 400, 2},
        // fullscreen on a 1080 monitor: the largest whole scale that fits
        {1920, 1080, 5},
        // fullscreen on the user's monitor
        {3840, 2160, 10},
        // a tall narrow window is limited by its width
        {800, 1000, 2},
        {100, 100, 1},
    }

    for _, testCase := range cases {
        got := DrawScaleForPixels(testCase.width, testCase.height)
        if got != testCase.expected {
            test.Errorf("window %vx%v: draw scale %v, expected %v", testCase.width, testCase.height, got, testCase.expected)
        }
    }
}

// every size in the resolution list must come out at exactly its draw scale in real pixels,
// with at most a pixel or two left over, at the usual Windows display scaling levels
func TestResolutionListFitsExactly(test *testing.T) {
    for _, factor := range []float64{1, 1.25, 1.5, 1.75, 2} {
        for _, widescreen := range []bool{true, false} {
            for drawScale := MinWindowScale; drawScale <= MaxWindowScale; drawScale++ {
                windowWidth, windowHeight := windowSizeFor(drawScale, widescreen, factor)
                realWidth := int(math.Round(float64(windowWidth) * factor))
                realHeight := int(math.Round(float64(windowHeight) * factor))

                got := DrawScaleForPixels(realWidth, realHeight)
                pictureWidth, pictureHeight := realSizeForScale(drawScale, widescreen)
                if got != float64(drawScale) || realHeight - pictureHeight > 2 || realWidth - pictureWidth > 2 {
                    test.Errorf("scaling %v, widescreen %v, draw scale %v: window %vx%v is %vx%v real, draws at %v with %vx%v left over",
                        factor, widescreen, drawScale, windowWidth, windowHeight, realWidth, realHeight, got,
                        realWidth - pictureWidth, realHeight - pictureHeight)
                }
            }
        }
    }
}
