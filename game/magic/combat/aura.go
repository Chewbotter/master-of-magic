package combat

// The outline of the figures of a unit that has a spell on it, matched to the original game.
//
// The original draws a rim of ONE PIXEL around the figure, outside of it, in the five shades of
// the realm of the spell, which run around the rim. It paints it into the picture of the figure:
// every empty pixel right next to a pixel of the figure (left, right, above, below) gets a color
// of a list of 8, picked by the place of the pixel in the picture, and the list is another one
// for each of 8 stages, one stage every second redraw. A unit with several spells shows the realm
// of ONE of them, the first of a fixed order. An invisible unit shows none.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/Combat.c:
// Combat_Unit_Enchantment_Outline_Set, Combat_Unit_Enchantment_Outline_Draw; MoM/src/MainScr.c:
// enchantment_outline_colors, Cycle_Unit_Enchantment_Animation; MoX/src/FLIC_Draw.c:
// Outline_Bitmap_Pixels_No_Glass, Bitmap_Aura_Pixels). The code is ours.
//
// Upstream drew a glow with a shader over the figure, wider than a pixel and not on the pixels
// of the art, in the color of the first spell of the unit's list.

import (
    "fmt"
    "image"
    "image/color"
    "image/draw"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/util"
    "github.com/kazzmir/master-of-magic/lib/lbx"
)

// false: the glow of upstream
const OriginalAura = true

// the realms in the order of the original's table of colors
type auraRealm int
const (
    auraNone auraRealm = iota - 1
    auraChaos
    auraDeath
    auraLife
    auraNature
    auraSorcery
)

// stages of the run of the colors, and redraws of the original one stage lasts
const auraStages = 8
const auraTicksPerStage = 2
// colors of the picture from this one on are no part of the figure (shadows and the like)
const auraFirstOutsideColor = 224

// palette indexes of the first stage, by realm. a stage later the list starts one further
var auraColors = [5][auraStages]int{
    {45, 44, 43, 42, 41, 42, 43, 44},
    {36, 35, 34, 33, 32, 33, 34, 35},
    {11, 12, 13, 14, 15, 14, 13, 12},
    {68, 67, 66, 65, 64, 65, 66, 67},
    {100, 99, 98, 97, 96, 97, 98, 99},
}

// the spells that give an outline, the first one a unit has decides
var auraOrder = []struct{
    Enchantment data.UnitEnchantment
    Realm auraRealm
}{
    {data.UnitEnchantmentRegeneration, auraNature},
    {data.UnitEnchantmentResistMagic, auraSorcery},
    {data.UnitEnchantmentRighteousness, auraLife},
    {data.UnitEnchantmentPlanarTravel, auraLife},
    {data.UnitEnchantmentHaste, auraSorcery},
    {data.UnitEnchantmentBlackChannels, auraDeath},
    {data.UnitEnchantmentLionHeart, auraLife},
    {data.UnitEnchantmentIronSkin, auraNature},
    {data.UnitEnchantmentMagicImmunity, auraSorcery},
    {data.UnitEnchantmentWindWalking, auraSorcery},
    {data.UnitEnchantmentElementalArmor, auraNature},
    {data.UnitEnchantmentTrueSight, auraLife},
    {data.UnitEnchantmentInvulnerability, auraLife},
    {data.UnitEnchantmentPathFinding, auraNature},
    {data.UnitEnchantmentFlight, auraSorcery},
    {data.UnitEnchantmentEldritchWeapon, auraChaos},
    {data.UnitEnchantmentWraithForm, auraDeath},
    {data.UnitEnchantmentGiantStrength, auraNature},
    {data.UnitEnchantmentImmolation, auraChaos},
    {data.UnitEnchantmentGuardianWind, auraSorcery},
    {data.UnitEnchantmentSpellLock, auraSorcery},
    {data.UnitEnchantmentHeroism, auraLife},
    {data.UnitEnchantmentBerserk, auraDeath},
    {data.UnitEnchantmentHolyArmor, auraLife},
    {data.UnitEnchantmentWaterWalking, auraNature},
    {data.UnitEnchantmentEndurance, auraLife},
    {data.UnitEnchantmentCloakOfFear, auraDeath},
    {data.UnitEnchantmentStoneSkin, auraNature},
    {data.UnitEnchantmentFlameBlade, auraChaos},
    {data.UnitEnchantmentBless, auraLife},
    {data.UnitEnchantmentHolyWeapon, auraLife},
    {data.UnitEnchantmentResistElements, auraNature},
}

// the realm of the outline of a unit, auraNone for no outline
func unitAura(unit *ArmyUnit) auraRealm {
    if unit.IsInvisible() || unit.HasEnchantment(data.UnitEnchantmentInvisibility) {
        return auraNone
    }

    for _, entry := range auraOrder {
        if unit.HasEnchantment(entry.Enchantment) {
            return entry.Realm
        }
    }

    return auraNone
}

func (combat *CombatScreen) auraStage() int {
    return int(combat.originalTick() / auraTicksPerStage % auraStages)
}

// which color of the list of a stage the pixels of a picture get, in the order the original
// keeps them in: column by column, each from its top. the step from a pixel to the next grows
// by one each time, so the numbers repeat every 16 pixels
func auraPicks(count int) []int {
    picks := make([]int, count)
    pick := 0
    step := 0
    for index := range picks {
        picks[index] = pick
        pick = (pick + step) % auraStages
        step = (step + 1) % auraStages
    }
    return picks
}

// the pixels of a picture that get the outline: no part of the figure, and right next to one
func auraPixels(picture *image.Paletted) []image.Point {
    bounds := picture.Bounds()

    outside := func(x int, y int) bool {
        index := picture.ColorIndexAt(x, y)
        if index >= auraFirstOutsideColor {
            return true
        }
        _, _, _, alpha := picture.Palette[index].RGBA()
        return alpha == 0
    }

    inside := func(x int, y int) bool {
        return image.Pt(x, y).In(bounds) && !outside(x, y)
    }

    var out []image.Point
    for x := bounds.Min.X; x < bounds.Max.X; x++ {
        for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
            if outside(x, y) && (inside(x - 1, y) || inside(x + 1, y) || inside(x, y - 1) || inside(x, y + 1)) {
                out = append(out, image.Pt(x, y))
            }
        }
    }

    return out
}

// paints the outline into the picture of a figure, after everything else was done to it
func withAura(base util.ImageTransformFunc, realm auraRealm, stage int) util.ImageTransformFunc {
    return func(original *image.Paletted) image.Image {
        // where the outline goes is a matter of the picture as the game has it
        pixels := auraPixels(original)
        bounds := original.Bounds()
        picks := auraPicks(bounds.Dx() * bounds.Dy())

        changed := base(original)
        out := image.NewNRGBA(bounds)
        draw.Draw(out, bounds, changed, bounds.Min, draw.Src)

        // the colors are the ones of the palette of the game
        palette := lbx.GetDefaultPalette()
        for _, pixel := range pixels {
            place := (pixel.X - bounds.Min.X) * bounds.Dy() + (pixel.Y - bounds.Min.Y)
            index := auraColors[realm][(picks[place] + stage) % auraStages]
            if index < len(palette) {
                out.Set(pixel.X, pixel.Y, color.NRGBAModel.Convert(palette[index]))
            }
        }

        return out
    }
}

func auraKey(key string, realm auraRealm, stage int) string {
    return fmt.Sprintf("%v-aura%v-%v", key, int(realm), stage)
}
