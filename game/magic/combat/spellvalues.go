package combat

// The values of the effects of spells: what goes with the pictures of a spell, which are drawn by
// hand. Not in the original (user, 2026-09-28).
//
// Every spell has its values by the name of its pictures (game/magic/mod/spells.go). The game has
// values of its own for them, below. effects.txt in the replacement folder can give others, see
// game/magic/mod/effects.go; EffectsTemplate writes such a file with the values of the game.

import (
    "fmt"
    "image/color"
    "slices"
    "strconv"
    "strings"

    "github.com/kazzmir/master-of-magic/game/magic/mod"
)

type SpellValues struct {
    // how much of the light of the picture is added on top of it, 0 to 1
    Emissive float64

    // in flight: particles a second, how long they live in seconds, their own speed in art pixels
    // a second, how fast they get faster downward (below 0: they rise), their colors
    TrailRate float64
    TrailLife float64
    TrailSpeed float64
    TrailGravity float64
    TrailColors []color.RGBA

    // where it hits: how many particles, how fast they fly apart, how fast they go up at first,
    // how long they live, how fast they get faster downward, their colors
    BurstCount int
    BurstSpeed float64
    BurstLift float64
    BurstLife float64
    BurstGravity float64
    BurstColors []color.RGBA
    // how far over the ground what hits hits, in art pixels
    ImpactHeight float64

    // what is thrown up from the ground where it hits, dirt: how many particles, how fast they fly
    // apart, how fast they go up at first, how long they live, how fast they get faster downward,
    // their colors. they start on the ground, not where it hits
    DebrisCount int
    DebrisSpeed float64
    DebrisLift float64
    DebrisLife float64
    DebrisGravity float64
    DebrisColors []color.RGBA

    // sparks that come off the pixels of the picture of a spell while it shows, in flight or not:
    // how many a second, their own speed in art pixels a second, how long they live, how fast they
    // get faster downward, their colors
    SparkRate float64
    SparkSpeed float64
    SparkLife float64
    SparkGravity float64
    SparkColors []color.RGBA

    // the unit that is hit shows in one color: for how long in seconds, how much of it 0 to 1
    FlashTime float64
    FlashStrength float64
    FlashColor color.RGBA

    // the battle stands still when it hits, seconds
    HitStop float64
    // the view shakes: by how many art pixels at most, for how long in seconds
    Shake float64
    ShakeTime float64

    // light that runs over the ground from where it hits: how many tiles far, for how long in
    // seconds, how much of it 0 to 1
    PulseRadius float64
    PulseTime float64
    PulseStrength float64
    PulseColor color.RGBA

    // the mark it leaves on the ground: the kind of the pictures of the game (scorch, crater,
    // frost, spark, none), and how much of it shows 0 to 1. pictures of the replacement folder
    // go before the kind, see decals.go
    Decal string
    DecalStrength float64

    // the light of the spell, see spelllight.go: how dark the field gets while it plays 0 to 1,
    // how far its light goes in art pixels, how much of the dark it takes away in its middle 0 to
    // 1, by how many art pixels its light flickers, how long it stays where the spell has hit in
    // seconds, how far from it the shadows of units turn away from it in art pixels
    Dark float64
    LightRadius float64
    LightStrength float64
    LightFlicker float64
    LightLinger float64
    ShadowReach float64
    // how much the rims of the figures in its light are lit on the side that faces it, 0 to 1,
    // and in which color
    RimLight float64
    RimColor color.RGBA
    // how much lighter the ground in its light gets, in the color of the rims: 0 not at all, 1 by
    // all of that color. see groundlight.go
    GroundLight float64

    // the color the figures it kills take on as they lie, see figurefall.go. without one they
    // are gray
    HasCorpseColor bool
    CorpseColor color.RGBA

    // the dark of the field has a little of this color, see spelllight.go
    DarkColor color.RGBA
    // what the spell does to its target is done when it hits, and not when its pictures have
    // played to their end
    ResolveAtImpact bool
    // how far the figures it kills are thrown, away from where it hit, in art pixels. 0: as far
    // as in a fight, and backward
    Throw float64
}

