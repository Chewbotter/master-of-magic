package combat

// Weather (user, 2026-09-29). Not in the original.
//
// "Let's start with light rain, heavy rain, light snow, heavy snow, and cloud shadows that roll
// across the ground. Everything but the cloud shadows should use particles that come down from the
// sky in non-uniform patterns. The clouds themselves are never visible, just their shadow shapes
// that scroll across the ground and darken units and terrain underneath."
//
// RAIN AND SNOW are particles of single art pixels on the original's screen, as all particles are
// (particles.go), so they are as large as the pixels of everything else at every zoom and stay
// with the ground when the view moves. Every one has a place on the ground it comes down on: it
// starts high above it, falls with the wind and is gone where it lands, a drop of rain with a
// small splash, a flake of snow after it has lain a moment. They fall in GUSTS: how many come
// down at a place follows a number that changes slowly from place to place and moves with the
// wind, so rain comes in sheets with thin air between them.
//
// CLOUD SHADOWS are one picture of art pixels that is laid over the field and what stands on it,
// again and again side by side, and moves with the wind. Its edges are made of a pattern of
// pixels, not of half dark ones.
//
// Weather is for the look. It changes no rule.

import (
    "image"
    "image/color"
    "math"
    "math/rand/v2"
    "slices"

    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
)

type Weather int
const (
    WeatherNone Weather = iota
    WeatherLightRain
    WeatherHeavyRain
    WeatherLightSnow
    WeatherHeavySnow
    WeatherClouds
)

var Weathers = []Weather{WeatherNone, WeatherLightRain, WeatherHeavyRain, WeatherLightSnow, WeatherHeavySnow, WeatherClouds}

// the name it is kept and asked for under
func (weather Weather) String() string {
    switch weather {
        case WeatherLightRain: return "rain"
        case WeatherHeavyRain: return "heavyrain"
        case WeatherLightSnow: return "snow"
        case WeatherHeavySnow: return "heavysnow"
        case WeatherClouds: return "clouds"
    }
    return "none"
}

func WeatherByName(name string) Weather {
    for _, weather := range Weathers {
        if weather.String() == name {
            return weather
        }
    }
    return WeatherNone
}

// what falls
type weatherLook struct {
    // how many come down in a second on 100 by 100 art pixels of the ground
    Rate float64
    // art pixels a second: down, and with the wind
    Speed float64
    Wind float64
    // of both, by chance, this much more or less
    Spread float64
    // art pixels above the ground they start at
    Height float64
    // the art pixels of one, in a line along its way
    Length int
    // a flake sways this many art pixels to each side, this many times a second
    Sway float64
    SwayRate float64
    // seconds it is seen where it has landed
    Linger float64
    // drops splash: a pixel to each side where they land
    Splash bool
    // how much there is of it, of 1, from far to near
    Shades []float32
    Color color.NRGBA
    // gusts: how many art pixels across, how fast they move, and how much they change how much
    // comes down, 0 for even and 1 for none at all between them
    GustSize float64
    GustSpeed float64
    GustDepth float64
}

var weatherLooks = map[Weather]weatherLook{
    WeatherLightRain: {
        Rate: 45, Speed: 230, Wind: -50, Spread: 0.15, Height: 130, Length: 3, Linger: 0.07,
        Shades: []float32{0.35, 0.5, 0.65}, Color: color.NRGBA{R: 178, G: 200, B: 226, A: 255},
        GustSize: 90, GustSpeed: 45, GustDepth: 0.7,
    },
    WeatherHeavyRain: {
        Rate: 190, Speed: 270, Wind: -95, Spread: 0.15, Height: 140, Length: 4, Linger: 0.09, Splash: true,
        Shades: []float32{0.4, 0.55, 0.75}, Color: color.NRGBA{R: 178, G: 200, B: 226, A: 255},
        GustSize: 110, GustSpeed: 80, GustDepth: 0.85,
    },
    WeatherLightSnow: {
        Rate: 4.5, Speed: 26, Wind: -6, Spread: 0.3, Height: 120, Length: 1, Sway: 2.5, SwayRate: 0.5, Linger: 0.5,
        Shades: []float32{0.6, 0.8, 1}, Color: color.NRGBA{R: 244, G: 248, B: 255, A: 255},
        GustSize: 100, GustSpeed: 8, GustDepth: 0.6,
    },
    WeatherHeavySnow: {
        Rate: 40, Speed: 40, Wind: -28, Spread: 0.35, Height: 130, Length: 1, Sway: 3, SwayRate: 0.7, Linger: 0.6,
        Shades: []float32{0.6, 0.8, 1}, Color: color.NRGBA{R: 244, G: 248, B: 255, A: 255},
        GustSize: 120, GustSpeed: 30, GustDepth: 0.8,
    },
}

