package combat

// The box in the upper right corner that tells about the unit under the mouse.
//
// THE LOOK is the original's: the box, its rims, the font, its colors and shadow, the pictures
// and the bar of the hits. The facts were read from the ReMoM project's reconstruction
// (MoM/src/Combat.c: Draw_Combat_Unit_Display, Draw_Active_Unit_Damage_Bar; MoM/src/MainScr.c:
// Print_Moves_String) and checked against the game's palette. The code is ours.
//
// THE ARRANGEMENT is not the original's (user, 2026-09-29): everything ends at the right rim.
// The name ends there, and under it is ONE column of numbers with their pictures: attack,
// defense, resistance, moves. The original had attack, ranged attack and moves in a column on
// the left and defense, resistance, ammunition and level in one on the right, the name in the
// middle. The bar of the hits is under the name and two pixels tall, the word "Hits" is gone,
// and the box is as far from the right edge of the WINDOW as from its top, also in widescreen,
// where it is drawn with the field for that (the original: one pixel from the edge).

import (
    "fmt"
    "image/color"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    "github.com/kazzmir/master-of-magic/lib/font"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/vector"
)

// false: the box of upstream
const OriginalUnitInfo = true

// the box, in pixels of the original's screen. the original's is 69 wide and 40 tall, 4 from
// the top. taller here by the row of the bar
const (
    unitInfoTop = 4
    unitInfoWidth = 69
    unitInfoHeight = 47
    // pixels between the box and the right edge of the window
    unitInfoGutter = 4
)

// palette indexes
const (
    // the rims: left and top, right and bottom
    unitInfoRimLight = 230
    unitInfoRimDark = 237
    unitInfoTextColor = 177
    unitInfoTextEdge = 182
    unitInfoTextShadow = 2
    // the line under the bar of the hits
    unitInfoBarLine = 1
)

const unitInfoFontIndex = 0

// true: the text stands in the panel of the original, dark with rims. false: on the field with
// nothing behind it (user, 2026-09-29: "remove the box panel since it's no longer necessary")
const unitInfoPanel = false

// how dark the box makes what is behind it, 0 to 1. the original maps the colors behind it
// through a table of its data (remap block 8), which was not read: this is by eye
const unitInfoDark = 0.5
// the track of the bar is what is behind it, darker by this (the original: by half)
const unitInfoTrackDark = 0.5

// the column of numbers at the right rim: a number ends on the first, its picture starts on
// the second. from the left corner of the box
const (
    unitInfoValueX = 55
    unitInfoIconX = 57
    // the name ends here
    unitInfoNameX = 65
    unitInfoNameY = 2
    // the first row of numbers and of their pictures, and how far the rows are apart
    unitInfoValueY = 17
    unitInfoIconY = 15
    unitInfoRowStep = 7

    // the bar of the hits, under the name. it ends where the name ends
    unitInfoBarX = unitInfoNameX - hudHealthBarLength + 1
    unitInfoBarY = 10
    // the original's is 1 tall
    unitInfoBarHeight = 2
)

// the rows of the column
const (
    unitInfoRowAttack = iota
    unitInfoRowDefense
    unitInfoRowResistance
    unitInfoRowMoves
)

// what is left of the column
const (
    // the ranged attack, in the row of the attack
    unitInfoRangedValueX = 32
    unitInfoRangedIconX = 34
    // ammunition or mana, in the row under it, ends here
    unitInfoAmmoX = 42
    unitInfoAmmoRow = 1
    // the pictures of the level
    unitInfoLevelX = 3
    unitInfoLevelY = 39
    unitInfoLevelStep = 5
)

// pictures of compix.lbx
const (
    unitInfoMeleePicture = 61
    unitInfoMagicPicture = 62
    unitInfoArrowPicture = 66
    unitInfoBoulderPicture = 67
    unitInfoDefensePicture = 70
    unitInfoWalkPicture = 72
    unitInfoFlyPicture = 73
    unitInfoSwimPicture = 74
    unitInfoResistancePicture = 75
)

