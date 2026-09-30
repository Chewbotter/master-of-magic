package ai

// What Chewbot researches and how it splits its power between mana, research and casting skill,
// and the globals it drops at the end of its turn (ReMoM MoM/src/AISPELL.c:
// AI_Spell_Research_Select, AI_Sanity_Check_Overland_Enchantments; MoM/src/AIDUDES.c:
// AI_Update_Magic_Power; MoM/src/AIDATA.c: AI_Evaluate_Magic_Power_Strategy). The code is ours.

import (
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

// AI_Spell_Research_Select: every candidate weighs its research cost, made larger for a group the
// wizard knows a spell of (12 times), smaller for the groups of battle (a fifth; summons of battle a
// tenth or less), for Militarists' summons of battle (half), for Spell Ward (and Consecration for
// sorcery and chaos wizards) (half), Disenchant and Disjunction (half), their True kinds and Summon
// Hero (a fifth), larger for Summoning Circle (twice) and for spells of more than 20 turns of research
// (twice) or 50 (five times more); then 52 less 50 times its part of the largest, at least 1
func (ai *ChewbotAI) pickResearch(self *playerlib.Player, services playerlib.AIServices, state *chewMagic) (spellbook.Spell, bool) {
    candidates := self.ResearchCandidateSpells.Spells
    if len(candidates) == 0 {
        return spellbook.Spell{}, false
    }
    known := make(map[int]bool)
    for _, spell := range self.KnownSpells.Spells {
        known[spell.AiGroup] = true
    }
    research := int(self.SpellResearchPerTurn(services.ComputePower(self)))
    primary, _ := chewRealms(self.Wizard)
    combatSummons := map[int]bool{chewGroupCombatSummon1: true, chewGroupCombatSummon2: true, chewGroupCombatSummon3: true, chewGroupCombatSummon4: true}
    combat := map[int]bool{
        chewGroupCombatHarm1: true, chewGroupCombatCurse: true, chewGroupCombatHarm2: true, chewGroupCombatHarm3: true,
        chewGroupCE1: true, chewGroupCE2: true, chewGroupCE3: true, chewGroupCE4: true, chewGroupDisrupt: true,
        chewGroupCombatMisc: true, chewGroupHaste: true, chewGroupAnimateDead: true,
    }
    for group := range combatSummons {
        combat[group] = true
    }

    weights := make([]int, len(candidates))
    highest := 0
    // AI_Spell_Research_Select: the cost with the wizard's research bonus, which the original works
    // out for a spell number that is the realm's number (the first spells of the book, of nature):
    // Sage Master, Nature Mastery and nature books beyond 7, whatever the spell's realm (kept)
    bonus := 0
    if self.Wizard.RetortEnabled(data.RetortSageMaster) {
        bonus += 25
    }
    if self.Wizard.RetortEnabled(data.RetortNatureMastery) {
        bonus += 15
    }
    bonus += 10 * max(0, self.Wizard.MagicLevel(data.NatureMagic) - 7)
    for index, spell := range candidates {
        cost := spell.ResearchCost * 100 / (100 + bonus)
        weight := cost
        group := spell.AiGroup
        if known[group] {
            weight *= 4
        } else {
            if combatSummons[group] {
                weight /= 10
            }
            if combat[group] {
                weight /= 5
            }
        }
        if self.Objective == playerlib.ObjectiveMilitarist && combatSummons[group] {
            weight /= 2
        }
        if !known[group] {
            switch group {
                case chewGroupCombatSummon1: weight /= 10
                case chewGroupCombatSummon2: weight /= 7
                case chewGroupCombatSummon3: weight /= 5
                case chewGroupCombatSummon4: weight /= 3
            }
        }
        // the original's test reads sorcery (1) or chaos
        if primary == data.SorceryMagic || primary == data.ChaosMagic {
            if spell.Name == "Consecration" || spell.Name == "Spell Ward" {
                weight /= 2
            }
        } else if spell.Name == "Spell Ward" {
            weight /= 2
        }
        switch spell.Name {
            case "Disenchant Area", "Disjunction": weight /= 2
            case "Disenchant True", "Disjunction True", "Summon Hero": weight /= 5
            case "Summoning Circle": weight *= 2
        }
        if known[group] {
            weight *= 3
        }
        if research > 0 && cost / 20 > research {
            weight *= 2
        }
        if research > 0 && cost / 50 > research {
            weight *= 5
        }
        if weight < 1 && cost > 0 {
            weight = 1
        }
        if group == 0 {
            weight = 0
        }
        weights[index] = weight
        highest = max(highest, weight)
    }
    for index := range weights {
        if weights[index] != 0 && highest > 0 {
            weights[index] = max(52 - weights[index] * 50 / highest, 1)
        }
    }
    choice := chewWeightedChoiceLong(weights)
    chewSpellLog(self, "researches %v", candidates[choice].Name)
    return candidates[choice], true
}

// Get_Weighted_Choice_Long: as Get_Weighted_Choice, with no halving
func chewWeightedChoiceLong(weights []int) int {
    total := 0
    for _, weight := range weights {
        total += weight
    }
    if total <= 0 {
        return 0
    }
    roll := 1 + chewRoll(total) - weights[0]
    choice := 0
    for roll > 0 && choice < len(weights) - 1 {
        choice += 1
        roll -= weights[choice]
    }
    return choice
}

// AI_Evaluate_Magic_Power_Strategy, every 16 to 25 turns: research (1), or war (4) when it is
// hostile (3) to someone, or against a Spell of Mastery or an equal (5) when it knows Spell Blast,
// Drain Power or Cruel Unminding. (The original's measures of skill and strength come first, but its
// test of how many spells it knows is always true and overwrites them with research.)
func (ai *ChewbotAI) evaluateMagicStrategy(self *playerlib.Player, services playerlib.AIServices, state *chewMagic) {
    state.StrategyCountdown -= 1
    if state.StrategyCountdown > 0 {
        return
    }
    state.StrategyCountdown = 15 + 1 + chewRoll(10)
    strategy := 1
    for _, relation := range self.PlayerRelations {
        if relation.Hostility >= 3 {
            strategy = 4
        }
    }
    if chewKnowsAny(self, "Spell Blast", "Drain Power", "Cruel Unminding") {
        human := chewHuman(self, services)
        if human != nil {
            if human.CastingSpell.Name == "Spell of Mastery" {
                strategy = 5
            }
            if relation, ok := self.PlayerRelations[human]; ok && relation.Hostility == 2 {
                cruel := chewCost(self, "Cruel Unminding")
                if cruel == 0 {
                    cruel = 1
                }
                if self.CastingSpell.Valid() && self.CastingSpell.CastCost >= cruel {
                    strategy = 5
                }
            }
        }
    }
    state.Strategy = strategy
}

// AI_Update_Magic_Power, every 16 to 25 turns: research, skill 10 and mana 30, then 5 times 10 more
// to one of the three by chance, slanted by the objective (Theurgist research, Militarist skill,
// Perfectionist mana), a skill above 100 (research), Alchemy (mana), the strategy, a low reserve
// (mana), a reserve above 1000 (research or skill), and no skill beyond 4 times the reserve. With
// nothing left to research, its research goes half to skill. Researching the Spell of Mastery 70,
// 10, 20; casting it 0 research, 10 skill; before turn 30: research 35, skill 25. Mana is the rest.
func (ai *ChewbotAI) updateMagicPower(self *playerlib.Player, services playerlib.AIServices, state *chewMagic) {
    state.PowerCountdown -= 1
    if state.PowerCountdown <= 0 {
        state.PowerCountdown = 15 + 1 + chewRoll(10)
        knowable := len(self.ResearchCandidateSpells.Spells) > 0 || self.ResearchingSpell.Valid()
        if !knowable {
            state.SkillRatio = state.ResearchRatio / 2
            state.ResearchRatio = 0
        } else {
            state.ResearchRatio, state.SkillRatio, state.ManaRatio = 10, 10, 30
            skill := self.ComputeCastingSkill()
            turn := int(services.GetTurnNumber())
            for range 5 {
                kind := chewRoll(3)
                third := func() bool {
                    return chewRoll(3) == 0
                }
                if self.Objective == playerlib.ObjectiveTheurgist && third() {
                    kind = 2
                }
                if self.Objective == playerlib.ObjectiveMilitarist && third() {
                    kind = 1
                }
                if self.Objective == playerlib.ObjectivePerfectionist && third() {
                    kind = 0
                }
                if skill > 100 && third() {
                    kind = 2
                }
                if self.Wizard.RetortEnabled(data.RetortAlchemy) && third() {
                    kind = 0
                }
                if state.Strategy == 6 && third() {
                    kind = 0
                }
                if state.Strategy == 1 && third() {
                    kind = 2
                }
                if state.Strategy == 2 && third() {
                    kind = 1
                }
                if self.Mana < turn * 3 && chewRoll(2) == 0 {
                    kind = 0
                }
                if self.Mana > 1000 {
                    if chewRoll(2) == 0 {
                        kind = 2
                    } else {
                        kind = 1
                    }
                }
                if kind == 1 && self.Mana * 4 < skill {
                    kind = 0
                }
                switch kind {
                    case 0: state.ManaRatio += 10
                    case 1: state.SkillRatio += 10
                    case 2: state.ResearchRatio += 10
                }
            }
            // the original tests for Fire Elemental here, meant: the Spell of Mastery
            if self.ResearchingSpell.Name == "Spell of Mastery" {
                state.ResearchRatio, state.SkillRatio, state.ManaRatio = 70, 10, 20
            }
            if self.CastingSpell.Name == "Spell of Mastery" {
                state.ResearchRatio, state.SkillRatio, state.ManaRatio = 0, 10, 90
            }
            if turn < 30 {
                state.ResearchRatio, state.SkillRatio, state.ManaRatio = 35, 25, 40
            }
        }
        state.ManaRatio = 100 - state.SkillRatio - state.ResearchRatio
    }
    if state.ManaRatio + state.SkillRatio + state.ResearchRatio == 100 {
        self.PowerDistribution = playerlib.PowerDistribution{
            Mana: float64(state.ManaRatio) / 100,
            Research: float64(state.ResearchRatio) / 100,
            Skill: float64(state.SkillRatio) / 100,
        }
    }
}

// AI_Sanity_Check_Overland_Enchantments: globals a wizard of the wrong realm does not keep, and none
// at all when its mana is gone
func (ai *ChewbotAI) sanityCheckEnchantments(self *playerlib.Player) {
    drop := func(enchantment data.Enchantment, when bool) {
        if when && self.HasEnchantment(enchantment) {
            self.RemoveEnchantment(enchantment)
        }
    }
    has := func(realm data.MagicType) bool {
        return chewHasRealm(self.Wizard, realm)
    }
    drop(data.EnchantmentEternalNight, !has(data.DeathMagic))
    drop(data.EnchantmentEvilOmens, has(data.LifeMagic) || has(data.NatureMagic))
    drop(data.EnchantmentChaosSurge, !has(data.ChaosMagic))
    drop(data.EnchantmentTranquility, has(data.ChaosMagic))
    drop(data.EnchantmentLifeForce, has(data.DeathMagic))
    if self.Mana == 0 {
        for _, enchantment := range chewGlobalSlots {
            drop(enchantment, true)
        }
    }
}
