package combat

// The numbers that rise from a unit when it is hurt. The original has none, so there is nothing to
// match: they are drawn like the rest of the original's text instead, in one of its fonts, in its
// palette's reds, with a black border of whole art pixels.

import (
    "fmt"
    "image"
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
// numbers keep this many art pixels apart. one that would touch another is put above it
const damageGap = 2
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

func (combat *CombatScreen) drawDamageNumbers(screen *ebiten.Image) {
    hud := combat.Fonts.Hud
    if hud == nil || hud.Damage == nil {
        return
    }

    matrix := combat.GetCameraMatrix()
    height := hud.Damage.Height()

    // the places taken so far, in screen pixels
    var taken []image.Rectangle

    for _, indicator := range combat.DamageIndicators {
        text := fmt.Sprintf("%d", indicator.Damage)
        width := hud.Damage.Width(text)

        // over the middle of the unit
        x, y := matrix.Apply(float64(indicator.X), float64(indicator.Y))
        y += damageStartY * combat.CameraScale
        y -= float64(indicator.Count) / damageRiseTicks

        place := func() image.Rectangle {
            left := int(math.Round((x - float64(width / 2) - damageGap) * scale.ScaleAmount))
            top := int(math.Round((y - damageGap) * scale.ScaleAmount))
            return image.Rect(left, top, left + scale.Scale(width + damageGap * 2), top + scale.Scale(height + damageGap * 2))
        }

        for tries := 0; tries < len(combat.DamageIndicators); tries++ {
            rect := place()
            free := true
            for _, other := range taken {
                if rect.Overlaps(other) {
                    free = false
                    // just above the one in the way
                    y = float64(other.Min.Y) / scale.ScaleAmount - float64(height + damageGap)
                    break
                }
            }
            if free {
                break
            }
        }
        taken = append(taken, place())

        var options ebiten.DrawImageOptions
        if indicator.Life < damageFadeTicks {
            options.ColorScale.ScaleAlpha(float32(indicator.Life) / damageFadeTicks)
        }

        hud.Damage.PrintFine(screen, x, y, font.FontOptions{Justify: font.FontJustifyCenter, Scale: scale.ScaleAmount, Options: &options}, text)
    }
}
