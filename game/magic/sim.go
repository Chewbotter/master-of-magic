package main

// Development: games of computer players without a window (-sim N). A game is made (a quick
// start as -start makes it, or a saved game with -sim-load), the human skips its turns (its battles
// are fought without a screen, random events are not shown), and the game's own update runs as
// fast as it can, with no frame to wait for. At the end a summary of every player: what it had at
// the start and at the end, its battles, cities and thinking time, and the orders of Chewbot by the
// step of the original that gave them. A run that stops going on (a screen that waits for the
// human) ends with what is on screen in the log.
//
// SCENARIOS: saved games in the folder `scenarios` beside `mod` (D:/Work/MasterMagic_open/scenarios,
// outside the repo). -sim-load takes a file there by its name ("coast", "coast.magic-save") or any
// path; a save of the game menu (_build/fileN.magic-save) or one of the original game (.GAM).
// -sim-save writes the game at the end of a run, so a run makes a scenario for the next ones.
// -sim-repeat K plays a saved game K times and adds averages: the same map and players each time,
// so two AIs or two versions can be compared fairly. -sim-war puts every wizard at war with every
// other at the start.
//
//   bash dev.sh a -sim 150 -capture-opponents 4 -sim-save turn150
//   bash dev.sh a -sim 100 -sim-load turn150 -sim-repeat 5 -sim-ai clone

import (
    "image"
    "bufio"
    "cmp"
    "compress/gzip"
    "encoding/json"
    "fmt"
    "log"
    rand "github.com/kazzmir/master-of-magic/lib/chance"
    "os"
    "path/filepath"
    "runtime/debug"
    "runtime/pprof"
    "slices"
    "strings"
    "time"

    "github.com/kazzmir/master-of-magic/game/magic/ai"
    "github.com/kazzmir/master-of-magic/game/magic/combat"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/game/magic/inputmanager"
    "github.com/kazzmir/master-of-magic/game/magic/load"
    "github.com/kazzmir/master-of-magic/game/magic/setup"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    "github.com/kazzmir/master-of-magic/lib/lbx"
    "github.com/kazzmir/master-of-magic/lib/system"
    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    gamelib "github.com/kazzmir/master-of-magic/game/magic/game"
    musiclib "github.com/kazzmir/master-of-magic/game/magic/music"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/relations"
    settingslib "github.com/kazzmir/master-of-magic/game/magic/settings"
)

// turns of a run without a window; 0 is a normal start
var simTurns int
// where the summary is written, beside the log
var simOut string
// the Enemy AI of the run: clone or chewbot, empty for the setting
var simAI string
// a saved game to start from, and where to save the game at the end
var simLoad string
var simSave string
// how many times a saved game is played
var simRepeat int
// every wizard at war with every other at the start
var simWar bool
// a CPU profile of the run is written here (go tool pprof -top <exe> <file>)
var simProfile string

// a run that has not got to another turn in this long waits for something that never comes
const simStallTime = 30 * time.Second

// the ending of a scenario file
const simSaveEnding = ".magic-save"

// what a player has at one moment
type simSnapshot struct {
    Cities int
    Citizens int
    Units int
    Strength int
    Gold int
    Mana int
    Fame int
    Spells int
    Out bool
}

func simTake(player *playerlib.Player) simSnapshot {
    snapshot := simSnapshot{
        Cities: len(player.Cities),
        Gold: player.Gold,
        Mana: player.Mana,
        Fame: player.Fame,
        Spells: len(player.KnownSpells.Spells),
        Out: player.Defeated || player.Banished,
    }
    for _, city := range player.CitiesInOrder() {
        snapshot.Citizens += city.Citizens()
    }
    for _, stack := range player.Stacks {
        for _, unit := range stack.Units() {
            snapshot.Units += 1
            snapshot.Strength += combat.ChewbotUnitStrength(unit, unit.VisibleFigures(), unit.GetHealth()) / 10
        }
    }
    return snapshot
}

func simName(player *playerlib.Player) string {
    kind := ""
    switch {
        case player.IsHuman(): kind = ", human"
        case player.IsNeutral(): kind = ", neutral"
    }
    return fmt.Sprintf("%v (%v%v)", player.Wizard.Name, player.GetBanner(), kind)
}

