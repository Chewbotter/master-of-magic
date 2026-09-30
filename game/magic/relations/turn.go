package relations

// The diplomacy of a turn, in the original's order (the ReMoM project's reconstruction,
// MoM/src/NEXTTURN.c Next_Turn_Calc; MoM/src/DIPLOMAC.c: Decrease_Peace_Duration, Diplomacy_Growth,
// Determine_First_Contacts, NPC_To_NPC_Treaty_Negotiations, NPC_To_Human_Diplomacy,
// Resolve_Delayed_Diplomacy_Orders, End_Of_Turn_Diplomacy_Adjustments,
// Modifier_Diplomacy_Adjustments; MoM/src/NEXTTURN.c and Spells128.c:
// Diplomacy_Growth_For_Enchantments, Change_Relations_For_Enchantments; MoM/src/AIDATA.c:
// AI_Evaluate_Hostility). The code is ours.

import (
    "github.com/kazzmir/master-of-magic/game/magic/data"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

// how a global enchantment of a wizard changes the others' relations to it: when it is cast, and a
// tenth of it every turn it lasts (Change_Relations_For_Enchantments)
type enchantmentReaction struct {
    Value int
    // only wizards with books of these realms mind it; none: all
    Minds []data.MagicType
    // only wizards without books of this realm mind it
    MindsWithout data.MagicType
}

var enchantmentReactions = map[data.Enchantment]enchantmentReaction{
    data.EnchantmentEternalNight: {Value: -12},
    data.EnchantmentEvilOmens: {Value: -20, Minds: []data.MagicType{data.LifeMagic, data.NatureMagic}},
    data.EnchantmentZombieMastery: {Value: -14},
    data.EnchantmentAuraOfMajesty: {Value: 10},
    data.EnchantmentWindMastery: {Value: -4},
    data.EnchantmentSuppressMagic: {Value: -4},
    data.EnchantmentNaturesWrath: {Value: -20, Minds: []data.MagicType{data.ChaosMagic, data.DeathMagic}},
    data.EnchantmentChaosSurge: {Value: -10, MindsWithout: data.ChaosMagic},
    data.EnchantmentDoomMastery: {Value: -8},
    data.EnchantmentGreatWasting: {Value: -20},
    data.EnchantmentMeteorStorm: {Value: -15},
    data.EnchantmentArmageddon: {Value: -25},
    data.EnchantmentTranquility: {Value: -20, Minds: []data.MagicType{data.ChaosMagic}},
    data.EnchantmentLifeForce: {Value: -20, Minds: []data.MagicType{data.DeathMagic}},
    data.EnchantmentCrusade: {Value: -10},
    data.EnchantmentHolyArms: {Value: -5},
}

// the Spell of Mastery: -50 when it is started, -25 every turn while it is cast
const spellOfMasteryReaction = -50

func books(player *playerlib.Player, magic data.MagicType) int {
    count := 0
    for _, book := range player.Wizard.Books {
        if book.Magic == magic {
            count += book.Count
        }
    }
    return count
}

// Change_Relations_For_Enchantments: every other wizard reacts to a global enchantment of the
// caster; divisor 1 when it is cast, 10 every turn after
func (rules *Rules) EnchantmentCast(caster *playerlib.Player, enchantment data.Enchantment, divisor int) {
    reaction, ok := enchantmentReactions[enchantment]
    if !ok {
        return
    }
    kind := ActionEnchantment
    if reaction.Value > 0 {
        kind = ActionGoodwill
    }
    for _, other := range rules.wizards() {
        if other == caster {
            continue
        }
        minds := len(reaction.Minds) == 0
        for _, magic := range reaction.Minds {
            if books(other, magic) > 0 {
                minds = true
            }
        }
        if reaction.MindsWithout != data.MagicNone && books(other, reaction.MindsWithout) > 0 {
            minds = false
        }
        if minds {
            rules.ChangeRelations(reaction.Value / divisor, caster, other, kind, "", enchantment.String())
        }
    }
}

// the Spell of Mastery: every other wizard's relation to the caster; divisor 1 when it is started,
// 2 every turn while it is cast
func (rules *Rules) MasteryCast(caster *playerlib.Player, divisor int) {
    for _, other := range rules.wizards() {
        if other != caster {
            rules.ChangeRelations(spellOfMasteryReaction / divisor, caster, other, ActionSpellOfMastery, "", "Spell of Mastery")
        }
    }
}

// the diplomacy of one turn, after every player has had its turn. timeStop: no diplomacy at all
func (rules *Rules) EndOfTurn(timeStop bool) {
    for _, player := range rules.computerWizards() {
        if isActive(player) {
            rules.EvaluateHostility(player)
        }
    }
    if timeStop {
        return
    }

    rules.decreasePeace()
    rules.growthForEnchantments()
    rules.growth()
    rules.firstContacts()

    human := rules.human()
    humanAway := human != nil && human.Banished
    if !humanAway {
        rules.computerToComputer()
        rules.computerToHuman()
        rules.tellHuman()
    }

    rules.endOfTurnAdjustments()
    rules.modifierAdjustments()
}

// Decrease_Peace_Duration
func (rules *Rules) decreasePeace() {
    for _, player := range rules.wizards() {
        for _, relation := range player.PlayerRelations {
            relation.PeaceCounter = max(relation.PeaceCounter - 1, 0)
        }
    }
}

// Diplomacy_Growth_For_Enchantments: a tenth of an enchantment's reaction every turn it lasts, half
// the Spell of Mastery's while it is cast
func (rules *Rules) growthForEnchantments() {
    for _, caster := range rules.wizards() {
        if !isActive(caster) {
            continue
        }
        if caster.GlobalEnchantments != nil {
            for _, enchantment := range caster.GlobalEnchantments.Values() {
                rules.EnchantmentCast(caster, enchantment, 10)
            }
        }
        if caster.CastingSpell.Name == "Spell of Mastery" {
            rules.MasteryCast(caster, 2)
        }
    }
}

// Diplomacy_Growth: goodwill of treaties; the others' fear of a human army larger than theirs; the
// dislike of a wizard with more cities than the land holds; the drift of every relation back to
// where it started; and the same relation both ways
func (rules *Rules) growth() {
    wizards := rules.wizards()

    for i := 0; i < len(wizards); i++ {
        for j := i + 1; j < len(wizards); j++ {
            if roll(2) != 1 {
                continue
            }
            switch treaty(wizards[i], wizards[j]) {
                case data.TreatyPact: rules.ChangeRelations(roll(3), wizards[i], wizards[j], ActionGoodwill, "", "")
                case data.TreatyAlliance: rules.ChangeRelations(roll(6), wizards[i], wizards[j], ActionGoodwill, "", "")
            }
        }
    }

    // a human army larger than a wizard's: the larger it is, the less often (as the original has
    // it; the strategy guide says the other way round)
    if human := rules.human(); human != nil && rules.ArmyStrength != nil {
        humanArmy := rules.ArmyStrength(human)
        if humanArmy > 0 {
            for _, other := range wizards {
                army := rules.ArmyStrength(other)
                if other == human || army <= 0 || army >= humanArmy {
                    continue
                }
                threshold := humanArmy * 50 / army
                if roll(100) >= threshold && roll(20) == 1 {
                    rules.ChangeRelations(-10, human, other, ActionConquest, "", "")
                }
            }
        }
    }

    // more cities than (land size + 1) * 3: the others like the wizard less, every second turn one
    // time in 4 (the original counts the human's cities for every wizard; here each its own)
    if rules.Turn % 2 == 0 {
        allowed := (rules.LandSize + 1) * 3
        for _, player := range wizards {
            cities := len(player.Cities)
            if cities <= allowed || roll(4) != 1 {
                continue
            }
            reaction := (-(cities - allowed) / (rules.LandSize + 6 - rules.Difficulty)) * roll(4) / 3
            reaction *= 4
            if !player.IsHuman() {
                reaction /= 2
            }
            reaction = max(reaction, -15)
            for _, other := range wizards {
                if other == player || other.IsHuman() {
                    continue
                }
                if relation := view(player, other); relation != nil && relation.Treaty != data.TreatyAlliance {
                    rules.ChangeRelations(reaction, player, other, ActionOverextension, "", "")
                }
            }
        }
    }

    // the drift back to the starting relation: up any turn, down every tenth; slower the stronger
    // the feeling
    for _, player := range wizards {
        if player.IsHuman() {
            continue
        }
        for _, other := range wizards {
            relation := view(player, other)
            if relation == nil {
                continue
            }
            drift := 0
            if roll(105) > abs(relation.VisibleRelation) {
                drift = roll(2)
            }
            if relation.ActionStrength != 0 {
                continue
            }
            if relation.VisibleRelation < relation.StartingRelation {
                relation.VisibleRelation = min(relation.VisibleRelation + drift, relation.StartingRelation)
            } else if rules.Turn % 10 == 0 {
                relation.VisibleRelation = max(relation.VisibleRelation - drift, relation.StartingRelation)
            }
            relation.VisibleRelation = clamp(relation.VisibleRelation, -100, 100)
        }
    }

    // the same both ways: the later wizard's view is taken
    for _, player := range wizards {
        for _, other := range wizards {
            if from, to := view(other, player), view(player, other); from != nil && to != nil {
                to.VisibleRelation = from.VisibleRelation
            }
        }
    }
}

// Determine_First_Contacts: a computer wizard the human has met is greeted, with no treaty
func (rules *Rules) firstContacts() {
    human := rules.human()
    if human == nil {
        return
    }
    for _, other := range rules.computerWizards() {
        humanView, otherView := view(human, other), view(other, human)
        if humanView == nil || otherView == nil || humanView.ContactStage != 0 {
            continue
        }
        humanView.ContactStage = 1
        humanView.Treaty = data.TreatyNone
        otherView.Treaty = data.TreatyNone
    }
}

// End_Of_Turn_Diplomacy_Adjustments: every patience comes back by 10 a turn while below 100
func (rules *Rules) endOfTurnAdjustments() {
    for _, player := range rules.wizards() {
        for _, relation := range player.PlayerRelations {
            if relation.TreatyInterest < 100 {
                relation.TreatyInterest += 10
            }
            if relation.TradeInterest < 100 {
                relation.TradeInterest += 10
            }
            if relation.PeaceInterest < 100 {
                relation.PeaceInterest += 10
            }
        }
    }
}

// Modifier_Diplomacy_Adjustments: toward computer wizards the patience comes back by 1 to 5 more
// while below 50 and 1 to 5 more while below 0; the will for peace stays within -200 and 200. Then
// the grievances and orders of the turn are forgotten
func (rules *Rules) modifierAdjustments() {
    for _, player := range rules.wizards() {
        for other, relation := range player.PlayerRelations {
            if other.IsHuman() {
                continue
            }
            for _, value := range []*int{&relation.TreatyInterest, &relation.TradeInterest, &relation.PeaceInterest} {
                if *value < 0 {
                    *value += roll(5)
                }
                if *value < 50 {
                    *value += roll(5)
                }
            }
            relation.PeaceInterest = clamp(relation.PeaceInterest, -200, 200)
        }
    }
    for _, player := range rules.wizards() {
        for _, relation := range player.PlayerRelations {
            relation.ActionStrength = 0
            relation.Action = ActionNone
            relation.ActionSpell = ""
            relation.ActionCity = ""
            relation.OfferGold = 0
            relation.OfferSpell = ""
            relation.ExchangeSpells = nil
            relation.WantSpell = ""
        }
    }
}

// Resolve_Delayed_Diplomacy_Orders: what every computer wizard has to say to the human this turn
func (rules *Rules) tellHuman() {
    human := rules.human()
    if human == nil || rules.Tell == nil {
        return
    }
    for _, other := range rules.computerWizards() {
        relation := view(human, other)
        if relation == nil || relation.Action == ActionNone {
            continue
        }
        rules.Tell(Message{
            From: other,
            Action: relation.Action,
            Strength: relation.ActionStrength,
            Spell: relation.ActionSpell,
            City: relation.ActionCity,
            OfferGold: relation.OfferGold,
            OfferSpell: relation.OfferSpell,
            ExchangeSpells: append([]string(nil), relation.ExchangeSpells...),
            WantSpell: relation.WantSpell,
            Treaty: relation.Treaty,
            BrokenTreaty: relation.BrokenTreaty,
        })
    }
}

// AI_Evaluate_Hostility: from turn 100, every 16 to 25 turns (and later after anything that changed
// the relations). The neutral player 3; a wizard at war 3, and a maniacal (4 in 10), ruthless
// (2 in 10) or chaotic (1 in 10) one sometimes 4, a holy war; one with a treaty or peace 0 unless it
// casts the Spell of Mastery; any other one it has met 2 by a chance that grows the worse the
// relation is, the more warlike its personality and the less it trusts
func (rules *Rules) EvaluateHostility(player *playerlib.Player) {
    if rules.Turn < 100 {
        return
    }
    player.HostilityCountdown -= 1
    if player.HostilityCountdown > 0 {
        return
    }
    resetHostilityCountdown(player)
    player.RaiderHostility = 3

    for other, relation := range player.PlayerRelations {
        relation.Hostility = 0
        if !isWizard(other) {
            continue
        }
        if relation.Treaty == data.TreatyWar {
            relation.Hostility = 3
            chance := 0
            switch player.Personality {
                case playerlib.PersonalityManiacal: chance = 4
                case playerlib.PersonalityRuthless: chance = 2
                case playerlib.PersonalityChaotic: chance = 1
            }
            if chance > 0 && roll(10) <= chance {
                relation.Hostility = 4
            }
            continue
        }
        masteryCaster := other.CastingSpell.Name == "Spell of Mastery"
        if (relation.Treaty == data.TreatyAlliance || relation.Treaty == data.TreatyPact || relation.PeaceCounter != 0) && !masteryCaster {
            continue
        }
        var chance int
        if relation.VisibleRelation < 0 {
            chance = (50 - relation.VisibleRelation) / 2
        } else {
            chance = (100 - relation.VisibleRelation) / 4
        }
        if player.Personality == playerlib.PersonalityLawful && relation.HiddenRelation < 0 {
            chance -= personalityWarBonus[playerlib.PersonalityLawful]
        } else {
            chance += personalityWarBonus[player.Personality]
        }
        chance += relation.HiddenRelation
        if masteryCaster {
            chance += 50
        }
        if roll(100) < chance {
            relation.Hostility = 2
        }
    }
}

// Update_Defender_Hostility: a wizard that was attacked is hostile to the attacker (2, at war 3, an
// ally 0) and looks at its hostility again 16 to 25 turns later
func (rules *Rules) WasAttacked(defender *playerlib.Player, attacker *playerlib.Player) {
    relation := view(defender, attacker)
    if relation == nil {
        return
    }
    relation.Hostility = 2
    switch relation.Treaty {
        case data.TreatyWar: relation.Hostility = 3
        case data.TreatyAlliance: relation.Hostility = 0
    }
    resetHostilityCountdown(defender)
}
