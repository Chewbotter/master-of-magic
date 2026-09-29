package main

// Which build this is: the lane it was built from, its commit (with a + when files that were not
// committed are in it) and when. Set by build.sh (-ldflags -X main.buildStamp), shown under the debug
// list of the start screen, so "which build am I looking at" is answered on the screen.

import (
    "log"
)

var buildStamp = "no stamp (not built by build.sh)"

// rows between the last entry of the debug list and the stamp
const BuildStampGap = 1

func logBuildStamp() {
    log.Printf("Build: %v", buildStamp)
}
