package main

// The folders of the biomes (game/magic/mod/biomes.go): every one starts as a copy of the folder of
// the landscape it is a kind of, to be painted over.

import (
    "fmt"
    "image/png"
    "log"
    "os"
    "path/filepath"
    "strings"

    "github.com/kazzmir/master-of-magic/game/magic/mod"
)

// pictures of the game that are copied into the folder of a biome that is planned, to look at: from
// which folder, and the files that start with what. no biome is planned now
var waterReference = map[string][][2]string{}

// the pictures of the coast (game/magic/combat/coast.go): the sand of the desert as the beach, the
// water of the battles on the ocean. from which folder, the files that start with what, as what
var coastPictures = [][3]string{
    {"Desert", "grass ", "sand "},
    {"Water", "water ", "water "},
}

const coastNote = "Shore\r\n\r\n" +
    "The coast: a feature, not a kind of a landscape. Not in the original game.\r\n" +
    "A battle on land with the sea beside it on the world map has a wide beach and then water on\r\n" +
    "that side of the field, whatever its landscape is. North of the world map is the upper right of\r\n" +
    "the screen, east the lower right, south the lower left, west the upper left.\r\n" +
    "The beach is rough ground, a step onto it costs twice what grass costs. Into the water go only\r\n" +
    "units that fly, swim or sail.\r\n\r\n" +
    "sand 1 to 4    the beach, one of these by chance. The sand of the desert (Desert, grass 1 to 4)\r\n" +
    "water 1 to 4   the water, one of these by chance, 5 frames each. The water of the battles on the\r\n" +
    "               ocean (Water, water 1 to 4)\r\n\r\n" +
    "A picture that is changed goes to mod\\environment\\Shore with the same name. Pictures can be\r\n" +
    "added: sand 5, water 5 and on, without a gap. A tile is 30 by 16 pixels.\r\n" +
    "Where ground meets sand and sand meets water the game lays the pixels of a tile of sand over a\r\n" +
    "part of the tile of the ground or of the water, with a ragged edge. There is nothing to paint\r\n" +
    "for it. Such a tile counts as what is under the sand.\r\n"

const riverNote = "River\r\n\r\n" +
    "The river: a feature, not a kind of a landscape. Not in the original game as it is here.\r\n" +
    "It lies on whatever the landscape of the battle is: the game draws the ground of the landscape\r\n" +
    "and lays shallow water over it, 2 to 3 tiles wide, with banks that are part water. There is\r\n" +
    "nothing to paint for the water.\r\n" +
    "A tile of the river is rough ground, a step into it costs twice what grass costs. A bank counts\r\n" +
    "as the ground. A battlefield with a river has no roads.\r\n\r\n" +
    "GLINTS go to mod\\environment\\River:\r\n" +
    "glint 1_0.png, glint 1_1.png, ...   the frames of the first glint\r\n" +
    "glint 2_0.png, ...                  the second, and on without a gap\r\n" +
    "Any size, any number of frames. The middle of the picture lies on its place, about one for\r\n" +
    "every two tiles of water, each at a time of its own, a frame for about a quarter of a second. A picture with see-through frames is a\r\n" +
    "glint that is gone for a while.\r\n\r\n" +
    "The files \"_game river ...\" are the pictures the original has of a river (Other), to look at.\r\n" +
    "The game does not use them.\r\n"

// writes the folder of the river. after the folders of the landscapes are written
func exportRiver(outPath string) int {
    environment := filepath.Join(outPath, "environment")
    folder := filepath.Join(environment, mod.FeatureRiver)
    err := os.MkdirAll(folder, 0755)
    if err != nil {
        log.Printf("%v", err)
        return 0
    }

    count := copyPictures(filepath.Join(environment, "Other"), folder, "river ", "_game ")
    os.WriteFile(filepath.Join(folder, sourceFile), []byte(riverNote), 0644)
    return count
}

const farmlandNote = "Farmland\r\n\r\n" +
    "The fields around a town: a feature, not a kind of a landscape. Not in the original game.\r\n" +
    "A battle one or two tiles from a town, on land whose landscape is Grass, is fought on its\r\n" +
    "fields: grass with square plots of crops, one row of grass between them, and houses of the\r\n" +
    "race of the town here and there, more of them right beside the town.\r\n\r\n" +
    "crop 1 to 4    the crops. A plot shows one of them in all of its tiles.\r\n\r\n" +
    "The game has no pictures of crops. The ones here are the tiles of grass (Grass, grass 1 to 4)\r\n" +
    "in the colors of ripe grain, as the game makes them until there are painted ones.\r\n" +
    "A picture that is changed goes to mod\\environment\\Farmland with the same name. Pictures can\r\n" +
    "be added: crop 5 and on, without a gap. A tile is 30 by 16 pixels.\r\n" +
    "The houses are the ones of the town, in Town.\r\n"

