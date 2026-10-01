#!/bin/bash
# development: pairs of runs of one seed each (two games a seed at once): prints the first turn each
# pair parts, the first lines of the journals and of the state logs that differ. Run from anywhere:
#     bash util/simbatch/seedcheck.sh TURNS OPPONENTS SEED... (builds lane a first)
turns=$1; shift
opponents=$1; shift
cd /d/Work/MasterMagic_open || exit 1
bash build.sh a 2>&1 | tail -1
cd _build/a || exit 1
for s in "$@"; do
    for i in 1 2; do
        bash ../../dev.sh a -sim "$turns" -sim-all-ai -capture-opponents "$opponents" -capture-land-size 1 -sim-seed "$s" \
            -sim-json probe/seedr$s-$i.json -sim-timeline 1 -sim-state-log probe/seedst$s-$i.txt -sim-journal probe/seedj$s-$i.jsonl -sim-out probe/seedo$s-$i.txt > probe/seedl$s-$i.txt 2>&1 &
    done
done
wait
python - "$@" <<'EOF'
import json, sys
for s in sys.argv[1:]:
    a = json.load(open(f"probe/seedr{s}-1.json", encoding="utf-8"))["Timeline"]
    b = json.load(open(f"probe/seedr{s}-2.json", encoding="utf-8"))["Timeline"]
    for pa, pb in zip(a, b):
        if pa != pb:
            print(s, "part at turn", pa["Turn"])
            break
    else:
        print(s, "same for", len(a), "points")
        continue
    ja = open(f"probe/seedj{s}-1.jsonl", encoding="utf-8").read().splitlines()
    jb = open(f"probe/seedj{s}-2.jsonl", encoding="utf-8").read().splitlines()
    for index, (x, y) in enumerate(zip(ja, jb)):
        if x != y:
            print("   1:", x[:230])
            print("   2:", y[:230])
            break
EOF
for s in "$@"; do
    echo "== state $s"
    diff probe/seedst$s-1.txt probe/seedst$s-2.txt | head -6 | cut -c1-240
done
