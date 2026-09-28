package mod

import (
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
