package combat

// Draws what terrain.go made: the pictures of the ground and the roads.

import (
    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
)

// the frames of a picture of the ground
func (combat *CombatScreen) groundTilePictures(lbx string, picture int) []*ebiten.Image {
    pictures, _ := combat.ImageCache.GetImages(lbx, picture)
    return pictures
}

// the frames of the picture of a tree, a rock or a house
func (combat *CombatScreen) sceneryPictures(piece *SceneryPiece) []*ebiten.Image {
    pictures, _ := combat.ImageCache.GetImages(piece.Lbx, piece.Index)
    return pictures
}

// the pieces of road of a tile, drawn with the options of its ground picture. the original steps
// their frames once a redraw
func (combat *CombatScreen) drawRoad(screen *ebiten.Image, x int, y int, options *ebiten.DrawImageOptions) {
    ground := combat.Model.Ground
    if ground == nil {
        return
    }

    for _, entry := range ground.roadPictures(ground.RoadOfTile(x, y)) {
        pictures, err := combat.ImageCache.GetImages(roadLbx, entry)
        if err != nil || len(pictures) == 0 {
            continue
        }
        scale.DrawScaled(screen, pictures[combat.originalTick() % uint64(len(pictures))], options)
    }
}
