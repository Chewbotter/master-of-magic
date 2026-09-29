package main

// The markers the mod makes by itself, written out so they can be repainted. See game/magic/mod/markers.go

import (
    "path/filepath"

    "github.com/kazzmir/master-of-magic/game/magic/mod"
    "github.com/kazzmir/master-of-magic/lib/lbx"
)

func exportMarkers(outPath string) int {
    flag := mod.DefaultFlag(lbx.GetDefaultPalette())
    if writePng(filepath.Join(outPath, "markers", mod.FlagFile), flag) != nil {
        return 0
    }

    return 1
}
