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
    "bufio"
    "cmp"
    "compress/gzip"
    "encoding/json"
    "fmt"
    "log"
    "math/rand/v2"
    "os"
    "path/filepath"
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
    for _, city := range player.Cities {
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
    spells, err := spellbook.ReadSpellsFromCache(magic.Cache)
    if err != nil {
        return nil, "", err
    }
    wizard, ok := gamelib.ChooseUniqueWizard(nil, spells)
    if !ok {
        return nil, "", fmt.Errorf("could not choose a wizard")
    }
    game := initializeGame(magic, settings, wizard)
    return game, fmt.Sprintf("a quick start, %v computer wizards, land size %v", settings.Opponents, settings.LandSize), nil
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
        if simWar {
            simDeclareWar(game)
            description += ", every wizard at war"
        }

        run := simPlay(game, description)
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
func simPlay(game *gamelib.Game, description string) simRun {
    game.Model.CurrentPlayer = 0
    game.RefreshUI()
    game.Stats = gamelib.MakeSimStats()
    ai.ChewbotOrderCounts = make(map[string]map[string]int)

    players := slices.Clone(game.Model.Players)
    before := make(map[*playerlib.Player]simSnapshot)
    for _, player := range players {
        before[player] = simTake(player)
    }

    game.SimSkipHuman()
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
    for game.Model.TurnNumber < startTurn + uint64(simTurns) {
        if game.Update(yield) == gamelib.GameStateQuit {
            stalled = "the game ended"
            break
        }
        if game.Model.TurnNumber != lastTurn {
            lastTurn = game.Model.TurnNumber
            lastProgress = time.Now()
            if (lastTurn - startTurn) % 25 == 0 {
                log.Printf("sim: turn %v after %v", lastTurn, time.Since(start).Round(time.Millisecond))
            }
        }
        if time.Since(lastProgress) > simStallTime {
            stalled = fmt.Sprintf("no new turn for %v; on screen: %v", simStallTime, game.DescribeDrawers())
            break
        }
    }

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
