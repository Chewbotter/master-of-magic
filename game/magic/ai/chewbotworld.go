package ai

// The world as the overland AI of Chewbot sees it in one turn: landmasses, their coasts, and the
// evaluation map (the strength of what is not the wizard's own on every square). Facts from the
// ReMoM project's reconstruction (MoM/src/AIMOVE.c: AI_Evaluation_Map; MoM/src/AIDUDES.c: the
// lists of land and dock squares; MoM/src/NEXTTURN.c: Effective_Unit_Strength,
// Effective_Unit_Type_Strength; MoX/src/special.c: Delta_XY_With_Wrap). The code is ours. The rules
// in words: docs/mod/ai-overland.md.

import (
    "fmt"
    "cmp"
    "image"
    "log"
    "slices"

    "github.com/kazzmir/master-of-magic/game/magic/combat"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/maplib"
    "github.com/kazzmir/master-of-magic/game/magic/terrain"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

// the flags of a square of the evaluation map (AI_TARGET_...)
const (
    chewEvalStrength = 0x3FFF
    chewEvalNonHostile = 0x4000
    chewEvalSite = 0x8000
)

// a city with its owner
type chewCity struct {
    City *citylib.City
    Owner *playerlib.Player
}

// one turn's view of the world of a computer wizard (the original's CONTXXX data)
type chewWorld struct {
    Self *playerlib.Player
    Services playerlib.AIServices
    Maps [2]*maplib.Map
    Width int
    Height int
    // landmass of every square, y * Width + x: 0 is water, 1 and on are the landmasses in the order
    // of their first square, row by row
    Landmass [2][]int
    // the squares of every landmass and the ones of them next to the sea, row by row
    LandSquares [2][][]image.Point
    DockSquares [2][][]image.Point
    Eval [2][]int
    // every player that is not defeated, the wizard itself too, and their cities
    Players []*playerlib.Player
    Cities []chewCity
    Turn int
}

func chewPlaneIndex(plane data.Plane) int {
    if plane == data.PlaneMyrror {
        return 1
    }
    return 0
}

func chewPlaneOf(index int) data.Plane {
    if index == 1 {
        return data.PlaneMyrror
    }
    return data.PlaneArcanus
}

// Delta_XY_With_Wrap: the larger of the distances across and along, across the shorter way round
func chewDistance(x1 int, y1 int, x2 int, y2 int, width int) int {
    across := x2 - x1
    if across < 0 {
        across = -across
    }
    if width - across < across {
        across = width - across
    }
    along := y2 - y1
    if along < 0 {
        along = -along
    }
    return max(across, along)
}

func (world *chewWorld) Distance(x1 int, y1 int, x2 int, y2 int) int {
    return chewDistance(x1, y1, x2, y2, world.Width)
}

func (world *chewWorld) WrapX(x int) int {
    return ((x % world.Width) + world.Width) % world.Width
}

func (world *chewWorld) inside(y int) bool {
    return y >= 0 && y < world.Height
}

// the landmass of a square, 0 for water and outside the map
func (world *chewWorld) LandmassAt(x int, y int, wp int) int {
    if !world.inside(y) {
        return 0
    }
    return world.Landmass[wp][y * world.Width + world.WrapX(x)]
}

func (world *chewWorld) EvalAt(x int, y int, wp int) int {
    if !world.inside(y) {
        return 0
    }
    return world.Eval[wp][y * world.Width + world.WrapX(x)]
}

func (world *chewWorld) addEval(x int, y int, wp int, amount int) {
    index := y * world.Width + world.WrapX(x)
    value := world.Eval[wp][index]
    strength := min(chewEvalStrength, (value & chewEvalStrength) + amount)
    world.Eval[wp][index] = (value &^ chewEvalStrength) | strength
}

func (world *chewWorld) flagEval(x int, y int, wp int, flag int) {
    world.Eval[wp][y * world.Width + world.WrapX(x)] |= flag
}

// the sea as the original's landmass 0 knows it: water that is not a lake
func chewIsSea(tile maplib.FullTile) bool {
    return tile.Tile.IsWater() && tile.Tile.TerrainType() != terrain.Lake
}

// the world without its evaluation map, which wants the hostility of this turn
func makeChewWorld(self *playerlib.Player, services playerlib.AIServices) *chewWorld {
    world := &chewWorld{
        Self: self,
        Services: services,
        Turn: int(services.GetTurnNumber()),
    }
    world.Players = append(world.Players, self)
    world.Players = append(world.Players, services.GetEnemies(self)...)
    for _, player := range world.Players {
        for _, city := range player.Cities {
            world.Cities = append(world.Cities, chewCity{City: city, Owner: player})
        }
    }

    for wp := range 2 {
        world.Maps[wp] = services.GetMap(chewPlaneOf(wp))
    }
    world.Width = world.Maps[0].Width()
    world.Height = world.Maps[0].Height()

    for wp := range 2 {
        world.findLandmasses(wp)
    }
    return world
}

// the landmasses: land squares that touch, the corners too; and their dock squares, the land
// squares with sea among their 8 neighbors (Build_Dock_Linked_List)
func (world *chewWorld) findLandmasses(wp int) {
    mapObject := world.Maps[wp]
    labels := make([]int, world.Width * world.Height)
    world.LandSquares[wp] = [][]image.Point{nil}
    world.DockSquares[wp] = [][]image.Point{nil}

    isLand := func(x int, y int) bool {
        return mapObject.GetTile(x, y).Tile.IsLand()
    }

    count := 0
    for y := range world.Height {
        for x := range world.Width {
            if labels[y * world.Width + x] != 0 || !isLand(x, y) {
                continue
            }
            count += 1
            labels[y * world.Width + x] = count
            search := []image.Point{image.Pt(x, y)}
            for len(search) > 0 {
                point := search[len(search) - 1]
                search = search[:len(search) - 1]
                for dy := -1; dy <= 1; dy++ {
                    for dx := -1; dx <= 1; dx++ {
                        nx := world.WrapX(point.X + dx)
                        ny := point.Y + dy
                        if !world.inside(ny) || labels[ny * world.Width + nx] != 0 || !isLand(nx, ny) {
                            continue
                        }
                        labels[ny * world.Width + nx] = count
                        search = append(search, image.Pt(nx, ny))
                    }
                }
            }
        }
    }

    world.Landmass[wp] = labels
    for range count {
        world.LandSquares[wp] = append(world.LandSquares[wp], nil)
        world.DockSquares[wp] = append(world.DockSquares[wp], nil)
    }

    for y := range world.Height {
        for x := range world.Width {
            landmass := labels[y * world.Width + x]
            if landmass == 0 {
                continue
            }
            world.LandSquares[wp][landmass] = append(world.LandSquares[wp][landmass], image.Pt(x, y))

            shore := false
            for dy := -1; dy <= 1 && !shore; dy++ {
                for dx := -1; dx <= 1 && !shore; dx++ {
                    if world.inside(y + dy) && chewIsSea(mapObject.GetTile(world.WrapX(x + dx), y + dy)) {
                        shore = true
                    }
                }
            }
            if shore {
                world.DockSquares[wp][landmass] = append(world.DockSquares[wp][landmass], image.Pt(x, y))
            }
        }
    }
}

// the number of landmasses of a plane
func (world *chewWorld) LandmassCount(wp int) int {
    return len(world.LandSquares[wp]) - 1
}

// a kind of unit, the guardians of a lair: what Effective_Unit_Type_Strength reads
type chewUnitKind struct {
    *units.Unit
}

func (kind chewUnitKind) HasEnchantment(data.UnitEnchantment) bool {
    return false
}

// Effective_Unit_Strength
func chewUnitStrength(unit units.StackUnit) int {
    return combat.ChewbotUnitStrength(unit, unit.VisibleFigures(), unit.GetHealth())
}

// Effective_Unit_Type_Strength
func chewKindStrength(unit units.Unit) int {
    kind := chewUnitKind{Unit: &unit}
    return combat.ChewbotUnitStrength(kind, unit.GetCount(), unit.GetMaxHealth())
}

// the strength of the guardians of a lair, a node or a tower, in tenths
func chewGuardianStrength(encounter *maplib.ExtraEncounter) int {
    strength := 0
    for _, unit := range encounter.Units {
        strength += chewKindStrength(unit) / 10
    }
    return strength
}

// the encounters of a map in a fixed order
func chewEncounterPoints(mapObject *maplib.Map) []image.Point {
    return chewSortPoints(mapObject.GetEncounterLocations())
}

func chewNodePoints(mapObject *maplib.Map) []image.Point {
    return chewSortPoints(mapObject.GetMagicNodeLocations())
}

func chewTowerPoints(world *chewWorld, wp int) []image.Point {
    var points []image.Point
    mapObject := world.Maps[wp]
    points = append(points, mapObject.GetOpenTowerLocations()...)
    for _, point := range mapObject.GetEncounterLocations() {
        if mapObject.GetEncounter(point.X, point.Y).Type == maplib.EncounterTypePlaneTower {
            points = append(points, point)
        }
    }
    return chewSortPoints(points)
}

func chewSortPoints(points []image.Point) []image.Point {
    points = slices.Clone(points)
    slices.SortFunc(points, func(a image.Point, b image.Point) int {
        if a.Y != b.Y {
            return cmp.Compare(a.Y, b.Y)
        }
        return cmp.Compare(a.X, b.X)
    })
    return points
}

// AI_Evaluation_Map: every unit of another player adds a tenth of its strength to its square, with
// a flag when its owner is not hostile; the guardians of a lair, node or tower add theirs; cities,
// lairs, nodes and towers are marked as sites, and a city that is not the wizard's own adds 1
func (world *chewWorld) makeEvaluationMap(hostility func(*playerlib.Player) int) {
    for wp := range 2 {
        world.Eval[wp] = make([]int, world.Width * world.Height)
    }

    for _, player := range world.Players {
        if player == world.Self {
            continue
        }
        nonHostile := chewNonHostile(world.Self, player, hostility)
        for _, stack := range player.Stacks {
            if !world.inside(stack.Y()) {
                // a stack off the map (seen once in a long run, cause not found): no square to mark
                log.Printf("chewbot: stack of %v off the map at %v,%v: %v", player.Wizard.Name, stack.X(), stack.Y(), stack.Units())
                continue
            }
            wp := chewPlaneIndex(stack.Plane())
            for _, unit := range stack.Units() {
                world.addEval(stack.X(), stack.Y(), wp, chewUnitStrength(unit) / 10)
            }
            if nonHostile && len(stack.Units()) > 0 {
                world.flagEval(stack.X(), stack.Y(), wp, chewEvalNonHostile)
            }
        }
    }

    for wp := range 2 {
        mapObject := world.Maps[wp]
        for _, point := range chewEncounterPoints(mapObject) {
            world.addEval(point.X, point.Y, wp, chewGuardianStrength(mapObject.GetEncounter(point.X, point.Y)))
            world.flagEval(point.X, point.Y, wp, chewEvalSite)
        }
        for _, point := range chewNodePoints(mapObject) {
            world.flagEval(point.X, point.Y, wp, chewEvalSite)
        }
        for _, point := range mapObject.GetOpenTowerLocations() {
            world.flagEval(point.X, point.Y, wp, chewEvalSite)
        }
    }

    for _, city := range world.Cities {
        wp := chewPlaneIndex(city.City.Plane)
        world.flagEval(city.City.X, city.City.Y, wp, chewEvalSite)
        if city.Owner != world.Self {
            world.addEval(city.City.X, city.City.Y, wp, 1)
        }
    }
}

// a player the wizard does not count as hostile: a pact, an alliance, peace, or hostility below 2
func chewNonHostile(self *playerlib.Player, other *playerlib.Player, hostility func(*playerlib.Player) int) bool {
    if relation, ok := self.PlayerRelations[other]; ok {
        if relation.Treaty == data.TreatyPact || relation.Treaty == data.TreatyAlliance || relation.PeaceCounter > 0 {
            return true
        }
    }
    return hostility(other) < 2
}

// the city on a square
func (world *chewWorld) CityAt(x int, y int, wp int) (chewCity, bool) {
    x = world.WrapX(x)
    for _, city := range world.Cities {
        if city.City.X == x && city.City.Y == y && chewPlaneIndex(city.City.Plane) == wp {
            return city, true
        }
    }
    return chewCity{}, false
}

// the fortress of a player, if it has one
func chewFortress(player *playerlib.Player) *citylib.City {
    return player.FindFortressCity()
}

// what stands on a square, for the journal of -sim: a city and its owner, a lair, a node, or the
// stack of a player
func (world *chewWorld) describe(x int, y int, wp int) string {
    plane := chewPlaneOf(wp)
    for _, player := range world.Players {
        if city := player.FindCity(x, y, plane); city != nil {
            return fmt.Sprintf("the city %v of %v", city.Name, player.Wizard.Name)
        }
    }
    if mapObject := world.Maps[wp]; mapObject != nil {
        if encounter := mapObject.GetEncounter(x, y); encounter != nil {
            return fmt.Sprintf("a %v", encounter.Type.Name())
        }
        if mapObject.GetMagicNode(x, y) != nil {
            return "a node"
        }
    }
    for _, player := range world.Players {
        if stack := player.FindStack(x, y, plane); stack != nil {
            return fmt.Sprintf("a stack of %v units of %v", len(stack.Units()), player.Wizard.Name)
        }
    }
    return "a site"
}
