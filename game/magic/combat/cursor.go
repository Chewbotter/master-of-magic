package combat

// The cursor pictures of the battlefield.
//
// The tile under the cursor is the one under the mouse position. The pictures of the cursor are 16
// by 16 and were drawn with their top left corner there, so the boot, the swords and the red X sat
// half a tile or more to the lower right of the tile they acted on, further when zoomed out. Each
// picture is now drawn with its own middle, or its tip, on the mouse position.
//
// The original picks the cell 4 right and 4 below the corner of the picture for every cursor
// (ReMoM: MoM/src/Combat.c Combat_Screen_Assign_Mouse_Images). With a view that zooms out that is
// not exact enough, so we differ from it here.

import (
    "image"

    globalMouse "github.com/kazzmir/master-of-magic/game/magic/mouse"
    "github.com/kazzmir/master-of-magic/game/magic/mod"
    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
)

// the size of a cursor picture in art pixels
const cursorPictureSize = 16

// the point of each picture that is put on the mouse position, in art pixels
var cursorHotMove = image.Pt(5, 3)
var cursorHotAttack = image.Pt(7, 7)
var cursorHotRanged = image.Pt(1, 1)
var cursorHotCross = image.Pt(7, 7)

// the red X: its arms are this many art pixels shorter than the original's 7
const crossShorten = 3
// the middle of the X, which stays as it is
const crossMiddleLeft = 5
const crossMiddleRight = 9
const crossMiddleTop = 5
const crossMiddleBottom = 8
// the column and row the arms are pulled toward
const crossCenterX = 7
const crossCenterY = 6

func (combat *CombatScreen) setCursor(picture *ebiten.Image, hot image.Point) {
    if picture == nil {
        return
    }

    globalMouse.Mouse.SetImageFunc(func(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
        var use ebiten.DrawImageOptions
        use.GeoM.Translate(float64(-hot.X), float64(-hot.Y))
        // as large as the tiles are drawn, so a picture never covers more of the field zoomed out
        use.GeoM.Scale(combat.CameraScale, combat.CameraScale)
        use.GeoM.Concat(options.GeoM)
        use.ColorScale = options.ColorScale
        scale.DrawScaled(screen, picture, &use)
    })
}

// the red X with shorter arms, at the same size of pixel. made once, from the original's picture
func (combat *CombatScreen) shortCross() *ebiten.Image {
    if combat.shortCrossPicture != nil {
        return combat.shortCrossPicture
    }

    source := combat.Mouse.Error
    if source == nil {
        return nil
    }

    if mod.HasCursor(mod.CursorCross) {
        // drawn by the user: as it is
        combat.shortCrossPicture = source
        return source
    }

    width := source.Bounds().Dx()
    height := source.Bounds().Dy()
    // the picture can be stored larger than its art
    size := max(1, width / cursorPictureSize)

    pixels := make([]byte, 4 * width * height)
    source.ReadPixels(pixels)
    out := make([]byte, len(pixels))

    for y := range height {
        for x := range width {
            from := (y * width + x) * 4
            if pixels[from + 3] == 0 {
                continue
            }

            artX := x / size
            artY := y / size
            toX := x
            toY := y

            middle := artX >= crossMiddleLeft && artX <= crossMiddleRight && artY >= crossMiddleTop && artY <= crossMiddleBottom
            if !middle {
                // an arm: along its diagonal toward the middle
                if artX < crossCenterX {
                    toX += crossShorten * size
                } else if artX > crossCenterX {
                    toX -= crossShorten * size
                }
                if artY <= crossCenterY {
                    toY += crossShorten * size
                } else {
                    toY -= crossShorten * size
                }
            }

            if toX < 0 || toY < 0 || toX >= width || toY >= height {
                continue
            }

            to := (toY * width + toX) * 4
            copy(out[to:to + 4], pixels[from:from + 4])
        }
    }

    picture := ebiten.NewImage(width, height)
    picture.WritePixels(out)
    combat.shortCrossPicture = picture
    return picture
}
