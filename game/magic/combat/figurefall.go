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
    "image/color"
    "math"
    rand "github.com/kazzmir/master-of-magic/lib/chance"

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
    // the color it takes on as it lies, by what killed it
    HasTint bool
    Tint color.RGBA
    // it fades out, since this tick of the battle: its tile has too many. see corpsefade.go
    Fading bool
    FadeStart uint64
    // the blood that has come down on it, see corpsesplat.go
    Splats []corpseSplat
    // it never fades: the corpse of a unit that is one figure, see corpsefade.go
    Lasts bool
}

// how much of the color of what killed it a corpse takes on, 0 to 1
const corpseTintStrength = 0.45

// what the colors of a corpse are multiplied by to take on a color by a part of 0 to 1
func corpseTintParts(tint color.RGBA, part float64) (float64, float64, float64) {
    most := float64(max(tint.R, tint.G, tint.B, 1))
    mix := func(value uint8) float64 {
        return 1 - part + part * float64(value) / most
    }
    return mix(tint.R), mix(tint.G), mix(tint.B)
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

// the facing that looks the most the way of a direction on the screen
func facingToward(x float64, y float64) units.Facing {
    best := units.FacingDown
    bestPart := math.Inf(-1)
    for _, facing := range allFacings {
        faceX, faceY := facingOnScreen(facing)
        part := faceX * x + faceY * y
        if part > bestPart {
            bestPart = part
            best = facing
        }
    }
    return best
}

var allFacings = []units.Facing{
    units.FacingUp, units.FacingUpRight, units.FacingRight, units.FacingDownRight,
    units.FacingDown, units.FacingDownLeft, units.FacingLeft, units.FacingUpLeft,
}

// the ground is seen from above at an angle: what is up and down the screen is this much of what it is
const groundSquash = 0.5
// of how far a spell throws a figure is thrown at least this part
const throwLeast = 0.6

// which way and how far a figure a spell has killed is thrown: away from where the spell hit.
// figureX, figureY is where the figure stands on the original's screen
func throwOf(cause deathCause, figureX float64, figureY float64, chance float64, turn float64) (float64, float64, float64) {
    awayX := figureX - cause.X
    awayY := (figureY - cause.Y) / groundSquash
    distance := math.Hypot(awayX, awayY)
    if distance < 1 {
        // where it hit: any way
        awayX = math.Cos(turn * 2 * math.Pi)
        awayY = math.Sin(turn * 2 * math.Pi)
        distance = 1
    }

    far := cause.Throw * (throwLeast + (1 - throwLeast) * chance)
    return awayX / distance, awayY / distance * groundSquash, far
}

// x, y is where the unit of the figure is, in tiles. load gives the pictures of the unit for a facing
func (combat *CombatScreen) addCorpse(unit *ArmyUnit, load func(units.Facing) []*ebiten.Image, frame int, figure int, count int, x float64, y float64) {
    points := unitview.CombatPoints(count)
    if figure < 0 || figure >= len(points) {
        return
    }

    field := MakeBattlefieldMatrix()
    screenX, screenY := field.Apply(x, y)

    // away from what the unit faces, which is what hit it
    facing := unit.Facing
    faceX, faceY := facingOnScreen(facing)
    pushX, pushY := -faceX, -faceY
    push := figureFallPushMin + rand.Float64() * (figureFallPushMax - figureFallPushMin)

    // a spell throws it away from where it hit, and it lies facing where that came from
    cause, bySpell := combat.deathCauseOf(unit)
    if bySpell && cause.Throw > 0 {
        pushX, pushY, push = throwOf(cause, screenX + float64(points[figure].X), screenY + float64(points[figure].Y), rand.Float64(), rand.Float64())
        facing = facingToward(-pushX, -pushY)
        faceX, faceY = facingOnScreen(facing)
    }

    frames := load(facing)
    if len(frames) == 0 {
        return
    }

    picture := frames[min(max(0, frame), len(frames) - 1)]

    // the death frames of the replacement folder
    var dying *ebiten.Image
    var dead *ebiten.Image
    archive := unit.Unit.GetCombatLbxFile()
    entry := unit.Unit.GetCombatIndex(facing)
    if len(frames) > mod.FrameDead && mod.HasFrame(archive, entry, mod.FrameDead) {
        dead = frames[mod.FrameDead]
        dying = dead
        if mod.HasFrame(archive, entry, mod.FrameDying) {
            dying = frames[mod.FrameDying]
        }
    }

    angle := (figureFallAngleMin + rand.Float64() * (figureFallAngleMax - figureFallAngleMin)) * math.Pi / 180
    // backward: to the side it is pushed to. straight up or down it can be either side
    if faceX > 0 || (faceX == 0 && rand.N(2) == 0) {
        angle = -angle
    }

    tint, hasTint := combat.corpseColor(unit)

    combat.corpses = append(combat.corpses, corpse{
        Lasts: corpseLasts(unit.GetCount()),
        HasTint: hasTint,
        Tint: tint,
        Picture: picture,
        Dying: dying,
        Dead: dead,
        X: screenX + float64(points[figure].X),
        Y: screenY + float64(points[figure].Y),
        PushX: pushX * push,
        PushY: pushY * push,
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

    // the oldest of a tile that has too many fade, see corpsefade.go
    combat.fadeCorpses()

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
        brightness *= fieldBrightness(int(x), int(y)) * combat.cloudShade(int(x), int(y))
        saturation := 1 - (1 - corpseSaturation) * darkened

        out = append(out, fieldDrawable{
            Order: depthOrder(x, y, int(math.Floor(x)), int(math.Floor(y))),
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
                if body.HasTint {
                    // toward the color of what killed it, as bright as it was
                    red, green, blue := corpseTintParts(body.Tint, corpseTintStrength * darkened)
                    colors.Scale(red, green, blue, 1)
                }
                // gone while Tab is held, see groundmarks.go
                shows := corpseShows(body, combat.Counter) * combat.corpsesShow()
                if shows <= 0 {
                    return
                }
                if shows < 1 {
                    colors.Scale(shows, shows, shows, shows)
                }
                colorm.DrawImage(screen, picture, colors, &options)
                // what is marked on the ground lies over it, see groundmarks.go
                combat.drawMarksOnCorpse(screen, picture, options.GeoM, shows)

                // the blood on it stays red, see corpsesplat.go
                combat.drawCorpseSplats(screen, body, x + offsetX, y + offsetY, shows, fieldBrightness(int(x), int(y)))
            },
        })
    }

    return out
}
