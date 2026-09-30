package game

// The astrologer (Info, Astrologer), matched to the original game.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/AdvsrScr.c: Status_Screen,
// Status_Screen_Draw) and the color sets of the fonts from the game's data. The code is ours.
// Upstream had fonts and colors of its own, a drop shadow of less than an art pixel, the picture
// at 10,10, rows 6 apart with gaps for wizards that are not shown, and black under the bars.
//
//   the picture: reload.lbx 1 at 20,11. from its corner:
//   title: font 4, letters 178 to 182 with soft edge 246, its middle at 138, y 10
//   headings: font 2 in its color set 3 with soft edge 246, middle at 138, y 23, 72 and 121
//   names: font 0 in the colors of the title, from x 14; the rows of a part start 12 under its
//     heading and are 7 apart
//   all text with a shadow below and right in 254
//   a bar: from x 64, two rows high from a row under the top of its name, in the palette color
//     of the banner; under it, a row lower and a pixel longer, the picture is darker
//
// OURS, not the original's: how long a bar is. The original draws its number as it is, up to
// 200; the fork keeps other numbers, so the highest of all wizards, or astrologerLeast, is
// astrologerBarLength long.

import (
    "image/color"

    fontslib "github.com/kazzmir/master-of-magic/game/magic/fonts"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/lib/font"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: the astrologer of upstream
var OriginalAstrologer = true

const astrologerLbx = "reload.lbx"
const astrologerIndex = 1

const (
    astrologerX = 20
    astrologerY = 11

    astrologerMiddle = 138
    astrologerTitleY = 10
    astrologerNameX = 14
    astrologerBarX = 64
    astrologerRowStep = 7
    // the rows of a part start this far under its heading
    astrologerRowsDown = 12
)

var astrologerHeadings = []struct{
    Text string
    Y int
}{
    {"Army Strength", 23},
    {"Magic Power", 72},
    {"Spell Research", 121},
}

const astrologerBarLength = 200
const astrologerLeast = 400

const (
    astrologerEdge = 246
    astrologerShadow = 254
    astrologerHeadingBlock = 3
)

// draws the astrologer on its picture, which has the background. false when it can not
func (game *Game) drawAstrologerOriginal(picture *ebiten.Image, players []*playerlib.Player) bool {
    if !OriginalAstrologer {
        return false
    }

    original := fontslib.LoadOriginal(game.Cache, astrologerLbx, astrologerIndex)
    if original == nil {
        return false
    }

    colors := original.Colors(astrologerEdge, 178, 179, 180, 181, 182)
    title := original.Style(4, colors, font.ShadowDown, astrologerShadow)
    heading := original.Style(2, original.Set(astrologerHeadingBlock, astrologerEdge), font.ShadowDown, astrologerShadow)
    name := original.Style(0, colors, font.ShadowDown, astrologerShadow)
    if title == nil || heading == nil || name == nil {
        return false
    }

    left := font.FontOptions{Scale: 1}
    center := font.FontOptions{Scale: 1, Justify: font.FontJustifyCenter}

    title.Print(picture, astrologerMiddle, astrologerTitleY, center, "Current Status Of Wizards")
    for _, part := range astrologerHeadings {
        heading.Print(picture, astrologerMiddle, part.Y, center, part.Text)
    }

    // the player first, and no rows for wizards that are not shown
    human := game.Model.GetHumanPlayer()
    var shown []*playerlib.Player
    if human != nil {
        shown = append(shown, human)
    }
    for _, player := range players {
        if player != human && !player.Defeated && !player.IsNeutral() {
            shown = append(shown, player)
        }
    }

    most := []int{astrologerLeast, astrologerLeast, astrologerLeast}
    values := func(player *playerlib.Player) []int {
        power := player.LatestWizardPower()
        return []int{power.Army, power.Magic, power.SpellResearch}
    }
    for _, player := range shown {
        for part, value := range values(player) {
            most[part] = max(most[part], value)
        }
    }

    for row, player := range shown {
        var bar color.Color = player.GetBanner().Color()
        index, ok := advisorBannerColors[player.GetBanner()]
        if ok {
            bar = original.Palette[index]
        }

        for part, value := range values(player) {
            y := astrologerHeadings[part].Y + astrologerRowsDown + row * astrologerRowStep
            name.Print(picture, astrologerNameX, y, left, player.Wizard.Name)

            length := max(0, value) * astrologerBarLength / most[part]
            // darker under the bar, then the bar
            fillArt(picture, astrologerBarX, y + 2, length + 2, 2, color.NRGBA{A: 128})
            fillArt(picture, astrologerBarX, y + 1, length + 1, 2, bar)
        }
    }

    return true
}