// the pictures of the levels, by the badge of the unit
var unitInfoLevelPictures = map[units.Badge]int{
    units.BadgeSilver: 76,
    units.BadgeGold: 77,
    units.BadgeRed: 78,
}

func makeUnitInfoFont(lbxFonts []*font.LbxFont, palette color.Palette) *font.StyledFont {
    if len(lbxFonts) <= unitInfoFontIndex || len(palette) < 256 {
        return nil
    }

    // pixel value 0 is the soft edge of the letters, 1 is the letters
    return font.MakeStyledFont(lbxFonts[unitInfoFontIndex], []color.Color{palette[unitInfoTextEdge], palette[unitInfoTextColor]}, font.ShadowDown, palette[unitInfoTextShadow])
}

// the moves as the original writes them: halves as ".5", no 0 before them
func movesText(moves float64) string {
    halves := int(moves * 2 + 0.5)
    if halves <= 0 {
        return "0"
    }

    text := ""
    if halves / 2 > 0 {
        text = fmt.Sprintf("%v", halves / 2)
    }
    if halves % 2 != 0 {
        text += ".5"
    }

    return text
}

// how long the bar of the hits of a unit is and its color. false when there is no bar
func (hud *HudFonts) healthBar(unit *ArmyUnit) (int, color.Color, bool) {
    maxHealth := unit.GetMaxHealth()
    health := unit.GetHealth()
    if maxHealth <= 0 || health <= 0 {
        return 0, nil, false
    }

    length := health * hudHealthBarLength / maxHealth - 1
    barColor := hud.Palette[hudBarRed]
    if length >= hudHealthGreenLength {
        barColor = hud.Palette[hudBarGreen]
    } else if length >= hudHealthYellowLength {
        barColor = hud.Palette[hudBarYellow]
    }

    // the line runs from its first pixel to its last, so it is one longer than the length
    return length + 1, barColor, true
}

// black that lets this much through of what is behind it: 0 all, 1 nothing
func unitInfoBlack(dark float64) color.NRGBA {
    return color.NRGBA{A: uint8(dark * 255)}
}

// a rectangle of art pixels
func fillUnitInfo(screen *ebiten.Image, x int, y int, width int, height int, fill color.Color) {
    vector.FillRect(screen, float32(scale.Scale(x)), float32(scale.Scale(y)), float32(scale.Scale(width)), float32(scale.Scale(height)), fill, false)
}

// true while the box is drawn with the field, on the picture as wide as the window
func (combat *CombatScreen) unitInfoOnField() bool {
    return OriginalUnitInfo && combat.wideField && display.BarWidth() > 0 && combat.Fonts.Hud != nil && combat.Fonts.Hud.UnitInfo != nil
}

