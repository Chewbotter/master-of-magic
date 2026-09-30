# Chewbot's turn glue

What the original's computer wizard does around its orders every turn (ReMoM `MoM/src/AIDUDES.c`
AI_Next_Turn and the functions named below), for the Enemy AI setting Chewbot. Code
`ai/chewbotglue.go`, switch `ai.ChewbotTurnGlue`, dev flag `-chewbot-turn-glue=false`.

## Order of a turn

Hostility and magic strategy, spells (a finished one cast, a new one picked), magic power, the check
of its globals, then the glue below: reserves, tax and food, the budget, too many settlers; then the
cities and the orders of its units. Units of computer players in Stasis that cannot come out go too.

## Reserves (AI_Update_Gold_And_Mana_Reserves)

- A wizard without Alchemy with no mana at all takes a quarter of its gold as mana and keeps half of
  its gold.
- One time in 10: mana below a fifth of the gold (and more than 100 gold): a quarter of the gold
  becomes mana and half the gold is kept; gold below a quarter of the mana (and more than 100 mana):
  the other way round; else before turn 100 with more than 50 mana, a quarter of the mana becomes
  gold and half the mana is kept.
- With Alchemy the two are split 90 gold to 10 mana, from turn 150 on 30 gold to 70 mana.
- Casting Spell of Return: all gold becomes mana (half without Alchemy). (The original replaces the
  mana with it; here it is added.)
- At most 30000 each.

## Tax and food (AI_Update_Gold_Income_And_Food_Income)

- The tax goes up one step (half a gold a citizen) while the gold income is below 75 for every 50
  turns, down one step when the income is above 200, never above 2 gold.
- With too little food, a farmer more in its cities by chance until the food is made up (counting 2 a
  farmer), 50 tries.
- The clone's new turn still sets the farmers its way before this; its reset of the tax to 1 gold and
  its turning of mana into gold are undone.

## The budget (AI_Disband_To_Balance_Budget)

- While the gold or food income is below 0 (200 tries at most): the weakest normal unit on land goes
  (heroes too), making up 1 food and its gold upkeep times the difficulty's maintenance (100, 90,
  75, 60, 30 percent from intro to impossible, the table of `docs/mod/difficulty.md`), at least 1.
- While the mana income is below 0: the weakest summoned unit on land goes, making up its mana upkeep
  the same way.
- The weakness: a tenth of its strength, twice on a landmass where the wizard is not clearly ahead
  (AI_Landmass_Values_And_Strengths: its stacks less a tenth of its cities and nodes there less the
  others' stacks and lair guardians, above 0 less 25).

## Too many builders (AI_Kill_Excess_Settlers_And_Engineers)

One settler a landmass, the others go; after turn 200 one engineer a landmass too.

## Hopeless Stasis (AI_Hopeless_Stasis)

A unit of a computer player in Stasis with a resistance below 7 dies.

## As the original

- The extra settlers and engineers go after the orders are set and before they are carried out
  (the orders of the units that go are dropped).
- A landmass's value is set by every city on it, the wizard's or not (another's sets it to 0), so
  the last city counts (`quirkLandmassLastCity`; the order: plane, row, column, where the original
  goes by the city's number); then its nodes add power x (magic + 1) x 2, magic 0 weak, 1 normal,
  2 powerful. Lair guardians count by their cost in the original's table (casting cost for
  creatures; the production cost used first was 0 for them).

## Our calls

- The disbanding of summoned units uses each unit's own strength (the original reads a strength by
  the unit's kind number, a slip; the fork keeps no unit numbers).

## Seen in runs

250 turns, 4 wizards: with the glue their mana stays above 0 (4 to 14 at the end); they disband
about 30 units for gold and food and 65 summons for mana over the game. Without it their mana is 0.
