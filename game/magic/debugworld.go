package main

// The World Map entry of the debug list: what the game of it starts with. see fastplay.go

import (
    "image"
    "math/rand/v2"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    gamelib "github.com/kazzmir/master-of-magic/game/magic/game"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

// true while the game of the World Map entry is made and runs
var debugWorldMap = false

// units that stand around the first city of the player, each on a tile of its own
const DebugWorldUnits = 3
// how far from the city they stand at most, in tiles
const DebugWorldUnitReach = 2

// puts the units of the debug game on land around the city, by chance, nearest tiles first
func addDebugWorldUnits(game *gamelib.Game, player *playerlib.Player, cityX int, cityY int, plane data.Plane) {
    mapUse := game.GetMap(plane)
    kinds := startingUnits(player.Wizard.Race)
    if len(kinds) == 0 {
        return
    }

    added := 0
    for reach := 1; reach <= DebugWorldUnitReach && added < DebugWorldUnits; reach++ {
        var tiles []image.Point
        for dx := -reach; dx <= reach; dx++ {
            for dy := -reach; dy <= reach; dy++ {
                if max(dx, -dx) != reach && max(dy, -dy) != reach {
                    continue
                }

                x := mapUse.WrapX(cityX + dx)
                y := cityY + dy
                if y < 0 || y >= mapUse.Height() {
                    continue
                }

                tile := mapUse.GetTile(x, y)
                if !tile.Valid() || !tile.Tile.IsLand() || tile.HasEncounter() || !player.IsExplored(x, y, plane) {
                    continue
                }

                tiles = append(tiles, image.Pt(x, y))
            }
        }

        rand.Shuffle(len(tiles), func(a int, b int) {
            tiles[a], tiles[b] = tiles[b], tiles[a]
        })

        for _, tile := range tiles {
            if added >= DebugWorldUnits {
                break
            }

            kind := kinds[added % len(kinds)]
            unit := player.AddUnit(units.MakeOverworldUnitFromUnit(kind, tile.X, tile.Y, plane, player.Wizard.Banner, player.MakeExperienceInfo(), player.MakeUnitEnchantmentProvider()))
            player.LiftFogSquare(tile.X, tile.Y, unit.GetSightRange(), plane)
            added += 1
        }
    }
}
