package combat

// The combat AI of Chewbot (the setting "Enemy AI", display/enemyai.go): the original game's
// combat AI, ported, before anything of it is changed. Written from the facts of the ReMoM
// project's reconstruction; the code is ours. Facts from MoM/src/CMBTAI.c (AI_Set_All_Action_Modes,
// AI_Execute_Unit_Action, Sort_Battle_Units, AI_Stage_Point_For_Unit, Auto_Do_Combat_Turn,
// AI_Set_Unit_Action_Mode, Choose_Target_And_Action, Do_Auto_Unit_Turn, Auto_Move_Unit) and
// MoM/src/Combat.c (Target_Unit_Value, Effective_Battle_Unit_Strength, Get_Effective_Melee_Strength,
// Get_Effective_Hits, Get_Effective_Ranged_Strength, Total_Ranged_Attack_Strength, AI_Player_Mode,
// Battle_Unit_Defense_Special, AI_Retreat_Check, Check_For_Winner). The rules in words, with what
// is kept of the original's mistakes and what is left out: docs/mod/ai-combat.md.
//
// How it fits the fork: the original plans the turn of a side once (the posture, the stage point)
// and then lets its units act one after the other, heroes a second time at the end. Here the plan
// is made at the first unit of a side's turn (chewbotPlan), and every unit acts in one call of
// doAIChewbot, which ends its turn; a hero that kept moves waits (chewbotWaits) and acts again
// after the others. Grid facts are in the original's cells, see toCell.

import (
    "image"
    stdlog "log"
    "math/rand/v2"
    "slices"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/game/magic/pathfinding"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    "github.com/kazzmir/master-of-magic/lib/fraction"
)

// false: the clone's combat AI whatever the setting says
var ChewbotCombatAI = true

// development: every decision of Chewbot in the log (-capture-ai-log)
var ChewbotLog = false

func chewLog(format string, args ...any) {
    if ChewbotLog {
        stdlog.Printf("chewbot: " + format, args...)
    }
}

// true when the units of computer players act by Chewbot's rules
func chewbotActive() bool {
    return ChewbotCombatAI && display.ChewbotAI()
}

// The original's own mistakes that change how it plays, kept on purpose (user 2026-09-29: "visible
// quirks are good ... we don't need to force the internal quirks"). false gives what it meant.
const (
    // Total_Ranged_Attack_Strength keeps the strength of the LAST unit with a ranged attack of a
    // side, not the sum of all
    quirkRangedStrengthOfLastUnit = true
    // a unit with the Healing ability picks it once and then does nothing in that turn: the
    // original has no action for it (Choose_Target_And_Action picks bua_Healing,
    // AI_Execute_Unit_Action has no case)
    quirkHealingWastesTurn = true
)

// what a unit does, the original's battle unit actions
type chewAction int
const (
    chewStab chewAction = iota
    chewShoot
    chewAdvanceAndStab
    chewAdvanceAndShoot
    // bua_Healing, see quirkHealingWastesTurn
    chewHealing
)

// the effects of an attack that count in its strength, the original's attack attributes
const (
    chewArmorPiercing = 1 << iota
    chewFirstStrike
    chewPoison
    chewLifeSteal
    chewDoom
    chewDestruction
    chewIllusion
    chewStoningTouch
    chewDeathTouch
    chewEldritch
)

// the gazes, the original's ranged types from srat_StoneGaze on
const (
    chewGazeNone = iota
    chewGazeOne
    // Chaos Spawn: all three gazes
    chewGazeMulti
)

// numbers of the original
const (
    // AI_Set_All_Action_Modes: the random offset Random(3) - 4, -3 to -1
    chewRandomOffsetLow = -3
    // Auto_Do_Combat_Turn: a defender leaves the shelter of its walls when the ranged strength of
    // the attacker is above this and its own is 0
    chewLeaveWallsRanged = 30
    // Auto_Do_Combat_Turn: neutral attackers that have not moved for more turns than this die
    chewImmobileTurns = 3
    // Target_Unit_Value
    chewUnreachableValue = -200
    chewNoValue = -100
    chewRangedTargetBonus = 2
    chewConfusedPenalty = 10
    chewMeldBonus = 3
    chewSafeTargetPenalty = 10
    chewSameTargetBonus = 10
    chewGateBonus = 5
    chewWallValue = -20
    chewSleepingBonus = 20
    chewReachBonus = 25
    // Choose_Target_And_Action: ranged targets this close in value count as equal
    chewRangedTie = 3
)

// the original's grid: 21 by 22 cells (cgx, cgy); ours is 30 by 30 tiles (x, y). see
// docs/mod/combat-overhaul.md
func toCell(point image.Point) image.Point {
    return image.Pt(point.Y - 3, 22 - point.X)
}

func fromCell(cell image.Point) image.Point {
    return image.Pt(22 - cell.Y, cell.X + 3)
}

// the city proper, the gate and the four cells in the middle of the city, in the original's cells
var chewCity = image.Rect(5, 10, 9, 14)
var chewGateCell = image.Pt(8, 12)
var chewCityMiddle = image.Rect(6, 11, 8, 13)

func withinCity(x int, y int) bool {
    return toCell(image.Pt(x, y)).In(chewCity)
}

// the walls of a battle as the AI sees them, Auto_Do_Combat_Turn's _ai_battlefield_city_walls
type chewWalls struct {
    Stone bool
    Fire bool
    Darkness bool
}

func (walls chewWalls) any() bool {
    return walls.Stone || walls.Fire || walls.Darkness
}

// what lasts through a battle
type chewbotState struct {
    // _ai_stay_in_city: a defender behind walls keeps to them
    StayInCity bool
    // every unit keeps its target from turn to turn, target_battle_unit_idx
    Targets map[*ArmyUnit]*ArmyUnit
    Plans map[Team]*chewbotPlan
    // the last turn a unit of a computer player found a way to move, for _ai_immobile_counter
    LastMoveTurn int
    MovedTurn int
    // units that have picked their Healing ability, see quirkHealingWastesTurn
    healed map[*ArmyUnit]bool
}

