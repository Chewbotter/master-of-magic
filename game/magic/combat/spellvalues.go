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
}

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
    return values
}

// a spell that hurts a unit where it stands: the unit shows in one color
func hurtValues() SpellValues {
    values := plainValues()
    values.FlashTime = 0.1
    return values
}

// the values of the game, by the name of the pictures of a spell
func gameValues(name string) SpellValues {
    switch name {
        case "Fire Bolt":
            return boltValues(fireColors, color.RGBA{R: 0xff, G: 0x90, B: 0x20, A: 0xff})
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
            return values
        case "Ice Bolt":
            values := boltValues(iceColors, color.RGBA{R: 0x70, G: 0xd0, B: 0xff, A: 0xff})
            // what comes off it sinks
            values.TrailGravity = 25
            return values
        case "Doom Bolt":
            values := boltValues(doomColors, color.RGBA{R: 0xff, G: 0x50, B: 0x30, A: 0xff})
            values.BurstCount = 70
            values.Shake = 3
            values.PulseRadius = 3
            return values
        case "Lightning Bolt":
            values := boltValues(lightningColors, color.RGBA{R: 0xc0, G: 0xd8, B: 0xff, A: 0xff})
            // it does not fly in
            values.TrailRate = 0
            values.BurstCount = 35
            values.BurstSpeed = 90
            values.BurstLife = 0.4
            values.FlashTime = 0.1
            values.HitStop = 0.06
            return values
        case "Warp Lightning":
            values := boltValues(lightningColors, color.RGBA{R: 0xc0, G: 0xd8, B: 0xff, A: 0xff})
            values.TrailRate = 0
            values.BurstCount = 20
            values.BurstLife = 0.4
            values.HitStop = 0
            values.Shake = 1
            values.PulseRadius = 0
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
            return values
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
    "flash-time", "flash-strength", "flash-color",
    "hit-stop", "shake", "shake-time",
    "pulse-radius", "pulse-time", "pulse-strength", "pulse-color",
}

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
        case "impact-height": values.ImpactHeight = number
        case "flash-time": values.FlashTime = number
        case "flash-strength": values.FlashStrength = number
        case "hit-stop": values.HitStop = number
        case "shake": values.Shake = number
        case "shake-time": values.ShakeTime = number
        case "pulse-radius": values.PulseRadius = number
        case "pulse-time": values.PulseTime = number
        case "pulse-strength": values.PulseStrength = number
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
#
# F7 in a battle turns all of this off and on.

[default]
# emissive = 0.25

`

// a file with the values of the game for every spell, to be changed
func EffectsTemplate() string {
    var out strings.Builder
    out.WriteString(effectsTemplateHead)

    for _, spell := range mod.SpellPictures {
        values := gameValues(spell.Name)
        out.WriteString(fmt.Sprintf("[%v]\n", spell.Name))
        for _, name := range valueNames {
            out.WriteString(fmt.Sprintf("%v = %v\n", name, values.text(name)))
        }
        out.WriteString("\n")
    }

    return strings.ReplaceAll(out.String(), "\n", "\r\n")
}
