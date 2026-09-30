# Conquest, banishment and the end of the game

The original's rules for a wizard that loses its fortress or its last city, the Spell of Return,
the win and the loss, the score and the Hall of Fame. Read from the ReMoM reconstruction
(`MoM/src/City_ovr55.c` Change_City_Ownership; `CONQUEST.c` Resolve_Wizard_Conquest,
Banish_Wizard, Conquest_Spells, CP_Is_Dead, Lose_Animation; `SCORE.c`; `SPLMASTR.c`;
`NEXTTURN.c` All_Players_Apply_Spell_Casting). Code: `game/conquest.go` (the rules, the end),
`halloffame/score.go` (score and Hall of Fame), `mastery/lose.go` (the lose animation). Switch
`game.ClassicConquest`. Tests `TestConquest`, `TestConquestWeakWizard`, `TestScore`, `TestInsert`.

## When a wizard is conquered

A city of a wizard is taken or razed, not by its own side. Its cities are counted before the city
changes hands, up to 2; the city taken counts, outposts count. When the city was its fortress, or
it was its last city, the wizard is conquered:

- 2 or more cities: **banished**.
- its last city: **defeated**.

The neutral player is never conquered. Raiders that take a wizard's city conquer it like a
wizard does; they get the fame and the loot and no spells.

## Banished (Banish_Wizard)

1. The wizard starts the Spell of Return at once; what it was casting is lost.
2. A computer wizard whose casting skill, with half the skill of its heroes on the square, is
   under 40, or that has no mana, can not come back: it is **defeated** instead.
3. Its fortress is gone. The winner gets 5 fame; the loser's mana is halved and the winner gets
   that half. No gold changes hands.
4. The scene of the banishment (when you are the winner or the loser).
5. Every other wizard likes the winner 20 less.
6. You, banished, are asked: "There are at least N turns remaining until you may return. Do you
   wish to resign?" (N: what the spell still costs over your casting skill). Yes: you are
   defeated. No: you play on, banished.
7. A computer wizard that was banished gives the winner up to 2 of its spells (below). Not when
   the human was banished: the original's mistake, kept.

While casting the Spell of Return a banished wizard gets no power (no mana income, no research,
no skill from power). A computer wizard pays the spell with its casting skill every turn, with
mana or without, and its mana goes down by as much, not below 0. A computer wizard that casts it
also has its mana set to its gold (half without Alchemy) and its gold to 0 every turn
(`ai.quirkReturnReservesOverwrite`: the original sets where it means adds). When the spell is cast,
the wizard picks a city and its fortress is there again.

## Defeated (Resolve_Wizard_Conquest)

1. The winner gets 5 fame (so 10 in all when a wizard resigns), half the loser's gold and half
   its mana, up to 30000. The loser keeps its own.
2. The loser's city enchantments end everywhere: its own and its curses on other cities.
3. In each of its cities: its fantastic units and heroes are dismissed, its other units become the
   neutral player's, and the city too; a defeated human keeps its cities (the original's, which it
   doubts; the game ends anyway).
4. Its nodes are free (no owner, no warp, no guardian).
5. All its other units are dismissed. A dismissed hero's items are gone.
6. All its global enchantments end.
7. The scene of the defeat (when you are the winner or the loser, and you did not resign).
8. Treaties with it end; every other wizard likes the winner 40 less.
9. The winner gets up to 2 of its spells.
10. You defeated: the lose animation, the score, the Hall of Fame, the start screen. A computer
    wizard defeated and no computer wizard left with a fortress or a city: the win animation, the
    score, the Hall of Fame, the start screen.

A defeated wizard plays no more turns.

## The spells from the ruins (Conquest_Spells)

Up to 2 spells, by chance, that the loser knew and the winner does not, of the realms the winner
has books of (arcane always). The human winner is told: "You found a X spell in the ruins."

## The Spell of Mastery

The wizard who casts it wins. You: the vortex, the win screen, the score. A computer wizard: the
lose animation and the score (the original first shows its own screen of the loss, SPELLOSE.LBX,
which is not made).

## The score

All of the human player:

| part | points |
|---|---|
| spells known | 1 each |
| people of its cities (thousands) | half, rounded down |
| wizards it banished or defeated, each once | 50 each; a win by the Spell of Mastery counts every wizard whose fortress still stands |
| fame (with Just Cause) | twice |
| time | 2000 - 2 a turn, only while its fortress stands and nobody else cast the Spell of Mastery; no floor at 0 |
| its own Spell of Mastery | 250 |

The sum is halved at Intro, three quarters at Easy, as it is at Average, doubled at Hard, three
times at Impossible (Extreme counts as Impossible).

## The Hall of Fame

10 places of name, score and race of the capital. A score takes the first place with a lower one,
the rest move down, the last is dropped. Kept in `hall-of-fame.json` beside the settings (runs of
Claude's own: `hall-of-fame-dev.json`); the original's MAGIC.SET is not read. Shown after the score
and from the start screen. The percent is the score out of 8000.

## Dev

`-capture-screen score`, `halloffame`, `lose` (`-capture-lose-stage N` for a later stage of the
fall), `resign`. `-sim-play-on` keeps a headless run going after the human is out; the `-sim`
summary lists every banishment and defeat and how the game ended.
