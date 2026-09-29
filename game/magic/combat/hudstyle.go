package combat

// The combat bar at the bottom of the screen, matched to the original game.
//
// Every number in this file is the original's: palette indexes, positions, thresholds. They were
// read from the ReMoM project's reconstruction (MoM/src/Combat.c: Tactical_Combat_Draw,
// Draw_Spell_Information_Window, Draw_Active_Unit_Stats_And_Icons, Draw_Active_Unit_Damage_Bar) and
// checked against the game's own palette data. The code is ours.

import (
    "image/color"
    "log"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/lib/font"
    "github.com/kazzmir/master-of-magic/lib/lbx"

    "github.com/hajimehoshi/ebiten/v2"
)

// the picture of the combat bar, which carries the palette colors 224 and up that the bar's text uses
const hudBackgroundLbx = "backgrnd.lbx"
const hudBackgroundIndex = 3

// palette indexes
const (
    // text on the stone: dark letters with a light edge below and right, as if chiseled
    hudTextColor = 243
    hudTextEdgeColor = 227
    // the shadow under the wizard names
    hudNameShadowColor = 241
    hudBarGreen = 217
    hudBarYellow = 211
    hudBarRed = 202
)

// the four shades of a wizard's name, top to bottom, by banner
var hudNameColors = map[data.BannerType][4]int{
    data.BannerBlue: {97, 98, 99, 100},
    data.BannerGreen: {66, 67, 68, 69},
    data.BannerPurple: {33, 34, 35, 36},
    data.BannerRed: {201, 202, 203, 166},
    data.BannerYellow: {160, 161, 162, 163},
    data.BannerBrown: {28, 27, 26, 25},
}

// fonts of the original, by index
const (
    hudTextFontIndex = 0
    hudNameFontIndex = 4
)

// the combat bar starts at this row, the battlefield is above it
const hudTop = 164

// positions, in original pixels
const (
    hudNameRightX = 278
    hudNameLeftX = 40
    hudNameY = 167
    // the name on the left sits a pixel lower unless the fight is at a lair
    hudNameLeftY = 168

    hudInfoLabelX = 200
    hudInfoValueX = 236
    hudSkillY = 172
    hudManaY = 181
    hudRangeY = 190

    // THE PANEL OF THE SELECTED UNIT is arranged anew (user, 2026-09-29): the figure further
    // left, its name in the middle above it, the bar of its hits under the name as in the box of
    // the unit under the mouse (unitinfo.go), and the column of numbers with more room between
    // its rows. the original's places are behind each value

    // the inside of the panel, which the name stays in
    hudUnitPanelLeft = 83
    hudUnitPanelRight = 140

    // the picture of the figure, and its middle: of the name and of the bar too
    hudUnitFigureX = 83 // 85. 81 first: the widest figures touched the rim of the panel
    hudUnitFigureY = 170
    hudUnitMiddleX = hudUnitFigureX + 14

    hudUnitNameX = hudUnitMiddleX // 112
    hudUnitNameY = 167
    // the space between the words of a unit's name
    hudUnitNameSpaceWidth = 1

    hudStatValueX = 126
    hudStatIconX = 128
    // the rows are 8 apart, the original's 7
    hudMeleeValueY = 176 // 174
    hudMeleeIconY = 174 // 172
    hudRangedValueY = 184 // 181
    hudRangedIconY = 182 // 179
    hudMovesValueY = 192 // 188
    hudMovesIconY = 190 // 186

    // the track of the health bar is its own small picture. not drawn any more: the bar has the
    // look of the one of the unit under the mouse
    hudHealthTrackX = 117
    hudHealthTrackY = 194
    hudHealthTrackIndex = 18
    hudHealthBarX = hudUnitMiddleX - hudHealthBarLength / 2 // 118
    hudHealthBarY = 175 // 195
    // the bar is up to this many pixels long
    hudHealthBarLength = 20
    // bars at least this long are green, then yellow, shorter ones are red
    hudHealthGreenLength = 13
    hudHealthYellowLength = 6
)

