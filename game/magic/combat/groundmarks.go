package combat

// Corpses and what is marked on the ground (user, 2026-09-29).
//
// THE MARKS OF THE GROUND LIE OVER THE CORPSES: where a unit can go, the outline of the cell under
// the mouse and the one of the selected unit are drawn on the ground, and corpses, which are
// drawn in order with the units, covered them. The marks are drawn a second time on a picture of
// their own (Marks.Layer), and every corpse that is drawn gets the part of that picture that
// lies on ITS OWN PIXELS drawn over it. So a unit that stands in front of a corpse covers both
// as before, and a unit is never under a mark.
//
// WHILE TAB IS HELD THE CORPSES ARE GONE, faded out and in over corpseHideTime, so the units and
// the ground are seen alone.

import (
    "image"
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: corpses cover what is marked on the ground
const MarksOverCorpses = true
// false: Tab does not hide the corpses
const TabHidesCorpses = true
// seconds the corpses take to go and to come back
const corpseHideTime = 0.12

type groundMarks struct {
    // what is marked on the ground in this frame, as large as the screen
    Layer *ebiten.Image
    // something is on it
    Drawn bool
    // a corpse with its marks, before it goes on the screen
    Piece *ebiten.Image
    // how far the corpses are hidden, 0 to 1, and the tick it was moved at
    Hidden float64
    HiddenTick uint64
}

// once a tick of the battle
func (combat *CombatScreen) updateCorpseHide() {
    marks := &combat.marks
    if marks.HiddenTick == combat.Counter {
        return
    }
    marks.HiddenTick = combat.Counter

    step := 1 / (corpseHideTime * float64(max(1, ebiten.TPS())))
    if TabHidesCorpses && combat.TabHeld {
        marks.Hidden = min(1, marks.Hidden + step)
    } else {
        marks.Hidden = max(0, marks.Hidden - step)
    }
}

// how much of the corpses shows, 1 to 0
func (combat *CombatScreen) corpsesShow() float64 {
    return 1 - combat.marks.Hidden
}

// the picture the marks of this frame are drawn on a second time, nil for none
func (combat *CombatScreen) startGroundMarks(screen *ebiten.Image) *ebiten.Image {
    marks := &combat.marks
    marks.Drawn = false
    if !MarksOverCorpses || len(combat.corpses) == 0 {
        return nil
    }

    size := screen.Bounds().Size()
    if marks.Layer == nil || marks.Layer.Bounds().Size() != size {
        if marks.Layer != nil {
            marks.Layer.Deallocate()
        }
        marks.Layer = ebiten.NewImage(size.X, size.Y)
    }
    marks.Layer.Clear()
    marks.Drawn = true
    return marks.Layer
}

// the marks of the ground on the pixels of a corpse that was drawn with this picture and matrix.
// shows is how much of the corpse shows
func (combat *CombatScreen) drawMarksOnCorpse(screen *ebiten.Image, picture *ebiten.Image, geom ebiten.GeoM, shows float64) {
    marks := &combat.marks
    if !marks.Drawn || marks.Layer == nil || shows <= 0 {
        return
    }

    // where the corpse is on the screen
    size := picture.Bounds().Size()
    left, top := math.Inf(1), math.Inf(1)
    right, bottom := math.Inf(-1), math.Inf(-1)
    for _, corner := range []image.Point{{0, 0}, {size.X, 0}, {0, size.Y}, {size.X, size.Y}} {
        x, y := geom.Apply(float64(corner.X), float64(corner.Y))
        left, top = min(left, x), min(top, y)
        right, bottom = max(right, x), max(bottom, y)
    }

    area := image.Rect(int(math.Floor(left)), int(math.Floor(top)), int(math.Ceil(right)), int(math.Ceil(bottom)))
    area = area.Intersect(marks.Layer.Bounds())
    if area.Empty() {
        return
    }

    if marks.Piece == nil || marks.Piece.Bounds().Dx() < area.Dx() || marks.Piece.Bounds().Dy() < area.Dy() {
        if marks.Piece != nil {
            marks.Piece.Deallocate()
        }
        marks.Piece = ebiten.NewImage(max(area.Dx(), scale.Scale(corpseTurnedSize)), max(area.Dy(), scale.Scale(corpseTurnedSize)))
    }
    piece := marks.Piece.SubImage(image.Rect(0, 0, area.Dx(), area.Dy())).(*ebiten.Image)
    piece.Clear()

    // the corpse, then the marks where the corpse is
    var body ebiten.DrawImageOptions
    body.GeoM = geom
    body.GeoM.Translate(float64(-area.Min.X), float64(-area.Min.Y))
    body.ColorScale.ScaleAlpha(float32(shows))
    piece.DrawImage(picture, &body)

    var inside ebiten.DrawImageOptions
    inside.Blend = ebiten.BlendSourceIn
    piece.DrawImage(marks.Layer.SubImage(area).(*ebiten.Image), &inside)

    var onto ebiten.DrawImageOptions
    onto.GeoM.Translate(float64(area.Min.X), float64(area.Min.Y))
    screen.DrawImage(piece, &onto)
}
