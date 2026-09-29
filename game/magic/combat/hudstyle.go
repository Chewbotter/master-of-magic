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
    "github.com/hajimehoshi/ebiten/v2/vector"
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

    hudUnitNameX = 112
    hudUnitNameY = 167
    // the space between the words of a unit's name
    hudUnitNameSpaceWidth = 1

    hudStatValueX = 126
    hudStatIconX = 128
    hudMeleeValueY = 174
    hudMeleeIconY = 172
    hudRangedValueY = 181
    hudRangedIconY = 179
    hudMovesValueY = 188
    hudMovesIconY = 186

    // the track of the health bar is its own small picture
    hudHealthTrackX = 117
    hudHealthTrackY = 194
    hudHealthTrackIndex = 18
    hudHealthBarX = 118
    hudHealthBarY = 195
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

// the health bar of the selected unit in the combat bar: a line on its track
func (combat *CombatScreen) drawHudHealthBar(screen *ebiten.Image, unit *ArmyUnit) {
    track, err := combat.ImageCache.GetImage("compix.lbx", hudHealthTrackIndex, 0)
    if err == nil {
        var options ebiten.DrawImageOptions
        options.GeoM.Translate(hudHealthTrackX, hudHealthTrackY)
        scale.DrawScaled(screen, track, &options)
    }

    hud := combat.Fonts.Hud
    if hud == nil {
        return
    }

    length, barColor, ok := hud.healthBar(unit)
    if !ok {
        return
    }

    vector.FillRect(screen, float32(scale.Scale(hudHealthBarX)), float32(scale.Scale(hudHealthBarY)), float32(scale.Scale(length)), float32(scale.Scale(1)), barColor, false)
}