var fireDark = color.RGBA{R: 0x48, G: 0x0c, B: 0x04, A: 0xff}
var iceDark = color.RGBA{R: 0x06, G: 0x18, B: 0x48, A: 0xff}
var black = color.RGBA{A: 0xff}
// the light of fire and of ice on the rims of figures
var fireRim = color.RGBA{R: 0xff, G: 0xa8, B: 0x50, A: 0xff}
var iceRim = color.RGBA{R: 0x90, G: 0xd8, B: 0xff, A: 0xff}

// the figures that are killed in a fight, by no spell, take this color on: blood
var combatCorpseColor = color.RGBA{R: 0xa0, G: 0x28, B: 0x28, A: 0xff}
// false: they are gray, and the blood on them is what is red. see corpsesplat.go. the file of
// values can still give them a color
const combatCorpseTinted = false
// the part of the file of values that is about the fight and no spell
const combatPart = "combat"

var burnedColor = color.RGBA{R: 0x8a, G: 0x58, B: 0x2c, A: 0xff}
var frozenColor = color.RGBA{R: 0x50, G: 0x88, B: 0xd0, A: 0xff}

func colors(text string) []color.RGBA {
    var out []color.RGBA
    for _, each := range strings.Fields(text) {
        value, ok := parseColor(each)
        if ok {
            out = append(out, value)
        }
    }
    return out
}

// a color as six letters and digits: ff8000
func parseColor(text string) (color.RGBA, bool) {
    text = strings.TrimPrefix(strings.TrimSpace(text), "#")
    if len(text) != 6 {
        return color.RGBA{}, false
    }

    value, err := strconv.ParseUint(text, 16, 32)
    if err != nil {
        return color.RGBA{}, false
    }

    return color.RGBA{R: uint8(value >> 16), G: uint8(value >> 8), B: uint8(value), A: 0xff}, true
}

func colorText(value color.RGBA) string {
    return fmt.Sprintf("%02x%02x%02x", value.R, value.G, value.B)
}

func colorsText(values []color.RGBA) string {
    var out []string
    for _, value := range values {
        out = append(out, colorText(value))
    }
    return strings.Join(out, " ")
}

var white = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}

var fireColors = colors("fff4b0 ffd040 ff9010 e04808 902008 401008")
var iceColors = colors("ffffff c8f4ff 70d0ff 3088f0 1848b0 102060")
var doomColors = colors("ffffff ffc0a0 ff5030 c01818 700818 300410")
var lightningColors = colors("ffffff e0ecff a0c0ff 5080f0 2040a0")
// the sparks where a Lightning Bolt strikes: blue from the start (user, 2026-09-30: "blue spark
// particles at the impact tile")
var lightningSparkColors = colors("d8ecff 90c4ff 4c8cff 2a5ce0 16309a")
// dirt: one dark brown for every ground (user: "It doesn't need to be authored to match the
// biome- a dark brown should work anywhere")
var dirtColors = colors("5c3c22 4c301a 3c2412 2e1a0c")
// the sparks that come off the bolt: bright (user, 2026-09-30: "I also don't see any bright blue
// sparks off of the lightning bolt itself")
var boltSparkColors = colors("ffffff c8f6ff 7ee0ff 36b4ff 1a78e0")
var lifeColors = colors("ffffff fff8c0 ffe070 c0a040")
var deathColors = colors("d0a0ff 8040c0 401870 200838")

