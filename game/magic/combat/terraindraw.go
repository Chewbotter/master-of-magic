package combat

// Draws what terrain.go made: the pictures of the ground, the game's and the added ones, and the
// roads.

import (
    "fmt"

    "github.com/kazzmir/master-of-magic/game/magic/mod"
    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
)

// the frames of a picture the replacement folder adds (mod/environment.go). read once a battle
func (combat *CombatScreen) addedPictures(set string, name string, number int) []*ebiten.Image {
    key := fmt.Sprintf("%v/%v/%v", set, name, number)
    if pictures, ok := combat.addedCache[key]; ok {
        return pictures
    }

    var pictures []*ebiten.Image
    for _, source := range mod.ReadExtra(set, name, number) {
        pictures = append(pictures, ebiten.NewImageFromImage(source))
    }

    if combat.addedCache == nil {
        combat.addedCache = make(map[string][]*ebiten.Image)
    }
    combat.addedCache[key] = pictures
    return pictures
}

// the frames of a picture of the ground: of the archive, or one the replacement folder adds
func (combat *CombatScreen) groundTilePictures(lbx string, picture int) []*ebiten.Image {
    // the beach and the water of a coast, see coast.go
    ground := combat.Model.Ground
    if ground != nil {
        if sand := groundVariant(picture, groundSandFirst, groundSandCount); sand >= 0 {
            return combat.coastPictures(coastSandName, sand, groundSandCount, ground.SandLbx, ground.SandFirst)
        }
        if water := groundVariant(picture, groundWaterFirst, groundWaterCount); water >= 0 {
            return combat.coastPictures(coastWaterName, water, groundWaterCount, ground.WaterLbx, ground.WaterFirst)
        }
    }

    role, number, added := groundExtra(picture)
    if added && combat.Model.Ground != nil {
        pictures := combat.addedPictures(combat.Model.Ground.Set, role.Name, number)
        if len(pictures) > 0 {
            return pictures
        }
    }
    if added {
        // a picture that can not be read any more: the first of the game for its use
        picture = role.First
    }

    pictures, _ := combat.ImageCache.GetImages(lbx, picture)
    return pictures
}

// the frames of a picture of a coast, by its number from 0: of the folder of the coast, or the
// game's. an added one that can not be read is the first of the game
func (combat *CombatScreen) coastPictures(name string, number int, count int, lbx string, first int) []*ebiten.Image {
    pictures := combat.addedPictures(combat.Model.Ground.CoastSet, name, number + 1)
    if len(pictures) > 0 {
        return pictures
    }
    if number >= count {
        number = 0
    }
    pictures, _ = combat.ImageCache.GetImages(lbx, first + number)
    return pictures
}

// the frames of the picture of a tree, a rock or a house
func (combat *CombatScreen) sceneryPictures(piece *SceneryPiece) []*ebiten.Image {
    if piece.Number > 0 {
        pictures := combat.addedPictures(piece.Set, piece.Name, piece.Number)
        if len(pictures) > 0 {
            return pictures
        }
    }

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
