package combat

import (
    "image"
    "math"
    "testing"
)

func TestWeatherNames(test *testing.T) {
    for _, weather := range Weathers {
        if WeatherByName(weather.String()) != weather {
            test.Fatalf("weather %v is not found by its name", weather)
        }
        look, falls := weatherLooks[weather]
        if look.Splash > len(rainSplashes) || look.Smooth != (weather == WeatherLightSnow || weather == WeatherHeavySnow) {
            test.Fatalf("weather %v: %v pictures of a splash, smooth %v", weather, look.Splash, look.Smooth)
        }
        if falls == (weather == WeatherNone || weather == WeatherClouds) {
            test.Fatalf("weather %v: falls %v", weather, falls)
        }
    }
    if WeatherByName("hail") != WeatherNone {
        test.Fatalf("a name that is not there")
    }
}

// rain and snow come down on the ground that is seen, land where they are to land and are gone,
// heavy weather has more of them than light, and they fall in gusts
func TestWeatherFalls(test *testing.T) {
    view := image.Rect(0, 0, 350, 200)
    counts := make(map[Weather]int)

    for _, weather := range []Weather{WeatherLightRain, WeatherHeavyRain, WeatherLightSnow, WeatherHeavySnow} {
        look := weatherLooks[weather]
        state := weatherState{Seed: 12345}
        area := view.Inset(-weatherAround)

        for range 600 {
            state.spawn(look, area, 1.0 / 60, func() float64 { return 0 })
            state.step(look, 1.0 / 60)
            for _, each := range state.Drops {
                if each.Y > each.Ground || each.Ground < float64(area.Min.Y) || each.Ground > float64(area.Max.Y) {
                    test.Fatalf("%v: a drop at %v that lands at %v", weather, each.Y, each.Ground)
                }
            }
        }
        counts[weather] = len(state.Drops)
        if len(state.Drops) < 30 || len(state.Drops) >= weatherMost {
            test.Fatalf("%v: %v in the air", weather, len(state.Drops))
        }

        // without new ones the air is empty after a while
        for range int((look.Height / look.Speed / (1 - look.Spread) + look.Linger) * 60) + 60 {
            state.step(look, 1.0 / 60)
        }
        if len(state.Drops) != 0 {
            test.Fatalf("%v: %v are left", weather, len(state.Drops))
        }

        // gusts: much comes down in places, little in others
        least, most := 1.0, 0.0
        for x := 0; x < 2000; x += 10 {
            gust := weatherGust(look, float64(x), 100, 3, 12345)
            least = min(least, gust)
            most = max(most, gust)
        }
        if least < 0 || most > 1 || most - least < 0.4 {
            test.Fatalf("%v: gusts from %v to %v", weather, least, most)
        }
    }

    if counts[WeatherHeavyRain] < counts[WeatherLightRain] * 2 || counts[WeatherHeavySnow] < counts[WeatherLightSnow] * 2 {
        test.Fatalf("in the air: %v", counts)
    }
}

// the shadows of clouds: a part of the ground is dark, a part is not, the edges between them are
// soft, and the picture goes on from its one edge to the other
func TestCloudShadows(test *testing.T) {
    picture := makeCloudShadows(99)
    bounds := picture.Bounds()
    if bounds.Dx() != cloudWidth || bounds.Dy() != cloudHeight {
        test.Fatalf("the picture is %v", bounds)
    }

    full := uint8(cloudDark * 255 + 0.5)
    dark, soft, none := 0, 0, 0
    for y := range cloudHeight {
        for x := range cloudWidth {
            pixel := picture.NRGBAAt(x, y)
            if pixel.R != 0 || pixel.G != 0 || pixel.B != 0 || pixel.A > full {
                test.Fatalf("a pixel %v", pixel)
            }
            switch {
                case pixel.A == 0: none += 1
                case pixel.A == full: dark += 1
                default: soft += 1
            }
            if pixel.A * 2 >= full {
                dark += 0
            }
            // from a pixel to the next it changes little
            if x > 0 {
                before := picture.NRGBAAt(x - 1, y).A
                if max(before, pixel.A) - min(before, pixel.A) > full / 4 {
                    test.Fatalf("a hard edge at %v, %v: %v and %v", x, y, before, pixel.A)
                }
            }
        }
    }
    if dark == 0 || none == 0 || soft == 0 {
        test.Fatalf("%v pixels in the shadow, %v at its edge, %v outside", dark, soft, none)
    }
    share := (float64(dark) + float64(soft) / 2) / float64(cloudWidth * cloudHeight)
    if share < cloudCover - 0.12 || share > cloudCover + 0.12 {
        test.Fatalf("%v of the ground is in the shadow", share)
    }

    for y := 0; y < cloudHeight; y += 7 {
        if cloudNoise(0, y, 99) != cloudNoise(cloudWidth, y, 99) {
            test.Fatalf("the picture does not go on from right to left at %v", y)
        }
    }
    for x := 0; x < cloudWidth; x += 7 {
        if cloudNoise(x, 0, 99) != cloudNoise(x, cloudHeight, 99) {
            test.Fatalf("the picture does not go on from bottom to top at %v", x)
        }
    }
}

// light rain splashes in one picture, heavy rain in two, and heavy snow has a thin layer of large
// flakes in front that fall through the view and are gone
func TestWeatherFront(test *testing.T) {
    if weatherLooks[WeatherLightRain].Splash != 1 || weatherLooks[WeatherHeavyRain].Splash != 2 || weatherLooks[WeatherHeavySnow].Splash != 0 {
        test.Fatalf("the pictures of the splashes")
    }

    look := weatherLooks[WeatherHeavySnow]
    if look.FrontSize != 2 || weatherLooks[WeatherLightSnow].FrontRate != 0 {
        test.Fatalf("the flakes in front")
    }

    area := image.Rect(0, 0, 350, 200)
    state := weatherState{Seed: 5}
    for range 900 {
        state.spawnFront(look, area, 1.0 / 60, func() float64 { return 0 })
        state.step(look, 1.0 / 60)
        for _, each := range state.Drops {
            if !each.Front || each.Landed > 0 || each.Y >= float64(area.Max.Y) {
                test.Fatalf("a flake in front: %+v", each)
            }
        }
    }
    if len(state.Drops) < 4 || len(state.Drops) > 30 {
        test.Fatalf("%v flakes in front", len(state.Drops))
    }

    for range 600 {
        state.step(look, 1.0 / 60)
    }
    if len(state.Drops) != 0 {
        test.Fatalf("%v flakes in front are left", len(state.Drops))
    }
}

// a flake of snow that falls between the pixels of the art is on the pixel it lands on when it
// lands, across and down
func TestSnowLands(test *testing.T) {
    look := weatherLooks[WeatherHeavySnow]
    for range 200 {
        drop := look.drop(100.4, 80.7, 0)
        if drop.Ground != 80 || drop.LandX != float64(int(drop.LandX)) {
            test.Fatalf("it lands at %v, %v", drop.LandX, drop.Ground)
        }

        state := weatherState{Drops: []weatherDrop{drop}}
        for len(state.Drops) > 0 && state.Drops[0].Landed == 0 {
            state.step(look, 1.0 / 60)
        }
        if len(state.Drops) == 0 {
            continue
        }
        landed := state.Drops[0]
        x := landed.X + math.Sin(landed.Age * look.SwayRate * 2 * math.Pi + landed.Phase) * look.Sway
        if landed.Y != 80 || math.Abs(x - landed.LandX - 0.5) > 1.5 {
            test.Fatalf("it has landed at %v, %v, and was to land at %v", x, landed.Y, landed.LandX)
        }
    }
}
