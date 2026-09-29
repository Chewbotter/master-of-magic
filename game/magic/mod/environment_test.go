package mod

import (
    "image"
    "os"
    "path/filepath"
    "testing"
)

// every picture of the environment has a file of its own, and no entry has two names
func TestEnvironmentNamesAreUnique(test *testing.T) {
    files := make(map[string]bool)
    entries := make(map[string]bool)

    for _, picture := range EnvironmentPictures {
        file := filepath.Join(picture.Set, EnvironmentFrameFile(picture.Name, 0))
        if files[file] {
            test.Fatalf("two pictures are %v", file)
        }
        files[file] = true

        key := entryKey(picture.Archive, picture.Entry)
        if entries[key] {
            test.Fatalf("%v has two names", key)
        }
        entries[key] = true

        if FolderName(picture.Set) != picture.Set || FolderName(picture.Name) != picture.Name {
            test.Fatalf("%v/%v is no name for a file", picture.Set, picture.Name)
        }
    }
}

// the replacement folder is read by name before by number
func TestEnvironmentGoesBeforeArchives(test *testing.T) {
    folder = "replace"
    defer func() { folder = "" }()

    paths := framePaths("CMBGRASS.LBX", 49, 0)
    want := []string{
        filepath.Join("replace", "environment", "Grass", "tree 2_0.png"),
        filepath.Join("replace", "archives", "cmbgrass", "049_0.png"),
    }
    if len(paths) != len(want) || paths[0] != want[0] || paths[1] != want[1] {
        test.Fatalf("got %v", paths)
    }
}

// the uses of the ground come from the names: 4 of grass, 2 of every edge of dirt, 1 of every piece
// of raised ground
func TestGroundRoles(test *testing.T) {
    byName := make(map[string]GroundRole)
    total := 0
    for _, role := range GroundRoles {
        byName[role.Name] = role
        total += role.Count
    }

    if total != len(groundNames) {
        test.Fatalf("%v pictures in the uses, %v names", total, len(groundNames))
    }
    if role := byName["grass"]; role.First != 0 || role.Count != 4 {
        test.Fatalf("grass: %+v", role)
    }
    if role := byName["dirt edge SE"]; role.First != 30 || role.Count != 2 {
        test.Fatalf("dirt edge SE: %+v", role)
    }
    if role := byName["rough single"]; role.First != 47 || role.Count != 1 {
        test.Fatalf("rough single: %+v", role)
    }
}

// added pictures are counted from the game's last up to the first that is not there
func TestGroundExtrasAreCounted(test *testing.T) {
    folder = test.TempDir()
    defer func() { folder = "" }()

    directory := filepath.Join(folder, "environment", "Grass")
    if err := os.MkdirAll(directory, 0755); err != nil {
        test.Fatal(err)
    }
    for _, name := range []string{"grass 5_0.png", "grass 6_0.png", "grass 8_0.png", "rough single 2_0.png", "tree 6_0.png", "tree 7_0.png", "tree 8_0.png"} {
        if err := os.WriteFile(filepath.Join(directory, name), []byte{}, 0644); err != nil {
            test.Fatal(err)
        }
    }

    counts := make(map[string]int)
    for _, role := range GroundRoles {
        counts[role.Name] = CountExtras("Grass", role.Name, role.Count)
    }
    if trees := CountExtras("Grass", "tree", 5); trees != 3 {
        test.Fatalf("trees: %v", trees)
    }
    if rocks := CountExtras("Grass", "rock", 5); rocks != 0 {
        test.Fatalf("rocks: %v", rocks)
    }
    if counts["grass"] != 2 || counts["rough single"] != 1 || counts["dirt"] != 0 {
        test.Fatalf("grass %v, rough single %v, dirt %v", counts["grass"], counts["rough single"], counts["dirt"])
    }
    if EnvironmentSet("CMBGRASS.LBX") != "Grass" || EnvironmentSet("cmbtundc.lbx") != "Tundra Myrror" {
        test.Fatalf("sets: %v, %v", EnvironmentSet("CMBGRASS.LBX"), EnvironmentSet("cmbtundc.lbx"))
    }
}