type HudFonts struct {
    // labels and numbers on the stone
    Text *font.StyledFont
    // the name of the selected unit
    UnitName *font.StyledFont
    AttackerName *font.StyledFont
    DefenderName *font.StyledFont
    // the numbers that rise from a unit that is hurt, see damagenumbers.go
    Damage *font.StyledFont
    // the box of the unit under the mouse, see unitinfo.go
    UnitInfo *font.StyledFont
    Palette color.Palette
    // the fonts of the game as they are in its data, for styles that are made later
    LbxFonts []*font.LbxFont
}

func makeHudFonts(cache *lbx.LbxCache, lbxFonts []*font.LbxFont, attackerBanner data.BannerType, defenderBanner data.BannerType) *HudFonts {
    backgroundLbx, err := cache.GetLbxFile(hudBackgroundLbx)
    if err != nil {
        log.Printf("Unable to read %v: %v", hudBackgroundLbx, err)
        return nil
    }

    palette, err := backgroundLbx.GetPalette(hudBackgroundIndex)
    if err != nil || palette == nil {
        log.Printf("Unable to read the palette of the combat bar: %v", err)
        return nil
    }

    if len(lbxFonts) <= hudNameFontIndex {
        return nil
    }

    makeText := func() *font.StyledFont {
        // pixel value 0 is the soft edge of the letters, 1 is the letters
        return font.MakeStyledFont(lbxFonts[hudTextFontIndex], []color.Color{palette[hudTextEdgeColor], palette[hudTextColor]}, font.ShadowDown, palette[hudTextEdgeColor])
    }

    makeName := func(banner data.BannerType) *font.StyledFont {
        shades, ok := hudNameColors[banner]
        if !ok {
            shades = hudNameColors[data.BannerBrown]
        }
        colors := []color.Color{nil, palette[shades[0]], palette[shades[1]], palette[shades[2]], palette[shades[3]]}
        return font.MakeStyledFont(lbxFonts[hudNameFontIndex], colors, font.ShadowDown, palette[hudNameShadowColor])
    }

    unitName := makeText()
    unitName.SpaceWidth = hudUnitNameSpaceWidth

    return &HudFonts{
        Text: makeText(),
        UnitName: unitName,
        AttackerName: makeName(attackerBanner),
        DefenderName: makeName(defenderBanner),
        Damage: makeDamageFont(lbxFonts, palette),
        UnitInfo: makeUnitInfoFont(lbxFonts, palette),
        Palette: palette,
        LbxFonts: lbxFonts,
    }
}

func hudOptions() font.FontOptions {
    return font.FontOptions{Scale: scale.ScaleAmount}
}

func hudOptionsRight() font.FontOptions {
    return font.FontOptions{Scale: scale.ScaleAmount, Justify: font.FontJustifyRight}
}

func hudOptionsCenter() font.FontOptions {
    return font.FontOptions{Scale: scale.ScaleAmount, Justify: font.FontJustifyCenter}
}

// where the name of the selected unit has its middle: over its figure, and no further left or
// right than the panel has room for a name this wide
func hudUnitNameMiddle(width int) int {
    half := width / 2
    middle := max(hudUnitNameX, hudUnitPanelLeft + half)
    return min(middle, max(hudUnitPanelRight - (width - half) + 1, hudUnitPanelLeft + half))
}

// the health bar of the selected unit in the combat bar, under its name: as the one of the unit
// under the mouse, see unitinfo.go. the original's is a line of one pixel on a track of its own
// at the bottom of the panel
func (combat *CombatScreen) drawHudHealthBar(screen *ebiten.Image, unit *ArmyUnit) {
    hud := combat.Fonts.Hud
    if hud == nil {
        return
    }

    fillUnitInfo(screen, hudHealthBarX, hudHealthBarY, hudHealthBarLength, unitInfoBarHeight, unitInfoBlack(unitInfoTrackDark))
    fillUnitInfo(screen, hudHealthBarX, hudHealthBarY + unitInfoBarHeight, hudHealthBarLength, 1, hud.Palette[unitInfoBarLine])

    length, barColor, ok := hud.healthBar(unit)
    if !ok {
        return
    }

    fillUnitInfo(screen, hudHealthBarX, hudHealthBarY, length, unitInfoBarHeight, barColor)
}
