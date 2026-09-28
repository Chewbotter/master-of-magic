package combat

import (
    "testing"
)

// only an army that is on auto and was told so casts no spells. an army nobody told anything, which
// is every army of the game itself, casts on auto and off
func TestNoSpellsOnAutoOnlyWhenAskedFor(test *testing.T) {
    cases := []struct {
        auto bool
        noSpells bool
        casts bool
    }{
        {auto: false, noSpells: false, casts: true},
        {auto: true, noSpells: false, casts: true},
        {auto: false, noSpells: true, casts: true},
        {auto: true, noSpells: true, casts: false},
    }

    for _, use := range cases {
        army := &Army{Auto: use.auto, NoSpellsOnAuto: use.noSpells}
        if army.autoCastsSpells() != use.casts {
            test.Errorf("auto %v, told not to cast %v: casts %v, should be %v", use.auto, use.noSpells, army.autoCastsSpells(), use.casts)
        }
    }
}
