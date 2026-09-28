package mod

// The pictures of the places battles are fought in, by name: the ground, trees and rocks of each
// landscape, towns, walls, lairs and nodes.
//
// The game keeps them in several archives by number. The export writes each of them to a folder
// with a name as well, and the replacement folder is read the same way:
//
//   environment/<set>/<name>_<frame>.png      environment/Grass/tree 1_0.png
//
// A picture in environment/ goes before one in archives/.

import (
    "fmt"
    "path/filepath"
)

const environmentFolder = "environment"

type EnvironmentPicture struct {
    // the folder of the set
    Set string
    // the name of the picture in the set
    Name string
    Archive string
    Entry int
    // where the game shows it
    Note string
}

// the landscapes of battles and the archives of their ground, trees and rocks
var environmentLandscapes = []struct {
    Set string
    Archive string
}{
    {Set: "Grass", Archive: "cmbgrass.lbx"},
    {Set: "Grass Myrror", Archive: "cmbgrasc.lbx"},
    {Set: "Desert", Archive: "cmbdesrt.lbx"},
    {Set: "Desert Myrror", Archive: "cmbdesrc.lbx"},
    {Set: "Mountain", Archive: "cmbmount.lbx"},
    {Set: "Mountain Myrror", Archive: "cmbmounc.lbx"},
    {Set: "Tundra", Archive: "cmbtundr.lbx"},
    {Set: "Tundra Myrror", Archive: "cmbtundc.lbx"},
}

// pictures of a landscape: ground 00 to 31 is the ground the game picks from at random, 32 to 47 is
// raised ground it does not use. then five trees and five rocks
const environmentGround = 32
const environmentRaisedGround = 16
const environmentTrees = 48
const environmentRocks = 53
const environmentScenery = 5

// the pieces of a wall, as in wallPieces of combat/scenery.go
const environmentStonePieces = 12
const environmentMagicPieces = 14

var EnvironmentPictures = makeEnvironmentPictures()

