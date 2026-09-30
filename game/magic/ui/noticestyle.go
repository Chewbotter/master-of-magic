package ui

// The box of a message (a warning) and the box that shows what a stack has found (a lair, a
// node, a tower), matched to the original game.
//
// The facts were read from the ReMoM project's reconstruction (MoX/src/GENDRAW.c: Warn,
// Warn_Draw; MoM/src/Lair.c: Lair_Confirm, Lair_Confirm_Draw). The code is ours. Upstream had
// both in yellow letters with a drop shadow of less than an art pixel in the middle of the box,
// at places that did not follow the height of the text.
//
// THE MESSAGE
//   pictures resource.lbx 38 and 39. the box at x 68, as far from the top of the screen as from
//     its bottom; of its upper part the rows of the text and 12 show, its lower part is 10
//     under the text
//   the text: font 4 in its color set 4 with soft edge 184, in lines of 166 each in the middle
//     (93 from the left of the box), 10 from its top; under it the same all in 228, a pixel
//     lower and a pixel lower and right
//   any click or key closes it
//
// WHAT WAS FOUND
//   pictures backgrnd.lbx 25, under it 26 (with buttons) or 27, the picture of the place at 7,7
//   the box at x 68, in the middle from top to bottom. the part for the picture and the text is
//     as high as the text, 75 when the text is lower than 70; 6 under it the lower part
//   the text: font 4 in its color set 4 with soft edge 18, from the left, the lines 2 further
//     apart than the font has them; next to the picture (the first 55 rows) from 58 to 175 of
//     the box, under it from 9 in lines of 166; its shadow as the message's, in 17
//   buttons: No 17 and Yes 100 from the left, 5 under the top of the lower part; the keys Y and
//     N, and Escape or a click beside the box for No

import (
    "image"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    fontslib "github.com/kazzmir/master-of-magic/game/magic/fonts"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/game/magic/util"
    "github.com/kazzmir/master-of-magic/lib/font"
    "github.com/kazzmir/master-of-magic/lib/lbx"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/inpututil"
)

// false: the boxes of upstream
var OriginalNotice = true
var OriginalLairBox = true

const (
    noticeLbx = "resource.lbx"
    noticeTopIndex = 38
    noticeBottomIndex = 39

    noticeX = 68
    noticeRoom = 33
    noticeTextMiddle = 93
    noticeTextY = 10
    noticeTextWidth = 166
    noticeTopRows = 12
    noticeBottomY = 10

    noticeEdge = 184
    noticeShadow = 228
)

const (
    lairLbx = "backgrnd.lbx"
    lairTopIndex = 25
    lairButtonsIndex = 26
    lairPlainIndex = 27

    lairX = 68
    lairRoom = 34
    lairPictureX = 7
    lairPictureY = 7
    // the part of picture and text is at least this high when the text is lower than lairLow
    lairLeast = 75
    lairLow = 70
    lairBottomDown = 6

    lairTextY = 10
    // next to the picture, and under it
    lairTextBesideX = 58
    lairTextBesideEnd = 175
    lairTextX = 9
    lairTextWidth = 166
    // the rows of the box the picture takes
    lairPictureRows = 55
    // lines are this much further apart than the font has them
    lairLineGap = 2

    lairYesX = 100
    lairNoX = 17
    lairButtonsDown = 5

    lairEdge = 18
    lairShadow = 17
)

// the box of a message, nil when it can not be made
func makeNoticeOriginal(ui UIContainer, cache *lbx.LbxCache, imageCache *util.ImageCache, message string, layer UILayer, clicked func()) *UIElement {
    if !OriginalNotice {
        return nil
    }

    original := fontslib.LoadOriginal(cache, noticeLbx, noticeTopIndex)
    top, err1 := imageCache.GetImage(noticeLbx, noticeTopIndex, 0)
    bottom, err2 := imageCache.GetImage(noticeLbx, noticeBottomIndex, 0)
    if original == nil || err1 != nil || err2 != nil {
        return nil
    }

    text := original.Shadowed(4, 4, noticeEdge, noticeShadow)
    if text == nil {
        return nil
    }

    lines := text.Wrap(message, noticeTextWidth)
    height := text.Height(lines)
    y := (data.ScreenHeight - (height + noticeRoom)) / 2
    rows := min(top.Bounds().Dy(), height + noticeTopRows)
    topPart := top.SubImage(image.Rect(0, 0, top.Bounds().Dx(), rows)).(*ebiten.Image)

    var element *UIElement
    close := func() {
        ui.RemoveElement(element)
        clicked()
    }

    element = &UIElement{
        Rect: image.Rect(0, 0, data.ScreenWidth, data.ScreenHeight),
        Layer: layer,
        LeftClick: func(this *UIElement){
            close()
        },
        RightClick: func(this *UIElement){
            close()
        },
        Draw: func(this *UIElement, window *ebiten.Image){
            var options ebiten.DrawImageOptions
            options.GeoM.Translate(noticeX, float64(y))
            scale.DrawScaled(window, topPart, &options)

            text.PrintLines(window, func(int) int { return noticeX + noticeTextMiddle }, y + noticeTextY, font.FontJustifyCenter, 1, lines)

            options.GeoM.Reset()
            options.GeoM.Translate(noticeX, float64(y + height + noticeBottomY))
            scale.DrawScaled(window, bottom, &options)
        },
    }

    return element
}

