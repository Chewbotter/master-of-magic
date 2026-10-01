#!/usr/bin/env python3
"""Development: one page of the reports of simbatch.py, for reading and sharing.

    python util/simbatch/report_html.py --round "Round 1=D:/x/fuzz/report1/report.json" \
        --round "Round 2=D:/x/fuzz2/report/report.json" --notes notes.json --out report.html

The page has the findings of --notes first (a JSON object: title, subtitle, and lists "fixed",
"open", "behavior", "tools", each of strings or of {"title", "text"}), then every batch of every
round: how the runs ended, the broken states, battles by the strength of the sides, the decisions
of the journal, the growth curves and every run. The data is embedded; the page needs nothing else.
"""

import argparse
import html
import json
from pathlib import Path

PAGE = r"""<title>__TITLE__</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Chivo:wght@500;700;800&family=IBM+Plex+Sans:wght@400;500;600&family=IBM+Plex+Mono:wght@400;500&display=swap">
<style>
/* Layout: a lab readout. Findings first, then one panel per batch; tables and small SVG charts. */
:root {
  --bg: #f3f5f8; --panel: #ffffff; --ink: #18202c; --muted: #5b6676; --rule: #d9dee6;
  --accent: #1f6f8b; --accent-soft: #e3f0f4;
  --good: #2f7d4f; --warn: #a86b12; --bad: #b3362f; --good-soft: #e5f2ea; --warn-soft: #f7ecd9; --bad-soft: #f6e1df;
  --display: "Chivo", "Segoe UI", system-ui, sans-serif;
  --body: "IBM Plex Sans", "Segoe UI", system-ui, sans-serif;
  --mono: "IBM Plex Mono", ui-monospace, Consolas, monospace;
}
@media (prefers-color-scheme: dark) { :root:not([data-theme="light"]) {
  --bg: #11151c; --panel: #181e27; --ink: #e5e9ef; --muted: #9aa4b2; --rule: #2a323e;
  --accent: #5fb4d0; --accent-soft: #1b2c35;
  --good: #6cc48f; --warn: #e2a64a; --bad: #ee7a70; --good-soft: #17291f; --warn-soft: #2e2414; --bad-soft: #321b19; color-scheme: dark } }
:root[data-theme="dark"] {
  --bg: #11151c; --panel: #181e27; --ink: #e5e9ef; --muted: #9aa4b2; --rule: #2a323e;
  --accent: #5fb4d0; --accent-soft: #1b2c35;
  --good: #6cc48f; --warn: #e2a64a; --bad: #ee7a70; --good-soft: #17291f; --warn-soft: #2e2414; --bad-soft: #321b19; color-scheme: dark }
body { background: var(--bg); color: var(--ink); font-family: var(--body); font-size: 15px; line-height: 1.5; }
.wrap { max-width: 1120px; margin: 0 auto; padding-inline: 16px; padding-block: 28px 64px; display: grid; gap: 28px; }
h1, h2, h3 { font-family: var(--display); text-wrap: balance; margin: 0; letter-spacing: 0.01em; }
h1 { font-size: 30px; font-weight: 800; }
h2 { font-size: 21px; font-weight: 700; }
h3 { font-size: 15px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.06em; color: var(--muted); }
p { margin: 0; max-width: 72ch; }
.sub { color: var(--muted); }
header { display: grid; gap: 8px; }
.mono, td.num, .num { font-family: var(--mono); font-variant-numeric: tabular-nums; }
.panel { background: var(--panel); border: 1px solid var(--rule); border-radius: 10px; padding: 20px; display: grid; gap: 16px; min-width: 0; }
.grid2 { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 16px; }
.cards { display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 12px; }
.card { border: 1px solid var(--rule); border-radius: 8px; padding: 14px; display: grid; gap: 8px; align-content: start; min-width: 0; }
.card .big { font-family: var(--mono); font-size: 26px; font-weight: 500; }
.pills { display: flex; flex-wrap: wrap; gap: 6px; }
.pill { font-size: 12px; padding: 2px 8px; border-radius: 999px; background: var(--accent-soft); color: var(--ink); white-space: nowrap; }
.pill.good { background: var(--good-soft); color: var(--good); }
.pill.warn { background: var(--warn-soft); color: var(--warn); }
.pill.bad { background: var(--bad-soft); color: var(--bad); }
.findings { display: grid; gap: 10px; padding: 0; margin: 0; list-style: none; }
.findings li { border-left: 3px solid var(--rule); padding: 4px 0 4px 12px; }
.findings.fixed li { border-color: var(--good); }
.findings.open li { border-color: var(--warn); }
.findings.behavior li { border-color: var(--accent); }
.findings b { font-weight: 600; }
.scroll { overflow-x: auto; min-width: 0; }
table { border-collapse: collapse; width: 100%; font-size: 13px; }
th, td { text-align: left; padding: 6px 8px; border-bottom: 1px solid var(--rule); vertical-align: top; }
th { font-weight: 600; color: var(--muted); font-size: 12px; text-transform: uppercase; letter-spacing: 0.04em; }
td.num, th.num { text-align: right; }
.bar { display: grid; grid-template-columns: 120px 1fr 64px; gap: 8px; align-items: center; font-size: 13px; }
.bar .track { height: 10px; background: var(--rule); border-radius: 5px; overflow: hidden; }
.bar .fill { height: 100%; background: var(--accent); }
.bar .fill.good { background: var(--good); } .bar .fill.bad { background: var(--bad); } .bar .fill.warn { background: var(--warn); }
details > summary { cursor: pointer; font-weight: 600; }
details > summary:focus-visible, a:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; }
.tabs { display: flex; flex-wrap: wrap; gap: 6px; }
.tabs button { font: inherit; font-size: 13px; padding: 6px 12px; border-radius: 6px; border: 1px solid var(--rule); background: var(--panel); color: var(--ink); cursor: pointer; }
.tabs button[aria-selected="true"] { background: var(--accent); border-color: var(--accent); color: var(--panel); }
.tabs button:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; }
svg text { fill: var(--muted); font-family: var(--mono); font-size: 10px; }
.reasons li { font-size: 13px; }
code { font-family: var(--mono); font-size: 0.92em; }
@media (prefers-reduced-motion: reduce) { * { transition: none !important; } }
</style>
<div class="wrap">
  <header>
    <h1>__TITLE__</h1>
    <p class="sub">__SUBTITLE__</p>
  </header>
  <section class="panel" id="findings"></section>
  <section class="panel" id="compare"></section>
  <section class="panel" id="batches">
    <h2>Batches</h2>
    <div class="tabs" role="tablist" id="tabs"></div>
    <div id="batch"></div>
  </section>
</div>
<script type="application/json" id="data">__DATA__</script>
<script>
(function () {
  const DATA = JSON.parse(document.getElementById("data").textContent);
  const esc = s => String(s ?? "").replace(/[&<>"]/g, c => ({"&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;"}[c]));
  const fmt = v => v == null ? "–" : (typeof v === "number" ? (Number.isInteger(v) ? v.toLocaleString() : v.toLocaleString(undefined, {maximumFractionDigits: 1})) : esc(v));

  // findings
  const notes = DATA.notes || {};
  const item = x => typeof x === "string" ? `<li>${x}</li>` : `<li><b>${esc(x.title)}</b> ${x.text}</li>`;
  const list = (title, key, cls) => (notes[key] && notes[key].length) ? `<div class="grid-item"><h3>${title}</h3><ul class="findings ${cls}">${notes[key].map(item).join("")}</ul></div>` : "";
  document.getElementById("findings").innerHTML = `<h2>What the runs found</h2>
    ${notes.summary ? `<p>${notes.summary}</p>` : ""}
    <div class="grid2">${list("Fixed", "fixed", "fixed")}${list("Open, for you to decide", "open", "open")}</div>
    ${list("How the AI plays", "behavior", "behavior")}
    ${list("The tools", "tools", "")}`;

  // compare broken states across rounds
  const kinds = new Set();
  DATA.rounds.forEach(r => r.batches.forEach(b => Object.keys(b.violations || {}).forEach(k => kinds.add(k))));
  const roundRuns = r => r.batches.reduce((n, b) => n + b.runs, 0);
  const row = kind => `<tr><td>${esc(kind)}</td>${DATA.rounds.map(r => {
      const runs = r.batches.reduce((n, b) => n + ((b.violations || {})[kind]?.runs || 0), 0);
      const total = roundRuns(r);
      const share = total ? runs / total : 0;
      const cls = runs === 0 ? "good" : share > 0.25 ? "bad" : "warn";
      return `<td class="num"><span class="pill ${cls}">${runs} of ${total}</span></td>`;
    }).join("")}</tr>`;
  const problems = r => r.batches.reduce((n, b) => n + (b.problems || []).filter(p => p.kind !== "debug").length, 0);
  document.getElementById("compare").innerHTML = `<h2>Rounds compared</h2>
    <p class="sub">Runs with each broken state that the checks of every turn found, by round. Round 2 ran on the build with the fixes.</p>
    <div class="scroll"><table><thead><tr><th>Broken state</th>${DATA.rounds.map(r => `<th class="num">${esc(r.name)}</th>`).join("")}</tr></thead>
    <tbody>${[...kinds].sort().map(row).join("") || `<tr><td colspan="9">none</td></tr>`}
    <tr><td><b>Panics, crashes and hangs</b></td>${DATA.rounds.map(r => { const n = problems(r); return `<td class="num"><span class="pill ${n ? "bad" : "good"}">${n}</span></td>`; }).join("")}</tr>
    <tr><td><b>Runs</b></td>${DATA.rounds.map(r => `<td class="num">${roundRuns(r)}</td>`).join("")}</tr></tbody></table></div>`;

  // batches
  const all = [];
  DATA.rounds.forEach(r => r.batches.forEach(b => all.push({round: r.name, b})));
  const tabs = document.getElementById("tabs");
  tabs.innerHTML = all.map((x, i) => `<button role="tab" id="tab-${i}" aria-selected="${i === 0}" data-i="${i}">${esc(x.round)}: ${esc(x.b.name)}</button>`).join("");
  tabs.addEventListener("click", e => { const btn = e.target.closest("button"); if (!btn) return; show(+btn.dataset.i); });

  function bars(table, title) {
    const entries = Object.entries(table || {});
    if (!entries.length) return "";
    return `<div class="card"><h3>${title}</h3>${entries.map(([bucket, v]) => {
      const rate = v.fought ? v.won / v.fought : 0;
      const cls = rate >= 0.66 ? "good" : rate >= 0.33 ? "warn" : "bad";
      return `<div class="bar"><span>${esc(bucket)}</span><span class="track"><span class="fill ${cls}" style="width:${(rate * 100).toFixed(0)}%"></span></span><span class="num">${v.won}/${v.fought}</span></div>`;
    }).join("")}</div>`;
  }

  function chart(points, label) {
    const entries = Object.entries(points || {}).map(([t, v]) => [+t, v]).sort((a, b) => a[0] - b[0]);
    if (entries.length < 2) return "";
    const W = 300, H = 120, L = 40, R = 8, T = 10, B = 22;
    const xs = entries.map(e => e[0]), ys = entries.map(e => e[1]);
    const x0 = Math.min(...xs), x1 = Math.max(...xs), y1 = Math.max(...ys) || 1;
    const sx = x => L + (x - x0) / Math.max(1, x1 - x0) * (W - L - R);
    const sy = y => H - B - y / y1 * (H - T - B);
    const path = entries.map((e, i) => `${i ? "L" : "M"}${sx(e[0]).toFixed(1)},${sy(e[1]).toFixed(1)}`).join("");
    const area = `${path}L${sx(x1).toFixed(1)},${H - B}L${sx(x0).toFixed(1)},${H - B}Z`;
    const last = entries[entries.length - 1];
    return `<div class="card"><h3>${esc(label)}</h3><svg viewBox="0 0 ${W} ${H}" role="img" aria-label="${esc(label)} by turn">
      <line x1="${L}" y1="${H - B}" x2="${W - R}" y2="${H - B}" stroke="var(--rule)"/>
      <line x1="${L}" y1="${sy(y1)}" x2="${W - R}" y2="${sy(y1)}" stroke="var(--rule)" stroke-dasharray="2 3"/>
      <path d="${area}" fill="var(--accent-soft)"/><path d="${path}" fill="none" stroke="var(--accent)" stroke-width="1.8"/>
      <circle cx="${sx(last[0])}" cy="${sy(last[1])}" r="3" fill="var(--accent)"/>
      <text x="${L - 4}" y="${sy(y1) + 3}" text-anchor="end">${fmt(Math.round(y1))}</text>
      <text x="${L - 4}" y="${H - B + 3}" text-anchor="end">0</text>
      <text x="${L}" y="${H - 6}">turn ${x0}</text><text x="${W - R}" y="${H - 6}" text-anchor="end">${x1}</text></svg></div>`;
  }

  function show(i) {
    [...tabs.children].forEach((b, j) => b.setAttribute("aria-selected", String(i === j)));
    const {round, b} = all[i];
    const s = b.settings || {};
    const endings = Object.entries(b.endings || {}).map(([k, v]) => `<span class="pill ${/hang|panic|crash|killed|stall/.test(k) ? "bad" : k === "all turns played" ? "good" : ""}">${esc(k)} ${v}</span>`).join("");
    const w = b.wizards || {};
    const stat = k => w[k] ? `${fmt(w[k].median)} <span class="sub">(${fmt(w[k].min)} to ${fmt(w[k].max)})</span>` : "–";
    const viol = Object.entries(b.violations || {});
    const probs = (b.problems || []);
    const journal = Object.entries(b.journalPerRun || {});
    const reasonKinds = ["build", "target", "research", "spell start", "event", "dismiss", "disband", "treaty", "war", "hire hero"];
    document.getElementById("batch").innerHTML = `<div style="display:grid;gap:16px">
      <div class="cards">
        <div class="card"><h3>Runs</h3><div class="big">${b.runs}</div><div class="sub">${s.turns} turns each; every wizard a computer one: ${s.allAI ? "yes" : "no (the human idles)"}</div></div>
        <div class="card"><h3>How they ended</h3><div class="pills">${endings}</div></div>
        <div class="card"><h3>Turns played</h3><div class="big">${fmt(b.turnsPlayed?.median)}</div><div class="sub">median; ${fmt(b.secondsPerTurn?.median)} s a turn</div></div>
        <div class="card"><h3>Settings drawn</h3><div class="sub mono">wizards ${esc(JSON.stringify(Object.fromEntries(Object.entries(s.opponents || {}).map(([k, v]) => [+k + 1, v]))))}<br>difficulty ${esc(JSON.stringify(s.difficulty))}<br>land ${esc(JSON.stringify(s.land))}<br>war at start ${esc(JSON.stringify(s.war))}</div></div>
      </div>
      ${probs.length ? `<div class="card"><h3>Panics, hangs, debug lines</h3><ul class="findings open">${probs.slice(0, 12).map(p => `<li><b>${esc(p.run)} ${esc(p.kind)}</b> <span class="mono">${esc((p.text || "").split("\n")[0].slice(0, 260))}</span></li>`).join("")}</ul></div>` : ""}
      <div class="card"><h3>Broken states</h3>${viol.length ? `<div class="scroll"><table><thead><tr><th>Kind</th><th class="num">Runs</th><th class="num">Times</th><th>Examples</th></tr></thead><tbody>${viol.map(([k, v]) => `<tr><td>${esc(k)}</td><td class="num">${v.runs}</td><td class="num">${fmt(v.count)}</td><td class="mono" style="font-size:12px">${v.examples.slice(0, 3).map(esc).join("<br>")}</td></tr>`).join("")}</tbody></table></div>` : `<span class="pill good">none</span>`}</div>
      <div class="grid2">
        <div class="card"><h3>The computer wizards at the end</h3><div class="scroll"><table><tbody>
          ${["cities", "citizens", "units", "strength", "gold", "mana", "spells", "lairs", "lairsWon", "attacks", "attacksWon", "taken", "lost", "slowestMs"].map(k => `<tr><td>${k}</td><td class="num">${stat(k)}</td></tr>`).join("")}
          <tr><td>out of the game</td><td class="num">${b.wizardsOut} of ${b.wizardCount}</td></tr></tbody></table></div></div>
        <div style="display:grid;gap:12px">${bars(b.battlesByRatio, "Wizards attacking, won of fought by strength ratio")}${bars(b.raidsByRatio, "Raiders and monsters attacking")}${bars(b.lairsByRatio, "Lairs, nodes, towers")}</div>
      </div>
      <div class="cards">${["Cities", "Units", "Strength", "Mana"].map(k => chart((b.curves || {})[k], k + " of a wizard, mean")).join("")}</div>
      <div class="card"><h3>The journal: decisions and happenings a run</h3><div class="pills">${journal.map(([k, v]) => `<span class="pill">${esc(k)} ${fmt(v)}</span>`).join("")}</div>
        <div class="grid2">${reasonKinds.filter(k => (b.whats || {})[k] || (b.reasons || {})[k]).map(k => `<div><b>${esc(k)}</b><ul class="reasons">${((b.whats || {})[k] || []).slice(0, 6).map(([n, c]) => `<li>${esc(n)} <span class="sub num">${c}</span></li>`).join("")}</ul>${((b.reasons || {})[k] || []).slice(0, 3).map(([n, c]) => `<div class="sub" style="font-size:12px">why: ${esc(n)} <span class="num">(${c})</span></div>`).join("")}</div>`).join("")}</div></div>
      ${b.milestones ? `<div class="card"><h3>Milestones: the first turn a wizard did it</h3><div class="scroll"><table><thead><tr><th>Decision or happening</th><th class="num">Median</th><th class="num">Earliest</th><th class="num">Latest</th><th class="num">Wizards that did</th></tr></thead><tbody>${Object.entries(b.milestones).map(([k, v]) => `<tr><td>${esc(k)}</td><td class="num">${fmt(v.median)}</td><td class="num">${fmt(v.min)}</td><td class="num">${fmt(v.max)}</td><td class="num">${v.n} of ${b.wizardCount}</td></tr>`).join("")}</tbody></table></div></div>` : ""}
      <div class="card"><h3>Diplomacy</h3><div class="sub">runs with a war: ${b.runsWithWar} of ${b.runs}; with a treaty: ${b.runsWithTreaty} of ${b.runs}; first treaty at turn ${fmt(b.firstTreatyTurn?.median)} (median); contacts a run ${fmt(b.contactsPerRun?.mean)}</div></div>
      <details><summary>Every run (${(b.runLines || []).length})</summary><div class="scroll"><table><thead><tr><th>Run</th><th class="num">Diff.</th><th class="num">Land</th><th class="num">Wizards</th><th>War</th><th>Ending</th><th class="num">Turns</th><th>Winner</th><th>Cities at the end</th><th>Broken states</th></tr></thead><tbody>
        ${(b.runLines || []).map(l => `<tr><td class="mono">${esc(l.run)}</td><td class="num">${l.difficulty}</td><td class="num">${l.land}</td><td class="num">${l.opponents + 1}</td><td>${l.war ? "yes" : ""}</td><td>${esc(l.ending)}</td><td class="num">${fmt(l.turns)}</td><td>${esc(l.winner)}</td><td class="mono">${esc((l.cities || []).join(", "))}</td><td style="font-size:12px">${esc((l.violations || []).join("; "))}</td></tr>`).join("")}
      </tbody></table></div></details>
      <details><summary>The slowest turns of the AI</summary><ul>${(b.slowest || []).map(s => `<li class="mono">${fmt(s.ms)} ms, ${esc(s.run)} ${esc(s.player)} (all its turns ${s.totalSeconds} s)</li>`).join("")}</ul></details>
    </div>`;
  }
  show(0);
})();
</script>
"""


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--round", action="append", required=True, help="NAME=path/report.json, in order")
    parser.add_argument("--notes", help="a JSON object of findings")
    parser.add_argument("--out", required=True)
    args = parser.parse_args()
    rounds = []
    for entry in args.round:
        name, path = entry.split("=", 1)
        rounds.append({"name": name, "batches": json.loads(Path(path).read_text(encoding="utf-8"))})
    notes = json.loads(Path(args.notes).read_text(encoding="utf-8")) if args.notes else {}
    data = json.dumps({"rounds": rounds, "notes": notes}).replace("</", "<\\/")
    page = PAGE.replace("__TITLE__", html.escape(notes.get("title", "Test Runs"))).replace("__SUBTITLE__", html.escape(notes.get("subtitle", ""))).replace("__DATA__", data)
    Path(args.out).write_text(page, encoding="utf-8")
    print(f"written {args.out} ({len(page) // 1024} KB)")


if __name__ == "__main__":
    main()
