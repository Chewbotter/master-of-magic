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
- Not ported yet: the terrain costs kept after a volcano rises or cools (the original's mistake).
  (Wind Mastery's count, Endurance on roads, the limits of 9, the drowning riders and the eviction
  were ported with the smaller rules.)

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
- Ported 2026-10-01 ("The last of the spells" in magic.md): Nature's Wrath after a cast that went
  through, the Nightshade count kept from the turn, Wall of Stone on walls sold or destroyed this
  turn, the human's Spell Blast price, the counters of cities for Death Wish and Great Unsummoning.
- Not ported: a computer wizard's Move Fortress moving the HUMAN's fortress (ReMoM's
  Cast_Move_Fortress has the human's index written in; not checked against the program, no
  community report of it, and AITP_Move_Fortress itself reads the caster's fortress: likely a
  mistake of the reconstruction; for the user to decide); what Death Wish and Great Unsummoning do
  (stubs in the reconstruction: as written they do nothing; the fork's effects stay); Nature's
  Wrath on a building another one needs (the reconstruction's Apply_Damage_To_City is garbled);
  the Stasis of the new units is not saved (a load lets them roll at once).

## The leftovers (item 9, last part)

- Kept, the original's: Arcane Power never for the Knight (`hero.quirkArcanePowerKnight`, meant for
  the Elven Archer); a Lucky roll gives Agility too (`hero.quirkLuckyFallsIntoAgility`); the plain
  inner desert gives no production (`city.quirkPlainDesertBarren`). Suggested: each as meant.
- DOUBTS FOR THE USER, not ported: production over 127 wrapping (the city's byte; with the
  computer's multiplier it would wrap from 32 at Impossible); the mana of the guilds (the
  reconstruction: Animists' +3, Wizards' +3, Alchemists' none; the manual and upstream:
  Alchemists' +3, Wizards' -3); Evil Presence taking no religious power (the reconstruction has no
  test, the manual says it stops the power); Super Sage 9 a level (kept 4.5). Each can be settled
  in the real game (cities.md and spec notes say how).
- MY CALL: a hero's Caster pick adds 2.5 (the fork's unit), the original's +1 of its own count.
- Not ported: movement costs frozen when the terrain changes (the original keeps the old costs, its
  mistake); the order of the turn: a computer wizard's spell lands at the end of the turn in the
  original, at its next turn here (a human turn later), events act before the incomes of the next
  turn here, a city taken by the human from a computer misses a turn of growth here; towers that
  see both planes; the fleeing defender's place (combat); the human's line in UNITS DISBANDED for a
  unit evicted from a full square; the running events not aging while time stands still.
- FOUND IN THE REVIEW AND FIXED: the computer's walk-in capture took the loot of a neutral town or
  a razing from the loser's gold with no floor (raiders and defeated wizards with gold below 0);
  the riders of a Floating Island drowned after every sea battle (it does not sail; upstream's
  check had the same hole); the seat rule hit the winner too and drowned heroes after the last seat;
  a refused Earth Gate left the waiting units as a second stack. MY CALL: a hero's worth for the
  dismissal of weak units is 100 + 10 for every point of fame it asks (the fork has no table cost
  for heroes).

## The review of item 9 (2026-09-30)

Four read-only reviewers went through the units, magic, battle and leftover commits; their reports
are in the session's scratchpad (review-*.md). FIXED: the gold floor of the walk-in capture; riders
of a Floating Island; sea losses on the winner and of heroes; heroes worth 0 when weak units are
dismissed; a refused Earth Gate; Wind Mastery in battles; an evicted unit put on the sea because a
unit it left walked the wind; the computer's melding spending several spirits an order; Armageddon
and Great Wasting 3 to 5 tries (4 to 6 now); Resurrection, Summon Hero and Incarnation with no
circle; the computer's Floating Island onto a full or foreign stack; Plane Shift into an empty enemy
city; Meteor Storm breaking the Fortress and the Summoning Circle; Disenchant always removing Lion
Heart and Invisibility (their names are spelled otherwise in the data); incomes on the turn a Time
Stop runs out; Holy Word reading the +50 of Death Immunity (undead never died); Entangle taking two
moves from walkers; a hero possessed by the side that lost dying (it goes back to its owner); webs
torn twice; Magic Immunity no longer shielding from Chaos Rift; the Demon Lord's Summon Demons.
LEFT, low: Metal Fires takes its to-hit off magic shots and thrown attacks too; thrown and breath
get no wall bonus; Raise Dead brings back the figures of a battle Lion Heart; Call Lightning picks
by its own tries, not the original's list; the battle AI keeps planning around walls a Disenchant
removed; the battle log calls a win in turn 50 "out of turns"; the human's Spell Blast price for
spells of both books; the computer's slider 3 times lost on save and load; `classicStasisNew` is not
pruned; a Time Stop that lapses costs 200 with `-classic-economy=false`; damaged units of an old
save whose table hits went down load with negative health (saves may break, project rule); some
rules of units and magic sit behind other switches than `-classic-units` and `-classic-magic`.

## Found by many runs (testing.md, 2026-09-30)

- THE STAND-IN (2026-10-01, the user: "Go ahead with 1, then 2, then 3"; 1 was: the rules that
  single out the human never ran in all-AI runs). `-sim-stand-in` (game/standin.go): the human's
  seat played by Chewbot, still the human for the rules. `Player.StandIn`, `IsAI()` true for it
  (an AI makes its choices; Spell Binding and Drain Power read `!IsHuman()` for the computer
  wizards' prices). FOUND AND FIXED by its first batches: casting a unit spell opened the window
  to pick a unit for the human (11 hangs in 36 runs); Great Unsummoning walked the stacks while
  removing units and crashed when a stack emptied (a bug for the real human too); battles of the
  human run on the battlefield without a screen need the army on auto (`combat.Run` waited for
  input forever; it now also ends a battle where no turn passes in 200000 updates, BATTLE STALLED in
  the log). The AI's stacks are all active before it plans (a path is searched for the active
  units). SEEN (36 games, 300 turns, the stand-in without bonuses and without diplomacy): its seat
  out of the game in 19, cities founded 2.5 (median); a settler sent behind a lair finds no path
  every turn (Chewbot's sites, item 3). OPEN: an empty stack once (seed 973199096, turn 184, Lo Pan,
  after orders only).

- FIXED (2026-10-01, round 5): a wind walker (the hero Aquilon) carried a stack of settlers and
  pikemen at sea; the orders sent it on alone and left the stack on the water. A stack that leaves
  the water with a ship or a wind walker takes the riders of its square along (`stackCarries`).
  And a computer player's stack moves with all its units active: the modes of a stack are worked
  out from its active units, while all its units move.
- FIXED (2026-10-01, the user: "I don't think water-based combat was common at all"): Triremes were
  the second build of every computer wizard (90 to 175 a game). Player_Colony_Autobuild_CP (AIBUILD.c)
  reads ai_transport_count, which the AI's data of the turn (AIDUDES.c) counts over ALL the wizard's
  units; our port counted the transports on the city's landmass, as the Grand Vizier's version
  (_HP) does, where a ship at sea never counts. Same seeds before and after: ships built 165 to 25
  a five-wizard game of 300 turns, 55 to 11 with two wizards; battles at sea 1.9 to 0.2 a game (they
  were rare before too: the ships were built, not used); founded cities the same, five-wizard
  armies 14 percent stronger at the end. The battle notes say "at sea" now.

- ROUND 4 (2026-10-01, `_build/a/fuzz8`, the settings of round 3 run for run, 90 runs): broken
  states in 2 runs (18 in round 3), attacks where nothing fought 0 (1,301), most units in a world
  951 (1,407), runs with a treaty 73 (31), no crash or hang. The 2 left are edge cases the original
  has too: (1) seed 833914234 (two wizards, 400 turns): Disjunction ended Crusade, a wounded unit's
  most hits fell below its damage and it stays on the map with 0 hits (Heal_Unit does not heal it,
  nothing kills it); (2) seed 83429930 (five wizards): Meteor Storm killed a ship of 3 hits and left
  its rider on the water (Kill_Unit drowns nobody). Open questions for the user: raiders (above),
  Triremes the second build (90 to 175 a game), 536 attacks at under half strength still (20 won).

- THE USER'S CALL (2026-10-01: "let's ignore the Nature's Awareness rule for AI diplomacy as it seems
  like strictly a downgrade"): computer wizards meet each other by sight (`ComputersMeetBySight`,
  contactclassic.go, flag `-computers-meet-by-sight=false` for the original). The original's
  NPC_To_NPC_Treaty_Negotiations needs contact, and computer wizards met only through Nature's
  Awareness, so in 59 all-AI test games that did not start at war there was not one treaty or war.
  The human is still met only by what the human sees.
