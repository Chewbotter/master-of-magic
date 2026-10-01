# Units

The original's rules. Read from the ReMoM reconstruction (`MoM/src/UNITTYPE.c` the table of units,
checked against the program of the Steam release; `NEXTTURN.c` Create_Unit, Heal_All_Units,
Heal_Unit; `COMBINIT.c` Unit_Hit_Points; `CITYCALC.c` Calc_Unit_Level, Evict_Weakest_Unit;
`SETTLE.c` Evict_Unit, Army_Do_Settle; `UNITSTK.c` Next_Turn_Process_Purify, Army_Do_Meld;
`ProdScr.c` Calculate_Product_Array; `Outpost.c` Map_Square_Survey). Code: `units/classicunits.go`
(switch `units.ClassicUnits`), `game/unitsclassic.go`; dev flag `-classic-units=false`. Tests
`TestClassicUnitData`, `TestClassicPurify`, `TestClassicEvictWeakest`, `TestClassicHealAmount`,
`TestClassicMeldOwnNode`.

## The table

The original's numbers where the fork's differed: Gorgons 4 figures, Lizardman Swordsmen 2 hits,
Sprites melee 2, the Demon defense 5, Greyfairer 5 hits, High Elf Settlers melee 0, Klackon Settlers
10 hits.

The worth of a creature is the table's cost, which is not always the spell's: Chaos Spawn 400,
Werewolves 250, Guardian Spirit 50, Demon Lord 900, Nagas 120, Gorgons 599, Fire Elemental 100,
Earth Elemental 160, Air Elemental 170, Phantom Beast 225, Phantom Warriors 20, Demon 80, Zombies
30. It picks the guardians of lairs, nodes and towers (so Demons, Zombies, elementals and phantoms
can guard) and the fame of a rare foe (600 or more: Gorgons do not count).

## Units made

- A unit built in a city with no better weapon gets magic weapons when its owner has Alchemy; so do
  the starting units.
- War College gives 60 experience.
- A city offers at most 12 units, in the table's order (the ships and the catapult first); when it
  has 12, Spearmen and Swordsmen are left out of the list.

## Hits and levels

- A hero gets 1 hit a level and none of the normal units' bonus.
- Charm of Life adds a quarter of the hits, at least 1 (Constitution counts before it).
- Crusade raises the level of normal units that are not undead.

## Healing on the world map

Every hurt unit heals a part of all its hits each turn: 1 in 20, 1 in 10 in its owner's city, 1 in
6 with an Animists' Guild; what is left over is a chance of one more (no least of 1: a unit of 10
hits in the field heals 1 point half of the time). A healer in the stack heals every unit a fifth
more. Undead and death creatures never heal. Stream of Life and Herb Mastery heal fully (kept).

## A full square

When more than 9 units stand on a square, the weakest (gold and mana upkeep and level, the newest
of equals) leaves for the first square around it, row by row from the upper left, that has no lair
or tower, no city of its owner, no units of another and room, and that it can stand on. With none,
or for a neutral unit, it is lost.

## Work

- Purifying: every priest counts its turns from the order; the square is clean when the counts of
  the priests there reach 4 together: one priest 5 turns, two or three 3, four or more 2.
- Melding: the first Guardian Spirit of the stack melds, else the last spirit. A wizard's own node
  takes only a Guardian Spirit, and only when none guards it. A warped node takes none.
- Settling: on corruption too, never on a tower; an undead settler can not settle.
- The road screen counts turns as the roads are built (movement.md).
