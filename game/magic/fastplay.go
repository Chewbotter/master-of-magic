package main

// Debug fast-play options, listed down the left edge of the start screen.
//
// Random Battle: an early army (a random race's starting units, twice over) attacks a cave held by a
// few low level monsters, the way a player might in the first turns of a game. The player controls
// the attackers. When the battle and its results screen are over, the start screen comes back.

import (
    "fmt"
    "image"
    "log"
    "math/rand/v2"
    "strconv"
    "strings"
    "time"

    buildinglib "github.com/kazzmir/master-of-magic/game/magic/building"
    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
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
    // a setting: a click changes it and the start screen stays. its text replaces the label
    Change func()
    Text func() string
    // the entry is not in the list while this says so
    Hidden func() bool
    // called before the state of the entry is returned
    Pick func()
}

// the entries that are in the list now
func shownFastPlayEntries() []fastPlayEntry {
    var out []fastPlayEntry
    for _, entry := range fastPlayEntries {
        if entry.Hidden == nil || !entry.Hidden() {
            out = append(out, entry)
        }
    }
    return out
}

func (entry fastPlayEntry) text() string {
    if entry.Text != nil {
        return entry.Text()
    }
    return entry.Label
}

var fastPlayEntries = []fastPlayEntry{
    {Label: "Random Battle", State: mainview.MainScreenStateRandomBattle},
    {Label: "Random City Battle", State: mainview.MainScreenStateRandomCityBattle},
    {
        // a battle of one kind of unit, picked from a list. see unitpicker.go
        Label: "Test Battle...",
        Change: func() {
            unitPickerOpen = true
        },
    },
    {
        // the unit of the last test battle once more, for looking at a change of its pictures
        State: mainview.MainScreenStateTestBattle,
        Hidden: func() bool {
            return testBattleLast == nil
        },
        Pick: func() {
            testBattleUnit = testBattleLast
        },
        Text: func() string {
            return fmt.Sprintf("Again: %v", testBattleLast.Name)
        },
    },
    {
        Change: func() {
            if randomBattleArmyScale == 1 {
                randomBattleArmyScale = RandomBattleLargeArmies
            } else {
                randomBattleArmyScale = 1
            }
            // kept for the next run of the game, see debugsaved.go
            saveDebugSaved()
        },
        Text: func() string {
            return fmt.Sprintf("Army Size: x%v", randomBattleArmyScale)
        },
    },
}

