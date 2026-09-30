# The neutral player's turn

The original's turn of the neutral player (raiders, monsters, neutral towns), for the Enemy AI
setting Chewbot (ReMoM `MoM/src/AIDUDES.c` AI_Next_Turn, NPC_Farmers; `MoM/src/AIDATA.c`
NPC_Destinations, Build_NPC_Stacks, Make_Raiders, Make_Monsters, NPC_Excess_Garrison;
`MoM/src/AIMOVE.c` Adjacent_Free_Square, Make_Monster_List). Code `ai/chewbotneutral.go`, switch
`ai.ChewbotNeutral`, dev flags `-chewbot-neutral=false`, `-capture-neutral-log`. The start of the
neutral towns is `classicneutral.go` (MAPGEN.c Generate_Neutral_Cities).

## Before: the clone's neutral player

Made a unit from nothing in every town without one (and now and then more), put every citizen on
the farms, walked toward the nearest city it had explored, and made monsters on top of lairs. All
of that is gone under Chewbot.

## The turn, in the original's order

1. What its towns build: the original's rule for neutral towns (already Chewbot's, ai-cities.md).
2. Farmers (NPC_Farmers): enough for the least farmers of the town plus one food for every unit in
   it, 2 food a farmer (3 for Halflings or with an Animists' Guild), at most all its people. The
   original adds a number of farmers to a number of units, its slip, kept; the game still keeps at
   least the farmers that feed the town.
3. Where its stacks outside the towns go (NPC_Destinations), up to 80 stacks:
   - a stack with no target picks, among the first 30 cities of wizards on its plane (fliers and
     swimmers: on its landmass; walkers anywhere on the plane, the original's flag being the wrong
     way round, `quirkNeutralLandmassInverted`), the one worth the most: 10 less the units in it less its distance, 5 more
     for a city of the human at Hard and 5 more at Impossible;
   - a stack on its way that stands next to another city of a wizard turns to it when that one is
     worth more;
   - a stack with no city to go to is disbanded.
   Then every stack walks toward its target. It sees every city (the original has no fog for it).
4. Raiders (Make_Raiders): an accumulator grows by 1 to level + 1 each turn (level 0 Intro to 4
   Impossible); at 30 a neutral town with a wizard's city on its landmass, a free square next to it
   and a garrison sends garrison x level / 6 units (two thirds when a computer wizard's fortress is
   on the landmass, half on Myrror before turn 200, at least 1), each a copy of a unit of the
   garrison by chance (not ships or settlers), and a third as many units of the garrison go. A
   failed try adds 15 to the monsters' accumulator. Kept quirk: Barbarian Spearmen and Swordsmen
   never raid (`quirkNoBarbarianRaiders`).
5. Monsters (Make_Monsters): an accumulator grows by 1 to level + 1 each turn; at 50 less 5 a
   level, from turn 50 on, a lair, tower or node with guardians (the first not of life) on a
   landmass with a city of a wizard sends creatures of the realm of its first guardian (death for a
   guardian of a race) to a free square next to it: a budget of (1 to level + 1, twice) x turn / 5,
   halved on a landmass with a computer wizard's fortress and not the human's; creatures by chance
   that fit the budget left, until less than 25 is left or 9 are made.
6. Too large garrisons (NPC_Excess_Garrison): a town with more units than its people (2 more with a
   Granary, 2 more with a Farmers' Market) loses its cheapest unit, one a turn.
7. Units in Stasis that will not come out die (the glue's AI_Hopeless_Stasis, as before).

A free square (Adjacent_Free_Square): the first of the 8 around, row by row from the upper left,
that is land with no unit, lair, tower, node or city.

## The start of a neutral town (original worlds only)

Buildings by its people less 2, from the smallest up: Barracks (0), Smithy (1), Builders' Hall (2),
Armory (3), Granary (5), Stables (6), City Walls (8), Fighters' Guild (10), Armorers' Guild (11),
Shrine (12), without what its race can not build and what lacks its buildings before it. Garrison:
people / 4 of its best close combat unit and as many of its best unit with bows or magic (else the
close combat one again), "best" being the last of its race it could build. So a town under 4 people
starts empty, as in the original. Before, neutral towns started with nothing and the clone made
their units.

## Our calls

- Cities are looked at in the order of plane, row and column (the original goes by the number of
  the city, which the game does not keep).
- A landmass is compared on its plane: the original compares its numbers alone, which are never the
  same on both planes (its count goes on from Arcanus to Myrror), so this is the same.
- The accumulators and targets are not saved.
- The clone setting keeps the clone's neutral player.

## Seen in runs

200 turns, 4 computer wizards, medium, Average: raiders of 1 unit every 15 or so turns, 6 to 9
bands of monsters (1 to 3 creatures) from turn 50, both going for the nearest weak cities; the
human, who does nothing in these runs, lost its only city in both runs with the new start of the towns. Lost lair fights of
some wizards (up to 26 of 27) are seen with the clone's neutral player too (Chewbot's targets of
lairs, not looked into).
