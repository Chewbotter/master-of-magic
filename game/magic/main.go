package main

import (
    "log"
    "fmt"
    "flag"
    "io"
    "errors"
    "math"
    "math/rand/v2"
    "os"
    "slices"
    "cmp"
    "bufio"
    "compress/gzip"
    "encoding/json"
    "image"
    "image/color"

    // for trace/pprof
    "net/http"
    _ "net/http/pprof"
    /*
    "runtime"
    "runtime/debug"
    "os"
    */

    "github.com/kazzmir/master-of-magic/lib/lbx"
    "github.com/kazzmir/master-of-magic/lib/font"
    "github.com/kazzmir/master-of-magic/lib/system"
    "github.com/kazzmir/master-of-magic/lib/coroutine"
    "github.com/kazzmir/master-of-magic/lib/fraction"
    introlib "github.com/kazzmir/master-of-magic/game/magic/intro"
    "github.com/kazzmir/master-of-magic/game/magic/audio"
    musiclib "github.com/kazzmir/master-of-magic/game/magic/music"
    settingslib "github.com/kazzmir/master-of-magic/game/magic/settings"
    "github.com/kazzmir/master-of-magic/game/magic/inputmanager"
    "github.com/kazzmir/master-of-magic/game/magic/setup"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    "github.com/kazzmir/master-of-magic/game/magic/mouse"
    "github.com/kazzmir/master-of-magic/game/magic/util"
    "github.com/kazzmir/master-of-magic/game/magic/ai"
    "github.com/kazzmir/master-of-magic/game/magic/load"
    "github.com/kazzmir/master-of-magic/game/magic/serialize"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    mouselib "github.com/kazzmir/master-of-magic/lib/mouse"
    "github.com/kazzmir/master-of-magic/game/magic/mainview"
    "github.com/kazzmir/master-of-magic/game/magic/mod"
    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/game/magic/camera"
    gamelib "github.com/kazzmir/master-of-magic/game/magic/game"
    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    buildinglib "github.com/kazzmir/master-of-magic/game/magic/building"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/ebitenutil"
    "github.com/hajimehoshi/ebiten/v2/inpututil"

    "gopkg.in/yaml.v3"
)

type DrawFunc func(*ebiten.Image)

type GameConfig struct {
    DataPath string `yaml:"datapath"`
    StartGame bool `yaml:"startgame"`
    Trace bool `yaml:"trace"`
    EnableMusic bool `yaml:"enablemusic"`
    LoadSave string `yaml:"loadsave"`
    WatchMode bool `yaml:"watchmode"`
    TrainMode bool `yaml:"trainmode"`
    AIMode string `yaml:"aimode"`
    Opponents int `yaml:"opponents"`
    Raiders bool `yaml:"raiders"`
}

func DefaultGameConfig() GameConfig {
    // set default values that are different from the zero values of the struct
    return GameConfig{
        EnableMusic: true,
        Opponents: 4,
        Raiders: true,
    }
}

type MagicGame struct {
    Cache *lbx.LbxCache

    MainCoroutine *coroutine.Coroutine
    Drawer DrawFunc

    Settings *settingslib.Settings
    Music *musiclib.Music

    Config GameConfig

    // screens without a widescreen layout are drawn here first
    Canvas *ebiten.Image
    // for the backdrop of a screen, as wide as the window
    WideCanvas *ebiten.Image

    // font for the frames per second counter, loaded once the data is available
    FPSFont *font.Font
}

func randomChoose[T any](choices... T) T {
    return choices[rand.N(len(choices))]
}

// true: play the animated intro before the main menu. false: go straight to the main menu
const PlayIntroOnStartup = false

func runIntro(yield coroutine.YieldFunc, game *MagicGame) {
    mouse.Mouse.Disable()
    defer mouse.Mouse.Enable()

    intro, err := introlib.MakeIntro(game.Cache, introlib.DefaultAnimationSpeed)
    if err != nil {
        log.Printf("Unable to run intro: %v", err)
        return
    }

    game.Drawer = func(screen *ebiten.Image) {
        intro.Draw(screen)
    }

    for intro.Update() == introlib.IntroStateRunning {
        if yield() != nil {
            return
        }

        if inputmanager.LeftClick() ||
           inpututil.IsKeyJustPressed(ebiten.KeySpace) ||
           inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
            return
        }
    }
}

func runNewGame(yield coroutine.YieldFunc, game *MagicGame) (bool, setup.NewGameSettings) {
    newGame := setup.MakeNewGameScreen(game.Cache)

    game.Drawer = func(screen *ebiten.Image) {
        newGame.Draw(screen)
    }

    state := newGame.Update()
    for state == setup.NewGameStateRunning {
        yield()
        state = newGame.Update()
    }

    return state == setup.NewGameStateCancel, newGame.Settings
}

func runNewWizard(yield coroutine.YieldFunc, game *MagicGame) (bool, setup.WizardCustom) {
    newWizard := setup.MakeNewWizardScreen(game.Cache)

    game.Drawer = func(screen *ebiten.Image) {
        newWizard.Draw(screen)
    }

    state := newWizard.Update()
    for state != setup.NewWizardScreenStateFinished && state != setup.NewWizardScreenStateCanceled {
        yield()
        state = newWizard.Update()
    }

    return state == setup.NewWizardScreenStateCanceled, newWizard.CustomWizard
}

func runMainMenu(yield coroutine.YieldFunc, game *MagicGame, gameLoader *OriginalGameLoader, music *musiclib.Music) (*gamelib.Game, mainview.MainScreenState) {
    menu := mainview.MakeMainScreen(game.Cache, gameLoader, music, game.Settings)

    game.Drawer = func(screen *ebiten.Image) {
        menu.Draw(screen)
    }

    // the debug fast-play options at the left edge, see fastplay.go
    fastPlayShowing = true
    defer func() {
        fastPlayShowing = false
    }()

    for {
        if state, clicked := game.updateFastPlay(menu); clicked {
            return nil, state
        }

        // the list of units of the test battle covers the start screen, see unitpicker.go
        if !unitPickerOpen {
            if menu.Update(yield) != mainview.MainScreenStateRunning {
                break
            }
        }

        select {
            case newGame := <-gameLoader.NewGame:
                return newGame, mainview.MainScreenStateLoadGame
            default:
        }

        if inputmanager.IsQuitPressed() {
            return nil, mainview.MainScreenStateQuit
        }

        if yield() != nil {
            return nil, mainview.MainScreenStateQuit
        }
    }

    return nil, menu.State
}

/* starting units are swordsmen and spearmen of the appropriate race
 */
