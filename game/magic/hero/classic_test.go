package hero

import (
    "maps"
    "reflect"
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// Init_Heroes: Torin is never Noble, a Magic Immune hero never Charmed, the Knight never has
// Arcane Power; every hero gets its picks
func TestClassicExtraAbilities(test *testing.T) {
    for range 50 {
        for _, heroType := range AllHeroTypes() {
            hero := MakeHero(units.MakeOverworldUnit(heroType.GetUnit(), 0, 0, data.PlaneArcanus), heroType, "test")
            before := maps.Clone(hero.Abilities)
            hero.classicExtraAbilities()
            if heroType == HeroTorin && hero.HasAbility(data.AbilityNoble) {
                test.Fatalf("Torin is never Noble")
            }
            if heroType == HeroSirHarold && (hero.HasAbility(data.AbilityArcanePower) || hero.HasAbility(data.AbilitySuperArcanePower)) {
                test.Fatalf("the Knight never has Arcane Power")
            }
            if heroType.RandomAbilityCount() > 0 && reflect.DeepEqual(before, hero.Abilities) {
                test.Errorf("%v got no picks", heroType)
            }
        }
    }
}
