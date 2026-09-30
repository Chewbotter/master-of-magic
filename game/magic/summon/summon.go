package summon

import (
    "fmt"
    "log"
    "image"
    "image/color"

    "github.com/kazzmir/master-of-magic/lib/lbx"
    "github.com/kazzmir/master-of-magic/lib/font"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    "github.com/kazzmir/master-of-magic/game/magic/util"

    "github.com/hajimehoshi/ebiten/v2"
)

type SummonState int
const (
    SummonStateRunning SummonState = iota
    SummonStateDone
)

type Summon struct {
    Counter uint64
    Cache *lbx.LbxCache
    ImageCache util.ImageCache
    Title string
    Wizard data.WizardBase
    State SummonState
    Font *font.Font
    // the title of the original, see style.go. nil for upstream's
    Styled *font.StyledFont
    CircleBack *util.Animation
    CircleFront *util.Animation
    Background *ebiten.Image
    SummonPic *ebiten.Image
    SummonHeight int
}

// THE COLORS OF THE SUMMONING CIRCLE (user, 2026-09-30: the gray ring of the floor stayed gray
// while the flames took the color of the realm). The pictures of the circle (spellscr.lbx 10 and
// 11) and of the floor (9) are indexed, all of their colors of the realm are at the indexes 224
// to 254. Before, the fork rewrote 225 to 247 with steps of one color of its own, which left 224
// and 248 to 254, the ring among them, as they were.
//
// The original does not compute anything: spellscr.lbx has a whole palette for every realm,
// entries 61 to 66 (SUMMGREY, SUMGREEN, SUMMBLUE, SUMMRED, SUMWHITE, SUMMPURP), and loads the
// one of the realm before it draws (ReMoM Spells137.c Summon_Animation: Load_Palette_From_Animation
// of that entry, then Apply_Palette). The code here is ours.
//
// So: the palette of the realm is read from that entry (lbx.GetPalette gives the game's 256
// colors with the entry's own laid over them) and the pictures are shown with it. If the entry
// can not be read, the old steps of the color of the realm are used.

// the entries of spellscr.lbx that hold the palette of a realm
const summonPaletteGrey = 61
const summonPaletteGreen = 62
const summonPaletteBlue = 63
const summonPaletteRed = 64
const summonPaletteWhite = 65
const summonPalettePurple = 66

// the first and the last color of a picture of the summoning that the palette of the realm changes
const summonRealmColorFirst = 224
const summonRealmColorLast = 254

// the palette entry of a realm. items and heroes are summoned in white
func summonPaletteEntry(realm data.MagicType) int {
    switch realm {
        case data.NatureMagic: return summonPaletteGreen
        case data.SorceryMagic: return summonPaletteBlue
        case data.ChaosMagic: return summonPaletteRed
        case data.LifeMagic: return summonPaletteWhite
        case data.DeathMagic: return summonPalettePurple
        case data.ArcaneMagic: return summonPaletteGrey
    }
    return summonPaletteWhite
}

// gives the palette of a picture of the summoning with the colors of the realm in it
func (summon *Summon) realmPalette(realm data.MagicType, baseColor color.Color) func(color.Palette) color.Palette {
    var ofRealm color.Palette
    file, err := summon.Cache.GetLbxFile("spellscr.lbx")
    if err == nil {
        ofRealm, err = file.GetPalette(summonPaletteEntry(realm))
        if err != nil {
            ofRealm = nil
        }
    }

    return func(palette color.Palette) color.Palette {
        out := make(color.Palette, len(palette))
        copy(out, palette)

        if len(ofRealm) > summonRealmColorLast {
            for index := summonRealmColorFirst; index <= summonRealmColorLast && index < len(out); index++ {
                out[index] = ofRealm[index]
            }
            return out
        }

        // the palette of the realm is not there: steps of its color, as before
        light := 0
        for index := 225; index <= 247 && index < len(out); index++ {
            out[index] = util.Lighten(baseColor, float64(light))
            light -= 4
        }
        return out
    }
}

