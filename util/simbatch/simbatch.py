#!/usr/bin/env python3
"""Development: many runs of the game without a window, and their report.

    python util/simbatch/simbatch.py run --name two --runs 12 --turns 300 --opponents 1 --all-ai --out D:/x/fuzz
    python util/simbatch/simbatch.py report D:/x/fuzz/two D:/x/fuzz/max --out D:/x/fuzz/report

`run` starts the newest build of a lane (through dev.sh) once per run, a few at a time, with settings
picked by chance from the lists given (the fuzzing: difficulty, land size, war, the number of
computer wizards), and keeps for every run its record (-sim-json), its journal of decisions
(-sim-journal), its summary and its log. A run that does not end in --timeout seconds is killed and
counted as hung.

`report` reads the records and journals of one or more batches and writes report.json (everything
counted) and report.md (the same in words): how the runs ended, the broken states the checks of
every turn found, panics and hangs, the wizards at the end, battles by the strength of the sides,
the decisions of the AI by kind and reason, wars and treaties, events, and the curves of growth.

The game's runner and its flags: game/magic/sim.go, game/magic/simrecord.go; docs/mod/testing.md.
"""

import argparse
import collections
import json
import math
import os
import random
import re
import statistics
import subprocess
import sys
import time
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path


def find_root(start):
    """the folder of dev.sh above the script (the lanes' root)"""
    for folder in [Path(start).resolve()] + list(Path(start).resolve().parents):
        if (folder / "dev.sh").exists():
            return folder
    return None


def parse_list(text, kind=int):
    return [kind(part) for part in str(text).split(",") if part.strip() != ""]


# ---------------------------------------------------------------- run

def one_run(args, root, out, index, seed):
    rng = random.Random(seed)
    difficulty = rng.choice(parse_list(args.difficulty))
    land = rng.choice(parse_list(args.land))
    opponents = rng.choice(parse_list(args.opponents))
    war = rng.random() < args.war
    name = f"run-{index:03d}"
    paths = {kind: (out / f"{name}.{ending}").resolve().as_posix()
             for kind, ending in [("json", "json"), ("journal", "jsonl"), ("summary", "txt")]}
    flags = ["-sim", str(args.turns), "-capture-opponents", str(opponents), "-capture-land-size", str(land),
             "-capture-difficulty", str(difficulty), "-sim-json", paths["json"], "-sim-journal", paths["journal"],
             "-sim-out", paths["summary"], "-sim-timeline", str(args.timeline)]
    if args.all_ai:
        flags.append("-sim-all-ai")
    else:
        flags.append("-sim-play-on")
    if war:
        flags.append("-sim-war")
    flags += args.extra.split() if args.extra else []
    config = {"index": index, "difficulty": difficulty, "land": land, "opponents": opponents, "war": war,
              "allAI": args.all_ai, "turns": args.turns, "flags": flags}
    started = time.time()
    with open(out / f"{name}.log", "w", encoding="utf-8", errors="replace") as log:
        process = subprocess.Popen(["bash", str(root / "dev.sh"), args.lane] + flags, stdout=log, stderr=subprocess.STDOUT, cwd=str(root))
        try:
            code = process.wait(timeout=args.timeout)
        except subprocess.TimeoutExpired:
            process.kill()
            code = "timeout"
    config["seconds"] = round(time.time() - started, 1)
    config["exit"] = code
    with open(out / f"{name}.config.json", "w", encoding="utf-8") as file:
        json.dump(config, file, indent=1)
    return name, config


