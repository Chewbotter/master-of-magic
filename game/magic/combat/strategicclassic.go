package combat

// The original's quick resolution of a battle nobody watches (ReMoM MoM/src/Combat.c: Strategic_Combat,
// Get_Effective_Hits, Get_Effective_Ranged_Strength, Get_Effective_Melee_Strength,
// Calc_Battlefield_Bonuses, Combat_Structure; MoM/src/CMBMAGIC.c: Combat_Casting_Cost_Multiplier).
// The original fights every battle without the human this way, and the human's own when Strategic
// Combat Only is on. The rules in words: docs/mod/ai-combat.md. The code is ours.
//
// Each side sums over its units: its hits weighed by its defense (Get_Effective_Hits), the threat of
// its missiles and magic, and the threat of its melee (with a thrown attack, breath or gaze). A
// computer wizard's side against a side that is not the human gets half as much again of all of it.
// A wizard with mana and skill adds attack, ranged and healing by its books. Three rounds of
// shooting, then melee until one side's melee is 10 or less: each side takes its strength times 1 to
// 10 percent as damage, and loses strength in proportion to the hits it loses. The loser loses every
// unit; the winner loses as many hits as its side lost of its weighed hits, 3 at a time, units of
// low defense most likely, one unit until it dies.

import (
    rand "github.com/kazzmir/master-of-magic/lib/chance"

    "github.com/kazzmir/master-of-magic/game/magic/data"
)

// false: the fork's own quick resolution and full battles of the AI
var ClassicAutoResolve = true

// dev: a line with the sums of each quick battle (-capture-battle-log)
var StrategicDebug func(format string, args ...any)

// the original's Random(n): 1 to n
var strategicRoll = func(n int) int {
    return rand.N(max(n, 1)) + 1
}

// Combat_Structure: what the square of the battle is
const (
    StructureNone = 0
    StructureCity = 1
    StructureSorceryNode = 2
    StructureNatureNode = 3
    StructureChaosNode = 4
    StructureOcean = 6
)

// a side of a quick battle as the original needs it
type StrategicSide struct {
    // a wizard (not the neutral player, not the monsters of a lair)
    Wizard bool
    Human bool
    Neutral bool
    // the wizard's mana (spent by the battle), casting skill (Nominal_Skill), books by realm
    Mana int
    Skill int
    Books map[data.MagicType]int
    // Combat_Casting_Cost_Multiplier: 5 at the fortress, else 10 and 5 more for every 5 squares beyond
    // 5 (10 for a Channeler)
    CastMultiplier int
}

// Combat_Casting_Cost_Multiplier by the distance to the fortress (the larger of across and down;
// the other plane counts as 30)
func StrategicCastMultiplier(distance int, channeler bool) int {
    if distance == 0 {
        return 5
    }
    if channeler {
        distance = 2
    }
    modifier := 10
    for _, step := range []int{5, 10, 15, 20} {
        if distance > step {
            modifier += 5
        }
    }
    return modifier
}

// Get_Weighted_Choice with the original's halving of all weights while their sum reaches 512
func strategicWeightedChoice(weights []int) int {
    weights = append([]int(nil), weights...)
    total := 0
    for index := 0; index < len(weights); index++ {
        total += weights[index]
        if total >= 512 {
            for other := range weights {
                weights[other] >>= 1
            }
            total = 0
            index = -1
        }
    }
    if total <= 0 {
        return 0
    }
    roll := strategicRoll(total) - weights[0]
    choice := 0
    for roll > 0 && choice < len(weights) - 1 {
        choice += 1
        roll -= weights[choice]
    }
    return choice
}

