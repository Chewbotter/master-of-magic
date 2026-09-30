package main

// Development: a game of computer players without a window (-sim N). A quick start is made as
// -start makes it, the human skips its turns (its battles are fought without a screen, random
// events are not shown), and the game's own update runs as fast as it can, with no frame to wait
// for. At the end a summary of every player: what it had at the start and at the end, its
// battles, cities and thinking time, and the orders of Chewbot by the step of the original that
// gave them. A run that stops going on (a screen that waits for the human) ends with what is on
// screen in the log.
//
//   bash dev.sh a -sim 150 -capture-opponents 4
//   bash dev.sh a -sim 150 -capture-opponents 4 -sim-ai clone

import (
    "cmp"
    "fmt"
    "log"
    "math/rand/v2"
    "os"
    "slices"
    "strings"
    "time"

    "github.com/kazzmir/master-of-magic/game/magic/ai"
    "github.com/kazzmir/master-of-magic/game/magic/combat"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/game/magic/inputmanager"
    "github.com/kazzmir/master-of-magic/game/magic/setup"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    "github.com/kazzmir/master-of-magic/lib/lbx"
    gamelib "github.com/kazzmir/master-of-magic/game/magic/game"
    musiclib "github.com/kazzmir/master-of-magic/game/magic/music"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    settingslib "github.com/kazzmir/master-of-magic/game/magic/settings"
)

// turns of a run without a window; 0 is a normal start
var simTurns int
// where the summary is written, beside the log
var simOut string
// the Enemy AI of the run: clone or chewbot, empty for the setting
var simAI string

// a run that has not got to another turn in this long waits for something that never comes
const simStallTime = 30 * time.Second

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

func runSimulation(config GameConfig) error {
    cache := lbx.CacheFromPath(config.DataPath)
    if cache == nil {
        return fmt.Errorf("could not load data from '%v'", config.DataPath)
    }
    magic := &MagicGame{Cache: cache, Config: config}
    magic.Music = musiclib.MakeMusic(cache)
    magic.Music.Enabled = false
    magic.Settings = settingslib.MakeSettings(cache)

    switch simAI {
        case "clone": display.Current.EnemyAIName = display.EnemyAIClone
        case "chewbot": display.Current.EnemyAIName = display.EnemyAIChewbot
        case "":
        default:
            return fmt.Errorf("unknown -sim-ai %v: clone or chewbot", simAI)
    }

    settings := setup.NewGameSettings{
        Opponents: rand.N(4) + 1,
        Difficulty: data.DifficultyAverage,
        Magic: data.MagicSettingNormal,
        LandSize: rand.N(3),
    }
    if captureOpponents > 0 {
        settings.Opponents = captureOpponents
    }

    spells, err := spellbook.ReadSpellsFromCache(cache)
    if err != nil {
        return err
    }
    wizard, ok := gamelib.ChooseUniqueWizard(nil, spells)
    if !ok {
        return fmt.Errorf("could not choose a wizard")
    }

    made := time.Now()
    game := initializeGame(magic, settings, wizard)
    defer game.Shutdown()
    log.Printf("sim: game made in %v", time.Since(made).Round(time.Millisecond))

    inputmanager.NoInput()
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
    elapsed := time.Since(start)

    summary := simSummary(game, settings, players, before, startTurn, elapsed, stalled)
    log.Printf("sim summary:\n%v", summary)
    if simOut != "" {
        if err := os.WriteFile(simOut, []byte(summary), 0644); err != nil {
            log.Printf("sim: could not write %v: %v", simOut, err)
        } else {
            log.Printf("sim: summary written to %v", simOut)
        }
    }
    return nil
}

func simSummary(game *gamelib.Game, settings setup.NewGameSettings, players []*playerlib.Player, before map[*playerlib.Player]simSnapshot, startTurn uint64, elapsed time.Duration, stalled string) string {
    var out strings.Builder
    turns := game.Model.TurnNumber - startTurn
    perTurn := time.Duration(0)
    if turns > 0 {
        perTurn = elapsed / time.Duration(turns)
    }
    fmt.Fprintf(&out, "Run without a window: %v turns (turn %v to %v) in %v, %v a turn\n", turns, startTurn, game.Model.TurnNumber, elapsed.Round(time.Millisecond), perTurn.Round(time.Millisecond))
    fmt.Fprintf(&out, "Enemy AI %v, %v computer wizards, land size %v\n", display.Current.EnemyAI().Name(), settings.Opponents, settings.LandSize)
    if stalled != "" {
        fmt.Fprintf(&out, "STOPPED EARLY: %v\n", stalled)
    }

    fmt.Fprintf(&out, "\nAt the start and at the end\n")
    fmt.Fprintf(&out, "%-26s %9s %9s %9s %11s %11s %11s %9s %9s\n", "player", "cities", "citizens", "units", "strength", "gold", "mana", "fame", "spells")
    change := func(from int, to int) string {
        return fmt.Sprintf("%v>%v", from, to)
    }
    for _, player := range players {
        start := before[player]
        end := simTake(player)
        name := simName(player)
        if end.Out {
            name += " OUT"
        }
        fmt.Fprintf(&out, "%-26s %9s %9s %9s %11s %11s %11s %9s %9s\n", name,
            change(start.Cities, end.Cities), change(start.Citizens, end.Citizens), change(start.Units, end.Units),
            change(start.Strength, end.Strength), change(start.Gold, end.Gold), change(start.Mana, end.Mana),
            change(start.Fame, end.Fame), change(start.Spells, end.Spells))
    }

    fmt.Fprintf(&out, "\nWhat happened (won of fought)\n")
    fmt.Fprintf(&out, "%-26s %9s %9s %9s %9s %9s %9s %10s %10s\n", "player", "attacks", "defenses", "lairs", "founded", "taken", "lost", "thinking", "slowest")
    for _, player := range players {
        stats := game.Stats.Of(player)
        fmt.Fprintf(&out, "%-26s %9s %9s %9s %9v %9v %9v %10v %10v\n", simName(player),
            fmt.Sprintf("%v/%v", stats.AttacksWon, stats.Attacks), fmt.Sprintf("%v/%v", stats.DefensesWon, stats.Defenses),
            fmt.Sprintf("%v/%v", stats.LairsWon, stats.LairsFought), stats.CitiesFounded, stats.CitiesTaken, stats.CitiesLost,
            stats.Think.Round(time.Millisecond), stats.ThinkMost.Round(time.Millisecond))
    }

    if len(ai.ChewbotOrderCounts) > 0 {
        fmt.Fprintf(&out, "\nOrders of Chewbot, units by the step of the original that gave them\n")
        for _, player := range players {
            counts := ai.ChewbotOrderCounts[player.Wizard.Name]
            if len(counts) == 0 {
                continue
            }
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
            fmt.Fprintf(&out, "%-26s %v\n", simName(player), strings.Join(parts, ", "))
        }
    }
    return out.String()
}
