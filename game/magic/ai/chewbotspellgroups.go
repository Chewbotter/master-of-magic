package ai

// Which spell Chewbot casts next: the kind of spell by weights, then the spell of that kind by
// weights (ReMoM MoM/src/AISPELL.c: AI_Spell_Select, AI_Select_Spell_Group and its
// AI_Select_Spell_Group_* pickers). The rules in words: docs/mod/ai-spells.md. The code is ours.

import (
    "fmt"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    herolib "github.com/kazzmir/master-of-magic/game/magic/hero"
    "github.com/kazzmir/master-of-magic/game/magic/maplib"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/relations"
)

// the items of a computer wizard are not made yet (see summonSpell)
const chewCastsItems = false

// the node is melded by the wizard
func chewNodeOwner(node *maplib.ExtraMagicNode, self *playerlib.Player) bool {
    return node.MeldingWizard != nil && node.MeldingWizard.GetBanner() == self.GetBanner()
}

// the original keeps these on purpose-looking but odd rules; kept where they show
const (
    // AI_Select_Spell_Group: when the weight of the global enchantments is above 400 it becomes 300
    quirkGlobalWeightCut = true
    // AI_Select_Spell_Group_Summon: Storm Giant's weight is not squared as the others are
    quirkStormGiantLinear = true
    // AI_Select_Spell_Group_Summon: Incarnation only when the wizard has Barbarian Spearmen (meant:
    // when it does not have The Chosen)
    quirkIncarnationSpearmen = true
    // AI_Select_Spell_Group_City_Enchantment: Astral Gate weighs 0, so it is never picked
    quirkAstralGateNever = true
)

// SPL_Is*Suppressed: any wizard (the AI too) has the global that suppresses the realm
func chewAnyHas(world *chewSpellWorld, enchantments ...data.Enchantment) bool {
    for _, player := range world.Players {
        for _, enchantment := range enchantments {
            if player.HasEnchantment(enchantment) {
                return true
            }
        }
    }
    return false
}

func chewLifeSuppressed(world *chewSpellWorld) bool {
    return chewAnyHas(world, data.EnchantmentEvilOmens)
}

func chewNatureSuppressed(world *chewSpellWorld) bool {
    return chewAnyHas(world, data.EnchantmentEvilOmens)
}

func chewDeathSuppressed(world *chewSpellWorld) bool {
    return chewAnyHas(world, data.EnchantmentLifeForce, data.EnchantmentNaturesWrath)
}

// Tranquility is not counted, as in the original
func chewChaosSuppressed(world *chewSpellWorld) bool {
    return chewAnyHas(world, data.EnchantmentNaturesWrath)
}

// the damping of a suppressed life or nature spell
func chewTwoThirds(weights []int, slots ...int) {
    for _, slot := range slots {
        weights[slot] = weights[slot] * 2 / 3
    }
}

// the damping of a suppressed death or chaos spell: under 20 none, under 50 a third, else half
func chewBand(weights []int, slots ...int) {
    for _, slot := range slots {
        switch weight := weights[slot]; {
            case weight < 20: weights[slot] = 0
            case weight < 50: weights[slot] = weight / 3
            default: weights[slot] = weight / 2
        }
    }
}

func chewSlotRange(from int, to int) []int {
    var out []int
    for slot := from; slot <= to; slot++ {
        out = append(out, slot)
    }
    return out
}

// a weighted pick of a slot of a table of spells: the spell's name, "" for none
func chewPickSlot(weights []int, names map[int]string) string {
    return names[chewWeightedChoice(weights)]
}