// the turn of one side
type chewbotPlan struct {
    Turn int
    Actions map[*ArmyUnit]chewAction
    Walls chewWalls
    Rally image.Point
    HasRally bool
    // 1: the hero acted and waits for its second turn; 2: the second turn has begun
    Pass map[*ArmyUnit]int
}

func (model *CombatModel) chewbotState() *chewbotState {
    if model.chewbot == nil {
        model.chewbot = &chewbotState{
            StayInCity: true,
            Targets: make(map[*ArmyUnit]*ArmyUnit),
            Plans: make(map[Team]*chewbotPlan),
        }
    }
    return model.chewbot
}

// a hero that acted with moves left and waits until the other units of its side have acted
func (model *CombatModel) chewbotWaits(unit *ArmyUnit) bool {
    if model.chewbot == nil {
        return false
    }
    plan := model.chewbot.Plans[unit.Team]
    return plan != nil && plan.Turn == model.CurrentTurn && plan.Pass[unit] == 1
}

// called when a unit's turn begins: a hero that waited now has its second turn
func (model *CombatModel) chewbotBeginUnit(unit *ArmyUnit) {
    if model.chewbotWaits(unit) {
        model.chewbot.Plans[unit.Team].Pass[unit] = 2
    }
}

func (model *CombatModel) chewAlive(unit *ArmyUnit) bool {
    return unit != nil && unit.GetHealth() > 0 && slices.Contains(model.GetArmy(unit).units, unit)
}

func (model *CombatModel) chewNeutral(army *Army) bool {
    wizard := army.Player.GetWizard()
    return wizard != nil && wizard.Banner == data.BannerBrown
}

// ------------------------------------------------------------------------------------------------
// the units as the original's numbers

// missile, boulder or magic, with shots left: Battle_Unit_Has_Ranged_Attack
func chewHasRanged(unit *ArmyUnit) bool {
    switch unit.GetRangedAttackDamageType() {
        case units.DamageRangedMagical, units.DamageRangedPhysical, units.DamageRangedBoulder:
            return unit.CanRangeAttack() && unit.GetRangedAttackPower() > 0
    }
    return false
}

// thrown, a breath or a gaze: the original's ranged types from srat_Thrown on
func chewShortRange(unit *ArmyUnit) bool {
    switch unit.GetRangedAttackDamageType() {
        case units.DamageThrown, units.DamageFire:
            return true
    }
    return unit.HasAbility(data.AbilityFireBreath) || unit.HasAbility(data.AbilityLightningBreath) || chewGaze(unit) != chewGazeNone
}

// no ranged attack of any kind: the original's rat_NONE
func chewNoRangedType(unit *ArmyUnit) bool {
    return unit.GetRangedAttackDamageType() == units.DamageNone && !chewShortRange(unit)
}

func chewGaze(unit *ArmyUnit) int {
    count := 0
    for _, gaze := range []data.AbilityType{data.AbilityStoningGaze, data.AbilityDeathGaze, data.AbilityDoomGaze} {
        if unit.HasAbility(gaze) {
            count += 1
        }
    }
    switch {
        case count > 1: return chewGazeMulti
        case count == 1: return chewGazeOne
    }
    return chewGazeNone
}

func chewMissile(unit *ArmyUnit) bool {
    return unit.GetRangedAttackDamageType() == units.DamageRangedPhysical
}

func chewMagicRanged(unit *ArmyUnit) bool {
    return unit.GetRangedAttackDamageType() == units.DamageRangedMagical
}

func chewMeleeFlags(unit *ArmyUnit) int {
    flags := 0
    add := func(ability data.AbilityType, flag int) {
        if unit.HasAbility(ability) {
            flags |= flag
        }
    }
    add(data.AbilityArmorPiercing, chewArmorPiercing)
    add(data.AbilityFirstStrike, chewFirstStrike)
    add(data.AbilityPoisonTouch, chewPoison)
    add(data.AbilityLifeSteal, chewLifeSteal)
    add(data.AbilityIllusion, chewIllusion)
    add(data.AbilityStoningTouch, chewStoningTouch)
    add(data.AbilityDeathTouch, chewDeathTouch)
    if unit.HasEnchantment(data.UnitEnchantmentEldritchWeapon) {
        flags |= chewEldritch
    }
    return flags
}

func chewRangedFlags(unit *ArmyUnit) int {
    flags := 0
    if unit.HasAbility(data.AbilityArmorPiercing) {
        flags |= chewArmorPiercing
    }
    if unit.HasAbility(data.AbilityIllusion) {
        flags |= chewIllusion
    }
    return flags
}

// the threat of an attack strength: Get_Effective_Melee_Strength and Get_Effective_Ranged_Strength
func chewThreat(strength int, figures int, perFigure int) int {
    if strength <= 0 {
        return 0
    }
    if strength > 3 {
        return figures * (perFigure / 2) * (strength - 2)
    }
    threat := figures * perFigure
    switch strength {
        case 1: return threat / 7
        case 2: return threat / 3
    }
    return threat * 2 / 3
}

func chewFlagThreat(threat int, figures int, flags int, melee bool) int {
    if flags == 0 {
        return threat
    }
    if flags & chewDoom != 0 {
        threat *= 2
    } else if flags & chewArmorPiercing != 0 {
        threat = threat * 5 / 4
    }
    for _, flag := range []int{chewPoison, chewLifeSteal, chewDestruction, chewStoningTouch, chewDeathTouch} {
        if flags & flag != 0 {
            threat += figures * 60
        }
    }
    if flags & chewIllusion != 0 {
        threat *= 5
    }
    if melee {
        if flags & chewFirstStrike != 0 {
            threat = threat * 5 / 4
        }
        if flags & chewEldritch != 0 {
            threat = threat * 5 / 4
        }
    }
    return threat
}

