package game

// The panel of the surveyor (Info, Surveyor), matched to the original game.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/Surveyor.c:
// Surveyor_Screen_Draw, Surveyor_Window_Display and what they call). The code is ours.
//
// BEFORE THIS the panel was the fork's with changes of the user of 2026-09-27: all text under the
// title at the Compact size, the messages in the middle of the panel, the table of a city's
// resources laid out by rules of our own (panel.go). Those were answers to the fork's text, which
// was too wide for the panel and had no fixed places. The original has a place for every line,
// at full size, and that is what is drawn now (user, 2026-09-29: "at the minimum the text needs
// to be updated to match"). OriginalSurveyor false brings the panel of before back.
//
//   pictures of main.lbx: 57 at 240,76, 47 at 240,173, the button Cancel at 263,181
//   the title "Surveyor": font 4 in its color set 0, its middle at 280, y 80
//   names (yellow): font 1, letters 161 with soft edge 8
//   what follows a name (white): font 1 in its color set 0
//   all text with a shadow below and right in 148, spaces 1 wide
//   the land: its name at y 91, what it gives at 98 and 105, all with their middle at 280
//   what is on the land, ONE of: a city, a mineral or game or nightshade, a lair, a node.
//     its name at y 114, what it does or who guards it from y 121, 7 apart
//     a city: "<size> of" at 114 and its name at 121, both yellow
//   a place for a city: "City Resources" with its middle at 280, y 143; "Maximum Pop",
//     "Prod Bonus", "Gold Bonus" from x 245 at y 151, 158, 165, the number of the first ending
//     on 306, the two others ("+N%") on 312
//   no place for a city: "Cities cannot be built " and why, yellow, from 246,143 in lines of 67

