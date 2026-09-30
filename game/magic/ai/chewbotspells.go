package ai

// Chewbot's spells of the world map: what a computer wizard researches, how it splits its power,
// which spell it casts next and at what (the target is picked when the casting is done, as the
// original does; game/aicast.go asks ChooseSpellTarget). Facts from the ReMoM project's
// reconstruction (MoM/src/AISPELL.c: AI_Spell_Research_Select, AI_Spell_Select,
// AI_Select_Spell_Group, AI_Compute_Spells_Info, the AI_Select_Spell_Group_* pickers and the
// AITP_* target pickers, AI_Sanity_Check_Overland_Enchantments; MoM/src/AIDUDES.c:
// AI_Update_Magic_Power, Player_Hostile_Opponents; MoM/src/AIDATA.c:
// AI_Evaluate_Magic_Power_Strategy; MoM/src/OverSpel.c: Cast_Spell_Overland). The code is ours.
// The rules in words: docs/mod/ai-spells.md. The pickers of the groups are in the files beside
// this one (chewbotspell*.go).

import (
    "log"
    "slices"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/game/magic/setup"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

// false: the clone's spells whatever the setting says
var ChewbotSpells = true

// development: every choice of a spell in the log
var ChewbotSpellLog = false

func chewbotSpellsActive() bool {
    return ChewbotSpells && display.ChewbotAI()
}

// the original's mistakes in these rules that change what is seen, kept on purpose
const (
    // AITP_Disjunction keeps its scores in signed bytes: 200 and 250 are below 0, so Disjunction
    // never aims at Suppress Magic, Evil Omens, Nature's Wrath, Tranquility, Life Force (and Eternal
    // Night for a caster of life)
    quirkDisjunctionBytes = true
    // AITP_Node takes only nodes that are warped already, of an owner by a number that is not a
    // wizard of the hostile list: the AI's Warp Node never warps a node
    quirkWarpNodeWarped = true
    // AITP_Attack_Wizard aims one time in 2 at the first hostile wizard, meant: the human
    quirkAttackWizardFirst = true
    // AI_Select_Spell_Group weighs the spells against the realms of the human, whoever the rival is
    quirkSuppressionHumanRealms = true
)

// the groups of spells of the original (spell_data_table AI_Group), as far as the AI's choice needs
// them
const (
    chewGroupScouting = 0
    chewGroupDispel = 1
    chewGroupResistBuff1 = 4
    chewGroupCombatHarm1 = 5
    chewGroupMoveBuff1 = 6
    chewGroupAttackBuff1 = 7
    chewGroupWallsRituals = 8
    chewGroupDefenseBuff1 = 10
    chewGroupDisrupt = 12
    chewGroupCorruption = 14
    chewGroupCombatSummon1 = 16
    chewGroupCE1 = 17
    chewGroupHeals = 18
    chewGroupSpellLock = 19
    chewGroupDispelTrue = 20
    chewGroupCombatCurse = 24
    chewGroupMoveBuff2 = 25
    chewGroupAttackBuff2 = 26
    chewGroupCityBuffWalls = 27
    chewGroupOverlandDamage = 28
    chewGroupImmolInvuln = 29
    chewGroupCombatMisc = 31
    chewGroupFamine = 33
    chewGroupCombatSummon2 = 35
    chewGroupCE2 = 36
    chewGroupTerrainMod = 37
    chewGroupHarmWizard = 38
    chewGroupStasis = 39
    chewGroupWarpNode = 41
    chewGroupNine = 9
    chewGroupCombatHarm2 = 47
    chewGroupMoveBuff3 = 48
    chewGroupFifty = 50
    chewGroupCityBuff1 = 51
    chewGroupEarthquake = 52
    chewGroupDefenseBuff2 = 53
    chewGroupHaste = 55
    chewGroupRaiseVolcano = 57
    chewGroupCombatSummon3 = 59
    chewGroupCE3 = 60
    chewGroupCityBuff2 = 66
    chewGroupCombatHarm3 = 70
    chewGroupGates = 71
    chewGroupAttackBuff3 = 72
    chewGroupCityBuff3 = 73
    chewGroupSeventyFour = 74
    chewGroupRegenMassInvis = 75
    chewGroupAnimateDead = 77
    chewGroupDestroyCity = 79
    chewGroupCombatSummon4 = 81
    chewGroupCE4 = 82
    chewGroupUnsummon = 83
    chewGroupTimeStop = 84
    chewGroupDeathWish = 86
    chewGroupAttackBuff4 = 0x32
    chewGroupAttackBuff5 = 0x48
)

// the groups of spells the AI never casts on the world map (AI_Compute_Spells_Info)
var chewCombatGroups = []int{
    chewGroupCombatHarm1, chewGroupCombatCurse, chewGroupCombatHarm2, chewGroupCombatHarm3,
    chewGroupDisrupt, chewGroupCombatMisc, chewGroupHaste, chewGroupAnimateDead,
    chewGroupCombatSummon1, chewGroupCombatSummon2, chewGroupCombatSummon3, chewGroupCombatSummon4,
    chewGroupCE1, chewGroupCE2, chewGroupCE3, chewGroupCE4, chewGroupDispel, chewGroupDispelTrue,
    43, chewGroupSpellLock, chewGroupScouting,
}

// the kinds of spell AI_Select_Spell_Group picks among
const (
    chewSpellNone = iota
    chewSpellSummon
    chewSpellUnit
    chewSpellCity
    chewSpellDisenchant
    chewSpellDisjunction
    chewSpellSummoningCircle
    chewSpellAttack
    chewSpellSuppression
    chewSpellGlobal
    chewSpellMastery
    chewSpellKinds
)

// the realms of the original's order (sbr_*), for the primary and secondary realm of a wizard
var chewRealmOrder = []data.MagicType{data.NatureMagic, data.SorceryMagic, data.ChaosMagic, data.LifeMagic, data.DeathMagic}

// Prim_Realm and Sec_Realm: the realms of most and second most books; one realm is both; no books
// is nature. On a tie the first of nature, sorcery, chaos, life, death
func chewRealms(wizard setup.WizardCustom) (data.MagicType, data.MagicType) {
    count := func(magic data.MagicType) int {
        total := 0
        for _, book := range wizard.Books {
            if book.Magic == magic {
                total += book.Count
            }
        }
        return total
    }
    primary, secondary := data.NatureMagic, data.MagicType(-1)
    most := 0
    for _, realm := range chewRealmOrder {
        if count(realm) > most {
            most = count(realm)
            primary = realm
        }
    }
    next := 0
    for _, realm := range chewRealmOrder {
        if realm != primary && count(realm) > next {
            next = count(realm)
            secondary = realm
        }
    }
    if secondary < 0 {
        secondary = primary
    }
    return primary, secondary
}

func chewHasRealm(wizard setup.WizardCustom, realm data.MagicType) bool {
    primary, secondary := chewRealms(wizard)
    return primary == realm || secondary == realm
}

// the state of a wizard's magic from turn to turn (the original keeps it in the player; not saved)
type chewMagic struct {
    // AI_Update_Magic_Power, AI_Evaluate_Magic_Power_Strategy: turns until they look again
    PowerCountdown int
    StrategyCountdown int
    // magic_power_strategy: 1 research, 2 skill, 3 summon, 4 war, 5 against the Spell of Mastery
    Strategy int
    // the ratios of power (percent)
    ManaRatio int
    ResearchRatio int
    SkillRatio int
    // cp_target_3: the hostile wizard the attacks of this turn are aimed at
    Target *playerlib.Player
    // _ai_reevaluate_summoning_circle_countdown
    CircleCountdown int
}

func (ai *ChewbotAI) magicState() *chewMagic {
    if ai.magic == nil {
        ai.magic = &chewMagic{}
    }
    return ai.magic
}

// the spell of this name the wizard knows; false: it does not
func chewKnows(self *playerlib.Player, name string) (spellbook.Spell, bool) {
    spell := self.KnownSpells.FindByName(name)
    return spell, spell.Valid()
}

// the original's casting cost of a known spell; 0 for one it does not know
func chewCost(self *playerlib.Player, name string) int {
    spell, ok := chewKnows(self, name)
    if !ok {
        return 0
    }
    return spell.CastCost
}

// the groups of the spells the wizard knows and could cast on the world map (AI_Compute_Spells_Info)
func chewKnownGroups(self *playerlib.Player) map[int]bool {
    groups := make(map[int]bool)
    for _, spell := range self.KnownSpells.Spells {
        if slices.Contains(chewCombatGroups, spell.AiGroup) {
            continue
        }
        groups[spell.AiGroup] = true
    }
    return groups
}

// Player_Hostile_Opponents: the other wizards the wizard is hostile to (2 or more) and has no
// peace with, in the game's order (the human first)
func chewHostileOpponents(self *playerlib.Player, services playerlib.AIServices) []*playerlib.Player {
    var out []*playerlib.Player
    for _, other := range services.GetEnemies(self) {
        if other.IsNeutral() {
            continue
        }
        relation, ok := self.PlayerRelations[other]
        if !ok || relation.Hostility < 2 || relation.PeaceCounter > 0 {
            continue
        }
        out = append(out, other)
    }
    return out
}

// the human player; nil when there is none
func chewHuman(self *playerlib.Player, services playerlib.AIServices) *playerlib.Player {
    if self.IsHuman() {
        return self
    }
    for _, other := range services.GetEnemies(self) {
        if other.IsHuman() {
            return other
        }
    }
    return nil
}

// the original's five levels, 0 intro to 4 impossible
func chewDifficulty(services playerlib.AIServices) int {
    switch services.GetDifficulty() {
        case data.DifficultyIntro: return 0
        case data.DifficultyEasy: return 1
        case data.DifficultyAverage: return 2
        case data.DifficultyHard: return 3
    }
    return 4
}

// the decisions of the wizard's magic this turn: research, the split of its power, the spell to
// cast; the clone's decisions of the same kind are dropped by Update
func (ai *ChewbotAI) spellDecisions(self *playerlib.Player, services playerlib.AIServices) []playerlib.AIDecision {
    var out []playerlib.AIDecision
    state := ai.magicState()

    ai.evaluateMagicStrategy(self, services, state)
    ai.updateMagicPower(self, services, state)

    if self.ResearchingSpell.Invalid() {
        if spell, ok := ai.pickResearch(self, services, state); ok {
            out = append(out, &playerlib.AIResearchSpellDecision{Spell: spell})
        }
    }

    if self.CastingSpell.Invalid() {
        if spell, ok := ai.pickSpell(self, services, state); ok {
            out = append(out, &playerlib.AICastSpellDecision{Spell: spell})
        }
    }

    ai.sanityCheckEnchantments(self)
    return out
}

func chewSpellLog(self *playerlib.Player, format string, args ...any) {
    if ChewbotSpellLog {
        log.Printf("chewbot spells: %v: "+format, append([]any{self.Wizard.Name}, args...)...)
    }
}