func chewEffectiveMelee(melee int, thrown int, figures int, flags int, gaze int) int {
    if melee == 0 {
        return 0
    }
    threat := chewThreat(melee, figures, 40) + chewThreat(thrown, figures, 60)
    threat = chewFlagThreat(threat, figures, flags, true)
    if gaze != chewGazeNone {
        threat += 300
        if gaze == chewGazeMulti {
            threat += 900
        }
    }
    return threat
}

// Get_Effective_Hits
func chewEffectiveHits(hits int, defense int) int {
    effective := hits * 3
    switch {
        case defense > 3: return effective * (((defense - 2) * (defense - 2)) / 2)
        case defense <= 0: return effective / 2
        case defense == 1: return effective * 3 / 5
        case defense == 2: return effective * 7 / 10
    }
    return effective * 4 / 5
}

// Effective_Battle_Unit_Strength
func chewEffectiveStrength(unit *ArmyUnit) int {
    figures := unit.Figures()
    thrown := 0
    gaze := chewGazeNone
    if chewShortRange(unit) {
        thrown = unit.GetRangedAttackPower()
        gaze = chewGaze(unit)
    }

    strength := chewEffectiveMelee(unit.GetMeleeAttackPower(), thrown, figures, chewMeleeFlags(unit), gaze)
    strength += chewEffectiveHits(unit.GetHealth(), unit.GetDefense())
    if chewHasRanged(unit) {
        strength += chewFlagThreat(chewThreat(unit.GetRangedAttackPower(), figures, 40), figures, chewRangedFlags(unit), false)
    }
    return strength
}

// Total_Ranged_Attack_Strength, see quirkRangedStrengthOfLastUnit
func (model *CombatModel) chewRangedStrength(army *Army) int {
    amount := 0
    for _, unit := range army.units {
        if unit.GetHealth() > 0 && chewHasRanged(unit) {
            if quirkRangedStrengthOfLastUnit {
                amount = unit.GetRangedAttackPower()
            } else {
                amount += unit.GetRangedAttackPower()
            }
        }
    }
    return amount
}

// AI_Player_Mode: 0 the enemy is more than 4 times as strong, 1 more than 3 times, 2 at least 1.5
// times, 3 about even, 4 own more than 3 times, 5 own more than 4 times or no enemy
func (model *CombatModel) chewPlayerMode(army *Army) int {
    chew := model.chewbotState()
    own, enemy := 0, 0
    ownAttack, enemyAttack := 0, 0

    for _, side := range []*Army{model.AttackingArmy, model.DefendingArmy} {
        for _, unit := range side.units {
            if unit.GetHealth() <= 0 {
                continue
            }
            // after the first turn a unit of the computer counts only while it has a target
            if !(model.CurrentTurn < 2 || chew.Targets[unit] != nil || side.Player.IsHuman()) {
                continue
            }
            strength := chewEffectiveStrength(unit)
            attack := unit.GetMeleeAttackPower() + unit.GetRangedAttackPower()
            if side == army {
                own += strength
                ownAttack += attack
            } else {
                enemy += strength
                enemyAttack += attack
            }
        }
    }

    switch {
        case own == 0 || ownAttack == 0: return 0
        case enemy == 0 || enemyAttack == 0: return 5
        case enemy / own > 3: return 0
        case enemy / own > 2: return 1
        case enemy * 2 >= own * 3: return 2
        case own / enemy > 3: return 5
        case own / enemy > 2: return 4
    }
    return 3
}

// Range_To_Battle_Unit: the larger of the two distances
func chewRange(a *ArmyUnit, b *ArmyUnit) int {
    return max(abs(a.X - b.X), abs(a.Y - b.Y))
}

func chewCrossesWalls(unit *ArmyUnit) bool {
    return unit.IsFlying() || unit.CanTeleport() || unit.HasAbility(data.AbilityMerging) || unit.HasAbility(data.AbilityNonCorporeal)
}

// the defense of a target against an attack: Battle_Unit_Defense_Special. 50 is immune
func (model *CombatModel) chewDefense(attacker *ArmyUnit, target *ArmyUnit, ranged bool) int {
    realm := data.MagicNone
    flags := chewMeleeFlags(attacker)
    if ranged {
        flags = chewRangedFlags(attacker)
        if chewMagicRanged(attacker) {
            realm = attacker.GetRangedMagicRealm()
        }
    } else if attacker.GetRace() == data.RaceFantastic {
        realm = attacker.GetRealm()
    }

    if flags & chewIllusion != 0 && !target.HasAbility(data.AbilityIllusionsImmunity) {
        return 0
    }

    defense := target.GetDefense()
    immune := false
    atLeastTen := false

    if ranged && target.HasAbility(data.AbilityLargeShield) {
        defense += 2
    }

    if ranged {
        switch {
            case chewMissile(attacker) && target.HasAbility(data.AbilityMissileImmunity): immune = true
            case chewMagicRanged(attacker) && target.HasAbility(data.AbilityMagicImmunity): immune = true
            case attacker.GetRangedAttackDamageType() == units.DamageFire && target.HasAbility(data.AbilityFireImmunity): immune = true
        }
    }

    // normal weapons against weapon immunity
    if (!ranged || chewMissile(attacker)) && target.HasAbility(data.AbilityWeaponImmunity) && !attacker.CanNegateWeaponImmunity() {
        atLeastTen = true
    }

    if realm == data.ChaosMagic || realm == data.DeathMagic {
        if target.HasEnchantment(data.UnitEnchantmentBless) {
            defense += 3
        }
        if ranged && target.HasEnchantment(data.UnitEnchantmentRighteousness) {
            immune = true
        }
    }

    if ranged && (realm == data.ChaosMagic || realm == data.NatureMagic) {
        if target.HasEnchantment(data.UnitEnchantmentElementalArmor) {
            defense += 10
        } else if target.HasEnchantment(data.UnitEnchantmentResistElements) {
            defense += 3
        }
    }

    if flags & chewArmorPiercing != 0 {
        defense /= 2
    }
    if atLeastTen && defense < 10 {
        defense = 10
    }
    if immune {
        return 50
    }
    return defense
}

