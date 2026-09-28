package combat

// Blood. The original shows it, the fork did not. Made of particles here (user, 2026-09-28).
//
// WHEN, as the original has it: a unit that is hurt in a fight or by a missile bleeds, more the
// more it is hurt for its size: damage plus half its figures, over twice its figures, from 1 to
// 5. (ReMoM: MoM/src/Combat.c Set_Gibs, called by Melee_Animation and for missiles. The original
// plays one of 5 pictures of 4 frames over every figure, CMBTCITY 24 to 28. The code here is
// ours, and so is how it looks.) Spells have effects of their own and draw no blood.
//
// HOW: drops fly off every figure of the unit, away from what the unit faces, which is what hit
// it. They rise a little, fall, and where one comes down on the ground it leaves a stain that
// stays for the battle. Drops and stains are single art pixels.
//
// Not everything bleeds: what is undead or of the realm of death gives off dark dust, what has
// no body a pale mist, and neither leaves stains.
//
// The values are in effects.txt, in the parts [blood], [blood undead] and [blood spirit].

import (
    "image"
    "image/color"
    "math"
    "math/rand/v2"
    "strconv"
    "strings"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/mod"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/game/magic/unitview"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: no blood
const Blood = true

// the most the original bleeds
const bloodMost = 5
// no more drops than this for one hit
const bloodDropsMost = 160
// the ground around the field that can take stains, in tiles
const bloodFieldMargin = BattlefieldBorder
// a spell that has hit a unit this many ticks ago or less is what hurt it: no blood
const bloodSpellTicks = 2

const (
    bloodPart = "blood"
    bloodUndeadPart = "blood undead"
    bloodSpiritPart = "blood spirit"
)

type BloodValues struct {
    // drops for every figure and every step of how much it bleeds
    Drops float64
    // art pixels a second: how fast they fly, and how fast they rise at first
    Speed float64
    Lift float64
    // how fast they get faster downward
    Gravity float64
    // seconds they live at most
    Life float64
    // how far over the ground they start, art pixels
    Height float64
    // how wide they fly apart, in degrees to each side of the way they fly
    Spread float64
    // the colors a drop goes through
    Colors []color.RGBA
    // the color of the stain a drop leaves, and how much of it shows 0 to 1. 0: none
    Stain color.RGBA
    StainStrength float64
}

func gameBlood(part string) BloodValues {
    switch part {
        case bloodUndeadPart:
            return BloodValues{
                Drops: 2, Speed: 30, Lift: 30, Gravity: 120, Life: 0.7, Height: 9, Spread: 70,
                Colors: colors("8a7a90 5a4a66 3a2e46 221a2a"),
            }
        case bloodSpiritPart:
            return BloodValues{
                Drops: 2, Speed: 22, Lift: 35, Gravity: -30, Life: 0.8, Height: 10, Spread: 180,
                Colors: colors("ffffff c8e8ff 88b8e8 5078b0"),
            }
    }

    return BloodValues{
        Drops: 3, Speed: 42, Lift: 48, Gravity: 260, Life: 0.9, Height: 9, Spread: 50,
        Colors: colors("ff3030 d81818 a80c0c 780606"),
        Stain: color.RGBA{R: 0x6a, G: 0x08, B: 0x08, A: 0xff},
        StainStrength: 0.8,
    }
}

var bloodValueNames = []string{"drops", "speed", "lift", "gravity", "life", "height", "spread", "colors", "stain", "stain-strength"}

func (values *BloodValues) text(name string) string {
    number := func(value float64) string {
        return strconv.FormatFloat(value, 'g', -1, 64)
    }

    switch name {
        case "drops": return number(values.Drops)
        case "speed": return number(values.Speed)
        case "lift": return number(values.Lift)
        case "gravity": return number(values.Gravity)
        case "life": return number(values.Life)
        case "height": return number(values.Height)
        case "spread": return number(values.Spread)
        case "colors": return colorsText(values.Colors)
        case "stain": return colorText(values.Stain)
        case "stain-strength": return number(values.StainStrength)
    }
    return ""
}

func (values *BloodValues) set(name string, text string) bool {
    switch name {
        case "colors":
            many := colors(text)
            if len(many) > 0 {
                values.Colors = many
            }
            return len(many) > 0
        case "stain":
            value, ok := parseColor(text)
            if ok {
                values.Stain = value
            }
            return ok
    }

    number, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
    if err != nil {
        return false
    }

    switch name {
        case "drops": values.Drops = number
        case "speed": values.Speed = number
        case "lift": values.Lift = number
        case "gravity": values.Gravity = number
        case "life": values.Life = number
        case "height": values.Height = number
        case "spread": values.Spread = number
        case "stain-strength": values.StainStrength = number
        default:
            return false
    }
    return true
}

// the values of a kind of blood: the ones of the game, then the ones of the file
func bloodValues(part string, file map[string]map[string]string) BloodValues {
    values := gameBlood(part)
    for name, text := range file[part] {
        values.set(name, text)
    }
    return values
}

// the parts of the file of values that are about blood, with the values of the game
func bloodTemplate() string {
    var out strings.Builder
    out.WriteString(`# Blood: what flies off a unit that is hurt in a fight or by a missile. [blood undead] is for
# what is undead or of the realm of death, [blood spirit] for what has no body.
#   drops           drops for every figure, times how much the unit bleeds (1 to 5)
#   speed           how fast they fly
#   lift            how fast they rise at first
#   gravity         how fast they get faster downward. below 0 they rise
#   life            how long they live at most
#   height          how far over the ground they start
#   spread          how wide they fly apart, in degrees to each side
#   colors          their colors
#   stain           the color of the stain a drop leaves on the ground, which stays
#   stain-strength  how much of the stain shows, 0 to 1. 0: no stains

`)

    for _, part := range []string{bloodPart, bloodUndeadPart, bloodSpiritPart} {
        values := gameBlood(part)
        out.WriteString("[" + part + "]\n")
        for _, name := range bloodValueNames {
            out.WriteString(name + " = " + values.text(name) + "\n")
        }
        out.WriteString("\n")
    }

    return out.String()
}

// how much a unit bleeds that takes damage, 0 to 5, as the original has it
func bloodAmount(damage int, figures int) int {
    if damage <= 0 {
        return 0
    }
    if figures <= 0 {
        return 3
    }

    amount := (damage + figures / 2) / (figures * 2)
    return max(1, min(bloodMost, amount))
}

// which blood a unit has
func bloodKind(unit *ArmyUnit) string {
    switch {
        case unit.HasAbility(data.AbilityNonCorporeal): return bloodSpiritPart
        case unit.IsUndead() || unit.GetRealm() == data.DeathMagic: return bloodUndeadPart
    }
    return bloodPart
}

// the stains on the ground: a picture of the whole field on the original's screen, one pixel an
// art pixel
type bloodStains struct {
    Picture *ebiten.Image
    // the place of its top left corner on the original's screen
    X int
    Y int
    Count int
}

// where the field with the ground around it lies on the original's screen
func bloodFieldBounds() image.Rectangle {
    field := MakeBattlefieldMatrix()
    low := float64(-bloodFieldMargin)
    high := float64(BattlefieldWidth + bloodFieldMargin)

    bounds := image.Rectangle{}
    first := true
    for _, corner := range [][2]float64{{low, low}, {high, low}, {low, high}, {high, high}} {
        x, y := field.Apply(corner[0], corner[1])
        point := image.Rect(int(math.Floor(x)), int(math.Floor(y)), int(math.Ceil(x)) + 1, int(math.Ceil(y)) + 1)
        if first {
            bounds = point
            first = false
        } else {
            bounds = bounds.Union(point)
        }
    }
    return bounds
}

func (combat *CombatScreen) stain(x int, y int, tint color.RGBA) {
    stains := &combat.effects.Stains
    if stains.Picture == nil {
        bounds := bloodFieldBounds()
        stains.Picture = ebiten.NewImage(bounds.Dx(), bounds.Dy())
        stains.X = bounds.Min.X
        stains.Y = bounds.Min.Y
    }

    at := image.Pt(x - stains.X, y - stains.Y)
    if !at.In(stains.Picture.Bounds()) {
        return
    }

    pixel := stains.Picture.SubImage(image.Rect(at.X, at.Y, at.X + 1, at.Y + 1)).(*ebiten.Image)
    pixel.WritePixels([]byte{tint.R, tint.G, tint.B, 0xff})
    stains.Count += 1
}

// the stains, on the ground and under what stands on it
func (combat *CombatScreen) drawStains(screen *ebiten.Image) {
    stains := &combat.effects.Stains
    if stains.Picture == nil || stains.Count == 0 {
        return
    }

    var options ebiten.DrawImageOptions
    options.GeoM.Translate(float64(stains.X), float64(stains.Y))
    options.GeoM.Concat(originalScreenMatrix(combat.GetCameraMatrix()))
    options.ColorScale.ScaleAlpha(float32(min(1, combat.effects.StainStrength)))
    scale.DrawScaled(screen, stains.Picture, &options)
}

// a unit is hurt: blood
func (combat *CombatScreen) bleed(unit *ArmyUnit, damage int) {
    if !Blood || !SpellEffects || unit == nil || damage <= 0 {
        return
    }

    effects := &combat.effects

    // a spell has effects of its own
    cause, ok := effects.Causes[unit]
    if ok && effects.Tick - cause.Tick <= bloodSpellTicks {
        return
    }

    figures := max(1, unit.VisibleFigures() + unit.LostUnits)
    amount := bloodAmount(damage, figures)
    if amount <= 0 {
        return
    }

    file, _ := mod.Effects()
    values := bloodValues(bloodKind(unit), file)
    if values.Drops <= 0 || len(values.Colors) == 0 {
        return
    }
    effects.StainStrength = values.StainStrength

    field := MakeBattlefieldMatrix()
    x, y := float64(unit.X), float64(unit.Y)
    if unit.Moving {
        x, y = unit.MoveX, unit.MoveY
    }
    screenX, screenY := field.Apply(x, y)

    // away from what it faces, which is what hit it
    faceX, faceY := facingOnScreen(unit.Facing)
    away := math.Atan2(-faceY / groundSquash, -faceX)

    points := unitview.CombatPoints(figures)
    drops := int(math.Round(values.Drops * float64(amount)))
    drops = max(1, min(drops, bloodDropsMost / len(points)))

    for _, point := range points {
        effects.Particles.emitSpray(
            screenX + float64(point.X), screenY + float64(point.Y), values.Height, drops,
            away, values.Spread * math.Pi / 180, values.Speed, values.Lift, values.Gravity, values.Life,
            values.Colors, values.StainStrength > 0, values.Stain)
    }
}

// what is thrown one way: particles that fly off over the ground within an angle to each side of
// a direction, up first, and come down. a particle that leaves a stain ends where it comes down
func (system *particleSystem) emitSpray(x float64, ground float64, height float64, count int, direction float64, spread float64, speed float64, lift float64, gravity float64, life float64, ramp []color.RGBA, stains bool, stain color.RGBA) {
    for range count {
        angle := direction + randomPart(-spread, spread)
        own := speed * randomPart(0.3, 1)
        rise := lift * randomPart(0.4, 1)

        each := particle{
            X: x + randomPart(-2, 2),
            Y: ground - height + randomPart(-3, 3),
            SpeedX: math.Cos(angle) * own,
            SpeedY: math.Sin(angle) * own * particleGroundSquash - rise,
            Gravity: gravity,
            Life: life * randomPart(0.6, 1),
            Colors: ramp,
        }

        if gravity > 0 {
            each.HasGround = true
            // where it comes down lies further up or down the screen by where it flew to
            each.Ground = ground + math.Sin(angle) * own * particleGroundSquash * life * 0.5 + rand.Float64() * 2
            each.Stains = stains
            each.Stain = stain
        }

        system.add(each)
    }
}
