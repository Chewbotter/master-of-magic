package combat

// The numbers that rise from a unit when it is hurt. The original has none, so there is nothing to
// match: they are drawn like the rest of the original's text instead, in one of its fonts, in its
// palette's reds, with a black border of whole art pixels.

import (
    "fmt"
    "image/color"
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/display"
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
// numbers keep this many art pixels apart
const damageGap = 4
// how far from the middle of the unit a number can start, in art pixels each way. a group of
// numbers is a loose cluster
const damageSpreadX = 14
const damageSpreadY = 7
// A NUMBER GETS ITS PLACE ONCE, when it appears (placeDamageNumber), among the numbers that are
// there at that moment; after that it only rises, as fast as all others, so numbers that were
// apart stay apart. Before, every number looked for its place anew in every frame, against the
// numbers before it in the list: with many numbers at once (a volley of arrows) a number jumped
// whenever one before it rose past it or ran out (user, 2026-09-29: "several of them spasm and
// pop out oddly").
// places by chance that are tried for a new number. the area they are picked from grows to
// damageSpreadGrowth times the spread over the tries, so a crowd gets wider and not denser
const damagePlaceTries = 48
const damageSpreadGrowth = 3.0
// the size of a number for its place when the font is not there, in art pixels
const damageWidthGuess = 6
const damageHeightGuess = 6
// art pixels around the letters in the picture of a number, room for the border
const damagePictureMargin = 2
// ticks the number takes to rise one art pixel
const damageRiseTicks = 5
// the last ticks of its life the number fades out over
const damageFadeTicks = 10

// the numbers over units of the player are this much as bright as the ones over units of the
// enemy, which are the reds as they are (user, 2026-09-29: "leave the enemy damage numbers as
// bright red while making the friendly damage numbers a dimmer shade of red")
const damageOwnBrightness = 0.6

// false: the numbers are as large at every zoom
const DamageNumbersZoom = true

// screen pixels per art pixel of a number
func DamagePixel(cameraScale float64) float64 {
    if !DamageNumbersZoom {
        return scale.ScaleAmount
    }
    return math.Max(1, math.Round(cameraScale * scale.ScaleAmount))
}

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
func (combat *CombatScreen) damagePicture(text string, pixel float64) *ebiten.Image {
    key := fmt.Sprintf("%v@%v", text, pixel)
    if picture, ok := combat.damagePictures[key]; ok {
        return picture
    }

    styled := combat.Fonts.Hud.Damage
    width := int(float64(styled.Width(text) + damagePictureMargin * 2) * pixel)
    height := int(float64(styled.Height() + damagePictureMargin * 2) * pixel)

    picture := ebiten.NewImage(width, height)
    styled.Print(picture, damagePictureMargin, damagePictureMargin, font.FontOptions{Scale: pixel}, text)

    if combat.damagePictures == nil {
        combat.damagePictures = make(map[string]*ebiten.Image)
    }
    combat.damagePictures[key] = picture
    return picture
}

// how far a number has risen, in art pixels
func (indicator *DamageIndicator) risen() float64 {
    return float64(indicator.Count) / damageRiseTicks
}

// the middle of a number on the original's screen, in art pixels
func (indicator *DamageIndicator) middle() (float64, float64) {
    matrix := MakeBattlefieldMatrix()
    x, y := matrix.Apply(float64(indicator.X), float64(indicator.Y))
    return x + float64(indicator.Offset), y + damageStartY + float64(indicator.OffsetY) - indicator.risen()
}

// true if the two numbers are closer than they should be
func (indicator *DamageIndicator) touches(other *DamageIndicator) bool {
    x, y := indicator.middle()
    otherX, otherY := other.middle()
    apartX := float64(indicator.Width + other.Width) / 2 + damageGap
    apartY := float64(indicator.Height + other.Height) / 2 + damageGap
    return math.Abs(x - otherX) < apartX && math.Abs(y - otherY) < apartY
}

// gives a new number its place: by chance around its unit, where it touches no number that is
// there. pick gives a number from 0 up to but not its argument
func placeDamageNumber(indicator *DamageIndicator, others []DamageIndicator, pick func(int) int) {
    for try := range damagePlaceTries {
        grow := 1 + (damageSpreadGrowth - 1) * float64(try) / damagePlaceTries
        spreadX := int(damageSpreadX * grow)
        spreadY := int(damageSpreadY * grow)
        indicator.Offset = pick(spreadX * 2 + 1) - spreadX
        indicator.OffsetY = pick(spreadY * 2 + 1) - spreadY

        free := true
        for index := range others {
            if indicator.touches(&others[index]) {
                free = false
                break
            }
        }
        if free {
            return
        }
    }
}

func (combat *CombatScreen) drawDamageNumbers(screen *ebiten.Image) {
    hud := combat.Fonts.Hud
    // the setting "Damage numbers", see display/damagenumbers.go
    if hud == nil || hud.Damage == nil || !display.DamageNumbers() {
        return
    }

    matrix := combat.GetCameraMatrix()
    // the numbers are as large as the field is drawn, and so is everything about where they are
    // (user, 2026-09-27). screen pixels per art pixel of the field, whole so the letters are sharp
    // the picture of a number has whole pixels. while the zoom changes the field is between two
    // sizes: the numbers are then where and as large as the field has them, so they do not jump
    // from one whole size to the next
    wholePixel := DamagePixel(combat.CameraScale)
    pixel := wholePixel
    if DamageNumbersZoom {
        pixel = math.Max(1, combat.CameraScale * scale.ScaleAmount)
    }
    stretch := pixel / wholePixel

    type placed struct {
        // the middle, in screen pixels
        X, Y float64
    }

    for _, indicator := range combat.DamageIndicators {
        text := fmt.Sprintf("%d", indicator.Damage)
        picture := combat.damagePicture(text, wholePixel)

        x, y := matrix.Apply(float64(indicator.X), float64(indicator.Y))
        // from the middle of the tile, in art pixels of the field
        fromX := float64(indicator.Offset)
        fromY := damageStartY + float64(indicator.OffsetY) - indicator.risen()

        // its place was found when it appeared, see placeDamageNumber
        place := placed{
            X: x * scale.ScaleAmount + fromX * pixel,
            Y: y * scale.ScaleAmount + fromY * pixel,
        }

        var options ebiten.DrawImageOptions
        if stretch != 1 {
            options.GeoM.Scale(stretch, stretch)
            options.Filter = ebiten.FilterLinear
        }
        options.GeoM.Translate(math.Round(place.X - float64(picture.Bounds().Dx()) * stretch / 2), math.Round(place.Y - float64(picture.Bounds().Dy()) * stretch / 2))
        if indicator.Own {
            // the border is black and stays so
            options.ColorScale.Scale(damageOwnBrightness, damageOwnBrightness, damageOwnBrightness, 1)
        }
        if indicator.Life < damageFadeTicks {
            options.ColorScale.ScaleAlpha(float32(indicator.Life) / damageFadeTicks)
        }
        screen.DrawImage(picture, &options)
    }
}
