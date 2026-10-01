# Overland magic

The original's rules. Read from the ReMoM reconstruction (`MoM/src/Spells129.c`, `Spells130.c`,
`Spells132.c`, `OverSpel.c`, `SBookScr.c`, `SPLMASTR.c`, `Terrain.c`, `NEXTTURN.c`; the functions
are named in `game/magicclassic.go`). Code: `game/magicclassic.go` (switch `ClassicMagic`), with
`spellbook.ClassicOverlandSlider` and `relations.ClassicSubversion`; dev flag
`-classic-magic=false` turns them all off. Tests `TestClassicCounters`, `TestClassicSpellBlast`,
`TestClassicMagicRolls`. The economy of spells (costs, research, casting, Time Stop's drain) is in
economy.md.

## Counters of a city

A spell aimed at a city, or at a stack in one, is checked after its target is picked; a ward no
longer keeps you from picking it. It is countered, with its mana spent, when any of these holds:

- Consecration is on the city and the spell is of death or chaos.
- The city has the ward of the spell's realm (each ward its own realm).
- The Nightshade of the city stops it: 100 for every Nightshade square the city works, when it has
  a Shrine, Temple, Parthenon, Cathedral, Sage's Guild, Oracle, Alchemists' Guild or Wizards' Guild,
  rolled out of 250 against the strength of the spell times the caster's retorts (Archmage, the
  realm's Mastery).

Curses, Call the Void, Corruption (a city on the square), Black Wind, Stasis, Ice Storm and Fire
Storm are checked; Earthquake, Raise Volcano and Warp Node are not.

## Spells

- Disenchant Area and True: strength the spell's cost with the extra (x3 for True, x2 Runemaster);
  every enchantment of the square that is not the caster's at its table cost and its owner's
  retorts; a unit with Spell Lock only loses the Spell Lock (at 150); the caster's own units in
  Stasis can be freed; a warped node can be mended, whoever warped it. Out of 250.
- The extra of a slider spell (Disenchant, Disjunction) is up to the spell's cost once more.
  Computer wizards cast them at 3 times cost and strength.
- Drain Power takes 10 rolls of 1 to 20.
- Cruel Unminding takes 1 to 10 percent of the skill, at least 1.
- Spell Binding of a computer wizard binds with the spell's own cost.
- Spell Blast: the Spell of Return is not lost, it starts over; a computer wizard pays the table
  cost less what is left of the spell, the human its cost less what is left.
- Stasis: no unit is freed on the turn after the cast; then each turn when 1 to 10 is at most its
  sorcery resistance less 5.
- Call the Void hits every unit in the city for 10, but those with Magic Immunity, Righteousness or
  Regeneration cast on them.
- Nature's Wrath spares fliers and non-corporeal units.
- Plane Shift moves the whole stack; a Planar Seal, any unit on the other side, or water it can not
  enter stops it.
- Warp Node only on another wizard's node that is not warped.
- Corruption not on a node or land already corrupted. Floating Island not onto a full stack. Word
  of Recall checks nothing.
- Raise Volcano breaks buildings only when the human takes part. Change Terrain turns any volcano
  into a mountain, with no mineral.
- Summon Hero and Summon Champion: up to 100 draws of the kind, no fame needed, the Priestess and
  Paladin need life books, the Black Knight and Necromancer death books; a champion not found gets
  one more draw among all heroes.
- Summons come to the summoning circle, else to the fortress (which takes the circle).
- Subversion: every computer wizard but the target takes the target's view of itself, and the
  peace ends.
- When a computer wizard starts the Spell of Mastery, you see it.

## The world's spells

- Armageddon and Great Wasting: 4 to 6 tries a turn for each caster, each up to 50 squares by
  chance, never within 2 squares of the caster's cities (Armageddon not on water, rivers, nodes or
  volcanoes; Great Wasting not on water or corruption). Wards and Consecration do not stop them.
- Meteor Storm, once a turn for each caster: every unit outside a city takes the storm, the
  caster's own too; buildings of other wizards' cities break at 1% unless the city counters it
  (as if the human cast it, the original's).
- While time stands still nothing of the world moves: no events, volcanoes stay, Armageddon, Great
  Wasting, Chaos Rift and Meteor Storm rest, no heroes, mercenaries or merchants come; the turn the
  Time Stop runs out is still stopped.
- No research while casting the Spell of Return.

## Cities

- Magic power of minerals per square in whole numbers (dwarves twice, Miners' Guild half again,
  shared squares half).
- The skill shown counts half the skill of each hero at the fortress.

## The last of the spells (2026-10-01)

User: "Let's update all spells to be classic". Read for this by three helpers (ReMoM OverSpel.c
Cast_Spell_Overland, Spells129.c Call_Forth_The_Force_Of_Nature and Apply_Automatic_Spell_Counters,
Terrain.c All_City_Nightshade_Count, City_ovr55.c All_City_Removed_Buildings, Spells132.c
Cast_Wall_Of_Stone, Spells137.c the Spell Blast of the human). Tests `TestClassicRemovedThisTurn`,
`TestClassicSpellBlastHumanPrice`.

- Nature's Wrath strikes after a chaos or death cast that went through: not when the human calls
  it off, a computer wizard finds no target or a city counters it (`game/natureswrath.go`: the
  strike is an event queued behind the cast). The human is told "Nature's Wrath strikes you" or
  "strikes <name>" every time.
- A city's Nightshade is counted once a turn, when all players have played, and kept
  (`City.NightshadeCount`, saved): only when the city has people and a building of religion or
  learning (one sold or destroyed in the turn counts); else the old count stays, never cleared. The
  counters read that count, not the city as it is now.
- A building sold or destroyed is remembered until the end of the turn (`City.RemovedThisTurn`,
  `City.RemoveBuilding`, every sale and destruction goes through it): Wall of Stone is refused on
  walls sold or destroyed in the turn, by the human and the computer. Wall of Stone leaves the
  city's build as it is.
- Spell Blast cast by the human costs the cost the spell was started at less what is left of it:
  for a spell of both books its progress, which the screen shows.
- Death Wish and Great Unsummoning: the units in a city meet its counters (Consecration, the ward,
  Nightshade), rolled once a city.
- Computer wizards never move a fortress, theirs or the human's (the user's rule, improvements.md;
  `ai.ComputersMoveFortress`). Test `TestChewbotNoMoveFortress`.