// AI_Spell_Select: the kind of spell, the spell, and whether its mana can be found (a spell of more
// than 50 turns of a tenth of the reserve and the income is not started)
func (ai *ChewbotAI) pickSpell(self *playerlib.Player, services playerlib.AIServices, state *chewMagic) (spellbook.Spell, bool) {
    world := ai.spellWorld(self, services)
    kind := ai.spellKind(self, services, state, world)
    name := ""
    switch kind {
        case chewSpellSummon: name = ai.summonSpell(self, services, world)
        case chewSpellUnit: name = chewUnitSpell(self, world)
        case chewSpellCity: name = ai.citySpell(self, services, world)
        case chewSpellDisenchant: name = chewFirstKnown(self, "Disenchant True", "Disenchant Area")
        case chewSpellDisjunction: name = chewFirstKnown(self, "Disjunction True", "Disjunction")
        case chewSpellSummoningCircle: name = "Summoning Circle"
        case chewSpellAttack: name = chewAttackSpell(self, world)
        case chewSpellSuppression: name = chewSuppressionSpell(self, world)
        case chewSpellGlobal: name = chewGlobalSpell(self, world)
        case chewSpellMastery: name = "Spell of Mastery"
    }
    spell, ok := chewKnows(self, name)
    if name == "" || !ok {
        chewSpellLog(self, "kind %v, no spell", kind)
        return spellbook.Spell{}, false
    }
    power := services.ComputePower(self)
    perTurn := self.Mana / 10 + self.PlanningManaPerTurn(power, services)
    if spell.CastCost / 50 > perTurn {
        chewSpellLog(self, "kind %v, %v is too costly (%v a turn)", kind, name, perTurn)
        playerlib.Note(self, "spell skipped", name, fmt.Sprintf("kind %v, costs %v, more than 50 turns of %v a turn", chewSpellKindName(kind), spell.CastCost, perTurn))
        return spellbook.Spell{}, false
    }
    chewSpellLog(self, "kind %v, casts %v", kind, name)
    playerlib.Note(self, "spell start", name, fmt.Sprintf("kind %v, cost %v, mana %v, %v a turn", chewSpellKindName(kind), spell.CastCost, self.Mana, perTurn))
    return spell, true
}

func chewFirstKnown(self *playerlib.Player, names ...string) string {
    for _, name := range names {
        if _, ok := chewKnows(self, name); ok {
            return name
        }
    }
    return ""
}

func chewKnowsAny(self *playerlib.Player, names ...string) bool {
    return chewFirstKnown(self, names...) != ""
}

// a known global enchantment the wizard does not have yet
func chewWantsGlobal(self *playerlib.Player, name string) bool {
    _, known := chewKnows(self, name)
    return known && !self.HasEnchantment(data.GetEnchantmentByName(name))
}

// Player_Hero_Count
func chewHeroCount(self *playerlib.Player) int {
    count := 0
    for _, hero := range self.Heroes {
        if hero != nil && hero.Status == herolib.StatusEmployed {
            count += 1
        }
    }
    return count
}