// all spells
func plainValues() SpellValues {
    return SpellValues{
        Emissive: 0.25,
        ImpactHeight: 8,
        FlashStrength: 0.85,
        FlashColor: white,
        PulseColor: white,
        Decal: DecalNone,
        DecalStrength: 0.85,
        Dark: 0.15,
        DarkColor: black,
        LightRadius: 45,
        LightStrength: 1,
        LightLinger: 0.25,
        ShadowReach: 60,
        RimLight: 0.65,
        RimColor: white,
        GroundLight: 0.45,
    }
}

// a bolt that flies in and hits
func boltValues(ramp []color.RGBA, light color.RGBA) SpellValues {
    values := plainValues()
    values.TrailRate = 70
    values.TrailLife = 0.45
    values.TrailSpeed = 9
    values.TrailGravity = -18
    values.TrailColors = ramp
    values.BurstCount = 45
    values.BurstSpeed = 60
    values.BurstLift = 55
    values.BurstLife = 0.8
    values.BurstGravity = 220
    values.BurstColors = ramp
    values.FlashTime = 0.12
    values.HitStop = 0.07
    values.Shake = 2
    values.ShakeTime = 0.22
    values.PulseRadius = 2
    values.PulseTime = 0.35
    values.PulseStrength = 0.3
    values.PulseColor = light
    values.Decal = "scorch"
    values.Dark = 0.28
    values.ResolveAtImpact = true
    values.Throw = 12
    values.LightRadius = 75
    values.LightFlicker = 2
    values.LightLinger = 0.6
    values.ShadowReach = 110
    return values
}

// a spell that hurts a unit where it stands: the unit shows in one color
func hurtValues() SpellValues {
    values := plainValues()
    values.FlashTime = 0.1
    values.Dark = 0.2
    values.ResolveAtImpact = true
    values.Throw = 5
    values.LightRadius = 55
    values.LightLinger = 0.4
    values.ShadowReach = 80
    return values
}

func burns(values SpellValues) SpellValues {
    values.HasCorpseColor = true
    values.CorpseColor = burnedColor
    values.DarkColor = fireDark
    values.RimColor = fireRim
    return values
}

func freezes(values SpellValues) SpellValues {
    values.HasCorpseColor = true
    values.CorpseColor = frozenColor
    values.DarkColor = iceDark
    values.RimColor = iceRim
    return values
}

