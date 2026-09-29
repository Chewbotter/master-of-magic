package combat

// Corpses do not pile up without end. Not in the original, which has no corpses.
//
// Every figure that falls stays as a corpse (figurefall.go). Where many fall on one tile they
// cover each other and the ground. A tile keeps its newest corpses, corpsesPerTile of them (4; 6 was the first try); once
// it has more, its oldest fade out, slowly, and are gone (user, 2026-09-28).

import (
    "image"
    "math"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: all corpses stay
const CorpseFade = true
// corpses a tile keeps
const corpsesPerTile = 4
// seconds a corpse takes to fade out
const corpseFadeTime = 3.0

// the tile a corpse lies on
func corpseTile(body *corpse) image.Point {
    field := MakeBattlefieldMatrix()
    field.Invert()
    x, y := field.Apply(body.X + body.PushX, body.Y + body.PushY)
    return image.Pt(int(math.Round(x)), int(math.Round(y)))
}

// the corpses that are to fade: of every tile the oldest, as many as the tile has too many.
// tiles is the tile of each corpse in the order they fell, fading says which of them fade already.
// returns the places in that order of the ones that start to fade
func corpsesToFade(tiles []image.Point, fading []bool, keep int) []int {
    staying := make(map[image.Point]int)
    for index, tile := range tiles {
        if !fading[index] {
            staying[tile] += 1
        }
    }

    var out []int
    for index, tile := range tiles {
        if fading[index] || staying[tile] <= keep {
            continue
        }
        staying[tile] -= 1
        out = append(out, index)
    }

    return out
}

// how much of a corpse shows, 1 to 0
func corpseShows(body *corpse, counter uint64) float64 {
    if !body.Fading {
        return 1
    }

    ticks := corpseFadeTime * float64(max(1, ebiten.TPS()))
    gone := 0.0
    if counter > body.FadeStart {
        gone = float64(counter - body.FadeStart) / ticks
    }

    // slowly at first
    return 1 - easeInOut(min(1, gone))
}

// lets the oldest corpses of full tiles fade and takes away the ones that are gone. once a tick
func (combat *CombatScreen) fadeCorpses() {
    if !CorpseFade || len(combat.corpses) == 0 || combat.corpseFadeTick == combat.Counter {
        return
    }
    combat.corpseFadeTick = combat.Counter

    kept := combat.corpses[:0]
    for _, body := range combat.corpses {
        if body.Fading && corpseShows(&body, combat.Counter) <= 0 {
            continue
        }
        kept = append(kept, body)
    }
    combat.corpses = kept

    tiles := make([]image.Point, len(combat.corpses))
    fading := make([]bool, len(combat.corpses))
    for index := range combat.corpses {
        tiles[index] = corpseTile(&combat.corpses[index])
        fading[index] = combat.corpses[index].Fading
    }

    for _, index := range corpsesToFade(tiles, fading, corpsesPerTile) {
        combat.corpses[index].Fading = true
        combat.corpses[index].FadeStart = combat.Counter
    }
}