// ------------------------------------------------------------------------------------------------
// the turn of a side

// the plan of a side's turn, made at its first unit: Auto_Do_Combat_Turn before its units act
func (model *CombatModel) chewbotPlanFor(army *Army) *chewbotPlan {
    chew := model.chewbotState()
    team := model.GetTeamForArmy(army)

    plan := chew.Plans[team]
    if plan != nil && plan.Turn == model.CurrentTurn {
        return plan
    }

    plan = &chewbotPlan{
        Turn: model.CurrentTurn,
        Actions: make(map[*ArmyUnit]chewAction),
        Pass: make(map[*ArmyUnit]int),
    }
    chew.Plans[team] = plan

    for unit, target := range chew.Targets {
        if !model.chewAlive(unit) || !model.chewAlive(target) {
            delete(chew.Targets, unit)
        }
    }

    other := model.GetOppositeArmyForTeam(team)

    // neutral attackers that have found no way to move for some turns are gone
    if model.chewNeutral(army) && team == TeamAttacker && model.chewImmobileTurns() > chewImmobileTurns {
        for _, unit := range slices.Clone(army.units) {
            unit.TakeDamage(unit.GetHealth(), DamageNormal)
            model.RemoveUnit(unit)
        }
        chewLog("neutral attackers could not move for %v turns and are gone", model.chewImmobileTurns())
        return plan
    }

    // a computer player that is losing badly flees to save its heroes or builders, not from its
    // own city under siege and not in the first turn
    if model.chewShouldFlee(army) {
        army.Fled = true
        chewLog("%v flees", team)
        return plan
    }

    ourRanged := model.chewRangedStrength(army)
    theirRanged := model.chewRangedStrength(other)

    if model.Zone.City != nil {
        _, gateY := model.GetCityGateCoordinates()
        plan.Walls.Stone = gateY >= 0
        plan.Walls.Fire = model.Zone.City.HasWallOfFire()
        plan.Walls.Darkness = model.Zone.City.HasWallOfDarkness()
    }

    // should the defender stay behind its walls?
    if team == TeamDefender {
        if chew.StayInCity {
            attackerVortexes := 0
            for _, vortex := range model.MagicVortexes {
                if vortex.Team == TeamAttacker {
                    attackerVortexes += 1
                }
            }

            leave := theirRanged > chewLeaveWallsRanged && ourRanged == 0
            if attackerVortexes > 0 || model.IsEnchantmentActive(data.CombatEnchantmentWrack, TeamAttacker) ||
               model.IsEnchantmentActive(data.CombatEnchantmentCallLightning, TeamAttacker) ||
               (model.IsEnchantmentActive(data.CombatEnchantmentManaLeak, TeamAttacker) && !model.chewNeutral(army)) {
                leave = true
            }

            if leave {
                plan.Walls = chewWalls{}
                chew.StayInCity = false
            }
        } else {
            plan.Walls = chewWalls{}
        }
    }

    model.chewSetAllActionModes(army, plan)

    // the stage point the melee units gather at: where the melee unit of middle speed gets to
    if ourRanged >= theirRanged {
        var melee []*ArmyUnit
        for _, unit := range army.units {
            if unit.GetHealth() > 0 && (plan.Actions[unit] == chewStab || plan.Actions[unit] == chewAdvanceAndStab) {
                melee = append(melee, unit)
            }
        }
        // slowest first, units of equal speed as they were
        slices.SortStableFunc(melee, func(a *ArmyUnit, b *ArmyUnit) int {
            switch {
                case a.MovesLeft.LessThan(b.MovesLeft): return -1
                case b.MovesLeft.LessThan(a.MovesLeft): return 1
            }
            return 0
        })
        if len(melee) > 0 {
            plan.Rally, plan.HasRally = model.chewStagePoint(melee[(len(melee) - 1) / 2], plan)
        }
    }

    chewLog("plan of %v in turn %v: walls %+v, rally %v %v", team, model.CurrentTurn, plan.Walls, plan.Rally, plan.HasRally)
    return plan
}

// turns in which no unit of a computer player found a way to move: _ai_immobile_counter
func (model *CombatModel) chewImmobileTurns() int {
    chew := model.chewbotState()
    if chew.MovedTurn == model.CurrentTurn {
        return -1
    }
    return model.CurrentTurn - chew.LastMoveTurn - 1
}

// AI_Retreat_Check, as Check_For_Winner asks it
func (model *CombatModel) chewShouldFlee(army *Army) bool {
    if army.Player.IsHuman() || model.chewNeutral(army) || army.Fled {
        return false
    }

    // a wizard's city under siege is never left
    if model.Zone.City != nil && !model.chewNeutral(model.DefendingArmy) {
        return false
    }

    if model.CurrentTurn <= 1 || model.chewPlayerMode(army) != 0 {
        return false
    }

    risked, heroes, builders := 0, 0, 0
    for _, unit := range army.units {
        if unit.GetHealth() <= 0 {
            continue
        }
        risked += 1
        if unit.Unit.IsHero() {
            heroes += 1
        }
        if unit.HasAbility(data.AbilityCreateOutpost) || unit.HasAbility(data.AbilityConstruction) {
            builders += 1
        }
    }

    return heroes > 0 || risked == builders
}