// draws the box. false when it can not, then upstream's is drawn
func (combat *CombatScreen) showUnitInfoOriginal(screen *ebiten.Image, unit *ArmyUnit) bool {
    hud := combat.Fonts.Hud
    if !OriginalUnitInfo || hud == nil || hud.UnitInfo == nil {
        return false
    }

    onField := combat.unitInfoOnField()
    if onField != (combat.pass == drawPassField) {
        // drawn by the other pass
        return true
    }

    text := hud.UnitInfo
    width := unitInfoWidth
    height := unitInfoHeight
    // the right edge of the picture that is drawn on
    edge := data.ScreenWidth
    if onField {
        edge += 2 * display.BarWidth()
    }
    left := edge - unitInfoGutter - width
    right := left + width - 1
    top := unitInfoTop
    bottom := top + height - 1

    if unitInfoPanel {
        fillUnitInfo(screen, left, top, width, height, unitInfoBlack(unitInfoDark))
        // left and top, then right and bottom, as the original draws its lines
        fillUnitInfo(screen, left, top, 1, height, hud.Palette[unitInfoRimLight])
        fillUnitInfo(screen, left, top, width - 1, 1, hud.Palette[unitInfoRimLight])
        fillUnitInfo(screen, right, top, 1, height, hud.Palette[unitInfoRimDark])
        fillUnitInfo(screen, left + 1, bottom, width - 1, 1, hud.Palette[unitInfoRimDark])
    }

    picture := func(index int, x int, y int) {
        image, err := combat.ImageCache.GetImage("compix.lbx", index, 0)
        if err != nil {
            return
        }
        var options ebiten.DrawImageOptions
        options.GeoM.Translate(float64(left + x), float64(top + y))
        scale.DrawScaled(screen, image, &options)
    }

    // a number of the column with its picture
    row := func(row int, index int, value string) {
        text.Print(screen, left + unitInfoValueX, top + unitInfoValueY + row * unitInfoRowStep, hudOptionsRight(), value)
        picture(index, unitInfoIconX, unitInfoIconY + row * unitInfoRowStep)
    }

    text.Print(screen, left + unitInfoNameX, top + unitInfoNameY, hudOptionsRight(), unit.Unit.GetName())

    row(unitInfoRowAttack, unitInfoMeleePicture, fmt.Sprintf("%v", unit.GetMeleeAttackPower()))
    row(unitInfoRowDefense, unitInfoDefensePicture, fmt.Sprintf("%v", unit.GetDefense()))
    row(unitInfoRowResistance, unitInfoResistancePicture, fmt.Sprintf("%v", unit.GetResistance()))

    movesPicture := unitInfoWalkPicture
    if unit.IsFlying() {
        movesPicture = unitInfoFlyPicture
    } else if unit.IsSwimmer() {
        movesPicture = unitInfoSwimPicture
    }
    row(unitInfoRowMoves, movesPicture, movesText(unit.MovesLeft.ToFloat()))

    rangedPicture := -1
    switch unit.Unit.GetRangedAttackDamageType() {
        case units.DamageRangedMagical: rangedPicture = unitInfoMagicPicture
        case units.DamageRangedBoulder: rangedPicture = unitInfoBoulderPicture
        case units.DamageRangedPhysical: rangedPicture = unitInfoArrowPicture
    }
    if rangedPicture != -1 {
        text.Print(screen, left + unitInfoRangedValueX, top + unitInfoValueY, hudOptionsRight(), fmt.Sprintf("%v", unit.GetRangedAttackPower()))
        picture(rangedPicture, unitInfoRangedIconX, unitInfoIconY)
    }

    // mana before ammunition, as the original
    ammoY := top + unitInfoValueY + unitInfoAmmoRow * unitInfoRowStep
    if unit.GetCastingSkill() > 0 {
        text.Print(screen, left + unitInfoAmmoX, ammoY, hudOptionsRight(), fmt.Sprintf("%v mp", int(unit.GetCastingSkill())))
    } else if unit.GetRangedAttacks() > 0 {
        text.Print(screen, left + unitInfoAmmoX, ammoY, hudOptionsRight(), fmt.Sprintf("%v ammo", unit.GetRangedAttacks()))
    }

    badge := units.GetExperienceBadge(unit)
    levelPicture, hasLevel := unitInfoLevelPictures[badge.Badge]
    if hasLevel {
        for index := range badge.Count {
            picture(levelPicture, unitInfoLevelX + index * unitInfoLevelStep, unitInfoLevelY)
        }
    }

    // the track, the line under it as the shadow of the bar, the bar
    fillUnitInfo(screen, left + unitInfoBarX, top + unitInfoBarY, hudHealthBarLength, unitInfoBarHeight, unitInfoBlack(unitInfoTrackDark))
    fillUnitInfo(screen, left + unitInfoBarX, top + unitInfoBarY + unitInfoBarHeight, hudHealthBarLength, 1, hud.Palette[unitInfoBarLine])
    length, barColor, ok := hud.healthBar(unit)
    if ok {
        fillUnitInfo(screen, left + unitInfoBarX, top + unitInfoBarY, length, unitInfoBarHeight, barColor)
    }

    return true
}