// AI_Select_Spell_Group: the weights of the kinds of spell
func (ai *ChewbotAI) spellKind(self *playerlib.Player, services playerlib.AIServices, state *chewMagic, world *chewSpellWorld) int {
    weights := make([]int, chewSpellKinds)
    groups := chewKnownGroups(self)
    primary, secondary := chewRealms(self.Wizard)
    human := world.Human

    // summons
    weights[chewSpellSummon] = 100 + (6 - chewHeroCount(self)) * 10
    if state.Strategy == 3 {
        weights[chewSpellSummon] += 30
    }
    if ai.nodeOpportunity(self, services) {
        weights[chewSpellSummon] += 30
    }
    if self.Wizard.RetortEnabled(data.RetortConjurer) || self.Wizard.RetortEnabled(data.RetortChanneler) {
        weights[chewSpellSummon] = weights[chewSpellSummon] * 3 / 2
    }
    if !chewSummonAtFortress(self) {
        weights[chewSpellSummon] *= 5
    }

    // disenchant
    if chewKnowsAny(self, "Disenchant Area", "Disenchant True") {
        curses := []data.CityEnchantment{data.CityEnchantmentChaosRift, data.CityEnchantmentEvilPresence, data.CityEnchantmentCursedLands, data.CityEnchantmentPestilence, data.CityEnchantmentFamine}
        for _, city := range world.Cities {
            owner := world.Owners[city]
            if owner == self {
                for _, curse := range curses {
                    if city.HasEnchantment(curse) {
                        weights[chewSpellDisenchant] += 40
                    }
                }
            } else if owner == human && city.HasEnchantment(data.CityEnchantmentFlyingFortress) {
                weights[chewSpellDisenchant] += 300
            }
        }
        for _, plane := range []data.Plane{data.PlaneArcanus, data.PlaneMyrror} {
            mapObject := services.GetMap(plane)
            for _, point := range mapObject.GetMagicNodeLocations() {
                node := mapObject.GetMagicNode(point.X, point.Y)
                if node != nil && node.Warped && chewNodeOwner(node, self) {
                    weights[chewSpellDisenchant] += 100
                }
            }
        }
        if fortress := self.FindFortressCity(); fortress != nil && human != nil {
            land := services.GetMap(fortress.Plane).Map.FindContinent(fortress.X, fortress.Y)
            for _, stack := range human.Stacks {
                if stack.Plane() != fortress.Plane || !land.Contains(chewPoint(stack.X(), stack.Y())) {
                    continue
                }
                for _, unit := range stack.Units() {
                    if len(unit.GetEnchantments()) > 0 {
                        weights[chewSpellDisenchant] += 5
                    }
                }
            }
        }
    }

    // disjunction: the globals of every other wizard, by the realms of the one who has them
    if chewKnowsAny(self, "Disjunction", "Disjunction True") {
        add := func(amount int) {
            weights[chewSpellDisjunction] += amount
        }
        for _, other := range world.Players {
            if other == self || other.IsNeutral() {
                continue
            }
            otherPrimary, otherSecondary := chewRealms(other.Wizard)
            has := other.HasEnchantment
            if primary != data.DeathMagic && has(data.EnchantmentEternalNight) {
                add(100)
            }
            if has(data.EnchantmentEvilOmens) {
                if otherPrimary == data.LifeMagic || otherPrimary == data.NatureMagic {
                    add(300)
                }
                if otherSecondary == data.LifeMagic || otherSecondary == data.NatureMagic {
                    add(150)
                }
            }
            if has(data.EnchantmentZombieMastery) {
                add(100)
            }
            if has(data.EnchantmentAuraOfMajesty) {
                // counted twice by the original
                add(50)
            }
            if has(data.EnchantmentWindMastery) {
                add(25)
            }
            if has(data.EnchantmentSuppressMagic) {
                add(2000)
            }
            if has(data.EnchantmentNaturesWrath) {
                if otherPrimary == data.DeathMagic || otherPrimary == data.ChaosMagic {
                    add(400)
                }
                if otherSecondary == data.DeathMagic || otherSecondary == data.ChaosMagic {
                    add(200)
                }
            }
            if has(data.EnchantmentHerbMastery) {
                add(50)
            }
            if has(data.EnchantmentChaosSurge) && otherPrimary != data.ChaosMagic && otherSecondary != data.ChaosMagic {
                add(100)
            }
            if has(data.EnchantmentDoomMastery) {
                add(200)
            }
            if has(data.EnchantmentGreatWasting) {
                add(50)
            }
            if has(data.EnchantmentMeteorStorm) {
                add(100)
            }
            if has(data.EnchantmentArmageddon) {
                add(200)
            }
            if has(data.EnchantmentTranquility) {
                if otherPrimary == data.ChaosMagic {
                    add(500)
                }
                if otherSecondary == data.ChaosMagic {
                    add(250)
                }
            }
            if has(data.EnchantmentLifeForce) {
                if otherPrimary == data.DeathMagic {
                    add(500)
                }
                if otherSecondary == data.DeathMagic {
                    add(250)
                }
            }
            if has(data.EnchantmentCrusade) {
                add(100)
            }
            if has(data.EnchantmentJustCause) {
                add(10)
            }
            if has(data.EnchantmentHolyArms) {
                add(100)
            }
            if has(data.EnchantmentPlanarSeal) {
                if other.IsHuman() || chewRoll(5) == 0 {
                    add(200)
                }
                add(25)
            }
            if has(data.EnchantmentCharmOfLife) {
                add(200)
            }
        }
    }

    if _, ok := chewKnows(self, "Spell of Mastery"); ok {
        weights[chewSpellMastery] = 500
    }

    // the hostile wizard of this turn's attacks
    state.Target = nil
    if len(world.Hostile) > 0 {
        opponents := make([]int, len(world.Hostile))
        difficulty := chewDifficulty(services)
        for index, other := range world.Hostile {
            relation := self.PlayerRelations[other]
            weight := max((100 - relation.VisibleRelation) - relation.HiddenRelation * 2, 0)
            if relation.Treaty == data.TreatyWar {
                weight += 50
            }
            if other.IsHuman() {
                weight *= difficulty + 1
            }
            opponents[index] = weight
        }
        choice := chewWeightedChoice(opponents)
        state.Target = world.Hostile[choice]
        attackGroups := []int{chewGroupCorruption, chewGroupFamine, chewGroupRaiseVolcano, chewGroupDestroyCity, chewGroupWarpNode, chewGroupStasis, chewGroupNine, chewGroupOverlandDamage, chewGroupEarthquake, chewGroupSeventyFour, chewGroupTimeStop, chewGroupUnsummon, chewGroupDeathWish, chewGroupHarmWizard}
        for _, group := range attackGroups {
            if groups[group] {
                weights[chewSpellAttack] = max((opponents[choice] + chewWarBonus(self)) / 4, 0)
                break
            }
        }
    }

    // cities
    if groups[chewGroupWallsRituals] || groups[chewGroupCityBuffWalls] {
        weights[chewSpellCity] = 5
        if human != nil {
            if relation, ok := human.PlayerRelations[self]; ok && relation != nil {
                weights[chewSpellCity] = 50
            }
        }
    }
    if groups[chewGroupCityBuff1] || groups[chewGroupCityBuff3] || groups[chewGroupTerrainMod] || groups[chewGroupCityBuff2] {
        weights[chewSpellCity] = 150
    }

    // units
    turn := world.Turn
    if groups[chewGroupMoveBuff1] || groups[chewGroupMoveBuff2] || groups[chewGroupMoveBuff3] || groups[chewGroupGates] {
        switch {
            case turn < 50: weights[chewSpellUnit] = 0
            case turn < 200: weights[chewSpellUnit] = 100
            default: weights[chewSpellUnit] = 200
        }
    }
    buffGroups := []int{chewGroupAttackBuff1, chewGroupAttackBuff2, chewGroupAttackBuff4, chewGroupAttackBuff5, chewGroupDefenseBuff1, chewGroupImmolInvuln, chewGroupDefenseBuff2, chewGroupRegenMassInvis, chewGroupHeals}
    for _, group := range buffGroups {
        if groups[group] {
            if turn > 20 && turn < 200 {
                weights[chewSpellUnit] += (250 - turn) / 2
            } else {
                weights[chewSpellUnit] += 25
            }
            break
        }
    }

    // globals and suppression; the suppression by the realms of the human
    realmHas := func(realm data.MagicType) bool {
        if quirkSuppressionHumanRealms && human != nil {
            return chewHasRealm(human.Wizard, realm)
        }
        return state.Target != nil && chewHasRealm(state.Target.Wizard, realm)
    }
    global := func(name string, amount int) {
        if chewWantsGlobal(self, name) {
            weights[chewSpellGlobal] += amount
        }
    }
    suppress := func(name string, amount int, realms ...data.MagicType) {
        if !chewWantsGlobal(self, name) {
            return
        }
        for _, realm := range realms {
            if realmHas(realm) {
                weights[chewSpellSuppression] += amount
                return
            }
        }
    }
    global("Herb Mastery", 100)
    suppress("Nature's Wrath", 100, data.DeathMagic, data.ChaosMagic)
    global("Wind Mastery", 75)
    global("Awareness", 500)
    global("Aura of Majesty", 100)
    if chewWantsGlobal(self, "Suppress Magic") {
        weights[chewSpellSuppression] += 100
    }
    if chewKnowsAny(self, "Spell Binding") {
        weights[chewSpellGlobal] += 100
    }
    for _, name := range []string{"Great Wasting", "Chaos Surge", "Doom Mastery", "Meteor Storm", "Armageddon", "Just Cause", "Holy Arms", "Charm of Life", "Crusade"} {
        global(name, 100)
    }
    if chewKnowsAny(self, "Great Unsummoning") {
        weights[chewSpellGlobal] += 100
    }
    if chewKnowsAny(self, "Death Wish") {
        weights[chewSpellGlobal] += 100
    }
    suppress("Life Force", 100, data.DeathMagic)
    suppress("Tranquility", 100, data.ChaosMagic)
    global("Eternal Night", 100)
    suppress("Evil Omens", 100, data.LifeMagic, data.NatureMagic)
    global("Zombie Mastery", 100)
    if quirkGlobalWeightCut && weights[chewSpellGlobal] > 400 {
        weights[chewSpellGlobal] = 300
    }
    if chewWantsGlobal(self, "Planar Seal") && chewMoreMyrrorCities(self, world) {
        weights[chewSpellGlobal] += 50
    }
    if self.Mana < 200 {
        weights[chewSpellGlobal] = 0
        weights[chewSpellSuppression] = 0
    }

    // the summoning circle: the original's target picker for it is not reconstructed, so Chewbot
    // does not cast it (chewSpellSummoningCircle stays 0)

    weights[chewSpellNone] = 1
    _ = secondary
    return chewWeightedChoice(weights)
}