// AI_Set_All_Action_Modes: what the units of a side do, by the strength of both sides. only the
// stage point follows from it: every unit decides again when it acts (chewSetUnitActionMode)
func (model *CombatModel) chewSetAllActionModes(army *Army, plan *chewbotPlan) {
    other := model.GetOppositeArmyForTeam(model.GetTeamForArmy(army))

    ownRanged, enemyRanged := 0, 0
    ownThreat, enemyThreat := 0, 0

    for _, unit := range army.units {
        if unit.GetHealth() <= 0 {
            continue
        }
        if chewHasRanged(unit) {
            plan.Actions[unit] = chewShoot
        } else {
            plan.Actions[unit] = chewStab
        }
        ownThreat += chewEffectiveStrength(unit)
        ownRanged += int(unit.CastingSkill) / 4
    }
    for _, unit := range other.units {
        if unit.GetHealth() > 0 {
            enemyThreat += chewEffectiveStrength(unit)
            enemyRanged += int(unit.CastingSkill) / 4
        }
    }

    ownRanged += model.chewRangedStrength(army)
    enemyRanged += model.chewRangedStrength(other)

    if ownThreat == 0 || enemyThreat == 0 {
        return
    }

    mode := model.chewPlayerMode(army)
    rangedDiff := ownRanged - enemyRanged

    ownMelee := 0
    for unit, action := range plan.Actions {
        if action == chewStab {
            ownMelee += chewEffectiveStrength(unit)
        }
    }

    offset := chewRandomOffsetLow + model.chewRandom(3)

    for _, unit := range army.units {
        action, ok := plan.Actions[unit]
        if !ok {
            continue
        }

        switch mode {
            case 0, 1:
                if action == chewStab && rangedDiff > offset && ownRanged > 0 {
                    plan.Actions[unit] = chewAdvanceAndStab
                }
            case 2:
                if action == chewStab && ownMelee > 0 && enemyThreat / ownMelee > 2 && rangedDiff > offset && ownRanged > 0 {
                    plan.Actions[unit] = chewAdvanceAndStab
                }
                if action == chewShoot && ownMelee != 0 && enemyThreat / ownMelee <= 2 && rangedDiff >= 0 {
                    if !plan.Walls.any() || unit.Team == TeamAttacker {
                        plan.Actions[unit] = chewAdvanceAndShoot
                    }
                }
            case 3:
                if action == chewStab {
                    if rangedDiff > 0 {
                        plan.Actions[unit] = chewAdvanceAndStab
                    }
                } else if rangedDiff < 0 {
                    plan.Actions[unit] = chewAdvanceAndShoot
                }
            default:
                if action == chewShoot && model.chewRandom(2) == 1 {
                    plan.Actions[unit] = chewAdvanceAndShoot
                }
        }
    }
}

// the original's Random(n): 1 to n
func (model *CombatModel) chewRandom(n int) int {
    return 1 + rand.IntN(n)
}

// AI_Stage_Point_For_Unit: where a unit gets to in this turn on its way to its target. its target
// is the one of the turn before: the stage point is worked out before the units pick their
// targets, so in the first turn there is none
func (model *CombatModel) chewStagePoint(unit *ArmyUnit, plan *chewbotPlan) (image.Point, bool) {
    action := plan.Actions[unit]
    if action == chewAdvanceAndStab || action == chewShoot {
        return image.Pt(unit.X, unit.Y), true
    }

    target := model.chewbotState().Targets[unit]
    if !model.chewAlive(target) {
        return image.Point{}, false
    }

    path := model.chewPath(unit, image.Pt(target.X, target.Y), target, plan)
    if len(path) < 2 {
        return image.Point{}, false
    }

    moves := unit.MovesLeft
    spent := fraction.Zero()
    for index := 1; index < len(path); index++ {
        spent = spent.Add(model.StepCost(path[index - 1], path[index], unit.IsFlying()))
        if spent.GreaterThanEqual(moves) {
            return path[index], true
        }
    }
    return path[len(path) - 1], true
}

// ------------------------------------------------------------------------------------------------
// the turn of a unit

// one unit of a computer player acts, by Chewbot's rules. it acts once, and then has no moves
// left, but for a hero with moves left in its first turn: that waits, see chewbotWaits
func doAIChewbot(model *CombatModel, spellSystem SpellSystem, aiActions AIUnitActionsInterface, aiUnit *ArmyUnit) {
    army := model.GetArmy(aiUnit)
    chew := model.chewbotState()
    plan := model.chewbotPlanFor(army)

    endTurn := func() {
        aiUnit.MovesLeft = fraction.Zero()
    }

    if army.Fled || aiUnit.GetHealth() <= 0 || !model.chewAlive(aiUnit) {
        endTurn()
        return
    }

    secondPass := plan.Pass[aiUnit] == 2

    // NOT YET THE ORIGINAL'S: the spells of units (casters, Doom Bolt, Fireball, Web, Healing,
    // Summon Demon) are cast as the clone casts them; the original's spell AI is the next part of
    // the port. see docs/mod/ai-combat.md
    if model.doAIUnitCharges(spellSystem, aiActions, aiUnit, army) {
        endTurn()
        return
    }

    // heroes keep away from targets that would hurt them, but for their second turn when no unit
    // of a computer player has moved in this turn
    safetyOff := secondPass && model.chewImmobileTurns() != -1

    model.chewSetUnitActionMode(aiUnit, plan, safetyOff)

    // the stage point: a defender outside of its walls goes for its target
    rally, hasRally := plan.Rally, plan.HasRally
    if aiUnit.Team == TeamDefender && plan.Walls.any() && !withinCity(aiUnit.X, aiUnit.Y) {
        hasRally = false
    }

    target := chew.Targets[aiUnit]
    chewLog("%v at %v,%v action %v target %v", aiUnit.Unit.GetName(), aiUnit.X, aiUnit.Y, plan.Actions[aiUnit], chewName(target))

    model.chewExecute(aiActions, aiUnit, plan, rally, hasRally, safetyOff)

    if aiUnit.GetHealth() > 0 && aiUnit.Unit.IsHero() && !secondPass && aiUnit.MovesLeft.GreaterThan(fraction.Zero()) && model.chewAlive(aiUnit) {
        plan.Pass[aiUnit] = 1
        return
    }

    endTurn()
}

func chewName(unit *ArmyUnit) string {
    if unit == nil {
        return "none"
    }
    return unit.Unit.GetName()
}