import (
    "fmt"
    "image"
    "strings"

    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    fontslib "github.com/kazzmir/master-of-magic/game/magic/fonts"
    "github.com/kazzmir/master-of-magic/game/magic/maplib"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/lib/font"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: the panel of before, see the top of the file
var OriginalSurveyor = true

const surveyorLbx = "main.lbx"
const surveyorIndex = 0

const (
    surveyorLandY = 76
    surveyorCancelBackY = 173
    surveyorCancelY = 181

    surveyorMiddle = 280
    surveyorTitleY = 80
    surveyorTerrainY = 91
    surveyorSpecialY = 114
    surveyorRowStep = 7
    // what a mineral does is a paragraph this wide
    surveyorEffectWidth = 66

    surveyorMessageX = 246
    surveyorMessageY = 143
    surveyorMessageWidth = 67

    surveyorTableX = 245
    surveyorTableY = 151
    surveyorPopulationRight = 306
    surveyorBonusRight = 312
)

const (
    surveyorYellow = 161
    surveyorYellowEdge = 8
    surveyorShadow = 148
)

type surveyorStyle struct {
    Title *font.StyledFont
    Yellow *font.StyledFont
    White *font.StyledFont
}

func (game *Game) getSurveyorStyle() *surveyorStyle {
    if !OriginalSurveyor {
        return nil
    }

    original := fontslib.LoadOriginal(game.Cache, surveyorLbx, surveyorIndex)
    if original == nil {
        return nil
    }

    style := &surveyorStyle{
        Title: original.Style(4, original.Set(0, -1), font.ShadowDown, surveyorShadow),
        Yellow: original.Style(1, original.Colors(surveyorYellowEdge, surveyorYellow, surveyorYellow), font.ShadowDown, surveyorShadow),
        White: original.Style(1, original.Set(0, -1), font.ShadowDown, surveyorShadow),
    }
    if style.Title == nil || style.Yellow == nil || style.White == nil {
        return nil
    }
    style.Yellow.SpaceWidth = 1
    style.White.SpaceWidth = 1

    return style
}

// the names of the original for what the fork names otherwise
var surveyorBonusNames = map[data.BonusType]string{
    data.BonusGem: "Gems",
    data.BonusQuorkCrystal: "Quork Crystals",
    data.BonusCrysxCrystal: "Crysx Crystals",
}

var surveyorLairNames = map[maplib.EncounterType]string{
    maplib.EncounterTypeLair: "Lair",
    maplib.EncounterTypeCave: "Cave",
    maplib.EncounterTypePlaneTower: "Tower",
    maplib.EncounterTypeAncientTemple: "Temple",
    maplib.EncounterTypeFallenTemple: "Temple",
    maplib.EncounterTypeRuins: "Ruins",
    maplib.EncounterTypeAbandonedKeep: "Keep",
    maplib.EncounterTypeDungeon: "Dungeon",
}

func isNodeEncounter(kind maplib.EncounterType) bool {
    return kind == maplib.EncounterTypeChaosNode || kind == maplib.EncounterTypeNatureNode || kind == maplib.EncounterTypeSorceryNode
}

// why no city can be built on a tile, "" if one can. the order is the original's
func (game *Game) surveyorNoCity(mapObject *maplib.Map, x int, y int, city *citylib.City) string {
    tile := mapObject.GetTile(x, y)
    encounter := mapObject.GetEncounter(x, y)

    switch {
        case city != nil: return ""
        case !tile.Tile.IsLand(): return "on water."
        case encounter != nil && encounter.Type == maplib.EncounterTypePlaneTower: return "on towers."
        case encounter != nil && isNodeEncounter(encounter.Type): return "on magic nodes."
        case mapObject.GetMagicNode(x, y) != nil: return "on magic nodes."
        case encounter != nil: return "on lairs."
        case game.Model.NearCity(image.Pt(x, y), 3, game.Model.Plane): return "less than 3 squares from any other city."
    }

    return ""
}

// the two guardians of a lair as the player knows them
func surveyorGuardians(encounter *maplib.ExtraEncounter, player *playerlib.Player) []string {
    if !encounter.ExploredBy.Contains(player) {
        return []string{"Unexplored"}
    }

    var names []string
    for _, unit := range encounter.Units {
        known := false
        for _, name := range names {
            if name == unit.Name {
                known = true
            }
        }
        if !known && len(names) < 2 {
            names = append(names, unit.Name)
        }
    }

    if len(names) == 0 {
        // not the original's, which has no lair without a guardian
        return []string{"Empty"}
    }
    return names
}

type surveyorResources struct {
    MaximumPopulation int
    ProductionBonus int
    GoldBonus int
}

// the text of the panel for a tile the player has explored
func (game *Game) drawSurveyorOriginal(screen *ebiten.Image, style *surveyorStyle, x int, y int, city *citylib.City, resources surveyorResources) {
    mapObject := game.Model.CurrentMap()
    player := game.Model.GetHumanPlayer()
    tile := mapObject.GetTile(x, y)
    node := mapObject.GetMagicNode(x, y)
    encounter := mapObject.GetEncounter(x, y)

    left := font.FontOptions{Scale: scale.ScaleAmount}
    right := font.FontOptions{Scale: scale.ScaleAmount, Justify: font.FontJustifyRight}
    center := font.FontOptions{Scale: scale.ScaleAmount, Justify: font.FontJustifyCenter}

    // the land
    name := tile.Name(mapObject)
    if node != nil {
        switch node.Kind {
            case maplib.MagicNodeNature: name = "Forest"
            case maplib.MagicNodeSorcery: name = "Grasslands"
            case maplib.MagicNodeChaos: name = "Mountain"
        }
    }
    style.Yellow.Print(screen, surveyorMiddle, surveyorTerrainY, center, name)

    var gives []string
    if tile.Corrupted() {
        gives = append(gives, "Corruption")
    } else {
        food := tile.FoodBonus()
        if !food.IsZero() {
            // spaces are one pixel wide here: the original has three between the 1 and the
            // half of "1   1/2 food", or they read as 11/2
            gives = append(gives, fmt.Sprintf("%v food", strings.Replace(food.NormalString(), " ", "   ", 1)))
        }
        production := tile.ProductionBonus(false)
        if production != 0 {
            gives = append(gives, fmt.Sprintf("+%v%% production", production))
        }
        gold := tile.GoldBonus(mapObject)
        if gold != 0 {
            gives = append(gives, fmt.Sprintf("+%v%% gold", gold))
        }
    }
    for row, text := range gives {
        style.White.Print(screen, surveyorMiddle, surveyorTerrainY + (row + 1) * surveyorRowStep, center, text)
    }

    // what is on it: one thing
    bonus := tile.GetBonus()
    switch {
        case city != nil:
            style.Yellow.Print(screen, surveyorMiddle, surveyorSpecialY, center, fmt.Sprintf("%v of", city.GetSize()))
            style.Yellow.Print(screen, surveyorMiddle, surveyorSpecialY + surveyorRowStep, center, city.Name)
        case tile.Corrupted():
        case bonus != data.BonusNone:
            bonusName, ok := surveyorBonusNames[bonus]
            if !ok {
                bonusName = bonus.String()
            }
            style.Yellow.Print(screen, surveyorMiddle, surveyorSpecialY, center, bonusName)

            var does []string
            if bonus.FoodBonus() != 0 {
                does = append(does, fmt.Sprintf("+%v food", bonus.FoodBonus()))
            }
            if bonus.GoldBonus() != 0 {
                does = append(does, fmt.Sprintf("+%v gold", bonus.GoldBonus()))
            }
            if bonus.PowerBonus() != 0 {
                does = append(does, fmt.Sprintf("+%v power", bonus.PowerBonus()))
            }
            if bonus.UnitReductionBonus() != 0 {
                does = append(does, fmt.Sprintf("Reduces normal unit cost by %v%%", bonus.UnitReductionBonus()))
            }
            if bonus == data.BonusNightshade {
                does = append(does, "Protects city from spells")
            }

            row := 0
            for _, text := range does {
                for _, line := range style.White.Wrap(text, surveyorEffectWidth, 1) {
                    style.White.Print(screen, surveyorMiddle, surveyorSpecialY + surveyorRowStep + row * style.White.LineHeight(), center, line)
                    row += 1
                }
            }
        case encounter != nil && !isNodeEncounter(encounter.Type):
            lairName, ok := surveyorLairNames[encounter.Type]
            if !ok {
                lairName = encounter.Type.Name()
            }
            style.Yellow.Print(screen, surveyorMiddle, surveyorSpecialY, center, lairName)
            for row, guardian := range surveyorGuardians(encounter, player) {
                style.White.Print(screen, surveyorMiddle, surveyorSpecialY + (row + 1) * surveyorRowStep, center, guardian)
            }
        case node != nil:
            style.Yellow.Print(screen, surveyorMiddle, surveyorSpecialY, center, node.Kind.Name())

            var state []string
            switch {
                case node.Warped: state = []string{"Warped"}
                case node.MeldingWizard != nil && node.GuardianSpiritMeld: state = []string{"Guardian Spirit"}
                case node.MeldingWizard != nil: state = []string{"Magic Spirit"}
                case encounter != nil: state = surveyorGuardians(encounter, player)
            }
            for row, text := range state {
                style.White.Print(screen, surveyorMiddle, surveyorSpecialY + (row + 1) * surveyorRowStep, center, text)
            }
    }

    // a place for a city or not
    why := game.surveyorNoCity(mapObject, x, y, city)
    if why != "" {
        lines := style.Yellow.Wrap("Cities cannot be built " + why, surveyorMessageWidth, 1)
        for row, line := range lines {
            style.Yellow.Print(screen, surveyorMessageX, surveyorMessageY + row * (style.Yellow.Height() + 1), left, line)
        }
        return
    }

    style.Yellow.Print(screen, surveyorMiddle, surveyorMessageY, center, "City Resources")
    rows := []struct{
        Label string
        Value string
        Right int
    }{
        {"Maximum Pop", fmt.Sprintf("%v", resources.MaximumPopulation), surveyorPopulationRight},
        {"Prod Bonus", fmt.Sprintf("+%v%%", resources.ProductionBonus), surveyorBonusRight},
        {"Gold Bonus", fmt.Sprintf("+%v%%", resources.GoldBonus), surveyorBonusRight},
    }
    for row, entry := range rows {
        rowY := surveyorTableY + row * surveyorRowStep
        style.White.Print(screen, surveyorTableX, rowY, left, entry.Label)
        style.White.Print(screen, entry.Right, rowY, right, entry.Value)
    }
}