// the values of the game, by the name of the pictures of a spell
func gameValues(name string) SpellValues {
    switch name {
        case "Fire Bolt":
            return burns(boltValues(fireColors, color.RGBA{R: 0xff, G: 0x90, B: 0x20, A: 0xff}))
        case "Fireball":
            values := boltValues(fireColors, color.RGBA{R: 0xff, G: 0x90, B: 0x20, A: 0xff})
            values.TrailRate = 110
            values.BurstCount = 90
            values.BurstSpeed = 80
            values.BurstLift = 70
            values.HitStop = 0.1
            values.Shake = 3
            values.ShakeTime = 0.3
            values.PulseRadius = 3
            values.PulseStrength = 0.4
            values.Decal = "crater"
            values.LightRadius = 95
            values.ShadowReach = 130
            values.Throw = 24
            return burns(values)
        case "Ice Bolt":
            values := boltValues(iceColors, color.RGBA{R: 0x70, G: 0xd0, B: 0xff, A: 0xff})
            // what comes off it sinks
            values.TrailGravity = 25
            values.Decal = "frost"
            values.Throw = 8
            return freezes(values)
        case "Doom Bolt":
            values := boltValues(doomColors, color.RGBA{R: 0xff, G: 0x50, B: 0x30, A: 0xff})
            values.BurstCount = 70
            values.Shake = 3
            values.PulseRadius = 3
            values.Decal = "crater"
            values.Throw = 20
            return burns(values)
        case "Lightning Bolt":
            values := boltValues(lightningColors, color.RGBA{R: 0xc0, G: 0xd8, B: 0xff, A: 0xff})
            // it does not fly in
            values.TrailRate = 0
            // blue sparks from the ground where it strikes
            values.BurstCount = 40
            values.BurstSpeed = 90
            values.BurstLift = 45
            values.BurstLife = 0.45
            values.BurstColors = lightningSparkColors
            values.ImpactHeight = 2
            // and dirt thrown up, straight as by a mortar (user, 2026-09-30: "more and have them
            // fly more vertically, like a mortar blast"; first 26 with 28 to the side and 120 up)
            values.DebrisCount = 64
            values.DebrisSpeed = 10
            values.DebrisLift = 175
            values.DebrisLife = 1.2
            values.DebrisGravity = 320
            values.DebrisColors = dirtColors
            // sparks off the bolt while it shows
            values.SparkRate = 450
            values.SparkSpeed = 65
            values.SparkLife = 0.45
            values.SparkGravity = 90
            values.SparkColors = boltSparkColors
            values.FlashTime = 0.1
            values.HitStop = 0.06
            values.Decal = "spark"
            // it strikes down: not far
            values.Throw = 5
            return values
        case "Warp Lightning":
            values := boltValues(lightningColors, color.RGBA{R: 0xc0, G: 0xd8, B: 0xff, A: 0xff})
            values.TrailRate = 0
            values.BurstCount = 20
            values.BurstLife = 0.4
            values.HitStop = 0
            values.Shake = 1
            values.PulseRadius = 0
            values.Decal = "spark"
            values.Throw = 4
            return values
        case "Flame Strike":
            // on every unit of a side: no standing still and little shaking, or it adds up
            values := hurtValues()
            values.BurstCount = 25
            values.BurstSpeed = 45
            values.BurstLift = 60
            values.BurstLife = 0.7
            values.BurstGravity = 220
            values.BurstColors = fireColors
            values.Shake = 1
            values.ShakeTime = 0.2
            values.Decal = "scorch"
            values.Throw = 10
            return burns(values)
        case "Star Fires", "Dispel Evil":
            values := hurtValues()
            values.BurstCount = 20
            values.BurstSpeed = 40
            values.BurstLift = 50
            values.BurstLife = 0.6
            values.BurstGravity = 200
            values.BurstColors = lifeColors
            return values
        case "Life Drain", "Death Spell":
            values := hurtValues()
            values.FlashColor = color.RGBA{R: 0x80, G: 0x40, B: 0xc0, A: 0xff}
            values.BurstCount = 20
            values.BurstSpeed = 30
            values.BurstLift = 40
            values.BurstLife = 0.7
            values.BurstGravity = 120
            values.BurstColors = deathColors
            return values
        case "Psionic Blast", "Disintegrate", "Petrify", "Cracks Call":
            return hurtValues()
    }

    return plainValues()
}

// the names of the values in the file, in the order the template has them
var valueNames = []string{
    "emissive",
    "trail-rate", "trail-life", "trail-speed", "trail-gravity", "trail-colors",
    "burst-count", "burst-speed", "burst-lift", "burst-life", "burst-gravity", "burst-colors", "impact-height",
    "debris-count", "debris-speed", "debris-lift", "debris-life", "debris-gravity", "debris-colors",
    "spark-rate", "spark-speed", "spark-life", "spark-gravity", "spark-colors",
    "flash-time", "flash-strength", "flash-color",
    "hit-stop", "shake", "shake-time",
    "pulse-radius", "pulse-time", "pulse-strength", "pulse-color",
    "decal", "decal-strength",
    "dark", "light-radius", "light-strength", "light-flicker", "light-linger", "shadow-reach", "rim-light", "rim-color", "ground-light",
    "corpse-color",
    "dark-color", "resolve", "throw",
}

const resolveImpact = "impact"
const resolveEnd = "end"

const noColor = "none"

