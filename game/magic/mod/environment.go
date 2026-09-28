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
    "image"
    "image/png"
    "os"
    "path/filepath"
    "strconv"
    "strings"
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

// pictures of a landscape: 48 of the ground, named by what combat/terrain.go uses them for (see
// groundNames), then five trees and five rocks
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
        for index, ground := range groundNames {
            add(landscape.Set, ground.Name, landscape.Archive, index, ground.Note)
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
    for index, piece := range roadNames {
        add("Town", piece.Name, "cmbtcity.lbx", 69 + index, piece.Note)
        add("Town", "enchanted " + piece.Name, "cmbtcity.lbx", 83 + index, piece.Note + ", enchanted")
    }

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

type environmentName struct {
    Name string
    Note string
}

// The 48 pictures of the ground of a landscape by number, named by what the original's
// Set_Terrain_Tile_Types picks them for. The directions are the ones of the diamond on the screen:
// its edges NE, SE, SW, NW and its corners N (top), E, S (bottom), W. In the grid of the battle, up
// is the NE edge, right the SE edge, down the SW edge, left the NW edge.
var groundNames = []environmentName{
    {"grass 1", "grass, one of 1 to 4 at random"},
    {"grass 2", "grass, one of 1 to 4 at random"},
    {"grass 3", "grass, one of 1 to 4 at random"},
    {"grass 4", "grass, one of 1 to 4 at random"},
    {"dirt 1", "dirt, one of 1 to 4 at random"},
    {"dirt 2", "dirt, one of 1 to 4 at random"},
    {"dirt 3", "dirt, one of 1 to 4 at random"},
    {"dirt 4", "dirt, one of 1 to 4 at random"},
    {"dirt corner S 1", "grass with dirt beyond its bottom corner, one of two"},
    {"dirt corner S 2", "grass with dirt beyond its bottom corner, one of two"},
    {"dirt edge SW 1", "grass with dirt beyond its SW edge, one of two"},
    {"dirt edge SW 2", "grass with dirt beyond its SW edge, one of two"},
    {"dirt edges SE SW 1", "grass with dirt beyond its SE and SW edges, one of two"},
    {"dirt edges SE SW 2", "grass with dirt beyond its SE and SW edges, one of two"},
    {"dirt corner W 1", "grass with dirt beyond its left corner, one of two"},
    {"dirt corner W 2", "grass with dirt beyond its left corner, one of two"},
    {"dirt edges SW NW 1", "grass with dirt beyond its SW and NW edges, one of two"},
    {"dirt edges SW NW 2", "grass with dirt beyond its SW and NW edges, one of two"},
    {"dirt edge NW 1", "grass with dirt beyond its NW edge, one of two"},
    {"dirt edge NW 2", "grass with dirt beyond its NW edge, one of two"},
    {"dirt corner N 1", "grass with dirt beyond its top corner, one of two"},
    {"dirt corner N 2", "grass with dirt beyond its top corner, one of two"},
    {"dirt edges NE NW 1", "grass with dirt beyond its NE and NW edges, one of two"},
    {"dirt edges NE NW 2", "grass with dirt beyond its NE and NW edges, one of two"},
    {"dirt edge NE 1", "grass with dirt beyond its NE edge, one of two"},
    {"dirt edge NE 2", "grass with dirt beyond its NE edge, one of two"},
    {"dirt corner E 1", "grass with dirt beyond its right corner, one of two"},
    {"dirt corner E 2", "grass with dirt beyond its right corner, one of two"},
    {"dirt edges NE SE 1", "grass with dirt beyond its NE and SE edges, one of two"},
    {"dirt edges NE SE 2", "grass with dirt beyond its NE and SE edges, one of two"},
    {"dirt edge SE 1", "grass with dirt beyond its SE edge, one of two"},
    {"dirt edge SE 2", "grass with dirt beyond its SE edge, one of two"},
    {"rough SW", "raised ground, goes on across its SW edge"},
    {"rough NE SW", "raised ground, goes on across its NE and SW edges"},
    {"rough SE", "raised ground, goes on across its SE edge"},
    {"rough SE NW", "raised ground, goes on across its SE and NW edges"},
    {"rough all", "raised ground, goes on across all edges"},
    {"rough NE", "raised ground, goes on across its NE edge"},
    {"rough NW", "raised ground, goes on across its NW edge"},
    {"rough SE SW", "raised ground, goes on across its SE and SW edges"},
    {"rough SE SW NW", "raised ground, goes on across its SE, SW and NW edges"},
    {"rough SW NW", "raised ground, goes on across its SW and NW edges"},
    {"rough NE SE SW", "raised ground, goes on across its NE, SE and SW edges"},
    {"rough NE SW NW", "raised ground, goes on across its NE, SW and NW edges"},
    {"rough NE SE", "raised ground, goes on across its NE and SE edges"},
    {"rough NE SE NW", "raised ground, goes on across its NE, SE and NW edges"},
    {"rough NE NW", "raised ground, goes on across its NE and NW edges"},
    {"rough single", "raised ground on its own"},
}

// the pieces of road, cmbtcity 69 on: which two edges of the diamond a piece joins, in two sets of
// pictures (combat/terrain.go roadPictures)
var roadNames = []environmentName{
    {"road NW SE 1", "a road across the tile, first set"},
    {"road NE SW 1", "a road along the tile, first set"},
    {"road NE NW 1", "a road bending, first set"},
    {"road NE SE 1", "a road bending, first set"},
    {"road SE SW 1", "a road bending, first set"},
    {"road SW NW 1", "a road bending, first set"},
    {"road unused 1", "not used by the game"},
    {"road NW SE 2", "a road across the tile, second set"},
    {"road NE SW 2", "a road along the tile, second set"},
    {"road NE NW 2", "a road bending, second set"},
    {"road NE SE 2", "a road bending, second set"},
    {"road SE SW 2", "a road bending, second set"},
    {"road SW NW 2", "a road bending, second set"},
    {"road unused 2", "not used by the game"},
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

// MORE PICTURES OF THE GROUND (user, 2026-09-28). The ground of a landscape has a few pictures for
// every use: 4 of grass, 2 of grass with dirt beyond its SE edge, 1 of each piece of raised ground.
// The replacement folder can add to them. The names go on counting where the game's end:
//
//   environment/Grass/grass 5_0.png, grass 6_0.png, ...
//   environment/Grass/dirt edge SE 3_0.png, ...
//   environment/Grass/rough NE SW 2_0.png, ...
//
// They end with the first number that is not there. The battle picks among all of them, the
// game's and the added ones (combat/terrain.go). Added pictures are shown as they are painted,
// with frames _0, _1 and on as the others.

// a use of pictures of the ground: its name, its first picture in the archive of a landscape, and
// how many the game has
type GroundRole struct {
    Name string
    First int
    Count int
}

var GroundRoles = makeGroundRoles()

// the uses, from the names of the pictures: "grass 1" to "grass 4" are the use "grass"
func makeGroundRoles() []GroundRole {
    var out []GroundRole
    for index, ground := range groundNames {
        name := ground.Name
        cut := strings.LastIndex(name, " ")
        if cut >= 0 {
            _, err := strconv.Atoi(name[cut + 1:])
            if err == nil {
                name = name[:cut]
            }
        }

        if len(out) > 0 && out[len(out) - 1].Name == name {
            out[len(out) - 1].Count += 1
        } else {
            out = append(out, GroundRole{Name: name, First: index, Count: 1})
        }
    }
    return out
}

// the most pictures that can be added to a use
const maxGroundExtras = 99

// the folder of the pictures of a landscape, by its archive. nothing if the archive is none
func EnvironmentSet(archive string) string {
    for _, landscape := range environmentLandscapes {
        if strings.EqualFold(landscape.Archive, archive) {
            return landscape.Set
        }
    }
    return ""
}

// the name of a picture of a use, by its number from 1
func GroundPictureName(role GroundRole, number int) string {
    return fmt.Sprintf("%v %v", role.Name, number)
}

// how many pictures the replacement folder adds to a use of the ground of a landscape
func CountGroundExtras(set string, role GroundRole) int {
    if folder == "" || set == "" {
        return 0
    }

    directory := filepath.Join(folder, environmentFolder, set)
    // the files of the folder are looked up anew
    delete(folderLists, directory)

    count := 0
    for count < maxGroundExtras {
        name := GroundPictureName(role, role.Count + count + 1)
        if !hasFile(filepath.Join(directory, EnvironmentFrameFile(name, 0))) {
            break
        }
        count += 1
    }
    return count
}

// the frames of an added picture of the ground, as they are
func ReadGroundExtra(set string, role GroundRole, number int) []image.Image {
    if folder == "" || set == "" {
        return nil
    }

    var out []image.Image
    name := GroundPictureName(role, number)
    for frame := 0; frame < maxFrames; frame++ {
        path := filepath.Join(folder, environmentFolder, set, EnvironmentFrameFile(name, frame))
        if !hasFile(path) {
            break
        }

        file, err := os.Open(path)
        if err != nil {
            break
        }
        picture, err := png.Decode(file)
        file.Close()
        if err != nil {
            reportOnce(fmt.Sprintf("Replacement picture %v can not be read: %v", path, err))
            break
        }

        reportOnce(fmt.Sprintf("Replacement picture %v", path))
        out = append(out, picture)
    }
    return out
}
