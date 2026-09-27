package combat

// The numbers that rise from a unit when it is hurt. The original has none, so there is nothing to
// match: they are drawn like the rest of the original's text instead, in one of its fonts, in its
// palette's reds, with a black border of whole art pixels.

import (
    "fmt"
    "image/color"
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/lib/font"

    "github.com/hajimehoshi/ebiten/v2"
)

// the font of the original, by index: its smallest, the one the combat bar uses. half the size of
// the medium font the numbers had first (user request)
const damageFontIndex = 0
// palette indexes, for the glyph's pixel values 1 and up. the reds of the red wizard's name
var damageColors = []int{201, 201, 202, 203, 166}
const damageOutlineColor = 0

// where a number starts, in art pixels from the middle of the unit's tile
const damageStartY = -22
// numbers keep this many art pixels apart. one that would touch another moves away from it, the
// shorter way of sideways and up or down
const damageGap = 4
// how far from the middle of the unit a number can start, in art pixels each way. a group of
// numbers is a loose cluster
const damageSpreadX = 14
const damageSpreadY = 7
// how often a number moves away from another before it stays where it is
const damagePlaceTries = 16
// art pixels around the letters in the picture of a number, room for the border
const damagePictureMargin = 2
// ticks the number takes to rise one art pixel
const damageRiseTicks = 5
// the last ticks of its life the number fades out over
const damageFadeTicks = 10

func makeDamageFont(lbxFonts []*font.LbxFont, palette color.Palette) *font.StyledFont {
    if len(lbxFonts) <= damageFontIndex {
        return nil
    }

    colors := []color.Color{nil}
    for _, index := range damageColors {
        colors = append(colors, palette[index])
    }

    black := color.RGBA{R: 0, G: 0, B: 0, A: 0xff}
    return font.MakeStyledFont(lbxFonts[damageFontIndex], colors, font.ShadowOutline, black)
}

// the picture of a number. a number is drawn from many layers, the letters and the copies that make
// its border, so it is put together once and fades as one picture
func (combat *CombatScreen) damagePicture(text string) *ebiten.Image {
    key := fmt.Sprintf("%v@%v", text, scale.ScaleAmount)
    if picture, ok := combat.damagePictures[key]; ok {
        return picture
    }

    styled := combat.Fonts.Hud.Damage
    width := scale.Scale(styled.Width(text) + damagePictureMargin * 2)
    height := scale.Scale(styled.Height() + damagePictureMargin * 2)

    picture := ebiten.NewImage(width, height)
    styled.Print(picture, damagePictureMargin, damagePictureMargin, font.FontOptions{Scale: scale.ScaleAmount}, text)

    if combat.damagePictures == nil {
        combat.damagePictures = make(map[string]*ebiten.Image)
    }
    combat.damagePictures[key] = picture
    return picture
}

func (combat *CombatScreen) drawDamageNumbers(screen *ebiten.Image) {
    hud := combat.Fonts.Hud
    if hud == nil || hud.Damage == nil {
        return
    }

    matrix := combat.GetCameraMatrix()
    gap := damageGap * scale.ScaleAmount

    type placed struct {
        // the middle and half the size, in screen pixels
        X, Y float64
        HalfWidth, HalfHeight float64
    }
    var taken []placed

    for index, indicator := range combat.DamageIndicators {
        text := fmt.Sprintf("%d", indicator.Damage)
        picture := combat.damagePicture(text)

        x, y := matrix.Apply(float64(indicator.X), float64(indicator.Y))
        x += float64(indicator.Offset)
        y += damageStartY * combat.CameraScale + float64(indicator.OffsetY)
        y -= float64(indicator.Count) / damageRiseTicks

        place := placed{
            X: x * scale.ScaleAmount,
            Y: y * scale.ScaleAmount,
            HalfWidth: scale.Scale(float64(hud.Damage.Width(text))) / 2,
            HalfHeight: scale.Scale(float64(hud.Damage.Height())) / 2,
        }

        for range damagePlaceTries {
            moved := false
            for _, other := range taken {
                overlapX := place.HalfWidth + other.HalfWidth + gap - math.Abs(place.X - other.X)
                overlapY := place.HalfHeight + other.HalfHeight + gap - math.Abs(place.Y - other.Y)
                if overlapX <= 0 || overlapY <= 0 {
                    continue
                }

                // away from the other, the shorter way
                if overlapX <= overlapY {
                    if place.X < other.X || (place.X == other.X && index % 2 == 0) {
                        place.X -= overlapX
                    } else {
                        place.X += overlapX
                    }
                } else {
                    if place.Y <= other.Y {
                        place.Y -= overlapY
                    } else {
                        place.Y += overlapY
                    }
                }
                moved = true
                break
            }
            if !moved {
                break
            }
        }
        taken = append(taken, place)

        var options ebiten.DrawImageOptions
        options.GeoM.Translate(math.Round(place.X - float64(picture.Bounds().Dx()) / 2), math.Round(place.Y - float64(picture.Bounds().Dy()) / 2))
        if indicator.Life < damageFadeTicks {
            options.ColorScale.ScaleAlpha(float32(indicator.Life) / damageFadeTicks)
        }
        screen.DrawImage(picture, &options)
    }
}