// a value as the file has it
func (values *SpellValues) text(name string) string {
    number := func(value float64) string {
        return strconv.FormatFloat(value, 'g', -1, 64)
    }

    switch name {
        case "emissive": return number(values.Emissive)
        case "trail-rate": return number(values.TrailRate)
        case "trail-life": return number(values.TrailLife)
        case "trail-speed": return number(values.TrailSpeed)
        case "trail-gravity": return number(values.TrailGravity)
        case "trail-colors": return colorsText(values.TrailColors)
        case "burst-count": return strconv.Itoa(values.BurstCount)
        case "burst-speed": return number(values.BurstSpeed)
        case "burst-lift": return number(values.BurstLift)
        case "burst-life": return number(values.BurstLife)
        case "burst-gravity": return number(values.BurstGravity)
        case "burst-colors": return colorsText(values.BurstColors)
        case "debris-count": return strconv.Itoa(values.DebrisCount)
        case "debris-speed": return number(values.DebrisSpeed)
        case "debris-lift": return number(values.DebrisLift)
        case "debris-life": return number(values.DebrisLife)
        case "debris-gravity": return number(values.DebrisGravity)
        case "debris-colors": return colorsText(values.DebrisColors)
        case "spark-rate": return number(values.SparkRate)
        case "spark-speed": return number(values.SparkSpeed)
        case "spark-life": return number(values.SparkLife)
        case "spark-gravity": return number(values.SparkGravity)
        case "spark-colors": return colorsText(values.SparkColors)
        case "impact-height": return number(values.ImpactHeight)
        case "flash-time": return number(values.FlashTime)
        case "flash-strength": return number(values.FlashStrength)
        case "flash-color": return colorText(values.FlashColor)
        case "hit-stop": return number(values.HitStop)
        case "shake": return number(values.Shake)
        case "shake-time": return number(values.ShakeTime)
        case "pulse-radius": return number(values.PulseRadius)
        case "pulse-time": return number(values.PulseTime)
        case "pulse-strength": return number(values.PulseStrength)
        case "pulse-color": return colorText(values.PulseColor)
        case "decal": return values.Decal
        case "decal-strength": return number(values.DecalStrength)
        case "dark": return number(values.Dark)
        case "light-radius": return number(values.LightRadius)
        case "light-strength": return number(values.LightStrength)
        case "light-flicker": return number(values.LightFlicker)
        case "light-linger": return number(values.LightLinger)
        case "shadow-reach": return number(values.ShadowReach)
        case "rim-light": return number(values.RimLight)
        case "ground-light": return number(values.GroundLight)
        case "rim-color": return colorText(values.RimColor)
        case "corpse-color":
            if !values.HasCorpseColor {
                return noColor
            }
            return colorText(values.CorpseColor)
        case "dark-color": return colorText(values.DarkColor)
        case "resolve":
            if values.ResolveAtImpact {
                return resolveImpact
            }
            return resolveEnd
        case "throw": return number(values.Throw)
    }
    return ""
}

