package spellbook

// The turn of a page of the spellbook.
//
// The original has 4 pictures of an empty leaf that turns. The fork drew the letters of the page
// on those, bent in 5 strips at the size of the screen. That gave 4 poses for the whole turn,
// kinks in the rows, letters beyond the rim of the leaf and pixels that did not fit the rest of
// the art.
//
// Here the leaf is made by the game (user, 2026-09-28). It is a sheet that is fixed to the spine
// and goes over from one side of the book to the other, with a bulge that runs through it, the
// way the original's leaf has it. Its two sides show the two pages it carries, paper and letters,
// as they are when they lie in the book, so the turn starts and ends with exactly what is there
// before and after it. It moves with every tick.
//
// The leaf is drawn in strips that are one art pixel wide and are moved up and down by whole art
// pixels, so all of its pixels are pixels of the art: a strip of the page that is seen from the
// side is left out, not pressed together.
//
// ProceduralPageTurn false gives the old turn back.

import (
    "image"
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
)

const ProceduralPageTurn = true

// ticks a turn takes
const PageTurnTicks = 30
// how much the leaf bulges, in parts of half a turn. 0: a stiff board
const pageTurnBulge = 0.3
// how many art pixels up the screen a point of the leaf is for every art pixel it is above the book
const pageTurnLift = 0.36
// the points of the leaf that are looked at, for every art pixel of its width
const pageTurnSamples = 4
// the side of the leaf that is turned away from the light is this much darker at most, 0 to 1,
// in this many steps
const pageTurnShade = 0.4
const pageTurnShadeSteps = 12
// the shadow of the leaf on the page under it, where the leaf is lifted off it: how dark at most
// 0 to 1, and for how many art pixels of lift
const pageTurnShadow = 0.3
const pageTurnShadowFull = 30.0
// the rim of the leaf is this much darker, 0 to 1
const pageTurnRim = 0.3

// where the light comes from: from the left and above
const pageTurnLightX = -0.35
const pageTurnLightZ = 0.94

// a leaf of a book: where its spine is on the screen and how large a page is, in art pixels
type pageLeaf struct {
    // the first column of the right page, the left page ends 2 before it
    RightX int
    // the last column of the left page
    LeftX int
    Top int
    Width int
    Height int
}

// one strip of the leaf, one art pixel wide
type leafStrip struct {
    // the column on the screen, in art pixels from the spine: 0 and up on the right page, below
    // 0 on the left
    Column int
    // the column of the page that shows in it, counted from the spine
    Source int
    // the side of the leaf that is seen is its back: the page that will lie on the left
    Back bool
    // art pixels it is moved up the screen
    Lift int
    // how bright it is, 0 to 1
    Shade float32
    // it is the outer rim of the leaf
    Rim bool
}

func easeTurn(part float64) float64 {
    part = max(0, min(1, part))
    return part * part * (3 - 2 * part)
}

// how far a turn has come, 0 to 1. ticks since it started, and the ticks it takes
func pageTurnPart(ticks uint64, all uint64) float64 {
    if all == 0 {
        return 1
    }
    return min(1, float64(ticks) / float64(all))
}

// the leaf comes down in the second half of the turn
const pageComesUpFrom = 0.5

// the old turn: how much of the new page shows on the side the leaf comes down on, 0 to 1
func pageComesUp(turned float64) float32 {
    if turned <= pageComesUpFrom {
        return 0
    }

    part := (turned - pageComesUpFrom) / (1 - pageComesUpFrom)
    // slowly at first and at the end
    return float32(part * part * (3 - 2 * part))
}

// how much a side of the leaf that is turned by an angle is lit, 0 to 1. flat in the book it is 1
func leafLight(angle float64, back bool) float64 {
    // the way the side faces, across and up
    normalX := -math.Sin(angle)
    normalZ := math.Cos(angle)
    if back {
        normalX = -normalX
        normalZ = -normalZ
    }

    lit := max(0, normalX * pageTurnLightX + normalZ * pageTurnLightZ) / pageTurnLightZ
    lit = min(1, lit)

    // in steps
    shade := math.Round((1 - lit) * pageTurnShadeSteps) / pageTurnShadeSteps
    return 1 - shade * pageTurnShade
}