- PORTED (2026-10-01): the limit of units (unitlimit.go): 1000 at most, none made for a computer
  player above 950 or for anyone above 980 (Create_Unit, City_Apply_Production; summons in battle
  and lair guardians are made anyway); a computer player's city picks again. Test games of five
  wizards had reached 1,407 units.
- FIXED (2026-10-01, found by replaying a traced run): a ship that left a square of water took
  along only the riders that were active; a settler left by the split of the orders was not, and
  stayed on the water (doAiMoveUnit looks at every unit now).
- LOOKED INTO, FAITHFUL (2026-10-01): a computer wizard can cast Lycanthropy on a swimming unit
  (Lizardmen) in a stack at sea; the Werewolves can not swim and stay on the water. The original's
  Select_Unit_For_Enchantment never looks at the ground either. Rare (1 of 24 runs).
- MEASURED (2026-10-01): the dismissals of AI_Kill_Lame_Units (about 200 units a game of five
  wizards) are the original's rule, not the user's changes: with drafting beyond need and
  expeditions by empire switched off, the same 12 seeds dismissed 207 a game against 193 with them.
  More than half are the garrison's weakest unit, one a city every 25 turns from turn 100.
- LOOKED INTO, FAITHFUL (2026-10-01): raiders and monsters attack without hope (0 of 976 won at under
  half the defender's strength in the final round of 2026-09-30). NPC_Destinations (AIDATA.c) sends
  every stack of the neutral player to the city of a wizard worth the most to it: 10, less the units
  in the city, less the distance, 5 more and 5 more for the human's at Hard and Impossible. It never
  weighs strength, and the battle then gives a computer wizard's defenders half again (Strategic
  Combat). In 6 games: all their battles were against cities, parties of 1 or 2 units mostly, none
  won below twice the defender's strength. Options for a change of our own, not made: a stack goes
  only for a city it could take (by the strength the quick resolution uses), or gathers into larger
  parties first, or roams near its town when nothing is in reach.

