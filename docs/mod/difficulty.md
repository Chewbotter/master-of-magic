# Difficulty

The original's bonuses of computer wizards by difficulty (ReMoM `MoX/src/MOM_DAT.c`
difficulty_modifiers_table, used in `MoM/src/CITYCALC.c`). Code `city/difficulty.go` (the table,
the cities), `player/difficulty.go` (who gets them, the upkeep of armies), `game/difficulty.go`
(nodes, the cap of the reserves). Switch `citylib.ClassicDifficulty`, dev flag
`-classic-difficulty=false`; `-capture-difficulty N` picks the difficulty of a quick start and of
`-sim` (0 intro to 5 impossible, Average when not given).

## The table (percent)

| level      | growth | outposts | production | gold | mana | research | food | maintenance |
|------------|--------|----------|------------|------|------|----------|------|-------------|
| intro      | 100    | 100      | 100        | 100  | 100  | 100      | 100  | 100         |
| easy       | 100    | 100      | 125        | 125  | 125  | 100      | 125  | 90          |
| normal     | 150    | 150      | 150        | 150  | 150  | 100      | 150  | 75          |
| hard       | 200    | 200      | 200        | 200  | 200  | 110      | 200  | 60          |
| impossible | 400    | 400      | 400        | 400  | 400  | 150      | 400  | 30          |

The fork has six levels: Average is normal, Extreme counts as impossible (as in the rules of
diplomacy).

## Where it applies

- Cities of computer wizards (not the human, not the neutral player): food, production, gold
  (before the upkeep of buildings), mana and research of the city times the table, the upkeep of its
  buildings times the maintenance (Do_City_Calculations).
- The power of a computer wizard's nodes times the mana, after the masteries (Player_Magic_Power_Income_Total).
- The gold, food and mana every player but the human pays for its units and all its enchantments
  (the neutral player's too) times the maintenance (Player_Resource_Income_Total).
- Growth of a computer wizard's city times the growth; a NEUTRAL city grows half as fast and not at
  all from (level + 1) x 2 citizens on (City_Growth_Rate).
- The chance an outpost grows times the outposts column, for EVERY owner, the human's outposts too
  (as the original has it).
- Every player's gold and mana are at most 30000 when a new turn starts (Next_Turn_Calc); the fork
  had no cap. Switch `game.CapReserves`.

## Our calls

- The table applies under both Enemy AI settings: it is a rule of the game, not of the AI.
- Only growth is scaled; a city that starves loses what it loses (the original keeps growth at 0 or
  more and shrinks cities elsewhere).

## Seen in runs

150 turns, 4 computer wizards, medium land, Chewbot (cities at the end per wizard): intro 2 to 3,
Average without the table 2 to 3, Average 3 to 7, Hard 6 to 9, Impossible 4 to 9 with gold at the
cap and mana in the thousands.
