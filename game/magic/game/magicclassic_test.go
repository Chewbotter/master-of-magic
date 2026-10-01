package game

import (
    "testing"

    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    "github.com/kazzmir/master-of-magic/lib/set"
)

// Apply_Automatic_Spell_Counters: a ward counters its realm, Consecration death and chaos
func TestClassicCounters(test *testing.T) {
    scenario := makeChewScenario(test, 50, chewIsland...)
    caster := scenario.wizard("Merlin", data.BannerRed)
    owner := scenario.wizard("Kali", data.BannerGreen)
    city := scenario.city(owner, 5, 5, 4, false)
    city.Enchantments = set.MakeSet[citylib.Enchantment]()

    earthquake := spellbook.Spell{Name: "Earthquake", Magic: data.NatureMagic, CastCost: 200, Eligibility: spellbook.EligibilityOverlandOnly}
    famine := spellbook.Spell{Name: "Famine", Magic: data.DeathMagic, CastCost: 120, Eligibility: spellbook.EligibilityOverlandOnly}

    if classicCountered(city, earthquake, caster, 200) || classicCountered(city, famine, caster, 120) {
        test.Fatalf("a city without wards counters nothing")
    }
    city.AddEnchantment(data.CityEnchantmentNatureWard, owner.GetBanner())
    if !classicCountered(city, earthquake, caster, 200) {
        test.Errorf("a Nature Ward counters every nature spell")
    }
    if classicCountered(city, famine, caster, 120) {
        test.Errorf("a Nature Ward does not counter a death spell")
    }
    city.AddEnchantment(data.CityEnchantmentConsecration, owner.GetBanner())
    if !classicCountered(city, famine, caster, 120) {
        test.Errorf("Consecration counters a death spell")
    }
}

// Spell Blast: the Spell of Return starts over, any other spell is lost
func TestClassicSpellBlast(test *testing.T) {
    scenario := makeChewScenario(test, 50, chewIsland...)
    caster := scenario.wizard("Merlin", data.BannerRed)
    target := scenario.wizard("Kali", data.BannerGreen)
    caster.Mana = 1000

    target.CastingSpell = spellbook.Spell{Name: "Spell of Return", CastCost: 500, Eligibility: spellbook.EligibilityOverlandWhileBanished}
    target.CastingSpellProgress = 300
    if !classicSpellBlast(caster, target) {
        test.Fatalf("the blast should work")
    }
    if !target.CastingSpell.Valid() || target.CastingSpellProgress != 0 {
        test.Errorf("the Spell of Return starts over, it is not lost")
    }

    target.CastingSpell = spellbook.Spell{Name: "Famine", CastCost: 120, Eligibility: spellbook.EligibilityOverlandOnly}
    target.CastingSpellProgress = 50
    if !classicSpellBlast(caster, target) || target.CastingSpell.Valid() {
        test.Errorf("any other spell is lost")
    }
}

// Apply_Drain_Power, Apply_Cruel_Unminding, Do_All_Units_XP_Check
func TestClassicMagicRolls(test *testing.T) {
    for range 200 {
        if amount := classicDrainAmount(); amount < 10 || amount > 200 {
            test.Fatalf("Drain Power takes 10 to 200, took %v", amount)
        }
        if lost := classicUnminding(50); lost < 1 || lost > 5 {
            test.Fatalf("Cruel Unminding of 50 skill takes 1 to 5, took %v", lost)
        }
    }

    scenario := makeChewScenario(test, 50, chewIsland...)
    player := scenario.wizard("Merlin", data.BannerRed)
    unit := scenario.units(player, 5, 5, units.HighMenSpearmen)[0]
    unit.SetBusy(units.BusyStatusStasis)
    classicStasisNew[unit] = true
    classicStasisTurn(player.FindStack(5, 5, data.PlaneArcanus))
    if unit.GetBusy() != units.BusyStatusStasis {
        test.Errorf("a unit is not freed on the first turn of Stasis")
    }
}