// TBL_AI_PRS_War_Mod of the wizard's personality
func chewWarBonus(self *playerlib.Player) int {
    return relations.WarBonus(self.Personality)
}

// AI_NodeOpportunity: no spirit of the wizard on its summoning plane, and a node there it does not
// have
func (ai *ChewbotAI) nodeOpportunity(self *playerlib.Player, services playerlib.AIServices) bool {
    summon := self.FindSummoningCity()
    if summon == nil {
        return false
    }
    for _, stack := range self.Stacks {
        if stack.Plane() != summon.Plane {
            continue
        }
        for _, unit := range stack.Units() {
            if unit.HasAbility(data.AbilityMeld) {
                return false
            }
        }
    }
    mapObject := services.GetMap(summon.Plane)
    for _, point := range mapObject.GetMagicNodeLocations() {
        node := mapObject.GetMagicNode(point.X, point.Y)
        if node != nil && !chewNodeOwner(node, self) {
            return true
        }
    }
    return false
}

// Player_Summon_City_Is_Fortress_City
func chewSummonAtFortress(self *playerlib.Player) bool {
    fortress := self.FindFortressCity()
    return fortress != nil && fortress == self.FindSummoningCity()
}

// AI_Has_More_Myrror_Cities: more cities on Myrror than the human
func chewMoreMyrrorCities(self *playerlib.Player, world *chewSpellWorld) bool {
    count := func(player *playerlib.Player) int {
        total := 0
        for _, city := range player.Cities {
            if city.Plane == data.PlaneMyrror {
                total += 1
            }
        }
        return total
    }
    human := world.Human
    if human == nil {
        return false
    }
    return count(self) > count(human)
}

