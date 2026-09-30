package game

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/maplib"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/terrain"
    "github.com/kazzmir/master-of-magic/game/magic/units"
)

func movementStack(kinds ...units.Unit) *playerlib.UnitStack {
    var all []units.StackUnit
    for _, kind := range kinds {
        all = append(all, units.MakeOverworldUnit(kind, 5, 5, data.PlaneArcanus))
    }
    return playerlib.MakeUnitStackFromUnits(all)
}

// TERRSTAT: the costs in half moves by the modes of the stack
func TestClassicStepCost(test *testing.T) {
    arcanus := &maplib.Map{Plane: data.PlaneArcanus}
    myrror := &maplib.Map{Plane: data.PlaneMyrror}
    tile := func(kind terrain.Tile) maplib.FullTile {
        return maplib.FullTile{Tile: kind}
    }

    walkers := movementStack(units.HighMenSwordsmen)
    foresters := movementStack(units.HighMenSwordsmen, units.WarBear)
    mountaineers := movementStack(units.HighMenSwordsmen, units.FireGiant)

    for _, check := range []struct {
        Name string
        Stack *playerlib.UnitStack
        Map *maplib.Map
        Tile maplib.FullTile
        Road bool
        Want int
    }{
        {"walkers on grass", walkers, arcanus, tile(terrain.TileGrasslands1), false, 2},
        {"walkers in forest", walkers, arcanus, tile(terrain.TileForest1), false, 4},
        {"walkers on mountains", walkers, arcanus, tile(terrain.TileMountain1), false, 8},
        {"a forester in forest", foresters, arcanus, tile(terrain.TileForest1), false, 2},
        {"a mountaineer on mountains", mountaineers, arcanus, tile(terrain.TileMountain1), false, 2},
        {"a mountaineer on grass (the original's)", mountaineers, arcanus, tile(terrain.TileGrasslands1), false, 6},
        {"a road on Arcanus", walkers, arcanus, tile(terrain.TileHills1), true, 1},
        {"a road on Myrror", walkers, myrror, tile(terrain.TileHills1), true, 0},
    } {
        if got := classicStepCost(check.Stack, check.Map, check.Tile, check.Road, false); got != check.Want {
            test.Errorf("%v: %v half moves, want %v", check.Name, got, check.Want)
        }
    }
}

// the stack limit of 9 and the seats of ships
func TestClassicRoom(test *testing.T) {
    five := movementStack(units.HighMenSwordsmen, units.HighMenSwordsmen, units.HighMenSwordsmen, units.HighMenSwordsmen, units.HighMenSwordsmen)
    four := movementStack(units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen)
    if !classicRoomAt(four, five, false) {
        test.Errorf("4 and 5 make 9, which fits")
    }
    if classicRoomAt(movementStack(units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen, units.HighMenSpearmen), five, false) {
        test.Errorf("5 and 5 are more than 9")
    }

    // a trireme seats 2
    boat := movementStack(units.Trireme, units.HighMenSwordsmen)
    if !classicRoomAt(movementStack(units.HighMenSpearmen), boat, true) {
        test.Errorf("a second rider fits a trireme")
    }
    if classicRoomAt(movementStack(units.HighMenSpearmen, units.HighMenSpearmen), boat, true) {
        test.Errorf("three riders do not fit a trireme")
    }
}