// AI_Set_Unit_Action_Mode: the unit picks its target and whether it shoots or fights
func (model *CombatModel) chewSetUnitActionMode(unit *ArmyUnit, plan *chewbotPlan, safetyOff bool) *ArmyUnit {
    chew := model.chewbotState()
    ranged := chewHasRanged(unit)

    target, healing := model.chewChooseTarget(unit, ranged, plan, safetyOff)
    if healing {
        plan.Actions[unit] = chewHealing
        return nil
    }

    // a missile against a target immune to it: fight instead if the unit is about as good at that
    if target != nil && ranged && chewMissile(unit) && target.HasAbility(data.AbilityMissileImmunity) {
        if unit.GetMeleeAttackPower() * 3 >= unit.GetRangedAttackPower() * 2 {
            target = nil
        }
    }

    if ranged && target == nil {
        ranged = false
        target, _ = model.chewChooseTarget(unit, false, plan, safetyOff)
    }

    if ranged {
        plan.Actions[unit] = chewShoot
    } else {
        plan.Actions[unit] = chewStab
    }

    // a closer unit of the same kind and defense instead; and melee units join what a unit before
    // them in the army goes for. not for attackers while there are walls
    if target != nil && (unit.Team == TeamDefender || !plan.Walls.any()) {
        other := model.GetOtherArmy(unit)
        for _, candidate := range other.units {
            if candidate.GetHealth() <= 0 || candidate == target {
                continue
            }
            if candidate.Unit.GetName() == target.Unit.GetName() && candidate.GetDefense() == target.GetDefense() && chewRange(unit, candidate) < chewRange(unit, target) {
                target = candidate
            }
        }

        if !ranged {
            for _, friend := range model.GetArmy(unit).units {
                if friend == unit {
                    break
                }
                friendTarget := chew.Targets[friend]
                if friend.GetHealth() <= 0 || !model.chewAlive(friendTarget) {
                    continue
                }
                if friendTarget.Unit.GetName() == target.Unit.GetName() && friendTarget.GetDefense() == target.GetDefense() {
                    target = friendTarget
                }
            }
        }
    }

    if target == nil {
        delete(chew.Targets, unit)
    } else {
        chew.Targets[unit] = target
    }
    return target
}

// Choose_Target_And_Action: the enemy of the highest value. true for a unit that picks its
// Healing ability, see quirkHealingWastesTurn
func (model *CombatModel) chewChooseTarget(unit *ArmyUnit, ranged bool, plan *chewbotPlan, safetyOff bool) (*ArmyUnit, bool) {
    chew := model.chewbotState()

    if quirkHealingWastesTurn && unit.HasAbility(data.AbilityHealingSpell) && !chew.usedHealing(unit) {
        for _, friend := range model.GetArmy(unit).units {
            if friend.GetHealth() > 0 && friend.GetHealth() < friend.GetMaxHealth() {
                chew.markHealing(unit)
                return nil, true
            }
        }
    }

    best := chewNoValue
    var target *ArmyUnit

    gateX, gateY := model.GetCityGateCoordinates()
    atGate := func(other *ArmyUnit) bool {
        return gateY >= 0 && other.X == gateX && other.Y == gateY
    }

    for _, other := range model.GetOtherArmy(unit).units {
        if other.GetHealth() <= 0 || other == unit {
            continue
        }

        if ranged && other.IsInvisible() && !unit.HasAbility(data.AbilityIllusionsImmunity) {
            continue
        }

        if ranged && model.Zone.City != nil && model.Zone.City.HasWallOfDarkness() && !unit.HasAbility(data.AbilityIllusionsImmunity) {
            if !withinCity(unit.X, unit.Y) && withinCity(other.X, other.Y) {
                continue
            }
        }

        // an invisible unit is still a target when it blocks the gate
        if !unit.CanSee(other) && !atGate(other) {
            continue
        }

        // heroes do not walk into fights they would lose
        if unit.Unit.IsHero() && !ranged && !safetyOff {
            if unit.GetMeleeAttackPower() < other.GetDefense() {
                continue
            }
            if unit.GetDefense() < other.GetMeleeAttackPower() * 2 / 3 {
                continue
            }
            if chewShortRange(other) && unit.GetDefense() < other.GetRangedAttackPower() * 2 / 3 {
                continue
            }
        }

        // a defender inside its walls shoots at what is inside, or next to it
        if plan.Walls.any() && ranged && chewRange(unit, other) != 1 {
            if unit.Team == TeamDefender && withinCity(unit.X, unit.Y) {
                if !(atGate(unit) && plan.Walls.Stone) && !withinCity(other.X, other.Y) {
                    continue
                }
            }
        }

        value := model.chewTargetValue(unit, other, ranged, plan)

        health := func(which *ArmyUnit) int {
            return which.GetHealth()
        }

        if ranged {
            // about as good: the one with less health left
            if target != nil && value >= best - chewRangedTie && value <= best + chewRangedTie {
                if health(other) < health(target) {
                    best = max(best, value)
                    target = other
                }
            } else if value > best {
                best = value
                target = other
            }
        } else if value > best || (value == best && target != nil && health(other) > health(target)) {
            // as good: the one with more health left
            best = value
            target = other
        } else if value == best && target == nil {
            best = value
            target = other
        }
    }

    return target, false
}

func (chew *chewbotState) usedHealing(unit *ArmyUnit) bool {
    return chew.healed[unit]
}

func (chew *chewbotState) markHealing(unit *ArmyUnit) {
    if chew.healed == nil {
        chew.healed = make(map[*ArmyUnit]bool)
    }
    chew.healed[unit] = true
}

