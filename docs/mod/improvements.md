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