def command_run(args):
    root = find_root(args.root or __file__)
    if root is None:
        sys.exit("no dev.sh found above the script; pass --root")
    out = Path(args.out) / args.name
    out.mkdir(parents=True, exist_ok=True)
    seeds = random.Random(args.seed).sample(range(1, 10**9), args.runs)
    print(f"{args.runs} runs of {args.turns} turns into {out}, {args.parallel} at a time", flush=True)
    with ThreadPoolExecutor(max_workers=args.parallel) as pool:
        futures = [pool.submit(one_run, args, root, out, index, seed) for index, seed in enumerate(seeds)]
        for future in as_completed(futures):
            name, config = future.result()
            print(f"  {name}: exit {config['exit']} in {config['seconds']}s (difficulty {config['difficulty']}, land {config['land']}, opponents {config['opponents']}, war {config['war']})", flush=True)


# ---------------------------------------------------------------- report

NUMBER = re.compile(r"-?\d+")


def reason_shape(text):
    """a reason with its numbers taken out, to count reasons of one kind together"""
    text = NUMBER.sub("#", text or "")
    text = re.sub(r"(the city|city) [A-Z][\w' ]+", r"\1 X", text)
    return text[:120]


def stats_of(values):
    values = [value for value in values if value is not None]
    if not values:
        return None
    return {"n": len(values), "min": min(values), "median": statistics.median(values), "mean": round(statistics.mean(values), 1), "max": max(values)}


def ratio_bucket(attack, defend):
    if defend <= 0:
        return "no defense"
    ratio = attack / defend
    for limit, name in [(0.5, "under 0.5"), (0.75, "0.5 to 0.75"), (1.0, "0.75 to 1"), (1.5, "1 to 1.5"), (2.0, "1.5 to 2"), (3.0, "2 to 3")]:
        if ratio < limit:
            return name
    return "3 and more"


BATTLE = re.compile(r"strength (\d+) \((\d+) units\) against (\d+) \((\d+) units\)")


def load_batch(folder):
    runs = []
    for config_path in sorted(Path(folder).glob("run-*.config.json")):
        name = config_path.name[:-len(".config.json")]
        config = json.loads(config_path.read_text(encoding="utf-8"))
        record_path = Path(folder) / f"{name}.json"
        record = None
        if record_path.exists():
            try:
                record = json.loads(record_path.read_text(encoding="utf-8"))
            except json.JSONDecodeError:
                record = None
        journal = []
        journal_path = Path(folder) / f"{name}.jsonl"
        if journal_path.exists():
            with open(journal_path, encoding="utf-8", errors="replace") as file:
                for line in file:
                    try:
                        journal.append(json.loads(line))
                    except json.JSONDecodeError:
                        pass
        log_tail = ""
        duplicates = 0
        debug_lines = []
        log_path = Path(folder) / f"{name}.log"
        if log_path.exists():
            text = log_path.read_text(encoding="utf-8", errors="replace")
            panic = text.find("panic:")
            log_tail = text[panic:panic + 3000] if panic >= 0 else ""
            duplicates = text.count("DUPLICATE first seen")
            for line in text.splitlines():
                if "DUPLICATE first seen" in line or " DEBUG " in line:
                    debug_lines.append(line[line.find("sim.go") if "sim.go" in line else 0:][:400])
        runs.append({"name": name, "config": config, "record": record, "journal": journal, "logPanic": log_tail, "duplicates": duplicates, "debug": debug_lines[:5]})
    return runs


def ending_of(run):
    record = run["record"]
    config = run["config"]
    if config.get("exit") == "timeout":
        return "killed after the timeout"
    if record is None:
        if run["logPanic"]:
            return "crashed (panic in the log)"
        return f"no record (exit {config.get('exit')})"
    if record.get("Panic"):
        return "panic"
    stalled = record.get("Stalled") or ""
    if stalled.startswith("HANG"):
        return "hang"
    if stalled.startswith("no new turn"):
        return "stalled"
    if stalled == "one wizard left":
        return "one wizard left"
    if stalled == "the game ended" or record.get("GameOver"):
        return "game over"
    return "all turns played"


