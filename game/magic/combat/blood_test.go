package combat

import (
    "image"
    "image/color"
    "math"
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/mod"
)

// as the original: damage plus half the figures, over twice the figures, 1 to 5
func TestBloodAmount(test *testing.T) {
    for _, each := range []struct{Damage, Figures, Amount int}{
        {0, 6, 0},
        {1, 6, 1},
        {10, 1, 5},
        {10, 8, 1},
        {30, 8, 2},
        {8, 2, 2},
        {3, 1, 1},
        {100, 4, 5},
        {4, 0, 3},
    } {
        if amount := bloodAmount(each.Damage, each.Figures); amount != each.Amount {
            test.Errorf("%v damage to %v figures: %v, should be %v", each.Damage, each.Figures, amount, each.Amount)
        }
    }
}

// a drop that leaves a stain ends where it comes down, and is noted once
func TestBloodLands(test *testing.T) {
    var system particleSystem
    ramp := []color.RGBA{{R: 255, A: 255}}
    stain := color.RGBA{R: 100, A: 255}

    system.emitSpray(50, 80, 9, 40, 0, math.Pi / 4, 40, 45, 260, 0.9, ramp, true, stain)
    if len(system.Particles) != 40 {
        test.Fatalf("%v drops", len(system.Particles))
    }

    for range 120 {
        system.step(1.0 / 60)
    }

    if len(system.Particles) != 0 {
        test.Errorf("%v drops are left", len(system.Particles))
    }
    if len(system.Landed) == 0 || len(system.Landed) > 40 {
        test.Fatalf("%v drops have come down", len(system.Landed))
    }

    right := 0
    for _, each := range system.Landed {
        if each.Color != stain {
            test.Errorf("a stain of %v", each.Color)
        }
        if each.Y < 60 || each.Y > 100 {
            test.Errorf("a stain at %v,%v, the ground is at 80", each.X, each.Y)
        }
        if each.X > 50 {
            right += 1
        }
    }
    // they were thrown to the right
    if right * 10 < len(system.Landed) * 8 {
        test.Errorf("%v of %v came down to the right", right, len(system.Landed))
    }
}

// the field with the ground around it has room for every tile
func TestBloodField(test *testing.T) {
    bounds := bloodFieldBounds()
    field := MakeBattlefieldMatrix()

    for _, tile := range []image.Point{{0, 0}, {29, 29}, {0, 29}, {29, 0}, {-10, 15}, {40, 15}} {
        x, y := field.Apply(float64(tile.X), float64(tile.Y))
        if !image.Pt(int(x), int(y)).In(bounds) {
            test.Errorf("tile %v at %v,%v is not in %v", tile, x, y, bounds)
        }
    }
}

// the file gives values over the ones of the game, and the template has the parts of the blood
func TestBloodValues(test *testing.T) {
    file := mod.ParseEffects("[blood]\ndrops = 9\ncolors = 00ff00\nstain-strength = 0\n")
    values := bloodValues(bloodPart, file)
    if values.Drops != 9 || len(values.Colors) != 1 || values.StainStrength != 0 || values.Speed != gameBlood(bloodPart).Speed {
        test.Errorf("%+v", values)
    }

    template := mod.ParseEffects(EffectsTemplate())
    for _, part := range []string{bloodPart, bloodUndeadPart, bloodSpiritPart} {
        if len(template[part]) != len(bloodValueNames) {
            test.Errorf("the template has %v values of [%v]", len(template[part]), part)
        }
        game := gameBlood(part)
        read := bloodValues(part, template)
        for _, name := range bloodValueNames {
            if game.text(name) != read.text(name) {
                test.Errorf("[%v] %v: %v in the template, %v in the game", part, name, read.text(name), game.text(name))
            }
        }
    }
}