func startingUnits(race data.Race) []units.Unit {
    switch race {
        case data.RaceLizard: return []units.Unit{units.LizardSwordsmen, units.LizardSpearmen}
        case data.RaceNomad: return []units.Unit{units.NomadSwordsmen, units.NomadSpearmen}
        case data.RaceOrc: return []units.Unit{units.OrcSwordsmen, units.OrcSpearmen}
        case data.RaceTroll: return []units.Unit{units.TrollSwordsmen, units.TrollSpearmen}
        case data.RaceBarbarian: return []units.Unit{units.BarbarianSwordsmen, units.BarbarianSpearmen}
        case data.RaceBeastmen: return []units.Unit{units.BeastmenSwordsmen, units.BeastmenSpearmen}
        case data.RaceDarkElf: return []units.Unit{units.DarkElfSwordsmen, units.DarkElfSpearmen}
        case data.RaceDraconian: return []units.Unit{units.DraconianSwordsmen, units.DraconianSpearmen}
        case data.RaceDwarf: return []units.Unit{units.DwarfSwordsmen, units.DwarfSwordsmen}
        case data.RaceGnoll: return []units.Unit{units.GnollSwordsmen, units.GnollSpearmen}
        case data.RaceHalfling: return []units.Unit{units.HalflingSwordsmen, units.HalflingSpearmen}
        case data.RaceHighElf: return []units.Unit{units.HighElfSwordsmen, units.HighElfSpearmen}
        case data.RaceHighMen: return []units.Unit{units.HighMenSwordsmen, units.HighMenSpearmen}
        case data.RaceKlackon: return []units.Unit{units.KlackonSwordsmen, units.KlackonSpearmen}
        default: return nil
    }
}

func euclideanDistance(x1, y1, x2, y2 int) float64 {
    dx := float64(x1 - x2)
    dy := float64(y1 - y2)

    return math.Sqrt(dx*dx + dy*dy)
}

func findCityLocation(game *gamelib.Game, startingPlane data.Plane, cityArea gamelib.CityValidArea) (int, int) {
    allCities := game.Model.AllCities()

    closestDistance := func(x, y int) int {
        distance := -1

        for _, city := range allCities {
            if city.Plane == startingPlane {
                d := int(euclideanDistance(x, y, city.X, city.Y))
                if distance == -1 || d < distance {
                    distance = d
                }
            }
        }

        if distance == -1 {
            return 0
        } else {
            return distance
        }
    }

    type CityLocation struct {
        X, Y int
        // distance to closest city
        Distance int
        Population int
    }

    var locations []CityLocation
    for range 10 {
        // x, y, ok := game.FindValidCityLocation(startingPlane)
        x, y, ok := cityArea.FindLocation()
        if ok {
            distance := closestDistance(x, y)
            // either there are no other cities nearby (distance=0) or the closest city is farther than 10 squares away
            if distance == 0 || distance > 10 {
                locations = append(locations, CityLocation{X: x, Y: y, Distance: distance, Population: game.Model.ComputeMaximumPopulation(x, y, startingPlane)})
            }
        }
    }

    // compute a weighted sum of distance to other cities and maximum population of the location
    computeValue := func (point CityLocation) float64 {
        distance := point.Distance
        // assume a distance if no other cities are nearby
        if distance == 0 {
            distance = 150
        }

        return math.Log2(float64(distance)) * 2 + float64(point.Population) * 0.5
    }

    slices.SortFunc(locations, func(pointA, pointB CityLocation) int {
        return cmp.Compare(computeValue(pointA), computeValue(pointB))
    })

    if len(locations) > 0 {
        // choose furthest point
        return locations[len(locations) - 1].X, locations[len(locations) - 1].Y
    } else {
        // couldn't find a good spot, just pick anything
        for range 100 {
            cityX, cityY, ok := cityArea.FindLocation()
            if ok {
                return cityX, cityY
            }
        }
    }

    return -1, -1
}

func initializePlayer(game *gamelib.Game, wizard setup.WizardCustom, isHuman bool, arcanusCityArea gamelib.CityValidArea, myrrorCityArea gamelib.CityValidArea) *playerlib.Player {
    area := arcanusCityArea
    startingPlane := data.PlaneArcanus
    if wizard.RetortEnabled(data.RetortMyrran) {
        startingPlane = data.PlaneMyrror
        area = myrrorCityArea
    }

    player := game.AddPlayer(wizard, isHuman)

    cityName := game.SuggestCityName(player.Wizard.Race)

    cityX, cityY := findCityLocation(game, startingPlane, area)
    area[image.Pt(cityX, cityY)] = false

    game.GetMap(startingPlane).SetRoad(cityX, cityY, startingPlane == data.PlaneMyrror)

    introCity := citylib.MakeCity(cityName, cityX, cityY, player.Wizard.Race, game.Model.BuildingInfo, game.GetMap(startingPlane), game.Model, player)
    introCity.Population = 4000
    introCity.Plane = startingPlane

    for _, building := range []buildinglib.Building{buildinglib.BuildingSmithy, buildinglib.BuildingBarracks, buildinglib.BuildingBuildersHall} {
        if introCity.GetBuildableBuildings().Contains(building) {
            introCity.Buildings.Insert(building)
        }
    }

    introCity.Buildings.Insert(buildinglib.BuildingFortress)
    introCity.Buildings.Insert(buildinglib.BuildingSummoningCircle)
    introCity.ProducingBuilding = buildinglib.BuildingHousing
    introCity.ProducingUnit = units.UnitNone
    introCity.Farmers = 4

    introCity.ResetCitizens()

    player.AddCity(introCity)

    for _, unit := range startingUnits(player.Wizard.Race) {
        player.AddUnit(units.MakeOverworldUnitFromUnit(unit, cityX, cityY, startingPlane, wizard.Banner, player.MakeExperienceInfo(), player.MakeUnitEnchantmentProvider()))
    }

    // for debugging purposes, to start the game super powered
    /*
    if isHuman {
        player.Admin = true
        player.Mana = 90000
        for range 3 {
            player.AddUnit(units.MakeOverworldUnitFromUnit(units.GreatDrake, cityX, cityY, startingPlane, wizard.Banner, player.MakeExperienceInfo(), player.MakeUnitEnchantmentProvider()))
        }
    }
    */

    player.LiftFog(cityX, cityY, 3, introCity.Plane)

    if isHuman {
        // a frame capture wants the plain world map, not the city naming prompt
        if capture.Path == "" || capture.CityPrompt {
            game.Events <- gamelib.StartingCityEvent(introCity)
        }
        game.Camera.Center(cityX, cityY)
        game.Model.Plane = startingPlane
    }

    return player
}