// the personality and objective of a computer wizard
func simKind(player *playerlib.Player) string {
    if player.IsHuman() || player.IsNeutral() {
        return ""
    }
    return fmt.Sprintf("%v %v", player.Personality, player.Objective)
}

// one player of one run
type simRow struct {
    Name string
    Start simSnapshot
    End simSnapshot
    Stats gamelib.SimPlayerStats
    Orders map[string]int
}

// one run
type simRun struct {
    // the treaties and relations of every pair of wizards at the end, and what computer wizards said
    // to the human
    Diplomacy string
    Description string
    StartTurn uint64
    EndTurn uint64
    Elapsed time.Duration
    Stalled string
    Rows []simRow
}

// the folder of scenarios: `scenarios` beside the folder `mod` above where the game runs
func simScenarioFolder() string {
    for _, up := range []string{".", "..", filepath.Join("..", "..")} {
        if info, err := os.Stat(filepath.Join(up, "mod")); err == nil && info.IsDir() {
            return filepath.Join(up, "scenarios")
        }
        if info, err := os.Stat(filepath.Join(up, "scenarios")); err == nil && info.IsDir() {
            return filepath.Join(up, "scenarios")
        }
    }
    return "scenarios"
}

// a scenario by its name or a path
func simFindSave(name string) (string, error) {
    folder := simScenarioFolder()
    candidates := []string{name, name + simSaveEnding, filepath.Join(folder, name), filepath.Join(folder, name + simSaveEnding)}
    for _, candidate := range candidates {
        if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
            return candidate, nil
        }
    }
    return "", fmt.Errorf("no saved game %v (looked in %v too)", name, folder)
}

