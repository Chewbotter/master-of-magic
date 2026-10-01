# The original's worlds

How a new world is built when `maplib.ClassicMaps` is on (the default), after the original
(ReMoM `MoM/src/MAPGEN.c` Init_New_Game and the functions named below). Land, climate, nodes,
towers, lairs, homes and neutral cities are in the ORIGINAL WORLDS entry of the project notes
(`terrain/classic.go`, `maplib/classic.go`, `classiccities.go`); neutral towns' buildings and
garrison in `ai-neutral.md` (`classicneutral.go`). This page has the parts after the cities.

## Order

Land, climate, nodes, towers, islands grown around lone nodes and towers, lairs; homes, neutral
cities (with their buildings and garrison); then minerals, roads and rivers of both planes
(`classicspecials.go`, called at the end of `initializeGame`), as the original does them after all
cities.

## Islands (Extend_Islands)

`terrain/classicislands.go`. A node or tower with the sea on all four sides (nodes land in the sea
now and then), two times in three, tries 2 to 9 of its 8 neighbors by chance (not a node or tower):
the one east and the one south become grassland, the one southwest too unless the squares south and
west of it are both sea. The original means every direction and works the neighbor out wrong, so
only these grow (`quirkIslandsGrowSouthEast`, kept).

## Lairs, towers and nodes: guardians and treasure (Generate_Lairs, Create_Lair)

`maplib/classiclairs.go`, wired in `MakeClassicMaps` (`maplib/classic.go`).
- Exactly 25 strong and then 32 weak lairs, each on a plane by chance (before: a coin per plane
  for each, so the number varied), each of the 7 kinds by chance.
- Budgets: a tower 700 to 1200 (one set of guardians and treasure, the same on both planes); a node
  power squared times 5 to 15 times the magic setting (weak half, normal one, powerful one and a
  half; power 5 to 10 on Arcanus, 10 to 20 on Myrror, the node's aura as its zone); a strong lair
  100 to 1500 (Myrror 200 to 2500), a weak one 10 to 100 (Myrror 10 to 200).
- The budget times (level + 1) / 4 (Intro a quarter to Impossible one and a quarter; Extreme counts
  as Impossible).
- Realm: a node its own; a tower death 2 in 6, chaos, life, nature, sorcery 1 in 6 each; temples and
  ruins death 3 in 4, life 1 in 4; caves, dungeons, keeps and monster lairs nature, always: the
  original means death 2 in 5, chaos 2 in 5, nature 1 in 5 and falls through to nature every time
  (`quirkLairsAllNature`, kept; the roll is still made). Before: the realms as meant (and it says
  the program does too; players see chaos and death in them).
- First guardians: up to 200 tries of a share of the budget (a quarter to all of it by chance), the
  costliest creature of the realm (not Floating Island) that costs less than the share; as many as
  the budget pays for, one fewer half the time, at most 8. Second guardians: the same with what is
  left, another kind, a share of 1 to (10 less the first count), up to 9 in all. A node or lair
  can end with none (it is then found empty, with its treasure).
- Treasure budget: the first guardians' cost and half the second's, the difficulty taken back out
  (not at Impossible), times 50 to 124 percent (Myrror 76 to 174), at least 50.
- Costs are the table's costs of the creatures with the original's units (units.md; Guardian Spirit 50), else the fork's casting costs.

## Treasure (Create_Lair's hoard, Lair_Generate_Treasure)

`game/treasureclassic.go`. The original rolls the hoard when the world is made; nobody sees it
before the place is taken, so it is rolled when it is taken, from the budget above.
- A tower first has a spell of rarity 1 to 4 (the budget changes by 100 less 50 times the rarity
  squared).
- While 50 or more is left, one of 15 by chance: gold (2 in 15) or mana (2 in 15), 10 to 200 in
  tens, at most what is left, for 200 each time; an item (5 in 15) of 400 to 3000 at most what is
  left, up to 3, from 300 left; a prisoner (1 in 15) for 1000, from 400 left; a spell (3 in 15) of
  rarity 1 to 4 for 50 times its square, the rarities of two spells adding up to 4 at most (kept
  quirk `quirkSpellRaritiesAdd`); a book or retort (2 in 15): from 1000 left one, from 2000 two
  (from 3000 two by chance too), for 3000. A book or retort takes the place of all else.
- Given: gold, mana, a hero for the prisoner, a spell of the rarity the wizard does not know (its
  realms or arcane), and each pick of the book or retort: 3 in 4 a book of the place's realm
  (nodes their own, dungeon, keep and ruins death, temples life, towers nature: kept quirk
  `quirkTowerBooksNature`; caves and monster lairs sorcery, chaos, life, death or arcane, never
  nature: the original rolls 1 to 5 where it means 0 to 4, kept), refused across life and death; an
  arcane book lands on the Alchemy retort (the original adds it past its list of books,
  `quirkArcaneBookAlchemy`); else a retort (with two picks one of Warlord, Channeler, Divine Power,
  Famous, Infernal Power can come). Nothing given: an item worth 1200, or 1000 mana when there are 3
  items.
- The items: the original calls Make_Item with the hoard's value where the "power" goes and no
  worth, so every item of a hoard is a random one of 800 to 1700 whatever its value
  (`artifact.MakeClassicRandomItem`, Create_Random_Item): a kind by chance, basic powers (attack,
  defense, to hit, movement, resistance, skill, spell save) by chance that fit the kind, up to 4, until
  its cost is over its worth. Names and the costs of powers are the game's data. Never a premade item
  or one with spell charges or abilities.

## Minerals (Generate_Terrain_Specials)

- One try in every cell of a grid, 4 squares on Arcanus (150 cells) and 3 on Myrror (280 cells),
  rows first: the square at the corner of the cell plus 1 to twice the cell size down and across.
- No try on a square with a mineral already, a lair, tower, node or city.
- Grassland always, forest one time in 2, becomes mountain (3 in 7), hills (3 in 7) or swamp
  (1 in 7) first. So many minerals sit on a lone hill or mountain in grassland, as in the original.
- Then: forest wild game, swamp nightshade, and:

| | Arcanus | Myrror |
|---|---|---|
| mountain | iron 4, silver 3, coal 5, gold 3, mithril 3 of 18 | iron, silver, coal 1, gold 2, mithril 3, adamantium 2 of 10 |
| hills | iron 6, silver 4, coal 3, gold 4, mithril 1 of 18 | iron, silver, coal 1, gold 4, mithril 2, adamantium 1 of 10 |
| desert | gems 6, quork 3 of 9 | gems 1, quork 3, crysx 1 of 5 |

- Kept quirk (`quirkSpecialsAtEdges`): the test meant to keep minerals off the edges never keeps
  any off.
- The grid runs off the maps as the original's does: past the right edge a square is the first of
  the next row, past the bottom of Arcanus one of Myrror's top rows (the land changes there, the
  mineral is wiped when Myrror's turn comes), past the bottom of Myrror nothing (the original reads
  beyond its maps). Such squares are not kept safe from cities, lairs and towers: the checks use the
  square as rolled.

