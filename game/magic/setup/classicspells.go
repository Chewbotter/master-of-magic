package setup

// The spells a wizard starts with when they are not picked by hand, as the original game has
// them (ReMoM MoM/src/NewGame.c _default_spells; INITGAME.c: computer players know the first
// books - 1 of the list of each realm, at 11 books all 13). The same list fills the picks of the
// new game screen. The code is ours; upstream picked at random.

import (
    "log"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
)

// false: upstream's picks by chance
var ClassicStartingSpells = true

// per realm: 10 commons, 2 uncommons, 1 rare, in the original's order
var defaultSpells = map[data.MagicType][]string{
    data.NatureMagic: {"War Bears", "Stone Skin", "Sprites", "Water Walking", "Giant Strength", "Web", "Earth to Mud", "Wall of Stone", "Resist Elements", "Earth Lore", "Cockatrices", "Change Terrain", "Gorgons"},
    data.SorceryMagic: {"Nagas", "Psionic Blast", "Phantom Warriors", "Floating Island", "Confusion", "Counter Magic", "Word of Recall", "Dispel Magic True", "Resist Magic", "Guardian Wind", "Flight", "Phantom Beast", "Storm Giant"},
    data.ChaosMagic: {"Fire Bolt", "Fire Elemental", "Eldritch Weapon", "Hell Hounds", "Corruption", "Warp Creature", "Shatter", "Wall of Fire", "Disrupt", "Warp Wood", "Lightning Bolt", "Doom Bat", "Efreet"},
    data.LifeMagic: {"Heroism", "Guardian Spirit", "Holy Armor", "Just Cause", "Healing", "Holy Weapon", "Star Fires", "Endurance", "True Light", "Bless", "Resurrection", "Unicorns", "Angel"},
    data.DeathMagic: {"Life Drain", "Ghouls", "Weakness", "Dark Rituals", "Black Sleep", "Darkness", "Terror", "Skeletons", "Mana Leak", "Cloak of Fear", "Black Prayer", "Black Channels", "Wraiths"},
}

func classicStartingSpells(wizard *WizardCustom, allSpells spellbook.Spells) spellbook.Spells {
    var out spellbook.Spells
    for _, magic := range []data.MagicType{data.NatureMagic, data.SorceryMagic, data.ChaosMagic, data.LifeMagic, data.DeathMagic} {
        books := wizard.MagicLevel(magic)
        count := books - 1
        if books >= 11 {
            count = len(defaultSpells[magic])
        }
        for _, name := range defaultSpells[magic][:max(0, min(count, len(defaultSpells[magic])))] {
            spell := allSpells.FindByName(name)
            if !spell.Valid() {
                log.Printf("Starting spells: no spell %q in the data", name)
                continue
            }
            out.AddSpell(spell)
        }
    }
    return out
}