func initializeNeutralPlayer(game *gamelib.Game, arcanusCityArea gamelib.CityValidArea, myrrorCityArea gamelib.CityValidArea) *playerlib.Player {
    wizard := setup.WizardCustom{
        Name: "Raiders",
        Base: data.WizardMerlin, // doesn't really matter
        Race: data.RaceBarbarian, // doesn't really matter
        Banner: data.BannerBrown,
    }

    player := game.AddPlayer(wizard, false)
    player.AIBehavior = ai.MakeRaiderAI()
    player.TaxRate = fraction.Zero()

    for _, plane := range []data.Plane{data.PlaneArcanus, data.PlaneMyrror} {
        randomRace := func() data.Race {
            switch plane {
                case data.PlaneArcanus: return randomChoose(data.ArcanianRaces()...)
                case data.PlaneMyrror: return randomChoose(data.MyrranRaces()...)
            }

            return data.RaceNone
        }

        area := arcanusCityArea
        if plane == data.PlaneMyrror {
            area = myrrorCityArea
        }

        for range 5 {
            cityX, cityY := findCityLocation(game, plane, area)

            // should every neutral town be a random race, or should they all be related?
            race := randomRace()
            cityName := game.SuggestCityName(race)
            city := citylib.MakeCity(cityName, cityX, cityY, race, game.Model.BuildingInfo, game.GetMap(plane), game.Model, player)
            city.Population = rand.N(5) * 1000 + 2000
            city.ProducingBuilding = buildinglib.BuildingHousing
            city.Plane = plane
            city.Farmers = city.Citizens()
            city.ResetCitizens()

            area[image.Pt(cityX, cityY)] = false

            player.AddCity(city)
        }
    }

    return player
}

type OriginalGameLoader struct {
    Cache *lbx.LbxCache
    NewGame chan *gamelib.Game
    FS system.WriteableFS
    Music *musiclib.Music
    Settings *settingslib.Settings
}

func (loader *OriginalGameLoader) LoadMetadata(path string) (serialize.SaveMetadata, bool) {
    return serialize.LoadMetadata(loader.FS, path)
}

func (loader *OriginalGameLoader) LoadNew(path string) error {
    file, err := loader.FS.Open(path)
    if err != nil {
        return fmt.Errorf("Could not open save game file '%v': %v", path, err)
    }
    defer file.Close()

    err = loader.LoadNewReader(file)
    if err != nil {
        return fmt.Errorf("Could not load save game file '%v': %v", path, err)
    }

    return nil
}

func (loader *OriginalGameLoader) GetFS() system.WriteableFS {
    return loader.FS
}

// we assume the reader is still compressed
func (loader *OriginalGameLoader) LoadNewReader(readerOriginal io.Reader) error {
    reader := bufio.NewReader(readerOriginal)
    gzipReader, err := gzip.NewReader(reader)
    if err != nil {
        log.Printf("Error: unable to create gzip reader for save game: %v", err)
        return fmt.Errorf("Could not load")
    }
    defer gzipReader.Close()

    decoder := json.NewDecoder(gzipReader)

    var serializedGame gamelib.SerializedGame
    err = decoder.Decode(&serializedGame)
    if err != nil {
        log.Printf("Error: unable to decode save game: %v", err)
        return fmt.Errorf("Could not load")
    }

    newGame := gamelib.MakeGameFromSerialized(loader.Cache, loader.Music, loader.Settings, &serializedGame)
    select {
        case loader.NewGame <- newGame:
        default:
            log.Printf("Warning: unable to send new game to channel")
    }

    return nil
}

func (loader *OriginalGameLoader) Load(reader io.Reader) error {
    saved, err := load.LoadSaveGame(reader)
    if err != nil {
        return err
    }

    newGame := saved.Convert(loader.Cache, loader.Music, loader.Settings)

    if newGame != nil {
        select {
            case loader.NewGame <- newGame:
            default:
        }

        return nil
    }

    return fmt.Errorf("Could not convert saved game")
}

/*
func writeHeapDump(filename string) {
    runtime.GC()
    out, err := os.Create(filename)
    if err == nil {
        debug.WriteHeapDump(out.Fd())
        out.Close()
    }
}
*/

func centerOnCity(game *gamelib.Game) {
    humanPlayer := game.Model.GetHumanPlayer()
    if humanPlayer != nil {
        if len(humanPlayer.Cities) > 0 {
            for _, city := range humanPlayer.Cities {
                game.Camera.Center(city.X, city.Y)
                game.Model.Plane = city.Plane
            }
        }
    }
}

