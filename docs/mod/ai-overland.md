# Armies of Chewbot on the world map

Where the armies, settlers, engineers, spirits, priests and ships of a computer wizard go with the
Enemy AI "Chewbot": the original game's rules, ported. The facts were read from the ReMoM
reconstruction (`MoM/src/AIMOVE.c`, `MoM/src/AIDUDES.c`, `MoM/src/AIDATA.c`, `MoM/src/NEXTTURN.c`,
`MoM/src/CITYCALC.c`, `MoM/src/Combat.c`, `MoM/src/DIPLOMAC.c`, `MoX/src/special.c`); the code is
ours, `game/magic/ai/chewbotmove.go` and the files beside it. Rule for the original's mistakes:
see `ai-combat.md` (the visible ones are kept, behind `quirk...` switches).

The clone's AI still does what is not ported yet (spells on the world map, diplomacy, taxes); its
orders to move, settle, build roads, meld and cross planes are dropped. The neutral player's
raiders still move as the clone moves them.

## What the wizard knows

- **Landmasses**: land squares that touch, corners too. The coast squares of a landmass are its
  land squares next to the sea.
- **The evaluation map**: on every square the strength of the units of all others (a tenth of the
  original's unit strength), guardians of lairs, nodes and towers too; cities, lairs, nodes and
  towers are marked as sites; a unit of a wizard it is not hostile to is marked so. The original
  AI sees the whole map, fog or not; so does this one.
- **Hostility** to every other wizard, looked at every 15 to 24 turns from turn 100 on: at war 3,
  a pact, alliance or peace 0, any other wizard it has met 2 by a chance that grows the worse the
  relation is; the neutral player 3. A wizard that is attacked is hostile to the attacker at once
  (2, at war 3, an ally 0). Before turn 100 it is 0 to everyone: the wizard only goes for neutral
  cities, lairs, nodes and towers.

## What every landmass is to the wizard (every 25 to 45 turns)

- **Own**: its cities, and the units of all others cost less than a tenth of its own.
- **Contested**: its cities, and others have more there.
- **No own city**, and **left alone** (an ally has a city there, or there is no free square).
- **Waiting to leave**: an army of 8 there found nothing to attack; it gathers at the coast toward
  the war landmass.
- **No targets**: nothing to attack.

Every landmass has a **stage point**, where expeditions gather: the free square nearest to the
middle of the wizard's cities there, a coast square for a landmass without them; on a landmass of
its own, while there is a war landmass elsewhere, the coast square nearest to it.

The **war landmass** of a plane: picked by the cities of the first wizard it is at war with (10
each) and neutral cities (1 each), weighed by the distance of their coast from the middle of the
wizard's cities; with no such city, by the size of every landmass that is not its own. It is picked
again only when the old one holds none of those cities any more, or the army waits to leave.

## Every turn, landmass by landmass

1. The units are grouped by square. On a site (a city, node, lair, tower) they are a **garrison**,
   elsewhere a **roamer**; a roamer of one unit is **unknown**. Units on their way somewhere,
   building a road or purifying are busy and left out.
2. Settlers and engineers in a garrison step out to a free square next to it; so does a spirit
   that is not on a node.
3. The **survey**: the strongest fighting units an expedition may take, 9 at most: all of unknown
   stacks and idle roamers, the best beyond 5 of a garrison, of the fortress only before turn 100.
4. **Spirits** go to the nearest node that is not the wizard's, has nothing on it, and lies on a
   landmass of its own (or holds its garrison), and meld there.
5. **Settlers**: where a city may be built, a settler builds one when a roll of 1 to 5 is not above
   the objective's chance (Expansionist 4, Theurgist 3, Militarist and Pragmatist 2, Perfectionist
   1), before turn 100 always, after it one time in 5. Else it goes to a tower of the wizard's own
   to cross to the other plane, else to the best square of the landmass (maximum population x 10,
   production, gold, minerals, the coast 50, all over the distance), else to a ship for the
   landmass for settlers. So settlers wander from square to square until the roll lets them.