def report_batch(name, runs):
    out = {"name": name, "runs": len(runs)}
    out["endings"] = collections.Counter(ending_of(run) for run in runs)
    out["settings"] = {
        "difficulty": collections.Counter(run["config"]["difficulty"] for run in runs),
        "land": collections.Counter(run["config"]["land"] for run in runs),
        "opponents": collections.Counter(run["config"]["opponents"] for run in runs),
        "war": collections.Counter(run["config"]["war"] for run in runs),
        "allAI": runs[0]["config"]["allAI"] if runs else None,
        "turns": runs[0]["config"]["turns"] if runs else None,
    }
    records = [run for run in runs if run["record"]]
    out["turnsPlayed"] = stats_of([run["record"]["EndTurn"] - run["record"]["StartTurn"] for run in records])
    out["secondsPerTurn"] = stats_of([round(run["record"]["Seconds"] / max(1, run["record"]["EndTurn"] - run["record"]["StartTurn"]), 3) for run in records])

    problems = []
    for run in runs:
        record = run["record"] or {}
        if record.get("Panic"):
            problems.append({"run": run["name"], "kind": "panic", "text": record["Panic"][:1500]})
        elif run["logPanic"]:
            problems.append({"run": run["name"], "kind": "crash", "text": run["logPanic"][:1500]})
        stalled = record.get("Stalled") or ""
        if stalled.startswith("HANG") or stalled.startswith("no new turn"):
            problems.append({"run": run["name"], "kind": "hang", "text": stalled[:600]})
        if run["config"].get("exit") == "timeout":
            problems.append({"run": run["name"], "kind": "timeout", "text": "killed after the timeout"})
    for run in runs:
        for line in run.get("debug", []):
            problems.append({"run": run["name"], "kind": "debug", "text": line})
    out["problems"] = problems

    violations = {}
    for run in records:
        for violation in run["record"].get("Violations") or []:
            entry = violations.setdefault(violation["Kind"], {"runs": 0, "count": 0, "firstTurns": [], "examples": []})
            entry["runs"] += 1
            entry["count"] += violation["Count"]
            entry["firstTurns"].append(violation["FirstTurn"])
            for example in violation["Examples"][:2]:
                if len(entry["examples"]) < 6:
                    entry["examples"].append(f"{run['name']} {example}")
    out["violations"] = dict(sorted(violations.items(), key=lambda item: -item[1]["runs"]))

    winners = collections.Counter()
    game_overs = []
    wizards = collections.defaultdict(list)
    neutral = collections.defaultdict(list)
    personalities = collections.defaultdict(lambda: collections.defaultdict(list))
    contacts = []
    for run in records:
        record = run["record"]
        if record.get("Winner"):
            winners[record["Winner"]] += 1
        if record.get("GameOver"):
            game_overs.append(record["GameOver"])
        contacts.append(len(record.get("Contacts") or []))
        for player in record["Players"]:
            end, stats = player["End"], player["Stats"]
            row = {"cities": end["Cities"], "citizens": end["Citizens"], "units": end["Units"], "strength": end["Strength"],
                   "gold": end["Gold"], "mana": end["Mana"], "fame": end["Fame"], "spells": end["Spells"], "out": end["Out"],
                   "attacks": stats["Attacks"], "attacksWon": stats["AttacksWon"], "defenses": stats["Defenses"],
                   "defensesWon": stats["DefensesWon"], "lairs": stats["LairsFought"], "lairsWon": stats["LairsWon"],
                   "founded": stats["CitiesFounded"], "taken": stats["CitiesTaken"], "lost": stats["CitiesLost"],
                   "thinkMs": stats["Think"] / 1e6, "slowestMs": stats["ThinkMost"] / 1e6}
            if player["Neutral"]:
                for key, value in row.items():
                    neutral[key].append(value)
            elif not player["Human"] or run["config"]["allAI"]:
                for key, value in row.items():
                    wizards[key].append(value)
                kind = f"{player['Personality']} {player['Objective']}"
                personalities[kind]["cities"].append(end["Cities"])
                personalities[kind]["strength"].append(end["Strength"])
                personalities[kind]["out"].append(1 if end["Out"] else 0)
    out["winners"] = winners
    out["gameOvers"] = game_overs
    out["contactsPerRun"] = stats_of(contacts)
    out["wizards"] = {key: stats_of([float(value) for value in values]) for key, values in wizards.items()}
    out["wizardsOut"] = sum(1 for value in wizards.get("out", []) if value)
    out["wizardCount"] = len(wizards.get("out", []))
    out["neutral"] = {key: stats_of([float(value) for value in values]) for key, values in neutral.items()}
    out["byPersonality"] = {kind: {"wizards": len(values["cities"]), "cities": stats_of(values["cities"]), "strength": stats_of(values["strength"]), "out": sum(values["out"])} for kind, values in sorted(personalities.items())}

    # the journal
    kinds = collections.Counter()
    reasons = collections.defaultdict(collections.Counter)
    whats = collections.defaultdict(collections.Counter)
    battles = collections.defaultdict(lambda: {"fought": 0, "won": 0})
    lairs = collections.defaultdict(lambda: {"fought": 0, "won": 0})
    raids = collections.defaultdict(lambda: {"fought": 0, "won": 0})
    first_war = []
    first_treaty = []
    for run in runs:
        war_seen = treaty_seen = False
        for line in run["journal"]:
            kind = line["kind"]
            kinds[kind] += 1
            reasons[kind][reason_shape(line.get("why"))] += 1
            what = line.get("what") or ""
            if kind in ("cast", "spell start", "research", "event", "hire hero", "dismiss", "disband"):
                whats[kind][what.split(" in ")[0].split(":")[0]] += 1
            elif kind == "build":
                whats[kind][what.split(": ", 1)[-1]] += 1
            elif kind == "target":
                target = re.sub(r".* go for ", "", what)
                target = re.sub(r" at \d+,\d+$", "", target)
                target = re.sub(r"the city .* of ", "a city of ", target)
                target = re.sub(r"a stack of \d+ units of .*", "a stack", target)
                target = re.sub(r"a city of .*", "a city", target)
                whats[kind][target] += 1
            elif kind in ("war", "treaty", "peace", "break treaty"):
                whats[kind][kind] += 1
            if kind == "battle":
                match = BATTLE.search(line.get("why") or "")
                if match:
                    attack, defend = int(match.group(1)), int(match.group(3))
                    won = what.endswith("attacker won") or what.endswith("defender fled")
                    is_lair = " at a " in what and "guardians" in what
                    if is_lair and int(match.group(4)) == 0:
                        pass  # an empty lair: treasure, no fight
                    else:
                        table = lairs if is_lair else (raids if line.get("banner") == "brown" else battles)
                        bucket = table[ratio_bucket(attack, defend)]
                        bucket["fought"] += 1
                        bucket["won"] += 1 if won else 0
            if kind == "war" and not war_seen:
                war_seen = True
                first_war.append(line["turn"])
            if kind == "treaty" and not treaty_seen:
                treaty_seen = True
                first_treaty.append(line["turn"])
    # milestones: the first turn of each key decision or happening, for every wizard of every run
    milestone_kinds = ["found city", "target", "battle", "take city", "hire hero", "hire mercenaries", "spell start", "research", "treaty", "war", "dismiss", "disband"]
    firsts = collections.defaultdict(list)
    for run in runs:
        seen = {}
        for line in run["journal"]:
            if line.get("banner") in (None, "brown") or line["player"] == "the game":
                continue
            key = (line["player"], line["kind"])
            if line["kind"] == "battle" and "guardians" in (line.get("what") or ""):
                key = (line["player"], "lair fight")
            if key not in seen:
                seen[key] = line["turn"]
        for (player, kind), turn in seen.items():
            firsts[kind].append(turn)
    out["milestones"] = {kind: stats_of(firsts[kind]) for kind in milestone_kinds + ["lair fight"] if firsts.get(kind)}

    order = ["under 0.5", "0.5 to 0.75", "0.75 to 1", "1 to 1.5", "1.5 to 2", "2 to 3", "3 and more", "no defense"]
    out["journalKinds"] = dict(kinds.most_common())
    out["journalPerRun"] = {kind: round(count / max(1, len(runs)), 1) for kind, count in kinds.most_common()}
    out["reasons"] = {kind: counter.most_common(8) for kind, counter in reasons.items()}
    out["whats"] = {kind: counter.most_common(12) for kind, counter in whats.items()}
    out["battlesByRatio"] = {bucket: battles[bucket] for bucket in order if bucket in battles}
    out["lairsByRatio"] = {bucket: lairs[bucket] for bucket in order if bucket in lairs}
    out["raidsByRatio"] = {bucket: raids[bucket] for bucket in order if bucket in raids}

    # one line a run, and the slowest turns of the AI
    lines = []
    slow = []
    for run in runs:
        record = run["record"] or {}
        config = run["config"]
        violations = sorted((record.get("Violations") or []), key=lambda value: -value["Count"])
        wizards = [player for player in record.get("Players", []) if not player["Neutral"]]
        lines.append({"run": run["name"], "difficulty": config["difficulty"], "land": config["land"], "opponents": config["opponents"], "war": config["war"],
                      "ending": ending_of(run), "turns": (record.get("EndTurn", 0) - record.get("StartTurn", 0)) if record else None,
                      "winner": record.get("Winner", ""), "seconds": config.get("seconds"),
                      "cities": [player["End"]["Cities"] for player in wizards], "out": sum(1 for player in wizards if player["End"]["Out"]),
                      "violations": [f"{value['Kind']} {value['Count']}" for value in violations[:3]], "duplicates": run.get("duplicates", 0)})
        for player in record.get("Players", []):
            slow.append((player["Stats"]["ThinkMost"] / 1e6, run["name"], player["Name"], round(player["Stats"]["Think"] / 1e9, 1)))
    out["runLines"] = lines
    out["slowest"] = [{"ms": round(ms), "run": name_, "player": player, "totalSeconds": total} for ms, name_, player, total in sorted(slow, reverse=True)[:8]]
    out["duplicateRuns"] = sum(1 for run in runs if run.get("duplicates"))
    out["firstWarTurn"] = stats_of(first_war)
    out["runsWithWar"] = len(first_war)
    out["firstTreatyTurn"] = stats_of(first_treaty)
    out["runsWithTreaty"] = len(first_treaty)

    # the curves: the mean of the computer wizards at every turn of the timeline
    curves = collections.defaultdict(lambda: collections.defaultdict(list))
    for run in records:
        names = {player["Name"] for player in run["record"]["Players"] if not player["Neutral"] and (not player["Human"] or run["config"]["allAI"])}
        for point in run["record"].get("Timeline") or []:
            for name_, snapshot in point["Players"].items():
                if name_ in names:
                    for key in ("Cities", "Citizens", "Units", "Strength", "Gold", "Mana"):
                        curves[key][point["Turn"]].append(snapshot[key])
    out["curves"] = {key: {str(turn): round(statistics.mean(values), 1) for turn, values in sorted(points.items())} for key, points in curves.items()}
    return out


