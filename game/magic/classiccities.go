package main

// Where the cities of a new game go on the original's worlds (maplib.ClassicMaps). Facts from the
// ReMoM project's reconstruction (MoM/src/MAPGEN.c: Generate_Home_Cities, Generate_Neutral_Cities;
// MoX/src/special.c: Range). The code is ours.
//
//   a wizard's home: 3 to 56 across and 3 to 36 down, on land, 15 or more squares from every other
//     wizard's home on its plane and half of 7 from every node, tower and lair, where a city can
//     grow to 8 (the fourth wizard and later 7); after 1000 tries 1 square less for both, to 10 and 1
//   neutral cities: 15 on each plane, 4 to 57 across and 4 to 37 down, on land, 5 or more from every
//     home (the original's Range: the larger distance and half the smaller) and 4 from every city;
//     1000 tries in all; of the usual race of their landmass three times in four, else of a race by
//     chance; 1 + (difficulty + 1) / 3 + 1 to 4 thousand people (the hardest levels one time in 5
//     (difficulty + 1) / 3 + 1 to 10)
//
// The original tries the homes of all wizards again together when one does not fit; here each
// wizard's is tried on its own.

import (
    "image"
    "math/rand/v2"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/terrain"
    gamelib "github.com/kazzmir/master-of-magic/game/magic/game"
)

// Random(n) of the original: 1 to n
func classicRoll(n int) int {
    return rand.N(max(n, 1)) + 1
}

// Delta_XY_With_Wrap: the larger of the distances across (around the world) and down
func classicDistance(game *gamelib.Game, plane data.Plane, x1 int, y1 int, x2 int, y2 int) int {
    width := game.GetMap(plane).Width()
    dx := x1 - x2
    if dx < 0 {
        dx = -dx
    }
    dx = min(dx, width - dx)
    dy := y1 - y2
    if dy < 0 {
        dy = -dy
    }
    return max(dx, dy)
}

// Range: the larger distance and half the smaller, not around the world
func classicRange(x1 int, y1 int, x2 int, y2 int) int {
    dx := x2 - x1
    if dx < 0 {
        dx = -dx
    }
    dy := y2 - y1
    if dy < 0 {
        dy = -dy
    }
    if dx > dy {
        return dx + dy / 2
    }
    return dy + dx / 2
}

func classicIsLand(game *gamelib.Game, plane data.Plane, x int, y int) bool {
    tile := game.GetMap(plane).GetTile(x, y).Tile
    return tile.IsLand() && !tile.IsMagic()
}

// the homes of the wizards so far
func classicHomes(game *gamelib.Game) []image.Point {
    var out []image.Point
    for _, player := range game.Model.Players {
        if player.IsNeutral() {
            continue
        }
        if fortress := player.FindFortressCity(); fortress != nil {
            out = append(out, image.Pt(fortress.X, fortress.Y))
        }
    }
    return out
}

// Generate_Home_Cities for one wizard, the index-th placed
func classicHomeLocation(game *gamelib.Game, plane data.Plane, index int) (int, int) {
    mapObject := game.GetMap(plane)
    var sites []image.Point
    sites = append(sites, mapObject.GetEncounterLocations()...)
    sites = append(sites, mapObject.GetMagicNodeLocations()...)
    var homes []image.Point
    for _, player := range game.Model.Players {
        if player.IsNeutral() {
            continue
        }
        if fortress := player.FindFortressCity(); fortress != nil && fortress.Plane == plane {
            homes = append(homes, image.Pt(fortress.X, fortress.Y))
        }
    }

    homeDistance := 16
    siteDistance := 8
    for {
        homeDistance = max(homeDistance - 1, 10)
        siteDistance = max(siteDistance - 1, 1)
        for range 1000 {
            x := 2 + classicRoll(mapObject.Width() - 6)
            y := 2 + classicRoll(mapObject.Height() - 6)
            if !classicIsLand(game, plane, x, y) {
                continue
            }
            ok := true
            for _, home := range homes {
                if classicDistance(game, plane, x, y, home.X, home.Y) < homeDistance {
                    ok = false
                }
            }
            for _, site := range sites {
                if classicDistance(game, plane, x, y, site.X, site.Y) < siteDistance / 2 {
                    ok = false
                }
            }
            if !ok {
                continue
            }
            if game.Model.ComputeMaximumPopulation(x, y, plane) < 8 - index / 3 {
                continue
            }
            return x, y
        }
        if homeDistance == 10 && siteDistance == 1 {
            // no room at all: any land will do
            for range 10000 {
                x := classicRoll(mapObject.Width() - 2)
                y := 2 + classicRoll(mapObject.Height() - 6)
                if classicIsLand(game, plane, x, y) && !game.Model.ContainsCity(x, y, plane) {
                    return x, y
                }
            }
            return -1, -1
        }
    }
}

