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
// splash (light rain in one picture, heavy rain in two), a flake of snow after it has lain a
// moment. Snow falls between the pixels of the art and lands on one. Heavy snow has a thin layer
// of flakes of 2 by 2 pixels in front of everything, which fall through the view. They fall in GUSTS: how many come
// down at a place follows a number that changes slowly from place to place and moves with the
// wind, so rain comes in sheets with thin air between them.
//
// CLOUD SHADOWS lie on the ground as a picture with soft edges that moves with the wind, and
// what stands on the ground is darker as a whole where it stands in one. See drawCloudGround.
//
// Weather is for the look. It changes no rule.

import (
    "image"
    "image/color"
    "math"
    "math/rand/v2"
    "slices"
    "time"

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
    // drops splash where they land: in so many pictures, one after the other. 0 for none
    Splash int
    // it falls between the pixels of the art and lands on one (user, 2026-09-29: "let snow fall
    // smoothly instead of pixel-locked, but always land on a solid pixel")
    Smooth bool
    // flakes in front of everything, near the eye: how many come in a second on 100 art pixels
    // of the width of the view, how many art pixels across they are, and how fast they fall and
    // go with the wind. they fall through the view and land nowhere
    FrontRate float64
    FrontSize int
    FrontSpeed float64
    FrontWind float64
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
        Rate: 45, Speed: 230, Wind: -50, Spread: 0.15, Height: 130, Length: 3, Linger: 0.09, Splash: 1,
        Shades: []float32{0.35, 0.5, 0.65}, Color: color.NRGBA{R: 178, G: 200, B: 226, A: 255},
        GustSize: 90, GustSpeed: 45, GustDepth: 0.7,
    },
    WeatherHeavyRain: {
        Rate: 190, Speed: 270, Wind: -95, Spread: 0.15, Height: 140, Length: 4, Linger: 0.16, Splash: 2,
        Shades: []float32{0.4, 0.55, 0.75}, Color: color.NRGBA{R: 178, G: 200, B: 226, A: 255},
        GustSize: 110, GustSpeed: 80, GustDepth: 0.85,
    },
    WeatherLightSnow: {
        Rate: 4.5, Speed: 26, Wind: -6, Spread: 0.3, Height: 120, Length: 1, Sway: 2.5, SwayRate: 0.5, Linger: 0.5, Smooth: true,
        Shades: []float32{0.6, 0.8, 1}, Color: color.NRGBA{R: 244, G: 248, B: 255, A: 255},
        GustSize: 100, GustSpeed: 8, GustDepth: 0.6,
    },
    WeatherHeavySnow: {
        Rate: 40, Speed: 40, Wind: -28, Spread: 0.35, Height: 130, Length: 1, Sway: 3, SwayRate: 0.7, Linger: 0.6, Smooth: true,
        Shades: []float32{0.6, 0.8, 1}, Color: color.NRGBA{R: 244, G: 248, B: 255, A: 255},
        GustSize: 120, GustSpeed: 30, GustDepth: 0.8,
        // a thin layer of flakes of 4 pixels in front (user, 2026-09-29)
        FrontRate: 0.9, FrontSize: 2, FrontSpeed: 75, FrontWind: -55,
    },
}

// the pixels of the pictures of a splash, from the place a drop has landed on, and how much of
// the drop is seen of them. the first picture is small, the second wider and weaker
type splashPixel struct {
    X int
    Y int
    Shade float32
}

var rainSplashes = [][]splashPixel{
    {{0, 0, 1}, {-1, -1, 1}, {1, -1, 1}},
    {{-2, -1, 0.6}, {2, -1, 0.6}, {-1, -2, 0.45}, {1, -2, 0.45}},
}

// no more than this many at a time
const weatherMost = 6000
// what is seen of the ground, and this many art pixels around it, gets rain and snow
const weatherAround = 40
// over its last art pixels above the ground a flake that falls between the pixels of the art goes
// over to the pixel it lands on, so it does not jump there when it lands
const weatherSettle = 6.0

// the shadows of clouds: the size of their picture in art pixels, of the clouds in it, how dark
// they are, how wide their edge of single pixels is, how much of the ground they cover, and how
// fast they move, in art pixels a second
const cloudWidth = 512
const cloudHeight = 256
const cloudCells = 4
const cloudDark = 0.3
const cloudEdge = 0.45
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
    // the pixel of the art it lands on, across
    LandX float64
    // a flake in front of everything
    Front bool
}