func makeSummon(cache *lbx.LbxCache, title string, wizard data.WizardBase, summonPic *ebiten.Image, realm data.MagicType, baseColor color.Color, short bool) *Summon {
    summon := &Summon{
        Cache: cache,
        ImageCache: util.MakeImageCache(cache),
        Wizard: wizard,
        Title: title,
        SummonPic: summonPic,
        State: SummonStateRunning,
    }

    if short {
        summon.SummonHeight = summon.SummonPic.Bounds().Dy()
        summon.Counter = 300
    }

    // the colors of the realm, see realmPalette
    realmColors := summon.realmPalette(realm, baseColor)
    updateColors := func (img *image.Paletted) image.Image {
        img.Palette = realmColors(img.Palette)
        return img
    }

    summonBack, _ := summon.ImageCache.GetImagesTransform("spellscr.lbx", 10, "colors", updateColors)
    summon.CircleBack = util.MakeAnimation(summonBack, true)

    summonFront, _ := summon.ImageCache.GetImagesTransform("spellscr.lbx", 11, "colors", updateColors)
    summon.CircleFront = util.MakeAnimation(summonFront, true)

    background, _ := summon.ImageCache.GetImageTransform("spellscr.lbx", 9, 0, "colors", updateColors)
    summon.Background = background

    fontLbx, err := cache.GetLbxFile("fonts.lbx")
    if err != nil {
        return nil
    }

    fonts, err := font.ReadFonts(fontLbx, 0)
    if err != nil {
        return nil
    }

    orange := color.RGBA{R: 0xc7, G: 0x82, B: 0x1b, A: 0xff}

    yellowPalette := color.Palette{
        color.RGBA{R: 0, G: 0, B: 0x00, A: 0},
        util.Lighten(orange, 0),
        util.Lighten(orange, 15),
        util.Lighten(orange, 30),
        util.Lighten(orange, 50),
        util.Lighten(orange, 70),
        util.Lighten(orange, 90),
    }

    infoFontYellow := font.MakeOptimizedFontWithPalette(fonts[4], yellowPalette)
    summon.Font = infoFontYellow

    summon.Styled = summonTitle(cache)
    if summon.Styled != nil && title == "Artifact Summoned" {
        summon.Title = summonItemTitle
    }

    return summon
}

func getMonsterIndex(unit units.Unit) int {
    monsterIndex := 0
    // magic spirit is monster.lbx, 0
    if unit.Equals(units.MagicSpirit) {
        monsterIndex = 0
    } else if unit.Equals(units.HellHounds) {
        monsterIndex = 1
    } else if unit.Equals(units.Gargoyle) {
        monsterIndex = 2
    } else if unit.Equals(units.FireGiant) {
        monsterIndex = 3
    } else if unit.Equals(units.ChaosSpawn) {
        monsterIndex = 5
    } else if unit.Equals(units.Chimeras) {
        monsterIndex = 6
    } else if unit.Equals(units.DoomBat) {
        monsterIndex = 7
    } else if unit.Equals(units.Efreet) {
        monsterIndex = 8
    } else if unit.Equals(units.Hydra) {
        monsterIndex = 9
    } else if unit.Equals(units.GreatDrake) {
        monsterIndex = 10
    } else if unit.Equals(units.Skeleton) {
        monsterIndex = 11
    } else if unit.Equals(units.Ghoul) {
        monsterIndex = 12
    } else if unit.Equals(units.NightStalker) {
        monsterIndex = 13
    } else if unit.Equals(units.WereWolf) {
        monsterIndex = 14
    } else if unit.Equals(units.Wraith) {
        monsterIndex = 16
    } else if unit.Equals(units.ShadowDemons) {
        monsterIndex = 17
    } else if unit.Equals(units.DeathKnights) {
        monsterIndex = 18
    } else if unit.Equals(units.DemonLord) {
        monsterIndex = 19
    } else if unit.Equals(units.Unicorn) {
        monsterIndex = 21
    } else if unit.Equals(units.GuardianSpirit) {
        monsterIndex = 22
    } else if unit.Equals(units.Angel) {
        monsterIndex = 23
    } else if unit.Equals(units.ArchAngel) {
        monsterIndex = 24
    } else if unit.Equals(units.WarBear) {
        monsterIndex = 25
    } else if unit.Equals(units.Sprites) {
        monsterIndex = 26
    } else if unit.Equals(units.Cockatrices) {
        monsterIndex = 27
    } else if unit.Equals(units.Basilisk) {
        monsterIndex = 28
    } else if unit.Equals(units.GiantSpiders) {
        monsterIndex = 29
    } else if unit.Equals(units.StoneGiant) {
        monsterIndex = 30
    } else if unit.Equals(units.Colossus) {
        monsterIndex = 31
    } else if unit.Equals(units.Gorgon) {
        monsterIndex = 32
    } else if unit.Equals(units.EarthElemental) {
        monsterIndex = 33
    } else if unit.Equals(units.Behemoth) {
        monsterIndex = 34
    } else if unit.Equals(units.GreatWyrm) {
        monsterIndex = 35
    } else if unit.Equals(units.StormGiant) {
        monsterIndex = 39
    } else if unit.Equals(units.Djinn) {
        monsterIndex = 41
    } else if unit.Equals(units.SkyDrake) {
        monsterIndex = 42
    } else if unit.Equals(units.Nagas) {
        monsterIndex = 43
    } else {
        log.Printf("Invalid summoning for unit %v", unit)
    }

    return monsterIndex
}

