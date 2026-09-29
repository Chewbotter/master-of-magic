package combat

// The edge of the battlefield.
//
// Units fight on a field of 30 by 30 tiles. Around it lies a border of ground nobody can enter,
// each row of tiles darker than the one before, down to black at the last. The camera stops where
// its view would leave the border, so the black beyond is never seen as a cut edge.
// Not in the original, whose view never moves.

import (
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
)

// rows of tiles around the field. with 16 the view of a 16 to 9 window fits at the farthest zoom level
const BattlefieldBorder = 16

// the middle of the screen, in art pixels
const battleScreenCenterX = data.ScreenWidth / 2
const battleScreenCenterY = data.ScreenHeight / 2

// tiles past the edge of the view that are still drawn
const borderCullMargin = 2

// how many rows a tile lies outside of the field. 0 for a tile of the field
func borderDepth(x float64, y float64) float64 {
    depth := math.Max(-x, x - float64(BattlefieldWidth - 1))
    depth = math.Max(depth, math.Max(-y, y - float64(BattlefieldHeight - 1)))
    return math.Max(0, depth)
}

// how bright something that many rows outside of the field is drawn, 1 to 0
func borderBrightness(depth float64) float32 {
    return float32(math.Max(0, 1 - depth / BattlefieldBorder))
}

// the ground pictures of a landscape: file, first picture, number of pictures
func groundPictures(landscape CombatLandscape, plane data.Plane) (string, int, int) {
    if landscape == CombatLandscapeWater {
        if plane == data.PlaneArcanus {
            return "cmbtcity.lbx", 109, 4
        }
        return "chriver.lbx", 12, 4
    }

    return terrainSetLbx(landscape, plane), 0, 32
}

// the picture of a tile of the border: always the same for the same tile of a battle
func (combat *CombatScreen) borderPicture(x int, y int, count int) int {
    value := uint32(x + BattlefieldBorder) * 7919 + uint32(y + BattlefieldBorder) * 104729 + combat.borderSeed
    value ^= value >> 13
    value *= 2654435761
    value ^= value >> 16
    return int(value % uint32(count))
}

func (combat *CombatScreen) drawFieldBorder(screen *ebiten.Image, animationIndex uint64) {
    lbx, start, count := groundPictures(combat.Landscape, combat.Model.Plane)
    matrix := combat.GetCameraMatrix()

    // the picture being drawn on, in art pixels
    width := float64(screen.Bounds().Dx()) / scale.ScaleAmount
    height := float64(screen.Bounds().Dy()) / scale.ScaleAmount
    marginX := CellStepX * borderCullMargin * combat.CameraScale
    marginY := CellStepY * borderCullMargin * combat.CameraScale

    var options ebiten.DrawImageOptions

    for y := -BattlefieldBorder; y < BattlefieldHeight + BattlefieldBorder; y++ {
        for x := -BattlefieldBorder; x < BattlefieldWidth + BattlefieldBorder; x++ {
            depth := borderDepth(float64(x), float64(y))
            if depth == 0 {
                continue
            }

            brightness := borderBrightness(depth)
            if brightness <= 0 {
                continue
            }

            tx, ty := matrix.Apply(float64(x), float64(y))
            if tx < -marginX || ty < -marginY || tx > width + marginX || ty > height + marginY {
                continue
            }

            // the ground goes on from the field, see terrain.go. on the water a random picture
            index := start + combat.borderPicture(x, y, count)
            if combat.Model.Ground != nil {
                index = combat.Model.Ground.PictureOfTile(x, y)
            }

            picture := combat.groundFrame(lbx, index, x, y, animationIndex)
            if picture == nil {
                continue
            }

            options.GeoM.Reset()
            options.GeoM.Translate(TerrainOffsetX, TerrainOffsetY)
            options.GeoM.Scale(combat.CameraScale, combat.CameraScale)
            options.GeoM.Translate(tx, ty)
            options.ColorScale.Reset()
            options.ColorScale.Scale(brightness, brightness, brightness, 1)
            scale.DrawScaled(screen, picture, &options)
            combat.drawRoad(screen, x, y, &options)
        }
    }
}

// how bright something at a position of the original's screen is drawn
func fieldBrightness(screenX int, screenY int) float32 {
    matrix := MakeBattlefieldMatrix()
    matrix.Invert()
    x, y := matrix.Apply(float64(screenX), float64(screenY))
    return borderBrightness(borderDepth(x, y))
}

// keeps the view of the camera inside the border
func (battle *BattleCamera) clampToField() {
    if battle.Level <= 0 || scale.ScaleAmount <= 0 {
        return
    }

    zoom := float64(battle.Level) / scale.ScaleAmount

    // half of what the view shows of the field, in art pixels of the field
    halfWidth := float64(battleScreenCenterX + display.BarWidth()) / zoom
    halfHeight := float64(battleScreenCenterY) / zoom

    // the field with its border is a diamond around its middle, this far across and down
    field := MakeBattlefieldMatrix()
    middleX, middleY := field.Apply(float64(BattlefieldWidth - 1) / 2, float64(BattlefieldHeight - 1) / 2)
    reachX := float64(CellStepX * (BattlefieldWidth + BattlefieldBorder * 2))
    reachY := float64(CellStepY * (BattlefieldHeight + BattlefieldBorder * 2))

    // how far the middle of the view can be from the middle of the field, 1 is the tip of the diamond
    room := math.Max(0, 1 - halfWidth / reachX - halfHeight / reachY)

    panX := battle.targetX / scale.ScaleAmount
    panY := battle.targetY / scale.ScaleAmount

    // the point of the field in the middle of the view
    viewX := (battleScreenCenterX - BattleViewCenterX - panX) / zoom + BattleViewCenterX
    viewY := (battleScreenCenterY - BattleViewCenterY - panY) / zoom + BattleViewCenterY

    offX := viewX - middleX
    offY := viewY - middleY
    distance := math.Abs(offX) / reachX + math.Abs(offY) / reachY
    if distance <= room || distance == 0 {
        return
    }

    offX *= room / distance
    offY *= room / distance

    panX = battleScreenCenterX - BattleViewCenterX - zoom * (middleX + offX - BattleViewCenterX)
    panY = battleScreenCenterY - BattleViewCenterY - zoom * (middleY + offY - BattleViewCenterY)

    hitX := math.Abs(panX * scale.ScaleAmount - battle.targetX) > 0.5
    hitY := math.Abs(panY * scale.ScaleAmount - battle.targetY) > 0.5

    battle.targetX = panX * scale.ScaleAmount
    battle.targetY = panY * scale.ScaleAmount

    // a release glide ends at the edge
    if hitX || hitY {
        battle.velocityX = 0
        battle.velocityY = 0
        battle.coasting = false
    }
}
