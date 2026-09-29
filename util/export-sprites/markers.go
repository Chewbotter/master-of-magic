package main

// The markers the mod makes by itself, written out so they can be repainted. See game/magic/mod/markers.go

import (
    "path/filepath"

    "github.com/kazzmir/master-of-magic/game/magic/mod"
    "github.com/kazzmir/master-of-magic/lib/lbx"
)

func exportMarkers(outPath string) int {
    count := 0
    flag := mod.DefaultFlag(lbx.GetDefaultPalette())
    if writePng(filepath.Join(outPath, "markers", mod.FlagFile), flag) == nil {
        count += 1
    }

    // the bookmarks of the spellbook, see game/magic/mod/bookmarks.go
    for _, kind := range mod.BookmarkKinds {
        if writePng(filepath.Join(outPath, "markers", mod.BookmarkName(kind) + ".png"), mod.DefaultBookmark(kind)) == nil {
            count += 1
        }
    }

    return count
}

// the tabs of the spellbooks as they look under the mouse, made from the pictures of the game
func exportTabs(dataPath string, outPath string) int {
    count := 0
    for _, tab := range mod.TabPictures {
        archive, err := openArchive(dataPath, tab.Archive)
        if err != nil {
            continue
        }

        frames, err := readPictures(archive, tab.Entry)
        if err != nil || len(frames) == 0 {
            continue
        }

        if writePng(filepath.Join(outPath, "markers", mod.TabHoverName(tab.Name) + ".png"), mod.MakeTabHover(frames[0])) == nil {
            count += 1
        }
    }

    return count
}