func runGameInstance(game *gamelib.Game, yield coroutine.YieldFunc, magic *MagicGame, gameLoader *OriginalGameLoader, maxTurns uint64) error {
    defer func() {
        // wrap the game variable so that only the remaining reference is shutdown
        game.Shutdown()
    }()
    game.GameLoader = gameLoader

    magic.Drawer = func(screen *ebiten.Image) {
        game.Draw(screen)
    }

    // the world map is the one screen that fills a wide picture by itself
    display.WideContent = func() bool {
        return game.IsWideOverland()
    }
    defer func() {
        display.WideContent = nil
    }()

    game.RefreshUI()

    if capture.Zoom > 0 {
        // development: capture the world map at a given zoom level
        game.Camera.Zoom = camera.ZoomForPixels(float64(capture.Zoom), scale.ScaleAmount)
        game.Camera.AnimatedZoom = capture.ZoomAnimation * float64(camera.ZoomMax) / scale.ScaleAmount
    }

    if capture.RevealAll {
        gamelib.DebugRevealAll = true
    }

    // these wait until the start of game events have settled, which rebuild the hud and center the camera
    if capture.TraceWide {
        capture.Describe = game.DescribeDrawers
        gamelib.CameraMoveTrace = func(x float64, y float64) {
            log.Printf("camera: %.3f, %.3f", x, y)
        }
    }

    if capture.PanX != 0 || capture.PanY != 0 || capture.DebugMenu || capture.Screen != "" || capture.CameraX >= 0 || capture.NextTurn || capture.CameraMove != "" || capture.Walk != "" {
      capture.Later = func() {
        if capture.NextTurn {
            select {
                case game.Events <- &gamelib.GameEventNextTurn{}:
                default:
            }
        }
        if capture.CameraX >= 0 {
            game.CaptureSetCameraX(capture.CameraX)
        }
        if capture.Walk != "" {
            // the walk starts a second later, once any pan has settled on screen
            capture.EachFrame = func(frame int) {
                if frame != CaptureWalkDelayFrames {
                    return
                }
                var walkX, walkY int
                fmt.Sscanf(capture.Walk, "%d,%d", &walkX, &walkY)
                gamelib.CameraMoveTrace = func(x float64, y float64) {
                    log.Printf("camera: %.3f, %.3f", x, y)
                }
                log.Printf("walk %v,%v from camera %.3f, %.3f: %v", walkX, walkY, game.Camera.GetOffsetX(), game.Camera.GetOffsetY(), game.CaptureWalk(walkX, walkY))
            }
        }
        if capture.CameraMove != "" {
            var moveX, moveY int
            fmt.Sscanf(capture.CameraMove, "%d,%d", &moveX, &moveY)
            startX, startY := game.Camera.GetOffsetX(), game.Camera.GetOffsetY()
            gamelib.CameraMoveTrace = func(x float64, y float64) {
                log.Printf("camera move: %.3f, %.3f", x, y)
            }
            log.Printf("camera move from %.3f, %.3f to tile %v, %v", startX, startY, game.Camera.GetX() + moveX, game.Camera.GetY() + moveY)
            select {
                case game.Events <- &gamelib.GameEventMoveCamera{Plane: game.Model.Plane, X: game.Camera.GetX() + moveX, Y: game.Camera.GetY() + moveY}:
                default:
            }
        }
        if capture.Screen == "nextunit" || capture.Screen == "research" || capture.Screen == "spellinfo" {
            // a second later, once any pan has settled on screen
            capture.EachFrame = func(frame int) {
                if frame == CaptureWalkDelayFrames {
                    game.CaptureOpenScreen(capture.Screen)
                }
            }
        } else if capture.Screen != "" && !game.CaptureOpenScreen(capture.Screen) {
            log.Printf("capture: unknown screen %v", capture.Screen)
        }
        if capture.PanX != 0 || capture.PanY != 0 {
            before := fmt.Sprintf("%.3f, %.3f", game.Camera.GetOffsetX(), game.Camera.GetOffsetY())
            game.PanBy(float64(capture.PanX), float64(capture.PanY))
            log.Printf("capture pan %v,%v: camera from %v to %.3f, %.3f", capture.PanX, capture.PanY, before, game.Camera.GetOffsetX(), game.Camera.GetOffsetY())
        }

        if capture.DebugMenu {
            game.HudUI.AddElements(game.DebugMenuForCapture())
        }
      }
    }

    if capture.CursorAt != "" {
        // development: draw the cursor at a fixed position through the between pixels path
        var cursorX, cursorY float64
        fmt.Sscanf(capture.CursorAt, "%f,%f", &cursorX, &cursorY)
        mouse.SmoothPosition = func(x int, y int) (float64, float64, bool) {
            return cursorX, cursorY, true
        }
    }

    if capture.Drag > 0 {
        gamelib.PanTrace = true
        capture.EachFrame = func(frame int) {
            if frame <= capture.Drag {
                game.SimulatePanStep(capture.DragSpeed, 0, true)
            } else if frame == capture.Drag + 1 {
                log.Printf("drag released")
                game.SimulatePanStep(0, 0, false)
            }
        }
    }

    if capture.NoSelection {
        // development: no unit selected, so the panel shows the income pictures
        humanPlayer := game.Model.GetHumanPlayer()
        if humanPlayer != nil {
            humanPlayer.SelectedStack = nil
        }
        game.RefreshUI()
    }

    if capture.Popup {
        // development: open the game menu so the capture shows a popup over the world map
        select {
            case game.Events <- &gamelib.GameEventGameMenu{}:
            default:
        }
    }

    /*
    runtime.AddCleanup(game, func(x int){
        log.Printf("Cleaned up game instance %v", x)
    }, 0)
    writeHeapDump("g1.dump")
    */

    for game.Update(yield) != gamelib.GameStateQuit {
        if inputmanager.IsQuitPressed() {
            return ebiten.Termination
        }

        if maxTurns > 0 && game.Model.TurnNumber >= maxTurns {
            log.Printf("Max turns reached: %v", maxTurns)
            return nil
        }

        select {
            case newGame := <-gameLoader.NewGame:
                game.Shutdown()
                game = newGame
                game.GameLoader = gameLoader
                game.Model.CurrentPlayer = 0

                centerOnCity(game)
                humanPlayer := game.Model.GetHumanPlayer()
                if humanPlayer != nil {
                    game.DoNextUnit(humanPlayer)
                }

                game.RefreshUI()

                magic.Drawer = func(screen *ebiten.Image) {
                    game.Draw(screen)
                }

                // writeHeapDump("g2.dump")
            default:
        }

        yield()
    }

    return nil
}

func initializeGame(magic *MagicGame, settings setup.NewGameSettings, humanWizard setup.WizardCustom) *gamelib.Game {
    game := gamelib.MakeGame(magic.Cache, magic.Music, magic.Settings, settings)

    switch magic.Config.AIMode {
        case "", "enemy2": game.Model.AIMode = ai.AIEnemy2
        case "net": game.Model.AIMode = ai.AINet
        default:
            log.Printf("Warning: unknown ai mode '%v', using default", magic.Config.AIMode)
    }

    game.RefreshUI()

    arcanusCityArea := game.MakeCityValidArea(data.PlaneArcanus)
    myrrorCityArea := game.MakeCityValidArea(data.PlaneMyrror)

    human := initializePlayer(game, humanWizard, true, arcanusCityArea, myrrorCityArea)

    for range settings.Opponents {
        wizard, ok := game.ChooseWizard()
        if ok {
            initializePlayer(game, wizard, false, arcanusCityArea, myrrorCityArea)
        } else {
            log.Printf("Warning: unable to add another wizard to the game")
        }
    }

    if magic.Config.Raiders {
        log.Printf("Create neutral player")
        neutral := initializeNeutralPlayer(game, arcanusCityArea, myrrorCityArea)
        log.Printf("done create neutral player with %v cities", len(neutral.Cities))
    }

    // hack
    // human.Admin = true
    game.Model.CurrentPlayer = 0
    game.StartPlayerTurn(human)

    // game.DoNextTurn()
    return game
}

func loadData(yield coroutine.YieldFunc, game *MagicGame, dataPath string) error {
    game.Drawer = func(screen *ebiten.Image) {
        ebitenutil.DebugPrintAt(screen, "Drag and drop a zip file that contains", 10, 10)
        ebitenutil.DebugPrintAt(screen, "the master of magic data files", 10, 30)
    }

    var cache *lbx.LbxCache
    if dataPath != "" {
        cache = lbx.CacheFromPath(dataPath)
        if cache == nil {
            return fmt.Errorf("Could not load data from '%v'", dataPath)
        }
        log.Printf("Loaded data from '%v'", dataPath)
    }

    for cache == nil {
        cache = lbx.AutoCache()
        if cache == nil {
            yield()
        }

        if inputmanager.IsQuitPressed() {
            return ebiten.Termination
        }
    }

    game.Cache = cache

    imageCache := util.MakeImageCache(cache)
    normalMouse, err := mouselib.GetMouseNormal(cache, &imageCache)
    if err == nil {
        mouse.Mouse.SetImage(normalMouse)
    }

    return nil
}

