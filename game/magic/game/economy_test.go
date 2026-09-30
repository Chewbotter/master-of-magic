package game

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// Unit_Mana_Upkeep with Conjurer, Player_Armies_And_Enchantments_Mana_Upkeep with Channeler
func TestManaUpkeepRetorts(test *testing.T) {
    scenario := makeChewScenario(test, 100, chewIsland...)
    wizard := scenario.wizard("Merlin", data.BannerRed)
    // war bears 2, unicorns 5, spearmen none
    scenario.units(wizard, 8, 5, units.WarBear, units.Unicorn, units.HighMenSpearmen)

    if got := wizard.TotalUnitUpkeepMana(); got != 7 {
        test.Errorf("upkeep %v, want 7", got)
    }

    wizard.Wizard.EnableRetort(data.RetortConjurer)
    // (2 * 3 + 3) / 4 = 2, (5 * 3 + 3) / 4 = 4
    if got := wizard.TotalUnitUpkeepMana(); got != 6 {
        test.Errorf("with Conjurer %v, want 6", got)
    }

    wizard.Wizard.EnableRetort(data.RetortChanneler)
    if got := wizard.ManaUpkeep(scenario.Model); got != 3 {
        test.Errorf("with Channeler too %v, want 3", got)
    }
}

// Player_Magic_Power_Distribution
func TestPowerSplit(test *testing.T) {
    scenario := makeChewScenario(test, 100, chewIsland...)
    wizard := scenario.wizard("Merlin", data.BannerRed)
    wizard.PowerDistribution.Mana = 1.0 / 3
    wizard.PowerDistribution.Research = 1.0 / 3
    wizard.PowerDistribution.Skill = 1.0 / 3

    if got := wizard.CastingSkillPerTurn(100); got != 33 {
        test.Errorf("skill %v, want 33", got)
    }
    if got := wizard.SpellResearchPerTurn(100); got != 34 {
        test.Errorf("research %v, want 34, the rest of the rounding", got)
    }

    wizard.Wizard.EnableRetort(data.RetortManaFocusing)
    wizard.Wizard.EnableRetort(data.RetortArchmage)
    if got := wizard.ManaPerTurn(100, scenario.Model); got != 41 {
        test.Errorf("mana with Mana Focusing %v, want 33 + 8", got)
    }
    if got := wizard.CastingSkillPerTurn(100); got != 49 {
        test.Errorf("skill with Archmage %v, want 33 * 3 / 2", got)
    }
}

// Casting_Cost: the amount of the reduction rounded down, so the cost rounds up
func TestClassicSpellCost(test *testing.T) {
    scenario := makeChewScenario(test, 100, chewIsland...)
    wizard := scenario.wizard("Merlin", data.BannerRed)
    wizard.Wizard.EnableRetort(data.RetortRunemaster)
    spell := spellbook.Spell{Name: "Test", Magic: data.ArcaneMagic, CastCost: 25, Eligibility: spellbook.EligibilityOverlandOnly}
    // 25 - 25 * 25 / 100 = 25 - 6
    if got := wizard.ComputeEffectiveSpellCost(spell, true); got != 19 {
        test.Errorf("cost %v, want 19", got)
    }
}

// Build_Research_List: the lowest tier of every realm; the Spell of Mastery only when nothing
// else is left
func TestResearchCandidates(test *testing.T) {
    scenario := makeChewScenario(test, 100, chewIsland...)
    wizard := scenario.wizard("Merlin", data.BannerRed)

    add := func(name string, magic data.MagicType, rarity spellbook.SpellRarity, cost int) spellbook.Spell {
        spell := spellbook.Spell{Name: name, Magic: magic, Rarity: rarity, ResearchCost: cost, Index: len(wizard.ResearchPoolSpells.Spells) + 1}
        wizard.ResearchPoolSpells.AddSpell(spell)
        return spell
    }
    var commons []string
    for i := range 9 {
        name := string(rune('A' + i))
        commons = append(commons, name)
        add(name, data.NatureMagic, spellbook.SpellRarityCommon, 100)
    }
    add("Uncommon", data.NatureMagic, spellbook.SpellRarityUncommon, 300)
    add("Arcane", data.ArcaneMagic, spellbook.SpellRarityUncommon, 200)
    add("Spell of Mastery", data.ArcaneMagic, spellbook.SpellRarityVeryRare, 60000)
    wizard.SpellOfMasteryCost = 50000

    names := func() map[string]bool {
        out := map[string]bool{}
        for _, spell := range wizard.ResearchCandidateSpells.Spells {
            out[spell.Name] = true
        }
        return out
    }

    wizard.ResearchCandidateSpells = spellbook.Spells{}
    wizard.UpdateResearchCandidates()
    if got := names(); len(got) != 8 || got["Uncommon"] || got["Spell of Mastery"] {
        test.Errorf("8 of the commons and the arcane spell, no uncommon while a common is left: %v", got)
    }

    // all but the uncommon learned: the uncommon and then the Spell of Mastery
    for _, name := range append(commons, "Arcane") {
        wizard.LearnSpell(wizard.ResearchPoolSpells.FindByName(name))
    }
    mastery := wizard.ResearchCandidateSpells.FindByName("Spell of Mastery")
    // 50000 less half of nine 100 and one 200
    if got := names(); !got["Uncommon"] || !mastery.Valid() || mastery.ResearchCost != 50000 - 550 {
        test.Errorf("the uncommon and the Spell of Mastery at 49450 should be on the list: %v %v", got, mastery.ResearchCost)
    }
}

// Players_Apply_Upkeeps: only the human loses units; food and gold take normal units, never
// creatures; a computer player's reserves just stop at 0
func TestClassicUpkeep(test *testing.T) {
    scenario := makeChewScenario(test, 100, chewIsland...)
    human := scenario.wizard("Merlin", data.BannerRed)
    human.Human = true
    human.Mana = 100
    scenario.units(human, 8, 5, units.HighMenSpearmen, units.WarBear, units.HighMenSwordsmen)
    game := &Game{Model: scenario.Model, headless: true}

    messages := game.classicUpkeep(human, 0)
    if human.UnitCount() != 1 || len(messages) != 2 {
        test.Errorf("the spearmen and swordsmen should desert, the bears stay: %v units, %v", human.UnitCount(), messages)
    }
    if len(messages) > 0 && messages[0] != "Swordsmen - deserted: lack of food." {
        test.Errorf("the newest unit first: %q", messages[0])
    }

    computer := scenario.wizard("Kali", data.BannerGreen)
    scenario.units(computer, 10, 5, units.HighMenSpearmen, units.WarBear)
    if messages := game.classicUpkeep(computer, 0); len(messages) != 0 || computer.UnitCount() != 2 {
        test.Errorf("a computer wizard loses nothing: %v", messages)
    }
}
