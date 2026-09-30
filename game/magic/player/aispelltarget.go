package player

import (
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// the target of a spell of the world map a computer player has cast: a square (a city, stack, node
// or piece of land), a unit, a wizard, a global enchantment of a wizard, a realm (Spell Ward). The
// original picks it when the casting is done, and without one the spell is lost
type AISpellTarget struct {
    X int
    Y int
    Plane data.Plane
    Unit units.StackUnit
    Player *Player
    Enchantment data.Enchantment
    CityEnchantment data.CityEnchantment
}

// an AI that picks the targets of its spells when they are cast
type AISpellTargetChooser interface {
    ChooseSpellTarget(self *Player, spell spellbook.Spell) (AISpellTarget, bool)
}