// AI_Select_Spell_Group_Summon
func (ai *ChewbotAI) summonSpell(self *playerlib.Player, services playerlib.AIServices, world *chewSpellWorld) string {
    names := map[int]string{
        1: "War Bears", 2: "Sprites", 3: "Giant Spiders", 4: "Basilisk", 5: "Stone Giant", 6: "Gorgons",
        7: "Behemoth", 8: "Colossus", 9: "Great Wyrm", 10: "Floating Island", 11: "Storm Giant",
        12: "Djinn", 13: "Sky Drake", 14: "Hell Hounds", 15: "Gargoyles", 16: "Doom Bat", 17: "Chaos Spawn",
        18: "Fire Giant", 19: "Chimeras", 20: "Efreet", 21: "Hydra", 22: "Great Drake", 23: "Unicorns",
        24: "Angel", 25: "Arch Angel", 27: "Skeletons", 28: "Ghouls", 29: "Night Stalker", 31: "Wraiths",
        32: "Shadow Demons", 33: "Death Knights", 34: "Demon Lord", 35: "Resurrection",
        36: "Guardian Spirit", 37: "Magic Spirit", 38: "Enchant Item", 39: "Create Artifact",
        40: "Incarnation", 41: "Summon Champion", 42: "Summon Hero", 43: "Cockatrices",
    }
    weights := make([]int, 50)
    transports := 0
    for _, stack := range self.Stacks {
        for _, unit := range stack.Units() {
            if unit.HasAbility(data.AbilityTransport) {
                transports += 1
            }
        }
    }
    for slot := 1; slot <= 34; slot++ {
        name, ok := names[slot]
        if !ok {
            continue
        }
        cost := chewCost(self, name)
        if cost == 0 {
            continue
        }
        weight := cost / 10
        if slot == 10 {
            weight = 0
            if transports < 10 && world.Turn / 60 > transports {
                weight = cost / 5
            }
        }
        if !(slot == 11 && quirkStormGiantLinear) {
            weight *= weight
        }
        weights[slot] = weight
    }

    heroes := chewHeroCount(self)
    if _, ok := chewKnows(self, "Resurrection"); ok && heroes < 6 && chewDeadHeroes(self) > 0 {
        weights[35] = 100
    }
    opportunity := ai.nodeOpportunity(self, services)
    if _, ok := chewKnows(self, "Guardian Spirit"); ok {
        if opportunity {
            weights[36] = 200
        }
    } else if _, ok := chewKnows(self, "Magic Spirit"); ok && opportunity {
        weights[37] = 100
    }
    // Enchant Item 20 and Create Artifact 30 (Artificer 30 and 40): what item the original's computer
    // wizard makes is not reconstructed, so Chewbot does not cast them
    if chewCastsItems {
        artificer := self.Wizard.RetortEnabled(data.RetortArtificer)
        if _, ok := chewKnows(self, "Enchant Item"); ok {
            weights[38] = 20
            if artificer {
                weights[38] = 30
            }
        }
        if _, ok := chewKnows(self, "Create Artifact"); ok {
            weights[39] = 30
            if artificer {
                weights[39] = 40
            }
        }
    }
    if heroes < 6 {
        chosen := false
        for _, stack := range self.Stacks {
            for _, unit := range stack.Units() {
                raw := unit.GetRawUnit()
                if quirkIncarnationSpearmen {
                    if raw.Name == units.BarbarianSpearmen.Name && raw.Race == units.BarbarianSpearmen.Race {
                        chosen = true
                    }
                } else if raw.Name == "The Chosen" {
                    chosen = true
                }
            }
        }
        wantIncarnation := chosen
        if !quirkIncarnationSpearmen {
            wantIncarnation = !chosen
        }
        if _, ok := chewKnows(self, "Incarnation"); ok && wantIncarnation {
            weights[40] = 500
        } else if _, ok := chewKnows(self, "Summon Champion"); ok {
            weights[41] = 400
        } else if _, ok := chewKnows(self, "Summon Hero"); ok {
            weights[42] = 200
        }
    }
    if cost := chewCost(self, "Cockatrices"); cost > 0 {
        weights[43] = cost / 5
    }

    if chewLifeSuppressed(world) {
        chewTwoThirds(weights, 23, 24, 25, 26, 35, 36, 40)
    }
    if chewNatureSuppressed(world) {
        chewTwoThirds(weights, append(chewSlotRange(1, 10), 43)...)
    }
    if chewDeathSuppressed(world) {
        chewBand(weights, chewSlotRange(27, 34)...)
    }
    if chewChaosSuppressed(world) {
        chewBand(weights, chewSlotRange(14, 22)...)
    }
    return chewPickSlot(weights, names)
}

