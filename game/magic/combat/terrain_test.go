package combat

import (
    "testing"

    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
)

func testGround(width int, height int) *BattleGround {
    return &BattleGround{
        Width: width,
        Height: height,
        Group: make([]TerrainGroup, width * height),
        Picture: make([]int, width * height),
        Roads: make([]int, width * height),
        Trees: make([]int, width * height),
    }
}

// grass shows the edge of the dirt next to it, as Set_Terrain_Tile_Types picks it
func TestGrassShowsTheEdgeOfDirt(test *testing.T) {
    cases := []struct {
        Name string
        DirtX []int
        DirtY []int
        First int
    }{
        {Name: "none", First: 0},
        {Name: "right", DirtX: []int{3}, DirtY: []int{2}, First: 30},
        {Name: "right and up", DirtX: []int{3, 2}, DirtY: []int{2, 1}, First: 28},
        {Name: "right and down", DirtX: []int{3, 2}, DirtY: []int{2, 3}, First: 12},
        {Name: "up", DirtX: []int{2}, DirtY: []int{1}, First: 24},
        {Name: "up and left", DirtX: []int{2, 1}, DirtY: []int{1, 2}, First: 22},
        {Name: "left", DirtX: []int{1}, DirtY: []int{2}, First: 18},
        {Name: "left and down", DirtX: []int{1, 2}, DirtY: []int{2, 3}, First: 16},
        {Name: "down", DirtX: []int{2}, DirtY: []int{3}, First: 10},
        {Name: "down right corner", DirtX: []int{3}, DirtY: []int{3}, First: 8},
        {Name: "up right corner", DirtX: []int{3}, DirtY: []int{1}, First: 26},
        {Name: "up left corner", DirtX: []int{1}, DirtY: []int{1}, First: 20},
        {Name: "down left corner", DirtX: []int{1}, DirtY: []int{3}, First: 14},
    }

    for _, testCase := range cases {
        ground := testGround(5, 5)
        for index := range testCase.DirtX {
            ground.setGroup(testCase.DirtX[index], testCase.DirtY[index], TerrainDirt)
        }

        last := testCase.First + 1
        if testCase.Name == "none" {
            last = 3
        }
        for range 20 {
            picture := ground.pictureOf(2, 2)
            if picture < testCase.First || picture > last {
                test.Fatalf("%v: picture %v, want %v to %v", testCase.Name, picture, testCase.First, last)
            }
        }
    }
}

// rough ground shows which of its sides go on as rough: the pictures 32 to 47
func TestRoughFollowsItsNeighbors(test *testing.T) {
    ground := testGround(5, 5)
    ground.setGroup(2, 2, TerrainRough)
    if picture := ground.pictureOf(2, 2); picture != 47 {
        test.Fatalf("single rough: %v, want 47", picture)
    }

    ground.setGroup(2, 3, TerrainRough)
    if picture := ground.pictureOf(2, 2); picture != 32 {
        test.Fatalf("rough below: %v, want 32", picture)
    }

    ground.setGroup(2, 1, TerrainRough)
    ground.setGroup(1, 2, TerrainRough)
    ground.setGroup(3, 2, TerrainRough)
    if picture := ground.pictureOf(2, 2); picture != 36 {
        test.Fatalf("rough on all sides: %v, want 36", picture)
    }
}

// no rough next to dirt, on a road or in a town, and a road runs from the town to the edge
func TestGroundKeepsTheOriginalsRules(test *testing.T) {
    zone := ZoneType{City: &citylib.City{}}
    zone.Ground.Roads[4] = true
    zone.Ground.Roads[5] = true

    for range 20 {
        ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeMountain, zone)

        reachesEdge := false
        for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
            for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
                picture := ground.Picture[ground.index(cgx, cgy)]
                if picture < 0 || picture > 47 {
                    test.Fatalf("picture %v at %v, %v", picture, cgx, cgy)
                }

                if ground.GroupAt(cgx, cgy) != TerrainRough {
                    continue
                }
                if ground.RoadAt(cgx, cgy) != 0 {
                    test.Fatalf("rough on a road at %v, %v", cgx, cgy)
                }
                if cgx >= townCellMinX && cgx <= townCellMaxX && cgy >= townCellMinY && cgy <= townCellMaxY {
                    test.Fatalf("rough in the town at %v, %v", cgx, cgy)
                }
                for dy := -1; dy <= 1; dy++ {
                    for dx := -1; dx <= 1; dx++ {
                        if ground.GroupAt(cgx + dx, cgy + dy) == TerrainDirt {
                            test.Fatalf("rough next to dirt at %v, %v", cgx, cgy)
                        }
                    }
                }
            }
            if ground.RoadAt(ground.MinX + ground.Width - 1, cgy) != 0 {
                reachesEdge = true
            }
        }

        // the cell a road starts on can lose its road as in the original: with no pair of neighbors
        // it only has bit 0, which the last pass clears for half of the roads
        if !reachesEdge {
            test.Fatalf("the road east does not reach the edge")
        }
    }
}

// without a road on the world map at the battle, the battlefield has none
func TestNoRoadsWithoutOne(test *testing.T) {
    zone := ZoneType{}
    zone.Ground.Roads[5] = true
    ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, zone)
    for _, bits := range ground.Roads {
        if bits != 0 {
            test.Fatalf("a road without one at the battle")
        }
    }
}
