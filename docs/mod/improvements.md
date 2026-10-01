# Suggested improvements

The port follows the original, its mistakes included, until the foundation is complete (user
2026-09-30: "I still want to do a faithful port (even mistakes) until we have a complete foundation
to start changing things"). What was first built differently from the original, then put back to
the original, is kept here as a suggestion for when the changes start. Each names its switch or
quirk constant where there is one: turning it is the change. New calls go here too.

## Worlds

- Caves, dungeons, keeps and monster lairs are meant to be guarded by death (2 in 5), chaos (2 in 5)
  or nature (1 in 5); the original's choice falls through to nature every time
  (`maplib.quirkLairsAllNature`). Players remember chaos and death creatures in these places, so
  this is worth checking against the game as played.
- The grid of minerals runs off the maps: past the right edge into the next row, past the bottom of
  Arcanus into Myrror's top rows (where it changes the land but the mineral is wiped), and those
  squares are not kept safe from cities, lairs and towers. Suggested: go around the world, stop at
  the bottom, keep every square's checks (`classicSpecials`).
- Rivers run under lairs, towers and cities, and those squares become river. Suggested: keep rivers
  off them (`classicRivers`).
- A one-square lake without a river becomes a lone desert. Suggested: keep the lake
  (`classicDryLakes`).
- The roads under Myrror's cities are plain; every road of Myrror moves as an enchanted one
  (`game.MyrrorRoadsEnchanted`). Suggested: enchanted roads under Myrror's cities, and only enchanted
  roads move as enchanted.

## Lairs and treasure

- A cave's or monster lair's book is sorcery, chaos, life, death or arcane, never nature (the
  original rolls 1 to 5 for 0 to 4), and an arcane book lands on the Alchemy retort
  (`game.quirkArcaneBookAlchemy`). Suggested: any of the five realms, no arcane.
- Every item of a hoard is a random item of 800 to 1700, whatever the hoard's value (the original
  swaps the arguments of Make_Item). Suggested: an item of the hoard's value (Make_Item as meant: a
  premade item of that value first, else a random one).
- A tower's book is always nature (`quirkTowerBooksNature`). Suggested: the tower's realm or any.
- A second spell in a hoard adds its rarity to the first (`quirkSpellRaritiesAdd`).

## Battles nobody watches

- When both sides fall in the same round, both lose every unit (the original reads a winner it never
  set; the value is not known). Suggested: the side with more hits left wins.
- No node helps its creatures in a quick battle (the original reads the node of the last battle
  fought on a screen). Suggested: the node of the square.
- The healing of the melee rounds comes from the ranged budget; nature books count twice, the
  second time with sorcery's numbers; city defenders get +3 defense without walls.

## The neutral player

- A neutral stack of walkers goes for any city of its plane, even across the sea, and a stack of
  fliers or swimmers only for cities on its landmass (`ai.quirkNeutralLandmassInverted`).
  Suggested: walkers bound to their landmass.
- Barbarian Spearmen and Swordsmen never raid (`quirkNoBarbarianRaiders`).

## Chewbot's spells

- Black Wind and Stasis never find a target and are lost with their mana
  (`quirkStackSpellsNoTarget`; unmarked in the reconstruction, may be its wiring). Suggested: the
  most valuable enemy stack in sight.
- An enemy stack's value is its last unit's strength (`quirkStackValueLastUnit`). Suggested: the sum.
- Raise Volcano counts only sorcery nodes as nodes (`quirkOnlySorceryNodes`). Suggested: every node.
- Stacks of treaty partners are targets of Fire Storm and the like. Suggested: leave them out.
- The research cost is lowered by the research bonus of the first nature spells, whatever the
  spell's realm. Suggested: the bonus for the spell's own realm.
- The second realm starts at nature (`quirkSecondaryFromNature`): a nature wizard has no second
  realm, a wizard of one realm gets nature as second. Suggested: the realm of the second most books.
- Not portable, the meaning is ported (kept): Fire Storm and Ice Storm take a unit number for a
  square; Disenchant's last step reads outside its list.

## Chewbot's turn

- A landmass's value is the last city's on it, whoever owns it (`quirkLandmassLastCity`).
  Suggested: the sum of the wizard's cities.
- The summoned units disbanded for mana are weighed by their own strength (kept: the original reads
  a strength by a unit number the fork does not have).

## Diplomacy

