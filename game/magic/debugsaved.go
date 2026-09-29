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