// the heroes of the wizard that died (Player_Dead_Hero_Count)
func chewDeadHeroes(self *playerlib.Player) int {
    count := 0
    for _, hero := range self.Heroes {
        if hero != nil && hero.Status == herolib.StatusDead {
            count += 1
        }
    }
    return count
}

// AI_Select_Spell_Group_Unit_Enchantment: flat weights
func chewUnitSpell(self *playerlib.Player, world *chewSpellWorld) string {
    table := []struct {
        Slot int
        Name string
        Weight int
    }{
        {1, "Resist Elements", 5}, {2, "Stone Skin", 15}, {3, "Elemental Armor", 5}, {4, "Iron Skin", 30},
        {5, "Regeneration", 10}, {6, "Resist Magic", 5}, {9, "Magic Immunity", 20}, {10, "Flame Blade", 30},
        {11, "Eldritch Weapon", 15}, {12, "Immolation", 10}, {13, "Holy Weapon", 10}, {14, "Bless", 5},
        {15, "Holy Armor", 10}, {16, "Heroism", 10}, {17, "True Sight", 5}, {18, "Invulnerability", 20},
        {19, "Lionheart", 30}, {20, "Righteousness", 5}, {21, "Cloak of Fear", 10}, {23, "Wraith Form", 5},
        {24, "Water Walking", 30}, {25, "Path Finding", 5}, {27, "Flight", 25}, {28, "Wind Walking", 50},
        {29, "Endurance", 10}, {30, "Plane Shift", 20}, {31, "Planar Travel", 5}, {32, "Nature's Cures", 5},
        {33, "Enchant Road", 10}, {34, "Spell Lock", 10}, {35, "Invisiblity", 40}, {36, "Chaos Channels", 25},
        {37, "Lycanthropy", 25}, {38, "Black Channels", 25}, {39, "Guardian Wind", 10}, {40, "Giant Strength", 10},
    }
    weights := make([]int, 50)
    names := make(map[int]string)
    for _, entry := range table {
        names[entry.Slot] = entry.Name
        if _, ok := chewKnows(self, entry.Name); !ok {
            continue
        }
        switch entry.Name {
            case "Plane Shift":
                // the original looks at the first four players only
                sealed := false
                for index, player := range world.Players {
                    if index < 4 && player.HasEnchantment(data.EnchantmentPlanarSeal) {
                        sealed = true
                    }
                }
                if sealed {
                    continue
                }
            case "Chaos Channels":
                if self.HasEnchantment(data.EnchantmentDoomMastery) {
                    continue
                }
            case "Lycanthropy":
                if self.HasEnchantment(data.EnchantmentZombieMastery) {
                    continue
                }
        }
        weights[entry.Slot] = entry.Weight
    }
    if chewLifeSuppressed(world) {
        chewTwoThirds(weights, append(chewSlotRange(13, 20), 29, 30, 31)...)
    }
    if chewNatureSuppressed(world) {
        chewTwoThirds(weights, 1, 2, 3, 4, 5, 24, 25, 26, 32)
        weights[40] /= 2
    }
    if chewDeathSuppressed(world) {
        chewBand(weights, 21, 22, 23, 37, 38)
    }
    if chewChaosSuppressed(world) {
        chewBand(weights, 10, 11, 12, 36)
    }
    return chewPickSlot(weights, names)
}

