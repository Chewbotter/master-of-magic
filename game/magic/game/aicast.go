package game

// The spells of the world map a computer wizard casts: when the casting is done the AI picks the
// target (playerlib.AISpellTargetChooser, Chewbot's is ai/chewbotspells.go), as the original does
// (ReMoM MoM/src/OverSpel.c Cast_Spell_Overland: AITP_* for every kind of target; a spell without
// a target is lost). The upstream code of a spell asks the human for a square or opens a window;
// for a computer wizard the square comes from the AI and the effect of a window is applied at once.
// A cast is shown (camera, animation, pictures) only where the human sees the square now.

import (
    "fmt"
    "log"
    rand "github.com/kazzmir/master-of-magic/lib/chance"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    "github.com/kazzmir/master-of-magic/lib/coroutine"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

// how often a computer wizard's spell may ask for a square again (a spell that finds its square
// taken asks anew; the AI's second answer is the same, so it must end)
const aiCastTriesMost = 2

// the AI's target for a spell
func (game *Game) aiSpellTarget(player *playerlib.Player, spell spellbook.Spell) (playerlib.AISpellTarget, bool) {
    chooser, ok := player.AIBehavior.(playerlib.AISpellTargetChooser)
    if !ok {
        return playerlib.AISpellTarget{}, false
    }
    target, ok := chooser.ChooseSpellTarget(player, spell)
    if !ok {
        log.Printf("%v finds no target for %v", player.Wizard.Name, spell.Name)
    }
    return target, ok
}

// runs a computer wizard's cast on a square of a plane: the code of the spell reads the plane shown,
// so it is that plane for the time; the cast is quiet unless the human sees the square
func (game *Game) runAiCast(x int, y int, plane data.Plane, run func()) {
    human := game.Model.GetHumanPlayer()
    shown := game.Model.Plane
    seen := human != nil && plane == shown && human.IsVisible(x, y, plane) && !game.headless && !game.captureSkipping
    game.Model.Plane = plane
    game.quietCast = !seen
    defer func() {
        game.Model.Plane = shown
        game.quietCast = false
    }()
    run()
}

// a computer wizard's spell asks for a square
func (game *Game) aiSelectLocation(yield coroutine.YieldFunc, event *GameEventSelectLocationForSpell) {
    player := event.Player
    if game.aiCastTries == nil {
        game.aiCastTries = make(map[*playerlib.Player]int)
    }
    if game.aiCastTries[player] >= aiCastTriesMost {
        game.naturesWrathFailed(player)
        return
    }
    game.aiCastTries[player] += 1
    target, ok := game.aiSpellTarget(player, event.Spell)
    if !ok {
        // no target: no cast for Nature's Wrath (natureswrath.go)
        game.naturesWrathFailed(player)
        return
    }
    game.runAiCast(target.X, target.Y, target.Plane, func() {
        event.SelectedFunc(yield, target.X, target.Y)
    })
}

// a notice of the code of a spell (a target that does not fit): shown for the human's own casts only
func (game *Game) spellNotice(caster *playerlib.Player, message string) {
    if caster != nil && caster.IsAI() {
        return
    }
    game.Events <- &GameEventNotice{Message: message}
}

// a notice to the human about a spell of another wizard on the human
func (game *Game) aiCastNotice(target *playerlib.Player, message string) {
    if target == nil || !target.IsHuman() || message == "" || game.headless || game.captureSkipping {
        return
    }
    select {
        case game.Events <- &GameEventNotice{Message: message}:
        default:
    }
}

// the spells whose upstream code opens a window to choose in: for a computer wizard the effect is
// applied at once to the AI's target. false: not such a spell
func (game *Game) doCastSpellAI(player *playerlib.Player, spell spellbook.Spell) bool {
    switch spell.Name {
        case "Spell Blast", "Cruel Unminding", "Drain Power", "Subversion",
             "Disjunction", "Disjunction True", "Spell Binding", "Spell Ward":
        default:
            return false
    }
    target, ok := game.aiSpellTarget(player, spell)
    if !ok {
        game.naturesWrathFailed(player)
        return true
    }
    switch spell.Name {
        case "Spell Blast":
            if target.Player != nil && spellBlastEffect(player, target.Player) {
                game.aiCastNotice(target.Player, fmt.Sprintf("%v has blasted your spell", player.Wizard.Name))
            }
        case "Cruel Unminding":
            if target.Player != nil {
                if ok, message := cruelUnmindingEffect(target.Player); ok {
                    game.aiCastNotice(target.Player, fmt.Sprintf("%v has cast Cruel Unminding: %v", player.Wizard.Name, message))
                }
            }
        case "Drain Power":
            if target.Player != nil {
                if ok, message := drainPowerEffect(target.Player); ok {
                    game.aiCastNotice(target.Player, fmt.Sprintf("%v has cast Drain Power: %v", player.Wizard.Name, message))
                }
            }
        case "Subversion":
            if target.Player != nil && !target.Player.Defeated && !target.Player.Banished {
                game.Model.RelationRules().Subversion(player, target.Player)
                game.aiCastNotice(target.Player, fmt.Sprintf("%v has cast Subversion on you", player.Wizard.Name))
            }
        case "Disjunction", "Disjunction True":
            if target.Player != nil && game.disjunctionEffect(player, spell, target.Enchantment, target.Player) {
                game.aiCastNotice(target.Player, fmt.Sprintf("%v has disjuncted your %v", player.Wizard.Name, target.Enchantment.String()))
            }
        case "Spell Binding":
            if target.Player != nil && game.spellBindingEffect(player, target.Enchantment, target.Player) {
                game.aiCastNotice(target.Player, fmt.Sprintf("%v has stolen your %v", player.Wizard.Name, target.Enchantment.String()))
            }
        case "Spell Ward":
            city := player.FindCity(target.X, target.Y, target.Plane)
            if city != nil && !city.HasEnchantment(target.CityEnchantment) {
                select {
                    case game.Events <- &GameEventInvokeRoutine{Routine: func(yield coroutine.YieldFunc) {
                        game.runAiCast(target.X, target.Y, target.Plane, func() {
                            game.doAddCityEnchantment(yield, city, player, spell, target.CityEnchantment)
                        })
                    }}:
                    default:
                }
            }
    }
    return true
}

// Spell Blast: the target's spell is lost, the caster pays what the target had put into it
func spellBlastEffect(caster *playerlib.Player, target *playerlib.Player) bool {
    if ClassicMagic {
        // Spell Blast, see classicSpellBlast
        return classicSpellBlast(caster, target)
    }
    if target.Defeated || target.Banished || !target.CastingSpell.Valid() || target.CastingSpellProgress > caster.Mana {
        return false
    }
    caster.Mana -= target.CastingSpellProgress
    target.InterruptCastingSpell()
    return true
}

// Cruel Unminding: the target loses a tenth to a hundredth of its casting skill
func cruelUnmindingEffect(target *playerlib.Player) (bool, string) {
    if target.Defeated || target.Banished {
        return false, ""
    }
    targetSkill := target.ComputeCastingSkill()
    minReduction := max(targetSkill / 100, 1)
    maxReduction := max(targetSkill / 10, 1)
    // never zero, or rand.N fails
    reductionRandomSpread := max(maxReduction - minReduction, 1)
    reduction := minReduction + rand.N(reductionRandomSpread)
    if ClassicMagic {
        // Apply_Cruel_Unminding
        reduction = classicUnminding(targetSkill)
    }
    actuallyReduced := target.ReduceCastingSkill(reduction)
    return true, fmt.Sprintf("%s loses %d points of casting ability", target.Wizard.Name, actuallyReduced)
}

// Drain Power: the target loses 50 to 150 mana, as far as it has it
func drainPowerEffect(target *playerlib.Player) (bool, string) {
    if target.Defeated || target.Banished {
        return false, ""
    }
    drainAmount := min(target.Mana, 50 + rand.N(101))
    if ClassicMagic {
        // Apply_Drain_Power: the number told is the roll, the reserve goes as far as it has
        rolled := classicDrainAmount()
        target.Mana = max(0, target.Mana - rolled)
        return true, fmt.Sprintf("%s loses %d points of mana", target.Wizard.Name, rolled)
    }
    target.Mana -= drainAmount
    return true, fmt.Sprintf("%s loses %d points of mana", target.Wizard.Name, drainAmount)
}

// the strength of a Disjunction against global enchantments
func disjunctionStrength(caster *playerlib.Player, spell spellbook.Spell) int {
    strength := spell.Cost(true)
    if spell.Name == "Disjunction True" {
        strength *= 3
    }
    if caster.Wizard.RetortEnabled(data.RetortRunemaster) {
        strength *= 2
    }
    return strength
}

// Disjunction: the owner's global enchantment ends when the dispel roll succeeds
func (game *Game) disjunctionEffect(caster *playerlib.Player, spell spellbook.Spell, enchantment data.Enchantment, owner *playerlib.Player) bool {
    if !owner.HasEnchantment(enchantment) {
        return false
    }
    allSpells := game.AllSpells()
    targetSpell := allSpells.FindByName(enchantment.String())
    if spellbook.RollDispelChance(spellbook.ComputeDispelChance(disjunctionStrength(caster, spell), targetSpell.Cost(true), targetSpell.Magic, &owner.Wizard)) {
        owner.RemoveEnchantment(enchantment)
        return true
    }
    return false
}

// Spell Binding: the caster takes the owner's global enchantment when the dispel roll succeeds
func (game *Game) spellBindingEffect(caster *playerlib.Player, enchantment data.Enchantment, owner *playerlib.Player) bool {
    if !owner.HasEnchantment(enchantment) {
        return false
    }
    allSpells := game.AllSpells()
    targetSpell := allSpells.FindByName(enchantment.String())
    strength := spellBindingStrength
    if ClassicMagic && caster.IsAI() {
        // Cast_Spell_Binding: a computer wizard binds with the spell's table cost
        binding := game.AllSpells()
        strength = binding.FindByName("Spell Binding").CastCost
    }
    if spellbook.RollDispelChance(spellbook.ComputeDispelChance(strength, targetSpell.Cost(true), targetSpell.Magic, &owner.Wizard)) {
        owner.RemoveEnchantment(enchantment)
        caster.AddEnchantment(enchantment)
        game.ApplyGlobalEnchantment(enchantment, caster)
        return true
    }
    return false
}

// the dispel strength of Spell Binding as upstream has it
const spellBindingStrength = 20000

// Spell Blast: a computer caster pays the target spell's table cost less what is left of it when it
// has more, the human its cost less what is left when it has that much; the Spell of Return is not
// lost, it starts over (never below 0, MY CALL: the original's price can be below 0)
func classicSpellBlast(caster *playerlib.Player, target *playerlib.Player) bool {
    if target.Defeated || !target.CastingSpell.Valid() {
        return false
    }
    remaining := target.ComputeEffectiveSpellCost(target.CastingSpell, true) - target.CastingSpellProgress
    var price int
    if caster.IsAI() {
        price = max(0, target.CastingSpell.CastCost - remaining)
        if caster.Mana <= price {
            return false
        }
    } else {
        // Spells137.c: the cost the spell was started at (the table's or the wizard's, whichever
        // is higher, see classicCastStrength) less what is left: for a spell of both books that is
        // the progress, which the screen shows
        price = max(0, classicCastStrength(target, target.CastingSpell) - remaining)
        if price > caster.Mana {
            return false
        }
    }
    caster.Mana -= price
    if target.CastingSpell.Name == "Spell of Return" {
        target.CastingSpellProgress = 0
    } else {
        target.InterruptCastingSpell()
    }
    return true
}