// Generate_Neutral_Cities for one plane: where they go. false when the tries are used up
func classicNeutralLocation(game *gamelib.Game, plane data.Plane, tries *int) (int, int, bool) {
    mapObject := game.GetMap(plane)
    homes := classicHomes(game)
    for *tries < 1000 {
        *tries += 1
        x := 3 + classicRoll(mapObject.Width() - 6)
        y := 3 + classicRoll(mapObject.Height() - 6)
        if !classicIsLand(game, plane, x, y) {
            continue
        }
        ok := true
        for _, home := range homes {
            if classicRange(x, y, home.X, home.Y) < 5 {
                ok = false
            }
        }
        for _, city := range game.Model.AllCities() {
            if city.Plane == plane && classicDistance(game, plane, x, y, city.X, city.Y) < 4 {
                ok = false
            }
        }
        if ok {
            return x, y, true
        }
    }
    return 0, 0, false
}

// the original's five levels, 0 intro to 4 impossible
func classicDifficulty(difficulty data.DifficultySetting) int {
    switch difficulty {
        case data.DifficultyIntro: return 0
        case data.DifficultyEasy: return 1
        case data.DifficultyAverage: return 2
        case data.DifficultyHard: return 3
    }
    return 4
}

// the people of a neutral city, in thousands
func classicNeutralPopulation(difficulty data.DifficultySetting) int {
    level := classicDifficulty(difficulty)
    population := 1 + (level + 1) / 3 + classicRoll(4)
    if level > 2 && classicRoll(5) == 1 {
        population = (level + 1) / 3 + classicRoll(10)
    }
    return population
}

// the races of neutral cities (Generate_Neutral_Cities): Arcanus 13 by chance, High Elves only in a
// forest (the usual race of a landmass takes High Men there), Myrror 5
var classicArcanusRaces = []data.Race{data.RaceBarbarian, data.RaceGnoll, data.RaceHalfling, data.RaceHighElf, data.RaceHighMen, data.RaceKlackon, data.RaceLizard, data.RaceNomad, data.RaceOrc, data.RaceBarbarian, data.RaceHighMen, data.RaceNomad, data.RaceHighMen}
var classicMyrrorRaces = []data.Race{data.RaceBeastmen, data.RaceDarkElf, data.RaceDraconian, data.RaceDwarf, data.RaceTroll}

func classicRace(plane data.Plane, forest bool) data.Race {
    if plane == data.PlaneMyrror {
        return classicMyrrorRaces[classicRoll(len(classicMyrrorRaces)) - 1]
    }
    race := classicArcanusRaces[classicRoll(len(classicArcanusRaces)) - 1]
    if race == data.RaceHighElf && !forest {
        race = data.RaceHighMen
    }
    return race
}

// the race of a neutral city: the usual race of its landmass three times in four
func classicNeutralRace(game *gamelib.Game, plane data.Plane, x int, y int, landRaces map[image.Point]data.Race) data.Race {
    mapObject := game.GetMap(plane)
    forest := mapObject.GetTile(x, y).Tile.TerrainType() == terrain.Forest
    race := classicRace(plane, forest)
    if classicRoll(4) > 1 {
        // a landmass is known by its first square
        key := image.Pt(x, y)
        for _, point := range mapObject.Map.FindContinent(x, y).Values() {
            if point.Y < key.Y || (point.Y == key.Y && point.X < key.X) {
                key = point
            }
        }
        usual, ok := landRaces[key]
        if !ok {
            usual = classicRace(plane, false)
            landRaces[key] = usual
        }
        race = usual
    }
    return race
}