- Breaking a treaty raises the breaker's trust in its victim and lowers every other wizard's trust
  in the victim (`relations.quirkBreakTreatiesBackwards`). Suggested: the breaker pays: its victim
  trusts it less, every other wizard a little less.
- A warning and a broken treaty say the grievance's words (`quirkWarningWordsOfGrievance`).
  Suggested: the messages of the warning and the broken treaty (records 15 and 24 past).
- A threat's gold is given from nowhere (`quirkTributeFromNowhere`). Suggested: the wizard pays.
- The exchange of spells compares the worth of the wizard's spell as a byte
  (`quirkExchangeWorthByte`), so a costly spell may ask little. Suggested: the full worth.

## Difficulty and banishment

- The table applies under both Enemy AI settings (kept: a rule of the game).
- Only growth is scaled; a city that starves shrinks by the fork's rule (the original keeps growth
  at 0 or more and handles starving elsewhere; to be looked at with the game's rules).
- Extreme counts as Impossible (the original has no Extreme).
- Spell of Return of a computer wizard: its city with the strongest garrison above the old
  fortress's (MY CALL: above 0, the old fortress being another's or gone); with none the spell does
  nothing and the wizard starts it again (MY CALL: the original's spell fails).

## Conquest and the end of the game (conquest.md)

- MY CALL, the reconstruction doubts itself: its scene of a conquest returns at once unless the
  neutral player takes part, which would leave "X banishes Y" and the wizards the human conquered
  (the score's Banished Wizards) to raiders alone. Here the scene shows when the human is the
  winner or the loser, as upstream, and the wizards the human conquers are counted, as an older
  note of the reconstruction says. Worth checking against the game as played: whether the scene
  also shows when two computer wizards fight.
- MY CALL, the reconstruction's gap: nothing in it gives a wizard its fortress back after the
  Spell of Return; here the fortress is where the spell was cast (upstream's).
- A computer wizard that banishes the human gets no spells (kept, the original's mistake).
  Suggested: spells for every winner.
- A wizard casting the Spell of Return has its mana set to its gold, not added
  (`ai.quirkReturnReservesOverwrite`). Suggested: added.
- A computer wizard pays the Spell of Return with its skill, mana or not (kept). Suggested: as
  anyone else, or say so in the game.
- A defeated human keeps its cities (kept; the game ends at once, so it shows nowhere).
- Raiders that take a wizard's last city defeat it and get the fame and loot (kept).
- The capturer's summoning point is moved to the loser's fortress when the loser's summoning city
  is taken (the original's, likely a mistake; not ported: the fork's summoning circle is a
  building).
- Not ported yet: outposts always destroyed when taken; the text "Raiders destroy X's fortress";
  the original's screen of a computer wizard's start of the Spell of Return ("X begins casting the
  Spell Of Return"), and of the loss by another's Spell of Mastery (SPELLOSE.LBX); the human's
  spellbook, Recall and diplomacy closed while banished; no offers of heroes, mercenaries and items
  to a banished wizard beyond upstream's; events that skip banished wizards.
- The frame of the score's bar is drawn by eye (the original's remap blocks were not read).
- The Hall of Fame is kept in a file of ours, not the original's MAGIC.SET.

## Economy of mana and units (economy.md)

- MY CALL, not reconstructed: the order within the mana groups of a human who can not pay (unit
  enchantments from the newest unit, city and global enchantments in the order kept; each by its
  upkeep, halved with Channeler as the creatures are).
- MY CALL: Torin and all heroes have no mana upkeep of their own (the reconstruction's line about
  the Chosen is doubtful; its intent is that Torin has none).
- The research bonus of a realm is judged on its first spell (`quirkResearchBonusOfFirstSpell`):
  here the first of the realm among the spells the wizard can research or knows, the original's
  first of all. Suggested: the bonus of the spell itself.
- At Hard a computer wizard that knows the granted summoning spell loses it
  (`quirkHardGrantUnlearns`); the guard spell does not match the grant for most realms. Suggested:
  guard and grant the same, and never unlearn.
- The Spell of Mastery on the list is sorted by its present cost, the original by its table cost.
- Only food is asked at the end of a turn, by the food total with starving cities, while units
  desert only by the surplus of the cities with more than they need (the original's two sums).
  Suggested: ask about gold and mana too, and by the same sums.
- Not ported yet: the mana slider of overland spells (the original up to the table cost, the fork
  4 times), computer wizards' Disenchant and Disjunction at three times cost and strength,
  AI_Kill_Lame_Units (computer wizards dismiss their weakest units every 25 turns from turn 100),
  Time Stop stopping events and the world's effects for everyone, a first level Sage giving 0.

## Cities (cities.md)

- MY CALL: Gaia's Blessing pacifies 2 and Stream of Life ends rebels as the city enchantments (the
  manual's and upstream's). The reconstruction reads two slots of the owner's GLOBAL enchantments
  with the numbers of the city enchantments, which as written means Crusade +2 and Life Force: no
  rebels in any city; it doubts itself there. Worth checking against the game as played.
- MY CALL, the meaning: an outpost's minerals count 5 for iron and silver, 10 for the others (the
  original reads the wrong square, a square by the city's number).
- MY CALL: Dwarves' taxes are not doubled (the reconstruction doubles them, with no note and no
  support in the manual; their minerals are doubled).
- Not changed: the mana of guilds (the reconstruction gives the Animists' Guild +3 and the
  Alchemists' none, and the Wizards' Guild +3; upstream and the manual: Alchemists' +3, Wizards'
  -3); left as upstream until it is checked.
- Kept, the original's: a city exactly at its maximum never shrinks, even starving; growth over a
  whole person is lost; Great Wasting and Armageddon SET the pacification (`quirkWastingSetsPacification`);
  Consecration's corruption block of 4 by 4 (`quirkConsecrationBlock`). Suggested: all four the way
  the manual means them.
- Kept from upstream (the reconstruction has only stubs): Gaia's Blessing turning desert to grass,
  volcanoes to hills and clearing corruption each turn (its mistake of the square fixed), Chaos
  Rift's damage.
- Not ported: the random city of an event (the original's picks only cities whose race is content
  under the capital's; the reconstruction's version is doubtful), a replaced building coming back
  when its replacement is sold, production over 127 wrapping.

## Movement (movement.md)

- Kept, the original's: a stack with a mountaineer pays 3 on grassland (TERRSTAT; a later patch of
  the data fixed it) (`quirkMountaineerGrass`); fliers move free on every road, Arcanus' too
  (`quirkFliersFreeOnRoads`). Suggested: both as walkers.
- MY CALL: a wind walker makes its stack fly (upstream's; the reconstruction's test can never fire).
- MY CALL: every city square counts as a road (the original lays a road with every outpost).
- MY CALL: ships enter coastal cities as upstream lets them (the reconstruction's pathfinder has
  roads closed to ships and no rule that opens a city to them; the game as played lets ships dock).
- MY CALL: every rider counts against the seats, heroes too (the original counts heroes one way in
  one function and the other way in another).
- MY CALL: the poles are closed only on maps higher than 2 rows (tests use maps of one row).
- Not ported yet: Wind Mastery's count of wizards (the original multiplies by one and a half or
  halves by the balance of friend and foe), Endurance adding to road building, the Earth Gate and
  Plane Shift limits of 9, the riders who drown when the ships lose a sea battle, units evicted by a
  full square, the terrain costs kept after a volcano rises or cools (the original's mistake).

## Tactical combat (combat-rules.md)

- Kept, the original's: the second fear roll off the attacker's own figures
  (`quirkFearHitsOwnFigures`); to-block counted twice in melee (`quirkToBlockDoubleCount`); Blur's
  shrinking loop and its immunity test on the wrong unit (`quirkBlurLoop`); Dispel Evil on any enemy
  (`quirkDispelEvilAnyTarget`); no Runemaster double for combat dispels (`quirkNoRunemasterDispel`);
  Wrack squared and Wraith Form sparing it (`quirkWrackSquared`); thrown attacks past Weapon
  Immunity; a dispel of one's own curses that never fails. Suggested: each the way the manual means.
- MY CALL: every battle unit of a lost side that stayed out dies at the end whatever the reason it
  stayed out (upstream's rules for who stays out).
- Not ported yet: Wall of Stone in a city battle (the original can not cast it in battle).
- Smaller rules (the LOWs, combat-rules.md "Smaller rules"). Kept, the original's: Wraith Form
  spares Flame Strike, Holy Word, Death Spell and Call Lightning (`quirkWraithFormSpares`); a
  countered spell costs only a tenth of the range in mana (`quirkCounteredCheap`); the human pays
  the full cost of fixed cost spells in battle while the book shows less
  (`quirkHumanNoCombatDiscount`; the reconstruction calls it faithful, it reads like `>` for `>=`).
  Suggested: each as meant. MY CALL: a hasted shooter's second magic shot by its own magic (the
  original tests battle unit number 3); the end of a battle shown on the screen asks the model
  every frame: it is finished once (FIXED, the fork rolled the fleeing again every frame, so nearly
  all of them died, and regenerated the winners again).
  Not ported: a fleeing defender's place by the original's scan and its 9 to a square (the fork's
  random free square, now with the row above, stays); the touch of an item (Death -3 more,
  Stoning -1 more); Cracks Call on a square and walls; the target filters of the buffs; Disenchant
  on a warped node in battle; Call Lightning of nature and at the start of every turn; the thrown
  strength of Mithril and Adamantium; Slingers as missiles (the fork's data has them as rocks).

## Heroes, mercenaries, merchants, items (heroes.md)

- Kept, the original's: computer wizards' offers figured on the human's fame
  (`quirkOffersOnHumanFame`); no veteran mercenaries (`quirkMercenariesNoVeteran`); Charismatic
  halving after the check of the full price (`quirkCharismaticAfterCheck`); a hero's fame strictly
  below the wizard's (`quirkHeroFameStrict`, doubtful: with no fame no hero comes). Suggested: each
  as meant.
- MY CALL: a computer wizard pays for its hero (upstream hired them for nothing; the original's
  AI_Accept_Hero was not read).
- MY CALL: mercenaries come from both planes always (the fork keeps no owner of a tower; the
  original opens both planes when any tower is not the wizard's).
- Not ported yet: the heroes' starting picks and their mistakes (Lucky falling into Agility and the
  others), Sage's research, the order of the hero chance's cap (the fork's already matches), the
  first turn of mercenaries' uninitialized cost test.

## Random events and conquest details (events.md)

- MY CALL: the city of an event (Pick_Random_City) is one whose race is content under the race of
  the capital, by the meaning; the reconstruction's code of it was not clear.
- MY CALL: Depletion and New Minerals keep upstream's effects (both are stubs in the
  reconstruction).
- MY CALL: Piracy keeps upstream's 30% to 50%, rounded down to tens as the original; the
  reconstruction's own formula is doubtful.
- Kept, the original's: the Gift goes to the human only (a computer wizard's draw is lost).
  Suggested: a gift for computer wizards too.
- Kept, the original's: the first 5 turns of every session have no event (the delay is not saved).
- Not ported yet: the order of the steps of a turn; RANDOM EVENTS in the chancellor's scroll.

## Units (units.md)

- Kept, the original's: an evicted unit may not go into a city of its own owner; the weakest of
  equals is the newest. Suggested: let it into its own city.
- MY CALL: Stream of Life heals fully, as upstream (the original's per turn hook for it,
  CTY_StreamOfLife, is a stub in the reconstruction).
- MY CALL: an evicted unit never goes off the map (the original takes the edge row again at the top
  of the map); a unit lost for want of room is only logged (the original tells the human).
- Not ported: the healer's mark that slows the healing of units made after the healer to about
  1 in 66 (the original's, likely a mistake); Torin's table cost of 600 for a rare foe.
- Fixed, the fork's: the squares around a unit or city skipped the row above and could reach past
  the bottom of the map (`FindEscapePosition`, the city's squares: `dy` tested for `cy`).

## Overland magic (magic.md)

- Kept, the original's: Meteor Storm asks a city's counters as if the human cast it, with no
  strength, so any Nightshade stops it (`quirkMeteorStormHumanCounter`); Subversion's -25 is
  overwritten at once by the target's view (`relations.ClassicSubversion`; the fork's own reading
  is the other way round). Suggested: the caster's counters; the -25 kept.
- MY CALL: Spell Blast never pays below 0 (the original's price of a computer wizard can be below
  0 for spells of both books, which would give it mana).
- MY CALL: Plane Shift still refuses a lair on the other side (a lair holds no units in the
  original); Disenchant read by its checked combat twin (the overland one as written can not be
  right).
- Not ported: Nature's Wrath striking only after a cast that went through (here at the start of
  every chaos and death cast, as upstream); the Nightshade count of a turn before and never cleared;
  Wall of Stone refused on walls sold or destroyed this turn (the fork keeps no such state); a
  computer wizard's Move Fortress moving the HUMAN's fortress (the original's mistake); Death Wish
  and Great Unsummoning counters (stubs in the reconstruction); the Stasis of the new units is not
  saved (a load lets them roll at once).
