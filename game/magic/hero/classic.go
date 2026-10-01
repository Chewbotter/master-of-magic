package hero

// Hero abilities as the original game has them (ReMoM MoM/src/UnitStat.c and COMBINIT.c: the
// ability bonuses of heroes). The code is ours.

import (
    rand "github.com/kazzmir/master-of-magic/lib/chance"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// false: upstream's abilities: Might half on ranged attacks, Super Blademaster +8 at Demigod
var ClassicAbilities = true

// Init_Heroes: a Lucky roll falls into Agility (the original's missing break)
const quirkLuckyFallsIntoAgility = true

// Init_Heroes: Arcane Power is never for the Knight (meant for the Elven Archer)
const quirkArcanePowerKnight = true

// Init_Heroes (INITGAME.c): rolls of 1 in 14 while a counter of picks is left; the warrior picks
// need the warrior counter, the mage picks the mage counter (a hero of both kinds has both);
// every pick taken lowers both counters; a roll that does not apply is rolled again
func (hero *Hero) classicExtraAbilities() {
    count := hero.HeroType.RandomAbilityCount()
    warrior, mage := 0, 0
    switch hero.HeroType.RandomAbilityType() {
        case abilityChoiceFighter: warrior = count
        case abilityChoiceMage: mage = count
        case abilityChoiceAny: warrior, mage = count, count
    }
    took := func() {
        warrior -= 1
        mage -= 1
    }
    warriorPicks := []data.AbilityType{data.AbilityLeadership, data.AbilityLegendary, data.AbilityArmsmaster,
        data.AbilityBlademaster, data.AbilityMight, data.AbilityConstitution}
    // a hero's magic shot of a bolt (not the lightning of the Warlock and the Chaos Warrior)
    raw := hero.GetRawUnit()
    boltCaster := raw.GetRangedAttackDamageType() == units.DamageRangedMagical &&
        hero.HeroType != HeroYramrag && hero.HeroType != HeroWarrax

    for tries := 0; (warrior > 0 || mage > 0) && tries < 10000; tries++ {
        roll := rand.N(14)
        switch {
            case roll <= 5:
                ability := warriorPicks[roll]
                if warrior > 0 && !hero.HasAbility(superVersion(ability)) && hero.AddAbility(ability) {
                    took()
                }
            case roll == 6:
                if mage <= 0 || (quirkArcanePowerKnight && hero.HeroType == HeroSirHarold) || hero.HasAbility(data.AbilitySuperArcanePower) {
                    continue
                }
                if hero.HasAbility(data.AbilityArcanePower) && !boltCaster {
                    // the second pick is used up and nothing changes
                    took()
                    continue
                }
                if hero.AddAbility(data.AbilityArcanePower) {
                    took()
                }
            case roll == 7 || roll == 13:
                ability := data.AbilityPrayermaster
                if roll == 13 {
                    ability = data.AbilitySage
                }
                if mage > 0 && !hero.HasAbility(superVersion(ability)) && hero.AddAbility(ability) {
                    took()
                }
            case roll == 8:
                if mage > 0 && hero.AddAbility(data.AbilityCaster) {
                    took()
                }
            case roll == 9:
                if hero.HeroType != HeroTorin && !hero.HasAbility(data.AbilityNoble) && hero.AddAbility(data.AbilityNoble) {
                    took()
                }
            case roll == 10:
                if !hero.HasAbility(data.AbilityMagicImmunity) && !hero.HasAbility(data.AbilityCharmed) && hero.AddAbility(data.AbilityCharmed) {
                    took()
                }
            case roll == 11 || roll == 12:
                if roll == 11 {
                    if !hero.HasAbility(data.AbilityLucky) && hero.AddAbility(data.AbilityLucky) {
                        took()
                    }
                    if !quirkLuckyFallsIntoAgility {
                        continue
                    }
                }
                if warrior > 0 && !hero.HasAbility(data.AbilitySuperAgility) && hero.AddAbility(data.AbilityAgility) {
                    took()
                }
        }
    }
}
