# Chewbot's spells of the world map

The original game's AI for overland spells (ReMoM `MoM/src/AISPELL.c` and the functions named in
the code), for the Enemy AI setting Chewbot. Switch `ai.ChewbotSpells`. Code: `ai/chewbotspells.go`
(core), `chewbotspellgroups.go` (which spell), `chewbotspelltargets.go` (at what),
`chewbotspellpower.go` (research, power), `chewbotspellworld.go` (values); the game side is
`game/aicast.go`. Dev: `-capture-spell-log`.

## The turn

When the wizard casts nothing, it picks a spell: first a kind, then a spell of that kind. A spell
costing more than 50 turns of (a tenth of its mana reserve plus its mana income) is not started.
The TARGET is picked when the casting is done, in the world of that turn; with no target the spell
is lost with all its mana (as in the original). Research is picked when there is none, and the
split of power every 16 to 25 turns.

## The kind of spell (AI_Select_Spell_Group)

Weights, one kind picked by chance (the original's weighted choice, which halves all weights while
their sum is 512 or more):
- nothing: 1
- summons: 100, 10 more for every free hero slot, +30 with a node on the summoning plane and no
  spirit there, x3/2 for Conjurer or Channeler, x5 when the summoning circle is not at the fortress
- unit spells: when it knows movement spells 0 before turn 50, 100 before 200, 200 after; buffs add
  (250 - turn) / 2 between turns 20 and 200, else 25
- city spells: walls and rituals 50 (5 when the human has not met it), city buffs and terrain 150
- disenchant: 40 for every curse on its cities, 300 for every Flying Fortress of the human, 100 for
  every warped node of its own, 5 for every enchanted unit of the human on its home land
- disjunction: for the globals of the others, by the realms of the one that has them (Suppress Magic
  2000, Tranquility against chaos 500, ...)
- attacks (curses, damage, spells on wizards): (the hostility weight of the target + the
  personality's war bonus) / 4, only when it knows such a spell; the target wizard of the turn is
  picked among the hostile ones by relation, war, and the human counts (difficulty + 1) times
- suppression (Nature's Wrath, Life Force, Tranquility, Evil Omens, Suppress Magic): 100 each it
  knows and has not cast, when the HUMAN has the realm it works against
- globals: 100 each it knows and has not cast (Awareness 500, Wind Mastery 75), above 400 it is 300;
  none of these two with less than 200 mana
- Spell of Mastery: 500 when known

## The spell of a kind

- summons: the cost / 10, squared (Storm Giant not), so the costliest known creature almost always;
  Floating Island when it has fewer transports than turn / 60; spirits with a node to meld; heroes
  and champions with a free slot
- unit spells: fixed weights (Wind Walking 50, Invisibility 40, Iron Skin, Flame Blade, Lionheart,
  Water Walking 30, ...)
- city spells: only those with a target now: Gaia's Blessing, Inspirations, Prosperity 300,
  Transmute 200, Flying Fortress, Dark Rituals, Spell Ward, Consecration 100, Change Terrain, Altar
  of Battle 50, walls and gates growing with the turn
- attacks: the cost divided by 2 (Spell Blast, Call the Void, Pestilence), 3 (Chaos Rift, Warp
  Node), 5 or 10; Time Stop with 3000 mana; curses of cities and terrain attacks only with Awareness
- globals: the cost / 10; suppression: the cost / 10 (it may cast one it has again)
- a realm suppressed (Evil Omens: life and nature, Life Force or Nature's Wrath: death, Nature's
  Wrath: chaos) weighs less

## The targets

- unit spells: in its strongest stack outside cities (else the one in its fortress), the unit with
  the highest upkeep that has not got it; spearmen and other units without upkeep, settlers,
  spirits and units with spells of their own are never picked
- city spells: its most valuable city without it (people, twice for some races, buildings, the
  fortress 2 per turn); Move Fortress to the city with a stronger garrison; Spell Ward against the
  human's realms; Consecration when the human has death or chaos books
- curses and Earthquake, Call the Void: the most valuable city of a hostile wizard it sees through
  Awareness, without Consecration or the ward
- Corruption, Raise Volcano: a square by chance near such a city
- Fire Storm, Ice Storm, Black Wind, Stasis: the most valuable enemy stack within sight of its
  cities or units (not of a treaty partner)
- spells on wizards: a hostile wizard that is casting, with no more in its spell than the AI's mana
  less 50
- Disenchant: a Flying Fortress of the human, its own warped nodes, its cursed cities, enchanted units
  of the human on its home land
- Disjunction, Spell Binding: the global of another wizard with the best score

## The power and research

- power: research 10, skill 10, mana 30, then 5 times 10 more by chance, slanted by the objective,
  Alchemy, the strategy, the reserve; before turn 30: 35 research, 25 skill, 40 mana
- research: among the 8 candidates by cost, far less for a group it knows a spell of, more for
  battle spells and summons, Disenchant, Disjunction, Spell Ward
- end of turn: globals a wizard of the wrong realm does not keep (Eternal Night without death, ...),
  all globals when its mana is 0

## Kept from the original, on purpose (quirk constants)

Disjunction's scores in a signed byte (it never aims at Suppress Magic and the like), Warp Node that
only takes warped nodes (so it never warps one), the first hostile wizard for spells on wizards,
suppression by the human's realms, Flying Fortress cast on a fortress that flies already, Evil
Presence on death wizards (where it does nothing), Storm Giant not squared, Incarnation only with
Barbarian Spearmen, Astral Gate never, Trolls' cities not doubled, Transmute's test of bits, Plane
Shift only for stacks with settlers, Enchant Road on the city's square.

## Our calls

- Where the reconstruction is wired wrong, the meaning is ported: Fire Storm and Ice Storm strike
  the square picked, Black Wind and Stasis a stack, the curses of cities use the enemy city scan,
  that scan wants cities seen (through Awareness), an enemy stack's value is the sum of its units,
  every node is a node, Disenchant's last step aims at the unit it found.
- Summoning Circle, Enchant Item, Create Artifact: their targets and items are not reconstructed;
  Chewbot does not cast them.
- The research cost is the spell's own, without the wizard's research bonus.
- Stacks of a treaty partner are no target.
- The tie of realms of equal books: nature, sorcery, chaos, life, death.

## Seen in runs

250 turns, 4 wizards: summons (the costliest they know), Floating Island, unit buffs (many without a
target: early armies are spearmen without upkeep), research. Their mana ends at 0: summons cost mana
every turn and the original's trimming of the army to the budget is not ported yet.