func startAITrainMode(yield coroutine.YieldFunc, game *MagicGame) error {
    game.Config.AIMode = "net"

    trainingRuns := 5

    for range trainingRuns {

        settings := setup.NewGameSettings{
            // Opponents: rand.N(4) + 1,
            Opponents: game.Config.Opponents,
            Difficulty: data.DifficultyAverage,
            Magic: data.MagicSettingNormal,
            LandSize: rand.N(3),
            DisableRaiders: !game.Config.Raiders,
        }

        spells, err := spellbook.ReadSpellsFromCache(game.Cache)
        if err != nil {
            return err
        }

        wizard, ok := gamelib.ChooseUniqueWizard(nil, spells)
        if !ok {
            return fmt.Errorf("Could not choose a wizard")
        }

        log.Printf("Starting game with settings=%+v wizard=%v race=%v", settings, wizard.Name, wizard.Race)

        realGame := initializeGame(game, settings, wizard)

        var netAI *ai.EnemyNetAI

        if len(realGame.Model.Players) >= 2 {
            aiPlayer := realGame.Model.Players[1]
            if aiPlayer.AIBehavior != nil {
                var ok bool
                netAI, ok = aiPlayer.AIBehavior.(*ai.EnemyNetAI)
                if !ok {
                    return fmt.Errorf("AI player is not a net AI, cannot train")
                }
            }
        }

        if netAI == nil {
            return fmt.Errorf("No AI player found, cannot train")
        }

        (func (){
            weights, err := os.Create("ai.json")
            if err == nil {
                defer weights.Close()
                err = netAI.LoadNeuralNet(weights)
                if err != nil {
                    log.Printf("Unable to load neural net: %v", err)
                }
            }
        })()

        realGame.SetWatchMode(500)

        human := realGame.Model.GetHumanPlayer()
        if human != nil {
            // make the human player an AI
            human.Admin = true
            human.Banished = true

            for _, city := range human.GetCities() {
                human.RemoveCity(city)
            }

            human.Stacks = nil
            human.SelectedStack = nil
            human.Skip = true

            // make sure all fog is visible
            human.UpdateFogVisibility()

            // consume initial events
            for range 10 {
                select {
                case <-realGame.Events:
                default:
                }
            }
        }

        // focus on the ai player
        realGame.FocusPlayerCity(1)

        // FIXME: we shouldn't need this
        gameLoader := &OriginalGameLoader{
            Cache: game.Cache,
            NewGame: make(chan *gamelib.Game, 1),
            FS: system.MakeFS(),
            Music: game.Music,
            Settings: game.Settings,
        }

        err = runGameInstance(realGame, yield, game, gameLoader, 800)
        if err != nil {
            return err
        }

        netAI.ApplyTraining()
        steps := netAI.Steps
        steps = steps[len(steps) - 10:]
        var rewards []float64
        for _, step := range steps {
            rewards = append(rewards, step.Reward)
        }
        log.Printf("AI last 10 step reward: %v", rewards)

        // log.Printf("Layer 0 weights: %v", netAI.NeuralNet.Layers[0])

        (func (){
            output, err := os.Create("ai.json")
            if err == nil {
                defer output.Close()
                err = netAI.SaveNeuralNet(output)
                if err != nil {
                    log.Printf("Unable to save neural net: %v", err)
                }
            } else {
                log.Printf("Unable to create ai.json: %v", err)
            }
        })()
    }

    return nil
}

// run a game with only AI players
func startWatchMode(yield coroutine.YieldFunc, game *MagicGame) error {
    settings := setup.NewGameSettings{
        // Opponents: rand.N(4) + 1,
        Opponents: game.Config.Opponents,
        Difficulty: data.DifficultyAverage,
        Magic: data.MagicSettingNormal,
        LandSize: rand.N(3),
        DisableRaiders: !game.Config.Raiders,
    }

    spells, err := spellbook.ReadSpellsFromCache(game.Cache)
    if err != nil {
        return err
    }

    wizard, ok := gamelib.ChooseUniqueWizard(nil, spells)
    if !ok {
        return fmt.Errorf("Could not choose a wizard")
    }

    log.Printf("Starting game with settings=%+v wizard=%v race=%v", settings, wizard.Name, wizard.Race)

    realGame := initializeGame(game, settings, wizard)

    realGame.SetWatchMode(100)

    human := realGame.Model.GetHumanPlayer()
    if human != nil {
        // make the human player an AI
        human.Admin = true
        human.Banished = true

        for _, city := range human.GetCities() {
            human.RemoveCity(city)
        }

        human.Stacks = nil
        human.SelectedStack = nil
        human.Skip = true

        // make sure all fog is visible
        human.UpdateFogVisibility()

        // consume initial events
        for range 10 {
            select {
                case <-realGame.Events:
                default:
            }
        }
    }

    // FIXME: we shouldn't need this
    gameLoader := &OriginalGameLoader{
        Cache: game.Cache,
        NewGame: make(chan *gamelib.Game, 1),
        FS: system.MakeFS(),
        Music: game.Music,
        Settings: game.Settings,
    }

    return runGameInstance(realGame, yield, game, gameLoader, 0)
}

func startQuickGame(yield coroutine.YieldFunc, game *MagicGame, gameLoader *OriginalGameLoader) error {
    settings := setup.NewGameSettings{
        Opponents: rand.N(4) + 1,
        Difficulty: data.DifficultyAverage,
        Magic: data.MagicSettingNormal,
        LandSize: rand.N(3),
    }

    spells, err := spellbook.ReadSpellsFromCache(game.Cache)
    if err != nil {
        return err
    }

    wizard, ok := gamelib.ChooseUniqueWizard(nil, spells)
    if !ok {
        return fmt.Errorf("Could not choose a wizard")
    }

    log.Printf("Starting game with settings=%+v wizard=%v race=%v", settings, wizard.Name, wizard.Race)

    realGame := initializeGame(game, settings, wizard)
    return runGameInstance(realGame, yield, game, gameLoader, 0)
}

