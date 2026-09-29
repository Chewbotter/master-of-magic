package main

// The folders of the biomes (game/magic/mod/biomes.go): every one starts as a copy of the folder of
// the landscape it is a kind of, to be painted over.

import (
    "fmt"
    "log"
    "os"
    "path/filepath"
    "strings"

    "github.com/kazzmir/master-of-magic/game/magic/mod"
)

// pictures of the water the game has, copied into the folders of the biomes of water to look at:
// from which folder, and the files that start with what
var waterReference = map[string][][2]string{
    mod.BiomeRiver: {{"Other", "river "}},
    mod.BiomeShore: {{"Water", "water "}},
    mod.BiomeLake: {{"Water", "water "}},
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
