package combat

// Draws what terrain.go made: the pictures of the ground, the game's and the added ones, and the
// roads.

import (
    "fmt"
    "strings"
    "image"

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

// PICTURES OF 2 BY 2 TILES, an experiment, see mod/highres.go. The frames of such a picture are
// its four quarters. A tile shows the quarter of where it lies among the four tiles of its block,
// the blocks being the cells of the grid two by two. The tiles of a block show the quarters of
// one picture where they are of one use: the one of the first cell of the block.

// which of the tiles of a block a picture of the ground is for, along the two sides of the grid:
// the edge of dirt lies on the side of the block it lies on in the tile. -1 for a side that does
// not matter, where the tile lies in its block then
func quarterPlace(name string) (int, int) {
    switch strings.TrimPrefix(name, "dirt ") {
        case "edge SE": return 1, -1
        case "edge NW": return 0, -1
        case "edge SW": return -1, 1
        case "edge NE": return -1, 0
        case "edges NE NW", "corner N": return 0, 0
        case "edges NE SE", "corner E": return 1, 0
        case "edges SW NW", "corner W": return 0, 1
        case "edges SE SW", "corner S": return 1, 1
    }
    return -1, -1
}

// true if two pictures of the ground are of one use: both grass, or both dirt
func sameGroundUse(a int, b int) bool {
    for _, first := range []int{groundGrassFirst, groundDirtFirst} {
        if a >= first && a < first + groundGrassCount {
            return b >= first && b < first + groundGrassCount
        }
    }
    return false
}

// the picture of the ground of a tile of the field or of its border, at a time
func (combat *CombatScreen) groundFrame(lbx string, picture int, x int, y int, animationIndex uint64) *ebiten.Image {
    ground := combat.Model.Ground
    if ground != nil && picture < groundSandFirst {
        cgx, cgy := TileToCell(x, y)
        across := cgx & 1
        down := cgy & 1

        // the picture of the block, if the tile is of its use
        first := ground.PictureOfTile(CellToTile(cgx - across, cgy - down))
        if sameGroundUse(picture, first) {
            picture = first
        }

        if combat.quartered(lbx, picture) {
            pictures := combat.groundTilePictures(lbx, picture)
            if len(pictures) == mod.Quarters {
                placeAcross, placeDown := quarterPlace(mod.GroundName(lbx, picture))
                if placeAcross >= 0 {
                    across = placeAcross
                }
                if placeDown >= 0 {
                    down = placeDown
                }
                return pictures[mod.QuarterFrame(across, down)]
            }
        }
    }

    pictures := combat.groundTilePictures(lbx, picture)
    if len(pictures) == 0 {
        return nil
    }
    return pictures[animationIndex % uint64(len(pictures))]
}

// true if the frames of an entry are the quarters of a picture. looked up once a battle
func (combat *CombatScreen) quartered(lbx string, index int) bool {
    key := fmt.Sprintf("%v/%v", lbx, index)
    quartered, ok := combat.quarteredEntries[key]
    if !ok {
        quartered = mod.Quartered(lbx, index)
        if combat.quarteredEntries == nil {
            combat.quarteredEntries = make(map[string]bool)
        }
        combat.quarteredEntries[key] = quartered
    }
    return quartered
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
        // the crops of farmland, see farmland.go
        if crop := groundVariant(picture, groundCropFirst, groundCropCount); crop >= 0 {
            return combat.cropPictures(lbx, crop)
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
// game's, as ice on tundra. an added one that can not be read is the first of the game
func (combat *CombatScreen) coastPictures(name string, number int, count int, lbx string, first int) []*ebiten.Image {
    ground := combat.Model.Ground
    set := ground.SandSet
    if name == coastWaterName {
        set = ground.WaterSet
    }

    pictures := combat.addedPictures(set, name, number + 1)
    if len(pictures) > 0 {
        return pictures
    }
    if number >= count {
        number = 0
    }
    pictures, _ = combat.ImageCache.GetImages(lbx, first + number)

    if name != coastWaterName || !ground.Frozen || len(pictures) == 0 {
        return pictures
    }

    // ice, made of the water as it lies still. see mod/shore.go
    if made, ok := combat.iceMade[number]; ok {
        return made
    }
    bounds := pictures[0].Bounds()
    pixels := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
    pictures[0].ReadPixels(pixels.Pix)
    made := []*ebiten.Image{ebiten.NewImageFromImage(mod.IceTile(pixels))}

    if combat.iceMade == nil {
        combat.iceMade = make(map[int][]*ebiten.Image)
    }
    combat.iceMade[number] = made
    return made
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
