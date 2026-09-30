# The economy of mana and units

The original's upkeep, disbanding, power, casting, research and costs. Read from the ReMoM
reconstruction (`MoM/src/CITYCALC.c`, `NEXTTURN.c`, `Spells128.c`, `INITGAME.c`, `NewGame.c`).
Code: `player/economy.go` (upkeep, power, research list), `game/economy.go` (disbanding, casting,
Time Stop, the spells of a new player), `spellbook/spell.go` `classicSpellCost`,
`setup/classicspells.go`. Switches `player.ClassicEconomy`, `spellbook.ClassicSpellCost` (also the
arcane tiers), `setup.ClassicStartingSpells`; dev flag `-classic-economy=false` turns all three
off. Tests `TestManaUpkeepRetorts`, `TestPowerSplit`, `TestClassicSpellCost`,
`TestResearchCandidates`, `TestClassicUpkeep`, `TestSpellCost`.

## Upkeep

- Gold: every unit its gold, creatures none, heroes none with Noble, undead none; less the fame,
  not below 0. Computer players' units cost less at Hard and Impossible (difficulty.md).
- Food: 1 for every normal unit; heroes, creatures and undead eat nothing.
- Mana: a creature its upkeep (undead half again); **Conjurer** makes that three quarters,
  rounded up ((x * 3 + 3) / 4); then the enchantments on the unit. Heroes and normal units have no
  upkeep of their own. Plus the city enchantments the wizard cast, its curses on others' cities
  too, and its global enchantments. **Channeler** halves the whole sum once, rounded down.
- Food sold for gold: half the surplus of each city on its own, rounded down, less half the
  units that eat, never below 0. A city that starves does not cost gold. (The summary the
  computer players plan with: half the food total when it is above 0.)

## Not paying

Only the human loses anything. A computer player's gold and mana just stop at 0.

1. Food: when the cities with a surplus have less over their people than the units eat, normal
   units (not creatures, heroes or ships) desert, the newest first, until it is enough.
2. Gold: when the gold with the turn's income is less than the upkeep of the units, normal units
   that cost gold desert, newest first.
3. Mana: when the mana with the turn's income is less than all the upkeep, one of four groups by
   chance, then the next, until it is covered: unit enchantments, city enchantments, global
   enchantments, creatures (newest first; with Channeler each counts half).

"UNITS DISBANDED" lists them ("<unit> - deserted: lack of food." / "lack of gold." /
"- dispelled: lack of mana."). At the end of the turn the human is asked only about food: "Some
units do not have enough food and will die ..." when the food total is below 0. Gold and mana are
not asked.

## Power

Power is split by the magic screen's shares in whole percent: mana and skill rounded to the
nearest, research gets the rest. With no research share the rest goes to skill (to mana with no
skill share either). Mana Focusing adds a quarter of the mana, rounded down. Archmage: the skill
income half again, rounded down.

## Casting

Each turn the skill left from the last turn and half the spell points of each hero standing on
the fortress go into the spell being cast (as far as mana and the cost allow). When next turn's
skill then finishes the spell, it is borrowed: the spell is done now and next turn has that much
less skill. Heroes help only here, not in casting a spell at once. A computer wizard never casts at
once; its spells are finished at the start of its turn.

## Costs

The table cost; for nature and life under Evil Omens half again; five times overland for a spell
that is in both books; then less the reduction in whole percent: 10 for every book over 7 in the
realm, 15 for the realm's Mastery, 25 Conjurer on summoning, 50 Artificer on items, 25 Runemaster
on arcane. The amount taken off is rounded down, so the cost rounds up.

## Time Stop

200 mana a turn in place of all upkeep; when the mana goes below 0 it is 0 and the spell ends.

## Research

- The list has 8 places. Each empty place takes a spell by chance from the pool: in every realm,
  every spell that can be researched and is not on the list, of the lowest tier that still has
  one. The pool is made anew after each pick, so when a realm's lowest tier is all on the list, its
  next tier comes in.
- Arcane tiers: Magic Spirit, Dispel Magic, Summoning Circle; Disenchant Area, Recall Hero, Detect
  Magic, Enchant Item, Summon Hero; Awareness, Disjunction, Create Artifact, Summon Champion.
- The Spell of Mastery comes on the list only when nothing else is left to research.
- The list is sorted by the turns each spell takes at the research income, the bonus of a realm
  judged on its first spell (`quirkResearchBonusOfFirstSpell`: Conjurer's summoning bonus counts
  for a whole realm or not at all).
- How many spells of a realm can be researched, by books: 1: 3 commons, 1 uncommon; 2: 5, 2, 1;
  3: 6, 3, 2, 1; ... 8: 10, all, 7, 6; 9: 10, all, 9, 7; 10: all of every rarity.

## The Spell of Mastery

It is researched at its table research cost, less half the research cost of every spell the
wizard gains (by research, trade, treasure, conquest; not the starting ones), and cast at its
table casting cost (Runemaster takes 25% off). With 11 books in all five realms its research is
3000 less.

## Starting spells

With b books in a realm a wizard starts with the first b - 1 spells of the original's list of the
realm (11 books: all 10 commons, 2 uncommons, 1 rare); the human picks on the new game screen,
where the list fills the picks. A computer wizard also knows the summoning spell of its realm of
most books (Sprites, Nagas, Hell Hounds, Guardian Spirit, Ghouls). At Hard and above it gets a
summoning spell to research (nature Basilisk, sorcery Phantom Beast, chaos Chimeras, life
Unicorns, death Shadow Demons) when it can not research the guard spell (Basilisk, for life Path
Finding); it loses that spell from what it knows if it had it (`quirkHardGrantUnlearns`).
Everyone knows Magic Spirit and the Spell of Return; Artificers Enchant Item and Create Artifact.

## Seen

200 turns, 4 computer wizards, 2 runs each: with the original's rules they keep 37 to 103 units
at the end (nobody but the human disbands), with upstream's 27 to 80.
