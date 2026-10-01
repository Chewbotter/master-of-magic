package game

// Development: what happens in a game that runs without a window (-sim, see sim.go of the main
// package), counted for its summary. Game.Stats is nil in every other game, and then nothing is
// counted.

import (
    "sync"
    "time"

    "github.com/kazzmir/master-of-magic/game/magic/combat"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

// what one player did
type SimPlayerStats struct {
    Attacks int
    AttacksWon int
    Defenses int
    DefensesWon int
    LairsFought int
    LairsWon int
    CitiesFounded int
    CitiesTaken int
    CitiesLost int
    Think time.Duration
    ThinkMost time.Duration
}

type SimStats struct {
    lock sync.Mutex
    Players map[*playerlib.Player]*SimPlayerStats
    // what computer wizards said to the human, by the original's action number
    Messages map[int]int
    // the turn two wizards first knew of each other
    Contacts []SimContact
    contactSeen map[[2]*playerlib.Player]bool
    // banishments and defeats (conquest.go), and how the game ended
    Conquests []string
    GameOver string
    // broken states found by the checks of every turn (simcheck.go)
    Violations map[string]*SimViolation
    // a panic of the AI's goroutine, with its stack: the run stops
    Panic string
}

func (stats *SimStats) noteConquest(text string) {
    playerlib.Note(nil, "conquest", text, "")
    if stats == nil {
        return
    }
    stats.lock.Lock()
    defer stats.lock.Unlock()
    stats.Conquests = append(stats.Conquests, text)
}

type SimContact struct {
    First *playerlib.Player
    Second *playerlib.Player
    Turn uint64
}

// notes the pairs of wizards that know of each other and were not noted before
func (stats *SimStats) noteContacts(players []*playerlib.Player, turn uint64) {
    if stats == nil {
        return
    }
    stats.lock.Lock()
    defer stats.lock.Unlock()
    if stats.contactSeen == nil {
        stats.contactSeen = make(map[[2]*playerlib.Player]bool)
    }
    for index, first := range players {
        for _, second := range players[index + 1:] {
            if first.IsNeutral() || second.IsNeutral() || stats.contactSeen[[2]*playerlib.Player{first, second}] {
                continue
            }
            if first.IsAwareOf(second) || second.IsAwareOf(first) {
                stats.contactSeen[[2]*playerlib.Player{first, second}] = true
                stats.Contacts = append(stats.Contacts, SimContact{First: first, Second: second, Turn: turn})
            }
        }
    }
}

func MakeSimStats() *SimStats {
    return &SimStats{Players: make(map[*playerlib.Player]*SimPlayerStats), Messages: make(map[int]int)}
}

func (stats *SimStats) Of(player *playerlib.Player) *SimPlayerStats {
    stats.lock.Lock()
    defer stats.lock.Unlock()
    out, ok := stats.Players[player]
    if !ok {
        out = &SimPlayerStats{}
        stats.Players[player] = out
    }
    return out
}

func (stats *SimStats) combat(attacker *playerlib.Player, defender *playerlib.Player, state combat.CombatState) {
    if stats == nil || attacker == nil || defender == nil {
        return
    }
    stats.Of(attacker).Attacks += 1
    stats.Of(defender).Defenses += 1
    switch state {
        case combat.CombatStateAttackerWin, combat.CombatStateDefenderFlee:
            stats.Of(attacker).AttacksWon += 1
        case combat.CombatStateDefenderWin, combat.CombatStateAttackerFlee:
            stats.Of(defender).DefensesWon += 1
    }
}

func (stats *SimStats) encounter(player *playerlib.Player, state combat.CombatState) {
    if stats == nil || player == nil {
        return
    }
    stats.Of(player).LairsFought += 1
    if state == combat.CombatStateAttackerWin || state == combat.CombatStateDefenderFlee {
        stats.Of(player).LairsWon += 1
    }
}

func (stats *SimStats) cityTaken(attacker *playerlib.Player, defender *playerlib.Player) {
    if stats == nil {
        return
    }
    stats.Of(attacker).CitiesTaken += 1
    stats.Of(defender).CitiesLost += 1
}

func (stats *SimStats) cityFounded(player *playerlib.Player) {
    if stats == nil {
        return
    }
    stats.Of(player).CitiesFounded += 1
}

func (stats *SimStats) thought(player *playerlib.Player, elapsed time.Duration) {
    if stats == nil {
        return
    }
    own := stats.Of(player)
    stats.lock.Lock()
    defer stats.lock.Unlock()
    own.Think += elapsed
    own.ThinkMost = max(own.ThinkMost, elapsed)
}

// the human skips its turns and nobody waits for it (see CaptureSkipTurns)
func (game *Game) SimSkipHuman() {
    game.captureSkipping = true
    game.headless = true
    if human := game.Model.GetHumanPlayer(); human != nil {
        human.Skip = true
    }
}

// a panic of the AI's goroutine: the first is kept
func (stats *SimStats) notePanic(text string) {
    if stats == nil {
        return
    }
    stats.lock.Lock()
    defer stats.lock.Unlock()
    if stats.Panic == "" {
        stats.Panic = text
    }
}

// the panic of the run, empty when none
func (stats *SimStats) PanicText() string {
    if stats == nil {
        return ""
    }
    stats.lock.Lock()
    defer stats.lock.Unlock()
    return stats.Panic
}

// a panic of the game's own loop, noted by the runner
func (stats *SimStats) NotePanic(text string) {
    stats.notePanic(text)
}
