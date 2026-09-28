package mod

// The pictures of the spells of battles, by name.
//
// The game keeps them in three archives by number. The export writes each of them to a folder with
// the name of the spell as well, and the replacement folder is read the same way:
//
//   spells/<name>/<frame>.png      <frame> with 2 digits: 00.png, 01.png
//
// A picture in spells/ goes before one in archives/. Several spells of a realm show the same
// picture, the one of the realm: a change to it shows for all of them.

import (
    "fmt"
    "path/filepath"
)

const spellsFolder = "spells"

type SpellPicture struct {
    // the name of the folder
    Name string
    Archive string
    Entry int
    // what shows it
    Note string
}

var SpellPictures = []SpellPicture{
    {Name: "Fire Bolt", Archive: "cmbtfx.lbx", Entry: 0, Note: "Fire Bolt. Frames 0 to 2 in flight, 3 where it hits"},
    {Name: "Disrupt", Archive: "cmbtfx.lbx", Entry: 1, Note: "Disrupt"},
    {Name: "Warp Wood", Archive: "cmbtfx.lbx", Entry: 2, Note: "Warp Wood"},
    {Name: "Warp Lightning", Archive: "cmbtfx.lbx", Entry: 3, Note: "Warp Lightning"},
    {Name: "Disintegrate", Archive: "cmbtfx.lbx", Entry: 4, Note: "Disintegrate"},
    {Name: "Doom Bolt", Archive: "cmbtfx.lbx", Entry: 5, Note: "Doom Bolt. It comes straight down, a frame for every 4 steps of its way"},
    {Name: "Life Drain", Archive: "cmbtfx.lbx", Entry: 6, Note: "Life Drain"},
    {Name: "Possession", Archive: "cmbtfx.lbx", Entry: 8, Note: "Possession"},
    {Name: "Star Fires", Archive: "cmbtfx.lbx", Entry: 9, Note: "Star Fires"},
    {Name: "Dispel Evil", Archive: "cmbtfx.lbx", Entry: 10, Note: "Dispel Evil"},
    {Name: "Ice Bolt", Archive: "cmbtfx.lbx", Entry: 11, Note: "Ice Bolt. Frames 0 to 2 in flight, 3 where it hits"},
    {Name: "Petrify", Archive: "cmbtfx.lbx", Entry: 12, Note: "Petrify"},
    {Name: "Web", Archive: "cmbtfx.lbx", Entry: 13, Note: "Web"},
    {Name: "Cracks Call", Archive: "cmbtfx.lbx", Entry: 15, Note: "Cracks Call. Lies on the ground, under the units"},
    {Name: "Psionic Blast", Archive: "cmbtfx.lbx", Entry: 16, Note: "Psionic Blast"},
    {Name: "Vertigo", Archive: "cmbtfx.lbx", Entry: 17, Note: "Vertigo, and the mark over a unit that has it"},
    {Name: "Banish", Archive: "cmbtfx.lbx", Entry: 19, Note: "Banish"},
    {Name: "Confusion", Archive: "cmbtfx.lbx", Entry: 20, Note: "Confusion"},
    {Name: "Mind Storm", Archive: "cmbtfx.lbx", Entry: 21, Note: "Mind Storm"},
    {Name: "Summoning Circle", Archive: "cmbtfx.lbx", Entry: 22, Note: "Every unit that is summoned in a battle. Lies on the ground"},
    {Name: "Fireball", Archive: "cmbtfx.lbx", Entry: 23, Note: "Fireball. Frames 0 to 10 in flight, 11 to 15 where it hits"},
    {Name: "Lightning Bolt", Archive: "cmbtfx.lbx", Entry: 24, Note: "Lightning Bolt. As high as the screen, its lower end is where it hits. One of the frames by chance for each flash"},
    {Name: "Dispel Magic", Archive: "cmbtfx.lbx", Entry: 26, Note: "Dispel Magic, Dispel Magic True"},
    {Name: "Flame Strike", Archive: "cmbtfx.lbx", Entry: 33, Note: "Flame Strike, on every unit of the enemy"},
    {Name: "Realm Nature", Archive: "specfx.lbx", Entry: 0, Note: "Elemental Armor, Giant Strength, Iron Skin, Stone Skin, Regeneration, Resist Elements"},
    {Name: "Realm Sorcery", Archive: "specfx.lbx", Entry: 1, Note: "Flight, Guardian Wind, Haste, Invisibility, Magic Immunity, Resist Magic, Spell Lock, Creature Binding, Word of Recall"},
    {Name: "Realm Chaos", Archive: "specfx.lbx", Entry: 2, Note: "Eldritch Weapon, Flame Blade, Immolation, Chaos Channels, Shatter, Warp Creature"},
    {Name: "Realm Life", Archive: "specfx.lbx", Entry: 3, Note: "Bless, Healing, Heroism, Holy Armor, Holy Weapon, Invulnerability, Lion Heart, Righteousness, True Sight, Holy Word"},
    {Name: "Realm Death", Archive: "specfx.lbx", Entry: 4, Note: "Berserk, Cloak of Fear, Wraith Form, Weakness, Black Sleep"},
    {Name: "Realm Arcane", Archive: "specfx.lbx", Entry: 5, Note: "Recall Hero"},
    {Name: "Death Spell", Archive: "specfx.lbx", Entry: 13, Note: "Death Spell, Word of Death"},
    {Name: "Magic Vortex", Archive: "cmbmagic.lbx", Entry: 120, Note: "Magic Vortex"},
}

// by archive and entry: the folder of the spell
var spellEntries = makeSpellEntries()

func makeSpellEntries() map[string]string {
    out := make(map[string]string)
    for _, spell := range SpellPictures {
        out[entryKey(spell.Archive, spell.Entry)] = spell.Name
    }
    return out
}

func SpellFrameFile(frame int) string {
    return fmt.Sprintf("%02d.png", frame)
}

// the file of a frame of a spell in the replacement folder, or nothing if the entry is no spell
func spellFramePath(archive string, entry int, frame int) string {
    name, ok := spellEntries[entryKey(archive, entry)]
    if !ok {
        return ""
    }
    return filepath.Join(folder, spellsFolder, name, SpellFrameFile(frame))
}
