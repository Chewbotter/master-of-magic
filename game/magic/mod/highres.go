package mod

// AN EXPERIMENT (user, 2026-09-29): pictures of the ground at twice the pixels. "There are
// double-res versions of the forest biome tiles ... Let's try implementing those at the same
// scale as the existing tiles and see how it looks. I realize this will change the pixel density
// for the combat tiles in particular."
//
// A folder of the environment can have a twin whose name ends in _highres:
//
//   environment/Forest/grass 1_0.png           30 by 16
//   environment/Forest_highres/grass 1_0.png   60 by 32
//
// with the same names and every picture twice as wide and as high. A picture that is in the twin
// is taken from there and drawn at half its size, so it takes the place on the screen the plain
// one takes, with pixels half as large as the pixels of everything else.

import (
    "path/filepath"
)

// turns it off: the twins are not looked at
var HighRes = true

const HighResSuffix = "_highres"
// how many pixels of a picture of a twin lie along one pixel of the art
const HighResDensity = 2

// the twin of a folder of the environment
func HighResFolder(set string) string {
    if set == "" {
        return ""
    }
    return set + HighResSuffix
}

// the file of a frame of a picture of a landscape in the twin of the folder of the biome that is
// fought in, nothing without a biome
func biomeHighResPath(archive string, entry int, frame int) string {
    if !HighRes || biome == "" {
        return ""
    }
    picture, ok := environmentEntries[entryKey(archive, entry)]
    if !ok || EnvironmentSet(archive) == "" {
        return ""
    }
    return filepath.Join(folder, environmentFolder, HighResFolder(BiomeFolder(biome, picture.Set)), EnvironmentFrameFile(picture.Name, frame))
}

// how many pixels of the picture of an entry lie along one pixel of the art: 1, or more for a
// picture that is taken from a twin
func Density(archive string, entry int) int {
    if folder == "" {
        return 1
    }
    path := biomeHighResPath(archive, entry, 0)
    if path != "" && hasFile(path) {
        return HighResDensity
    }
    return 1
}