// the strips of a leaf of a width that has turned by a part of 0, on the right, to 1, on the left.
// mirror: the leaf is turned back, from 0 on the left to 1 on the right, and does what a leaf
// that is turned forward does seen in a mirror. without it a leaf that is turned back would run
// the turn forward backward, which is not how a leaf goes over (user, 2026-09-28)
func leafStrips(width int, turned float64, mirror bool) []leafStrip {
    turned = max(0, min(1, turned))

    type found struct {
        strip leafStrip
        height float64
        source float64
        has bool
    }
    // by column, from -width to width - 1
    columns := make([]found, width * 2)

    step := 1.0 / pageTurnSamples
    across := 0.0
    height := 0.0

    base := math.Pi * turned
    bulge := math.Pi * pageTurnBulge * math.Sin(math.Pi * turned)

    for sample := range width * pageTurnSamples {
        along := (float64(sample) + 0.5) * step
        // the part of the leaf near the spine is ahead, the part near the rim behind
        angle := base + bulge * math.Cos(math.Pi * along / float64(width))

        across += math.Cos(angle) * step
        height += math.Sin(angle) * step
        height = max(0, height)

        column := int(math.Floor(across))
        if mirror {
            column = int(math.Floor(-across))
            // seen in a mirror the leaf stands the other way
            angle = math.Pi - angle
        }
        index := column + width
        if index < 0 || index >= len(columns) {
            continue
        }

        // what is higher is nearer to who looks at the book
        if columns[index].has && columns[index].height > height {
            continue
        }

        back := math.Cos(angle) < 0
        columns[index] = found{
            has: true,
            height: height,
            source: along,
            strip: leafStrip{
                Column: column,
                Source: min(width - 1, int(along)),
                Back: back,
                Lift: int(math.Round(height * pageTurnLift)),
                Shade: float32(leafLight(angle, back)),
            },
        }
    }

    var out []leafStrip
    rim := -1
    rimSource := -1.0
    for _, column := range columns {
        if !column.has {
            continue
        }
        out = append(out, column.strip)
        if column.source > rimSource {
            rimSource = column.source
            rim = len(out) - 1
        }
    }

    if rim >= 0 && turned > 0 && turned < 1 {
        out[rim].Rim = true
    }

    return out
}

// draws a leaf that has turned by a part of 0, on the right, to 1, on the left, or with mirror
// from 0 on the left to 1 on the right. front is the page
// it shows while it lies on the right, back the one it shows when it lies on the left, both as
// large as a page at the size of the screen. pixel is a picture of one white pixel
func drawLeaf(screen *ebiten.Image, leaf pageLeaf, front *ebiten.Image, back *ebiten.Image, turned float64, mirror bool, alpha float32, pixel *ebiten.Image) {
    size := scale.ScaleAmount
    strips := leafStrips(leaf.Width, turned, mirror)

    place := func(strip leafStrip) float64 {
        if strip.Column >= 0 {
            return float64(leaf.RightX + strip.Column)
        }
        return float64(leaf.LeftX + 1 + strip.Column)
    }

    // the shadow of the leaf on the page under it, where the leaf is lifted off it
    for _, strip := range strips {
        if strip.Lift <= 0 {
            continue
        }

        dark := pageTurnShadow * min(1, float64(strip.Lift) / pageTurnShadowFull)
        dark = math.Ceil(dark * pageTurnShadeSteps) / pageTurnShadeSteps

        var options ebiten.DrawImageOptions
        options.GeoM.Scale(size, float64(strip.Lift) * size)
        options.GeoM.Translate(place(strip) * size, float64(leaf.Top + leaf.Height - strip.Lift) * size)
        options.ColorScale.Scale(0, 0, 0, 1)
        options.ColorScale.ScaleAlpha(float32(dark) * alpha)
        screen.DrawImage(pixel, &options)
    }

    for _, strip := range strips {
        face := front
        source := strip.Source
        if strip.Back {
            face = back
            // the page that will lie on the left has the spine on its right
            source = leaf.Width - 1 - strip.Source
        }
        if face == nil {
            continue
        }

        left := int(math.Round(float64(source) * size))
        right := int(math.Round(float64(source + 1) * size))
        part := face.SubImage(image.Rect(left, 0, right, face.Bounds().Dy())).(*ebiten.Image)

        shade := strip.Shade
        if strip.Rim {
            shade *= 1 - pageTurnRim
        }

        x := place(strip) * size
        y := float64(leaf.Top - strip.Lift) * size

        var options ebiten.DrawImageOptions
        options.GeoM.Translate(x, y)
        options.ColorScale.Scale(shade, shade, shade, 1)
        options.ColorScale.ScaleAlpha(alpha)
        screen.DrawImage(part, &options)

        if strip.Lift > 0 {
            // the lower rim of the leaf
            var rim ebiten.DrawImageOptions
            rim.GeoM.Scale(size, size)
            rim.GeoM.Translate(x, float64(leaf.Top + leaf.Height - strip.Lift - 1) * size)
            rim.ColorScale.Scale(0, 0, 0, 1)
            rim.ColorScale.ScaleAlpha(pageTurnRim * alpha)
            screen.DrawImage(pixel, &rim)
        }
    }
}

// a page as it lies in the book, paper and letters, at the size of the screen. region is the
// page in the picture of the book, in art pixels. draw puts the letters on it, at a place in art
// pixels from the corner of the page
func makeLeafFace(book *ebiten.Image, region image.Rectangle, draw func(face *ebiten.Image, options ebiten.DrawImageOptions)) *ebiten.Image {
    width, height := scale.Scale2(region.Dx(), region.Dy())
    face := ebiten.NewImage(width, height)

    var paper ebiten.DrawImageOptions
    paper.GeoM.Translate(float64(-region.Min.X), float64(-region.Min.Y))
    scale.DrawScaled(face, book, &paper)

    if draw != nil {
        draw(face, ebiten.DrawImageOptions{})
    }

    return face
}
