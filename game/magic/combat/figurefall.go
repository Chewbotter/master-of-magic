package combat

// Figures that are killed fall over and stay on the field as corpses: each is pushed back from
// where the blow came from and tips over backward around its feet, a little later or earlier than
// the others. Once it lies it darkens a little and stays for the rest of the battle, as the picture
// it had when it was killed. Corpses are drawn in the same order as everything else on the field,
// by where they lie.
// Not in the original, which shows a splash of blood on a lost figure that then is gone.
//
// ON WHOLE ART PIXELS (user, 2026-09-27). A figure is pushed back by a distance that is no whole
// number of art pixels, and it can die while it walks. Drawn where that puts it, its pixels lie
// between the pixels of the field and of the corpses next to it, which shows as thin gaps between
// them. A corpse is drawn on the nearest whole art pixel, while it goes down and where it lies. One
// that is tipped over is turned on a small picture of whole art pixels first, for the same reason.
//
// DEATH FRAMES (user, 2026-09-27): a figure that has the frames 7 and 8 in the replacement folder
// (game/magic/mod) does not tip over. It shows 7 while it goes down and 8 from then on. With only 8
// it shows 8 at once. It is still pushed back, darkens and loses its color.

import (
    "math"
    "math/rand/v2"

    "github.com/kazzmir/master-of-magic/game/magic/mod"
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
// the size of the picture a figure is turned on, in art pixels. a figure of 28 by 30 turned around
// its feet fits
const corpseTurnedSize = 72

// how bright a corpse is, how much of its color it keeps, and the seconds it takes to get there
// once it lies
const corpseBrightness = 0.65
const corpseSaturation = 0.3
// how far the black of a corpse is lifted to gray, of 1: its outline is dark gray, not black, which
// sets it apart from the figures that stand (user, 2026-09-27)
const corpseBlackLift = 0.17
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
    // the death frames of the figure, if it has them: on its way down, and lying. with them it does
    // not tip over
    Dying *ebiten.Image
    Dead *ebiten.Image
    // for a figure that is tipped over: the picture turned, in whole art pixels, its feet in the
    // middle. made again while it falls, kept once it lies
    turned *ebiten.Image
    lies bool
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
func (combat *CombatScreen) addCorpse(unit *ArmyUnit, frames []*ebiten.Image, frame int, figure int, count int, x float64, y float64) {
    points := unitview.CombatPoints(count)
    if len(frames) == 0 || figure < 0 || figure >= len(points) {
        return
    }

    picture := frames[min(max(0, frame), len(frames) - 1)]

    // the death frames of the replacement folder
    var dying *ebiten.Image
    var dead *ebiten.Image
    archive := unit.Unit.GetCombatLbxFile()
    entry := unit.Unit.GetCombatIndex(unit.Facing)
    if len(frames) > mod.FrameDead && mod.HasFrame(archive, entry, mod.FrameDead) {
        dead = frames[mod.FrameDead]
        dying = dead
        if mod.HasFrame(archive, entry, mod.FrameDying) {
            dying = frames[mod.FrameDying]
        }
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
        Dying: dying,
        Dead: dead,
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

        // on whole art pixels, see the top of the file
        x := math.Round(body.X + body.PushX * pushed)
        y := math.Round(body.Y + body.PushY * pushed)

        darkened := max(0, min(1, (seconds - figureFallTime) / corpseDarkenTime))
        brightness := float32(1 - (1 - corpseBrightness) * darkened)
        brightness *= fieldBrightness(int(x), int(y))
        saturation := 1 - (1 - corpseSaturation) * darkened

        out = append(out, fieldDrawable{
            Order: DrawOrder(int(math.Floor(x)), int(math.Floor(y))),
            Layer: layerFigure,
            Render: func() {
                picture := body.Picture
                offsetX, offsetY := unitview.FigureOffset(picture)

                if body.Dead != nil {
                    // its own pictures of dying, not the standing one tipped over
                    picture = body.Dying
                    if progress >= 1 {
                        picture = body.Dead
                    }
                    offsetX, offsetY = unitview.FigureOffset(picture)
                } else {
                    // tipped over on a picture of whole art pixels, its feet in the middle
                    if body.turned == nil {
                        body.turned = ebiten.NewImage(corpseTurnedSize, corpseTurnedSize)
                    }
                    if !body.lies {
                        body.turned.Clear()
                        var turn ebiten.DrawImageOptions
                        // the feet on the origin, so the figure turns around them
                        turn.GeoM.Translate(unitview.FigureOffset(picture))
                        turn.GeoM.Rotate(body.Angle * tipped)
                        turn.GeoM.Translate(corpseTurnedSize / 2, corpseTurnedSize / 2)
                        body.turned.DrawImage(picture, &turn)
                        body.lies = progress >= 1
                    }
                    picture = body.turned
                    offsetX, offsetY = -corpseTurnedSize / 2, -corpseTurnedSize / 2
                }

                var options colorm.DrawImageOptions
                options.GeoM.Translate(offsetX, offsetY)
                options.GeoM.Translate(x, y)
                options.GeoM.Concat(originalScreenMatrix(combat.GetCameraMatrix()))
                options.GeoM.Scale(scale.ScaleAmount, scale.ScaleAmount)

                var colors colorm.ColorM
                colors.ChangeHSV(0, saturation, float64(brightness))
                // lifts the dark end: black becomes gray, the bright colors hardly change
                lift := corpseBlackLift * darkened * float64(fieldBrightness(int(x), int(y)))
                colors.Scale(1 - lift, 1 - lift, 1 - lift, 1)
                colors.Translate(lift, lift, lift, 0)
                colorm.DrawImage(screen, picture, colors, &options)
            },
        })
    }

    return out
}
