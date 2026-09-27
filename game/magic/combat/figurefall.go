package combat

// Figures that are killed fall over and stay on the field as corpses: each is pushed back from
// where the blow came from and tips over backward around its feet, a little later or earlier than
// the others. Once it lies it darkens a little and stays for the rest of the battle, as the picture
// it had when it was killed. Corpses are drawn in the same order as everything else on the field,
// by where they lie.
// Not in the original, which shows a splash of blood on a lost figure that then is gone.
//
// PLANNED (user, 2026-09-27): pictures of death poses, made by the user, to lie here in place of
// the tipped over standing picture. addCorpse is where a corpse gets its picture.

import (
    "math"
    "math/rand/v2"

    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    "github.com/kazzmir/master-of-magic/game/magic/unitview"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/colorm"
)

// turns it off: lost figures fade where they stand, in a color
const FigureFall = true

// seconds a figure takes to fall
const figureFallTime = 0.3
// the latest a figure starts to fall after it was lost, in seconds
const figureFallDelayMax = 0.12
// how far a figure is pushed back, in art pixels
const figureFallPushMin = 4.0
const figureFallPushMax = 9.0
// how far it tips over, in degrees. 90 is flat on its back
const figureFallAngleMin = 75.0
const figureFallAngleMax = 100.0
// how bright a corpse is, how much of its color it keeps, and the seconds it takes to get there
// once it lies
const corpseBrightness = 0.65
const corpseSaturation = 0.3
const corpseDarkenTime = 0.5

// the way a unit faces on the screen, across and down
func facingOnScreen(facing units.Facing) (float64, float64) {
    // the sides of a tile run 2 across for 1 down
    diagonalX := 2 / math.Sqrt(5)
    diagonalY := 1 / math.Sqrt(5)

    switch facing {
        case units.FacingUp: return 0, -1
        case units.FacingUpRight: return diagonalX, -diagonalY
        case units.FacingRight: return 1, 0
        case units.FacingDownRight: return diagonalX, diagonalY
        case units.FacingDown: return 0, 1
        case units.FacingDownLeft: return -diagonalX, diagonalY
        case units.FacingLeft: return -1, 0
        case units.FacingUpLeft: return -diagonalX, -diagonalY
    }

    return 1, 0
}

type corpse struct {
    // the picture of the figure when it was killed
    Picture *ebiten.Image
    // where its feet stood, on the original's screen, in art pixels
    X float64
    Y float64
    // art pixels
    PushX float64
    PushY float64
    // radians, to the side it falls to
    Angle float64
    // the tick the figure was killed
    Start uint64
    // seconds
    Delay float64
}

// true the first time it is called for a figure that was lost: the figure is to become a corpse
func (combat *CombatScreen) figureJustLost(unit *ArmyUnit, figure int, count int, lost bool) bool {
    state := combat.figuresOf(unit, count, float64(unit.X), float64(unit.Y))

    if unit.LostUnits == 0 {
        // the places of lost figures are free again
        for index := unit.VisibleFigures(); index < len(state.Figures); index++ {
            state.Figures[index].Fallen = false
        }
    }

    each := &state.Figures[figure]
    if !lost {
        each.Fallen = false
        return false
    }

    if each.Fallen {
        return false
    }

    each.Fallen = true
    return true
}

// x, y is where the unit of the figure is, in tiles
func (combat *CombatScreen) addCorpse(unit *ArmyUnit, picture *ebiten.Image, figure int, count int, x float64, y float64) {
    points := unitview.CombatPoints(count)
    if picture == nil || figure < 0 || figure >= len(points) {
        return
    }

    field := MakeBattlefieldMatrix()
    screenX, screenY := field.Apply(x, y)

    // away from what the unit faces, which is what hit it
    faceX, faceY := facingOnScreen(unit.Facing)
    push := figureFallPushMin + rand.Float64() * (figureFallPushMax - figureFallPushMin)

    angle := (figureFallAngleMin + rand.Float64() * (figureFallAngleMax - figureFallAngleMin)) * math.Pi / 180
    // backward: to the side it is pushed to. straight up or down it can be either side
    if faceX > 0 || (faceX == 0 && rand.N(2) == 0) {
        angle = -angle
    }

    combat.corpses = append(combat.corpses, corpse{
        Picture: picture,
        X: screenX + float64(points[figure].X),
        Y: screenY + float64(points[figure].Y),
        PushX: -faceX * push,
        PushY: -faceY * push,
        Angle: angle,
        Start: combat.Counter,
        Delay: rand.Float64() * figureFallDelayMax,
    })
}

// seconds since a corpse started to fall
func (combat *CombatScreen) corpseTime(body *corpse) float64 {
    return float64(combat.Counter - body.Start) / float64(max(1, ebiten.TPS())) - body.Delay
}

func (combat *CombatScreen) corpseDrawables(screen *ebiten.Image) []fieldDrawable {
    var out []fieldDrawable

    for index := range combat.corpses {
        body := &combat.corpses[index]

        seconds := combat.corpseTime(body)
        progress := max(0, min(1, seconds / figureFallTime))
        // pushed fast and slowing, tipping slow and faster
        pushed := 1 - (1 - progress) * (1 - progress)
        tipped := progress * progress

        x := body.X + body.PushX * pushed
        y := body.Y + body.PushY * pushed

        darkened := max(0, min(1, (seconds - figureFallTime) / corpseDarkenTime))
        brightness := float32(1 - (1 - corpseBrightness) * darkened)
        brightness *= fieldBrightness(int(x), int(y))
        saturation := 1 - (1 - corpseSaturation) * darkened

        out = append(out, fieldDrawable{
            Order: DrawOrder(int(math.Floor(x)), int(math.Floor(y))),
            Layer: layerFigure,
            Render: func() {
                var options colorm.DrawImageOptions
                // the feet on the origin, so the figure turns around them
                options.GeoM.Translate(unitview.FigureOffset(body.Picture))
                options.GeoM.Rotate(body.Angle * tipped)
                options.GeoM.Translate(x, y)
                options.GeoM.Concat(originalScreenMatrix(combat.GetCameraMatrix()))
                options.GeoM.Scale(scale.ScaleAmount, scale.ScaleAmount)

                var colors colorm.ColorM
                colors.ChangeHSV(0, saturation, float64(brightness))
                colorm.DrawImage(screen, body.Picture, colors, &options)
            },
        })
    }

    return out
}
