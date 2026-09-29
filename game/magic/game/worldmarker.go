package game

// The markers on the path of the selected stack: a boot on the tiles the stack can walk to in this
// turn, a flag in the color of its wizard on the tiles it will only reach in a later turn.
// And the moment the camera rests on a stack that goes on along its path when a turn starts.

import (
    "image"
    "image/color"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/maplib"
    "github.com/kazzmir/master-of-magic/game/magic/mod"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    "github.com/kazzmir/master-of-magic/game/magic/util"
    "github.com/kazzmir/master-of-magic/lib/coroutine"
    "github.com/kazzmir/master-of-magic/lib/fraction"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: boots on every tile of a path, as upstream
const PathFlags = true

// seconds the camera rests on a stack that goes on along its path by itself, before it walks. 0: none
const AutoMoveHoldTime = 0.4

// how far along a path the stack gets in this turn: the number of points of the path it reaches,
// and the last of them. a point the stack stands on counts as reached
func (game *Game) reachAlong(player *playerlib.Player, stack *playerlib.UnitStack, path worldPath, mapUse *maplib.Map, getStack func(int, int) (playerlib.PathStack, bool)) (int, int, int, bool) {
    moves := stack.GetRemainingMoves()
    x := stack.X()
    y := stack.Y()
    found := false
    reached := 0

    for index, step := range path {
        if step.X == x && step.Y == y {
            reached = index + 1
            continue
        }

        if !moves.GreaterThan(fraction.Zero()) {
            break
        }

        cost, canMove := game.Model.ComputeTerrainCost(stack, x, y, step.X, step.Y, mapUse, getStack)
        if !canMove {
            break
        }
        cost = debugMoveCost(player, cost)

        x = step.X
        y = step.Y
        found = true
        reached = index + 1
        moves = moves.Subtract(cost)

        if mapUse.GetEncounter(mapUse.WrapX(step.X), step.Y) != nil {
            // the stack stops there
            break
        }
    }

    return reached, x, y, found
}

// the number of points of the path that get a boot. the ones after them get a flag
func (game *Game) pathReach(player *playerlib.Player, stack *playerlib.UnitStack, path worldPath) int {
    if !PathFlags || player == nil || stack == nil || len(path) == 0 {
        return len(path)
    }

    mapUse := game.GetMap(stack.Plane())
    getStack := func(x int, y int) (playerlib.PathStack, bool) {
        found := player.FindStack(mapUse.WrapX(x), y, stack.Plane())
        return found, found != nil
    }

    reached, _, _, _ := game.reachAlong(player, stack, path, mapUse, getStack)
    return reached
}

// the flag in the color of a wizard. made from the picture of the boot only to have the palette of
// the game and a place in the image cache, which is made anew with the pictures of the replacement folder
func flagPicture(imageCache *util.ImageCache, banner data.BannerType) *ebiten.Image {
    picture, err := imageCache.GetImageTransform("compix.lbx", 72, 0, "path-flag-" + banner.String(), func (boot *image.Paletted) image.Image {
        flag := mod.MarkerFlag(boot.Palette)
        flag.Palette = bannerShades(flag.Palette, banner)
        return flag
    })
    if err != nil {
        return nil
    }

    return picture
}

// the camera rests on the stack for a moment. called before a stack walks on by itself
func (game *Game) holdBeforeAutoMove(yield coroutine.YieldFunc) {
    ticks := int(AutoMoveHoldTime * float64(ebiten.TPS()))
    for tick := 0; tick < ticks; tick++ {
        game.Counter += 1
        if yield() != nil {
            return
        }
    }
}

// the colors of the palette that carry the color of a wizard, and how many of them there are
const bannerColorFirst = 215
const bannerColorCount = 4

// the palette with the four greens of the wizard's color turned into shades of the color of the banner.
// every shade keeps how light and how pale its green is, so four greens give four tones of any color.
// figures are colored by units.MakeUpdateUnitColorsFunc, which gives tones that are nearly one for the
// strong colors (red, blue, yellow)
func bannerShades(palette color.Palette, banner data.BannerType) color.Palette {
    out := util.ClonePalette(palette)
    if len(out) < bannerColorFirst + bannerColorCount {
        return out
    }

    // the color of the banner as the figures have it: their second shade is the color itself
    probe := image.NewPaletted(image.Rect(0, 0, 1, 1), util.ClonePalette(palette))
    colored, ok := units.MakeUpdateUnitColorsFunc(banner)(probe).(*image.Paletted)
    if !ok {
        return out
    }
    baseRed, baseGreen, baseBlue, _ := colored.Palette[bannerColorFirst + 1].RGBA()
    base := [3]float64{float64(baseRed >> 8), float64(baseGreen >> 8), float64(baseBlue >> 8)}
    strongest := max(base[0], base[1], base[2], 1)

    for index := bannerColorFirst; index < bannerColorFirst + bannerColorCount; index++ {
        red, green, blue, _ := palette[index].RGBA()
        shade := [3]float64{float64(red >> 8), float64(green >> 8), float64(blue >> 8)}
        light := max(shade[0], shade[1], shade[2])
        if light <= 0 {
            out[index] = color.RGBA{A: 0xff}
            continue
        }
        pale := min(shade[0], shade[1], shade[2]) / light

        var channels [3]uint8
        for channel := range channels {
            full := base[channel] / strongest
            channels[channel] = uint8(min(255, light * (pale + (1 - pale) * full) + 0.5))
        }
        out[index] = color.RGBA{R: channels[0], G: channels[1], B: channels[2], A: 0xff}
    }

    return out
}
