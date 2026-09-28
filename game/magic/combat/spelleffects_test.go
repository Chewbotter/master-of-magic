package combat

import (
    "image"
    "image/color"
    "strings"
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/mod"
)

// particles go out when their time is over, fall, and lie on their ground
func TestParticlesLive(test *testing.T) {
    var system particleSystem
    ramp := []color.RGBA{{R: 255, A: 255}, {G: 255, A: 255}, {B: 255, A: 255}}

    system.add(particle{X: 10, Y: 10, SpeedX: 20, SpeedY: -30, Gravity: 200, Life: 1, Colors: ramp, HasGround: true, Ground: 20})
    system.add(particle{X: 0, Y: 0, Life: 0.1, Colors: ramp})
    // nothing to show: not taken
    system.add(particle{X: 0, Y: 0, Life: 1})
    if len(system.Particles) != 2 {
        test.Fatalf("%v particles", len(system.Particles))
    }

    first := system.Particles[0].color()
    if first.R != 255 {
        test.Errorf("first color %v", first)
    }

    lowest := 0.0
    for range 30 {
        system.step(1.0 / 60)
        if len(system.Particles) > 0 {
            lowest = max(lowest, system.Particles[0].Y)
        }
    }

    if len(system.Particles) != 1 {
        test.Fatalf("%v particles after half a second, the short one should be out", len(system.Particles))
    }
    if lowest > 20 {
        test.Errorf("the particle was at %v, under its ground at 20", lowest)
    }
    if system.Particles[0].X <= 10 {
        test.Errorf("the particle did not fly")
    }
    if system.Particles[0].color().G != 255 {
        test.Errorf("color in the middle of its life %v", system.Particles[0].color())
    }

    for range 40 {
        system.step(1.0 / 60)
    }
    if len(system.Particles) != 0 {
        test.Errorf("%v particles after their time", len(system.Particles))
    }
}

func TestParticlesLimit(test *testing.T) {
    var system particleSystem
    ramp := []color.RGBA{{R: 255, A: 255}}
    system.emitBurst(100, 100, 8, particleMax + 50, 60, 50, 200, 1, ramp)
    if len(system.Particles) != particleMax {
        test.Errorf("%v particles, the most is %v", len(system.Particles), particleMax)
    }
}

// the file gives values over the ones of the game, for all spells and for one
func TestSpellValuesFromFile(test *testing.T) {
    game := gameValues("Fire Bolt")
    if game.BurstCount == 0 || game.TrailRate == 0 || game.FlashTime == 0 {
        test.Fatalf("the fire bolt of the game has no effects: %+v", game)
    }

    file := mod.ParseEffects(`
# a note
[default]
emissive = 0.5
shake = 0

[Fire Bolt]
burst-count = 7     # few
burst-colors = ffffff 102030
flash-color = 00ff00
nonsense = 3
trail-rate = many
`)

    values, notes := spellValues("Fire Bolt", file)
    if values.Emissive != 0.5 || values.Shake != 0 {
        test.Errorf("values for all spells: emissive %v shake %v", values.Emissive, values.Shake)
    }
    if values.BurstCount != 7 {
        test.Errorf("burst count %v", values.BurstCount)
    }
    if len(values.BurstColors) != 2 || values.BurstColors[1] != (color.RGBA{R: 0x10, G: 0x20, B: 0x30, A: 0xff}) {
        test.Errorf("burst colors %v", values.BurstColors)
    }
    if values.FlashColor != (color.RGBA{G: 0xff, A: 0xff}) {
        test.Errorf("flash color %v", values.FlashColor)
    }
    if values.TrailRate != game.TrailRate {
        test.Errorf("a value that is not understood changed the trail rate to %v", values.TrailRate)
    }
    if len(notes) != 2 {
        test.Errorf("notes: %v", notes)
    }

    other, _ := spellValues("Ice Bolt", file)
    if other.Emissive != 0.5 || other.BurstCount != gameValues("Ice Bolt").BurstCount {
        test.Errorf("ice bolt: emissive %v burst %v", other.Emissive, other.BurstCount)
    }
}

// the template gives the values of the game back
func TestEffectsTemplate(test *testing.T) {
    text := EffectsTemplate()
    file := mod.ParseEffects(text)

    for _, spell := range mod.SpellPictures {
        part, ok := file[strings.ToLower(spell.Name)]
        if !ok {
            test.Errorf("no part for %v", spell.Name)
            continue
        }
        if len(part) != len(valueNames) {
            test.Errorf("%v has %v values, there are %v", spell.Name, len(part), len(valueNames))
        }

        values, notes := spellValues(spell.Name, file)
        if len(notes) != 0 {
            test.Errorf("%v: %v", spell.Name, notes)
        }

        game := gameValues(spell.Name)
        for _, name := range valueNames {
            if values.text(name) != game.text(name) {
                test.Errorf("%v %v: %v from the template, %v in the game", spell.Name, name, values.text(name), game.text(name))
            }
        }
    }
}

// the light on the ground runs from the middle outward, in steps, and ends
func TestGroundPulse(test *testing.T) {
    pulse := groundPulse{X: 10, Y: 10, Start: 100, Ticks: 20, Radius: 2, Strength: 0.3}

    early := pulseTiles(pulse, 101)
    if early[image.Pt(10, 10)] == 0 {
        test.Errorf("no light in the middle at the start: %v", early)
    }
    if early[image.Pt(12, 10)] != 0 {
        test.Errorf("light at the rim at the start")
    }

    late := pulseTiles(pulse, 112)
    if late[image.Pt(12, 10)] == 0 {
        test.Errorf("no light at the rim later: %v", late)
    }
    if late[image.Pt(10, 10)] != 0 {
        test.Errorf("light in the middle later")
    }

    for point, band := range late {
        if band < 1 || band > pulseBands {
            test.Errorf("step %v at %v", band, point)
        }
        if point.X < 8 || point.X > 12 || point.Y < 8 || point.Y > 12 {
            test.Errorf("light at %v, further than the pulse goes", point)
        }
    }

    if len(pulseTiles(pulse, 120)) != 0 || len(pulseTiles(pulse, 99)) != 0 {
        test.Errorf("light outside of its time")
    }
}

func TestFlashPart(test *testing.T) {
    flash := unitFlash{Start: 10, Ticks: 10, Strength: 0.8}
    if flashPart(flash, 10) != 0.8 || flashPart(flash, 14) != 0.8 {
        test.Errorf("at the start %v, before the middle %v", flashPart(flash, 10), flashPart(flash, 14))
    }
    if part := flashPart(flash, 18); part <= 0 || part >= 0.8 {
        test.Errorf("near the end %v", part)
    }
    if flashPart(flash, 20) != 0 || flashPart(flash, 9) != 0 {
        test.Errorf("outside of its time")
    }
}