// Target_Unit_Value: how much a unit wants to attack a target
func (model *CombatModel) chewTargetValue(attacker *ArmyUnit, target *ArmyUnit, ranged bool, plan *chewbotPlan) int {
    chew := model.chewbotState()

    // it cannot reach what flies
    if target.IsFlying() && !attacker.IsFlying() && chewNoRangedType(attacker) {
        return chewUnreachableValue
    }

    targetAttack := max(target.GetRangedAttackPower(), target.GetMeleeAttackPower())

    attack := attacker.GetMeleeAttackPower()
    if ranged {
        attack = attacker.GetRangedAttackPower()
    }

    value := attack - model.chewDefense(attacker, target, ranged)

    // what shoots is dangerous
    if chewHasRanged(target) {
        value += target.GetRangedAttackPower() / 3 + chewRangedTargetBonus
    }

    if target.HasCurse(data.UnitCurseConfusion) {
        value -= chewConfusedPenalty
    }

    skipSafety := false
    if model.InsideMagicNode() && value > 0 && attacker.Team == TeamDefender && target.HasAbility(data.AbilityMeld) {
        // spirits that would take the node
        value += chewMeldBonus
        skipSafety = true
    }

    // a target that can hardly hurt the attacker is not worth much
    if !skipSafety && value > 0 && attacker.GetDefense() / 2 > targetAttack {
        value -= chewSafeTargetPenalty
    }

    if chew.Targets[attacker] == target {
        value += chewSameTargetBonus
    }

    targetCell := toCell(image.Pt(target.X, target.Y))

    if plan.Walls.Stone && !ranged && attacker.Team == TeamAttacker && !chewCrossesWalls(attacker) {
        if model.ContainsWall(target.X, target.Y) {
            if targetCell == chewGateCell {
                value += chewGateBonus
            } else {
                value = chewWallValue
            }
        }
        if targetCell.In(chewCityMiddle) {
            value = chewWallValue
        }
    }

    // the original weighs the damage a Fireball would do to the attacker as what the wall of fire
    // does; here any unit that is not immune to fire counts as hurt by it
    if plan.Walls.Fire && !ranged && attacker.Team == TeamAttacker && !attacker.IsFlying() && !attacker.CanTeleport() && !attacker.HasAbility(data.AbilityMerging) {
        if withinCity(target.X, target.Y) && !withinCity(attacker.X, attacker.Y) {
            if !attacker.HasAbility(data.AbilityFireImmunity) && !attacker.HasAbility(data.AbilityMagicImmunity) {
                value = chewWallValue
            }
        }
    }

    if ranged && target.IsAsleep() {
        value += chewSleepingBonus
    }

    if !ranged {
        // it gets there in this turn
        if fraction.FromInt(chewRange(attacker, target)).LessThanEqual(attacker.MovesLeft) {
            if target.IsAsleep() {
                value += chewSleepingBonus
            }
            value += chewReachBonus
        }
    }

    return value
}

// AI_Execute_Unit_Action
func (model *CombatModel) chewExecute(actions AIUnitActionsInterface, unit *ArmyUnit, plan *chewbotPlan, rally image.Point, hasRally bool, safetyOff bool) {
    chew := model.chewbotState()

    switch plan.Actions[unit] {
        case chewHealing:
            // see quirkHealingWastesTurn
            return

        case chewShoot:
            for range 8 {
                target := chew.Targets[unit]
                if !model.chewAlive(target) || unit.MovesLeft.LessThanEqual(fraction.Zero()) || unit.GetHealth() <= 0 {
                    return
                }
                before := unit.MovesLeft

                if chewHasRanged(unit) && model.canRangeAttack(unit, target) {
                    actions.RangeAttack(unit, target)
                } else {
                    model.chewAutoTurn(actions, unit, image.Pt(target.X, target.Y), target, rally, hasRally, plan)
                }

                if !model.chewAlive(target) {
                    model.chewSetUnitActionMode(unit, plan, safetyOff)
                }
                if !unit.MovesLeft.LessThan(before) {
                    return
                }
            }

        default:
            if unit.GetMeleeAttackPower() <= 0 {
                return
            }

            target := chew.Targets[unit]

            // walls: without a target an attacker goes for the gate, and a defender guards it
            if plan.Walls.Stone && target == nil {
                gateX, gateY := model.GetCityGateCoordinates()
                if gateY >= 0 {
                    gate := image.Pt(gateX, gateY)
                    model.chewAutoTurn(actions, unit, gate, nil, gate, false, plan)
                }
                return
            }

            // neutral attackers of a city with nothing to attack go into the city
            if target == nil {
                if model.chewNeutral(model.GetArmy(unit)) && unit.Team == TeamAttacker && model.Zone.City != nil {
                    model.chewAutoTurn(actions, unit, image.Pt(unit.X, unit.Y), nil, image.Point{}, false, plan)
                }
                return
            }

            for range 8 {
                target = chew.Targets[unit]
                if !model.chewAlive(target) {
                    return
                }
                before := unit.MovesLeft

                model.chewAutoTurn(actions, unit, image.Pt(target.X, target.Y), target, rally, hasRally, plan)

                if !model.chewAlive(target) {
                    model.chewSetUnitActionMode(unit, plan, safetyOff)
                }
                if !unit.MovesLeft.LessThan(before) || chew.Targets[unit] == nil || unit.MovesLeft.LessThanEqual(fraction.Zero()) || unit.GetHealth() <= 0 {
                    return
                }
            }
    }
}

// Do_Auto_Unit_Turn: an attacker that has no target, or cannot get to it, first goes for the
// nearest cell of the city it can get to; then the move toward the destination
func (model *CombatModel) chewAutoTurn(actions AIUnitActionsInterface, unit *ArmyUnit, destination image.Point, target *ArmyUnit, rally image.Point, hasRally bool, plan *chewbotPlan) {
    if unit.Team == TeamAttacker && model.Zone.City != nil {
        blocked := target == nil || len(model.chewPath(unit, destination, target, plan)) == 0
        if blocked {
            best := 0
            var stomp image.Point
            for cx := chewCity.Min.X; cx < chewCity.Max.X; cx++ {
                for cy := chewCity.Min.Y; cy < chewCity.Max.Y; cy++ {
                    cell := fromCell(image.Pt(cx, cy))
                    path := model.chewPath(unit, cell, nil, plan)
                    if len(path) > 0 && (best == 0 || len(path) < best) {
                        best = len(path)
                        stomp = cell
                    }
                }
            }
            if best > 0 {
                model.chewAutoMove(actions, unit, stomp, nil, stomp, false, plan)
            }
        }
    }

    if unit.GetHealth() > 0 && unit.MovesLeft.GreaterThan(fraction.Zero()) && model.chewAlive(unit) {
        model.chewAutoMove(actions, unit, destination, target, rally, hasRally, plan)
    }
}

