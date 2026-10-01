# Testing with runs without a window

Games of computer players run headless as fast as they can (`-sim N`, game/magic/sim.go). On top of
the summary every run can keep a record, a journal of the AI's decisions, and the checks of every
turn; many runs at once go through `util/simbatch/simbatch.py`, which also writes the report.

## One run

    bash dev.sh a -sim 300 -sim-all-ai -capture-opponents 4 -capture-land-size 1 \
        -sim-json probe/r.json -sim-journal probe/j.jsonl -sim-out probe/s.txt

| flag | what |
|---|---|
| `-sim N` | play N turns, then write the summary (`-sim-out`) |
| `-sim-all-ai` | the human's slot is a computer wizard played by Chewbot, so every wizard plays; the run ends when one wizard is left, not when the wizard of the human's slot falls (`-sim-play-on` is set) |
| `-sim-stand-in` | the human's seat is played by Chewbot but stays the human for every rule: no difficulty bonuses, the computer wizards' rules that single out the human (defense on its land, raiders' and wars' preference for it, diplomacy toward it), the human's upkeep. Its turns are the computer players' turns; every window the game would open for the human is left out or answered by the AI (game/standin.go); its battles are fought out on the battlefield with Chewbot leading both armies, as the human on auto. The diplomacy addressed to the human is never answered. The run goes on when it falls |
| `-sim-stand-in-quick` | with `-sim-stand-in`: its battles by the quick resolution, as Strategic Combat Only |
| `-sim-play-on` | without `-sim-all-ai`: the run goes on when the idle human is beaten |
| `-sim-war` | every wizard at war with every other at the start |
| `-capture-opponents N` | computer wizards beside the human's slot (1 to 4: 2 to 5 wizards) |
| `-capture-land-size N`, `-capture-difficulty N` | the world (0 small to 2 large) and the difficulty (0 intro to 5 impossible) |
| `-sim-json FILE` | the record of the run as JSON: settings, every player at the start and the end, a timeline every `-sim-timeline` turns (10), battles, conquests, contacts, the broken states found, a panic, the winner, the counts of the journal |
| `-sim-journal FILE` | the journal: one JSON line for every key decision or happening: turn, player, kind, what, why |
| `-sim-stop-alone` | end the run when one wizard is left (always with `-sim-all-ai`) |
| `-sim-seed N` | the seed of all chance (`lib/chance`): a run of one seed plays the same game again, turn by turn, so a problem can be watched as often as needed; `-sim-repeat` adds the index to the seed |
| `-sim-seed-trace FILE` | with `-sim-seed`: the line of the game behind every draw (and the line that called it), so two runs that should be the same show where they part |
| `-sim-save-every N` | the game is saved every N turns into a folder beside the record (`run-NNN.saves`); the last two are kept, and the one before the first trace of each kind is kept for good and named in the record, so the case can be loaded (`-sim-load`) and played from just before it |
| `-sim-state-log FILE` | every city and stack of every player at every turn, one line each, so two runs show the first thing that differs |
| `-sim-trace` | the checks run after every step of the game's loop, and the first time each kind of broken state is seen the log gets a line `TRACE <kind> first seen` with the example and the 30 lines of the journal before it (slower; how the causes of the broken states were found) |

The journal's kinds:

- the AI: `build` (a city's production, with the reason: army building, Trade Goods roll, the
  weighted pick with objective, defenders and needs), `buy`, `target` (a stack goes for a city, lair,
  node or stack, with both strengths), `settle`, `research`, `spell start` (the kind of spell and its
  cost against the mana), `spell skipped` (too costly), `tax`, `disband` (gold and food, mana, a
  settler too many), `hire hero`, `hire mercenaries`;
