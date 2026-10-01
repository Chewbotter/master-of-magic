package ai

// The targets of Chewbot's spells, picked when a spell is cast (the original's AITP_* pickers;
// ReMoM MoM/src/AISPELL.c, and the routing of MoM/src/OverSpel.c Cast_Spell_Overland). The rules in
// words: docs/mod/ai-spells.md. The code is ours.
//
// The reconstruction's wiring is kept where nothing says it is not the original's: Black Wind and
// Stasis ask pickers that never answer (quirkStackSpellsNoTarget), the value of an enemy stack is
// its last unit's (chewbotspellworld.go), a node is only a sorcery node (quirkOnlySorceryNodes),
// stacks of treaty partners are targets too. Where the reconstruction's own notes, checked against
// the program, say otherwise, they are followed: the curses of cities use the enemy city scan and
// it wants cities seen. Not portable, the meaning is ported: Fire Storm and Ice Storm take a unit
// number for a square (the fork has no unit numbers), Disenchant's last step reads outside its list.

import (
    "image"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/maplib"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    "github.com/kazzmir/master-of-magic/game/magic/terrain"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    buildinglib "github.com/kazzmir/master-of-magic/game/magic/building"
    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

const (
    // AITP_Flying_Fortress: only a fortress that flies already (the test is the wrong way round)
    quirkFlyingFortressInverted = true
    // AITP_City_Enchantment: Evil Presence only on cities of wizards with death books, where it does
    // nothing
    quirkEvilPresenceDeath = true
    // AITP_Plane_Shift: only stacks with settlers, only from Arcanus
    quirkPlaneShiftSettlers = true
    // AITP_Enchant_Road looks at the city's own square only
    quirkEnchantRoadCity = true
    // AITP_Transmute: the specials are tested as bits (Iron, Coal, Silver, Gems, Mithril,
    // Adamantium, Crysx), and all 25 squares around a city
    quirkTransmuteBits = true
    // Black Wind and Stasis: the AI's casting asks pickers that never answer, so the spell is lost
    quirkStackSpellsNoTarget = true
    // Square_Is_Node compares with the sorcery node three times: Raise Volcano takes nature nodes
    quirkOnlySorceryNodes = true
)

// ChooseSpellTarget: the target of a spell of the world map, when it is cast (game/aicast.go)
func (ai *ChewbotAI) ChooseSpellTarget(self *playerlib.Player, spell spellbook.Spell) (playerlib.AISpellTarget, bool) {
    if !chewbotSpellsActive() || ai.services == nil {
        return playerlib.AISpellTarget{}, false
    }
    world := ai.spellWorld(self, ai.services)
    target, ok := ai.spellTarget(self, ai.services, world, spell)
    if ChewbotSpellLog {
        unitName := ""
        if target.Unit != nil {
            unitName = target.Unit.GetName()
        }
        chewSpellLog(self, "target of %v: %v,%v %v unit %v, %v", spell.Name, target.X, target.Y, target.Plane, unitName, ok)
    }
    return target, ok
}

func chewCityTarget(city *citylib.City) (playerlib.AISpellTarget, bool) {
    if city == nil {
        return playerlib.AISpellTarget{}, false
    }
    return playerlib.AISpellTarget{X: city.X, Y: city.Y, Plane: city.Plane}, true
}

// the enchantment a unit spell gives, and the original's flags for the unit it picks
// (AITP_Unit_Enchantment): not summoned, only summoned, not the first 41 kinds (heroes, ships,
// catapults, barbarian infantry), movement (fliers and swimmers count nothing, Floating Island may)
type chewUnitSpellRule struct {
    Enchantment data.UnitEnchantment
    NotFantastic bool
    OnlyFantastic bool
    NotFirstKinds bool
    Movement bool
}

var chewUnitSpellRules = map[string]chewUnitSpellRule{
    "Resist Elements": {Enchantment: data.UnitEnchantmentResistElements},
    "Giant Strength": {Enchantment: data.UnitEnchantmentGiantStrength, NotFantastic: true},
    "Stone Skin": {Enchantment: data.UnitEnchantmentStoneSkin},
    "Water Walking": {Enchantment: data.UnitEnchantmentWaterWalking, Movement: true},
    "Path Finding": {Enchantment: data.UnitEnchantmentPathFinding},
    "Elemental Armor": {Enchantment: data.UnitEnchantmentElementalArmor},
    "Iron Skin": {Enchantment: data.UnitEnchantmentIronSkin},
    "Regeneration": {Enchantment: data.UnitEnchantmentRegeneration},
    "Resist Magic": {Enchantment: data.UnitEnchantmentResistMagic},
    "Guardian Wind": {Enchantment: data.UnitEnchantmentGuardianWind},
    "Spell Lock": {Enchantment: data.UnitEnchantmentSpellLock, OnlyFantastic: true},
    "Flight": {Enchantment: data.UnitEnchantmentFlight, Movement: true},
    "Invisiblity": {Enchantment: data.UnitEnchantmentInvisibility},
    "Wind Walking": {Enchantment: data.UnitEnchantmentWindWalking},
    "Magic Immunity": {Enchantment: data.UnitEnchantmentMagicImmunity},
    "Eldritch Weapon": {Enchantment: data.UnitEnchantmentEldritchWeapon, NotFantastic: true},
    "Chaos Channels": {NotFantastic: true},
    "Flame Blade": {Enchantment: data.UnitEnchantmentFlameBlade, NotFantastic: true},
    "Immolation": {Enchantment: data.UnitEnchantmentImmolation},
    "Bless": {Enchantment: data.UnitEnchantmentBless},
    "Endurance": {Enchantment: data.UnitEnchantmentEndurance},
    "Holy Weapon": {Enchantment: data.UnitEnchantmentHolyWeapon, NotFantastic: true},
    "Holy Armor": {Enchantment: data.UnitEnchantmentHolyArmor},
    "Heroism": {Enchantment: data.UnitEnchantmentHeroism},
    "True Sight": {Enchantment: data.UnitEnchantmentTrueSight},
    "Planar Travel": {Enchantment: data.UnitEnchantmentPlanarTravel, Movement: true},
    "Lionheart": {Enchantment: data.UnitEnchantmentLionHeart},
    "Invulnerability": {Enchantment: data.UnitEnchantmentInvulnerability},
    "Righteousness": {Enchantment: data.UnitEnchantmentRighteousness},
    "Cloak of Fear": {Enchantment: data.UnitEnchantmentCloakOfFear},
    "Lycanthropy": {NotFantastic: true, NotFirstKinds: true},
    "Black Channels": {Enchantment: data.UnitEnchantmentBlackChannels},
    "Wraith Form": {Enchantment: data.UnitEnchantmentWraithForm},
}

// the spell-like abilities of a unit type (the original's Attribs_2): such units are never picked
var chewInnateSpells = []data.AbilityType{
    data.AbilityHealingSpell, data.AbilityFireballSpell, data.AbilityDoomBoltSpell, data.AbilityImmolation,
    data.AbilityWebSpell, data.AbilityCauseFear, data.AbilityResistanceToAll, data.AbilityHolyBonus,
}

// the first 41 kinds of unit of the original: heroes, ships, the catapult, barbarian infantry
func chewFirstKinds(unit units.StackUnit) bool {
    if unit.GetRace() == data.RaceHero {
        return true
    }
    raw := unit.GetRawUnit()
    for _, kind := range []units.Unit{units.Trireme, units.Galley, units.Catapult, units.Warship, units.BarbarianSpearmen, units.BarbarianSwordsmen} {
        if raw.Name == kind.Name && raw.Race == kind.Race {
            return true
        }
    }
    return false
}

// Select_Unit_For_Enchantment: the unit of the strongest stack of the wizard outside cities (else
// the stack in its fortress) with the highest upkeep that fits the rule and does not have the
// enchantment
func chewUnitForEnchantment(world *chewSpellWorld, rule chewUnitSpellRule) (units.StackUnit, bool) {
    var chosen *chewSpellStack
    best := 0
    for index := range world.OwnStacks {
        stack := &world.OwnStacks[index]
        if world.cityAt(stack.X, stack.Y, stack.Plane) != nil {
            continue
        }
        if stack.Value > best {
            best = stack.Value
            chosen = stack
        }
    }
    if chosen == nil {
        fortress := world.Self.FindFortressCity()
        if fortress == nil {
            return nil, false
        }
        for index := range world.OwnStacks {
            stack := &world.OwnStacks[index]
            if stack.X == fortress.X && stack.Y == fortress.Y && stack.Plane == fortress.Plane {
                chosen = stack
            }
        }
    }
    if chosen == nil {
        return nil, false
    }

    var pick units.StackUnit
    bestScore := 0
    for _, unit := range chosen.Units {
        fantastic := unit.GetRace() == data.RaceFantastic
        raw := unit.GetRawUnit()
        if rule.NotFantastic && fantastic || rule.OnlyFantastic && !fantastic {
            continue
        }
        innate := false
        for _, ability := range chewInnateSpells {
            if raw.HasAbility(ability) {
                innate = true
            }
        }
        if innate || unit.HasAbility(data.AbilityCreateOutpost) || unit.HasAbility(data.AbilityMeld) {
            continue
        }
        if rule.NotFirstKinds && chewFirstKinds(unit) {
            continue
        }
        if rule.Enchantment != data.UnitEnchantmentNone && unit.HasEnchantment(rule.Enchantment) {
            continue
        }
        if !rule.Movement && raw.Name == units.FloatingIsland.Name {
            continue
        }
        score := raw.UpkeepGold
        if fantastic {
            score = raw.UpkeepMana
        }
        if rule.Movement && (raw.Flying || raw.Swimming) {
            score = 0
        }
        if score > bestScore {
            bestScore = score
            pick = unit
        }
    }
    return pick, pick != nil
}

// the target of a spell in the world as it is now; false: none (the spell is lost)
func (ai *ChewbotAI) spellTarget(self *playerlib.Player, services playerlib.AIServices, world *chewSpellWorld, spell spellbook.Spell) (playerlib.AISpellTarget, bool) {
    if rule, ok := chewUnitSpellRules[spell.Name]; ok {
        unit, found := chewUnitForEnchantment(world, rule)
        if !found {
            return playerlib.AISpellTarget{}, false
        }
        return playerlib.AISpellTarget{X: unit.GetX(), Y: unit.GetY(), Plane: unit.GetPlane(), Unit: unit}, true
    }

    fortress := self.FindFortressCity()
    lacks := func(enchantment data.CityEnchantment) func(*citylib.City) bool {
        return func(city *citylib.City) bool {
            return !city.HasEnchantment(enchantment)
        }
    }

    switch spell.Name {
        // the cities of the wizard
        case "Wall of Stone":
            return chewCityTarget(world.bestOwnCity(func(city *citylib.City) bool {
                return !city.Buildings.Contains(buildinglib.BuildingCityWalls)
            }))
        case "Move Fortress":
            if fortress == nil {
                return playerlib.AISpellTarget{}, false
            }
            var best *citylib.City
            threshold := world.Garrison[fortress]
            for _, city := range world.ownCities() {
                if city != fortress && world.Garrison[city] > threshold {
                    threshold = world.Garrison[city]
                    best = city
                }
            }
            return chewCityTarget(best)
        case "Flying Fortress":
            if fortress == nil {
                return playerlib.AISpellTarget{}, false
            }
            flying := fortress.HasEnchantment(data.CityEnchantmentFlyingFortress)
            if flying == quirkFlyingFortressInverted {
                return chewCityTarget(fortress)
            }
            return playerlib.AISpellTarget{}, false
        case "Earth Gate": return chewCityTarget(world.bestOwnCity(lacks(data.CityEnchantmentEarthGate)))
        case "Gaia's Blessing": return chewCityTarget(world.bestOwnCity(lacks(data.CityEnchantmentGaiasBlessing)))
        case "Wall of Fire": return chewCityTarget(world.bestOwnCity(lacks(data.CityEnchantmentWallOfFire)))
        case "Wall of Darkness": return chewCityTarget(world.bestOwnCity(lacks(data.CityEnchantmentWallOfDarkness)))
        case "Heavenly Light": return chewCityTarget(world.bestOwnCity(lacks(data.CityEnchantmentHeavenlyLight)))
        case "Altar of Battle": return chewCityTarget(world.bestOwnCity(lacks(data.CityEnchantmentAltarOfBattle)))
        case "Stream of Life": return chewCityTarget(world.bestOwnCity(lacks(data.CityEnchantmentStreamOfLife)))
        case "Inspirations": return chewCityTarget(world.bestOwnCity(lacks(data.CityEnchantmentInspirations)))
        case "Prosperity": return chewCityTarget(world.bestOwnCity(lacks(data.CityEnchantmentProsperity)))
        case "Astral Gate": return chewCityTarget(world.bestOwnCity(lacks(data.CityEnchantmentAstralGate)))
        case "Cloud of Shadow": return chewCityTarget(world.bestOwnCity(lacks(data.CityEnchantmentCloudOfShadow)))
        case "Dark Rituals":
            return chewCityTarget(world.bestOwnCity(func(city *citylib.City) bool {
                return !city.HasEnchantment(data.CityEnchantmentDarkRituals) && city.Buildings.Contains(buildinglib.BuildingTemple)
            }))
        case "Consecration":
            return chewConsecrationTarget(world)
        case "Spell Ward":
            return chewSpellWardTarget(world)
        case "Transmute", "Change Terrain":
            return chewTerrainTarget(world, services, spell.Name)

        // the enemy's cities
        case "Earthquake", "Call the Void", "Chaos Rift", "Evil Presence", "Cursed Lands", "Pestilence", "Famine":
            return chewCurseTarget(world, spell.Name)
        case "Corruption", "Raise Volcano":
            return chewAttackTerrainTarget(world, services, spell.Name)

        // stacks
        case "Black Wind", "Stasis":
            if quirkStackSpellsNoTarget {
                return playerlib.AISpellTarget{}, false
            }
            return chewAttackStackTarget(world, services)
        case "Ice Storm", "Fire Storm":
            return chewAttackStackTarget(world, services)
        case "Floating Island":
            return chewFloatingIslandTarget(world, services)
        case "Enchant Road":
            return chewEnchantRoadTarget(world, services)
        case "Plane Shift":
            return chewPlaneShiftTarget(world, services)
        case "Nature's Cures":
            var best units.StackUnit
            most := 0
            for _, stack := range self.Stacks {
                for _, unit := range stack.Units() {
                    if damage := unit.GetDamage(); damage > most {
                        most = damage
                        best = unit
                    }
                }
            }
            if best == nil {
                return playerlib.AISpellTarget{}, false
            }
            return playerlib.AISpellTarget{X: best.GetX(), Y: best.GetY(), Plane: best.GetPlane()}, true
        case "Disenchant Area", "Disenchant True":
            return chewDisenchantTarget(world, services)

        // wizards, nodes, globals
        case "Spell Blast", "Cruel Unminding", "Drain Power", "Subversion":
            return chewAttackWizardTarget(world, spell.Name)
        case "Warp Node":
            return chewWarpNodeTarget(world, services)
        case "Disjunction", "Disjunction True", "Spell Binding":
            return chewDisjunctionTarget(world, spell.Name)
    }
    return playerlib.AISpellTarget{}, false
}

// AITP_Consecration: when the human has death or chaos books, else one time in 10: the best own
// city without Consecration
func chewConsecrationTarget(world *chewSpellWorld) (playerlib.AISpellTarget, bool) {
    human := world.Human
    if human == nil {
        return playerlib.AISpellTarget{}, false
    }
    if !chewHasRealm(human.Wizard, data.DeathMagic) && !chewHasRealm(human.Wizard, data.ChaosMagic) && chewRoll(10) != 0 {
        return playerlib.AISpellTarget{}, false
    }
    return chewCityTarget(world.bestOwnCity(func(city *citylib.City) bool {
        return !city.HasEnchantment(data.CityEnchantmentConsecration)
    }))
}

// the ward against a realm
var chewWards = map[data.MagicType]data.CityEnchantment{
    data.NatureMagic: data.CityEnchantmentNatureWard,
    data.SorceryMagic: data.CityEnchantmentSorceryWard,
    data.ChaosMagic: data.CityEnchantmentChaosWard,
    data.LifeMagic: data.CityEnchantmentLifeWard,
    data.DeathMagic: data.CityEnchantmentDeathWard,
}

// AITP_Spell_Ward: the best own city, warded against the human's first realm, else its second,
// else one time in 10 against a realm by chance (which the city may have already)
func chewSpellWardTarget(world *chewSpellWorld) (playerlib.AISpellTarget, bool) {
    human := world.Human
    if human == nil {
        return playerlib.AISpellTarget{}, false
    }
    primary, secondary := chewRealms(human.Wizard)
    var best *citylib.City
    var ward data.CityEnchantment
    bestValue := 0
    for _, city := range world.ownCities() {
        if world.OwnValue[city] <= bestValue {
            continue
        }
        chosen, ok := data.CityEnchantment(0), false
        if !city.HasEnchantment(chewWards[primary]) {
            chosen, ok = chewWards[primary], true
        } else if !city.HasEnchantment(chewWards[secondary]) {
            chosen, ok = chewWards[secondary], true
        } else if chewRoll(10) == 0 {
            chosen, ok = chewWards[chewRealmOrder[chewRoll(5)]], true
        }
        if ok {
            best = city
            ward = chosen
            bestValue = world.OwnValue[city]
        }
    }
    if best == nil || best.HasEnchantment(ward) {
        return playerlib.AISpellTarget{}, false
    }
    return playerlib.AISpellTarget{X: best.X, Y: best.Y, Plane: best.Plane, CityEnchantment: ward}, true
}

// the original's numbers of the specials, for Transmute's test of bits
var chewSpecialNumber = map[data.BonusType]int{
    data.BonusIronOre: 1, data.BonusCoal: 2, data.BonusSilverOre: 3, data.BonusGoldOre: 4, data.BonusGem: 5,
    data.BonusMithrilOre: 6, data.BonusAdamantiumOre: 7, data.BonusQuorkCrystal: 8, data.BonusCrysxCrystal: 9,
}

// AITP_Transmute, AITP_Change_Terrain: the first fitting square around the best own city that has one
// (25 squares with the corners)
func chewTerrainTarget(world *chewSpellWorld, services playerlib.AIServices, name string) (playerlib.AISpellTarget, bool) {
    bestValue := 0
    var target playerlib.AISpellTarget
    found := false
    for _, city := range world.ownCities() {
        if world.OwnValue[city] <= bestValue {
            continue
        }
        mapObject := services.GetMap(city.Plane)
    search:
        for dy := -2; dy <= 2; dy++ {
            y := city.Y + dy
            if y < 0 || y >= mapObject.Height() {
                continue
            }
            for dx := -2; dx <= 2; dx++ {
                x := mapObject.WrapX(city.X + dx)
                fits := false
                if name == "Transmute" {
                    number := chewSpecialNumber[mapObject.GetBonusTile(x, y)]
                    if quirkTransmuteBits {
                        fits = number & 3 != 0
                    } else {
                        fits = number == 1 || number == 2 || number == 3
                    }
                } else {
                    kind := mapObject.GetTile(x, y).Tile.TerrainType()
                    fits = kind == terrain.Desert || kind == terrain.Swamp
                }
                if fits {
                    bestValue = world.OwnValue[city]
                    target = playerlib.AISpellTarget{X: x, Y: y, Plane: city.Plane}
                    found = true
                    break search
                }
            }
        }
    }
    return target, found
}

// a city of the enemy the wizard sees: the original marks those through Awareness
func chewSeesCity(world *chewSpellWorld, city *citylib.City) bool {
    return world.Self.HasEnchantment(data.EnchantmentAwareness)
}

// AITP_City_Enchantment for the enemy's cities: the most valued city of a hostile wizard (not
// neutral) seen, without Consecration and the ward of the spell's realm, not cursed with it already
func chewCurseTarget(world *chewSpellWorld, name string) (playerlib.AISpellTarget, bool) {
    ward := data.CityEnchantmentChaosWard
    curse := data.CityEnchantment(0)
    switch name {
        case "Evil Presence": ward, curse = data.CityEnchantmentDeathWard, data.CityEnchantmentEvilPresence
        case "Cursed Lands": ward, curse = data.CityEnchantmentDeathWard, data.CityEnchantmentCursedLands
        case "Pestilence": ward, curse = data.CityEnchantmentDeathWard, data.CityEnchantmentPestilence
        case "Famine": ward, curse = data.CityEnchantmentDeathWard, data.CityEnchantmentFamine
        case "Chaos Rift": curse = data.CityEnchantmentChaosRift
    }
    var best *citylib.City
    bestValue := 0
    for _, city := range world.Cities {
        owner := world.Owners[city]
        if world.EnemyValue[city] <= bestValue || owner.IsNeutral() {
            continue
        }
        if city.HasEnchantment(data.CityEnchantmentConsecration) || city.HasEnchantment(ward) {
            continue
        }
        if name == "Evil Presence" {
            deathBooks := chewBooks(owner, data.DeathMagic) > 0
            if deathBooks != quirkEvilPresenceDeath {
                continue
            }
        }
        if !chewSeesCity(world, city) {
            continue
        }
        if curse != 0 && city.HasEnchantment(curse) {
            continue
        }
        best = city
        bestValue = world.EnemyValue[city]
    }
    return chewCityTarget(best)
}

func chewBooks(player *playerlib.Player, magic data.MagicType) int {
    count := 0
    for _, book := range player.Wizard.Books {
        if book.Magic == magic {
            count += book.Count
        }
    }
    return count
}

// AITP_Attack_Terrain: 25 tries by chance around the most valued enemy city seen for a square of
// land that is no volcano and not corrupted (Raise Volcano: no mountain, hills, river or node either)
func chewAttackTerrainTarget(world *chewSpellWorld, services playerlib.AIServices, name string) (playerlib.AISpellTarget, bool) {
    bestValue := 0
    var target playerlib.AISpellTarget
    found := false
    for _, city := range world.Cities {
        owner := world.Owners[city]
        if owner == world.Self || owner.IsNeutral() || world.EnemyValue[city] <= bestValue || !chewSeesCity(world, city) {
            continue
        }
        mapObject := services.GetMap(city.Plane)
        for range 25 {
            dx := chewRoll(5) - 2
            dy := chewRoll(5) - 2
            if (dx == 2 || dx == -2) && (dy == 2 || dy == -2) {
                continue
            }
            x, y := city.X + dx, city.Y + dy
            if x <= 0 || x >= mapObject.Width() || y <= 0 || y >= mapObject.Height() {
                continue
            }
            tile := mapObject.GetTile(x, y).Tile
            kind := tile.TerrainType()
            if !tile.IsLand() || kind == terrain.Volcano || kind == terrain.ChaosNode || mapObject.HasCorruption(x, y) {
                continue
            }
            if name == "Raise Volcano" {
                node := tile.IsMagic()
                if quirkOnlySorceryNodes {
                    node = kind == terrain.SorceryNode
                }
                if kind == terrain.Mountain || kind == terrain.Hill || kind == terrain.River || node {
                    continue
                }
            }
            bestValue = world.EnemyValue[city]
            target = playerlib.AISpellTarget{X: x, Y: y, Plane: city.Plane}
            found = true
            break
        }
    }
    return target, found
}

// AITP_Attack_Stack: the most valued enemy stack within sight of the wizard's cities (2, walls 3,
// Oracle 5) or units (2) on its plane
func chewAttackStackTarget(world *chewSpellWorld, services playerlib.AIServices) (playerlib.AISpellTarget, bool) {
    bestValue := 0
    var target playerlib.AISpellTarget
    found := false
    for _, stack := range world.EnemyStacks {
        // every other player's stack, a treaty partner's too
        if stack.Value <= bestValue {
            continue
        }
        width := services.GetMap(stack.Plane).Width()
        seen := false
        for _, city := range world.ownCities() {
            if city.Plane != stack.Plane {
                continue
            }
            radius := 2
            if city.Buildings.Contains(buildinglib.BuildingCityWalls) {
                radius = 3
            }
            if city.Buildings.Contains(buildinglib.BuildingOracle) {
                radius = 5
            }
            if chewSpellDistance(width, stack.X, stack.Y, city.X, city.Y) <= radius {
                seen = true
                break
            }
        }
        if !seen {
            for _, own := range world.OwnStacks {
                if own.Plane == stack.Plane && chewSpellDistance(width, stack.X, stack.Y, own.X, own.Y) <= 2 {
                    seen = true
                    break
                }
            }
        }
        if seen {
            bestValue = stack.Value
            target = playerlib.AISpellTarget{X: stack.X, Y: stack.Y, Plane: stack.Plane}
            found = true
        }
    }
    return target, found
}

// AITP_Floating_Island: a city of the wizard by chance (200 tries), then the first square of open
// water on the lines east, south, west and north of it, 1 to 9 squares out (the city's own square
// first)
func chewFloatingIslandTarget(world *chewSpellWorld, services playerlib.AIServices) (playerlib.AISpellTarget, bool) {
    if len(world.Cities) == 0 {
        return playerlib.AISpellTarget{}, false
    }
    var city *citylib.City
    for range 200 {
        candidate := world.Cities[chewRoll(len(world.Cities))]
        if world.Owners[candidate] == world.Self {
            city = candidate
            break
        }
    }
    if city == nil {
        return playerlib.AISpellTarget{}, false
    }
    mapObject := services.GetMap(city.Plane)
    legal := func(x int, y int) bool {
        if y < 0 || y >= mapObject.Height() {
            return false
        }
        kind := mapObject.GetTile(mapObject.WrapX(x), y).Tile.TerrainType()
        return kind == terrain.Ocean || kind == terrain.Shore
    }
    probe := []image.Point{image.Pt(-0, 0), image.Pt(0, -0)}
    for distance := 1; distance <= 9; distance++ {
        probe = append(probe, image.Pt(distance, 0), image.Pt(0, distance), image.Pt(-distance, 0), image.Pt(0, -distance))
    }
    for _, offset := range probe {
        x, y := city.X + offset.X, city.Y + offset.Y
        if legal(x, y) {
            return playerlib.AISpellTarget{X: mapObject.WrapX(x), Y: y, Plane: city.Plane}, true
        }
    }
    return playerlib.AISpellTarget{}, false
}

// AITP_Enchant_Road: the own city with the strongest garrison whose square has a road that is not
// enchanted
func chewEnchantRoadTarget(world *chewSpellWorld, services playerlib.AIServices) (playerlib.AISpellTarget, bool) {
    best := 0
    var target *citylib.City
    for _, city := range world.ownCities() {
        if world.Garrison[city] <= best {
            continue
        }
        mapObject := services.GetMap(city.Plane)
        extra, ok := mapObject.ExtraMap[image.Pt(city.X, city.Y)][maplib.ExtraKindRoad]
        if !ok {
            continue
        }
        road, isRoad := extra.(*maplib.ExtraRoad)
        if isRoad && !road.Enchanted {
            best = world.Garrison[city]
            target = city
        }
    }
    return chewCityTarget(target)
}

// AITP_Plane_Shift: the most valued stack of the wizard with settlers on Arcanus, not on a city
// square, whose square on Myrror it can stand on
func chewPlaneShiftTarget(world *chewSpellWorld, services playerlib.AIServices) (playerlib.AISpellTarget, bool) {
    best := 0
    var target playerlib.AISpellTarget
    found := false
    myrror := services.GetMap(data.PlaneMyrror)
    for _, stack := range world.OwnStacks {
        settlers, transport := false, false
        for _, unit := range stack.Units {
            if unit.HasAbility(data.AbilityCreateOutpost) {
                settlers = true
            }
            if unit.HasAbility(data.AbilityTransport) {
                transport = true
            }
        }
        value := stack.Value
        if quirkPlaneShiftSettlers && !settlers {
            value = 0
        }
        if world.cityAt(stack.X, stack.Y, data.PlaneArcanus) != nil || world.cityAt(stack.X, stack.Y, data.PlaneMyrror) != nil {
            value = 0
        }
        if stack.Plane != data.PlaneArcanus || value <= best {
            continue
        }
        water := myrror.IsWater(stack.X, stack.Y)
        if water == transport {
            best = value
            target = playerlib.AISpellTarget{X: stack.X, Y: stack.Y, Plane: stack.Plane}
            found = true
        }
    }
    return target, found
}

// AITP_Disenchant: a Flying Fortress of the human (one time in 2 each), else the strongest warped
// node of the wizard, else its most valued cursed city, else the costliest enchanted unit of the
// human on the landmass of its fortress
func chewDisenchantTarget(world *chewSpellWorld, services playerlib.AIServices) (playerlib.AISpellTarget, bool) {
    self := world.Self
    human := world.Human
    if human != nil {
        for _, city := range world.Cities {
            if world.Owners[city] == human && city.HasEnchantment(data.CityEnchantmentFlyingFortress) && chewRoll(2) == 0 {
                return chewCityTarget(city)
            }
        }
    }
    bestPower := 0
    var target playerlib.AISpellTarget
    found := false
    for _, plane := range []data.Plane{data.PlaneArcanus, data.PlaneMyrror} {
        mapObject := services.GetMap(plane)
        for _, point := range mapObject.GetMagicNodeLocations() {
            node := mapObject.GetMagicNode(point.X, point.Y)
            if node != nil && node.Warped && chewNodeOwner(node, self) && len(node.Zone) > bestPower {
                bestPower = len(node.Zone)
                target = playerlib.AISpellTarget{X: point.X, Y: point.Y, Plane: plane}
                found = true
            }
        }
    }
    if found {
        return target, true
    }
    cursed := world.bestOwnCity(func(city *citylib.City) bool {
        return city.HasEnchantment(data.CityEnchantmentCursedLands) || city.HasEnchantment(data.CityEnchantmentFamine) ||
            city.HasEnchantment(data.CityEnchantmentChaosRift) || city.HasEnchantment(data.CityEnchantmentEvilPresence)
    })
    if cursed != nil {
        return chewCityTarget(cursed)
    }
    fortress := self.FindFortressCity()
    if human == nil || fortress == nil {
        return playerlib.AISpellTarget{}, false
    }
    land := services.GetMap(fortress.Plane).Map.FindContinent(fortress.X, fortress.Y)
    bestCost := 0
    for _, stack := range human.Stacks {
        if stack.Plane() != fortress.Plane || !land.Contains(image.Pt(stack.X(), stack.Y())) {
            continue
        }
        for _, unit := range stack.Units() {
            if len(unit.GetEnchantments()) > 0 && unit.GetRawUnit().ProductionCost > bestCost {
                bestCost = unit.GetRawUnit().ProductionCost
                target = playerlib.AISpellTarget{X: stack.X(), Y: stack.Y(), Plane: stack.Plane()}
                found = true
            }
        }
    }
    return target, found
}

// AITP_Attack_Wizard: one time in 2 the first hostile wizard, else one of them by chance; it must be
// casting, and what it has put in the spell no more than the caster's mana less 50
func chewAttackWizardTarget(world *chewSpellWorld, name string) (playerlib.AISpellTarget, bool) {
    if len(world.Hostile) == 0 {
        return playerlib.AISpellTarget{}, false
    }
    var target *playerlib.Player
    if chewRoll(2) == 0 {
        if quirkAttackWizardFirst || world.Human == nil {
            target = world.Hostile[0]
        } else {
            target = world.Human
        }
    } else {
        target = world.Hostile[chewRoll(len(world.Hostile))]
    }
    if !target.CastingSpell.Valid() || target.CastingSpellProgress > world.Self.Mana - 50 {
        return playerlib.AISpellTarget{}, false
    }
    return playerlib.AISpellTarget{Player: target}, true
}

// AITP_Node: one time in 2 a node of the human, else of the player whose place in the game is a
// roll of the number of hostile wizards; its strongest node that is warped already (the original's
// test, meant: not warped)
func chewWarpNodeTarget(world *chewSpellWorld, services playerlib.AIServices) (playerlib.AISpellTarget, bool) {
    if len(world.Hostile) == 0 {
        return playerlib.AISpellTarget{}, false
    }
    var owner *playerlib.Player
    if chewRoll(2) == 0 {
        owner = world.Human
    } else {
        index := chewRoll(len(world.Hostile)) + 1
        order := chewGameOrder(world)
        if index < len(order) {
            owner = order[index]
        }
    }
    if owner == nil {
        return playerlib.AISpellTarget{}, false
    }
    bestPower := 0
    var target playerlib.AISpellTarget
    found := false
    for _, plane := range []data.Plane{data.PlaneArcanus, data.PlaneMyrror} {
        mapObject := services.GetMap(plane)
        for _, point := range mapObject.GetMagicNodeLocations() {
            node := mapObject.GetMagicNode(point.X, point.Y)
            if node == nil || !chewNodeOwner(node, owner) || len(node.Zone) <= bestPower {
                continue
            }
            if node.Warped != quirkWarpNodeWarped {
                continue
            }
            bestPower = len(node.Zone)
            target = playerlib.AISpellTarget{X: point.X, Y: point.Y, Plane: plane}
            found = true
        }
    }
    return target, found
}

// the wizards in the game's order, the human first
func chewGameOrder(world *chewSpellWorld) []*playerlib.Player {
    var out []*playerlib.Player
    if world.Human != nil {
        out = append(out, world.Human)
    }
    for _, player := range world.Players {
        if player != world.Human && !player.IsNeutral() {
            out = append(out, player)
        }
    }
    return out
}

// the global enchantments in the original's order of slots
var chewGlobalSlots = []data.Enchantment{
    data.EnchantmentEternalNight, data.EnchantmentEvilOmens, data.EnchantmentZombieMastery,
    data.EnchantmentAuraOfMajesty, data.EnchantmentWindMastery, data.EnchantmentSuppressMagic,
    data.EnchantmentTimeStop, data.EnchantmentNatureAwareness, data.EnchantmentNaturesWrath,
    data.EnchantmentHerbMastery, data.EnchantmentChaosSurge, data.EnchantmentDoomMastery,
    data.EnchantmentGreatWasting, data.EnchantmentMeteorStorm, data.EnchantmentArmageddon,
    data.EnchantmentTranquility, data.EnchantmentLifeForce, data.EnchantmentCrusade,
    data.EnchantmentJustCause, data.EnchantmentHolyArms, data.EnchantmentPlanarSeal,
    data.EnchantmentCharmOfLife, data.EnchantmentDetectMagic, data.EnchantmentAwareness,
}

// AITP_Disjunction: the global enchantment of another wizard with the highest score (the first on a
// tie); Spell Binding scores what it would like to have, Disjunction what hurts it
func chewDisjunctionTarget(world *chewSpellWorld, name string) (playerlib.AISpellTarget, bool) {
    self := world.Self
    has := func(realm data.MagicType) bool {
        return chewHasRealm(self.Wizard, realm)
    }
    primary, _ := chewRealms(self.Wizard)
    pick := func(yes bool, a int, b int) int {
        if yes {
            return a
        }
        return b
    }
    score := func(enchantment data.Enchantment) int {
        binding := name == "Spell Binding"
        switch enchantment {
            case data.EnchantmentEternalNight:
                if binding {
                    return pick(has(data.DeathMagic), 30, 100)
                }
                return pick(has(data.LifeMagic), 200, 50)
            case data.EnchantmentEvilOmens:
                if binding {
                    return 10
                }
                return pick(has(data.LifeMagic) || has(data.NatureMagic), 200, 0)
            case data.EnchantmentZombieMastery: return 50
            case data.EnchantmentAuraOfMajesty: return 20
            case data.EnchantmentWindMastery: return 15
            case data.EnchantmentSuppressMagic: return pick(binding, 100, 250)
            case data.EnchantmentTimeStop, data.EnchantmentNatureAwareness, data.EnchantmentPlanarSeal: return 10
            case data.EnchantmentNaturesWrath:
                if binding {
                    return 20
                }
                return pick(has(data.DeathMagic) || has(data.ChaosMagic), 200, 0)
            case data.EnchantmentHerbMastery:
                if binding {
                    return 20
                }
                return pick(primary == data.DeathMagic, 0, 50)
            case data.EnchantmentChaosSurge:
                if binding {
                    return pick(has(data.ChaosMagic), 30, 100)
                }
                return pick(has(data.ChaosMagic), 0, 100)
            case data.EnchantmentDoomMastery: return 50
            case data.EnchantmentGreatWasting: return 20
            case data.EnchantmentMeteorStorm: return 30
            case data.EnchantmentArmageddon: return 40
            case data.EnchantmentTranquility, data.EnchantmentLifeForce:
                if binding {
                    return 30
                }
                return pick(has(data.ChaosMagic), 200, 0)
            case data.EnchantmentCrusade: return 20
            case data.EnchantmentJustCause: return 15
            case data.EnchantmentHolyArms: return 30
            case data.EnchantmentCharmOfLife: return 60
            case data.EnchantmentDetectMagic, data.EnchantmentAwareness: return 1
        }
        return 0
    }
    best := 0
    var target playerlib.AISpellTarget
    found := false
    for _, other := range chewGameOrder(world) {
        if other == self {
            continue
        }
        for _, enchantment := range chewGlobalSlots {
            if !other.HasEnchantment(enchantment) {
                continue
            }
            value := score(enchantment)
            if quirkDisjunctionBytes {
                // kept in a signed byte
                value = int(int8(value))
            }
            if value > best {
                best = value
                target = playerlib.AISpellTarget{Player: other, Enchantment: enchantment}
                found = true
            }
        }
    }
    return target, found
}
