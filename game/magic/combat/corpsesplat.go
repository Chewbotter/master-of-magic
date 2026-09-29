package combat

// Blood on the corpses. Not in the original.
//
// A figure that is killed in a fight lies gray, as one that a spell without a color of its own
// has killed (user, 2026-09-28; red was tried first). The blood is what is red: a drop that comes
// down on a corpse that lies stays on it as a pixel, on top of the corpse, in the color of the
// stains. It belongs to the corpse: it is drawn with it, under what stands in front of it, and
// fades with it. The drop leaves its stain on the ground as well, which shows when the corpse is
// gone.

import (
    "image/color"
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/game/magic/unitview"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: blood is on the ground only
const CorpseSplats = true
// no more pixels of blood on one corpse than this
const corpseSplatsMost = 60

// a pixel of blood on a corpse, by its place in the picture of the corpse
type corpseSplat struct {
    X int
    Y int
    Color color.RGBA
}

// the picture of a corpse that lies and where its top left corner is on the original's screen.
// false if it does not lie yet
func (combat *CombatScreen) corpseLying(body *corpse) (*ebiten.Image, int, int, bool) {
    seconds := combat.corpseTime(body)
    if seconds < figureFallTime {
        return nil, 0, 0, false
    }

    picture := body.Dead
    var offsetX, offsetY float64
    if picture != nil {
        offsetX, offsetY = unitview.FigureOffset(picture)
    } else {
        if body.turned == nil || !body.lies {
            return nil, 0, 0, false
        }
        picture = body.turned
        offsetX, offsetY = -corpseTurnedSize / 2, -corpseTurnedSize / 2
    }

    x := math.Round(body.X + body.PushX)
    y := math.Round(body.Y + body.PushY)
    return picture, int(math.Round(x + offsetX)), int(math.Round(y + offsetY)), true
}

// which pixels of a picture are there, by row and column
func (combat *CombatScreen) pictureSolid(picture *ebiten.Image) []bool {
    effects := &combat.effects
    if solid, ok := effects.Solid[picture]; ok {
        return solid
    }

    bounds := picture.Bounds()
    pixels := make([]byte, 4 * bounds.Dx() * bounds.Dy())
    picture.ReadPixels(pixels)

    solid := make([]bool, bounds.Dx() * bounds.Dy())
    for index := range solid {
        solid[index] = pixels[index * 4 + 3] != 0
    }

    if effects.Solid == nil {
        effects.Solid = make(map[*ebiten.Image][]bool)
    }
    effects.Solid[picture] = solid
    return solid
}

// a drop of blood has come down at a place on the original's screen: it stays on the corpse that
// lies there, the newest if there are several
func (combat *CombatScreen) splatCorpse(x int, y int, tint color.RGBA) {
    if !CorpseSplats {
        return
    }

    for index := len(combat.corpses) - 1; index >= 0; index-- {
        body := &combat.corpses[index]
        if body.Fading || len(body.Splats) >= corpseSplatsMost {
            continue
        }

        picture, left, top, ok := combat.corpseLying(body)
        if !ok {
            continue
        }

        width := picture.Bounds().Dx()
        height := picture.Bounds().Dy()
        inX := x - left
        inY := y - top
        if inX < 0 || inY < 0 || inX >= width || inY >= height {
            continue
        }

        if !combat.pictureSolid(picture)[inY * width + inX] {
            continue
        }

        body.Splats = append(body.Splats, corpseSplat{X: inX, Y: inY, Color: tint})
        return
    }
}

// the blood on a corpse, over its picture. left and top is where the picture is on the original's
// screen, shows how much of the corpse shows, brightness how bright the ground is there
func (combat *CombatScreen) drawCorpseSplats(screen *ebiten.Image, body *corpse, left float64, top float64, shows float64, brightness float32) {
    if len(body.Splats) == 0 {
        return
    }

    pixel := combat.effectPixel()
    matrix := originalScreenMatrix(combat.GetCameraMatrix())
    strength := float32(shows * min(1, combat.effects.StainStrength))
    if combat.effects.StainStrength <= 0 {
        strength = float32(shows)
    }

    for _, splat := range body.Splats {
        var options ebiten.DrawImageOptions
        options.GeoM.Translate(left + float64(splat.X), top + float64(splat.Y))
        options.GeoM.Concat(matrix)
        options.ColorScale.Scale(float32(splat.Color.R) / 255 * brightness, float32(splat.Color.G) / 255 * brightness, float32(splat.Color.B) / 255 * brightness, 1)
        options.ColorScale.ScaleAlpha(strength)
        scale.DrawScaled(screen, pixel, &options)
    }
}