// where -sim-save writes: a bare name goes to the folder of scenarios
func simSavePath(name string) string {
    if strings.ContainsAny(name, `/\`) {
        return name
    }
    if !strings.HasSuffix(name, simSaveEnding) {
        name += simSaveEnding
    }
    folder := simScenarioFolder()
    os.MkdirAll(folder, 0755)
    return filepath.Join(folder, name)
}

func simLoadGame(magic *MagicGame, path string) (*gamelib.Game, error) {
    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    if strings.EqualFold(filepath.Ext(path), ".gam") {
        saved, err := load.LoadSaveGame(file)
        if err != nil {
            return nil, err
        }
        game := saved.Convert(magic.Cache, magic.Music, magic.Settings)
        if game == nil {
            return nil, fmt.Errorf("could not convert the original's saved game %v", path)
        }
        return game, nil
    }

    reader, err := gzip.NewReader(bufio.NewReader(file))
    if err != nil {
        return nil, err
    }
    defer reader.Close()
    var serialized gamelib.SerializedGame
    if err := json.NewDecoder(reader).Decode(&serialized); err != nil {
        return nil, err
    }
    return gamelib.MakeGameFromSerialized(magic.Cache, magic.Music, magic.Settings, &serialized), nil
}

// what is on each plane at the start: size, land, lairs and towers, nodes, neutral cities
func simWorld(game *gamelib.Game) string {
    var out strings.Builder
    for _, plane := range []data.Plane{data.PlaneArcanus, data.PlaneMyrror} {
        mapObject := game.GetMap(plane)
        land := 0
        minerals := 0
        roads := 0
        rivers := 0
        for x := 0; x < mapObject.Width(); x++ {
            for y := 0; y < mapObject.Height(); y++ {
                if mapObject.GetTile(x, y).Tile.IsLand() {
                    land += 1
                }
                if mapObject.GetBonusTile(x, y) != data.BonusNone {
                    minerals += 1
                }
                if mapObject.ContainsRoad(x, y) {
                    roads += 1
                }
                if mapObject.GetTile(x, y).Tile.IsRiver() {
                    rivers += 1
                }
            }
        }
        neutral := 0
        for _, city := range game.Model.AllCities() {
            if city.Plane == plane && city.GetBanner() == data.BannerBrown {
                neutral += 1
            }
        }
        encounters := len(mapObject.GetEncounterLocations()) - len(mapObject.GetMagicNodeLocations())
        fmt.Fprintf(&out, "%v: %v by %v, land %v, lairs and towers %v, nodes %v, neutral cities %v, minerals %v, road squares %v, river squares %v\n", plane, mapObject.Width(), mapObject.Height(), land, encounters, len(mapObject.GetMagicNodeLocations()), neutral, minerals, roads, rivers)
    }
    return strings.TrimSuffix(out.String(), "\n")
}

// a quick start as -start makes it
func simQuickGame(magic *MagicGame) (*gamelib.Game, string, error) {
    settings := setup.NewGameSettings{
        Opponents: rand.N(4) + 1,
        Difficulty: data.DifficultyAverage,
        Magic: data.MagicSettingNormal,
        LandSize: rand.N(3),
    }
    if captureOpponents > 0 {
        settings.Opponents = captureOpponents
    }
    if captureLandSize >= 0 {
        settings.LandSize = captureLandSize
    }
    if captureDifficulty >= 0 {
        settings.Difficulty = data.DifficultySetting(captureDifficulty)
    }
    spells, err := spellbook.ReadSpellsFromCache(magic.Cache)
    if err != nil {
        return nil, "", err
    }
    wizard, ok := gamelib.ChooseUniqueWizard(nil, spells)
    if !ok {
        return nil, "", fmt.Errorf("could not choose a wizard")
    }
    game := initializeGame(magic, settings, wizard)
    return game, fmt.Sprintf("a quick start, %v computer wizards, land size %v (0 small, 1 medium, 2 large), difficulty %v (0 intro to 5 impossible), difficulty bonuses %v\n%v", settings.Opponents, settings.LandSize, settings.Difficulty, citylib.ClassicDifficulty, simWorld(game)), nil
}

// every wizard at war with every other (not the neutral player)
func simDeclareWar(game *gamelib.Game) {
    for _, player := range game.Model.Players {
        for _, other := range game.Model.Players {
            if player == other || player.IsNeutral() || other.IsNeutral() {
                continue
            }
            player.WarWithPlayer(other)
        }
    }
}

func runSimulation(config GameConfig) error {
    cache := lbx.CacheFromPath(config.DataPath)
    if cache == nil {
        return fmt.Errorf("could not load data from '%v'", config.DataPath)
    }
    magic := &MagicGame{Cache: cache, Config: config}
    magic.Music = musiclib.MakeMusic(cache)
    magic.Music.Enabled = false
    magic.Settings = settingslib.MakeSettings(cache)
    inputmanager.NoInput()

    if simProfile != "" {
        profile, err := os.Create(simProfile)
        if err != nil {
            return err
        }
        defer profile.Close()
        if err := pprof.StartCPUProfile(profile); err != nil {
            return err
        }
        defer pprof.StopCPUProfile()
    }

    switch simAI {
        case "clone": display.Current.EnemyAIName = display.EnemyAIClone
        case "chewbot": display.Current.EnemyAIName = display.EnemyAIChewbot
        case "":
        default:
            return fmt.Errorf("unknown -sim-ai %v: clone or chewbot", simAI)
    }

    loadPath := ""
    if simLoad != "" {
        path, err := simFindSave(simLoad)
        if err != nil {
            return err
        }
        loadPath = path
    }
    repeat := max(1, simRepeat)
    if loadPath == "" && repeat > 1 {
        return fmt.Errorf("-sim-repeat needs -sim-load: quick starts differ every time")
    }

    var runs []simRun
    var summary strings.Builder
    for index := range repeat {
        if simSeed != 0 {
            // every draw of chance from here on, the world included, comes from this seed (lib/chance)
            rand.Seed(simSeed + uint64(index))
            if simSeedTrace != "" && index == 0 {
                // the line of every draw, to find where two runs of one seed part
                if file, err := os.Create(simSeedTrace); err == nil {
                    rand.TraceTo(file, 2000000)
                }
            }
        }
        var game *gamelib.Game
        description := ""
        if loadPath != "" {
            loaded, err := simLoadGame(magic, loadPath)
            if err != nil {
                return fmt.Errorf("could not load %v: %v", loadPath, err)
            }
            game = loaded
            description = fmt.Sprintf("the saved game %v", loadPath)
        } else {
            made, text, err := simQuickGame(magic)
            if err != nil {
                return err
            }
            game = made
            description = text
        }
        if simSeed != 0 {
            description += fmt.Sprintf(", seed %v", simSeed + uint64(index))
        }
        if simWar {
            simDeclareWar(game)
            description += ", every wizard at war"
        }

        run := simPlay(game, description, index, repeat)
        if index == repeat - 1 && simSave != "" {
            path := simSavePath(simSave)
            saver := &gamelib.GameSaver{Game: game, FS: system.MakeFS()}
            if err := saver.SaveToPath(path, fmt.Sprintf("sim turn %v", game.Model.TurnNumber)); err != nil {
                log.Printf("sim: could not save to %v: %v", path, err)
            } else {
                log.Printf("sim: the game at turn %v is saved to %v", game.Model.TurnNumber, path)
                run.Description += fmt.Sprintf("; saved at the end to %v", path)
            }
        }
        game.Shutdown()

        runs = append(runs, run)
        if repeat > 1 {
            fmt.Fprintf(&summary, "=== Run %v of %v\n", index + 1, repeat)
        }
        summary.WriteString(simSummary(run))
        summary.WriteString("\n")
    }
    if repeat > 1 {
        summary.WriteString(simAverages(runs))
    }

    log.Printf("sim summary:\n%v", summary.String())
    if simOut != "" {
        if err := os.WriteFile(simOut, []byte(summary.String()), 0644); err != nil {
            log.Printf("sim: could not write %v: %v", simOut, err)
        } else {
            log.Printf("sim: summary written to %v", simOut)
        }
    }
    return nil
}

// plays the turns of one game
func simPlay(game *gamelib.Game, description string, index int, count int) simRun {
    game.Model.CurrentPlayer = 0
    game.RefreshUI()
    game.Stats = gamelib.MakeSimStats()
    ai.ChewbotOrderCounts = make(map[string]map[string]int)

    if simAllAI {
        // every wizard a computer one (simrecord.go)
        simMakeAllAI(game)
        description += ", the human's slot played by Chewbot"
    } else if simStandIn {
        // the human's seat played by Chewbot, still the human for the rules (game/standin.go)
        simMakeStandIn(game)
        description += ", the human played by a stand-in (Chewbot)"
    }
    var sink *simJournalSink
    if simJournal != "" || simJSON != "" {
        sink = simOpenJournal(game, simRunPath(simJournal, index, count))
        defer sink.close()
    }
    stopAlone := simStopAlone || simAllAI || simStandIn

    players := slices.Clone(game.Model.Players)
    before := make(map[*playerlib.Player]simSnapshot)
    for _, player := range players {
        before[player] = simTake(player)
    }

    game.SimSkipHuman()
    if simAllAI || simStandIn {
        // the human's slot plays its turns
        game.Model.GetHumanPlayer().Skip = false
    }
    select {
        case game.Events <- &gamelib.GameEventNextTurn{}:
        default:
    }

    yield := func() error {
        return nil
    }

    start := time.Now()
    startTurn := game.Model.TurnNumber
    lastTurn := startTurn
    lastProgress := time.Now()
    stalled := ""

    // a line of every player every simTimeline turns (simrecord.go)
    var timeline []simTimelinePoint
    takeTimeline := func() {
        point := simTimelinePoint{Turn: game.Model.TurnNumber, Players: make(map[string]simSnapshot)}
        for _, player := range players {
            point.Players[player.Wizard.Name] = simTake(player)
        }
        timeline = append(timeline, point)
    }
    takeTimeline()

    jsonPath := simRunPath(simJSON, index, count)
    watchdog := simStartWatchdog(func(stacks string) {
        run := simRun{Description: description, StartTurn: startTurn, EndTurn: game.Model.TurnNumber, Elapsed: time.Since(start),
            Stalled: fmt.Sprintf("HANG at turn %v: the game's loop did not come back for %v", game.Model.TurnNumber, simHangTime)}
        simWriteRecord(jsonPath, simMakeRecord(game, run, players, before, timeline, sink))
        if sink != nil {
            sink.close()
        }
    })
    defer watchdog.close()

    // one update of the game; a panic is recorded and ends the run
    update := func() (state gamelib.GameState, panicked bool) {
        defer func() {
            if recovered := recover(); recovered != nil {
                game.Stats.NotePanic(fmt.Sprintf("turn %v, the game's loop: %v; %v", game.Model.TurnNumber, recovered, string(debug.Stack())))
                panicked = true
            }
        }()
        return game.Update(yield), false
    }

    // -sim-state-log: every city and stack at every turn (simrecord.go)
    var stateLog *bufio.Writer
    if path := simRunPath(simStateLog, index, count); path != "" {
        if file, err := os.Create(path); err == nil {
            stateLog = bufio.NewWriter(file)
            defer file.Close()
            defer stateLog.Flush()
        }
    }

    // -sim-save-every: the last saves of the run, and the ones kept for a trace (simrecord.go)
    simTraces = nil
    var saves []string
    keptSaves := make(map[string]bool)
    saveFolder := ""
    if simSaveEvery > 0 {
        base := simRunPath(simJSON, index, count)
        if base == "" {
            base = "sim"
        }
        saveFolder = strings.TrimSuffix(base, filepath.Ext(base)) + ".saves"
        os.MkdirAll(saveFolder, 0755)
    }
    rollingSave := func() {
        path := filepath.Join(saveFolder, fmt.Sprintf("turn-%03d%v", game.Model.TurnNumber, simSaveEnding))
        saver := &gamelib.GameSaver{Game: game, FS: system.MakeFS()}
        if err := saver.SaveToPath(path, fmt.Sprintf("sim turn %v", game.Model.TurnNumber)); err != nil {
            log.Printf("sim: could not save to %v: %v", path, err)
            return
        }
        saves = append(saves, path)
        for len(saves) > 2 {
            if !keptSaves[saves[0]] {
                os.Remove(saves[0])
            }
            saves = saves[1:]
        }
    }

    duplicateSeen := false
    traced := make(map[string]bool)
    for game.Model.TurnNumber < startTurn + uint64(simTurns) {
        state, panicked := update()
        watchdog.alive()
        if simTrace {
            // the step of the loop that first makes each kind of broken state (game/simcheck.go)
            for kind, example := range game.SimBrokenNow() {
                if !traced[kind] {
                    traced[kind] = true
                    log.Printf("sim: TRACE %v first seen at turn %v, current player %v: %v; the journal before it: | %v", kind, game.Model.TurnNumber, game.Model.CurrentPlayer, example, sink.recentText())
                    trace := simTraceRecord{Kind: kind, Turn: game.Model.TurnNumber, Example: example, Journal: sink.recentLines()}
                    if current := game.Model.CurrentPlayer; current >= 0 && current < len(game.Model.Players) {
                        trace.Player = game.Model.Players[current].Wizard.Name
                    }
                    if len(saves) > 0 {
                        // the save before it is kept: -sim-load it to play the case again
                        trace.Save = saves[len(saves) - 1]
                        keptSaves[trace.Save] = true
                    }
                    simTraces = append(simTraces, trace)
                }
            }
        }
        if !duplicateSeen {
            // the step of the loop that puts a unit in two stacks (game/simcheck.go)
            if found := game.SimDuplicateUnit(); found != "" {
                duplicateSeen = true
                log.Printf("sim: DUPLICATE first seen at turn %v, current player %v: %v; the journal before it:%v%v", game.Model.TurnNumber, game.Model.CurrentPlayer, found, " | ", sink.recentText())
            }
        }
        if panicked {
            stalled = "a panic"
            break
        }
        if state == gamelib.GameStateQuit {
            stalled = "the game ended"
            break
        }
        if text := game.Stats.PanicText(); text != "" {
            stalled = "a panic of the AI"
            log.Printf("sim: PANIC %v", text)
            break
        }
        if game.Model.TurnNumber != lastTurn {
            lastTurn = game.Model.TurnNumber
            lastProgress = time.Now()
            if stateLog != nil {
                simWriteState(stateLog, game)
            }
            if simSaveEvery > 0 && (lastTurn - startTurn) % uint64(simSaveEvery) == 0 {
                rollingSave()
            }
            // the checks of every turn (game/simcheck.go)
            game.SimCheck()
            if simTimeline > 0 && (lastTurn - startTurn) % uint64(simTimeline) == 0 {
                takeTimeline()
            }
            if (lastTurn - startTurn) % 25 == 0 {
                log.Printf("sim: turn %v after %v", lastTurn, time.Since(start).Round(time.Millisecond))
            }
            if stopAlone && simWizardsLeft(players) <= 1 {
                stalled = "one wizard left"
                break
            }
        }
        if time.Since(lastProgress) > simStallTime {
            stalled = fmt.Sprintf("no new turn for %v; on screen: %v", simStallTime, game.DescribeDrawers())
            break
        }
    }
    takeTimeline()

    run := simRun{
        Description: fmt.Sprintf("%v, Enemy AI %v", description, display.Current.EnemyAI().Name()),
        StartTurn: startTurn,
        EndTurn: game.Model.TurnNumber,
        Elapsed: time.Since(start),
        Stalled: stalled,
    }
    run.Diplomacy = simDiplomacy(game, players)
    for _, player := range players {
        run.Rows = append(run.Rows, simRow{
            Name: simName(player),
            Start: before[player],
            End: simTake(player),
            Stats: *game.Stats.Of(player),
            Orders: ai.ChewbotOrderCounts[player.Wizard.Name],
        })
    }
    if jsonPath != "" {
        simWriteRecord(jsonPath, simMakeRecord(game, run, players, before, timeline, sink))
    }
    if violations := game.Stats.ViolationList(); len(violations) > 0 {
        run.Diplomacy += "\nBroken states found by the checks of every turn\n"
        for _, violation := range violations {
            run.Diplomacy += fmt.Sprintf("  %v: %v times, turns %v to %v; %v\n", violation.Kind, violation.Count, violation.FirstTurn, violation.LastTurn, strings.Join(violation.Examples, " | "))
        }
    }
    if text := game.Stats.PanicText(); text != "" {
        run.Diplomacy += "\nPANIC: " + text + "\n"
    }
    if winner := simWinner(players); winner != "" {
        run.Diplomacy += "\nOne wizard left: " + winner + "\n"
    }
    return run
}

func simSummary(run simRun) string {
    var out strings.Builder
    turns := run.EndTurn - run.StartTurn
    perTurn := time.Duration(0)
    if turns > 0 {
        perTurn = run.Elapsed / time.Duration(turns)
    }
    fmt.Fprintf(&out, "Run without a window: %v turns (turn %v to %v) in %v, %v a turn\n", turns, run.StartTurn, run.EndTurn, run.Elapsed.Round(time.Millisecond), perTurn.Round(time.Millisecond))
    fmt.Fprintf(&out, "%v\n", run.Description)
    if run.Stalled != "" {
        fmt.Fprintf(&out, "STOPPED EARLY: %v\n", run.Stalled)
    }

    fmt.Fprintf(&out, "\nAt the start and at the end\n")
    fmt.Fprintf(&out, "%-26s %9s %9s %9s %11s %11s %11s %9s %9s\n", "player", "cities", "citizens", "units", "strength", "gold", "mana", "fame", "spells")
    change := func(from int, to int) string {
        return fmt.Sprintf("%v>%v", from, to)
    }
    for _, row := range run.Rows {
        name := row.Name
        if row.End.Out {
            name += " OUT"
        }
        fmt.Fprintf(&out, "%-26s %9s %9s %9s %11s %11s %11s %9s %9s\n", name,
            change(row.Start.Cities, row.End.Cities), change(row.Start.Citizens, row.End.Citizens), change(row.Start.Units, row.End.Units),
            change(row.Start.Strength, row.End.Strength), change(row.Start.Gold, row.End.Gold), change(row.Start.Mana, row.End.Mana),
            change(row.Start.Fame, row.End.Fame), change(row.Start.Spells, row.End.Spells))
    }

    fmt.Fprintf(&out, "\nWhat happened (won of fought)\n")
    fmt.Fprintf(&out, "%-26s %9s %9s %9s %9s %9s %9s %10s %10s\n", "player", "attacks", "defenses", "lairs", "founded", "taken", "lost", "thinking", "slowest")
    for _, row := range run.Rows {
        stats := row.Stats
        fmt.Fprintf(&out, "%-26s %9s %9s %9s %9v %9v %9v %10v %10v\n", row.Name,
            fmt.Sprintf("%v/%v", stats.AttacksWon, stats.Attacks), fmt.Sprintf("%v/%v", stats.DefensesWon, stats.Defenses),
            fmt.Sprintf("%v/%v", stats.LairsWon, stats.LairsFought), stats.CitiesFounded, stats.CitiesTaken, stats.CitiesLost,
            stats.Think.Round(time.Millisecond), stats.ThinkMost.Round(time.Millisecond))
    }

    out.WriteString(run.Diplomacy)

    header := false
    for _, row := range run.Rows {
        if len(row.Orders) == 0 {
            continue
        }
        if !header {
            fmt.Fprintf(&out, "\nOrders of Chewbot, units by the step of the original that gave them\n")
            header = true
        }
        fmt.Fprintf(&out, "%-26s %v\n", row.Name, simOrderText(row.Orders))
    }
    return out.String()
}

// for every pair of wizards that have not met: the nearest two of their cities (squares, the larger
// of the distances across and down, across the edge of the map) and whether they are on one landmass
func simNearest(game *gamelib.Game, players []*playerlib.Player) string {
    var out strings.Builder
    for index, first := range players {
        for _, second := range players[index + 1:] {
            if first.IsNeutral() || second.IsNeutral() || first.IsAwareOf(second) {
                continue
            }
            best := -1
            sameLand := false
            for _, a := range first.CitiesInOrder() {
                for _, b := range second.CitiesInOrder() {
                    if a.Plane != b.Plane {
                        continue
                    }
                    mapObject := game.Model.GetMap(a.Plane)
                    dx := a.X - b.X
                    if dx < 0 {
                        dx = -dx
                    }
                    dx = min(dx, mapObject.Width() - dx)
                    dy := a.Y - b.Y
                    if dy < 0 {
                        dy = -dy
                    }
                    distance := max(dx, dy)
                    if best < 0 || distance < best {
                        best = distance
                        sameLand = mapObject.Map.FindContinent(a.X, a.Y).Contains(image.Pt(b.X, b.Y))
                    }
                }
            }
            if best >= 0 {
                fmt.Fprintf(&out, "  not met: %v and %v, nearest cities %v squares apart, one landmass %v\n", first.Wizard.Name, second.Wizard.Name, best, sameLand)
            }
        }
    }
    return out.String()
}

// the diplomacy at the end of a run: personalities, every pair's treaty, relation and hostility, and
// what computer wizards said to the human
func simDiplomacy(game *gamelib.Game, players []*playerlib.Player) string {
    var out strings.Builder
    fmt.Fprintf(&out, "\nDiplomacy at the end (treaty, relation, hostility of the first to the second)\n")
    for _, player := range players {
        if kind := simKind(player); kind != "" {
            fmt.Fprintf(&out, "%-26s %v\n", simName(player), kind)
        }
    }
    for index, first := range players {
        for _, second := range players[index + 1:] {
            relation, ok := first.PlayerRelations[second]
            if !ok {
                continue
            }
            back := 0
            if other, ok := second.PlayerRelations[first]; ok {
                back = other.Hostility
            }
            fmt.Fprintf(&out, "  %v - %v: %v, relation %v (%v), hostility %v / %v\n", first.Wizard.Name, second.Wizard.Name, relation.Treaty, relation.VisibleRelation, relation.Description(), relation.Hostility, back)
        }
    }
    for _, conquest := range game.Stats.Conquests {
        fmt.Fprintf(&out, "Conquest: %v\n", conquest)
    }
    if game.Stats.GameOver != "" {
        fmt.Fprintf(&out, "Game over: %v\n", game.Stats.GameOver)
    }
    if len(game.Stats.Contacts) == 0 {
        fmt.Fprintf(&out, "No two wizards met\n")
    }
    fmt.Fprintf(&out, "%v", simNearest(game, players))
    for _, contact := range game.Stats.Contacts {
        fmt.Fprintf(&out, "  met in turn %v: %v and %v\n", contact.Turn, contact.First.Wizard.Name, contact.Second.Wizard.Name)
    }
    if len(game.Stats.Messages) > 0 {
        var parts []string
        for action, count := range game.Stats.Messages {
            parts = append(parts, fmt.Sprintf("%v: %v", simActionName(action), count))
        }
        slices.Sort(parts)
        fmt.Fprintf(&out, "Said to the human: %v\n", strings.Join(parts, ", "))
    }
    return out.String()
}

func simActionName(action int) string {
    switch {
        case action >= relations.ActionGreeting && action < relations.ActionGreeting + 6: return "greeting"
        case action == relations.ActionWarDeclared: return "war declared"
        case action == relations.ActionChaoticWar: return "chaotic war"
        case action == relations.ActionSuperiorityWar: return "war, superiority"
        case action == relations.ActionSuperiorityBreak: return "treaty broken"
        case action == relations.ActionProposePact: return "pact proposed"
        case action == relations.ActionProposeAlliance: return "alliance proposed"
        case action == relations.ActionProposePeace: return "peace proposed"
        case action == relations.ActionProposeExchange: return "exchange proposed"
        case action == relations.ActionUnitsNearCity: return "units near a city"
    }
    return fmt.Sprintf("action %v", action)
}

func simOrderText(counts map[string]int) string {
    var steps []string
    for step := range counts {
        steps = append(steps, step)
    }
    slices.SortFunc(steps, func(a string, b string) int {
        return cmp.Or(cmp.Compare(counts[b], counts[a]), cmp.Compare(a, b))
    })
    var parts []string
    for _, step := range steps {
        parts = append(parts, fmt.Sprintf("%v %v", step, counts[step]))
    }
    return strings.Join(parts, ", ")
}

// the averages of the runs of one saved game, player by player (the players are the same in all)
func simAverages(runs []simRun) string {
    var out strings.Builder
    count := float64(len(runs))
    fmt.Fprintf(&out, "=== Average of %v runs\n", len(runs))
    fmt.Fprintf(&out, "%-26s %8s %8s %8s %9s %8s %8s %8s %8s %8s %8s\n", "player", "cities", "citizens", "units", "strength", "gold", "attacks", "won", "taken", "lost", "out")
    for index, first := range runs[0].Rows {
        var cities, citizens, units, strength, gold, attacks, won, taken, lost, out2 float64
        for _, run := range runs {
            if index >= len(run.Rows) {
                continue
            }
            row := run.Rows[index]
            cities += float64(row.End.Cities)
            citizens += float64(row.End.Citizens)
            units += float64(row.End.Units)
            strength += float64(row.End.Strength)
            gold += float64(row.End.Gold)
            attacks += float64(row.Stats.Attacks)
            won += float64(row.Stats.AttacksWon)
            taken += float64(row.Stats.CitiesTaken)
            lost += float64(row.Stats.CitiesLost)
            if row.End.Out {
                out2 += 1
            }
        }
        fmt.Fprintf(&out, "%-26s %8.1f %8.1f %8.1f %9.0f %8.0f %8.1f %8.1f %8.1f %8.1f %8.0f\n", first.Name,
            cities / count, citizens / count, units / count, strength / count, gold / count, attacks / count, won / count, taken / count, lost / count, out2)
    }

    totals := make(map[string]map[string]int)
    for _, run := range runs {
        for _, row := range run.Rows {
            for step, value := range row.Orders {
                if totals[row.Name] == nil {
                    totals[row.Name] = make(map[string]int)
                }
                totals[row.Name][step] += value
            }
        }
    }
    if len(totals) > 0 {
        fmt.Fprintf(&out, "\nOrders of Chewbot in all runs together\n")
        for _, row := range runs[0].Rows {
            if len(totals[row.Name]) > 0 {
                fmt.Fprintf(&out, "%-26s %v\n", row.Name, simOrderText(totals[row.Name]))
            }
        }
    }
    return out.String()
}