// the cells a unit may walk through, the original's movement cost map: units other than the target
// and magic vortexes are in the way, and:
// - a unit outside of stone walls enters the city only at its gate
// - a defender that keeps to walls of fire or darkness does not leave them
func (model *CombatModel) chewPath(unit *ArmyUnit, to image.Point, target *ArmyUnit, plan *chewbotPlan) pathfinding.Path {
    chew := model.chewbotState()

    inside := withinCity(unit.X, unit.Y)
    gateX, gateY := model.GetCityGateCoordinates()
    gate := image.Pt(gateX, gateY)

    keepOutOfCity := plan.Walls.Stone && !inside && !chewCrossesWalls(unit)
    keepInCity := unit.Team == TeamDefender && chew.StayInCity && (plan.Walls.Fire || plan.Walls.Darkness) && inside

    avoid := func(x int, y int) bool {
        point := image.Pt(x, y)
        if keepOutOfCity && withinCity(x, y) && point != gate && point != to {
            return true
        }
        if keepInCity && !withinCity(x, y) {
            return true
        }
        return false
    }

    if avoid(to.X, to.Y) && !(target != nil && chewPoint(target) == to) {
        return nil
    }

    // the target's tile counts as free, as for the clone
    var tile *Tile
    var old *ArmyUnit
    if target != nil {
        tile = model.GetTile(target.X, target.Y)
        if tile != nil {
            old = tile.Unit
            tile.Unit = nil
        }
    }

    path, ok := model.computePathAvoiding(unit.X, unit.Y, to.X, to.Y, unit.CanTraverseWall(), unit.IsFlying(), unit.CanEnterWater(), avoid)

    if tile != nil {
        tile.Unit = old
    }

    if !ok {
        return nil
    }
    return path
}

func chewPoint(unit *ArmyUnit) image.Point {
    return image.Pt(unit.X, unit.Y)
}

// the cells a unit may move to in one move toward a destination, in the original's cells: it does
// not go past the stage point in the direction it moves, so the melee units of a side gather there
type chewBox struct {
    Min image.Point
    Max image.Point
}

const chewBoxFar = 1000

func makeChewBox(origin image.Point, destination image.Point, rally image.Point, hasRally bool) chewBox {
    full := chewBox{Min: image.Pt(-chewBoxFar, -chewBoxFar), Max: image.Pt(chewBoxFar, chewBoxFar)}
    if !hasRally || rally == destination {
        return full
    }

    from, to, stage := toCell(origin), toCell(destination), toCell(rally)
    box := full
    // the original leaves the stage point as the far corner where it sets nothing else
    box.Max = stage

    deltaX := abs(from.X - to.X)
    deltaY := abs(from.Y - to.Y)

    limitX := func() {
        if from.X < to.X {
            box.Min.X = -chewBoxFar
        } else {
            box.Min.X = stage.X
            box.Max.X = chewBoxFar
        }
    }
    limitY := func() {
        if from.Y < to.Y {
            box.Min.Y = -chewBoxFar
        } else {
            box.Min.Y = stage.Y
            box.Max.Y = chewBoxFar
        }
    }

    switch {
        case deltaX < deltaY:
            box.Min.X, box.Max.X = -chewBoxFar, chewBoxFar
            limitY()
        case deltaX > deltaY:
            box.Min.Y, box.Max.Y = -chewBoxFar, chewBoxFar
            limitX()
        default:
            limitX()
            limitY()
    }
    return box
}

func (box chewBox) contains(tile image.Point) bool {
    cell := toCell(tile)
    return cell.X >= box.Min.X && cell.X <= box.Max.X && cell.Y >= box.Min.Y && cell.Y <= box.Max.Y
}

// Auto_Move_Unit: the unit walks along its path as far as its moves and the box let it, and
// attacks its target when it gets to it
func (model *CombatModel) chewAutoMove(actions AIUnitActionsInterface, unit *ArmyUnit, destination image.Point, target *ArmyUnit, rally image.Point, hasRally bool, plan *chewbotPlan) {
    chew := model.chewbotState()

    if target != nil {
        destination = chewPoint(target)
    }

    path := model.chewPath(unit, destination, target, plan)
    if len(path) == 0 {
        return
    }

    if !model.GetArmy(unit).Player.IsHuman() {
        chew.MovedTurn = model.CurrentTurn
        chew.LastMoveTurn = model.CurrentTurn
    }

    box := makeChewBox(chewPoint(unit), destination, rally, hasRally)

    attack := func() {
        if target != nil && model.chewAlive(target) && unit.GetHealth() > 0 && model.withinMeleeRange(unit, target) && model.canMeleeAttack(unit, target, true) {
            actions.MeleeAttack(unit, target)
        }
    }

    // a unit that teleports or merges goes to the cell before its target, or to the end of its path
    if unit.CanTeleport() || unit.HasAbility(data.AbilityMerging) {
        last := len(path) - 1
        if target != nil && path[last] == chewPoint(target) {
            last -= 1
        }
        if last >= 1 {
            cell := path[last]
            if box.contains(cell) || cell == destination {
                actions.Teleport(unit, cell.X, cell.Y, unit.HasAbility(data.AbilityMerging))
            }
        }
        attack()
        return
    }

    var walk pathfinding.Path
    for _, cell := range path[1:] {
        if target != nil && cell == chewPoint(target) {
            break
        }
        if !box.contains(cell) && cell != destination {
            break
        }
        walk = append(walk, cell)
    }

    // as far as its moves take it by the fork's rules
    for len(walk) > 0 && !unit.CanFollowPath(append(pathfinding.Path{chewPoint(unit)}, walk...), false) {
        walk = walk[:len(walk) - 1]
    }

    if len(walk) > 0 {
        actions.MoveUnit(unit, walk)
    }

    attack()
}