// the magic of a side: attack, ranged percent and healing by its books (the last realm in the
// original's order wins each), none of a realm a node on the square counters for the attacker
func strategicMagic(side *StrategicSide, structure int, attacker bool) (int, int, int) {
    attack, ranged, healing := 0, 0, 0
    zero := func(ranks int, structures ...int) int {
        if !attacker {
            return ranks
        }
        for _, other := range structures {
            if structure == other {
                return 0
            }
        }
        return ranks
    }
    nature := zero(side.Books[data.NatureMagic], StructureSorceryNode, StructureChaosNode)
    if nature > 0 {
        attack, ranged, healing = nature * 50, nature * 10, nature * 25
    }
    // quirk kept: the original checks the nature ranks a second time with the numbers of sorcery
    nature = zero(nature, StructureNatureNode, StructureChaosNode)
    if nature > 0 {
        attack, healing = nature * 75, nature * 25
    }
    if sorcery := zero(side.Books[data.SorceryMagic], StructureSorceryNode, StructureChaosNode); sorcery > 0 {
        attack, healing = sorcery * 75, sorcery * 25
    }
    if chaos := zero(side.Books[data.ChaosMagic], StructureSorceryNode, StructureNatureNode); chaos > 0 {
        attack, ranged = chaos * 100, chaos * 30
    }
    if life := zero(side.Books[data.LifeMagic], StructureSorceryNode, StructureChaosNode, StructureNatureNode); life > 0 {
        ranged, healing = life * 15, life * 40
    }
    if death := zero(side.Books[data.DeathMagic], StructureSorceryNode, StructureChaosNode, StructureNatureNode); death > 0 {
        attack, healing = death * 75, death * 15
    }
    return attack / 10, ranged / 10, healing / 20
}

// the sums of a side
type strategicSums struct {
    hitsLeft, hitsStart, melee, ranged, rawHits int
    // magic: what the wizard's mana allows, a third of its skill, and the budgets
    manaAllows, thirdSkill int
    attackMagic, rangedMagic, healingMagic int
    spellAttack, spellRangedPercent, spellHealing int
}

// a unit that can not fight on the sea (the original's Undeployable_Battle_Units_On_Water)
func strategicOffWater(unit *ArmyUnit) bool {
    return !(unit.IsFlying() || unit.IsSwimmer() || unit.Unit.IsSailing())
}

func strategicSum(army *Army, defender bool, structure int) strategicSums {
    var sums strategicSums
    for _, unit := range army.units {
        if unit.GetHealth() <= 0 || (structure == StructureOcean && strategicOffWater(unit)) {
            continue
        }
        defense := unit.GetDefense()
        // Calc_Battlefield_Bonuses: defenders of a city +3 defense, walls or not (kept)
        if defender && structure == StructureCity {
            defense += 3
        }
        hits := unit.GetHealth()
        figures := unit.Figures()
        sums.hitsLeft += chewEffectiveHits(hits, defense)
        if chewHasRanged(unit) {
            flags := chewRangedFlags(unit)
            if unit.HasEnchantment(data.UnitEnchantmentEldritchWeapon) {
                flags |= chewEldritch
            }
            threat := chewFlagThreat(chewThreat(unit.GetRangedAttackPower(), figures, 40), figures, flags, false)
            if flags & chewEldritch != 0 {
                threat = threat * 5 / 4
            }
            sums.ranged += threat
        }
        thrown := 0
        gaze := chewGazeNone
        if chewShortRange(unit) {
            thrown = unit.GetRangedAttackPower()
            gaze = chewGaze(unit)
        }
        sums.melee += chewEffectiveMelee(unit.GetMeleeAttackPower(), thrown, figures, chewMeleeFlags(unit), gaze)
        sums.rawHits += hits
    }
    return sums
}

// the weight of a unit of the winner to take the losses: 30 less its defense
func strategicWeight(unit *ArmyUnit, defender bool, structure int) int {
    defense := unit.GetDefense()
    if defender && structure == StructureCity {
        defense += 3
    }
    return 30 - defense
}

