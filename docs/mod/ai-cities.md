# Cities of Chewbot

What the cities of a computer player build and buy with the Enemy AI "Chewbot": the original game's
rules, ported. The facts were read from the ReMoM reconstruction (`MoM/src/AIBUILD.c`,
`MoM/src/NEXTTURN.c`, `MoM/src/INITGAME.c`, `MoM/src/NewGame.c`, `MoX/src/MOM_DAT.c`,
`MoX/src/random.c`); the code is ours, `game/magic/ai/chewbot.go`. Rule for the original's
mistakes: see `ai-combat.md` (the visible ones are kept, behind `quirk...` switches).

Chewbot wraps the clone's AI (`ChewbotAI.AIBehavior`): the clone still moves the armies, casts,
researches and deals; Chewbot drops the clone's production orders and gives its own.

## When a city picks (Player_All_Colony_Autobuild, City_Apply_Production)

Every turn of the computer player, a city picks anew when:
- it has finished what it built (a unit or a building);
- it builds nothing;
- it is on Housing or Trade Goods, one turn in 6.
Any other city may buy what it builds (below). Outposts do neither.

## The objective of a wizard (INITGAME.c, TBL_AI_Realm_OBJ)

Picked once at random by weights: by the realm of the wizard's most books

| realm | Pragmatist | Militarist | Theurgist | Perfectionist | Expansionist |
|---|---|---|---|---|---|
| nature | 0 | 2 | 2 | 4 | 2 |
| sorcery | 0 | 1 | 4 | 2 | 3 |
| chaos | 0 | 3 | 2 | 2 | 3 |
| life | 0 | 2 | 3 | 4 | 1 |
| death | 0 | 4 | 1 | 1 | 4 |

plus Warlord (Militarist 2, Expansionist 3), Chaos or Nature Mastery (Theurgist 1), Alchemy
(Perfectionist 2), Archmage (Theurgist 4), Myrran (Expansionist 4), Conjurer (Militarist 3).

## What a wizard's city picks (Player_Colony_Autobuild_CP)

1. Settlers and engineers one at a time: none is picked while the player has one on the city's
   continent or another city builds one.
2. Defenders: on the continent of the human's fortress (first 200 turns) the city wants units when
   a roll of 1 to 30 is above the number of units in it; elsewhere a roll of 1 to 15.
3. Away from the human's continent, first the buildings of an army, one more for every unit in the
   city: Barracks (1 unit), Builders' Hall (2), Shrine (3), Smithy (4), Granary (5).
4. Trade Goods, 40 in 100 and 10 more for every 8 cities, less 10 for every city on it (see the
   quirk below).
5. Otherwise a pick by weight among all it can build:
   - a building: the base weight of its category (projects 10, gold 15, religion 30, research 10,
     military 10, production 15, food 15, navy 10) plus the objective's bonus for it (Theurgist
     +10 religion and research, Militarist +10 military and +5 navy, Perfectionist +5 projects,
     gold, production, food, Expansionist +2 military +5 navy);
   - a unit: its cost in tens, squared;
   - Housing counts twice in a city of fewer than 5 citizens;
   - buildings count nothing while the city wants units;
   - settlers (only in a city of 3 or more) and engineers: when wanted, half again before turn 50,
     times (10 + the objective's builder bonus, Expansionist 10) / 10; else nothing;
   - ships: only when there are fewer than one per 40 turns and 8 at most, times what is missing;
   - soldiers: only when the city wants units, times (10 + the objective's bonus, Militarist 5,
     Expansionist 3) / 10.

## Neutral cities (Player_Colony_Autobuild_NP)

Barracks first; then units while a roll of 5 plus 1 to 15 is above the units in the city
(never settlers, engineers or ships), else buildings by their base weight (never the shipwrights'
and maritime guilds).

## Buying (AI_Player_City_Buy_Production)

A wizard's city buys what it builds when the gold is above a multiple of the price: 1 to 4 times
for a building, 0 to 3 times for settlers, 3 to 7 times for other units. The price is the fork's,
the one of the city screen (`City.BuyCost`); the original's is a multiple of the whole cost.

## The original's mistakes that are kept (visible)

- `quirkWeightHalving`: the pick by weight halves all weights when their sum reaches 512, again
  until it is below: next to an expensive unit the buildings drop to nothing.
- `quirkTradeGoodsCountsCity`: the count of cities on Trade Goods looks at the city that picks, once
  for every city: a city on Trade Goods finds all of them on it, any other finds none and so picks
  Trade Goods 40 times in 100 or more. That is why the original's cities sit on Trade Goods so
  much.

## Left out, and differences

- The original doubles Housing by the population of another city (the one after the last in the
  list): here the city's own.
- The objective is not saved: after loading a game it is picked again.
- A city that finished a unit or a building before a game was loaded is not known to have finished.
- The clone's AI still plans with its own goals; its orders to produce are dropped, which it does
  not know.

## Checking it

- `-capture-city-log` writes every pick and every buy to the log.
- `-capture-turns N` lets the human player skip N turns, so the computer players play: 
  `bash dev.sh a -start -capture-turns 60 -capture-city-log -capture probe/x.png -capture-frames 8000`.
- Tests: `TestChewbotWeightedChoice`, `TestChewbotRealmObjectives` (game/magic/ai).