func makeEnvironmentPictures() []EnvironmentPicture {
    var out []EnvironmentPicture

    add := func(set string, name string, archive string, entry int, note string) {
        out = append(out, EnvironmentPicture{Set: set, Name: name, Archive: archive, Entry: entry, Note: note})
    }

    // a row of pictures in a row of entries: name 1, name 2, ... or name 00, name 01, ...
    addRow := func(set string, name string, archive string, first int, count int, fromZero bool, note string) {
        for index := range count {
            number := fmt.Sprintf("%v", index + 1)
            if fromZero {
                number = fmt.Sprintf("%02d", index)
            }
            add(set, name + " " + number, archive, first + index, note)
        }
    }

    for _, landscape := range environmentLandscapes {
        addRow(landscape.Set, "ground", landscape.Archive, 0, environmentGround, true, "the ground, one of these at random for every tile")
        for index := range environmentRaisedGround {
            add(landscape.Set, fmt.Sprintf("ground %02d", environmentGround + index), landscape.Archive, environmentGround + index, "raised ground, not used by the game")
        }
        addRow(landscape.Set, "tree", landscape.Archive, environmentTrees, environmentScenery, false, "a tree, one of these at random")
        addRow(landscape.Set, "rock", landscape.Archive, environmentRocks, environmentScenery, false, "a rock, one of these at random")
    }

    // water: 4 pictures of animated ground each
    addRow("Water", "water", "cmbtcity.lbx", 109, 4, false, "the ground of a battle on the water, one of these at random for every tile")
    addRow("Water Myrror", "water", "chriver.lbx", 12, 4, false, "the ground of a battle on the water, one of these at random for every tile")

    // towns
    add("Town", "roads", "cmbtcity.lbx", 0, "the ground of a town")
    add("Town", "outpost ground", "cmbtcity.lbx", 1, "the ground under an outpost")
    addRow("Town", "house", "cmbtcity.lbx", 2, 5, false, "the houses of most races")
    addRow("Town", "hut", "cmbtcity.lbx", 7, 5, false, "the houses of races that live in huts")
    addRow("Town", "tree house", "cmbtcity.lbx", 12, 5, false, "the houses of races that live in trees")
    add("Town", "fortress", "cmbtcity.lbx", 17, "the fortress of a town")
    add("Town", "outpost", "cmbtcity.lbx", 18, "an outpost")
    add("Town", "clouds", "cmbtcity.lbx", 113, "the ground of a town with a flying fortress")
    addRow("Town", "road", "cmbtcity.lbx", 69, 14, true, "pieces of road, not used by the game")
    addRow("Town", "enchanted road", "cmbtcity.lbx", 83, 14, true, "pieces of road, not used by the game")

    add("Other", "mud", "cmbtcity.lbx", 118, "on a tile that was turned to mud")
    addRow("Other", "river", "cmbtcity.lbx", 103, 6, true, "pieces of river, not used by the game")
    add("Other", "floating island", "cmbtcity.lbx", 117, "not used by the game")
    add("Other", "stones", "cmbtcity.lbx", 121, "not used by the game")

    // lairs and nodes
    add("Lairs", "cave", "cmbtcity.lbx", 19, "a lair")
    add("Lairs", "tower", "cmbtcity.lbx", 20, "a wizard's tower")
    add("Lairs", "ruins", "cmbtcity.lbx", 21, "ruins, a dungeon, a fallen temple")
    add("Lairs", "keep", "cmbtcity.lbx", 22, "an abandoned keep")
    add("Lairs", "temple", "cmbtcity.lbx", 23, "an ancient temple")
    add("Lairs", "nature node", "cmbtcity.lbx", 65, "a node of nature")
    add("Lairs", "sorcery node", "cmbtcity.lbx", 66, "a node of sorcery, part of the ground")
    add("Lairs", "chaos node", "chriver.lbx", 32, "a node of chaos: the mountain")
    addRow("Lairs", "chaos node fire", "chriver.lbx", 24, 8, false, "a node of chaos: the fire on the mountain, one after the other")

    // walls. stone has two frames, standing and broken
    addRow("Walls", "stone", "citywall.lbx", 0, environmentStonePieces, true, "a wall of stone. frame 0 standing, 1 broken")
    addRow("Walls", "stone myrror", "citywall.lbx", environmentStonePieces, environmentStonePieces, true, "a wall of stone on Myrror. frame 0 standing, 1 broken")
    addRow("Walls", "stone mossy", "citywall.lbx", environmentStonePieces * 2, environmentStonePieces, true, "not used by the game")
    addRow("Walls", "fire", "citywall.lbx", 36, environmentMagicPieces, true, "a wall of fire")
    addRow("Walls", "darkness", "citywall.lbx", 50, environmentMagicPieces, true, "a wall of darkness")
    addRow("Walls", "fire rising", "wallrise.lbx", 36, environmentMagicPieces, true, "a wall of fire while it is cast")
    addRow("Walls", "darkness rising", "wallrise.lbx", 50, environmentMagicPieces, true, "a wall of darkness while it is cast")

    return out
}

// by archive and entry: the picture
var environmentEntries = makeEnvironmentEntries()

func makeEnvironmentEntries() map[string]EnvironmentPicture {
    out := make(map[string]EnvironmentPicture)
    for _, picture := range EnvironmentPictures {
        out[entryKey(picture.Archive, picture.Entry)] = picture
    }
    return out
}

func EnvironmentFrameFile(name string, frame int) string {
    return fmt.Sprintf("%v_%v.png", name, frame)
}

// the file of a frame of the environment in the replacement folder, or nothing if the entry is none
func environmentFramePath(archive string, entry int, frame int) string {
    picture, ok := environmentEntries[entryKey(archive, entry)]
    if !ok {
        return ""
    }
    return filepath.Join(folder, environmentFolder, picture.Set, EnvironmentFrameFile(picture.Name, frame))
}
