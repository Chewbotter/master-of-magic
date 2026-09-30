package game

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    herolib "github.com/kazzmir/master-of-magic/game/magic/hero"
)

// Pick_Random_Hero: the Priestess needs life books; the fame asked is strictly below the wizard's
func TestClassicPickHero(test *testing.T) {
    scenario := makeChewScenario(test, 100, chewIsland...)
    wizard := scenario.wizard("Merlin", data.BannerRed)

    // only the Priestess is left to offer
    for heroType, hero := range wizard.HeroPool {
        if heroType != herolib.HeroElana {
            hero.SetStatus(herolib.StatusEmployed)
        }
    }
    priestess := wizard.HeroPool[herolib.HeroElana]
    wizard.Fame = priestess.GetRequiredFame() + 50

    for range 200 {
        if classicPickHero(wizard) != nil {
            test.Fatalf("without life books the Priestess should not come")
        }
    }

    wizard.Wizard.AddMagicLevel(data.LifeMagic, 1)
    found := false
    for range 200 {
        if classicPickHero(wizard) != nil {
            found = true
        }
    }
    if !found {
        test.Errorf("with life books the Priestess should come")
    }

    wizard.Fame = priestess.GetRequiredFame()
    for range 200 {
        if classicPickHero(wizard) != nil {
            test.Fatalf("with fame equal to what she asks the Priestess should not come (the original's)")
        }
    }
}