6. **Priests** of unknown stacks purify corrupted land within 2 squares of a site.
7. **Engineers** go to the nearest city of the landmass and build a road from next to it to the
   nearest other city there that has no road to it.
8. **Targets**: fortresses of wizards it is hostile to (3; 500, ten times that at war), cities of
   those wizards and neutral cities on the landmass ((citizens + buildings) x 10 + 100, ten times at
   war), hostile stacks off sites (their strength when the fortress is on the landmass, else a
   third), nodes (power x 10 + 50 when a hostile wizard holds it, + 25 with guardians), lairs (50),
   towers on the fortress's plane when the other side is worth it (150 nobody's, 10 a hostile
   wizard's). The human's places count a quarter stronger.
9. Every roamer with no busy unit takes the target of the highest value over 4 times the distance
   that it is stronger than three quarters of. A roamer that was attacking a stack goes on to the
   nearest hostile stack.
10. Units that fly or swim go to the stage point of the war landmass.
11. On a landmass of its own, roamers fill the stack at the stage point up to 9, and ask for ships
    next to it; ships next to them take them on.
12. **Expeditions**: while fewer than the size of an expedition (2, and 1 more every 30 turns, up
    to 9) wait at the stage point or are on their way, the surveyed units go there, when enough
    could come or the landmass is not held. With 9 there, one time in 20 the landmass is looked at
    again.
13. **Garrisons**: the fortress wants 9, a city 2 and a unit for 3 citizens (for 4 where the wizard
    feels safe or the race is dwarf, troll or draconian), a free node 8. Unknown stacks and
    roamers away from the stage point send units to the site with the smallest distance less what
    it wants, a city before any node.

Before all that: stacks on a tower of the fortress's plane (or in a city of its own with an Astral
Gate) cross to the other plane; ships in a city go out to sea; stacks on their way next to a city
of another player with fewer than 4 units in it stop there, and so may attack it.

## At sea

Ships alone go to the nearest place ships are asked to that has room. A ship with others on board
lands them on a free square next to it on the war landmass or on a landmass with no city of the
wizard or an ally (settlers: on the landmass for settlers, where no enemy is within 5), else sails
the settlers to that landmass or the army to the coast of the war landmass. Units at sea that fly
or swim go to the stage point of the war landmass.

## The original's mistakes that are kept (visible)

- `quirkSettleOpponentCheck`: the check for enemies near a settler finds the settler itself, so
  after turn 100 a settler builds one time in 5 only, enemies near or not.
- `quirkWarOnMastery`: a wizard casting the Spell of Mastery was meant to become the enemy; the
  original looks for a spell that is never cast overland.
- The garrison of a node wants 8 where the wizard feels safe and 4 where it does not (meant the other
  way round); a garrison sends at most as many units as it looked at, not as it sent.

## Left out, and differences

- Not yet: disbanding units when gold or food runs short, and the killing of settlers and
  engineers beyond one per landmass (turn glue, part 5). Personalities: without them hostility is
  never 4 (jihad).
- The game keeps no owner of a tower: it is the player whose units stand on it.
- Roads: the game builds roads only over land the wizard has seen, so a city a road can not reach
  that way is passed over.
- Orders are not saved: after loading a game every unit starts ready.
- Stacks that cross to the other plane get their other orders in the next turn.
- The original's plain mistakes that only break its own sums (distances read from a wrong
  variable, strengths of a wrong unit, a division by zero) are done as meant.

## Checking it

- `-capture-move-log` writes every order with the step of the original that gave it.
- `-capture-opponents N` gives a quick start N computer wizards.
- `-capture-turns N` lets the human skip N turns; its battles then end without a screen and random
  events are not shown:
  `bash dev.sh a -start -capture-opponents 4 -capture-turns 260 -capture-move-log -capture probe/x.png -capture-frames 33000`.
- Tests: `TestChewbotDistances`, `TestChewbotUnitStrength`, `TestChewbotEvaluationSquare`,
  `TestChewbotNearestFree`, `TestChewbotLandmassType` (game/magic/ai).
