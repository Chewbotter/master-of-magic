package mod

// Biomes: kinds of a landscape with pictures of their own (user, 2026-09-28: "proper forests",
// "sub-biomes, really"). Not in the original, where a battle in a forest is fought on grass with
// more trees.
//
// A biome is a folder of the replacement folder, next to the folders of the landscapes:
//
//   environment/Forest/grass 1_0.png, tree 1_0.png, ...
//
// with the names of the pictures of the landscape it is a kind of (its base). In a battle of the
// biome a picture is looked for
//   1. in the folder of the biome
//   2. in the folder of its base, as a changed picture of the game
//   3. in the game
// so a biome only needs the pictures that are different, and a battle never misses one.
//
// Pictures that are ADDED (grass 5, tree 6, flat 1, large 3, see environment.go) are counted in
// the folder of the biome alone: a forest does not take the flowers that were added to the grass.
//
// On Myrror the folder is "<name> Myrror", with the landscape of Myrror as its base.
//
// Which battle is of which biome is decided by the world map, see game/battleground.go.

import (
    "path/filepath"
    "strings"
)

type Biome struct {
    // the name of the folder
    Name string
    // the folder of the landscape it is a kind of, on Arcanus
    Base string
    // where it is
    Note string
    // the game does not show it yet: its folder is there to be filled
    Planned bool
}

const myrrorSuffix = " Myrror"

const (
    BiomeForest = "Forest"
    BiomeSwamp = "Swamp"
    BiomeHills = "Hills"
    BiomeVolcano = "Volcano"
    BiomeSnowyMountain = "Snowy Mountain"
    BiomeLake = "Lake"
)

// FEATURES lie on whatever the landscape or biome of a battle is, they are no kinds of it: the
// coast (combat/coast.go), and what water and the nearness of towns are to add. A feature has a
// folder of its own, which holds what it adds and nothing of a landscape
const FeatureShore = "Shore"
// the river (combat/river.go)
const FeatureRiver = "River"

var Biomes = []Biome{
    {Name: BiomeForest, Base: "Grass", Note: "a forest of the world map, and a node of nature"},
    {Name: BiomeSwamp, Base: "Grass", Note: "a swamp of the world map"},
    {Name: BiomeHills, Base: "Grass", Note: "hills of the world map"},
    {Name: BiomeVolcano, Base: "Mountain", Note: "a volcano of the world map"},
    {Name: BiomeSnowyMountain, Base: "Mountain", Note: "a mountain of the world map with tundra next to it"},
    {Name: BiomeLake, Base: "Grass", Note: "a lake of the world map", Planned: true},
}

// the biome of the battle that is fought, nothing for a plain landscape
var biome string

// the biome of the battle that starts. its pictures are looked for from here on
func SetBiome(name string) {
    biome = name
}

// the folder of a biome for a landscape, by the folder of the landscape: "Forest" on "Grass",
// "Forest Myrror" on "Grass Myrror". the folder of the landscape without a biome
func BiomeFolder(name string, set string) string {
    if name == "" || set == "" {
        return set
    }
    if strings.HasSuffix(set, myrrorSuffix) {
        return name + myrrorSuffix
    }
    return name
}

// the file of a frame of a picture of a landscape in the folder of the biome that is fought in,
// nothing without a biome or for a picture that is of no landscape
func biomeFramePath(archive string, entry int, frame int) string {
    if biome == "" {
        return ""
    }
    picture, ok := environmentEntries[entryKey(archive, entry)]
    if !ok || EnvironmentSet(archive) == "" {
        return ""
    }
    return filepath.Join(folder, environmentFolder, BiomeFolder(biome, picture.Set), EnvironmentFrameFile(picture.Name, frame))
}