// AI_Select_Spell_Group_City_Enchantment: a spell only when its target picker finds a target
func (ai *ChewbotAI) citySpell(self *playerlib.Player, services playerlib.AIServices, world *chewSpellWorld) string {
    turn := world.Turn
    table := []struct {
        Slot int
        Name string
        Weight int
    }{
        {1, "Wall of Stone", turn / 6}, {2, "Transmute", 200}, {3, "Change Terrain", 50},
        {5, "Move Fortress", turn / 15}, {6, "Earth Gate", turn / 15}, {7, "Gaia's Blessing", 300},
        {8, "Flying Fortress", 100}, {9, "Wall of Fire", turn / 20}, {10, "Heavenly Light", turn / 25},
        {11, "Stream of Life", turn / 10}, {12, "Inspirations", 300}, {13, "Prosperity", 300},
        {14, "Astral Gate", 0}, {15, "Dark Rituals", 100}, {16, "Cloud of Shadow", turn / 20},
        {17, "Spell Ward", 100}, {18, "Consecration", 100}, {19, "Wall of Darkness", turn / 20},
        {20, "Altar of Battle", 50},
    }
    weights := make([]int, 50)
    names := make(map[int]string)
    for _, entry := range table {
        names[entry.Slot] = entry.Name
        spell, ok := chewKnows(self, entry.Name)
        if !ok {
            continue
        }
        if _, found := ai.spellTarget(self, services, world, spell); !found {
            continue
        }
        weight := entry.Weight
        if entry.Name == "Astral Gate" && !quirkAstralGateNever {
            weight = turn / 15
        }
        weights[entry.Slot] = weight
    }
    if chewLifeSuppressed(world) {
        chewTwoThirds(weights, 10, 11, 12, 13, 14, 18, 20)
    }
    if chewNatureSuppressed(world) {
        chewTwoThirds(weights, chewSlotRange(1, 7)...)
    }
    if chewDeathSuppressed(world) {
        chewBand(weights, 15, 16, 19)
    }
    if chewChaosSuppressed(world) {
        chewBand(weights, 9)
    }
    return chewPickSlot(weights, names)
}

// AI_Has_Targetable_City: the original marks the cities a wizard sees through Awareness; without
// any it has no city to aim at
func chewHasTargetableCity(self *playerlib.Player, world *chewSpellWorld) bool {
    return self.HasEnchantment(data.EnchantmentAwareness) && len(world.Cities) > 0
}

