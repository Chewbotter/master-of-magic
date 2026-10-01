package ai

import (
    "testing"

    buildinglib "github.com/kazzmir/master-of-magic/game/magic/building"
    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    "github.com/kazzmir/master-of-magic/lib/set"
)

// computer wizards never move their fortress (the user's rule): Move Fortress finds no target even
// when another city of the wizard holds the stronger garrison
func TestChewbotNoMoveFortress(test *testing.T) {
    self := chewSpellWizard("Kali", data.BannerPurple, false, data.WizardBook{Magic: data.SorceryMagic, Count: 8})
    home := &citylib.City{Name: "Home", Buildings: set.MakeSet[buildinglib.Building]()}
    home.Buildings.Insert(buildinglib.BuildingFortress)
    other := &citylib.City{Name: "Other", X: 3, Buildings: set.MakeSet[buildinglib.Building]()}
    self.AddCity(home)
    self.AddCity(other)

    world := &chewSpellWorld{
        Self: self,
        Players: []*playerlib.Player{self},
        Cities: []*citylib.City{home, other},
        Owners: map[*citylib.City]*playerlib.Player{home: self, other: self},
        Garrison: map[*citylib.City]int{home: 1, other: 50},
    }

    ai := &ChewbotAI{}
    if _, found := ai.spellTarget(self, nil, world, spellbook.Spell{Name: "Move Fortress"}); found != ComputersMoveFortress {
        test.Errorf("Move Fortress found a target: %v", found)
    }
}