// no more than this many at a time
const weatherMost = 6000
// what is seen of the ground, and this many art pixels around it, gets rain and snow
const weatherAround = 40

// the shadows of clouds: the size of their picture in art pixels, of the clouds in it, how dark
// they are, how wide their edge of single pixels is, how much of the ground they cover, and how
// fast they move, in art pixels a second
const cloudWidth = 512
const cloudHeight = 256
const cloudCells = 4
const cloudDark = 0.3
const cloudEdge = 0.12
const cloudCover = 0.45
const cloudSpeedX = -14.0
const cloudSpeedY = 5.0

type weatherDrop struct {
    // on the original's screen, in art pixels
    X float64
    Y float64
    SpeedX float64
    SpeedY float64
    // where it lands
    Ground float64
    // seconds: it has fallen, and it has lain
    Age float64
    Landed float64
    Phase float64
    Shade float32
}

type weatherState struct {
    Drops []weatherDrop
    // seconds of weather
    Time float64
    // what is left of the ones that are to come down
    Owed float64
    // what the view shows of the original's screen, from the last draw
    View image.Rectangle
    Seen bool
    Filled bool
    Seed uint32
    Clouds *ebiten.Image
}

func (combat *CombatScreen) weatherKind() Weather {
    return combat.Model.Zone.Ground.Weather
}

// how much comes down at a place of the ground, 0 to 1, at a time
func weatherGust(look weatherLook, x float64, y float64, time float64, seed uint32) float64 {
    if look.GustSize <= 0 {
        return 1
    }
    // gusts are wider than deep on the screen, as the ground is seen from above at an angle
    // and move with the wind
    moved := time * math.Copysign(look.GustSpeed, look.Wind)
    at := patchNoise(int(math.Floor(x - moved)), int(math.Floor(y * 2)), look.GustSize, seed)
    return min(max(0.5 + at * 2 * look.GustDepth, 1 - look.GustDepth), 1)
}

// one that comes down on a place of the ground. fallen: how much of its way it has fallen, 0 to 1
func (look weatherLook) drop(x float64, ground float64, fallen float64) weatherDrop {
    speed := look.Speed * randomPart(1 - look.Spread, 1 + look.Spread)
    wind := look.Wind * randomPart(1 - look.Spread, 1 + look.Spread)
    left := look.Height / speed * (1 - fallen)

    return weatherDrop{
        X: x - wind * left,
        Y: ground - speed * left,
        SpeedX: wind,
        SpeedY: speed,
        Ground: ground,
        Phase: rand.Float64() * 2 * math.Pi,
        Shade: look.Shades[rand.N(len(look.Shades))],
    }
}

// new ones for a part of a second, over an area of the ground
func (state *weatherState) spawn(look weatherLook, area image.Rectangle, seconds float64, fallen func() float64) {
    state.Owed += look.Rate * float64(area.Dx()) * float64(area.Dy()) / 10000 * seconds
    count := int(state.Owed)
    state.Owed -= float64(count)

    for range count {
        if len(state.Drops) >= weatherMost {
            return
        }
        x := float64(area.Min.X) + rand.Float64() * float64(area.Dx())
        y := float64(area.Min.Y) + rand.Float64() * float64(area.Dy())
        if rand.Float64() >= weatherGust(look, x, y, state.Time, state.Seed) {
            continue
        }
        state.Drops = append(state.Drops, look.drop(x, y, fallen()))
    }
}