// writes the folder of the farmland, its crops made of the grass that is in the folder of the
// landscape. after the folders of the landscapes are written
func exportFarmland(outPath string) int {
    environment := filepath.Join(outPath, "environment")
    folder := filepath.Join(environment, mod.FeatureFarmland)
    err := os.MkdirAll(folder, 0755)
    if err != nil {
        log.Printf("%v", err)
        return 0
    }

    count := 0
    for number := 1; number <= 4; number++ {
        file, err := os.Open(filepath.Join(environment, "Grass", fmt.Sprintf("grass %v_0.png", number)))
        if err != nil {
            log.Printf("%v", err)
            continue
        }
        grass, err := png.Decode(file)
        file.Close()
        if err != nil {
            log.Printf("%v", err)
            continue
        }

        out, err := os.Create(filepath.Join(folder, fmt.Sprintf("crop %v_0.png", number)))
        if err != nil {
            log.Printf("%v", err)
            continue
        }
        if png.Encode(out, mod.CropTile(grass)) == nil {
            count += 1
        }
        out.Close()
    }

    os.WriteFile(filepath.Join(folder, sourceFile), []byte(farmlandNote), 0644)
    return count
}

// writes the folder of the coast. after the folders of the landscapes are written
func exportCoast(outPath string) int {
    environment := filepath.Join(outPath, "environment")
    folder := filepath.Join(environment, mod.FeatureShore)
    err := os.MkdirAll(folder, 0755)
    if err != nil {
        log.Printf("%v", err)
        return 0
    }

    count := 0
    for _, pictures := range coastPictures {
        entries, err := os.ReadDir(filepath.Join(environment, pictures[0]))
        if err != nil {
            log.Printf("%v", err)
            continue
        }
        for _, entry := range entries {
            name := entry.Name()
            if !strings.HasPrefix(name, pictures[1]) || !strings.HasSuffix(name, ".png") {
                continue
            }
            to := pictures[2] + strings.TrimPrefix(name, pictures[1])
            if copyFile(filepath.Join(environment, pictures[0], name), filepath.Join(folder, to)) == nil {
                count += 1
            }
        }
    }

    os.WriteFile(filepath.Join(folder, sourceFile), []byte(coastNote), 0644)
    return count
}

const sourceFile = "_source.txt"

func copyFile(from string, to string) error {
    data, err := os.ReadFile(from)
    if err != nil {
        return err
    }
    return os.WriteFile(to, data, 0644)
}

// copies the pictures of a folder whose names start with a prefix, all of them without one
func copyPictures(from string, to string, prefix string, addPrefix string) int {
    entries, err := os.ReadDir(from)
    if err != nil {
        log.Printf("%v", err)
        return 0
    }

    count := 0
    for _, entry := range entries {
        name := entry.Name()
        if entry.IsDir() || !strings.HasSuffix(name, ".png") || !strings.HasPrefix(name, prefix) {
            continue
        }
        if copyFile(filepath.Join(from, name), filepath.Join(to, addPrefix + name)) == nil {
            count += 1
        }
    }
    return count
}

// writes the folders of the biomes. after the folders of the landscapes are written
func exportBiomes(outPath string) int {
    count := 0
    environment := filepath.Join(outPath, "environment")

    for _, biome := range mod.Biomes {
        folder := filepath.Join(environment, biome.Name)
        err := os.MkdirAll(folder, 0755)
        if err != nil {
            log.Printf("%v", err)
            continue
        }

        count += copyPictures(filepath.Join(environment, biome.Base), folder, "", "")

        var note strings.Builder
        note.WriteString(fmt.Sprintf("%v\r\n\r\n", biome.Name))
        note.WriteString(fmt.Sprintf("A biome: a kind of the landscape %v with pictures of its own. Not in the original game.\r\n", biome.Base))
        note.WriteString(fmt.Sprintf("Where: %v.\r\n\r\n", biome.Note))
        note.WriteString(fmt.Sprintf("The pictures here are copies of the ones of %v, to start from. A picture that is changed goes\r\n", biome.Base))
        note.WriteString(fmt.Sprintf("to mod\\environment\\%v with the same name. A battle of this biome looks for a picture there\r\n", biome.Name))
        note.WriteString(fmt.Sprintf("first, then in mod\\environment\\%v, then takes the game's. So only what is different has to\r\n", biome.Base))
        note.WriteString("be there.\r\n")
        note.WriteString("Pictures can be added as in a landscape (grass 5, tree 6, flat 1, prop 1, large 3): the added\r\n")
        note.WriteString(fmt.Sprintf("ones of a biome are the ones of its own folder, the ones added to %v do not show in it.\r\n", biome.Base))
        note.WriteString(fmt.Sprintf("On Myrror the folder is \"%v Myrror\", a kind of \"%v Myrror\".\r\n", biome.Name, biome.Base))

        if biome.Planned {
            note.WriteString("\r\nNOT SHOWN BY THE GAME YET. How water is shown in a battle is not decided. The folder is here\r\n")
            note.WriteString("to be filled.\r\n")
            for _, reference := range waterReference[biome.Name] {
                copied := copyPictures(filepath.Join(environment, reference[0]), folder, reference[1], "_game ")
                count += copied
                note.WriteString(fmt.Sprintf("The files \"_game %v...\" are the %v pictures the game has of it in %v, to look at.\r\n", reference[1], copied, reference[0]))
            }
        }

        // what the pictures are, from the folder of the landscape
        base, err := os.ReadFile(filepath.Join(environment, biome.Base, sourceFile))
        if err == nil {
            lines := strings.SplitN(string(base), "\r\n\r\n", 2)
            if len(lines) == 2 {
                note.WriteString(fmt.Sprintf("\r\nThe pictures, as in %v:\r\n\r\n", biome.Base))
                note.WriteString(lines[1])
            }
        }

        os.WriteFile(filepath.Join(folder, sourceFile), []byte(note.String()), 0644)
    }

    return count
}
