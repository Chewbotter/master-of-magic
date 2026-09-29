package aura

// The outline of a unit that has a spell on it, and of an item with powers, matched to the
// original game. For battles, the world map and every window that shows a unit.
//
// The original draws a rim of ONE PIXEL around the figure, outside of it, in the five shades of
// the realm of the spell, which run around the rim. It paints it into the picture: every empty
// pixel right next to a pixel of the figure (left, right, above, below) gets a color of a list of
// 8, picked by the place of the pixel in the picture, and the list is another one for each of 8
// stages, one stage every second redraw. A unit with several spells shows the realm of ONE of
// them, the first of a fixed order. An invisible unit shows none.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/MainScr.c:
// Draw_Unit_Enchantment_Outline, enchantment_outline_colors, Cycle_Unit_Enchantment_Animation;
// MoM/src/Combat.c: Combat_Unit_Enchantment_Outline_Set, Combat_Unit_Enchantment_Outline_Draw;
// MoM/src/ItemView.c; MoX/src/FLIC_Draw.c: Outline_Bitmap_Pixels_No_Glass, Bitmap_Aura_Pixels).
// The code is ours.
//
// Upstream drew a glow with a shader over the picture (util.DrawOutline), wider than a pixel and
// not on the pixels of the art, in the color of the first spell of the unit's list.

import (
    "image"
    "image/color"
    "slices"
    "time"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/lib/lbx"

    "github.com/hajimehoshi/ebiten/v2"
)

// the realms in the order of the original's table of colors
type Realm int
const (
    None Realm = iota - 1
    Chaos
    Death
    Life
    Nature
    Sorcery
)

// stages of the run of the colors
const Stages = 8
// redraws of the original one stage lasts, and its redraws in a second
const TicksPerStage = 2
const ticksPerSecond = 18.2
// colors of a picture from this one on are no part of the figure (shadows and the like)
const firstOutsideColor = 224
// pictures of outlines that are kept. when there are more all are made anew
const keptPictures = 4000

// palette indexes of the first stage, by realm. a stage later the list starts one further
var colors = [5][Stages]int{
    {45, 44, 43, 42, 41, 42, 43, 44},
    {36, 35, 34, 33, 32, 33, 34, 35},
    {11, 12, 13, 14, 15, 14, 13, 12},
    {68, 67, 66, 65, 64, 65, 66, 67},
    {100, 99, 98, 97, 96, 97, 98, 99},
}

// the spells that give an outline, the first one a unit has decides
var order = []struct{
    Enchantment data.UnitEnchantment
    Realm Realm
}{
    {data.UnitEnchantmentRegeneration, Nature},
    {data.UnitEnchantmentResistMagic, Sorcery},
    {data.UnitEnchantmentRighteousness, Life},
    {data.UnitEnchantmentPlanarTravel, Life},
    {data.UnitEnchantmentHaste, Sorcery},
    {data.UnitEnchantmentBlackChannels, Death},
    {data.UnitEnchantmentLionHeart, Life},
    {data.UnitEnchantmentIronSkin, Nature},
    {data.UnitEnchantmentMagicImmunity, Sorcery},
    {data.UnitEnchantmentWindWalking, Sorcery},
    {data.UnitEnchantmentElementalArmor, Nature},
    {data.UnitEnchantmentTrueSight, Life},
    {data.UnitEnchantmentInvulnerability, Life},
    {data.UnitEnchantmentPathFinding, Nature},
    {data.UnitEnchantmentFlight, Sorcery},
    {data.UnitEnchantmentEldritchWeapon, Chaos},
    {data.UnitEnchantmentWraithForm, Death},
    {data.UnitEnchantmentGiantStrength, Nature},
    {data.UnitEnchantmentImmolation, Chaos},
    {data.UnitEnchantmentGuardianWind, Sorcery},
    {data.UnitEnchantmentSpellLock, Sorcery},
    {data.UnitEnchantmentHeroism, Life},
    {data.UnitEnchantmentBerserk, Death},
    {data.UnitEnchantmentHolyArmor, Life},
    {data.UnitEnchantmentWaterWalking, Nature},
    {data.UnitEnchantmentEndurance, Life},
    {data.UnitEnchantmentCloakOfFear, Death},
    {data.UnitEnchantmentStoneSkin, Nature},
    {data.UnitEnchantmentFlameBlade, Chaos},
    {data.UnitEnchantmentBless, Life},
    {data.UnitEnchantmentHolyWeapon, Life},
    {data.UnitEnchantmentResistElements, Nature},
}

// the realm of the outline of a unit with these spells on it, None for no outline
func Of(enchantments []data.UnitEnchantment) Realm {
    if len(enchantments) == 0 || slices.Contains(enchantments, data.UnitEnchantmentInvisibility) {
        return None
    }

    for _, entry := range order {
        if slices.Contains(enchantments, entry.Enchantment) {
            return entry.Realm
        }
    }

    return None
}