// how many times the units of both armies of a random battle are multiplied. a setting of the debug list
var randomBattleArmyScale = 1
const RandomBattleLargeArmies = 3

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

    // what the last run of the game left, see debugsaved.go
    loadDebugSaved()

    if capture.UnitPicker != "" {
        // development: the list of units, with races open
        for _, race := range unitPickerRaces {
            if capture.UnitPicker == "all" || strings.Contains(strings.ToLower(capture.UnitPicker), strings.ToLower(race.String())) {
                unitPickerExpanded[race] = true
            }
        }
        capture.UnitPicker = ""
        unitPickerOpen = true
    }

    if unitPickerOpen {
        // the list of units of the test battle takes the clicks, see unitpicker.go
        if updateUnitPicker() {
            return mainview.MainScreenStateTestBattle, true
        }
        return 0, false
    }

    entries := shownFastPlayEntries()
    cursor := fastPlayCursor()
    for index, entry := range entries {
        if cursor.In(fastPlayRow(index + 1, entry.text()).Inset(-1)) {
            fastPlayHover = index
        }
    }

    if fastPlayHover >= 0 && inputmanager.LeftClick() {
        entry := entries[fastPlayHover]
        if entry.Change != nil {
            entry.Change()
            return 0, false
        }
        if entry.Pick != nil {
            entry.Pick()
        }
        return entry.State, true
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

    if unitPickerOpen {
        drawUnitPicker(screen)
        return
    }

    print(fastPlayFontHighlight, 0, "Debug")
    for index, entry := range shownFastPlayEntries() {
        use := fastPlayFont
        if index == fastPlayHover {
            use = fastPlayFontHighlight
        }
        print(use, index + 1, entry.text())
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

// the city of a random city battle
const RandomCityMinCitizens = 2
const RandomCityMaxCitizens = 16
// one in this many
const RandomCityWallsChance = 2
const RandomCityFortressChance = 3
const RandomCityMagicWallChance = 5

type randomCity struct {
    Citizens int
    Walls bool
    Fortress bool
    Fire bool
    Darkness bool
    Outpost bool
    Myrror bool
}

func makeRandomCity() randomCity {
    city := randomCity{
        Citizens: RandomCityMinCitizens + rand.N(RandomCityMaxCitizens - RandomCityMinCitizens + 1),
        Walls: rand.N(RandomCityWallsChance) == 0,
        Fortress: rand.N(RandomCityFortressChance) == 0,
        Fire: rand.N(RandomCityMagicWallChance) == 0,
        Darkness: rand.N(RandomCityMagicWallChance) == 0,
    }

    // development: the city a capture asks for
    if capture.CityBattle != "" && capture.CityBattle != "random" {
        city = randomCity{Citizens: city.Citizens}
        for _, word := range strings.Split(capture.CityBattle, ",") {
            switch {
                case word == "walls": city.Walls = true
                case word == "fortress": city.Fortress = true
                case word == "fire": city.Fire = true
                case word == "darkness": city.Darkness = true
                case word == "outpost": city.Outpost = true
                case word == "myrror": city.Myrror = true
                case strings.HasPrefix(word, "size="):
                    size, err := strconv.Atoi(strings.TrimPrefix(word, "size="))
                    if err == nil {
                        city.Citizens = size
                    }
            }
        }
    }

    return city
}

// the mana and the casting skill the player has in a debug battle, filled up again all the time.
// as large as the combat bar has room for
const DebugBattleMana = 9999
// enough for the casting skill to be above the mana
const DebugBattleSkillPower = 100000000
// the rule of one spell a turn is off in debug battles, to try one spell after the other
const DebugBattleManySpells = true

// development: shows the picture of a spell on the first unit of the enemy, without what it does
func captureSpell(screen *combat.CombatScreen, model *combat.CombatModel, attackers *combat.Army, defenders *combat.Army) {
    name, frameText, _ := strings.Cut(capture.Spell, "@")
    startFrame, err := strconv.Atoi(frameText)
    if err != nil {
        startFrame = 1
    }

    capture.EachFrame = func(frame int) {
        if frame != startFrame || len(defenders.GetUnits()) == 0 {
            return
        }

        target := defenders.GetUnits()[0]
        text := attackers.Player.GetWizard().Name + " has cast " + name

        switch name {
            case "firebolt": model.AddProjectile(screen.CreateFireBoltProjectile(target, 0))
            case "fireball": model.AddProjectile(screen.CreateFireballProjectile(target, 0))
            case "icebolt": model.AddProjectile(screen.CreateIceBoltProjectile(target, 0))
            case "doombolt": model.AddProjectile(screen.CreateDoomBoltProjectile(target))
            case "lightning": model.AddProjectile(screen.CreateLightningBoltProjectile(target, 0))
            case "warplightning": model.AddProjectile(screen.CreateWarpLightningProjectile(target))
            case "confusion": model.AddProjectile(screen.CreateConfusionProjectile(target, 100))
            case "bless": model.AddProjectile(screen.CreateBlessProjectile(target))
            case "cracks": model.AddProjectile(screen.CreateCracksCallProjectile(target))
            case "web": model.AddProjectile(screen.CreateWebProjectile(target))
            case "flamestrike":
                for _, unit := range defenders.GetUnits() {
                    model.AddProjectile(screen.CreateFlameStrikeProjectile(unit))
                }
            default:
                log.Printf("No spell named %v for a capture", name)
                return
        }

        screen.CastMessage(text)
    }
}

// keeps the player of a debug battle able to cast: mana and skill never run out
func debugMagic(player *playerlib.Player, army *combat.Army) {
    player.Mana = DebugBattleMana
    army.ManaPool = DebugBattleMana
    if DebugBattleManySpells {
        army.Casted = false
    }
}

// a cave held by monsters, or a city held by a garrison
func runRandomBattle(yield coroutine.YieldFunc, game *MagicGame, cityBattle bool) {
    allSpells, err := spellbook.ReadSpellsFromCache(game.Cache)
    if err != nil {
        log.Printf("Random battle: unable to read spells: %v", err)
        allSpells = spellbook.Spells{}
    }

    race := randomChoose(randomBattleRaces...)
    if capture.SameBattle {
        race = randomBattleRaces[0]
    }
    if capture.InTurn {
        combat.ActTogether = false
    }
    banner := randomChoose(data.BannerGreen, data.BannerBlue, data.BannerRed, data.BannerPurple, data.BannerYellow)

    attacker := playerlib.MakePlayer(setup.WizardCustom{Name: race.String(), Banner: banner, Race: race}, true, 0, 0, nil, &playerlib.NoGlobalEnchantments{})
    defender := playerlib.MakePlayer(setup.WizardCustom{Name: "Cave", Banner: data.BannerBrown}, false, 0, 0, nil, &playerlib.NoGlobalEnchantments{})

    armyScale := max(1, randomBattleArmyScale)
    if capture.ArmyScale > 0 {
        armyScale = capture.ArmyScale
    }

    attackingArmy := &combat.Army{Player: attacker}
    for range RandomBattleArmyCopies * armyScale {
        for _, unit := range startingUnits(race) {
            attackingArmy.AddUnit(units.MakeOverworldUnitFromUnit(unit, 1, 1, data.PlaneArcanus, attacker.Wizard.Banner, attacker.MakeExperienceInfo(), attacker.MakeUnitEnchantmentProvider()))
        }
    }

    defendingArmy := &combat.Army{Player: defender}
    landscape := randomChoose(randomBattleLandscapes...)
    if capture.SameBattle {
        landscape = randomBattleLandscapes[0]
    }
    zone := combat.ZoneType{Encounter: combat.ZoneLair}
    plane := data.PlaneArcanus

    if testBattleUnit != nil {
        // a test battle: one kind of unit against its own kind on open ground. see unitpicker.go
        unit := *testBattleUnit
        testBattleLast = testBattleUnit
        // kept for the next run of the game, see debugsaved.go
        saveDebugSaved()
        testBattleUnit = nil

        defenderBanner := data.BannerBlue
        if banner == defenderBanner {
            defenderBanner = data.BannerRed
        }
        attacker = playerlib.MakePlayer(setup.WizardCustom{Name: "Attackers", Banner: banner, Race: race}, true, 0, 0, nil, &playerlib.NoGlobalEnchantments{})
        defender = playerlib.MakePlayer(setup.WizardCustom{Name: "Defenders", Banner: defenderBanner, Race: race}, false, 0, 0, nil, &playerlib.NoGlobalEnchantments{})

        attackingArmy = &combat.Army{Player: attacker}
        defendingArmy = &combat.Army{Player: defender}
        for range TestBattleUnits * armyScale {
            attackingArmy.AddUnit(units.MakeOverworldUnitFromUnit(unit, 1, 1, data.PlaneArcanus, attacker.Wizard.Banner, attacker.MakeExperienceInfo(), attacker.MakeUnitEnchantmentProvider()))
            defendingArmy.AddUnit(units.MakeOverworldUnitFromUnit(unit, 1, 1, data.PlaneArcanus, defender.Wizard.Banner, defender.MakeExperienceInfo(), defender.MakeUnitEnchantmentProvider()))
        }

        landscape = combat.CombatLandscapeGrass
        zone = combat.ZoneType{}

        log.Printf("Test battle: %v %v against %v of their kind", TestBattleUnits * armyScale, unitFullName(&unit), TestBattleUnits * armyScale)
    } else if cityBattle {
        // a city of another race, held by its starting units
        cityRace := randomChoose(randomBattleRaces...)
        cityBanner := data.BannerBlue
        if banner == cityBanner {
            cityBanner = data.BannerRed
        }
        defender = playerlib.MakePlayer(setup.WizardCustom{Name: cityRace.String(), Banner: cityBanner, Race: cityRace}, false, 0, 0, nil, &playerlib.NoGlobalEnchantments{})
        defendingArmy = &combat.Army{Player: defender}
        for range armyScale {
            for _, unit := range startingUnits(cityRace) {
                defendingArmy.AddUnit(units.MakeOverworldUnitFromUnit(unit, 1, 1, data.PlaneArcanus, defender.Wizard.Banner, defender.MakeExperienceInfo(), defender.MakeUnitEnchantmentProvider()))
            }
        }

        choice := makeRandomCity()
        city := citylib.MakeCity("City", 10, 10, cityRace, nil, nil, nil, defender)
        city.Population = choice.Citizens * 1000
        city.Outpost = choice.Outpost
        if choice.Fortress {
            city.Buildings.Insert(buildinglib.BuildingFortress)
        }
        if choice.Walls {
            city.Buildings.Insert(buildinglib.BuildingCityWalls)
        }
        if choice.Fire {
            city.AddEnchantment(data.CityEnchantmentWallOfFire, cityBanner)
        }
        if choice.Darkness {
            city.AddEnchantment(data.CityEnchantmentWallOfDarkness, cityBanner)
        }
        if choice.Myrror {
            plane = data.PlaneMyrror
        }

        zone = combat.ZoneType{City: city}

        log.Printf("Random battle: %v army attacks a %v city, %+v, landscape %v", race, cityRace, choice, landscape)
    } else {
        monster := randomChoose(randomBattleMonsters...)
        count := (RandomBattleMinMonsters + rand.N(RandomBattleMaxMonsters - RandomBattleMinMonsters + 1)) * armyScale
        if capture.SameBattle {
            monster = randomBattleMonsters[0]
            count = RandomBattleMaxMonsters * armyScale
        }
        for range count {
            defendingArmy.AddUnit(units.MakeOverworldUnitFromUnit(monster, 1, 1, data.PlaneArcanus, defender.Wizard.Banner, defender.MakeExperienceInfo(), defender.MakeUnitEnchantmentProvider()))
        }

        log.Printf("Random battle: %v army attacks a cave with %v %v, landscape %v", race, count, monster.Name, landscape)
    }

    // on auto the army of the player fights without spells: they are for the player to try
    attackingArmy.NoSpellsOnAuto = true

    // for trying spells: the player knows every spell and does not run out of mana, see debugMagic
    attacker.KnownSpells.AddAllSpells(allSpells)
    attacker.Mana = DebugBattleMana
    attacker.CastingSkillPower = DebugBattleSkillPower

    events := make(chan combat.CombatEvent, 1000)
    model := combat.MakeCombatModel(allSpells, defendingArmy, attackingArmy, landscape, plane, zone, data.MagicNone, 0, 0, events)
    combatScreen := combat.MakeCombatScreen(game.Cache, defendingArmy, attackingArmy, optional.Of[combat.ArmyPlayer](attacker), landscape, plane, zone, model)
    if combatScreen == nil {
        log.Printf("Random battle: unable to make the combat screen")
        return
    }

    if capture.BattleLevel > 0 || capture.BattlePanX != 0 || capture.BattlePanY != 0 {
        // development: a fixed view for a capture
        level := capture.BattleLevel
        if level == 0 {
            level = int(scale.ScaleAmount)
        }
        combatScreen.Camera.SetView(level, float64(capture.BattlePanX), float64(capture.BattlePanY))
    }

    defer game.restoreNormalMouse()
    // the field fills the width of the window
    defer combatScreen.UseWideField()()

    game.Drawer = func(screen *ebiten.Image) {
        combatScreen.Draw(screen)
    }

    game.Music.PlaySong(randomChoose(musiclib.SongCombat1, musiclib.SongCombat2))

    if capture.Auto {
        // development: the army of the player fights by itself
        attackingArmy.Auto = true
    }

    if capture.Spell != "" {
        captureSpell(combatScreen, model, attackingArmy, defendingArmy)
    }

    if capture.KillFigures > 0 {
        // development: figures fall. by drawn frames, the battle does not come back here on every tick
        capture.EachFrame = func(frame int) {
            if frame == capture.KillFigures {
                for _, unit := range append(attackingArmy.GetUnits(), defendingArmy.GetUnits()...) {
                    unit.TakeDamage(unit.GetHealth() / 2, combat.DamageNormal)
                }
            }
        }
    }

    loggedTurn := -1
    state := combat.CombatStateRunning
    for state == combat.CombatStateRunning {
        debugMagic(attacker, attackingArmy)
        if capture.SameBattle && model.CurrentTurn != loggedTurn {
            // development: where everybody stands at the start of the turn
            loggedTurn = model.CurrentTurn
            places := ""
            for _, unit := range append(attackingArmy.GetUnits(), defendingArmy.GetUnits()...) {
                places += fmt.Sprintf(" %v,%v", unit.X, unit.Y)
            }
            log.Printf("places turn %v:%v", loggedTurn, places)
        }
        state = combatScreen.Update(yield)
        if capture.DamageNumbers && len(combatScreen.DamageIndicators) == 0 {
            // development: numbers of one, two and three digits over both armies
            for index, unit := range append(attackingArmy.GetUnits(), defendingArmy.GetUnits()...) {
                combatScreen.AddDamageIndicator(unit, []int{3, 12, 7, 108}[index % 4])
                if index == 0 {
                    // several on one unit
                    combatScreen.AddDamageIndicator(unit, 5)
                    combatScreen.AddDamageIndicator(unit, 24)
                }
            }
        }
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