- FIXED (2026-10-01): a unit in battle had the base hit points and the spells of the battle only
  (`ArmyUnit.GetFullHitPoints`), not what it has on the map: its own enchantments (Lionheart +3,
  Black Channels +1), a hero's Constitution (`ClassicAbilities`) and Charm of Life. It fought with
  fewer hits than the original gives it, and a unit killed in battle kept the difference on the
  map and lived (a hero lost to a node 40 times in one run). Under `combat.ClassicRules` the battle
  starts from `Unit.GetFullHitPoints()`; and dead in battle is dead on the map (`TakeDamage`).
- FIXED (2026-10-01): a battle on a Shore or Lake square was fought on grass (`GetCombatLandscape`,
  an upstream FIXME), while those squares are water for moving and in the quick resolution: ships
  were left out of the battle and walkers stranded at sea fought; a lone ship then "lost" every
  attack with no unit lost and tried again each turn (1,301 lost attacks in the final round of
  2026-09-30, many repeated 10 times and more). Under `combat.ClassicRules` they are water battles.

- FIXED: the AI's hires of mercenaries and heroes did not keep 9 to a square (the clone's handlers
  have a FIXME; the human's hires did); the orders of a turn are applied after all are made, and a
  stack merged in between was split as it was, putting its units in two stacks (units then moved
  two or three steps a step, off the map: the crash of a stack at row -37); neutral towns ignored
  the original's distance of 4 to nodes, towers and lairs (my port of the world builder), so a town
  could stand on a lair and an army that won the lair stood in the town with its garrison; the
  fizzle window of Tranquility and Suppress Magic opened for the player of slot 0 even when it is a
  computer wizard; a run without a window skips every window that waits for a click.
- FIXED with `-sim-trace` (the step that first breaks each thing): the room of a square counted one
  stack of its owner, while a computer wizard's groups stand there in several stacks within a turn,
  merged at its end into one of 10 or 11 (`classicRoomAmong`, `stacksBeside` in movement.go); an AI
  attack fought one of the defender's stacks on a square (the clone's FIXME) and the winner stood
  beside the other: they are merged before the battle (doAiMoveUnit); a walker that boarded a ship
  stayed a stack of its own until the end of the turn, so the ship sailed off and left it on the
  water: it joins the ship's stack at once, and a ship that leaves a square of water takes the
  riders of its owner there along (the orders of a turn gave ship and riders different ways and
  split them); units that were not in a battle (a ship in a battle on land) stood on the square of
  the other side: an attacker's go back (also from a lair or node it lost at), a beaten defender's get away as fleeing ones do or are
  lost; `Player.RemoveUnit` took the unit from the first stack on its square, so a dead unit of a
  second stack stayed on the map: it takes it from its own; `ResolveStackAt` (9 a square after
  hires, units built, summons) looked at one stack: it merges the player's stacks on the square
  first. The merges within the AI's moves left the stack merged away with its units in the loop of
  moves, which then walked them off (unit away from its stack): the loop skips a stack that is no
  longer the player's; a fleeing defender's emptied stack is removed. All-AI runs go on when the
  wizard of the human's slot falls (`-sim-all-ai` sets `SimPlayOn`). OPEN after the final round
  (90 traced runs, 2026-09-30, `_build/a/fuzz3`): a walker on water in 17 runs, most of them a
  settler alone on a Shore square (cause not found; some were armies under Water Walking, which
  the check did not know then and now does); a dead unit on the map in 2 of 30 five-wizard runs
  (one after a lost attack, one with no battle in the journal); more than 9 on a square in 1 of 30
  two-wizard runs.
- Seen, the original's rules as ported: allied computer wizards turn an alliance into a pact at the
  next negotiation and back (`relations.quirkAllianceBecomesPact`); two wizards at war can go
  straight to an alliance; computer wizards never meet each other by sight (contact.md), so an
  all-AI game has little diplomacy; AI_Kill_Lame_Units dismisses many of the Spearmen and Swordsmen
  the cities build.
