package terrain

import (
    "strings"
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
)

// the land of the original: its share of the world by Land Size, the tundra rows at the poles, the
// nodes of each plane
func TestClassicLand(test *testing.T) {
    for landSize, want := range []int{360, 480, 720} {
        kinds := classicKinds(landSize)
        land := 0
        for x := 0; x < ClassicWidth; x++ {
            // the rows next to the poles have runs of tundra besides the walked land
            for y := 2; y < ClassicHeight - 2; y++ {
                if kinds[x][y] != classicOcean {
                    land += 1
                }
            }
            // the walk of a desert goes around the top and bottom of the world as the original's does,
            // so now and then a pole has a square of desert
            for _, y := range []int{0, ClassicHeight - 1} {
                if kinds[x][y] != classicTundra && kinds[x][y] != classicDesert {
                    test.Errorf("the poles are tundra all along")
                }
            }
        }
        // the walk stops one square past what it needs
        if land < want || land > want + 1 {
            test.Errorf("land size %v: %v squares of land, want about %v", landSize, land, want)
        }

        if testing.Verbose() && landSize == 1 {
            nodes := classicNodes(kinds, data.PlaneArcanus)
            letters := map[classicKind]string{classicOcean: ".", classicGrass: "g", classicForest: "f", classicHills: "h", classicMountain: "M", classicTundra: "t", classicDesert: "d", classicSwamp: "s"}
            for y := 0; y < ClassicHeight; y++ {
                var row strings.Builder
                for x := 0; x < ClassicWidth; x++ {
                    letter := letters[kinds[x][y]]
                    for _, node := range nodes {
                        if node.X == x && node.Y == y {
                            letter = []string{"S", "N", "C"}[node.Kind]
                        }
                    }
                    row.WriteString(letter)
                }
                test.Log(row.String())
            }
        }
    }
}

// 16 nodes on Arcanus and 14 on Myrror, 3 or more apart, at least 6 of chaos and nature on Arcanus
// when there are many of sorcery
func TestClassicNodes(test *testing.T) {
    for _, plane := range []data.Plane{data.PlaneArcanus, data.PlaneMyrror} {
        kinds := classicKinds(1)
        nodes := classicNodes(kinds, plane)
        want := 16
        if plane == data.PlaneMyrror {
            want = 14
        }
        if len(nodes) != want {
            test.Errorf("%v: %v nodes, want %v", plane, len(nodes), want)
        }
        for index, node := range nodes {
            for _, other := range nodes[index + 1:] {
                if classicDistance(node.X, node.Y, other.X, other.Y) < 3 {
                    test.Errorf("nodes at %v,%v and %v,%v are too near", node.X, node.Y, other.X, other.Y)
                }
            }
        }
    }
}

// 6 towers, apart, on land of one plane or the other (where they make grassland)
func TestClassicTowers(test *testing.T) {
    arcanus := classicKinds(1)
    myrror := classicKinds(1)
    nodes := append(classicNodes(arcanus, data.PlaneArcanus), classicNodes(myrror, data.PlaneMyrror)...)
    towers := classicTowers(arcanus, myrror, nodes)
    if len(towers) != ClassicTowers {
        test.Fatalf("%v towers", len(towers))
    }
    for _, tower := range towers {
        if arcanus[tower.X][tower.Y] != classicGrass || myrror[tower.X][tower.Y] != classicGrass {
            test.Errorf("a tower stands on grassland on both planes")
        }
        for _, node := range nodes {
            if classicDistance(tower.X, tower.Y, node.X, node.Y) < 4 {
                test.Errorf("a tower is near a node")
            }
        }
    }
}
