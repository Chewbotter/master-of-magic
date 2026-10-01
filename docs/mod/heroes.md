# Heroes, mercenaries, merchants, items and fame

The original's rules. Read from the ReMoM reconstruction (`MoM/src/NEXTTURN.c` Determine_Offer,
Pick_Random_Hero; `CITYCALC.c` Generate_Mercenaries; `HIRE.c`; `Combat.c` End_Of_Combat; `ItemMake.c`).
Code: `game/heroesclassic.go` (switch `ClassicHeroes`), `hero/classic.go` (`ClassicAbilities`),
`artifact/classicitems.go` (`ClassicItems`); dev flag `-classic-heroes=false` turns all three off.
Test `TestClassicPickHero`.

## Offers each turn

None for a banished wizard.

- **Hero**: chance 3 + fame / 25 percent, twice with Famous, at most 10, computer wizards 10 more;
  then divided by (heroes + 1) / 2 + 1. Up to 10 draws among all 34 heroes (not Torin): it must be
  free, the Priestess and the Paladin need life books, the Black Knight and the Necromancer death
  books, and the fame it asks for must be BELOW the wizard's (`quirkHeroFameStrict`: with no fame
  no hero comes). The human who can not pay the full fee turns it away and the hero gains a level;
  Charismatic halves the fee after that check (`quirkCharismaticAfterCheck`). A computer wizard pays
  the fee (MY CALL; upstream: nothing).
- **Mercenaries**: from turn 30. Chance 1 + fame / 20, Famous twice, at most 10, computer wizards 10
  more. Level: regular (20 experience), elite (120) when a roll of 1 to 100 plus the fame is over 90;
  never veteran (`quirkMercenariesNoVeteran`, the original's). Up to 20 draws of any unit but
  creatures, settlers, ships, heroes, the Nomads' Magicians and the Draconians' engineers, of a plane
  the wizard has units or cities on (both planes when any tower is not its own, which is nearly
  always). 1 to 3 of them by a roll plus the fame (over 60: 2, over 90: 3). Cost: the unit's cost
  times (3 + level) / 2, times the number; the full cost against the gold, then half for
  Charismatic.
- **Merchant**: only the human. Chance 2 + fame / 25, Famous twice, at most 10. A random item (the
  original's, as a lair's), at 3 times its cost; gone when the gold does not reach it; half for
  Charismatic after that.
- The chances of a computer wizard's offers are figured on the HUMAN's fame (`quirkOffersOnHumanFame`,
  the original's).

## Fame of a battle

The winner +1 for 4 dead enemy units or more, and +1 once when a dead enemy unit costs 600 or more
(not for raiders). The loser -1 only when it is the human with more than 20 fame, and (level + 1) /
2 for each of its dead heroes. A town taken or razed costs its size: 1 to 5, a town 3.

## Heroes

- Might is for melee only.
- Super Blademaster: +6 to-hit at Demigod.
- A dismissed hero's items are destroyed.

## Items

- Enchant Item offers no power over 200, special powers too.
- Jewelry pays twice for its basic powers only.
- Destroying an item gives half its cost in mana, for every wizard.
- The items of dead heroes go to the winner of the battle (combat-rules.md).

## Added with the smaller rules

- The heroes' random abilities as the original picks them (`hero.classicExtraAbilities`): rolls of 1
  in 14 while picks are left, warrior picks for warriors, mage picks for mages; Torin is never
  Noble, a Magic Immune hero never Charmed, the Knight never gets Arcane Power
  (`quirkArcanePowerKnight`); Super Arcane Power only for heroes who shoot bolts; a Lucky roll also
  gives Agility (`quirkLuckyFallsIntoAgility`).
- Sage: 3 a level counted from 0, so a first level Sage gives nothing; Super Sage 4.5 a level from
  0 (the reconstruction's 9 is doubtful).
