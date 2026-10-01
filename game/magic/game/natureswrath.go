package game

// NATURE'S WRATH STRIKES AFTER A CAST THAT WENT THROUGH (user 2026-10-01: "Let's update all spells
// to be classic"). ReMoM OverSpel.c Cast_Spell_Overland: after the spell's effect, when the cast
// could go on (no fizzle, a target found, not countered by a city, the spell's own code did not
// refuse) and another wizard has Nature's Wrath, a chaos or death spell calls
// Call_Forth_The_Force_Of_Nature (Spells129.c) on its caster, and the human is told. Upstream
// struck at the start of every chaos and death cast, before a target was even picked.
//
// Here: doCastSpell notes the cast (wrathCast) and queues GameEventNaturesWrath behind whatever the
// spell queued itself (the choice of its square), so the strike comes after the effect. What stops
// a cast marks it failed: the human's cancel, a computer wizard with no target, a city's counters.

import (
    "fmt"

    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
)

// a chaos or death cast while another wizard has Nature's Wrath
type naturesWrathCast struct {
    Caster *playerlib.Player
    // the cast did not go through
    Failed bool
}

// the strike after a cast, queued behind it
type GameEventNaturesWrath struct {
    Cast *naturesWrathCast
}

// notes a cast that may call down Nature's Wrath. returns the function that queues the strike
func (game *Game) noteNaturesWrath(player *playerlib.Player, spell spellbook.Spell) func() {
    if !ClassicMagic || !(spell.IsOfRealm(data.ChaosMagic) || spell.IsOfRealm(data.DeathMagic)) {
        return func(){}
    }
    if !game.Model.HasRivalEnchantment(player, data.EnchantmentNaturesWrath) {
        return func(){}
    }
    cast := &naturesWrathCast{Caster: player}
    game.wrathCast = cast
    return func() {
        select {
            case game.Events <- &GameEventNaturesWrath{Cast: cast}:
            default:
        }
    }
}

// the cast of a player that is being made did not go through
func (game *Game) naturesWrathFailed(player *playerlib.Player) {
    if game.wrathCast != nil && game.wrathCast.Caster == player {
        game.wrathCast.Failed = true
    }
}

// the strike, if the cast went through
func (game *Game) doNaturesWrath(cast *naturesWrathCast) {
    if game.wrathCast == cast {
        game.wrathCast = nil
    }
    if cast.Failed || cast.Caster.Defeated {
        return
    }
    if !game.Model.HasRivalEnchantment(cast.Caster, data.EnchantmentNaturesWrath) {
        return
    }

    game.maybeDoNaturesWrath(cast.Caster)

    // the original tells the human every time
    message := fmt.Sprintf("Nature's Wrath strikes %v", cast.Caster.Wizard.Name)
    if cast.Caster.IsHuman() {
        message = "Nature's Wrath strikes you"
    }
    if human := game.Model.GetHumanPlayer(); human != nil {
        game.aiCastNotice(human, message)
    }
    game.RefreshUI()
}
