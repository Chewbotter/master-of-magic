package main

// What the debug options of the start screen remember between runs of the game: the unit of the
// last test battle, so "Again" is there after a restart, its ground, coast and river, and the army
// size.
//
// Kept in a file of its own beside the settings of the window, in the folder the game runs in.
// Development runs (-corner, -capture) neither read nor write it, so they do not change what the
// user left.

import (
    "encoding/json"
    "github.com/kazzmir/master-of-magic/game/magic/combat"
    "log"
    "os"
)

const DebugSavedFile = "debug.json"

type debugSaved struct {
    // the unit of the last test battle, by its name with its race
    TestBattleUnit string `json:"test-battle-unit"`
    // how many times the armies of the debug battles are multiplied
    ArmyScale int `json:"army-scale"`
    // the ground and the coast of the test battles, by their names
    TestBattleGround string `json:"test-battle-ground"`
    TestBattleCoast string `json:"test-battle-coast"`
    TestBattleRiver string `json:"test-battle-river"`
    // how many tiles from a town, 0 to 2
    TestBattleFarmland int `json:"test-battle-farmland"`
    // the weather, by its name
    TestBattleWeather string `json:"test-battle-weather"`
    // the roads, by their name
    TestBattleRoads string `json:"test-battle-roads"`
    // who moves first, Player or Enemy
    TestBattleFirstMove string `json:"test-battle-first-move"`
}

var debugSavedLoaded bool

func debugSavedInUse() bool {
    return !capture.Corner && capture.Path == ""
}

// reads the file once. called when the start screen first shows
func loadDebugSaved() {
    if debugSavedLoaded {
        return
    }
    debugSavedLoaded = true

    if !debugSavedInUse() {
        return
    }

    content, err := os.ReadFile(DebugSavedFile)
    if err != nil {
        // there is none before the first test battle
        return
    }

    var saved debugSaved
    err = json.Unmarshal(content, &saved)
    if err != nil {
        log.Printf("Unable to read %v: %v", DebugSavedFile, err)
        return
    }

    if saved.ArmyScale == 1 || saved.ArmyScale == RandomBattleLargeArmies {
        randomBattleArmyScale = saved.ArmyScale
    }

    // what is not there any more is grass without a coast
    testBattleGround = testGroundByName(saved.TestBattleGround)
    testBattleCoast = testCoastByName(saved.TestBattleCoast)
    testBattleRiver = testRiverByName(saved.TestBattleRiver)
    testBattleWeather = combat.WeatherByName(saved.TestBattleWeather)
    testBattleRoad = testRoadByName(saved.TestBattleRoads)
    // the player unless the enemy was picked
    testBattleEnemyFirst = saved.TestBattleFirstMove == testFirstMoveNames[1]
    testBattleFarmland = min(max(saved.TestBattleFarmland, combat.FarmlandNone), combat.FarmlandFar)

    if saved.TestBattleUnit != "" && testBattleLast == nil {
        // a unit that is not there any more is no unit
        testBattleLast = findUnit(saved.TestBattleUnit)
    }
}

func saveDebugSaved() {
    if !debugSavedInUse() {
        return
    }

    var saved debugSaved
    saved.ArmyScale = randomBattleArmyScale
    saved.TestBattleGround = testGrounds[testBattleGround].Name
    saved.TestBattleCoast = testBattleCoast.String()
    saved.TestBattleRiver = testBattleRiver.String()
    saved.TestBattleFarmland = testBattleFarmland
    saved.TestBattleWeather = testBattleWeather.String()
    saved.TestBattleRoads = testRoads[testBattleRoad].Key
    saved.TestBattleFirstMove = testFirstMoveName()
    if testBattleLast != nil {
        saved.TestBattleUnit = unitFullName(testBattleLast)
    }

    content, err := json.MarshalIndent(saved, "", "    ")
    if err != nil {
        return
    }

    err = os.WriteFile(DebugSavedFile, content, 0644)
    if err != nil {
        log.Printf("Unable to write %v: %v", DebugSavedFile, err)
    }
}