// AI_Select_Spell_Group_Attack: weights by the cost of the spell
func chewAttackSpell(self *playerlib.Player, world *chewSpellWorld) string {
    table := []struct {
        Slot int
        Name string
        Divisor int
    }{
        {1, "Ice Storm", 10}, {2, "Earthquake", 10}, {3, "Spell Blast", 2}, {4, "Stasis", 5},
        {5, "Time Stop", 10}, {6, "Corruption", 5}, {7, "Fire Storm", 10}, {8, "Raise Volcano", 10},
        {10, "Chaos Rift", 3}, {11, "Call the Void", 2}, {12, "Famine", 5}, {13, "Warp Node", 3},
        {14, "Black Wind", 10}, {15, "Drain Power", 5}, {16, "Evil Presence", 5}, {17, "Subversion", 10},
        {18, "Pestilence", 2}, {19, "Cursed Lands", 5}, {20, "Cruel Unminding", 5},
    }
    weights := make([]int, 50)
    names := make(map[int]string)
    for _, entry := range table {
        names[entry.Slot] = entry.Name
        cost := chewCost(self, entry.Name)
        if cost == 0 {
            continue
        }
        if entry.Name == "Time Stop" && self.Mana < 3000 {
            continue
        }
        if entry.Name == "Raise Volcano" && self.HasEnchantment(data.EnchantmentArmageddon) {
            continue
        }
        weights[entry.Slot] = cost / entry.Divisor
    }
    if !chewHasTargetableCity(self, world) {
        for _, slot := range []int{16, 19, 18, 12, 2, 10, 6, 8, 11, 13} {
            weights[slot] = 0
        }
    }
    if chewNatureSuppressed(world) {
        chewTwoThirds(weights, 1, 2)
    }
    if chewDeathSuppressed(world) {
        chewBand(weights, chewSlotRange(12, 20)...)
    }
    if chewChaosSuppressed(world) {
        chewBand(weights, chewSlotRange(7, 11)...)
    }
    return chewPickSlot(weights, names)
}

// AI_Select_Spell_Group_Global
func chewGlobalSpell(self *playerlib.Player, world *chewSpellWorld) string {
    names := map[int]string{
        1: "Herb Mastery", 2: "Wind Mastery", 3: "Aura of Majesty", 4: "Great Unsummoning",
        5: "Suppress Magic", 6: "Spell Binding", 7: "Great Wasting", 8: "Chaos Surge", 9: "Doom Mastery",
        10: "Meteor Storm", 11: "Armageddon", 12: "Just Cause", 13: "Planar Seal", 14: "Holy Arms",
        15: "Charm of Life", 16: "Crusade", 17: "Eternal Night", 18: "Death Wish", 19: "Zombie Mastery",
        20: "Awareness",
    }
    instants := map[string]bool{"Great Unsummoning": true, "Spell Binding": true, "Death Wish": true}
    weights := make([]int, 50)
    for slot := 1; slot <= 20; slot++ {
        name := names[slot]
        cost := chewCost(self, name)
        if cost == 0 {
            continue
        }
        if !instants[name] && self.HasEnchantment(data.GetEnchantmentByName(name)) {
            continue
        }
        weight := cost / 10
        if name == "Planar Seal" {
            if !chewMoreMyrrorCities(self, world) {
                continue
            }
            weight = cost / 20
        }
        weights[slot] = weight
    }
    if chewLifeSuppressed(world) {
        chewTwoThirds(weights, chewSlotRange(12, 16)...)
    }
    if chewNatureSuppressed(world) {
        chewTwoThirds(weights, 1)
    }
    if chewDeathSuppressed(world) {
        chewBand(weights, 17, 18, 19)
    }
    if chewChaosSuppressed(world) {
        chewBand(weights, chewSlotRange(7, 11)...)
    }
    return chewPickSlot(weights, names)
}

// AI_Select_Spell_Group_Suppression: no test for one it has already
func chewSuppressionSpell(self *playerlib.Player, world *chewSpellWorld) string {
    names := map[int]string{1: "Nature's Wrath", 2: "Life Force", 3: "Tranquility", 4: "Evil Omens"}
    weights := make([]int, 50)
    for slot, name := range names {
        weights[slot] = chewCost(self, name) / 10
    }
    if chewLifeSuppressed(world) {
        chewTwoThirds(weights, 2, 3)
    }
    if chewNatureSuppressed(world) {
        chewTwoThirds(weights, 1)
    }
    if chewDeathSuppressed(world) {
        chewBand(weights, 4)
    }
    return chewPickSlot(weights, names)
}

// the name of a kind of spell AI_Select_Spell_Group picks, for the journal
func chewSpellKindName(kind int) string {
    switch kind {
        case chewSpellSummon: return "summon"
        case chewSpellUnit: return "unit enchantment"
        case chewSpellCity: return "city"
        case chewSpellDisenchant: return "disenchant"
        case chewSpellDisjunction: return "disjunction"
        case chewSpellSummoningCircle: return "summoning circle"
        case chewSpellAttack: return "attack"
        case chewSpellSuppression: return "suppression"
        case chewSpellGlobal: return "global"
        case chewSpellMastery: return "mastery"
    }
    return "none"
}