def markdown(reports):
    lines = ["# Runs without a window: the report", ""]
    for report in reports:
        settings = report["settings"]
        lines += [f"## {report['name']}", "",
                  f"{report['runs']} runs of {settings['turns']} turns; every wizard a computer one: {settings['allAI']}; "
                  f"computer wizards {dict(settings['opponents'])}; difficulty {dict(settings['difficulty'])}; land {dict(settings['land'])}; war at the start {dict(settings['war'])}.", ""]
        lines.append("How the runs ended: " + ", ".join(f"{name} {count}" for name, count in report["endings"].most_common()))
        if report["turnsPlayed"]:
            lines.append(f"Turns played: median {report['turnsPlayed']['median']}, {report['turnsPlayed']['min']} to {report['turnsPlayed']['max']}. Seconds a turn: median {report['secondsPerTurn']['median']}.")
        if report["winners"]:
            lines.append("One wizard left: " + ", ".join(f"{name} {count}" for name, count in report["winners"].most_common()))
        lines.append("")
        if report["problems"]:
            lines += ["### Panics, crashes, hangs", ""]
            for problem in report["problems"]:
                first = problem["text"].splitlines()[0] if problem["text"] else ""
                lines.append(f"- {problem['run']} {problem['kind']}: {first[:300]}")
            lines.append("")
        lines += ["### Broken states (checks of every turn)", ""]
        if report["violations"]:
            for kind, entry in report["violations"].items():
                lines.append(f"- **{kind}**: {entry['runs']} runs, {entry['count']} times, first at turns {sorted(entry['firstTurns'])[:6]}; e.g. {' | '.join(entry['examples'][:3])}")
        else:
            lines.append("- none")
        lines.append("")
        wizards = report["wizards"]
        if wizards:
            lines += ["### The computer wizards at the end (median, min to max)", ""]
            for key in ("cities", "citizens", "units", "strength", "gold", "mana", "fame", "spells", "attacks", "attacksWon", "lairs", "lairsWon", "founded", "taken", "lost", "thinkMs", "slowestMs"):
                value = wizards.get(key)
                if value:
                    lines.append(f"- {key}: {value['median']} ({value['min']} to {value['max']})")
            lines.append(f"- out of the game: {report['wizardsOut']} of {report['wizardCount']}")
            lines.append("")
        if report["byPersonality"]:
            lines += ["### By personality and objective (cities and strength at the end, median)", ""]
            for kind, value in report["byPersonality"].items():
                lines.append(f"- {kind}: {value['wizards']} wizards, cities {value['cities']['median'] if value['cities'] else '-'}, strength {value['strength']['median'] if value['strength'] else '-'}, out {value['out']}")
            lines.append("")
        lines += ["### Battles by the strength of the attacker against the defender (won of fought)", ""]
        for title, table in (("Wizards attacking", report["battlesByRatio"]), ("Raiders and monsters attacking", report["raidsByRatio"]), ("Lairs, nodes, towers (not the empty ones)", report["lairsByRatio"])):
            if table:
                lines.append(f"- {title}: " + "; ".join(f"{bucket} {value['won']}/{value['fought']}" for bucket, value in table.items()))
        lines.append("")
        if report.get("milestones"):
            lines += ["### Milestones: the first turn a wizard did it (median, min to max, wizards that did)", ""]
            for kind, value in report["milestones"].items():
                lines.append(f"- {kind}: {value['median']} ({value['min']} to {value['max']}), {value['n']} wizards")
            lines.append("")
        lines += ["### Diplomacy", "",
                  f"- runs with a war: {report['runsWithWar']} of {report['runs']}, first war turn {report['firstWarTurn']}",
                  f"- runs with a treaty: {report['runsWithTreaty']} of {report['runs']}, first treaty turn {report['firstTreatyTurn']}",
                  f"- contacts per run: {report['contactsPerRun']}", ""]
        lines += ["### Decisions and happenings of the journal (a run)", ""]
        lines.append(", ".join(f"{kind} {count}" for kind, count in report["journalPerRun"].items()))
        lines.append("")
        for kind in ("build", "target", "research", "spell start", "cast", "event", "dismiss", "disband", "war", "treaty", "break treaty", "hire hero"):
            if kind in report["whats"] or kind in report["reasons"]:
                lines.append(f"- **{kind}**")
                if kind in report["whats"]:
                    lines.append("  - what: " + "; ".join(f"{name} {count}" for name, count in report["whats"][kind][:10]))
                if kind in report["reasons"]:
                    lines.append("  - why: " + "; ".join(f"{name} ({count})" for name, count in report["reasons"][kind][:5]))
        lines.append("")
        lines += ["### The slowest turns of the AI", ""]
        for entry in report["slowest"]:
            lines.append(f"- {entry['ms']} ms: {entry['run']} {entry['player']} (all its turns {entry['totalSeconds']} s)")
        lines += ["", "### Every run", "", "| run | difficulty | land | wizards | war | ending | turns | winner | cities at the end | broken states |", "|---|---|---|---|---|---|---|---|---|---|"]
        for line in report["runLines"]:
            lines.append(f"| {line['run']} | {line['difficulty']} | {line['land']} | {line['opponents'] + 1} | {line['war']} | {line['ending']} | {line['turns']} | {line['winner']} | {line['cities']} | {'; '.join(line['violations'])} |")
        lines.append("")
        curves = report["curves"]
        if curves:
            lines += ["### Growth of the computer wizards (mean by turn)", ""]
            turns = list(curves.get("Cities", {}).keys())
            step = max(1, len(turns) // 10)
            shown = turns[::step]
            lines.append("| turn | " + " | ".join(shown) + " |")
            lines.append("|---|" + "---|" * len(shown))
            for key in ("Cities", "Citizens", "Units", "Strength", "Gold", "Mana"):
                if key in curves:
                    lines.append(f"| {key} | " + " | ".join(str(curves[key].get(turn, "")) for turn in shown) + " |")
            lines.append("")
    return "\n".join(lines)


def command_report(args):
    reports = []
    for folder in args.folders:
        runs = load_batch(folder)
        if not runs:
            print(f"no runs in {folder}")
            continue
        reports.append(report_batch(Path(folder).name, runs))
    out = Path(args.out)
    out.mkdir(parents=True, exist_ok=True)
    (out / "report.json").write_text(json.dumps(reports, indent=1, default=lambda value: dict(value) if isinstance(value, collections.Counter) else str(value)), encoding="utf-8")
    (out / "report.md").write_text(markdown(reports), encoding="utf-8")
    print(f"report of {len(reports)} batches written to {out}")


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    commands = parser.add_subparsers(dest="command", required=True)
    run = commands.add_parser("run", help="play a batch of runs")
    run.add_argument("--name", required=True, help="the batch, a folder under --out")
    run.add_argument("--out", required=True)
    run.add_argument("--runs", type=int, default=8)
    run.add_argument("--turns", type=int, default=250)
    run.add_argument("--parallel", type=int, default=6)
    run.add_argument("--lane", default="a")
    run.add_argument("--root", default=None, help="the folder of dev.sh (found above the script)")
    run.add_argument("--opponents", default="1,2,3,4", help="computer wizards, a list picked from by chance")
    run.add_argument("--difficulty", default="2", help="0 intro to 5 impossible, a list")
    run.add_argument("--land", default="0,1,2", help="land size 0 to 2, a list")
    run.add_argument("--war", type=float, default=0.0, help="the chance of every wizard at war at the start")
    run.add_argument("--all-ai", action="store_true", help="the human's slot played by Chewbot")
    run.add_argument("--timeline", type=int, default=10)
    run.add_argument("--timeout", type=int, default=1800, help="seconds before a run is killed")
    run.add_argument("--seed", type=int, default=1, help="picks the settings of the runs (the game's own chance can not be seeded)")
    run.add_argument("--extra", default="", help="more flags for the game")
    run.set_defaults(func=command_run)
    report = commands.add_parser("report", help="the report of batches")
    report.add_argument("folders", nargs="+")
    report.add_argument("--out", required=True)
    report.set_defaults(func=command_report)
    args = parser.parse_args()
    args.func(args)


if __name__ == "__main__":
    main()
