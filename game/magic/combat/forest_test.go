package combat

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/mod"
)

// a forest has many more trees than the original's 60, none where the armies start, and both
// woods and clearings
func TestWoods(test *testing.T) {
    zone := ZoneType{}
    zone.Ground.SetBiome(mod.BiomeForest)

    for range 10 {
        ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, zone)
        scenery := makeScenery(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, zone, ground)

        trees := 0
        for _, piece := range scenery {
            if piece.Kind == SceneryTree {
                trees += 1
            }
        }

        onField, wooded, open := 0, 0, 0
        for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
            for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
                count := ground.TreesAt(cgx, cgy)
                if count > forestTreesMost {
                    test.Fatalf("%v trees at %v, %v", count, cgx, cgy)
                }
                if count > 0 && (armyStarts(cgx, cgy) || forestBeyondArmies(cgx, cgy) == 0) {
                    test.Fatalf("trees in the clearing of the armies at %v, %v", cgx, cgy)
                }

                x, y := CellToTile(cgx, cgy)
                if x < 0 || y < 0 || x >= BattlefieldWidth || y >= BattlefieldHeight {
                    continue
                }
                onField += count
                if count > 0 {
                    wooded += 1
                } else {
                    open += 1
                }
            }
        }

        // the cells the first rows of the armies stand in have no trees
        for _, team := range []Team{TeamDefender, TeamAttacker} {
            for _, cell := range deployCells(team, deployPlaces, false) {
                if ground.TreesAt(cell.X, cell.Y) > 0 {
                    test.Fatalf("trees where an army starts, at %v", cell)
                }
            }
        }

        if onField < 200 {
            test.Fatalf("%v trees on the field, %v in all", onField, trees)
        }
        if wooded * 5 < open || open * 5 < wooded {
            test.Fatalf("%v cells of the field with trees, %v without", wooded, open)
        }
    }

    // grass land has the trees of the original
    ground := makeBattleGround(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, ZoneType{})
    trees := 0
    for _, piece := range makeScenery(BattlefieldWidth, BattlefieldHeight, CombatLandscapeGrass, 0, ZoneType{}, ground) {
        if piece.Kind == SceneryTree {
            trees += 1
        }
    }
    if trees > 20 * (1 + sceneryBeyondScreen(BattlefieldWidth, BattlefieldHeight)) {
        test.Fatalf("%v trees on grass land", trees)
    }
}
