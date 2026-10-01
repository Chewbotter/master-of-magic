package game

// Development: checks of a game's state that should always hold, run once a turn by a game without
// a window (-sim, see sim.go of the main package). What breaks is counted by kind with the first
// turn and a few examples, for the summary and the batch reports. Nothing here changes the game.

import (
    "fmt"
    "sort"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    herolib "github.com/kazzmir/master-of-magic/game/magic/hero"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// how many examples of a kind are kept
const simCheckExamples = 5

// one kind of broken state in a run
type SimViolation struct {
    Kind string
    Count int
    FirstTurn uint64
    LastTurn uint64
    Examples []string
}

func (stats *SimStats) violation(turn uint64, kind string, example string) {
    if stats == nil {
        return
    }
    stats.lock.Lock()
    defer stats.lock.Unlock()
    if stats.Violations == nil {
        stats.Violations = make(map[string]*SimViolation)
    }
    found, ok := stats.Violations[kind]
    if !ok {
        found = &SimViolation{Kind: kind, FirstTurn: turn}
        stats.Violations[kind] = found
    }
    found.Count += 1
    found.LastTurn = turn
    if len(found.Examples) < simCheckExamples {
        found.Examples = append(found.Examples, fmt.Sprintf("turn %v: %v", turn, example))
    }
}

// the broken states of the run, the most frequent first
func (stats *SimStats) ViolationList() []SimViolation {
    if stats == nil {
        return nil
    }
    stats.lock.Lock()
    defer stats.lock.Unlock()
    var out []SimViolation
    for _, found := range stats.Violations {
        out = append(out, *found)
    }
    sort.Slice(out, func(i int, j int) bool {
        if out[i].Count != out[j].Count {
            return out[i].Count > out[j].Count
        }
        return out[i].Kind < out[j].Kind
    })
    return out
}

type simSquare struct {
    X int
    Y int
    Plane data.Plane
}

// checks the state of the game once; called by the runner at every new turn
func (game *Game) SimCheck() {
    stats := game.Stats
    if stats == nil {
        return
    }
    turn := game.Model.TurnNumber
    note := func(kind string, format string, args ...any) {
        stats.violation(turn, kind, fmt.Sprintf(format, args...))
    }

    stackOwner := make(map[simSquare]*playerlib.Player)
    unitSeen := make(map[units.StackUnit]bool)
    citySquares := make(map[simSquare]string)

    for _, city := range game.Model.AllCities() {
        square := simSquare{city.X, city.Y, city.Plane}
        if other, taken := citySquares[square]; taken {
            note("two cities on one square", "%v and %v at %v,%v", other, city.Name, city.X, city.Y)
        }
        citySquares[square] = city.Name
        if city.Population < 0 {
            note("city with negative population", "%v: %v", city.Name, city.Population)
        }
        mapUse := game.GetMap(city.Plane)
        if mapUse != nil && (city.Y < 0 || city.Y >= mapUse.Height() || city.X < 0 || city.X >= mapUse.Width()) {
            note("city off the map", "%v at %v,%v", city.Name, city.X, city.Y)
        } else if mapUse != nil && mapUse.GetTile(city.X, city.Y).Tile.IsWater() {
            note("city on water", "%v at %v,%v", city.Name, city.X, city.Y)
        }
    }

    for _, player := range game.Model.Players {
        name := player.Wizard.Name
        if player.Gold < 0 {
            note("gold below 0", "%v has %v", name, player.Gold)
        }
        if player.Mana < 0 {
            note("mana below 0", "%v has %v", name, player.Mana)
        }
        if player.Defeated && !player.IsNeutral() {
            if len(player.Cities) > 0 {
                note("defeated wizard keeps cities", "%v has %v", name, len(player.Cities))
            }
            count := 0
            for range player.Units() {
                count += 1
            }
            if count > 0 && !player.IsHuman() {
                note("defeated wizard keeps units", "%v has %v", name, count)
            }
        }
        for _, city := range player.Cities {
            if city.GetBanner() != player.GetBanner() {
                note("city of the wrong banner", "%v of %v shows %v", city.Name, name, city.GetBanner())
            }
        }

        for _, stack := range player.Stacks {
            stackUnits := stack.Units()
            if len(stackUnits) == 0 {
                note("empty stack", "%v", name)
                continue
            }
            x, y, plane := stack.X(), stack.Y(), stack.Plane()
            mapUse := game.GetMap(plane)
            if mapUse == nil {
                continue
            }
            if y < 0 || y >= mapUse.Height() || x < 0 || x >= mapUse.Width() {
                note("stack off the map", "%v: %v units at %v,%v", name, len(stackUnits), x, y)
                continue
            }
            if len(stackUnits) > data.MaxUnitsInStack {
                note("more than 9 units on a square", "%v: %v units at %v,%v", name, len(stackUnits), x, y)
            }
            square := simSquare{x, y, plane}
            if other, taken := stackOwner[square]; taken && other != player {
                note("two players on one square", "%v and %v at %v,%v", other.Wizard.Name, name, x, y)
            }
            stackOwner[square] = player
            if city, owner := game.Model.FindCity(x, y, plane); city != nil && owner != player {
                note("stack in another's city", "%v in %v of %v", name, city.Name, owner.Wizard.Name)
            }

            water := mapUse.GetTile(x, y).Tile.IsWater()
            carried := false
            for _, unit := range stackUnits {
                if unit.GetAbilityValue(data.AbilityTransport) > 0 || unit.HasAbility(data.AbilityWindWalking) {
                    carried = true
                }
            }
            for _, unit := range stackUnits {
                if unitSeen[unit] {
                    note("unit in two stacks", "%v of %v", unit.GetName(), name)
                }
                unitSeen[unit] = true
                if unit.GetX() != x || unit.GetY() != y || unit.GetPlane() != plane {
                    note("unit away from its stack", "%v of %v at %v,%v, its stack at %v,%v", unit.GetName(), name, unit.GetX(), unit.GetY(), x, y)
                }
                if unit.GetHealth() <= 0 {
                    note("dead unit on the map", "%v of %v at %v,%v", unit.GetName(), name, x, y)
                }
                if water && !carried && unit.IsLandWalker() && !simCityAt(game, x, y, plane) {
                    note("walker on open water", "%v of %v at %v,%v", unit.GetName(), name, x, y)
                }
            }
        }

        for _, hero := range player.Heroes {
            if hero == nil || hero.Status != herolib.StatusEmployed {
                continue
            }
            if !unitSeen[hero] && player.FindStackByUnit(hero) == nil {
                note("hired hero in no stack", "%v of %v", hero.Name, name)
            }
        }
    }
}

// a city of anyone on the square
func simCityAt(game *Game, x int, y int, plane data.Plane) bool {
    city, _ := game.Model.FindCity(x, y, plane)
    return city != nil
}
