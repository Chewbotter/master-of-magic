package combat

// Props: small things on the battlefield that are neither trees nor rocks, as a patch of flowers,
// a bush, a stump, bones (user, 2026-09-28). Not in the original, and the game has no pictures of
// them: they are there when the replacement folder has some (mod/environment.go).
//
//   environment/Grass/flat 1_0.png, flat 2_0.png, ...   lies on the ground, under the units
//   environment/Grass/prop 1_0.png, prop 2_0.png, ...   stands, in one order with units and trees
//
// From 1 on without a gap, for every landscape. Any size. More frames (_1, _2, ...) make them
// move. A flat one lies with its middle on its place. A standing one stands on the middle of its
// width, propAnchorBelow above its bottom edge, as a rock does.
//
// They are scattered one by one over grass without a road, outside of the town and the lair and
// off the large pieces, as rocks are. They are for the look only and cost nothing to walk through.

import (
    "github.com/hajimehoshi/ebiten/v2"
)

// turns them off
var Props = true

const propFlatName = "flat"
const propStandingName = "prop"

// how many there are on the original's screen, of each kind. beyond it as many for its size
const propsFlat = 8
const propsStanding = 5

// the point of a standing prop that is put on its place is this far above its bottom edge
const propAnchorBelow = 1

// places within a cell, in steps of 2 across and 1 down along each side of the diamond
const propSubcellMax = 6

// props of one kind, scattered one by one
func scatterProps(count int, kind SceneryKind, pool sceneryPool, zone ZoneType, area sceneryArea, ground *BattleGround) []SceneryPiece {
    var out []SceneryPiece
    if pool.Count + pool.Extras <= 0 {
        return out
    }

    tries := 0
    for len(out) < count && tries < sceneryMaxTries {
        cgx, cgy := area.randomCell()
        screenX, screenY := cellScreen(cgx, cgy, roll(propSubcellMax), roll(propSubcellMax))

        if sceneryCellTaken(zone, cgx, cgy, true) || !ground.sceneryAllowed(cgx, cgy) || !area.Accept(screenX, screenY) {
            tries += 1
            continue
        }

        out = append(out, pool.piece(kind, screenX, screenY))
    }

    return out
}

// the props of a battlefield, of the pictures of the folder of its landscape
func makeProps(width int, height int, set string, zone ZoneType, ground *BattleGround) []SceneryPiece {
    var out []SceneryPiece
    if !Props {
        return out
    }

    beyond := sceneryBeyondScreen(width, height)
    kinds := []struct {
        Kind SceneryKind
        Name string
        Count int
    }{
        {Kind: SceneryFlat, Name: propFlatName, Count: propsFlat},
        {Kind: SceneryProp, Name: propStandingName, Count: propsStanding},
    }

    for _, kind := range kinds {
        // the game has none of its own
        pool := makeSceneryPool(set, kind.Name, "", 0, 0)
        out = append(out, scatterProps(kind.Count, kind.Kind, pool, zone, originalArea(), ground)...)
        out = append(out, scatterProps(kind.Count * beyond, kind.Kind, pool, zone, beyondArea(width, height), ground)...)
    }

    return out
}

// the props that lie on the ground, over it and under everything that stands
func (combat *CombatScreen) drawFlatProps(screen *ebiten.Image, animationIndex uint64) {
    for index := range combat.Model.Scenery {
        piece := &combat.Model.Scenery[index]
        if piece.Kind != SceneryFlat {
            continue
        }

        brightness := fieldBrightness(piece.ScreenX, piece.ScreenY)
        if brightness <= 0 {
            continue
        }

        pictures := combat.sceneryPictures(piece)
        if len(pictures) == 0 {
            continue
        }
        picture := pictures[animationIndex % uint64(len(pictures))]
        anchorX, anchorY := piece.anchor(picture.Bounds().Dx(), picture.Bounds().Dy())
        combat.drawOnFieldDimmed(screen, picture, piece.ScreenX - anchorX, piece.ScreenY - anchorY, brightness)
    }
}