// takes a value of the file. false if the name or the value is not known
func (values *SpellValues) set(name string, text string) bool {
    number, numberError := strconv.ParseFloat(strings.TrimSpace(text), 64)
    oneColor, colorOk := parseColor(text)
    many := colors(text)

    switch name {
        case "trail-colors":
            values.TrailColors = many
            return len(many) > 0 || strings.TrimSpace(text) == ""
        case "burst-colors":
            values.BurstColors = many
            return len(many) > 0 || strings.TrimSpace(text) == ""
        case "debris-colors":
            values.DebrisColors = many
            return len(many) > 0 || strings.TrimSpace(text) == ""
        case "spark-colors":
            values.SparkColors = many
            return len(many) > 0 || strings.TrimSpace(text) == ""
        case "flash-color":
            if colorOk {
                values.FlashColor = oneColor
            }
            return colorOk
        case "pulse-color":
            if colorOk {
                values.PulseColor = oneColor
            }
            return colorOk
        case "rim-color":
            if colorOk {
                values.RimColor = oneColor
            }
            return colorOk
        case "dark-color":
            if colorOk {
                values.DarkColor = oneColor
            }
            return colorOk
        case "resolve":
            when := strings.ToLower(strings.TrimSpace(text))
            if when != resolveImpact && when != resolveEnd {
                return false
            }
            values.ResolveAtImpact = when == resolveImpact
            return true
        case "corpse-color":
            if strings.ToLower(strings.TrimSpace(text)) == noColor {
                values.HasCorpseColor = false
                return true
            }
            if colorOk {
                values.HasCorpseColor = true
                values.CorpseColor = oneColor
            }
            return colorOk
        case "decal":
            kind := strings.ToLower(strings.TrimSpace(text))
            known := kind == DecalNone || slices.Contains(DecalKinds(), kind)
            if known {
                values.Decal = kind
            }
            return known
    }

    if numberError != nil {
        return false
    }

    switch name {
        case "emissive": values.Emissive = number
        case "trail-rate": values.TrailRate = number
        case "trail-life": values.TrailLife = number
        case "trail-speed": values.TrailSpeed = number
        case "trail-gravity": values.TrailGravity = number
        case "burst-count": values.BurstCount = int(number)
        case "burst-speed": values.BurstSpeed = number
        case "burst-lift": values.BurstLift = number
        case "burst-life": values.BurstLife = number
        case "burst-gravity": values.BurstGravity = number
        case "debris-count": values.DebrisCount = int(number)
        case "debris-speed": values.DebrisSpeed = number
        case "debris-lift": values.DebrisLift = number
        case "debris-life": values.DebrisLife = number
        case "debris-gravity": values.DebrisGravity = number
        case "spark-rate": values.SparkRate = number
        case "spark-speed": values.SparkSpeed = number
        case "spark-life": values.SparkLife = number
        case "spark-gravity": values.SparkGravity = number
        case "impact-height": values.ImpactHeight = number
        case "flash-time": values.FlashTime = number
        case "flash-strength": values.FlashStrength = number
        case "hit-stop": values.HitStop = number
        case "shake": values.Shake = number
        case "shake-time": values.ShakeTime = number
        case "pulse-radius": values.PulseRadius = number
        case "pulse-time": values.PulseTime = number
        case "pulse-strength": values.PulseStrength = number
        case "decal-strength": values.DecalStrength = number
        case "dark": values.Dark = number
        case "light-radius": values.LightRadius = number
        case "light-strength": values.LightStrength = number
        case "light-flicker": values.LightFlicker = number
        case "light-linger": values.LightLinger = number
        case "shadow-reach": values.ShadowReach = number
        case "rim-light": values.RimLight = number
        case "ground-light": values.GroundLight = number
        case "throw": values.Throw = number
        default:
            return false
    }

    return true
}

// the values of a spell: the ones of the game, then the ones the file has for all spells, then
// the ones it has for this spell. file is what mod.Effects gives. notes says what was wrong
func spellValues(name string, file map[string]map[string]string) (SpellValues, []string) {
    values := gameValues(name)
    var notes []string

    for _, part := range []string{mod.EffectsDefault, strings.ToLower(name)} {
        for valueName, text := range file[part] {
            if !values.set(valueName, text) {
                notes = append(notes, fmt.Sprintf("%v: [%v] %v = %v is not understood", mod.EffectsFile, part, valueName, text))
            }
        }
    }

    return values, notes
}

