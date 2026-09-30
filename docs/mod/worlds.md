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
- MY CALL: a square past the right edge goes around the world (the original runs into the next
  row) and one below the map is none (the original runs into the other plane's rows).

## Roads (Generate_Roads, Path_Wrap)

- Every two NEUTRAL cities of a plane on one landmass within a Range of 10 (the larger distance
  and half the smaller) get a straight road (the original's line with a fraction starting at one
  half), unless a square between them is water. The Range is measured without going around the
  world (the original's slip, kept), the line goes the short way.
- Every city has a road on its square. Roads of Myrror are enchanted. MY CALL: city squares of
  Myrror enchanted too (the original writes that flag beside the square by a slip, but every road of
  Myrror moves as an enchanted one there).
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
- MY CALL: no river on a lair, tower or city square (the original lets the square become river
  under them). Not ported: the original turns every one-square lake without a river into desert
  and cuts extra rivers into a lake or shore to grassland; the fork keeps its lakes and refuses
  such rivers.
- Seen: 3 to 9 of the 10 made a plane, 9 to 47 river squares. A river may touch itself and make
  a small loop, as the original's may.

## Before

The fork put minerals on 6 (Arcanus) or 10 (Myrror) percent of the hills, forest, mountain, swamp
and desert of every landmass before the cities were placed (so a city could sit on one) and made no
roads but the one under each home. Its rivers came one per tenth of the land of a landmass
with a walk in any direction, before the cities, and could run over mountains and nodes.
