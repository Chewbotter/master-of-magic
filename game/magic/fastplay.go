package main

// Debug fast-play options, listed down the left edge of the start screen.
//
// Random Battle: an early army (a random race's starting units, twice over) attacks a cave held by a
// few low level monsters, the way a player might in the first turns of a game. The player controls
// the attackers. When the battle and its results screen are over, the start screen comes back.

import (
    "image"
    "log"
    "math/rand/v2"
    "time"

    "github.com/kazzmir/master-of-magic/game/magic/combat"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/display"
    fontslib "github.com/kazzmir/master-of-magic/game/magic/fonts"
    "github.com/kazzmir/master-of-magic/game/magic/inputmanager"
    "github.com/kazzmir/master-of-magic/game/magic/mainview"
    musiclib "github.com/kazzmir/master-of-magic/game/magic/music"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/game/magic/setup"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    "github.com/kazzmir/master-of-magic/game/magic/util"
    "github.com/kazzmir/master-of-magic/lib/coroutine"
    "github.com/kazzmir/master-of-magic/lib/font"
    mouselib "github.com/kazzmir/master-of-magic/lib/mouse"
    "github.com/kazzmir/master-of-magic/game/magic/mouse"
    "github.com/kazzmir/master-of-magic/lib/optional"

    "github.com/hajimehoshi/ebiten/v2"
)

const ShowFastPlay = true

// where the list starts, in original pixels from the top left corner of the window's picture
const FastPlayX = 4
const FastPlayY = 40
// extra pixels between rows
const FastPlayRowGap = 3

type fastPlayEntry struct {
    Label string
    State mainview.MainScreenState
}

var fastPlayEntries = []fastPlayEntry{
    {Label: "Random Battle", State: mainview.MainScreenStateRandomBattle},
}

// true while the start screen shows, so the list is drawn and clickable
var fastPlayShowing bool
// the start screen's menu itself is up: not its settings, not its load screen
var fastPlayOnMenu bool
// when the start screen's loop last ran. the load screen runs its own loop inside the start screen,
// so the list hides when this goes stale
var fastPlayLastUpdate time.Time
// how long the list stays up without an update
const FastPlayStaleTime = 100 * time.Millisecond
// the entry under the cursor, -1 for none
var fastPlayHover = -1

var fastPlayFont *font.Font
var fastPlayFontHighlight *font.Font

func (game *MagicGame) loadFastPlayFonts() bool {
    if fastPlayFont != nil {
        return true
    }
    if game.Cache == nil {
        return false
    }
    loader, err := fontslib.Loader(game.Cache)
    if err != nil {
        return false
    }
    fastPlayFont = loader(fontslib.SmallYellow)
    fastPlayFontHighlight = loader(fontslib.SmallWhite)
    return fastPlayFont != nil && fastPlayFontHighlight != nil
}

// the rectangle of a row, in original pixels relative to the window's picture. row 0 is the Debug title
func fastPlayRow(row int, text string) image.Rectangle {
    height := fastPlayFont.Height() + FastPlayRowGap
    width := int(fastPlayFont.MeasureTextWidth(text, 1))
    y := FastPlayY + row * height
    return image.Rect(FastPlayX, y, FastPlayX + width, y + fastPlayFont.Height())
}

// the cursor in original pixels relative to the window's picture, ignoring the black bars of the start screen
func fastPlayCursor() image.Point {
    x, y := display.CursorPosition()
    x += display.ContentOffsetX() - display.PictureX()
    return image.Pt(scale.Unscale(x), scale.Unscale(y))
}

// called every tick of the start screen. returns the state of a clicked entry
func (game *MagicGame) updateFastPlay(menu *mainview.MainScreen) (mainview.MainScreenState, bool) {
    fastPlayHover = -1
    fastPlayLastUpdate = time.Now()
    // hidden while the start screen shows its settings
    fastPlayOnMenu = menu.MenuUI == nil
    if !ShowFastPlay || !fastPlayOnMenu || !game.loadFastPlayFonts() {
        return 0, false
    }

    cursor := fastPlayCursor()
    for index, entry := range fastPlayEntries {
        if cursor.In(fastPlayRow(index + 1, entry.Label).Inset(-1)) {
            fastPlayHover = index
        }
    }

    if fastPlayHover >= 0 && inputmanager.LeftClick() {
        return fastPlayEntries[fastPlayHover].State, true
    }

    return 0, false
}

// drawn over the start screen at the left edge of the window, outside its black bars
func (game *MagicGame) drawFastPlay(screen *ebiten.Image) {
    if !ShowFastPlay || !fastPlayShowing || !fastPlayOnMenu || time.Since(fastPlayLastUpdate) > FastPlayStaleTime || !game.loadFastPlayFonts() {
        return
    }

    // positions are in original pixels of the whole window picture, so add its margin
    offsetX := float64(display.PictureX()) / scale.ScaleAmount
    offsetY := float64(display.ContentOffsetY()) / scale.ScaleAmount

    print := func(useFont *font.Font, row int, text string) {
        rect := fastPlayRow(row, text)
        useFont.PrintOutlined(screen, float64(rect.Min.X) + offsetX, float64(rect.Min.Y) + offsetY, font.FontOptions{Scale: scale.ScaleAmount}, font.OutlineFull, text)
    }

    print(fastPlayFontHighlight, 0, "Debug")
    for index, entry := range fastPlayEntries {
        use := fastPlayFont
        if index == fastPlayHover {
            use = fastPlayFontHighlight
        }
        print(use, index + 1, entry.Label)
    }
}