- diplomacy: `treaty`, `peace`, `war`, `break treaty`, each with its reason (negotiation, need for
  war, superiority, threats to the human, units near a city, an ally's war) and the relation;
- the game: `battle` (where, against whom, the result, both strengths and the units left), `take
  city`, `raze city`, `found city`, `event`, `cast` (a spell that lands), `dismiss` (weak units of
  computer wizards), `conquest`.

The checks of every turn (game/magic/game/simcheck.go) count broken states by kind with the first
turn and examples: gold or mana below 0, empty stacks, stacks off the map, more than 9 units on a
square, two players on one square, a stack in another's city, a unit in two stacks or away from its
stack, a dead unit on the map, a walker on open water with no ship, two cities on one square, a city
on water or off the map, a city of the wrong banner, a defeated wizard that keeps cities or units, a
hired hero in no stack.

A panic of the AI's goroutine or of the game's loop ends the run and is recorded with its stack; a run
whose loop does not come back for 90 seconds is ended by a watchdog that writes the record so far and
the stacks of every goroutine into the log.

RUNS OF ONE SEED ARE THE SAME GAME (2026-10-01). The game draws all its chance from `lib/chance`,
imported under the name `rand` in place of Go's `math/rand/v2`, which can not be seeded; with
`-sim-seed` every draw comes from one generator. Go also ranges over a map in an order of its own
chance, so anything that ranges over a map and decides, or draws, by it, must go in one order:
`Player.CitiesInOrder()`, `RelationsInOrder()`, `HeroesInOrder()`, `GetCities()`, `chance.SortedKeys`
for a map of weights, and `lib/set` gives its values in the order they were put in. A NEW RANGE OVER
A MAP THAT DECIDES ANYTHING BREAKS THE SEED: check with two runs of one seed and `-sim-state-log`
(or `-sim-seed-trace`), as `util/simbatch/seedcheck.sh` does. Checked: three five-wizard games of 300
turns each played the same twice.

## Many runs and the report

    python util/simbatch/simbatch.py run --name two --out D:/x/fuzz --runs 16 --turns 300 \
        --opponents 1 --all-ai --difficulty 0,1,2,3,4,5 --land 0,1,2 --war 0.3 --parallel 8
    python util/simbatch/simbatch.py report D:/x/fuzz/two D:/x/fuzz/max --out D:/x/fuzz/report
    python util/simbatch/simbatch.py run --name trace --out D:/x/fuzz --runs 10 --turns 300 \
        --all-ai --extra=-sim-trace
    grep -h "TRACE" D:/x/fuzz/trace/*.log

`--stand-in` plays the human's seat by the stand-in (`--stand-in-quick` for quick battles); the
report then has a part on how the human's seat fared. `--exe` runs a given build in place of the
newest (two builds on the same seeds).

`run` picks every run's settings by chance from the lists (the fuzzing) and a seed of its own
(`-sim-seed`), copies the newest build of the lane into the batch's folder (`game.exe`, so building
while a batch plays changes nothing in it; `batch.json` names the build) and starts that copy as
dev.sh would, at most 12 at a time (the user's machine runs other work), with `-sim-trace` and a
save every 10 turns unless `--no-trace` (tracing costs well under one percent). It keeps for every
run its record, journal, summary, log and settings (`run-NNN.*`); a run over `--timeout` seconds is
killed.

    python util/simbatch/simbatch.py replay D:/x/fuzz/two/run-003 --extra=-sim-state-log=probe/s.txt
    python util/simbatch/simbatch.py regress D:/x/fuzz/report/cases.json --out D:/x/fuzz --name regress

`replay` plays one run again with the batch's build and the run's flags and seed: the same game, so
anything can be added to look at it (`--extra`; `--newest` for the newest build, which plays a
different game once the code has changed). `report` also writes `cases.json`, the runs where a broken
state was traced; `regress` plays them again on the newest build, each with its settings and seed,
and tells which kinds are gone and which are new. `--extra=...` passes flags to every run (the `=` form,
for flags that start with a dash). `report` writes `report.json`
and `report.md`: how the runs ended, panics and hangs, the broken states and the first trace of each
(turn, whose turn, the journal before it, the save before it), the computer wizards at the end, by
personality, the same split by the settings of the runs (war at the start, wizards, difficulty,
land), battles and lairs by the strength of the sides, diplomacy, the decisions of the
journal by kind with their commonest reasons, and the growth by turn.

## The page of a report

    python util/simbatch/report_html.py --round "Round 1=D:/x/fuzz/report/report.json" \
        --round "Round 2=D:/x/fuzz2/report/report.json" --notes notes.json --out report.html

One page with the findings of `notes.json` first (title, subtitle, summary, and lists fixed, open,
behavior, tools), the rounds compared by broken state, then a tab for every batch: how the runs
ended, the broken states with examples, the wizards at the end, battles and lairs by the strength of
the sides, the growth curves, the journal by kind with reasons, milestones (the first turn a wizard
founded a city, fought a lair, took a city, went to war...), diplomacy, every run and the slowest
turns of the AI.

## Speed

A profile of a late game: `-sim-load NAME -sim 15 -sim-profile FILE`, then `go tool pprof -top -cum
<exe> FILE`. Found that way (2026-09-30): the clone AI that Chewbot wraps worked out all its goals and
their path searches, whose decisions Chewbot drops; `Enemy2AI.SkipGoal`, set by `ChewbotAI.Update`,
leaves them out (`ai/chewbot.go`). The modes of a stack are worked out once a path search
(`keepStackModes`, `game/movement.go`). A late game of five wizards went from 7.5 to 1.3 seconds a
turn. Next in the profile: the gold of a city asks whether it is connected to the capital by road,
a path search of its own, every time.