func runGame(yield coroutine.YieldFunc, game *MagicGame, config GameConfig) error {

    err := loadData(yield, game, config.DataPath)
    if err != nil {
        return err
    }

    game.Music = musiclib.MakeMusic(game.Cache)

    game.Settings = settingslib.MakeSettings(game.Cache)
    game.Music.Enabled = config.EnableMusic
    defer game.Music.Stop()

    shutdown := func (screen *ebiten.Image){
        ebitenutil.DebugPrintAt(screen, "Shutting down", 10, 10)
    }

    if config.WatchMode {
        return startWatchMode(yield, game)
    }

    if config.TrainMode {
        return startAITrainMode(yield, game)
    }

    gameLoader := &OriginalGameLoader{
        Cache: game.Cache,
        NewGame: make(chan *gamelib.Game, 1),
        FS: system.MakeFS(),
        Music: game.Music,
        Settings: game.Settings,
    }

    // start a game immediately
    if config.StartGame {
        return startQuickGame(yield, game, gameLoader)
    }

    if config.LoadSave == "" {
        if PlayIntroOnStartup {
            game.Music.PlaySong(musiclib.SongIntro)
            runIntro(yield, game)

            yield()
        }

        game.Music.PlaySong(musiclib.SongTitle)
    }

    if config.LoadSave != "" {
        err := gameLoader.LoadNew(config.LoadSave)
        // couldn't load game, just play title music
        if err != nil {
            game.Music.PlaySong(musiclib.SongTitle)
        }
    }

    if capture.MouseAt != "" {
        // development: the game sees the mouse at a fixed position
        fmt.Sscanf(capture.MouseAt, "%d,%d", &inputmanager.FixedMouseX, &inputmanager.FixedMouseY)
        inputmanager.FixedMouse = true
        // the interface reads the mouse by itself
        display.FixedCursor = true
        display.FixedCursorX = inputmanager.FixedMouseX
        display.FixedCursorY = inputmanager.FixedMouseY
    }

    if capture.RandomBattle {
        // development: straight into a random battle
        runRandomBattle(yield, game, false)
    }

    if capture.CityBattle != "" {
        runRandomBattle(yield, game, true)
    }

    if capture.UnitBattle != "" {
        // development: straight into a test battle of a unit
        testBattleUnit = findUnit(capture.UnitBattle)
        if testBattleUnit == nil {
            log.Printf("No unit named %v", capture.UnitBattle)
        } else {
            runRandomBattle(yield, game, false)
        }
    }

    for {
        newGame, state := runMainMenu(yield, game, gameLoader, game.Music)
        switch state {
            case mainview.MainScreenStateQuit:
                game.Drawer = shutdown
                yield()
                return ebiten.Termination
            case mainview.MainScreenStateLoadGame:
                if newGame != nil {
                    game.Music.Stop()
                    // FIXME: should this go here?
                    newGame.Model.CurrentPlayer = 0
                    centerOnCity(newGame)

                    humanPlayer := newGame.Model.GetHumanPlayer()
                    if humanPlayer != nil {
                        newGame.DoNextUnit(humanPlayer)
                    }

                    err := runGameInstance(newGame, yield, game, gameLoader, 0)
                    if err != nil {
                        game.Drawer = shutdown
                        yield()
                        return err
                    }

                    game.Music.PlaySong(musiclib.SongTitle)
                }
            case mainview.MainScreenStateRandomBattle, mainview.MainScreenStateRandomCityBattle, mainview.MainScreenStateTestBattle:
                game.Music.Stop()
                yield()
                runRandomBattle(yield, game, state == mainview.MainScreenStateRandomCityBattle)
                game.Music.PlaySong(musiclib.SongTitle)
            case mainview.MainScreenStateDebugWorldMap:
                // a quick start game for testing the world map, with the debug options that help there
                game.Music.Stop()
                yield()
                gamelib.DebugUnlimitedMoves = true
                gamelib.DebugNoGreetings = true
                err := startQuickGame(yield, game, gameLoader)
                gamelib.DebugUnlimitedMoves = false
                gamelib.DebugNoGreetings = false
                if err != nil {
                    game.Drawer = shutdown
                    yield()
                    return err
                }
                game.Music.PlaySong(musiclib.SongTitle)
            case mainview.MainScreenStateQuickGame:
                game.Music.Stop()
                yield()
                err := startQuickGame(yield, game, gameLoader)
                if err != nil {
                    game.Drawer = shutdown
                    yield()
                    return err
                }
                game.Music.PlaySong(musiclib.SongTitle)
            case mainview.MainScreenStateNewGame:
                var settings setup.NewGameSettings
                var wizard setup.WizardCustom
                restart := true
                cancel := false
                for restart && !cancel {
                    // yield so that clicks from the menu don't bleed into the next part
                    yield()
                    cancel, settings = runNewGame(yield, game)
                    if cancel {
                        break
                    }
                    yield()
                    restart, wizard = runNewWizard(yield, game)
                }
                yield()
                if cancel {
                    break
                }

                game.Music.Stop()

                realGame := initializeGame(game, settings, wizard)
                err := runGameInstance(realGame, yield, game, gameLoader, 0)

                if err != nil {
                    game.Drawer = shutdown
                    yield()
                    return err
                }

                game.Music.PlaySong(musiclib.SongTitle)
        }
    }
}

func NewMagicGame(config GameConfig) (*MagicGame, error) {
    var game *MagicGame

    run := func(yield coroutine.YieldFunc) error {
        return runGame(yield, game, config)
    }

    game = &MagicGame{
        MainCoroutine: coroutine.MakeCoroutine(run),
        Drawer: nil,
        Config: config,
    }

    return game, nil
}

func (game *MagicGame) Update() error {
    display.FinishStartup(capture.Corner || capture.Path != "")
    inputmanager.Update()

    if ebiten.IsWindowBeingClosed() {
        game.MainCoroutine.Stop()
    }

    err := game.MainCoroutine.Run()
    if err != nil {
        if errors.Is(err, coroutine.CoroutineFinished) || errors.Is(err, coroutine.CoroutineCancelled) {
            return ebiten.Termination
        }

        return err
    }

    return nil
}

func (game *MagicGame) Layout(outsideWidth int, outsideHeight int) (int, int) {
    // draw at the window's real pixel size, see display/drawscale.go
    return display.Layout(outsideWidth, outsideHeight)
}