// low level monsters that hold caves early in a game
var randomBattleMonsters = []units.Unit{units.WarBear, units.GiantSpiders, units.Skeleton, units.Zombie, units.HellHounds, units.Ghoul}
const RandomBattleMinMonsters = 2
const RandomBattleMaxMonsters = 4
// how many copies of the race's starting units the attacking army has
const RandomBattleArmyCopies = 2

var randomBattleRaces = []data.Race{
    data.RaceLizard, data.RaceNomad, data.RaceOrc, data.RaceTroll, data.RaceBarbarian, data.RaceBeastmen, data.RaceDarkElf,
    data.RaceDraconian, data.RaceDwarf, data.RaceGnoll, data.RaceHalfling, data.RaceHighElf, data.RaceHighMen, data.RaceKlackon,
}

var randomBattleLandscapes = []combat.CombatLandscape{combat.CombatLandscapeGrass, combat.CombatLandscapeDesert, combat.CombatLandscapeMountain, combat.CombatLandscapeTundra}

func runRandomBattle(yield coroutine.YieldFunc, game *MagicGame) {
    allSpells, err := spellbook.ReadSpellsFromCache(game.Cache)
    if err != nil {
        log.Printf("Random battle: unable to read spells: %v", err)
        allSpells = spellbook.Spells{}
    }

    race := randomChoose(randomBattleRaces...)
    banner := randomChoose(data.BannerGreen, data.BannerBlue, data.BannerRed, data.BannerPurple, data.BannerYellow)

    attacker := playerlib.MakePlayer(setup.WizardCustom{Name: race.String(), Banner: banner, Race: race}, true, 0, 0, nil, &playerlib.NoGlobalEnchantments{})
    defender := playerlib.MakePlayer(setup.WizardCustom{Name: "Cave", Banner: data.BannerBrown}, false, 0, 0, nil, &playerlib.NoGlobalEnchantments{})

    attackingArmy := &combat.Army{Player: attacker}
    for range RandomBattleArmyCopies {
        for _, unit := range startingUnits(race) {
            attackingArmy.AddUnit(units.MakeOverworldUnitFromUnit(unit, 1, 1, data.PlaneArcanus, attacker.Wizard.Banner, attacker.MakeExperienceInfo(), attacker.MakeUnitEnchantmentProvider()))
        }
    }

    defendingArmy := &combat.Army{Player: defender}
    monster := randomChoose(randomBattleMonsters...)
    count := RandomBattleMinMonsters + rand.N(RandomBattleMaxMonsters - RandomBattleMinMonsters + 1)
    for range count {
        defendingArmy.AddUnit(units.MakeOverworldUnitFromUnit(monster, 1, 1, data.PlaneArcanus, defender.Wizard.Banner, defender.MakeExperienceInfo(), defender.MakeUnitEnchantmentProvider()))
    }

    landscape := randomChoose(randomBattleLandscapes...)
    zone := combat.ZoneType{Encounter: combat.ZoneLair}

    log.Printf("Random battle: %v army attacks a cave with %v %v, landscape %v", race, count, monster.Name, landscape)

    events := make(chan combat.CombatEvent, 1000)
    model := combat.MakeCombatModel(allSpells, defendingArmy, attackingArmy, landscape, data.PlaneArcanus, zone, data.MagicNone, 0, 0, events)
    combatScreen := combat.MakeCombatScreen(game.Cache, defendingArmy, attackingArmy, optional.Of[combat.ArmyPlayer](attacker), landscape, data.PlaneArcanus, zone, model)
    if combatScreen == nil {
        log.Printf("Random battle: unable to make the combat screen")
        return
    }

    defer game.restoreNormalMouse()

    game.Drawer = func(screen *ebiten.Image) {
        combatScreen.Draw(screen)
    }

    game.Music.PlaySong(randomChoose(musiclib.SongCombat1, musiclib.SongCombat2))

    state := combat.CombatStateRunning
    for state == combat.CombatStateRunning {
        state = combatScreen.Update(yield)
        // the battle's own events are for the world map, nobody here needs them
        for len(events) > 0 {
            <-events
        }
        if yield() != nil {
            return
        }
    }

    result := combat.CombatEndScreenResultWin
    switch state {
        case combat.CombatStateDefenderWin: result = combat.CombatEndScreenResultLose
        case combat.CombatStateAttackerFlee: result = combat.CombatEndScreenResultRetreat
    }

    endScreen := combat.MakeCombatEndScreen(game.Cache, result, model.DefeatedAttackers, 0, 0, 0)
    game.Drawer = func(screen *ebiten.Image) {
        endScreen.Draw(screen)
    }
    for endScreen.Update() == combat.CombatEndScreenRunning {
        if yield() != nil {
            return
        }
    }
    yield()
}

func (game *MagicGame) restoreNormalMouse() {
    imageCache := util.MakeImageCache(game.Cache)
    normalMouse, err := mouselib.GetMouseNormal(game.Cache, &imageCache)
    if err == nil {
        mouse.Mouse.SetImage(normalMouse)
    }
}
