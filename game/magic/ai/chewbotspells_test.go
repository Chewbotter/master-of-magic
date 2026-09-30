package ai

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/setup"
    herolib "github.com/kazzmir/master-of-magic/game/magic/hero"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

func chewSpellWizard(name string, banner data.BannerType, human bool, books ...data.WizardBook) *playerlib.Player {
    return playerlib.MakePlayer(setup.WizardCustom{Name: name, Banner: banner, Books: books}, human, 10, 10, make(map[herolib.HeroType]string), &playerlib.NoGlobalEnchantments{})
}

// the realms of most and second most books; the second starts at nature (quirkSecondaryFromNature);
// none is nature
func TestChewbotRealms(test *testing.T) {
    wizard := setup.WizardCustom{Books: []data.WizardBook{{Magic: data.DeathMagic, Count: 6}, {Magic: data.ChaosMagic, Count: 5}}}
    if primary, secondary := chewRealms(wizard); primary != data.DeathMagic || secondary != data.ChaosMagic {
        test.Errorf("death 6, chaos 5: %v %v", primary, secondary)
    }
    wizard = setup.WizardCustom{Books: []data.WizardBook{{Magic: data.LifeMagic, Count: 11}}}
    if primary, secondary := chewRealms(wizard); primary != data.LifeMagic || secondary != data.NatureMagic {
        test.Errorf("life only: %v %v", primary, secondary)
    }
    wizard = setup.WizardCustom{Books: []data.WizardBook{{Magic: data.NatureMagic, Count: 6}, {Magic: data.ChaosMagic, Count: 5}}}
    if primary, secondary := chewRealms(wizard); primary != data.NatureMagic || secondary != data.NatureMagic {
        test.Errorf("nature 6, chaos 5: %v %v", primary, secondary)
    }
    if primary, _ := chewRealms(setup.WizardCustom{}); primary != data.NatureMagic {
        test.Errorf("no books: %v", primary)
    }
}

// Disjunction keeps its scores in a signed byte (quirkDisjunctionBytes): Suppress Magic (250) is
// never its target, Chaos Surge (100) is
func TestChewbotDisjunctionTarget(test *testing.T) {
    self := chewSpellWizard("Kali", data.BannerPurple, false, data.WizardBook{Magic: data.DeathMagic, Count: 8})
    human := chewSpellWizard("Merlin", data.BannerRed, true, data.WizardBook{Magic: data.LifeMagic, Count: 8})
    world := &chewSpellWorld{Self: self, Human: human, Players: []*playerlib.Player{self, human}}

    human.AddEnchantment(data.EnchantmentSuppressMagic)
    if _, ok := chewDisjunctionTarget(world, "Disjunction"); ok == quirkDisjunctionBytes {
        test.Errorf("Suppress Magic as the only global: target %v", ok)
    }
    human.AddEnchantment(data.EnchantmentChaosSurge)
    target, ok := chewDisjunctionTarget(world, "Disjunction")
    if !ok || target.Player != human || (quirkDisjunctionBytes && target.Enchantment != data.EnchantmentChaosSurge) {
        test.Errorf("with Chaos Surge: %+v %v", target, ok)
    }
    // Spell Binding scores without the bytes going below 0: Suppress Magic 100 over Chaos Surge 100
    // (the first slot wins a tie)
    target, ok = chewDisjunctionTarget(world, "Spell Binding")
    if !ok || target.Enchantment != data.EnchantmentSuppressMagic {
        test.Errorf("Spell Binding: %+v %v", target, ok)
    }
}

// the weighted choice of research: a spell of a group the wizard knows weighs far less
func TestChewbotResearchWeights(test *testing.T) {
    defer ChewbotFixedRolls(0)()
    if got := chewWeightedChoiceLong([]int{0, 5, 3}); got != 1 {
        test.Errorf("the lowest roll picks the first with weight: %v", got)
    }
}

// the damping of suppressed realms: under 20 none, under 50 a third, else half
func TestChewbotBand(test *testing.T) {
    weights := []int{19, 20, 49, 50, 100}
    chewBand(weights, 0, 1, 2, 3, 4)
    want := []int{0, 6, 16, 25, 50}
    for index := range want {
        if weights[index] != want[index] {
            test.Errorf("banded %v, want %v", weights, want)
            break
        }
    }
}
