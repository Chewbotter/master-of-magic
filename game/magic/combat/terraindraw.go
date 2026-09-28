package combat

// Draws what terrain.go made: the pictures of the ground, the game's and the added ones, and the
// roads.

import (
    "github.com/kazzmir/master-of-magic/game/magic/mod"
    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
)

// the frames of a picture of the ground: of the archive, or one the replacement folder adds
func (combat *CombatScreen) groundTilePictures(lbx string, picture int) []*ebiten.Image {
    role, number, added := groundExtra(picture)
    if !added {
        pictures, _ := combat.ImageCache.GetImages(lbx, picture)
        return pictures
    }

    if pictures, ok := combat.groundExtras[picture]; ok {
        return pictures
    }

    var pictures []*ebiten.Image
    if combat.Model.Ground != nil {
        for _, source := range mod.ReadGroundExtra(combat.Model.Ground.Set, role, number) {
            pictures = append(pictures, ebiten.NewImageFromImage(source))
        }
    }
    // a picture that can not be read any more: the first of the game for its use
    if len(pictures) == 0 {
        pictures, _ = combat.ImageCache.GetImages(lbx, role.First)
    }

    if combat.groundExtras == nil {
        combat.groundExtras = make(map[int][]*ebiten.Image)
    }
    combat.groundExtras[picture] = pictures
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
