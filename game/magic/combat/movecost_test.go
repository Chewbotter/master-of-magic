package combat

import (
    "image"
    "testing"

    "github.com/kazzmir/master-of-magic/lib/fraction"
)

// the original's costs of the ground, in moves
func TestStepCostsOfTheGround(test *testing.T) {
    ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, ZoneType{})
    for index := range ground.Trees {
        ground.Trees[index] = 0
    }
    model := &CombatModel{
        Tiles: makeTiles(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, ZoneType{}),
        Ground: ground,
    }

    from := image.Pt(10, 10)
    to := image.Pt(10, 11)
    diagonal := image.Pt(11, 11)
    cgx, cgy := TileToCell(to.X, to.Y)

    set := func(group TerrainGroup, road bool, mud bool, trees int) {
        model.Tiles[to.Y][to.X].Ground = group
        model.Tiles[to.Y][to.X].Road = road
        model.Tiles[to.Y][to.X].Mud = mud
        ground.Trees[ground.index(cgx, cgy)] = trees
    }

    cases := []struct {
        Name string
        Group TerrainGroup
        Road bool
        Mud bool
        Trees int
        Flying bool
        Want fraction.Fraction
    }{
        {Name: "grass", Group: TerrainGrass, Want: fraction.FromInt(1)},
        {Name: "dirt", Group: TerrainDirt, Want: fraction.FromInt(1)},
        {Name: "rough", Group: TerrainRough, Want: fraction.FromInt(2)},
        {Name: "road", Group: TerrainGrass, Road: true, Want: fraction.Make(1, 2)},
        {Name: "a tree", Group: TerrainGrass, Trees: 1, Want: fraction.Make(3, 2)},
        {Name: "three trees", Group: TerrainGrass, Trees: 3, Want: fraction.FromInt(2)},
        {Name: "mud", Group: TerrainGrass, Mud: true, Want: fraction.FromInt(6)},
        {Name: "flying over rough", Group: TerrainRough, Flying: true, Want: fraction.FromInt(1)},
        {Name: "flying over mud", Group: TerrainGrass, Mud: true, Flying: true, Want: fraction.FromInt(1)},
    }

    for _, testCase := range cases {
        set(testCase.Group, testCase.Road, testCase.Mud, testCase.Trees)
        got := model.StepCost(from, to, testCase.Flying)
        if !got.Equals(testCase.Want) {
            test.Fatalf("%v: %v, want %v", testCase.Name, got, testCase.Want)
        }
    }

    // a diagonal step costs half a move more
    set(TerrainRough, false, false, 0)
    model.Tiles[diagonal.Y][diagonal.X].Ground = TerrainRough
    if got := model.StepCost(from, diagonal, false); !got.Equals(fraction.Make(5, 2)) {
        test.Fatalf("diagonal into rough: %v, want 5/2", got)
    }
}

// earth to mud leaves rough ground as it is
func TestNoMudOnRough(test *testing.T) {
    model := &CombatModel{
        Tiles: makeTiles(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, ZoneType{}),
    }
    model.Tiles[10][10].Ground = TerrainRough
    model.CreateEarthToMud(10, 10)
    if model.Tiles[10][10].Mud || !model.Tiles[10][11].Mud {
        test.Fatalf("mud on rough %v, mud next to it %v", model.Tiles[10][10].Mud, model.Tiles[10][11].Mud)
    }
}
