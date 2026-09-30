# The original's worlds

How a new world is built when `maplib.ClassicMaps` is on (the default), after the original
(ReMoM `MoM/src/MAPGEN.c` Init_New_Game and the functions named below). Land, climate, nodes,
towers, lairs, homes and neutral cities are in the ORIGINAL WORLDS entry of the project notes
(`terrain/classic.go`, `maplib/classic.go`, `classiccities.go`); neutral towns' buildings and
garrison in `ai-neutral.md` (`classicneutral.go`). This page has the parts after the cities.

## Order

Land, climate, nodes, towers, lairs; homes, neutral cities (with their buildings and garrison);
then minerals and roads of both planes (`classicspecials.go`, called at the end of
`initializeGame`), as the original does them after all cities.

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

## Before

The fork put minerals on 6 (Arcanus) or 10 (Myrror) percent of the hills, forest, mountain, swamp
and desert of every landmass before the cities were placed (so a city could sit on one) and made no
roads but the one under each home.
