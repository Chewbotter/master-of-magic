package game

// The historian (Info, Historian), matched to the original game.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/AdvsrScr.c:
// Powergraph_Screen, Powergraph_Screen_Draw, IDK_Powergraph_Draw_Turn_Data). The code is ours.
// Upstream had fonts, colors and places of its own, a drop shadow of less than an art pixel, a
// line of time from 10 to 310 with 23 marks and 10 years spread evenly whatever the turn, and
// the turns pressed into that width.
//
//   the picture: reload.lbx 0, whose palette has the colors from 224 on
//   title: font 4, letters 178 to 182 with soft edge 246, its middle at 160, y 10
//   date: font 2 in the same colors, ending on 309, y 11
//   names: font 3, all of a letter in the color of the banner, from 10,23, 9 apart
//   years: font 1 in its color set 0 with soft edge 246, y 186
//   all text with a shadow below and right in 254
//   the line of time: y 183 from 16 to 303 in 180, the row under it darker; a mark from y 182
//     to 184 every 12 pixels from 16, which is a year: a turn is ONE pixel. the first year
//     stands at 10, and every year that can be divided by 5 at its mark less 6
//   the lines of the wizards: from x 16, a pixel a turn, up from y 181 and never above y 22, in
//     the palette color of the banner; the line of the player last, over the others
//
// OURS, not the original's: how high a line is. The original draws the number of its history as
// it is, which is one the fork does not keep; here the power is scaled so that the highest of
// all wizards, or historianPowerLeast, reaches the top.

import (
    "fmt"
    "image/color"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    fontslib "github.com/kazzmir/master-of-magic/game/magic/fonts"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/lib/font"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/vector"
)

// false: the historian of upstream
var OriginalHistorian = true

const historianLbx = "reload.lbx"
const historianIndex = 0

const (
    historianTitleX = 160
    historianTitleY = 10
    historianDateX = 309
    historianDateY = 11
    historianNameX = 10
    historianNameY = 23
    historianNameStep = 9

    historianLeft = 16
    historianRight = 303
    historianLineY = 183
    historianMarkTop = 182
    historianMarkBottom = 184
    historianYearStep = 12
    historianYearY = 186
    historianFirstYearX = 10
    // a year stands this far left of its mark
    historianYearLeft = 6
    historianYearsEvery = 5

    historianGraphBottom = 181
    historianGraphTop = 22
    // years the line of time has room for before its start moves on
    historianYearsShown = 23
    historianFirstYear = 1400
)

// the power that reaches the top of the graph when no wizard has more
const historianPowerLeast = 1000

const (
    historianEdge = 246
    historianShadow = 254
    historianLineColor = 180
)
var historianTitleColors = []int{historianEdge, 178, 179, 180, 181, 182}

// palette indexes of the banners in the graphs of the advisors
var advisorBannerColors = map[data.BannerType]int{
    data.BannerBlue: 172,
    data.BannerGreen: 216,
    data.BannerPurple: 124,
    data.BannerRed: 201,
    data.BannerYellow: 211,
}

// a rectangle of art pixels on a picture that has one pixel an art pixel
func fillArt(picture *ebiten.Image, x int, y int, width int, height int, fill color.Color) {
    vector.FillRect(picture, float32(x), float32(y), float32(width), float32(height), fill, false)
}

// draws the historian on its picture, which has the background. false when it can not
func (game *Game) drawHistorianOriginal(picture *ebiten.Image, players []*playerlib.Player) bool {
    if !OriginalHistorian {
        return false
    }

    original := fontslib.LoadOriginal(game.Cache, historianLbx, historianIndex)
    if original == nil {
        return false
    }

    title := original.Style(4, original.Colors(historianTitleColors...), font.ShadowDown, historianShadow)
    date := original.Style(2, original.Colors(historianTitleColors...), font.ShadowDown, historianShadow)
    year := original.Style(1, original.Set(0, historianEdge), font.ShadowDown, historianShadow)
    if title == nil || date == nil || year == nil {
        return false
    }

    left := font.FontOptions{Scale: 1}
    right := font.FontOptions{Scale: 1, Justify: font.FontJustifyRight}
    center := font.FontOptions{Scale: 1, Justify: font.FontJustifyCenter}

    title.Print(picture, historianTitleX, historianTitleY, center, "History Of Wizards Power")
    date.Print(picture, historianDateX, historianDateY, right, game.TurnDate())

    // the line of time
    lineColor := original.Palette[historianLineColor]
    fillArt(picture, historianLeft, historianLineY, historianRight - historianLeft + 1, 1, lineColor)
    fillArt(picture, historianLeft, historianLineY + 1, historianRight - historianLeft + 1, 1, color.NRGBA{A: 128})

    years := int(game.Model.TurnNumber / 12)
    firstYear := max(0, years - historianYearsShown)
    firstTurn := firstYear * 12

    year.Print(picture, historianFirstYearX, historianYearY, left, fmt.Sprintf("%v", historianFirstYear + firstYear))
    for mark := 0; historianLeft + mark * historianYearStep <= historianRight; mark++ {
        x := historianLeft + mark * historianYearStep
        fillArt(picture, x, historianMarkTop, 1, historianMarkBottom - historianMarkTop + 1, lineColor)

        if mark > 0 && (firstYear + mark) % historianYearsEvery == 0 {
            year.Print(picture, x - historianYearLeft, historianYearY, left, fmt.Sprintf("%v", historianFirstYear + firstYear + mark))
        }
    }

    // how high the lines are is ours, see the top of the file
    most := historianPowerLeast
    for _, player := range players {
        for _, power := range player.PowerHistory {
            most = max(most, power.TotalPower())
        }
    }

    human := game.Model.GetHumanPlayer()
    // the player first in the names, and last in the lines
    var names []*playerlib.Player
    var lines []*playerlib.Player
    if human != nil {
        names = append(names, human)
    }
    for _, player := range players {
        if player != human {
            names = append(names, player)
            lines = append(lines, player)
        }
    }
    if human != nil {
        lines = append(lines, human)
    }

    bannerColor := func(player *playerlib.Player) color.Color {
        index, ok := advisorBannerColors[player.GetBanner()]
        if !ok {
            return player.GetBanner().Color()
        }
        return original.Palette[index]
    }

    for _, player := range lines {
        use := bannerColor(player)
        height := historianGraphBottom - historianGraphTop

        lastY := 0
        for step := 0; historianLeft + step <= historianRight; step++ {
            turn := uint64(firstTurn + step)
            if turn >= game.Model.TurnNumber {
                break
            }

            history, ok := player.GetPowerHistoryForTurn(turn)
            if !ok {
                break
            }

            power := max(0, history.TotalPower())
            y := max(historianGraphTop, historianGraphBottom - power * height / most)

            x := historianLeft + step
            if step == 0 {
                fillArt(picture, x, y, 1, 1, use)
            } else {
                // from the point of the turn before to this one
                top := min(lastY, y)
                fillArt(picture, x, top, 1, max(lastY, y) - top + 1, use)
            }
            lastY = y
        }
    }

    for row, player := range names {
        use := bannerColor(player)
        var colors []color.Color
        for range font.ColorBlockSize {
            colors = append(colors, use)
        }
        name := font.MakeStyledFont(original.Fonts[3], colors, font.ShadowDown, original.Palette[historianShadow])
        name.Print(picture, historianNameX, historianNameY + row * historianNameStep, left, player.Wizard.Name)
    }

    return true
}