func getRealmColor(unit units.Unit) color.Color {
    baseColor := color.RGBA{R: 0, B: 0, G: 0xff, A: 0xff}
    switch unit.Realm {
        case data.LifeMagic: baseColor = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
        case data.SorceryMagic: baseColor = color.RGBA{R: 0, G: 0, B: 0xff, A: 0xff}
        case data.NatureMagic: baseColor = color.RGBA{R: 0, B: 0, G: 0xff, A: 0xff}
        case data.DeathMagic: baseColor = color.RGBA{R: 0xd6, G: 0x63, B: 0xff, A: 0xff}
        case data.ChaosMagic: baseColor = color.RGBA{R: 0xff, G: 0, B: 0, A: 0xff}
        case data.ArcaneMagic: baseColor = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
    }

    return baseColor
}

func MakeSummonUnit(cache *lbx.LbxCache, unit units.Unit, wizard data.WizardBase, short bool) *Summon {
    imageCache := util.MakeImageCache(cache)

    monsterIndex := getMonsterIndex(unit)
    monsterPicture, err := imageCache.GetImage("monster.lbx", monsterIndex, 0)
    if err != nil {
        log.Printf("Error: could not load monster image at index %v: %v", monsterIndex, err)
    }

    return makeSummon(cache, fmt.Sprintf("%v Summoned", unit.Name), wizard, monsterPicture, unit.Realm, getRealmColor(unit), short)
}

func MakeSummonArtifact(cache *lbx.LbxCache, wizard data.WizardBase, short bool) *Summon {
    imageCache := util.MakeImageCache(cache)

    artifactIndex := 46
    monsterPicture, err := imageCache.GetImage("monster.lbx", artifactIndex, 0)
    if err != nil {
        log.Printf("Error: could not load artifact image at index %v: %v", artifactIndex, err)
    }

    return makeSummon(cache, "Artifact Summoned", wizard, monsterPicture, data.LifeMagic, color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, short)
}

func MakeSummonHero(cache *lbx.LbxCache, wizard data.WizardBase, champion bool, short bool, female bool) *Summon {
    imageCache := util.MakeImageCache(cache)

    // female 44
    // male 45

    heroIndex := 45

    if female {
        heroIndex = 44
    }

    heroPicture, err := imageCache.GetImage("monster.lbx", heroIndex, 0)
    if err != nil {
        log.Printf("Error: could not load hero image at index %v: %v", heroIndex, err)
    }

    title := "Hero Summoned"
    if champion {
        title = "Champion Summoned"
    }

    return makeSummon(cache, title, wizard, heroPicture, data.LifeMagic, color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, short)
}

