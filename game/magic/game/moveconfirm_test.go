package game

import (
    "image/color"
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

func TestNextToStack(test *testing.T) {
    cases := []struct {
        DistanceX int
        FromY int
        ToY int
        Next bool
    }{
        {0, 5, 5, true},
        {1, 5, 5, true},
        {-1, 5, 6, true},
        {1, 5, 4, true},
        {2, 5, 5, false},
        {0, 5, 7, false},
        {-2, 5, 6, false},
        {1, 5, 3, false},
    }

    for _, use := range cases {
        if nextToStack(use.DistanceX, use.FromY, use.ToY) != use.Next {
            test.Errorf("distance %v, row %v to %v: next to the stack should be %v", use.DistanceX, use.FromY, use.ToY, use.Next)
        }
    }
}

// a stack that lost all of its units in a fight has no place the camera could go to
func TestStackWithoutUnitsHasNoPlace(test *testing.T) {
    gone := playerlib.MakeUnitStack()
    if gone.X() != 0 || gone.Y() != 0 {
        test.Fatalf("a stack without units is expected to say 0, 0, it says %v, %v", gone.X(), gone.Y())
    }

    if stackHasPlace(gone) {
        test.Errorf("the camera would be sent to the stack without units, which is the corner of the map")
    }

    if stackHasPlace(nil) {
        test.Errorf("no stack has no place")
    }
}

// four greens of a picture give four tones of the color of any wizard
func TestBannerShadesAreApart(test *testing.T) {
    palette := make(color.Palette, 256)
    for index := range palette {
        palette[index] = color.RGBA{A: 0xff}
    }
    palette[215] = color.RGBA{R: 116, G: 228, B: 112, A: 0xff}
    palette[216] = color.RGBA{R: 0, G: 188, B: 0, A: 0xff}
    palette[217] = color.RGBA{R: 0, G: 164, B: 0, A: 0xff}
    palette[218] = color.RGBA{R: 0, G: 124, B: 0, A: 0xff}

    for _, banner := range []data.BannerType{data.BannerBlue, data.BannerGreen, data.BannerPurple, data.BannerRed, data.BannerYellow, data.BannerBrown} {
        shades := bannerShades(palette, banner)
        seen := make(map[color.Color]bool)
        for index := 215; index <= 218; index++ {
            seen[shades[index]] = true
        }
        if len(seen) != 4 {
            test.Errorf("%v: the four shades are %v, %v, %v, %v", banner, shades[215], shades[216], shades[217], shades[218])
        }
    }
}