func (game *MagicGame) Draw(screen *ebiten.Image) {
    // screen.Fill(color.RGBA{0x80, 0xa0, 0xc0, 0xff})

    // solid black underneath everything, this is what the bars are made of
    screen.Fill(color.RGBA{R: 0, G: 0, B: 0, A: 0xff})

    if game.Drawer != nil {
        offsetX := display.ContentOffsetX()
        offsetY := display.ContentOffsetY()
        if offsetX > 0 || offsetY > 0 {
            // the picture is drawn on its own and placed in the window: in the middle with black bars when
            // the screen has no widescreen layout, and inside the margins the native draw scale leaves
            pictureWidth := data.ScreenWidth
            if display.IsWideContentActive() {
                pictureWidth = display.LogicalWidth()
            }
            width, height := scale.Scale2(pictureWidth, data.ScreenHeight)
            if game.Canvas == nil || game.Canvas.Bounds().Dx() != width || game.Canvas.Bounds().Dy() != height {
                game.Canvas = ebiten.NewImage(width, height)
            }
            game.Canvas.Clear()

            if display.IsWideBackdropActive() {
                // the screen fills the bars with a backdrop of its own, see display/backdrop.go
                wideWidth, wideHeight := scale.Scale2(display.LogicalWidth(), data.ScreenHeight)
                if game.WideCanvas == nil || game.WideCanvas.Bounds().Dx() != wideWidth || game.WideCanvas.Bounds().Dy() != wideHeight {
                    game.WideCanvas = ebiten.NewImage(wideWidth, wideHeight)
                }
                game.WideCanvas.Clear()
                display.WideBackdrop(game.WideCanvas)

                var wideOptions ebiten.DrawImageOptions
                wideOptions.GeoM.Translate(float64(display.PictureX()), float64(offsetY))
                screen.DrawImage(game.WideCanvas, &wideOptions)
            }

            game.Drawer(game.Canvas)

            var options ebiten.DrawImageOptions
            options.GeoM.Translate(float64(offsetX), float64(offsetY))
            screen.DrawImage(game.Canvas, &options)
        } else {
            game.Drawer(screen)
        }
    }

    game.drawFastPlay(screen)
    game.drawFPS(screen)

    mouse.Mouse.Draw(screen)

    if capture.Update(screen) {
        game.MainCoroutine.Stop()
    }
}

func loadGameConfigFromFile(path string) GameConfig {
    // try to load as yaml file, if not then try to load as ini file (x=y format)

    out := DefaultGameConfig()

    file, err := os.Open(path)
    if err != nil {
        log.Printf("Warning: unable to open config file '%v': %v", path, err)
        return out
    }

    defer file.Close()

    decoder := yaml.NewDecoder(file)

    err = decoder.Decode(&out)
    if err != nil {
        log.Printf("Warning: unable to decode config file '%v': %v", path, err)
        return out
    }

    return out
}

