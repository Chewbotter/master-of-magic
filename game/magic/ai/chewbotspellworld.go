package ai

// What Chewbot's spells look at: the values of cities and stacks as the original computes them for
// its spells every turn (ReMoM MoM/src/AIDUDES.c AI_Player_Calculate_Target_Values). The code is
// ours.
//
//   own city: its people (thousands), twice for Beastmen, Dark Elf, Draconian, Dwarf (the original
//     means Trolls too, and misses them: quirkTrollCityValue), 2 for every building, 2 times the
//     turn for the fortress
//   enemy city: the same for a city of a wizard the AI is hostile to (2 or more, no peace) and for
//     neutral cities, 200 more for another wizard's fortress, 0 for all others
//   garrison: the strength of the AI's units in a city; settlers, engineers, transports count 0
//   stacks: the units of a square outside cities, their strengths added (the original sets the value
//     of an enemy stack to its last unit's strength, a slip of the reconstruction or the original;
//     here the sum)

import (
    "image"
    "slices"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

// the original misses Trolls in the races whose cities count twice (rt_Standard for Trolls)
const quirkTrollCityValue = true

// a square of units of one player
type chewSpellStack struct {
    X, Y int
    Plane data.Plane
    Owner *playerlib.Player
    Units []units.StackUnit
    Value int
}

type chewSpellWorld struct {
    Self *playerlib.Player
    Turn int
    Cities []*citylib.City
    Owners map[*citylib.City]*playerlib.Player
    OwnValue map[*citylib.City]int
    EnemyValue map[*citylib.City]int
    Garrison map[*citylib.City]int
    OwnStacks []chewSpellStack
    EnemyStacks []chewSpellStack
    Hostile []*playerlib.Player
    Human *playerlib.Player
    Players []*playerlib.Player
}

// the strength of a unit for these values: settlers, engineers and transports none
func chewSpellStrength(unit units.StackUnit) int {
    if unit.HasAbility(data.AbilityCreateOutpost) || unit.HasAbility(data.AbilityConstruction) || unit.HasAbility(data.AbilityTransport) {
        return 0
    }
    return chewUnitStrength(unit)
}

func chewCityWorth(city *citylib.City) int {
    value := city.Citizens()
    switch city.Race {
        case data.RaceBeastmen, data.RaceDarkElf, data.RaceDraconian, data.RaceDwarf:
            value *= 2
        case data.RaceTroll:
            if !quirkTrollCityValue {
                value *= 2
            }
    }
    return value + 2 * city.Buildings.Size()
}

// the cities in a fixed order: by plane, row, column (the original's is the order of founding)
func chewSortCities(cities []*citylib.City) {
    slices.SortFunc(cities, func(a *citylib.City, b *citylib.City) int {
        if a.Plane != b.Plane {
            return int(a.Plane) - int(b.Plane)
        }
        if a.Y != b.Y {
            return a.Y - b.Y
        }
        return a.X - b.X
    })
}

// AI_Player_Calculate_Target_Values, as far as the spells use it
func (ai *ChewbotAI) spellWorld(self *playerlib.Player, services playerlib.AIServices) *chewSpellWorld {
    world := &chewSpellWorld{
        Self: self,
        Turn: int(services.GetTurnNumber()),
        Owners: make(map[*citylib.City]*playerlib.Player),
        OwnValue: make(map[*citylib.City]int),
        EnemyValue: make(map[*citylib.City]int),
        Garrison: make(map[*citylib.City]int),
        Hostile: chewHostileOpponents(self, services),
        Human: chewHuman(self, services),
    }
    world.Players = append([]*playerlib.Player{self}, services.GetEnemies(self)...)

    fortress := self.FindFortressCity()
    for _, player := range world.Players {
        for _, city := range player.Cities {
            world.Cities = append(world.Cities, city)
            world.Owners[city] = player
        }
    }
    chewSortCities(world.Cities)

    for _, city := range world.Cities {
        owner := world.Owners[city]
        if owner == self {
            value := chewCityWorth(city)
            if city == fortress {
                value += 2 * world.Turn
            }
            world.OwnValue[city] = value
            continue
        }
        enemy := owner.IsNeutral()
        if relation, ok := self.PlayerRelations[owner]; ok && relation.Hostility >= 2 && relation.PeaceCounter == 0 {
            enemy = true
        }
        if !enemy {
            continue
        }
        value := chewCityWorth(city)
        if !owner.IsNeutral() && owner.FindFortressCity() == city {
            value += 200
        }
        world.EnemyValue[city] = value
    }

    cityAt := func(x int, y int, plane data.Plane) *citylib.City {
        for _, city := range world.Cities {
            if city.X == x && city.Y == y && city.Plane == plane {
                return city
            }
        }
        return nil
    }

    for _, player := range world.Players {
        for _, stack := range player.Stacks {
            if stack.IsEmpty() {
                continue
            }
            value := 0
            for _, unit := range stack.Units() {
                value += chewSpellStrength(unit)
            }
            city := cityAt(stack.X(), stack.Y(), stack.Plane())
            entry := chewSpellStack{X: stack.X(), Y: stack.Y(), Plane: stack.Plane(), Owner: player, Units: stack.Units(), Value: value}
            if player == self {
                if city != nil {
                    world.Garrison[city] += value
                }
                world.OwnStacks = append(world.OwnStacks, entry)
            } else if city == nil {
                world.EnemyStacks = append(world.EnemyStacks, entry)
            }
        }
    }
    return world
}

// the city on a square, nil for none
func (world *chewSpellWorld) cityAt(x int, y int, plane data.Plane) *citylib.City {
    for _, city := range world.Cities {
        if city.X == x && city.Y == y && city.Plane == plane {
            return city
        }
    }
    return nil
}

// the own cities in order
func (world *chewSpellWorld) ownCities() []*citylib.City {
    var out []*citylib.City
    for _, city := range world.Cities {
        if world.Owners[city] == world.Self {
            out = append(out, city)
        }
    }
    return out
}

// the highest valued own city that passes, the first on a tie; nil for none (a value of 0 never wins)
func (world *chewSpellWorld) bestOwnCity(pass func(*citylib.City) bool) *citylib.City {
    var best *citylib.City
    bestValue := 0
    for _, city := range world.ownCities() {
        if world.OwnValue[city] > bestValue && pass(city) {
            best = city
            bestValue = world.OwnValue[city]
        }
    }
    return best
}

// the distance of the original: the larger of across (around the world) and down
func chewSpellDistance(width int, x1 int, y1 int, x2 int, y2 int) int {
    dx := x1 - x2
    if dx < 0 {
        dx = -dx
    }
    if width > 0 {
        dx = min(dx, width - dx)
    }
    dy := y1 - y2
    if dy < 0 {
        dy = -dy
    }
    return max(dx, dy)
}

func chewPoint(x int, y int) image.Point {
    return image.Pt(x, y)
}