// what happens in a part of a second
func (state *weatherState) step(look weatherLook, seconds float64) {
    state.Time += seconds

    kept := state.Drops[:0]
    for _, each := range state.Drops {
        if each.Landed > 0 || each.Y >= each.Ground {
            each.Y = each.Ground
            each.Landed += seconds
            if each.Landed >= look.Linger {
                continue
            }
        } else {
            each.Age += seconds
            each.X += each.SpeedX * seconds
            each.Y = math.Min(each.Y + each.SpeedY * seconds, each.Ground)
        }
        kept = append(kept, each)
    }
    state.Drops = kept
}

// one tick of the weather with the clock of the battle. called with the effects, see
// spelleffects.go
func (combat *CombatScreen) weatherTick(seconds float64) {
    state := &combat.weather
    kind := combat.weatherKind()
    if kind == WeatherNone {
        return
    }
    if state.Seed == 0 {
        state.Seed = rand.Uint32() | 1
    }

    look, falls := weatherLooks[kind]
    if !falls {
        state.Time += seconds
        return
    }

    if state.Seen {
        area := state.View.Inset(-weatherAround)
        if !state.Filled {
            // the air is full of it when the battle starts
            state.Filled = true
            state.spawn(look, area, look.Height / look.Speed, rand.Float64)
        }
        state.spawn(look, area, seconds, func() float64 { return 0 })
    }
    state.step(look, seconds)
}

// what the view shows of the original's screen
func (combat *CombatScreen) weatherView(screen *ebiten.Image) image.Rectangle {
    matrix := originalScreenMatrix(combat.GetCameraMatrix())
    matrix.Concat(scale.ScaledGeom)
    if !matrix.IsInvertible() {
        return image.Rectangle{}
    }
    matrix.Invert()

    bounds := screen.Bounds()
    x1, y1 := matrix.Apply(float64(bounds.Min.X), float64(bounds.Min.Y))
    x2, y2 := matrix.Apply(float64(bounds.Max.X), float64(bounds.Max.Y))
    return image.Rect(int(math.Floor(min(x1, x2))), int(math.Floor(min(y1, y2))), int(math.Ceil(max(x1, x2))), int(math.Ceil(max(y1, y2))))
}

// rain and snow, over everything that is on the field
func (combat *CombatScreen) drawWeather(screen *ebiten.Image) {
    kind := combat.weatherKind()
    if kind == WeatherNone {
        return
    }

    state := &combat.weather
    state.View = combat.weatherView(screen)
    state.Seen = !state.View.Empty()

    look, falls := weatherLooks[kind]
    if !falls {
        return
    }

    pixel := combat.effectPixel()
    matrix := originalScreenMatrix(combat.GetCameraMatrix())
    red := float32(look.Color.R) / 255
    green := float32(look.Color.G) / 255
    blue := float32(look.Color.B) / 255

    var options ebiten.DrawImageOptions
    draw := func(x int, y int, shade float32) {
        if !image.Pt(x, y).In(state.View) {
            return
        }
        options.GeoM.Reset()
        options.GeoM.Translate(float64(x), float64(y))
        options.GeoM.Concat(matrix)
        options.ColorScale.Reset()
        options.ColorScale.Scale(red * shade, green * shade, blue * shade, shade)
        scale.DrawScaled(screen, pixel, &options)
    }

    for index := range state.Drops {
        each := &state.Drops[index]
        x := each.X
        if look.Sway > 0 {
            x += math.Sin(each.Age * look.SwayRate * 2 * math.Pi + each.Phase) * look.Sway
        }

        if each.Landed > 0 {
            // less of it the longer it has lain
            shade := each.Shade * float32(1 - each.Landed / look.Linger)
            draw(int(math.Floor(x)), int(math.Floor(each.Y)), shade)
            if look.Splash {
                draw(int(math.Floor(x)) - 1, int(math.Floor(each.Y)) - 1, shade)
                draw(int(math.Floor(x)) + 1, int(math.Floor(each.Y)) - 1, shade)
            }
            continue
        }

        // its pixels, back along its way
        slant := each.SpeedX / each.SpeedY
        for part := range look.Length {
            draw(int(math.Floor(x - float64(part) * slant)), int(math.Floor(each.Y)) - part, each.Shade)
        }
    }
}

