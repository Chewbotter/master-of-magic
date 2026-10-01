# Movement on the world map

The original's movement. Read from the ReMoM reconstruction (`MoM/src/UnitMove.c`
Army_Movement_Modes, Init_MovePathMap; `MainScr.c` Update_MovePathMap, Eval_Move_Path__WIP,
Army_Moves2, Army_Boatriders; `NEXTTURN.c` Unit_Moves2; `MovePath.c`; `Explore.c`; `Terrain.c`
Turns_To_Build_Road; the table of TERRSTAT.LBX). Code: `game/movement.go` (costs, modes, room,
go-to, road turns), `units/classicmove.go` (moves of a unit), `player/stack.go`
(`GetRemainingMoves`), `player/player.go` (`UpdateFogVisibility`, `ScoutFogSquare`), `city/city.go`
(`GetSightRange`). Switch `units.ClassicMovement`, dev flag `-classic-movement=false`. Tests
`TestClassicStepCost`, `TestClassicRoom`, upstream's path tests (one follows the switch).

## Costs (moves a square)

| land | walking | forester | mountaineer | swimming |
|---|---|---|---|---|
| grassland | 1 | 1 | 3 | 1 |
| forest | 2 | 1 | 2 | 2 |
| hills | 3 | 3 | 1 | 3 |
| mountains | 4 | 4 | 1 | 4 |
| desert | 1 | 1 | 1 | 1 |
| swamp | 3 | 3 | 3 | 1 |
| volcano | 3 | 3 | 3 | 1 |
| tundra | 2 | 2 | 2 | 1 |
| river | 2 | 2 | 2 | 1 |
| sorcery node | 1 | 1 | 1 | 1 |
| nature node | 2 | 1 | 2 | 2 |
| chaos node | 4 | 4 | 1 | 4 |
| water | - | - | - | 1 |

- One forester or mountaineer in a stack gives the whole stack its column; with both (and
  swimming) the lower of them counts. A stack swims when all its units swim or fly and one swims.
  A mountaineer's stack pays 3 on grassland: the original's table (`quirkMountaineerGrass`).
- Pathfinding on any unit, or every unit non-corporeal: every land square half a move (water too
  when the stack swims).
- Roads (and every city square): half a move on Arcanus, nothing on Myrror and on enchanted
  roads. A flying stack pays 1 a square and nothing on any road (`quirkFliersFreeOnRoads`).
- Ships sail 1 a square.

## The stack

- A stack moves at the pace of its slowest unit, but the first ship or wind walker in it sets the
  pace for all.
- Moves of a unit: its own, Flight at least 3, the wings of Chaos Channels at least 2, Endurance 1
  more. Haste does nothing on the world map. The Magic Spirit has 1 move.
- A stack with any moves left may always make one more step, into a mountain too.
- At most 9 units on a square: a step onto a square of the player's own that would make more
  can not be made, and a path stops before it.
- Boarding: land units may enter the sea only onto the player's ship, and only while the ships'
  seats cover every rider there and coming (Trireme 2, Galley 5, Warship 3; units that fly or swim
  need none, none at all with a wind walker).

## Paths

- Steps to the 8 neighbours all cost the same; the map wraps east and west. The top and bottom rows
  can not be entered.
- The path finder sees the land under the fog as it is, at no extra cost.
- Enemy stacks, cities not the player's and lairs are only a path's end.
- A path the stack goes on along by itself from a turn before never attacks and never enters a
  lair: it stops before and ends.
- Refusing to enter a lair still costs the moves of that step.

## Seeing and exploring

- Only moving explores: each step the stack's largest sight (at least 2 for fliers). Also a new
  outpost 2 around it, a city taken 3, a finished Oracle 6, Nature's Eye 5.
- Units and cities that stand see the explored squares around them and explore nothing: a unit its
  sight (fliers at least 2, patrolling 1 more), a city 2, with walls 3, Nature's Eye 5, an Oracle 4
  over all that (the original's).

## Roads

A road takes the turns of its land over the builders on the square, at least 1: grassland 3,
desert and sorcery nodes 4, rivers, nature and chaos nodes 5, forest, hills, tundra 6, mountains,
swamp, volcano 8. Purifying takes 5 turns with one unit and 3 with two; melding a node with an
enemy's Guardian Spirit on it works one time in 4; cities are at least 4 squares apart (as upstream
had them).

## Added with the smaller rules

- Wind Mastery, for ships with seats: the owner's +1, every other wizard's -1; above 0 half again
  as fast, below 0 half.
- Plane travel and the Earth Gate take no more than 9 units onto a square.
- After a lost sea battle the seats of the ships left carry the riders that can not fly or swim,
  heroes take no seat while a ship is left; the others drown (with no ship every rider).
