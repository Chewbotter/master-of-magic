package main

// Development: what a run without a window (-sim, sim.go) records beyond its summary, for many runs
// and their reports (util/simbatch):
//
//   -sim-all-ai          the human's slot is played by Chewbot too: every wizard is a computer one
//   -sim-json FILE       the run as JSON: settings, every player at the start and the end, a line of
//                        every player every -sim-timeline turns, battles, conquests, contacts, the
//                        broken states the checks of every turn found, a panic, the winner
//   -sim-journal FILE    the key decisions and happenings, one JSON line each: turn, player, kind,
//                        what, why (player/journal.go; the AI notes builds, buys, targets, settling,
//                        research, spells, taxes, disbanding, hires, treaties and wars; the game
//                        notes battles, cities, events, casts, conquests)
//   -sim-timeline N      the turns between two lines of the timeline (10)
//   -sim-stop-alone      the run ends when one wizard is left (with -sim-all-ai, on by default)
//   -sim-seed N          the seed of all chance (lib/chance), so a run can be played again
//   -sim-trace           the checks after every update; the first of each kind is logged with the
//                        journal before it (finds the step that breaks something)
//
// A panic of the AI's goroutine or of the game's loop ends the run with the panic recorded; a run
// that hangs (the game's loop never comes back) is ended by a watchdog that writes what it knows
// and the stacks of every goroutine to the log.

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "os"
    "runtime/pprof"
    "slices"
    "sync"
    "sync/atomic"
    "time"

    "github.com/kazzmir/master-of-magic/game/magic/ai"
    gamelib "github.com/kazzmir/master-of-magic/game/magic/game"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/relations"
)

var simAllAI bool
var simJSON string
var simJournal string
var simTimeline int = 10
var simStopAlone bool
// -sim-seed: the seed of chance of a run (0: none, the game's own chance); repeats add their index
var simSeed uint64
// -sim-seed-trace FILE: the line of the game behind every draw of a seeded run
var simSeedTrace string
// -sim-state-log FILE: every city and stack of every player at every turn, one line each, so two
// runs of one seed show the first thing that differs
var simStateLog string

// one turn of the state log
func simWriteState(writer io.Writer, game *gamelib.Game) {
    turn := game.Model.TurnNumber
    for _, player := range game.Model.Players {
        fmt.Fprintf(writer, "%v %v gold %v mana %v research %v skill %v\n", turn, player.Wizard.Name, player.Gold, player.Mana, player.ResearchProgress, player.CastingSkillPower)
        for _, city := range player.CitiesInOrder() {
            fmt.Fprintf(writer, "%v %v city %v at %v,%v people %v farmers %v workers %v production %.3f making %v%v gold %v food %v work %.3f\n",
                turn, player.Wizard.Name, city.Name, city.X, city.Y, city.Population, city.Farmers, city.Workers, city.Production,
                city.ProducingBuilding, city.ProducingUnit.Name, city.GoldSurplus(), city.FoodProductionRate(), city.WorkProductionRate())
        }
        for _, stack := range player.Stacks {
            fmt.Fprintf(writer, "%v %v stack at %v,%v,%v units", turn, player.Wizard.Name, stack.X(), stack.Y(), stack.Plane())
            for _, unit := range stack.Units() {
                fmt.Fprintf(writer, " %v/%v", unit.GetName(), unit.GetHealth())
            }
            fmt.Fprintf(writer, "\n")
        }
    }
}
// -sim-trace: after every update the checks run, and the first time each kind of broken state is
// seen the journal before it is logged (slower)
var simTrace bool

// how long the game's loop may not come back before the watchdog ends the run
const simHangTime = 90 * time.Second

// one line of the journal
type simJournalLine struct {
    Turn uint64 `json:"turn"`
    Player string `json:"player"`
    Banner string `json:"banner,omitempty"`
    Kind string `json:"kind"`
    What string `json:"what"`
    Why string `json:"why,omitempty"`
}

// the journal of one run: lines to a file, counts by player and kind
type simJournalSink struct {
    lock sync.Mutex
    // the last lines, for the moment something breaks
    recent []simJournalLine
    game *gamelib.Game
    file *os.File
    encoder *json.Encoder
    Counts map[string]map[string]int
}

