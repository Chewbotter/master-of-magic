package game

// The box that tells a hero has made a level, matched to the original game.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/LVLMAKE.c:
// Hero_LevelUp_Popup, Hero_LevelUp_Popup_Draw). The code is ours.
//
//   the box (reload 23, its lower end reload 24) at 31,41 (upstream 30,40), cut off under its
//     last row; the portrait 10,10 in it
//   "<name> has made a level." font 4, letters 178 179 180 180 with soft edge 240, at 48,9
//   what the hero gains: font 2, letters all 178 with soft edge 240, at 55 and 147, y 24 and 34,
//     the bullet 8 left and a pixel lower
//   the level and the abilities that grow with it: pictures at 13 and 128, rows 17 apart from 48,
//     the name 19 right and 5 down of its picture, in the text of what the hero gains
//   all text with a shadow below and right in 240 of the palette of the box
//
// OURS: the height of the box, 49 and 17 for every row (ReMoM has 49 and the number of rows,
// which cuts the box off over its pictures).

import (
    "fmt"
    "image"

    "github.com/kazzmir/master-of-magic/lib/font"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    fontslib "github.com/kazzmir/master-of-magic/game/magic/fonts"
    herolib "github.com/kazzmir/master-of-magic/game/magic/hero"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: the box of upstream
var OriginalLevelUp = true

const (
    levelUpX = 31
    levelUpY = 41
    levelUpShadow = 240
    levelUpTitleX = 48
    levelUpTitleY = 9
    levelUpGainX = 55
    levelUpGainY = 24
    levelUpGainColumn = 92
    levelUpGainRow = 10
    levelUpBulletLeft = 8
    levelUpListX = 13
    levelUpListY = 48
    levelUpListColumn = 115
    levelUpListRow = 17
    levelUpListMost = 8
    levelUpNameX = 19
    levelUpNameY = 5
    levelUpHeight = 49
)

var levelUpTitleColors = []int{240, 178, 179, 180, 180}
var levelUpTextColors = []int{240, 178, 178, 178, 178}

// draws the box. false when it can not be made and nothing was drawn
func (game *Game) drawLevelUpOriginal(screen *ebiten.Image, hero *herolib.Hero, abilities []data.Ability, alpha float32) bool {
    if !OriginalLevelUp {
        return false
    }

    original := fontslib.LoadOriginal(game.Cache, "reload.lbx", 23)
    if original == nil {
        return false
    }

    title := original.Style(4, original.Colors(levelUpTitleColors...), font.ShadowDown, levelUpShadow)
    text := original.Style(2, original.Colors(levelUpTextColors...), font.ShadowDown, levelUpShadow)
    top, err1 := game.ImageCache.GetImage("reload.lbx", 23, 0)
    bottom, err2 := game.ImageCache.GetImage("reload.lbx", 24, 0)
    bullet, err3 := game.ImageCache.GetImage("itemisc.lbx", 26, 0)
    if title == nil || text == nil || err1 != nil || err2 != nil || err3 != nil {
        return false
    }

    // the level is the first of the list
    listed := min(len(abilities) + 1, levelUpListMost)
    height := min(levelUpHeight + (listed + 1) / 2 * levelUpListRow, top.Bounds().Dy())

    var options ebiten.DrawImageOptions
    options.ColorScale.ScaleAlpha(alpha)
    print := font.FontOptions{Options: &options, Scale: scale.ScaleAmount}

    draw := func(picture *ebiten.Image, x int, y int) {
        var at ebiten.DrawImageOptions
        at.ColorScale.ScaleAlpha(alpha)
        at.GeoM.Translate(float64(levelUpX + x), float64(levelUpY + y))
        scale.DrawScaled(screen, picture, &at)
    }

    draw(top.SubImage(image.Rect(0, 0, top.Bounds().Dx(), height).Add(top.Bounds().Min)).(*ebiten.Image), 0, 0)
    draw(bottom, 0, height)

    portraitLbx, portraitIndex := hero.GetPortraitLbxInfo()
    portrait, err := game.ImageCache.GetImage(portraitLbx, portraitIndex, 0)
    if err == nil {
        draw(portrait, 10, 10)
    }

    title.Print(screen, levelUpX + levelUpTitleX, levelUpY + levelUpTitleY, print, fmt.Sprintf("%v has made a level.", hero.Name))

    for index, gain := range hero.GetBaseProgression() {
        x := levelUpGainX + levelUpGainColumn * (index / 2)
        y := levelUpGainY + levelUpGainRow * (index % 2)
        draw(bullet, x - levelUpBulletLeft, y + 1)
        text.Print(screen, levelUpX + x, levelUpY + y, print, gain)
    }

    place := func(index int) (int, int) {
        return levelUpListX + levelUpListColumn * (index % 2), levelUpListY + levelUpListRow * (index / 2)
    }

    experience := hero.GetExperienceData()
    badge, err := game.ImageCache.GetImage("special.lbx", 102 + experience.ToInt(), 0)
    if err == nil {
        x, y := place(0)
        draw(badge, x, y)
        text.Print(screen, levelUpX + x + levelUpNameX, levelUpY + y + levelUpNameY, print, experience.Name())
    }

    for index, ability := range abilities {
        if index + 1 >= levelUpListMost {
            break
        }

        picture, err := game.ImageCache.GetImage(ability.LbxFile(), ability.LbxIndex(), 0)
        if err != nil {
            continue
        }
        x, y := place(index + 1)
        draw(picture, x, y)
        text.Print(screen, levelUpX + x + levelUpNameX, levelUpY + y + levelUpNameY, print, ability.Name())
    }

    return true
}