// the spell that decides about the outline of a unit, data.UnitEnchantmentNone for no outline
func Deciding(enchantments []data.UnitEnchantment) data.UnitEnchantment {
    if slices.Contains(enchantments, data.UnitEnchantmentInvisibility) {
        return data.UnitEnchantmentNone
    }

    for _, entry := range order {
        if slices.Contains(enchantments, entry.Enchantment) {
            return entry.Enchantment
        }
    }

    return data.UnitEnchantmentNone
}

// the realm of the outline of an item with powers of this magic
func OfMagic(magic data.MagicType) Realm {
    switch magic {
        case data.ChaosMagic: return Chaos
        case data.DeathMagic: return Death
        case data.LifeMagic: return Life
        case data.NatureMagic: return Nature
        case data.SorceryMagic: return Sorcery
    }
    return None
}

// the stage at a tick of the original
func StageAt(tick uint64) int {
    return int(tick / TicksPerStage % Stages)
}

var started = time.Now()

// the stage of now, for screens that do not count the ticks of the original
func StageNow() int {
    return StageAt(uint64(time.Since(started).Seconds() * ticksPerSecond))
}

// the color of the outline of a realm
func Color(realm Realm, stage int, pick int) color.Color {
    palette := lbx.GetDefaultPalette()
    index := colors[realm][(pick + stage) % Stages]
    if index >= len(palette) {
        return color.NRGBA{}
    }
    return color.NRGBAModel.Convert(palette[index])
}

// which color of the list of a stage the pixels of a picture get, in the order the original
// keeps them in: column by column, each from its top. the step from a pixel to the next grows
// by one each time, so the numbers repeat every 16 pixels
func Picks(count int) []int {
    picks := make([]int, count)
    pick := 0
    step := 0
    for index := range picks {
        picks[index] = pick
        pick = (pick + step) % Stages
        step = (step + 1) % Stages
    }
    return picks
}

// the pixels of a picture that get the outline: no part of the figure, and right next to one.
// outside says whether a pixel is no part of the figure
func pixelsOf(bounds image.Rectangle, outside func(x int, y int) bool) []image.Point {
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

// the pixels of the outline of a picture with its palette
func Pixels(picture *image.Paletted) []image.Point {
    return pixelsOf(picture.Bounds(), func(x int, y int) bool {
        index := picture.ColorIndexAt(x, y)
        if index >= firstOutsideColor {
            return true
        }
        _, _, _, alpha := picture.Palette[index].RGBA()
        return alpha == 0
    })
}

// paints the outline on a picture as large as the one the pixels are of
func Paint(out *image.NRGBA, pixels []image.Point, realm Realm, stage int) {
    bounds := out.Bounds()
    picks := Picks(bounds.Dx() * bounds.Dy())
    for _, pixel := range pixels {
        place := (pixel.X - bounds.Min.X) * bounds.Dy() + (pixel.Y - bounds.Min.Y)
        out.Set(pixel.X, pixel.Y, Color(realm, stage, picks[place]))
    }
}

type pictureKey struct {
    Picture *ebiten.Image
    Realm Realm
    Stage int
}

var pictures = make(map[pictureKey]*ebiten.Image)
var outlines = make(map[*ebiten.Image][]image.Point)

// the outline of a picture that is drawn, as a picture of its own to draw over it. such a
// picture has no palette any more: every pixel that is not empty counts as the figure
func overlay(picture *ebiten.Image, realm Realm, stage int) *ebiten.Image {
    key := pictureKey{Picture: picture, Realm: realm, Stage: stage}
    made, ok := pictures[key]
    if ok {
        return made
    }

    if len(pictures) >= keptPictures {
        for _, old := range pictures {
            old.Deallocate()
        }
        clear(pictures)
        clear(outlines)
    }

    size := picture.Bounds().Size()
    bounds := image.Rect(0, 0, size.X, size.Y)

    pixels, ok := outlines[picture]
    if !ok {
        read := make([]byte, 4 * size.X * size.Y)
        picture.ReadPixels(read)
        pixels = pixelsOf(bounds, func(x int, y int) bool {
            return read[(y * size.X + x) * 4 + 3] == 0
        })
        outlines[picture] = pixels
    }

    painted := image.NewNRGBA(bounds)
    Paint(painted, pixels, realm, stage)
    made = ebiten.NewImageFromImage(painted)
    pictures[key] = made
    return made
}

// draws the outline over a picture that was drawn with this matrix
func Draw(screen *ebiten.Image, picture *ebiten.Image, geom ebiten.GeoM, colorScale ebiten.ColorScale, realm Realm) {
    if realm == None || picture == nil {
        return
    }

    var options ebiten.DrawImageOptions
    options.GeoM = geom
    options.ColorScale = colorScale
    screen.DrawImage(overlay(picture, realm, StageNow()), &options)
}

// the outline of a unit with these spells on it
func DrawUnit(screen *ebiten.Image, picture *ebiten.Image, geom ebiten.GeoM, colorScale ebiten.ColorScale, enchantments []data.UnitEnchantment) {
    Draw(screen, picture, geom, colorScale, Of(enchantments))
}