// the box of what was found. with confirm it asks: yes and no. without, any click lets go
func makeLairOriginal(ui UIContainer, cache *lbx.LbxCache, imageCache *util.ImageCache, lairPicture *util.Animation, layer UILayer, message string, asks bool, yes func(), no func()) []*UIElement {
    if !OriginalLairBox {
        return nil
    }

    original := fontslib.LoadOriginal(cache, "", 0)
    top, err1 := imageCache.GetImage(lairLbx, lairTopIndex, 0)
    bottomIndex := lairPlainIndex
    if asks {
        bottomIndex = lairButtonsIndex
    }
    bottom, err2 := imageCache.GetImage(lairLbx, bottomIndex, 0)
    yesButtons, err3 := imageCache.GetImages("resource.lbx", 3)
    noButtons, err4 := imageCache.GetImages("resource.lbx", 4)
    if original == nil || err1 != nil || err2 != nil || err3 != nil || err4 != nil || len(yesButtons) < 2 || len(noButtons) < 2 {
        return nil
    }

    text := original.Shadowed(4, 4, lairEdge, lairShadow)
    if text == nil {
        return nil
    }
    text.LineStep = text.Text.Height() + lairLineGap

    // a line is next to the picture while it starts in the rows of the picture
    beside := func(line int) bool {
        return lairTextY + line * text.LineStep < lairPictureRows
    }
    lines := text.Flow(message, func(line int) int {
        if beside(line) {
            return lairTextBesideEnd - lairTextBesideX
        }
        return lairTextWidth
    })

    height := text.Height(lines)
    y := (data.ScreenHeight - (lairRoom + height)) / 2
    part := height
    if part < lairLow {
        part = lairLeast
    }
    bottomY := y + part + lairBottomDown
    rows := min(top.Bounds().Dy(), bottomY - y)
    topPart := top.SubImage(image.Rect(0, 0, top.Bounds().Dx(), rows)).(*ebiten.Image)

    const fadeSpeed = 7
    getAlpha := ui.MakeFadeIn(fadeSpeed)

    var elements []*UIElement
    answered := false
    answer := func(then func()) {
        if answered {
            return
        }
        answered = true
        getAlpha = ui.MakeFadeOut(fadeSpeed)
        ui.AddDelay(fadeSpeed, func(){
            ui.RemoveElements(elements)
            then()
        })
    }

    box := image.Rect(lairX, y, lairX + top.Bounds().Dx(), bottomY + bottom.Bounds().Dy())

    // all of the screen: a click beside the box is a no, or lets the box go when it does not ask
    elements = append(elements, &UIElement{
        Rect: image.Rect(0, 0, data.ScreenWidth, data.ScreenHeight),
        Layer: layer,
        LeftClick: func(this *UIElement){
            if asks {
                answer(no)
            } else {
                answer(yes)
            }
        },
    })

    elements = append(elements, &UIElement{
        Rect: box,
        Layer: layer,
        Order: 1,
        LeftClick: func(this *UIElement){
            if !asks {
                answer(yes)
            }
        },
        Draw: func(this *UIElement, window *ebiten.Image){
            // the keys of the original
            if asks && !answered {
                switch {
                    case inpututil.IsKeyJustPressed(ebiten.KeyY): answer(yes)
                    case inpututil.IsKeyJustPressed(ebiten.KeyN), inpututil.IsKeyJustPressed(ebiten.KeyEscape): answer(no)
                }
            }

            var options ebiten.DrawImageOptions
            options.GeoM.Translate(lairX, float64(y))
            options.ColorScale.ScaleAlpha(getAlpha())
            scale.DrawScaled(window, topPart, &options)

            options.GeoM.Translate(lairPictureX, lairPictureY)
            scale.DrawScaled(window, lairPicture.Frame(), &options)

            options.GeoM.Reset()
            options.GeoM.Translate(lairX, float64(bottomY))
            scale.DrawScaled(window, bottom, &options)

            text.PrintLines(window, func(line int) int {
                if beside(line) {
                    return lairX + lairTextBesideX
                }
                return lairX + lairTextX
            }, y + lairTextY, font.FontJustifyLeft, getAlpha(), lines)
        },
    })

    if !asks {
        return elements
    }

    button := func(pictures []*ebiten.Image, x int, then func()) *UIElement {
        pressed := false
        at := image.Pt(lairX + x, bottomY + lairButtonsDown)
        return &UIElement{
            Rect: util.ImageRect(at.X, at.Y, pictures[0]),
            Layer: layer,
            Order: 2,
            PlaySoundLeftClick: true,
            LeftClick: func(this *UIElement){
                pressed = true
            },
            LeftClickRelease: func(this *UIElement){
                pressed = false
                answer(then)
            },
            Draw: func(this *UIElement, window *ebiten.Image){
                var options ebiten.DrawImageOptions
                options.GeoM.Translate(float64(at.X), float64(at.Y))
                options.ColorScale.ScaleAlpha(getAlpha())
                index := 0
                if pressed {
                    index = 1
                }
                scale.DrawScaled(window, pictures[index], &options)
            },
        }
    }

    elements = append(elements, button(yesButtons, lairYesX, yes), button(noButtons, lairNoX, no))
    return elements
}
