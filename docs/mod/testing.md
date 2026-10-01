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
| `-sim-all-ai` | the human's slot is a computer wizard played by Chewbot, so every wizard plays; the run ends when one wizard is left |
| `-sim-play-on` | without `-sim-all-ai`: the run goes on when the idle human is beaten |
| `-sim-war` | every wizard at war with every other at the start |
| `-capture-opponents N` | computer wizards beside the human's slot (1 to 4: 2 to 5 wizards) |
| `-capture-land-size N`, `-capture-difficulty N` | the world (0 small to 2 large) and the difficulty (0 intro to 5 impossible) |
| `-sim-json FILE` | the record of the run as JSON: settings, every player at the start and the end, a timeline every `-sim-timeline` turns (10), battles, conquests, contacts, the broken states found, a panic, the winner, the counts of the journal |
| `-sim-journal FILE` | the journal: one JSON line for every key decision or happening: turn, player, kind, what, why |
| `-sim-stop-alone` | end the run when one wizard is left (always with `-sim-all-ai`) |

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

The game's own chance can not be seeded (Go's `math/rand/v2`), so runs differ every time; the batch
runner seeds only its picks of settings.

## Many runs and the report

    python util/simbatch/simbatch.py run --name two --out D:/x/fuzz --runs 16 --turns 300 \
        --opponents 1 --all-ai --difficulty 0,1,2,3,4,5 --land 0,1,2 --war 0.3 --parallel 8
    python util/simbatch/simbatch.py report D:/x/fuzz/two D:/x/fuzz/max --out D:/x/fuzz/report

`run` picks every run's settings by chance from the lists (the fuzzing), starts the newest build of
the lane through dev.sh, a few at a time, and keeps for every run its record, journal, summary, log
and settings (`run-NNN.*`); a run over `--timeout` seconds is killed. `report` writes `report.json`
and `report.md`: how the runs ended, panics and hangs, the broken states, the computer wizards at the
end, by personality, battles and lairs by the strength of the sides, diplomacy, the decisions of the
journal by kind with their commonest reasons, and the growth by turn.

## The page of a report

    python util/simbatch/report_html.py --round "Round 1=D:/x/fuzz/report/report.json"         --round "Round 2=D:/x/fuzz2/report/report.json" --notes notes.json --out report.html

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