// Strategic_Combat: the state, the units the attackers and the defenders lost, and the mana each
// side's wizard spent
func DoClassicStrategicCombat(attackingArmy *Army, defendingArmy *Army, attackerSide StrategicSide, defenderSide StrategicSide, structure int) (CombatState, int, int, int, int) {
    attack := strategicSum(attackingArmy, false, structure)
    defend := strategicSum(defendingArmy, true, structure)

    magic := func(side *StrategicSide, sums *strategicSums, attacker bool) {
        if !side.Wizard || side.Neutral || side.CastMultiplier <= 0 {
            return
        }
        sums.manaAllows = min(10 * side.Mana / side.CastMultiplier, side.Skill)
        sums.thirdSkill = min(side.Skill / 3, sums.manaAllows)
        if sums.manaAllows >= 10 {
            sums.attackMagic, sums.rangedMagic, sums.healingMagic = strategicMagic(side, structure, attacker)
        }
    }
    magic(&attackerSide, &attack, true)
    magic(&defenderSide, &defend, false)

    attack.hitsStart = attack.hitsLeft
    defend.hitsStart = defend.hitsLeft

    // the computer wizards' help against anyone but the human
    cheat := func(sums *strategicSums) {
        for _, value := range []*int{&sums.spellRangedPercent, &sums.spellAttack, &sums.spellHealing, &sums.hitsStart, &sums.hitsLeft,
            &sums.melee, &sums.ranged, &sums.attackMagic, &sums.rangedMagic, &sums.healingMagic} {
            *value += *value / 2
        }
    }
    if !attackerSide.Human && !defenderSide.Human {
        if !attackerSide.Neutral {
            cheat(&attack)
        }
        if !defenderSide.Neutral {
            cheat(&defend)
        }
    }

    if StrategicDebug != nil {
        StrategicDebug("sums: attack hits %v melee %v ranged %v magic %v/%v/%v; defend hits %v melee %v ranged %v magic %v/%v/%v",
            attack.hitsLeft, attack.melee, attack.ranged, attack.attackMagic, attack.rangedMagic, attack.healingMagic,
            defend.hitsLeft, defend.melee, defend.ranged, defend.attackMagic, defend.rangedMagic, defend.healingMagic)
    }

    spentAttacker, spentDefender := 0, 0
    castRanged := func(side *StrategicSide, sums *strategicSums, spent *int) {
        if sums.manaAllows > 10 {
            sums.spellAttack = sums.attackMagic * sums.thirdSkill / 5
            sums.spellRangedPercent = sums.rangedMagic * sums.thirdSkill / 5
            sums.spellHealing = sums.healingMagic * sums.thirdSkill / 5
            sums.spellAttack += sums.ranged * sums.spellRangedPercent / 100
            sums.healingMagic /= 2
            *spent += side.CastMultiplier * sums.thirdSkill / 10
            sums.manaAllows -= sums.thirdSkill
            // kept: the third can only grow here
            if sums.thirdSkill < sums.manaAllows {
                sums.thirdSkill = sums.manaAllows
            }
        } else {
            sums.spellRangedPercent, sums.spellAttack, sums.spellHealing = 0, 0, 0
        }
    }
    castMelee := func(side *StrategicSide, sums *strategicSums, spent *int) {
        if sums.manaAllows <= 10 {
            sums.spellRangedPercent, sums.spellAttack, sums.spellHealing = 0, 0, 0
            return
        }
        sums.spellAttack = sums.attackMagic * sums.thirdSkill / 5
        sums.spellRangedPercent = sums.rangedMagic * sums.thirdSkill / 5
        // kept: the healing of the melee rounds comes from the ranged budget
        sums.spellHealing = sums.rangedMagic * sums.thirdSkill / 5
        sums.spellAttack += sums.ranged * sums.spellRangedPercent / 100
        *spent += side.CastMultiplier * sums.thirdSkill / 10
        sums.manaAllows -= sums.thirdSkill / 2
        if sums.manaAllows / 2 < sums.thirdSkill {
            sums.thirdSkill = sums.manaAllows / 2
        }
    }
    // strength less in proportion to the hits lost (the hits after healing)
    lose := func(strength int, damage int, hitsLeft int) int {
        if hitsLeft <= 0 {
            return 0
        }
        return strength - damage * strength / hitsLeft
    }

    for range 3 {
        if (attack.ranged > 10 || defend.ranged > 10) && attack.hitsLeft > 0 && defend.hitsLeft > 0 {
            castRanged(&attackerSide, &attack, &spentAttacker)
            castRanged(&defenderSide, &defend, &spentDefender)
            toDefender := (attack.ranged + attack.spellAttack) * strategicRoll(10) / 100
            toAttacker := (defend.ranged + defend.spellAttack) * strategicRoll(10) / 100
            attack.hitsLeft += attack.spellHealing
            defend.hitsLeft += defend.spellHealing
            attack.ranged = max(lose(attack.ranged, toAttacker, attack.hitsLeft), 0)
            defend.ranged = max(lose(defend.ranged, toDefender, defend.hitsLeft), 0)
            attack.melee = lose(attack.melee, toAttacker, attack.hitsLeft)
            defend.melee = lose(defend.melee, toDefender, defend.hitsLeft)
            attack.hitsLeft -= toAttacker
            defend.hitsLeft -= toDefender
        }
    }
    for attack.melee > 10 && defend.melee > 10 {
        castMelee(&attackerSide, &attack, &spentAttacker)
        castMelee(&defenderSide, &defend, &spentDefender)
        toDefender := (attack.melee + attack.spellAttack) * strategicRoll(10) / 100
        toAttacker := (defend.melee + defend.spellAttack) * strategicRoll(10) / 100
        attack.hitsLeft += attack.spellHealing
        defend.hitsLeft += defend.spellHealing
        attack.melee = lose(attack.melee, toAttacker, attack.hitsLeft)
        defend.melee = lose(defend.melee, toDefender, defend.hitsLeft)
        attack.hitsLeft -= toAttacker
        defend.hitsLeft -= toDefender
    }

    // the winner; when both sides fall in the same round the original reads a winner it never set:
    // nobody wins and both sides lose every unit (MY CALL for a value the original leaves to chance)
    var winner, loser *Army
    var winnerSums strategicSums
    winnerIsDefender := false
    switch {
        case attack.melee > 10: winner, loser, winnerSums = attackingArmy, defendingArmy, attack
        case defend.melee > 10: winner, loser, winnerSums, winnerIsDefender = defendingArmy, attackingArmy, defend, true
    }

    killAll := func(army *Army) int {
        count := 0
        for _, unit := range army.units {
            if unit.GetHealth() > 0 {
                unit.TakeDamage(unit.GetHealth(), DamageNormal)
                count += 1
            }
        }
        return count
    }

    lostAttackers, lostDefenders := 0, 0
    if winner == nil {
        lostAttackers = killAll(attackingArmy)
        lostDefenders = killAll(defendingArmy)
        return CombatStateDefenderWin, lostAttackers, lostDefenders, spentAttacker, spentDefender
    }
    if winnerIsDefender {
        lostAttackers = killAll(loser)
    } else {
        lostDefenders = killAll(loser)
    }

    // the winner's losses: as much of its hits as it lost of its weighed hits, 3 at a time
    percent := 0
    if winnerSums.hitsStart > 0 {
        percent = 100 * winnerSums.hitsLeft / winnerSums.hitsStart
    }
    floor := percent * winnerSums.rawHits / 100
    countdown := winnerSums.rawHits
    weights := make([]int, len(winner.units))
    for index, unit := range winner.units {
        if unit.GetHealth() > 0 && !(structure == StructureOcean && strategicOffWater(unit)) {
            weights[index] = strategicWeight(unit, winnerIsDefender, structure)
        }
    }
    repeat := -1
    for countdown > floor && len(weights) > 0 {
        target := repeat
        if target < 0 {
            target = strategicWeightedChoice(weights)
        }
        weights[target] += 50
        unit := winner.units[target]
        alive := unit.GetHealth() > 0
        if alive {
            unit.TakeDamage(min(3, unit.GetHealth()), DamageNormal)
        }
        if unit.GetHealth() <= 0 {
            weights[target] = 0
            repeat = -1
            if alive && winnerIsDefender {
                lostDefenders += 1
            } else if alive {
                lostAttackers += 1
            }
        } else {
            repeat = target
        }
        countdown -= 3
    }

    if winnerIsDefender {
        return CombatStateDefenderWin, lostAttackers, lostDefenders, spentAttacker, spentDefender
    }
    return CombatStateAttackerWin, lostAttackers, lostDefenders, spentAttacker, spentDefender
}