// in a battle of a biome a picture of the landscape is looked for in the folder of the biome, then
// in the one of the landscape, then by number. other pictures know no biome
func TestBiomeGoesBeforeLandscape(test *testing.T) {
    folder = "replace"
    SetBiome(BiomeForest)
    defer func() {
        folder = ""
        SetBiome("")
    }()

    paths := framePaths("cmbgrass.lbx", 49, 0)
    want := []string{
        filepath.Join("replace", "environment", "Forest", "tree 2_0.png"),
        filepath.Join("replace", "environment", "Grass", "tree 2_0.png"),
        filepath.Join("replace", "archives", "cmbgrass", "049_0.png"),
    }
    if len(paths) != len(want) || paths[0] != want[0] || paths[1] != want[1] || paths[2] != want[2] {
        test.Fatalf("got %v", paths)
    }

    // on Myrror the folder of the biome of Myrror
    paths = framePaths("cmbgrasc.lbx", 0, 0)
    if len(paths) != 3 || paths[0] != filepath.Join("replace", "environment", "Forest Myrror", "grass 1_0.png") {
        test.Fatalf("on Myrror: %v", paths)
    }

    // a house is no picture of a landscape
    paths = framePaths("cmbtcity.lbx", 2, 0)
    if len(paths) != 2 || paths[0] != filepath.Join("replace", "environment", "Town", "house 1_0.png") {
        test.Fatalf("a house: %v", paths)
    }

    if BiomeFolder(BiomeSwamp, "Grass") != "Swamp" || BiomeFolder(BiomeSwamp, "Grass Myrror") != "Swamp Myrror" || BiomeFolder("", "Grass") != "Grass" {
        test.Fatalf("folders of biomes")
    }

    // every biome is a kind of a landscape there is
    for _, biome := range Biomes {
        found := false
        for _, landscape := range environmentLandscapes {
            found = found || landscape.Set == biome.Base
        }
        if !found || FolderName(biome.Name) != biome.Name {
            test.Fatalf("biome %+v", biome)
        }
    }
}

// a picture of 2 by 2 tiles is cut where its tiles lie, and only grass and dirt have one
func TestQuarters(test *testing.T) {
    size := image.Pt(60, 32)
    tile := image.Pt(30, 16)
    want := map[[2]int]image.Point{{0, 0}: image.Pt(15, 0), {1, 0}: image.Pt(30, 8), {0, 1}: image.Pt(0, 8), {1, 1}: image.Pt(15, 16)}
    for place, corner := range want {
        if got := quarterCorner(size, tile, place[0], place[1]); got != corner {
            test.Fatalf("the tile %v starts at %v", place, got)
        }
    }
    if QuarterFrame(0, 0) != 0 || QuarterFrame(1, 0) != 1 || QuarterFrame(0, 1) != 2 || QuarterFrame(1, 1) != 3 {
        test.Fatalf("the frames of the quarters")
    }

    if GroundName("cmbgrass.lbx", 0) != "grass" || GroundName("cmbgrass.lbx", 49) != "tree" || GroundName("cmbtcity.lbx", 2) != "" {
        test.Fatalf("names: %v, %v, %v", GroundName("cmbgrass.lbx", 0), GroundName("cmbgrass.lbx", 49), GroundName("cmbtcity.lbx", 2))
    }

    folder = "replace"
    SetBiome(BiomeForest)
    defer func() {
        folder = ""
        SetBiome("")
    }()
    if quartersPath("cmbgrass.lbx", 0) != filepath.Join("replace", "environment", "Forest_highres", "grass 1_0.png") {
        test.Fatalf("grass: %v", quartersPath("cmbgrass.lbx", 0))
    }
    // a tree and raised ground have none
    if quartersPath("cmbgrass.lbx", 49) != "" || quartersPath("cmbgrass.lbx", 32) != "" {
        test.Fatalf("a tree: %v, raised ground: %v", quartersPath("cmbgrass.lbx", 49), quartersPath("cmbgrass.lbx", 32))
    }
    SetBiome("")
    if quartersPath("cmbgrass.lbx", 0) != "" {
        test.Fatalf("without a biome: %v", quartersPath("cmbgrass.lbx", 0))
    }
}