func (summon *Summon) Update() SummonState {
    summon.Counter += 1

    if summon.Counter % 7 == 0 {
        summon.CircleBack.Next()
        summon.CircleFront.Next()
    }

    if summon.Counter % 2 == 0 {
        if summon.SummonHeight < summon.SummonPic.Bounds().Dy() {
            summon.SummonHeight += 1
        }
    }

    if summon.Counter > 400 {
        return SummonStateDone
    }

    return summon.State
}

func (summon *Summon) Draw(screen *ebiten.Image){

    // background, _ := summon.ImageCache.GetImage("spellscr.lbx", 9, 0)
    frameY, wizardX, wizardTop, circleX, subjectX := 40, 7, 0, 53, 75
    if summon.Styled != nil {
        frameY, wizardX, wizardTop, circleX, subjectX = summonFrameY, summonWizardX, summonWizardTop - 3, summonCircleX, summonSubjectX
    }

    var options ebiten.DrawImageOptions
    options.GeoM.Translate(float64(30), float64(frameY))
    scale.DrawScaled(screen, summon.Background, &options)

    wizardIndex := 46
    switch summon.Wizard {
        case data.WizardMerlin: wizardIndex = 46
        case data.WizardRaven: wizardIndex = 47
        case data.WizardSharee: wizardIndex = 48
        case data.WizardLoPan: wizardIndex = 49
        case data.WizardJafar: wizardIndex = 50
        case data.WizardOberic: wizardIndex = 51
        case data.WizardRjak: wizardIndex = 52
        case data.WizardSssra: wizardIndex = 53
        case data.WizardTauron: wizardIndex = 54
        case data.WizardFreya: wizardIndex = 55
        case data.WizardHorus: wizardIndex = 56
        case data.WizardAriel: wizardIndex = 57
        case data.WizardTlaloc: wizardIndex = 58
        case data.WizardKali: wizardIndex = 59
    }

    circleOptions := options
    circleOptions.GeoM.Translate(float64(circleX), float64(54))
    scale.DrawScaled(screen, summon.CircleBack.Frame(), &circleOptions)

    wizard, _ := summon.ImageCache.GetImage("spellscr.lbx", wizardIndex, 0)
    wizardOptions := options
    wizardOptions.GeoM.Translate(float64(wizardX), float64(3 + wizardTop))
    if wizardTop > 0 && wizard != nil {
        // nothing of the wizard shows over the inside of the frame
        bounds := wizard.Bounds()
        wizard = wizard.SubImage(image.Rect(bounds.Min.X, bounds.Min.Y + wizardTop, bounds.Max.X, bounds.Max.Y)).(*ebiten.Image)
    }
    scale.DrawScaled(screen, wizard, &wizardOptions)

    monster := summon.SummonPic
    monsterOptions := options
    monsterOptions.GeoM.Translate(float64(subjectX), float64((30 + 70)) - float64(summon.SummonHeight))
    partialMonster := monster.SubImage(image.Rect(0, 0, monster.Bounds().Dx(), summon.SummonHeight)).(*ebiten.Image)
    scale.DrawScaled(screen, partialMonster, &monsterOptions)

    circleOptions.GeoM.Translate(float64(53 + 11 - circleX), float64(26))
    circleOptions.ColorScale.ScaleAlpha(1.0)
    scale.DrawScaled(screen, summon.CircleFront.Frame(), &circleOptions)

    if summon.Styled != nil {
        var plain ebiten.DrawImageOptions
        summon.Styled.Print(screen, 30 + summonTitleMiddle, frameY + summonTitleY, font.FontOptions{Options: &plain, Scale: scale.ScaleAmount, Justify: font.FontJustifyCenter}, summon.Title)
        return
    }

    x, y := options.GeoM.Apply(float64(summon.Background.Bounds().Dx())/2, float64(summon.Background.Bounds().Dy() - 18))
    summon.Font.PrintCenter(screen, x, y, scale.ScaleAmount, options.ColorScale, summon.Title)
}