func simOpenJournal(game *gamelib.Game, path string) *simJournalSink {
    sink := &simJournalSink{game: game, Counts: make(map[string]map[string]int)}
    if path != "" {
        file, err := os.Create(path)
        if err != nil {
            log.Printf("sim: could not write the journal %v: %v", path, err)
        } else {
            sink.file = file
            sink.encoder = json.NewEncoder(file)
        }
    }
    playerlib.Journal = sink.note
    return sink
}

func (sink *simJournalSink) note(player *playerlib.Player, kind string, what string, why string) {
    sink.lock.Lock()
    defer sink.lock.Unlock()
    line := simJournalLine{Turn: sink.game.Model.TurnNumber, Player: "the game", Kind: kind, What: what, Why: why}
    if player != nil {
        line.Player = player.Wizard.Name
        line.Banner = player.GetBanner().String()
    }
    counts, ok := sink.Counts[line.Player]
    if !ok {
        counts = make(map[string]int)
        sink.Counts[line.Player] = counts
    }
    counts[kind] += 1
    sink.recent = append(sink.recent, line)
    if len(sink.recent) > 30 {
        sink.recent = sink.recent[len(sink.recent) - 30:]
    }
    if sink.encoder != nil {
        sink.encoder.Encode(line)
    }
}

func (sink *simJournalSink) close() {
    playerlib.Journal = nil
    sink.lock.Lock()
    defer sink.lock.Unlock()
    if sink.file != nil {
        sink.file.Close()
        sink.file = nil
        sink.encoder = nil
    }
}

func (sink *simJournalSink) countsCopy() map[string]map[string]int {
    sink.lock.Lock()
    defer sink.lock.Unlock()
    out := make(map[string]map[string]int)
    for player, counts := range sink.Counts {
        out[player] = make(map[string]int)
        for kind, count := range counts {
            out[player][kind] = count
        }
    }
    return out
}

// the human's slot is a computer wizard played by Chewbot
func simMakeAllAI(game *gamelib.Game) {
    human := game.Model.GetHumanPlayer()
    if human == nil || !human.IsHuman() {
        return
    }
    human.Human = false
    human.AIBehavior = ai.MakeChewbotAI(ai.MakeEnemy2AI(), false)
    human.Personality, human.Objective = relations.PickPersonality(human.Wizard)
    // the wizard of the human's slot is one of the computer wizards: its fall does not end the run
    // (the game's own end after a conquest is the human's)
    gamelib.SimPlayOn = true
}

// one line of the timeline: every player at one turn
type simTimelinePoint struct {
    Turn uint64
    Players map[string]simSnapshot
}

// what -sim-json writes for one run
type simRecord struct {
    Description string
    Settings map[string]any
    StartTurn uint64
    EndTurn uint64
    Seconds float64
    Stalled string
    Panic string
    GameOver string
    Winner string
    Conquests []string
    Contacts []string
    Violations []gamelib.SimViolation
    Players []simRecordPlayer
    Timeline []simTimelinePoint
    Journal map[string]map[string]int
}

type simRecordPlayer struct {
    Name string
    Banner string
    Human bool
    Neutral bool
    Personality string
    Objective string
    Retorts []string
    Books map[string]int
    Start simSnapshot
    End simSnapshot
    Stats gamelib.SimPlayerStats
    Orders map[string]int
}

// the record of a run so far: at its end, or when the watchdog ends a hang
func simMakeRecord(game *gamelib.Game, run simRun, players []*playerlib.Player, before map[*playerlib.Player]simSnapshot, timeline []simTimelinePoint, sink *simJournalSink) simRecord {
    record := simRecord{
        Description: run.Description,
        Settings: simSettings(game),
        StartTurn: run.StartTurn,
        EndTurn: game.Model.TurnNumber,
        Seconds: run.Elapsed.Seconds(),
        Stalled: run.Stalled,
        Panic: game.Stats.PanicText(),
        GameOver: game.Stats.GameOver,
        Winner: simWinner(players),
        Conquests: slices.Clone(game.Stats.Conquests),
        Violations: game.Stats.ViolationList(),
        Timeline: timeline,
    }
    for _, contact := range game.Stats.Contacts {
        record.Contacts = append(record.Contacts, fmt.Sprintf("turn %v: %v and %v", contact.Turn, contact.First.Wizard.Name, contact.Second.Wizard.Name))
    }
    if sink != nil {
        record.Journal = sink.countsCopy()
    }
    for _, player := range players {
        row := simRecordPlayer{
            Name: player.Wizard.Name,
            Banner: player.GetBanner().String(),
            Human: player.IsHuman(),
            Neutral: player.IsNeutral(),
            Start: before[player],
            End: simTake(player),
            Stats: *game.Stats.Of(player),
            Orders: ai.ChewbotOrderCounts[player.Wizard.Name],
            Books: make(map[string]int),
        }
        if !player.IsNeutral() {
            row.Personality = player.Personality.String()
            row.Objective = player.Objective.String()
        }
        for _, retort := range player.Wizard.Retorts {
            row.Retorts = append(row.Retorts, retort.String())
        }
        for _, book := range player.Wizard.Books {
            row.Books[book.Magic.String()] += book.Count
        }
        record.Players = append(record.Players, row)
    }
    return record
}