type weatherState struct {
    Drops []weatherDrop
    // seconds of weather
    Time float64
    // what is left of the ones that are to come down, and of the ones in front
    Owed float64
    FrontOwed float64
    // what the view shows of the original's screen, from the last draw
    View image.Rectangle
    Seen bool
    Filled bool
    Seed uint32
    Clouds *ebiten.Image
    CloudPixels *image.NRGBA
    // when it was last moved on
    Last time.Time
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
    phase := rand.Float64() * 2 * math.Pi

    // where it will be when it lands, with its sway
    landX := x
    if look.Sway > 0 {
        landX += math.Sin(left * look.SwayRate * 2 * math.Pi + phase) * look.Sway
    }
    if look.Smooth {
        // it lands on a pixel of the art
        ground = math.Floor(ground)
    }

    return weatherDrop{
        X: x - wind * left,
        Y: ground - speed * left,
        SpeedX: wind,
        SpeedY: speed,
        Ground: ground,
        Phase: phase,
        Shade: look.Shades[rand.N(len(look.Shades))],
        LandX: math.Floor(landX),
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

// new flakes in front of everything for a part of a second. they fall through an area, from its
// top to its bottom
func (state *weatherState) spawnFront(look weatherLook, area image.Rectangle, seconds float64, fallen func() float64) {
    if look.FrontRate <= 0 {
        return
    }
    state.FrontOwed += look.FrontRate * float64(area.Dx()) / 100 * seconds
    count := int(state.FrontOwed)
    state.FrontOwed -= float64(count)

    for range count {
        if len(state.Drops) >= weatherMost {
            return
        }
        speed := look.FrontSpeed * randomPart(1 - look.Spread, 1 + look.Spread)
        wind := look.FrontWind * randomPart(1 - look.Spread, 1 + look.Spread)
        left := float64(area.Dy()) / speed * (1 - fallen())
        // where it leaves the area
        x := float64(area.Min.X) + rand.Float64() * float64(area.Dx())

        state.Drops = append(state.Drops, weatherDrop{
            X: x - wind * left,
            Y: float64(area.Max.Y) - speed * left,
            SpeedX: wind,
            SpeedY: speed,
            Ground: float64(area.Max.Y),
            Phase: rand.Float64() * 2 * math.Pi,
            Shade: 1,
            Front: true,
        })
    }
}

// what happens in a part of a second
func (state *weatherState) step(look weatherLook, seconds float64) {
    state.Time += seconds

    kept := state.Drops[:0]
    for _, each := range state.Drops {
        if each.Front {
            each.Age += seconds
            each.X += each.SpeedX * seconds
            each.Y += each.SpeedY * seconds
            if each.Y < each.Ground {
                kept = append(kept, each)
            }
            continue
        }

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

// the weather does not stand still with the battle: a spell that is cast holds the clock of the
// battle, and rain that hangs in the air shows it. it goes by the clock on the wall, once for every
// picture that is drawn of the field
const weatherStepMost = 0.1

func (combat *CombatScreen) weatherAdvance() {
    if combat.weatherKind() == WeatherNone {
        return
    }
    state := &combat.weather
    now := time.Now()
    if !state.Last.IsZero() {
        // not all at once after a pause
        combat.weatherTick(min(now.Sub(state.Last).Seconds(), weatherStepMost))
    }
    state.Last = now
}

// what happens to the weather in a part of a second
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
            if look.FrontRate > 0 {
                state.spawnFront(look, area, float64(area.Dy()) / look.FrontSpeed, rand.Float64)
            }
        }
        state.spawn(look, area, seconds, func() float64 { return 0 })
        state.spawnFront(look, area, seconds, func() float64 { return 0 })
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

// draws squares of art pixels of the weather at places of the original's screen, which can lie
// between the pixels of the art
func (combat *CombatScreen) weatherPainter(screen *ebiten.Image, look weatherLook) func(x float64, y float64, size int, shade float32) {
    state := &combat.weather
    state.View = combat.weatherView(screen)
    state.Seen = !state.View.Empty()

    pixel := combat.effectPixel()
    matrix := originalScreenMatrix(combat.GetCameraMatrix())
    red := float32(look.Color.R) / 255
    green := float32(look.Color.G) / 255
    blue := float32(look.Color.B) / 255

    var options ebiten.DrawImageOptions
    return func(x float64, y float64, size int, shade float32) {
        if !image.Pt(int(math.Floor(x)), int(math.Floor(y))).In(state.View.Inset(-size)) {
            return
        }
        options.GeoM.Reset()
        options.GeoM.Scale(float64(size), float64(size))
        options.GeoM.Translate(x, y)
        options.GeoM.Concat(matrix)
        options.ColorScale.Reset()
        options.ColorScale.Scale(red * shade, green * shade, blue * shade, shade)
        scale.DrawScaled(screen, pixel, &options)
    }
}

// what has landed: the splashes of rain and the flakes that lie. on the ground, under the units
// and what else stands on it (user, 2026-09-29: "have rain splashes appear underneath units")
func (combat *CombatScreen) drawWeatherGround(screen *ebiten.Image) {
    kind := combat.weatherKind()
    look, falls := weatherLooks[kind]
    if !falls {
        return
    }

    state := &combat.weather
    draw := combat.weatherPainter(screen, look)

    for index := range state.Drops {
        each := &state.Drops[index]
        if each.Front || each.Landed <= 0 {
            continue
        }
        landedY := math.Floor(each.Y)

        if look.Splash > 0 {
            // the pictures of its splash, one after the other
            picture := min(int(each.Landed / look.Linger * float64(look.Splash)), look.Splash - 1, len(rainSplashes) - 1)
            for _, part := range rainSplashes[picture] {
                draw(each.LandX + float64(part.X), landedY + float64(part.Y), 1, each.Shade * part.Shade)
            }
            continue
        }

        // less of it the longer it has lain
        draw(each.LandX, landedY, 1, each.Shade * float32(1 - each.Landed / look.Linger))
    }
}

// rain and snow that fall, over everything that is on the field
func (combat *CombatScreen) drawWeather(screen *ebiten.Image) {
    kind := combat.weatherKind()
    if kind == WeatherNone {
        return
    }

    state := &combat.weather
    look, falls := weatherLooks[kind]
    if !falls {
        state.View = combat.weatherView(screen)
        state.Seen = !state.View.Empty()
        return
    }
    draw := combat.weatherPainter(screen, look)

    swayed := func(each *weatherDrop) float64 {
        if look.Sway <= 0 {
            return each.X
        }
        return each.X + math.Sin(each.Age * look.SwayRate * 2 * math.Pi + each.Phase) * look.Sway
    }

    for index := range state.Drops {
        each := &state.Drops[index]
        if each.Front || each.Landed > 0 {
            continue
        }
        x := swayed(each)

        if look.Smooth {
            // over to the pixel it lands on
            settle := slopeSmooth(min(max((each.Ground - each.Y) / weatherSettle, 0), 1))
            draw(each.LandX + (x - each.LandX) * settle, each.Y, 1, each.Shade)
            continue
        }

        // its pixels, back along its way
        slant := each.SpeedX / each.SpeedY
        for part := range look.Length {
            draw(math.Floor(x - float64(part) * slant), math.Floor(each.Y) - float64(part), 1, each.Shade)
        }
    }

    // the flakes in front, over all others
    for index := range state.Drops {
        each := &state.Drops[index]
        if each.Front {
            draw(swayed(each), each.Y, look.FrontSize, each.Shade)
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

// the picture of the shadows of the clouds: black, the more of it the deeper in the shadow
func makeCloudShadows(seed uint32) *image.NRGBA {
    out := image.NewNRGBA(image.Rect(0, 0, cloudWidth, cloudHeight))

    // the numbers of all places in their order: what is higher than cloudCover of them is shadow
    values := make([]float64, 0, cloudWidth * cloudHeight / 16)
    for y := 0; y < cloudHeight; y += 4 {
        for x := 0; x < cloudWidth; x += 4 {
            values = append(values, cloudNoise(x, y, seed))
        }
    }
    slices.Sort(values)
    level := values[int(float64(len(values) - 1) * (1 - cloudCover))]

    for y := range cloudHeight {
        for x := range cloudWidth {
            // 0 outside of the shadow, 1 inside, between them along its edge
            inside := min(max((cloudNoise(x, y, seed) - level) / cloudEdge + 0.5, 0), 1)
            alpha := uint8(math.Round(slopeSmooth(inside) * cloudDark * 255))
            if alpha > 0 {
                out.SetNRGBA(x, y, color.NRGBA{A: alpha})
            }
        }
    }
    return out
}

// the picture of the shadows, made when it is first asked for
func (combat *CombatScreen) cloudShadows() *weatherState {
    state := &combat.weather
    if state.CloudPixels == nil {
        if state.Seed == 0 {
            state.Seed = rand.Uint32() | 1
        }
        state.CloudPixels = makeCloudShadows(state.Seed)
        state.Clouds = ebiten.NewImageFromImage(state.CloudPixels)
    }
    return state
}

// how far the shadows have moved, in art pixels
func (state *weatherState) cloudShift() (float64, float64) {
    return state.Time * cloudSpeedX, state.Time * cloudSpeedY
}

// THE SHADOWS OF CLOUDS, SECOND BUILD (user, 2026-09-29). The first was one picture with edges of
// a pattern of pixels over everything, moved a pixel of the art at a time: the whole pattern
// jumped with every step ("a lot of popping for something so large"), and the edge of a shadow cut
// through a unit where it cut the ground behind it ("clearly an overlaid image ... it breaks the
// illusion of depth"). Now
//   the ground has the shadows as a picture with soft edges, which moves a pixel of the SCREEN at
//   a time. it lies under what stands on the ground
//   what stands on the ground, units, corpses, trees, rocks, houses, is darker AS A WHOLE by the
//   shadow at the place it stands on. the edge is soft, so it gets darker and lighter slowly

// the shadows of the clouds on the ground
func (combat *CombatScreen) drawCloudGround(screen *ebiten.Image) {
    if combat.weatherKind() != WeatherClouds {
        return
    }
    state := combat.cloudShadows()

    view := combat.weatherView(screen)
    if view.Empty() {
        return
    }

    shiftX, shiftY := state.cloudShift()
    matrix := originalScreenMatrix(combat.GetCameraMatrix())
    matrix.Concat(scale.ScaledGeom)

    firstX, _ := floorDivide(view.Min.X - int(math.Ceil(shiftX)), cloudWidth)
    firstY, _ := floorDivide(view.Min.Y - int(math.Ceil(shiftY)), cloudHeight)

    var options ebiten.DrawImageOptions
    for tileY := firstY; float64(tileY * cloudHeight) + shiftY < float64(view.Max.Y); tileY++ {
        for tileX := firstX; float64(tileX * cloudWidth) + shiftX < float64(view.Max.X); tileX++ {
            options.GeoM.Reset()
            options.GeoM.Translate(float64(tileX * cloudWidth) + shiftX, float64(tileY * cloudHeight) + shiftY)
            options.GeoM.Concat(matrix)
            // on whole pixels of the screen, so the pixels of the picture keep their shape
            options.GeoM.SetElement(0, 2, math.Round(options.GeoM.Element(0, 2)))
            options.GeoM.SetElement(1, 2, math.Round(options.GeoM.Element(1, 2)))
            screen.DrawImage(state.Clouds, &options)
        }
    }
}

// how bright something that stands at a place of the original's screen is under the clouds: 1
// without a shadow
func (combat *CombatScreen) cloudShade(screenX int, screenY int) float32 {
    if combat.weatherKind() != WeatherClouds {
        return 1
    }
    state := combat.cloudShadows()

    shiftX, shiftY := state.cloudShift()
    _, x := floorDivide(screenX - int(math.Floor(shiftX)), cloudWidth)
    _, y := floorDivide(screenY - int(math.Floor(shiftY)), cloudHeight)
    return 1 - float32(state.CloudPixels.NRGBAAt(x, y).A) / 255
}

// the same for a unit, by the tile it stands on or walks over
func (combat *CombatScreen) cloudShadeOfUnit(unit *ArmyUnit) float32 {
    if combat.weatherKind() != WeatherClouds {
        return 1
    }
    x, y := float64(unit.X), float64(unit.Y)
    if unit.Moving {
        x, y = unit.MoveX, unit.MoveY
    }
    field := MakeBattlefieldMatrix()
    screenX, screenY := field.Apply(x, y)
    return combat.cloudShade(int(math.Round(screenX)), int(math.Round(screenY)))
}