const effectsTemplateHead = `# The effects of spells in battles.
#
# This file goes into the replacement folder (mod), next to README.txt, with the name
# effects.txt. The game reads it again about once a second while it runs: change a value, save,
# cast the spell again. A line that is taken out, or a file that is not there, gives the value
# of the game.
#
# A part in brackets is the name of a spell as its folder in spells\ has it. Several spells of a
# realm show one picture, those are the parts Realm Nature, Realm Life and so on.
# [default] is for all spells: a value there counts for every spell that does not name it.
#
# Times are seconds, ways are art pixels, speeds art pixels a second.
# Colors are six letters and digits as in paint programs: ff8000. Particles go through their
# colors from the first to the last while they live.
#
#   emissive        how much the picture of the spell lights itself up, 0 to 1
#
#   in flight (bolts):
#   trail-rate      particles a second that come off it. 0: none
#   trail-life      how long they live
#   trail-speed     how fast they go their own way
#   trail-gravity   how fast they get faster downward. below 0 they rise
#   trail-colors    their colors
#
#   where it hits:
#   burst-count     particles that are thrown up. 0: none
#   burst-speed     how fast they fly apart
#   burst-lift      how fast they go up at first
#   burst-life      how long they live
#   burst-gravity   how fast they get faster downward
#   burst-colors    their colors
#   impact-height   how far over the ground it hits
#   flash-time      the unit that is hit shows in one color for this long. 0: not
#   flash-strength  how much, 0 to 1
#   flash-color     in which color
#   hit-stop        the battle stands still for this long. 0: not
#   shake           the view shakes by this many art pixels at most. 0: not
#   shake-time      for this long
#   pulse-radius    light runs over the ground this many tiles far. 0: none
#   pulse-time      for this long
#   pulse-strength  how much, 0 to 1
#   pulse-color     in which color
#   decal           the mark it leaves on the ground, which stays for the battle: scorch, crater,
#                   frost, spark, or none. Pictures of your own go into the folder of the spell
#                   in the replacement folder as decal_00.png, decal_01.png and on, in any
#                   size; they are used in place of these, whatever is named here but none
#   decal-strength  how much of the mark shows, 0 to 1
#
#   the light of the spell:
#   dark            how dark the battlefield gets while the spell plays, 0 to 1. 0: not at all,
#                   and then the spell gives no light and turns no shadows
#   light-radius    how far the light of the spell goes, in art pixels across
#   light-strength  how much of the dark it takes away in its middle, 0 to 1
#   light-flicker   its radius changes by up to this many art pixels all the time. 0: steady
#   light-linger    how long the light stays where the spell has hit, getting smaller
#   shadow-reach    units this near to the spell cast their shadows away from it
#   rim-light       the figures in the light of the spell have the rim that faces it lit,
#                   this much 0 to 1. 0: not
#   ground-light    the ground in the light of the spell gets lighter by the color of the rims,
#                   added to what is there: 0 not at all, 1 by all of that color
#   rim-color       in which color
#
#   dark-color      the dark has a little of this color. 000000: none
#
#   resolve         when the spell does to its target what it does: impact, when it hits, or
#                   end, when its pictures have played to their end
#   throw           how far the figures it kills are thrown, away from where it hit. They lie
#                   facing where it came from. 0: as in a fight
#   corpse-color    the color the figures it kills take on as they lie, or none for gray
#
# [combat] is no spell: corpse-color there is for the figures that are killed in a fight. They
# are gray (none), and the drops of blood that come down on them stay on them in red.
#
# F7 in a battle turns all of this off and on.

[default]
# emissive = 0.25

[combat]
corpse-color = COMBATCOLOR

`

// a file with the values of the game for every spell, to be changed
func EffectsTemplate() string {
    var out strings.Builder
    fight := noColor
    if combatCorpseTinted {
        fight = colorText(combatCorpseColor)
    }
    out.WriteString(strings.ReplaceAll(effectsTemplateHead, "COMBATCOLOR", fight))

    for _, spell := range mod.SpellPictures {
        values := gameValues(spell.Name)
        out.WriteString(fmt.Sprintf("[%v]\n", spell.Name))
        for _, name := range valueNames {
            out.WriteString(fmt.Sprintf("%v = %v\n", name, values.text(name)))
        }
        out.WriteString("\n")
    }

    out.WriteString(bloodTemplate())

    return strings.ReplaceAll(out.String(), "\n", "\r\n")
}