## Roads (Generate_Roads, Path_Wrap)

- Every two NEUTRAL cities of a plane on one landmass within a Range of 10 (the larger distance
  and half the smaller) get a straight road (the original's line with a fraction starting at one
  half), unless a square between them is water. The Range is measured without going around the
  world (the original's slip, kept), the line goes the short way.
- Every city has a plain road on its square, on Myrror too (the original writes the flag of the
  enchanted road beside the square by a slip; the stray flag is left out). The roads between towns
  on Myrror are enchanted. Every road of Myrror moves as an enchanted one, whatever its flag
  (Movement_Mode_Cost_Maps, `game.MyrrorRoadsEnchanted`).
- Wizards' homes get no road to anything.

## Rivers (Init_New_Game, Generate_River)

`terrain/classicislands.go` `ClassicRiver`, `classicRivers` in `classicspecials.go`.
- 10 rivers a plane, Arcanus and Myrror in turn, each tried up to 2000 times.
- A try starts 5 to 56 across and 5 to 36 down, with no sea in the 3 by 3 around it, not on
  mountain, hills, a node, a river or a mineral (desert is allowed at the start).
- It picks a way (south, west, north, east); every step goes that way half the time, else a way by
  chance that is not back. A step onto mountain, hills, desert, a node or a mineral is lost and the
  river stays.
- It ends on a square next to the sea (or on the sea's edge) or on another river, which it joins.
  More than 30 steps, more than 28 squares or fewer than 4: no river.
- The original then refuses a mouth on a shore that has no picture of a river mouth; here a river
  is refused when its squares or their neighbors find no picture (the fork's own check). The
  pictures are the fork's first fitting ones (the original picks among up to 4).
- A river may run under a lair, tower or city: the square becomes river under it, as in the
  original. A one-square lake that no river runs into becomes a lone desert (River_Autotile,
  `classicDryLakes`). Not ported: the original cuts extra rivers into a lake or a shore to
  grassland; here the picture check refuses such a river.
- Seen: 3 to 9 of the 10 made a plane, 9 to 47 river squares. A river may touch itself and make
  a small loop, as the original's may.

## Pictures (Shuffle_Terrains)

`classicShuffle` in `classicspecials.go`, last of all: the plain pictures of grassland (4 looks),
forest (3), desert (4) and swamp (3) become one of their look-alikes by chance; plain tundra stays
two times in three and takes the second look one time in three, never the third (the original's
switch misses it; kept). Before, desert, swamp and tundra always showed their first look. The
autotiling itself (which piece of shore, river, mountain range) stays the fork's, with its first
fitting picture where the original picks among up to 4 river pieces.

## Before

The fork put minerals on 6 (Arcanus) or 10 (Myrror) percent of the hills, forest, mountain, swamp
and desert of every landmass before the cities were placed (so a city could sit on one) and made no
roads but the one under each home. Its treasure was rolled from the guardians' budget by rules of its own (books and retorts beside
other loot, items as premade ones that fit). Its lairs had budgets of its own (up to about 3300), realms mixed for
every kind, up to 9 of one guardian, nodes always guarded, and towers different on each plane. Its rivers came one per tenth of the land of a landmass
with a walk in any direction, before the cities, and could run over mountains and nodes.