// the settings of the game, for grouping runs
func simSettings(game *gamelib.Game) map[string]any {
    wizards := 0
    for _, player := range game.Model.Players {
        if !player.IsNeutral() {
            wizards += 1
        }
    }
    return map[string]any{
        "wizards": wizards,
        "difficulty": int(game.Model.Settings.Difficulty),
        "landSize": game.Model.Settings.LandSize,
        "magic": int(game.Model.Settings.Magic),
        "allAI": simAllAI,
        "war": simWar,
        "playOn": gamelib.SimPlayOn,
        "turns": simTurns,
    }
}

// the one wizard left, empty when more are
func simWinner(players []*playerlib.Player) string {
    var left []*playerlib.Player
    for _, player := range players {
        if !player.IsNeutral() && !player.Defeated {
            left = append(left, player)
        }
    }
    if len(left) == 1 {
        return left[0].Wizard.Name
    }
    return ""
}

func simWizardsLeft(players []*playerlib.Player) int {
    left := 0
    for _, player := range players {
        if !player.IsNeutral() && !player.Defeated {
            left += 1
        }
    }
    return left
}

func simWriteRecord(path string, record simRecord) {
    if path == "" {
        return
    }
    data, err := json.MarshalIndent(record, "", " ")
    if err != nil {
        log.Printf("sim: could not make the record: %v", err)
        return
    }
    if err := os.WriteFile(path, data, 0644); err != nil {
        log.Printf("sim: could not write %v: %v", path, err)
    } else {
        log.Printf("sim: record written to %v", path)
    }
}

// the path of a record or journal of run index of count: a suffix when there are more runs
func simRunPath(path string, index int, count int) string {
    if path == "" || count <= 1 {
        return path
    }
    return fmt.Sprintf("%v.%v", path, index + 1)
}

// a watchdog: when the game's loop has not come back for simHangTime, the record so far is written
// with the stacks of every goroutine in the log, and the program ends
type simWatchdog struct {
    beat atomic.Int64
    stop chan struct{}
}

func simStartWatchdog(onHang func(stacks string)) *simWatchdog {
    watchdog := &simWatchdog{stop: make(chan struct{})}
    watchdog.beat.Store(time.Now().UnixNano())
    go func() {
        ticker := time.NewTicker(5 * time.Second)
        defer ticker.Stop()
        for {
            select {
                case <-watchdog.stop:
                    return
                case <-ticker.C:
                    last := time.Unix(0, watchdog.beat.Load())
                    if time.Since(last) > simHangTime {
                        var stacks bytes.Buffer
                        pprof.Lookup("goroutine").WriteTo(&stacks, 2)
                        log.Printf("sim: HANG, the game's loop has not come back for %v; the goroutines:\n%v", simHangTime, stacks.String())
                        onHang(stacks.String())
                        os.Exit(3)
                    }
            }
        }
    }()
    return watchdog
}

func (watchdog *simWatchdog) alive() {
    watchdog.beat.Store(time.Now().UnixNano())
}

func (watchdog *simWatchdog) close() {
    close(watchdog.stop)
}

// the last lines of the journal, as text
func (sink *simJournalSink) recentText() string {
    if sink == nil {
        return ""
    }
    sink.lock.Lock()
    defer sink.lock.Unlock()
    out := ""
    for _, line := range sink.recent {
        out += fmt.Sprintf("  turn %v %v %v: %v (%v)\n", line.Turn, line.Player, line.Kind, line.What, line.Why)
    }
    return out
}