// load cli arguments and return a GameConfig struct
// if -config "path" is given then load the config from the given file
// but cli arguments override the config file settings
func loadGameConfig() GameConfig {
    var dataPath string
    var startGame bool
    var trace bool
    var enableMusic bool
    var loadSave string
    var watchMode bool
    var trainMode bool
    var aiMode string
    var config string

    flag.StringVar(&modPath, "mod", "", "folder of pictures that take the place of the game's own. The folders mod and ../mod are looked at if not given.")
    flag.StringVar(&dataPath, "data", "", "path to master of magic lbx data files. Give either a directory or a zip file. Data is searched for in the current directory if not given.")
    flag.BoolVar(&enableMusic, "music", true, "enable music playback")
    flag.BoolVar(&startGame, "start", false, "start the game immediately with a random wizard")
    flag.BoolVar(&trace, "trace", false, "enable profiling (pprof)")
    flag.StringVar(&loadSave, "load", "", "load a saved game from the given file and start immediately")
    flag.BoolVar(&watchMode, "watch", false, "run in watch mode, where you can watch the AI play against itself (no human players)")
    flag.BoolVar(&trainMode, "train", false, "train the AI in watch mode. model weights are saved as ai.json")
    flag.StringVar(&aiMode, "ai", "", "select ai mode. 'default', 'enemy2', 'net'")
    flag.StringVar(&config, "config", "", "path to config file (yaml or ini)")
    flag.StringVar(&capture.Path, "capture", "", "development: write one frame to this png file and exit")
    flag.IntVar(&capture.Frames, "capture-frames", 120, "development: frames to draw before the capture")
    flag.BoolVar(&capture.RevealAll, "capture-reveal-all", false, "development: turn on the Reveal All debug option")
    flag.IntVar(&capture.KillFigures, "capture-kill-figures", 0, "development: the frame of a random battle at which every unit loses about half of its figures")
    flag.IntVar(&capture.ArmyScale, "capture-army-scale", 0, "development: multiplies the units of both armies of a random battle")
    flag.StringVar(&capture.Spell, "capture-spell", "", "development: show a spell in a debug battle: firebolt, fireball, icebolt, doombolt, lightning, warplightning, confusion, bless, flamestrike, cracks, web, as name@frame")
    flag.StringVar(&capture.UnitBattle, "capture-unit-battle", "", "development: start a test battle of this unit, by its name with or without its race")
    flag.StringVar(&capture.UnitPicker, "capture-unit-picker", "", "development: open the list of units of the test battle with these races open, or all")
    flag.BoolVar(&capture.InTurn, "capture-in-turn", false, "development: units act in turn, not together")
    flag.BoolVar(&capture.SameBattle, "capture-same-battle", false, "development: a random battle with the same armies every time, and the places of all units in the log every turn")
    flag.IntVar(&capture.AutoToggle, "capture-auto-toggle", 0, "development: the frame of a random battle at which auto is toggled as by its key, counted from 100 frames before the capture")
    flag.IntVar(&capture.MoveArea, "capture-move-area", -1, "development: the look of the move area of a battle: 0 off, 1 fill, 2 edge, 3 fill and edge, 4 dots")
    flag.BoolVar(&capture.UnitOrder, "capture-unit-order", false, "development: logs the order WAIT goes through the units of the player of a random battle in, and picks a unit as a click does")
    flag.BoolVar(&capture.Tab, "capture-tab", false, "development: a battle is shown as if Tab was held")
    flag.IntVar(&spellbook.CaptureFlip, "capture-flip", -1, "development: the spellbook for casting shows the turn of its first page, held at this picture of the turn")
    flag.IntVar(&spellbook.CaptureFlipPage, "capture-flip-page", 0, "development: which turn of a page -capture-flip shows, 0 the first")
    flag.BoolVar(&spellbook.CaptureFlipBack, "capture-flip-back", false, "development: the turn -capture-flip holds is one back")
    flag.IntVar(&capture.Leave, "capture-leave", 0, "development: the frame at which a debug battle is left as by Escape, counted from 250 frames before the capture")
    flag.BoolVar(&capture.Auto, "capture-auto", false, "development: the army of the player of a random battle is set to auto")
    flag.BoolVar(&capture.DamageNumbers, "capture-damage-numbers", false, "development: keep damage numbers over the units of a random battle")
    flag.StringVar(&capture.MouseAt, "capture-mouse-at", "", "development: the game sees the mouse at x,y screen pixels of the picture")
    flag.StringVar(&capture.CursorAt, "capture-cursor-at", "", "development: draw the cursor at x,y screen pixels, fractions allowed")
    flag.BoolVar(&capture.TraceWide, "capture-trace-wide", false, "development: log every change of the widescreen layout and the layers on screen")
    flag.Float64Var(&gamelib.FogFadeProbe, "capture-fog-fade", -1, "development: hold every fade of the fog this many seconds after its start, the map of the start counts as just revealed")
    flag.BoolVar(&gamelib.FogFadeTrace, "capture-fog-trace", false, "development: log the number of tiles whose fog fades, every frame")
    flag.BoolVar(&gamelib.DebugUnlimitedMoves, "capture-unlimited-moves", false, "development: turn on the Unlimited Moves debug option")
    flag.StringVar(&capture.Walk, "capture-walk", "", "development: send the selected unit walking dx,dy tiles and log the camera")
    flag.StringVar(&capture.CameraMove, "capture-camera-move", "", "development: move the camera by dx,dy tiles the way a right click does, and log each frame")
    flag.IntVar(&capture.BattleLevel, "capture-battle-level", 0, "development: zoom level of the battlefield, in screen pixels per art pixel")
    flag.IntVar(&capture.BattlePanX, "capture-battle-pan-x", 0, "development: move the battlefield right by this many screen pixels")
    flag.IntVar(&capture.BattlePanY, "capture-battle-pan-y", 0, "development: move the battlefield down by this many screen pixels")
    flag.BoolVar(&capture.RandomBattle, "capture-random-battle", false, "development: start a random battle instead of the start screen")
    flag.StringVar(&capture.CityBattle, "capture-city-battle", "", "development: start a random battle for a city. a list of: walls, fortress, fire, darkness, outpost, myrror, size=N, or random")
    flag.StringVar(&capture.BattleGround, "capture-battle-ground", "", "development: the ground of a random battle. a list of: grass, desert, mountain, tundra, forest, hills, roads, road=N, enchanted")
    flag.BoolVar(&capture.NextTurn, "capture-next-turn", false, "development: press Next Turn, and trace the widescreen layout while the turn runs")
    flag.Float64Var(&capture.CameraX, "capture-camera-x", -1, "development: put the camera at this column before the capture, fractions allowed")
    flag.IntVar(&capture.DragSpeed, "capture-drag-speed", 1, "development: screen pixels the simulated drag moves each frame")
    flag.IntVar(&capture.Drag, "capture-drag", 0, "development: simulate a slow drag of this many frames and log the view")
    flag.BoolVar(&capture.DebugMenu, "capture-debug-menu", false, "development: open the debug menu")
    flag.IntVar(&capture.PanX, "capture-pan-x", 0, "development: pan the map this many screen pixels to the right before the capture")
    flag.IntVar(&capture.PanY, "capture-pan-y", 0, "development: pan the map this many screen pixels down before the capture")
    flag.BoolVar(&capture.CityPrompt, "capture-city-prompt", false, "development: keep the starting city name prompt in the capture")
    flag.BoolVar(&capture.NoSelection, "capture-no-selection", false, "development: no unit selected in the capture")
    flag.BoolVar(&capture.Popup, "capture-popup", false, "development: open the game menu over the world map before the capture")
    flag.IntVar(&capture.Zoom, "capture-zoom", 0, "development: world map zoom for the capture, in screen pixels per art pixel")
    flag.IntVar(&display.CornerWidthOverride, "capture-window-width", 0, "development: width of the corner window in pixels, its height is 200")
    flag.Float64Var(&capture.ZoomAnimation, "capture-zoom-animation", 0, "development: freeze the zoom animation this many screen pixels per art pixel away from the zoom, -1 to 1")
    flag.BoolVar(&capture.Corner, "corner", false, "development: small window in the lower right corner of the screen. implied by -capture")
    var nearest bool
    flag.BoolVar(&nearest, "capture-nearest", false, "development: draw the world map zoom the original way")
    flag.Float64Var(&display.DrawScaleOverride, "scale", 0, "development: screen pixels per art pixel the game draws at. development runs default to the saved window scale")
    flag.StringVar(&capture.Screen, "capture-screen", "", "development: open a screen before the capture: armies, cities, magic, spellbook, city, surveyor, cartographer, advisors")
    flag.Parse()

    out := DefaultGameConfig()

    if nearest {
        gamelib.OverworldCleanZoom = false
    }

    if config != "" {
        out = loadGameConfigFromFile(config)
    }

    flag.Visit(func (f *flag.Flag) {
        switch {
            case f.Name == "data" && dataPath != "": out.DataPath = dataPath
            case f.Name == "music": out.EnableMusic = enableMusic
            case f.Name == "start": out.StartGame = startGame
            case f.Name == "trace": out.Trace = trace
            case f.Name == "load": out.LoadSave = loadSave
            case f.Name == "watch": out.WatchMode = watchMode
            case f.Name == "ai" && aiMode != "": out.AIMode = aiMode
            case f.Name == "train": out.TrainMode = trainMode
        }
    })

    return out
}

// the replacement folder that was asked for, see game/magic/mod
var modPath string

func setupReplacementPictures() {
    if mod.Setup(modPath) == "" {
        return
    }

    for _, unit := range units.AllUnits {
        if unit.CombatLbxFile != "" {
            mod.RegisterUnit(fmt.Sprintf("%v %v", unit.Race, unit.Name), unit.CombatLbxFile, unit.CombatIndex)
        }
    }

    util.ReplacePictures = mod.Replace
    mouselib.ReplaceCursor = mod.ReplaceCursor
}

func main() {
    log.SetFlags(log.Ldate | log.Lshortfile | log.Lmicroseconds)

    config := loadGameConfig()
    setupReplacementPictures()

    if config.Trace {
        go func() {
            log.Printf("Starting pprof server on localhost:8000")
            log.Println(http.ListenAndServe("localhost:8000", nil))
        }()
    }

    developmentRun := capture.Corner || capture.Path != ""
    if developmentRun {
        display.SettingsFile = display.DevelopmentSettingsFile
    }

    display.Load().Apply()
    if developmentRun {
        // development runs stay out of the way. a normal launch is centered
        display.PlaceInCorner()
        // the small corner window would draw at scale 1. draw as the saved window would, so captures show the real thing
        if display.NativeDrawScale && display.DrawScaleOverride == 0 {
            display.DrawScaleOverride = float64(display.Current.WindowScale)
        }
    }
    ebiten.SetWindowTitle("magic")
    ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
    ebiten.SetWindowClosingHandled(true)

    audio.Initialize()
    mouse.Initialize()

    ebiten.SetCursorMode(ebiten.CursorModeHidden)

    game, err := NewMagicGame(config)

    if err != nil {
        log.Printf("Error: unable to load game: %v", err)
        return
    }

    err = ebiten.RunGame(game)
    if err != nil {
        log.Printf("Error: %v", err)
    }

    log.Printf("Bye")
}
