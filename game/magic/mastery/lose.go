package mastery

// The lose animation: the wizard of the player falls into the vortex, turning and shrinking.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/CONQUEST.c:
// Lose_Animation, Lose_Animation_Draw, TBL_DefeatAnimSteps; MoX/src/FLIC_Draw.c:
// Draw_Bitmap_Rotated, Get_Rotated_Pixels). The code is ours; upstream had no lose screen.
//
//   the vortex: lose.lbx 0, an animation, a frame a stage
//   the wizard: lose.lbx 1 + wizard, in 27 stages of place (its upper left corner), angle and
//     size; the picture is made smaller first and then turned around its middle, the turn
//     leaning the way the original's does (a sideways part times 1.2, a downward part times 0.88)
//   a stage every 4 ticks of the original's clock (55 ms), so about 4.5 a second; the screen ends
//     after 46 stages or with a click or Escape
//   the palette of fonts.lbx for palette 6 (Load_Palette(6), entry 8)

import (
    "image"
    "image/color"
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/game/magic/util"
    "github.com/kazzmir/master-of-magic/game/magic/inputmanager"
    "github.com/kazzmir/master-of-magic/lib/coroutine"
    "github.com/kazzmir/master-of-magic/lib/lbx"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/inpututil"
)

type loseStep struct {
    X, Y, Angle, Size int
}

var loseSteps = []loseStep{
    {0, 90, 0, 100}, {12, 73, 10, 100}, {25, 57, 20, 100}, {32, 45, 30, 100}, {40, 29, 40, 100},
    {42, 15, 50, 100}, {55, 3, 60, 100}, {77, -4, 70, 100}, {119, -10, 80, 95}, {147, -8, 90, 90},
    {170, -5, 100, 85}, {190, 7, 110, 80}, {212, 19, 120, 75}, {218, 40, 130, 70}, {227, 59, 140, 65},
    {217, 79, 150, 60}, {212, 97, 160, 55}, {196, 106, 170, 50}, {179, 113, 180, 45}, {160, 109, 190, 40},
    {145, 105, 200, 35}, {135, 100, 210, 30}, {130, 94, 220, 25}, {135, 87, 230, 20}, {147, 87, 240, 15},
    {152, 89, 250, 10}, {159, 98, 260, 5},
}

const (
    loseStages = 46
    // frames of the game (60 a second) a stage: 4 ticks of 55 ms
    loseFramesPerStage = 13
    losePaletteEntry = 8
)

// development: the stage the fall starts at (-capture-lose-stage)
var CaptureLoseStage = 0

func LoseScreen(cache *lbx.LbxCache, wizard data.WizardBase) (coroutine.AcceptYieldFunc, func (*ebiten.Image)) {
    imageCache := util.MakeImageCache(cache)

    palette := losePalette(cache)
    vortex, _ := imageCache.GetImagesTransform("lose.lbx", 0, "lose-palette", paletteTransform(palette))
    figures, _ := imageCache.GetImagesTransform("lose.lbx", 1 + int(wizard), "lose-palette", paletteTransform(palette))

    stage := CaptureLoseStage
    frames := 0

    logic := func (yield coroutine.YieldFunc) error {
        if yield() != nil {
            return nil
        }
        for stage <= loseStages {
            if inputmanager.LeftClick() || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
                break
            }
            frames += 1
            if frames >= loseFramesPerStage {
                frames = 0
                stage += 1
            }
            if yield() != nil {
                return nil
            }
        }
        return nil
    }

    picture := ebiten.NewImage(data.ScreenWidth, data.ScreenHeight)

    draw := func (screen *ebiten.Image) {
        picture.Fill(color.Black)
        if len(vortex) > 0 {
            picture.DrawImage(vortex[stage % len(vortex)], &ebiten.DrawImageOptions{})
        }
        if stage < len(loseSteps) && len(figures) > 0 {
            step := loseSteps[stage]
            figure := figures[0]
            size := float64(step.Size) / 100
            width := float64(figure.Bounds().Dx()) * size
            height := float64(figure.Bounds().Dy()) * size
            angle := float64(step.Angle) * math.Pi / 180
            cos, sin := math.Cos(angle), math.Sin(angle)

            var options ebiten.DrawImageOptions
            options.GeoM.Scale(size, size)
            options.GeoM.Translate(-width / 2, -height / 2)
            // Get_Rotated_Pixels: x' = cos x - 1.2 sin y, y' = cos y + 0.88 sin x
            var turn ebiten.GeoM
            turn.SetElement(0, 0, cos)
            turn.SetElement(0, 1, -sin * 12 / 10)
            turn.SetElement(1, 0, sin * 22 / 25)
            turn.SetElement(1, 1, cos)
            options.GeoM.Concat(turn)
            options.GeoM.Translate(float64(step.X) + width / 2, float64(step.Y) + height / 2)
            picture.DrawImage(figure, &options)
        }
        scale.DrawScaled(screen, picture, &ebiten.DrawImageOptions{})
    }

    return logic, draw
}

func losePalette(cache *lbx.LbxCache) color.Palette {
    fonts, err := cache.GetLbxFile("fonts.lbx")
    if err != nil {
        return nil
    }
    raw, err := fonts.RawData(losePaletteEntry)
    if err != nil || len(raw) < 768 {
        return nil
    }
    // a palette of the data has 6 bits a color; 0 is not drawn
    palette := color.Palette{color.RGBA{}}
    for index := 1; index < 256; index++ {
        palette = append(palette, color.RGBA{R: raw[index * 3] << 2, G: raw[index * 3 + 1] << 2, B: raw[index * 3 + 2] << 2, A: 255})
    }
    return palette
}

// the picture with the colors of the palette of the screen
func paletteTransform(palette color.Palette) util.ImageTransformFunc {
    return func(picture *image.Paletted) image.Image {
        if palette == nil {
            return picture
        }
        out := *picture
        out.Palette = palette
        return &out
    }
}