// a number for a place of the picture of the clouds, -1 to 1, that goes on from its right edge to
// its left and from its bottom to its top
func cloudNoise(x int, y int, seed uint32) float64 {
    at := func(cells int, seed uint32) float64 {
        sizeX := float64(cloudWidth) / float64(cells)
        sizeY := float64(cloudHeight) / float64(cells)
        cellX := float64(x) / sizeX
        cellY := float64(y) / sizeY
        beforeX := int(math.Floor(cellX))
        beforeY := int(math.Floor(cellY))
        partX := slopeSmooth(cellX - float64(beforeX))
        partY := slopeSmooth(cellY - float64(beforeY))

        corner := func(dx int, dy int) float64 {
            return coastNoise((beforeX + dx) % cells, (beforeY + dy) % cells, seed)
        }
        top := corner(0, 0) * (1 - partX) + corner(1, 0) * partX
        bottom := corner(0, 1) * (1 - partX) + corner(1, 1) * partX
        return top * (1 - partY) + bottom * partY
    }

    // large shapes, with smaller ones along their edges
    return at(cloudCells, seed) * 0.7 + at(cloudCells * 2, seed + 7) * 0.3
}

// the pattern the edge of a shadow is made of: a pixel is dark if there is more shadow at its
// place than its number of 16
var cloudPattern = [4][4]int{{0, 8, 2, 10}, {12, 4, 14, 6}, {3, 11, 1, 9}, {15, 7, 13, 5}}

// the picture of the shadows of the clouds
func makeCloudShadows(seed uint32) *image.NRGBA {
    out := image.NewNRGBA(image.Rect(0, 0, cloudWidth, cloudHeight))

    // the numbers of all places in their order: what is higher than cloudCover of them is shadow
    values := make([]float64, 0, cloudWidth * cloudHeight / 16)
    for y := 0; y < cloudHeight; y += 4 {
        for x := 0; x < cloudWidth; x += 4 {
            values = append(values, cloudNoise(x, y, seed))
        }
    }
    sorted := append([]float64(nil), values...)
    slices.Sort(sorted)
    level := sorted[int(float64(len(sorted) - 1) * (1 - cloudCover))]

    dark := color.NRGBA{A: uint8(math.Round(cloudDark * 255))}
    for y := range cloudHeight {
        for x := range cloudWidth {
            // 0 outside of the shadow, 1 inside, between them along its edge
            inside := (cloudNoise(x, y, seed) - level) / cloudEdge + 0.5
            if inside * 16 > float64(cloudPattern[y % 4][x % 4]) + 0.5 {
                out.SetNRGBA(x, y, dark)
            }
        }
    }
    return out
}

// the shadows of the clouds, over the ground and what stands on it
func (combat *CombatScreen) drawCloudShadows(screen *ebiten.Image) {
    if combat.weatherKind() != WeatherClouds {
        return
    }

    state := &combat.weather
    if state.Clouds == nil {
        if state.Seed == 0 {
            state.Seed = rand.Uint32() | 1
        }
        state.Clouds = ebiten.NewImageFromImage(makeCloudShadows(state.Seed))
    }

    view := combat.weatherView(screen)
    if view.Empty() {
        return
    }

    // whole art pixels
    shiftX := int(math.Floor(state.Time * cloudSpeedX))
    shiftY := int(math.Floor(state.Time * cloudSpeedY))

    firstX, _ := floorDivide(view.Min.X - shiftX, cloudWidth)
    firstY, _ := floorDivide(view.Min.Y - shiftY, cloudHeight)
    for y := firstY * cloudHeight + shiftY; y < view.Max.Y; y += cloudHeight {
        for x := firstX * cloudWidth + shiftX; x < view.Max.X; x += cloudWidth {
            combat.drawOnField(screen, state.Clouds, x, y)
        }
    }
}
