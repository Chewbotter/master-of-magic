package combat

// Corpses do not pile up without end. Not in the original, which has no corpses.
//
// Every figure that falls stays as a corpse (figurefall.go). Where many fall on one tile they
// cover each other and the ground. A tile keeps its newest corpses, corpsesPerTile of them (4; 6 was the first try); once
// it has more, its oldest fade out, slowly, and are gone (user, 2026-09-28).
//
// The corpse of a unit that is a single figure, as an earth elemental or a hero, never fades and
// is not counted: such units are special, and what falls on their tile after them does not take
// their place.

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

// true for the corpse of a unit of this many figures when it is whole: it never fades
func corpseLasts(figures int) bool {
    return figures <= 1
}

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

    // the ones that can fade, in the order they fell
    var places []int
    var tiles []image.Point
    var fading []bool
    for index := range combat.corpses {
        body := &combat.corpses[index]
        if body.Lasts {
            continue
        }
        places = append(places, index)
        tiles = append(tiles, corpseTile(body))
        fading = append(fading, body.Fading)
    }

    for _, index := range corpsesToFade(tiles, fading, corpsesPerTile) {
        body := &combat.corpses[places[index]]
        body.Fading = true
        body.FadeStart = combat.Counter
    }
}
