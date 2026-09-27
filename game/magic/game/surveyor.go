package game

import (
    "fmt"
    "image"

    "github.com/kazzmir/master-of-magic/lib/coroutine"
    // "github.com/kazzmir/master-of-magic/lib/lbx"
    "github.com/kazzmir/master-of-magic/lib/font"
    "github.com/kazzmir/master-of-magic/game/magic/util"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/game/magic/inputmanager"
    fontslib "github.com/kazzmir/master-of-magic/game/magic/fonts"
    uilib "github.com/kazzmir/master-of-magic/game/magic/ui"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    "github.com/kazzmir/master-of-magic/game/magic/maplib"

    "github.com/hajimehoshi/ebiten/v2"
)

func (game *Game) doSurveyor(yield coroutine.YieldFunc) {
    fonts := fontslib.MakeSurveyorFonts(game.Cache)

    var cityMap map[image.Point]*citylib.City

    makeOverworld := func () Overworld {
        cityMap = make(map[image.Point]*citylib.City)

        var cities []*citylib.City
        var stacks []*playerlib.UnitStack
        var fog data.FogMap

        for i, player := range game.Model.Players {
            for _, city := range player.Cities {
                if city.Plane == game.Model.Plane {
                    cities = append(cities, city)
                    cityMap[image.Pt(city.X, city.Y)] = city
                }
            }

            for _, stack := range player.Stacks {
                if stack.Plane() == game.Model.Plane {
                    stacks = append(stacks, stack)
                }
            }

            if i == 0 {
                fog = player.GetFog(game.Model.Plane)
            }
        }

        return Overworld{
            Camera: game.Camera,
            Counter: game.Counter,
            Map: game.Model.CurrentMap(),
            Cities: cities,
            Stacks: stacks,
            SelectedStack: nil,
            ImageCache: &game.ImageCache,
            Fog: fog,
            ShowAnimation: game.State == GameStateUnitMoving,
            FogBlack: game.GetFogImage(),
        }
    }

    overworld := makeOverworld()

    selectedPoint := image.Pt(-1, -1)

    var cityInfoText font.WrappedText
    type Resources struct {
        Enabled bool
        MaximumPopulation int
        ProductionBonus int
        GoldBonus int
    }

    var resources Resources

    cancelBackground, _ := game.ImageCache.GetImage("main.lbx", 47, 0)

    quit := false

    ui := &uilib.UI{
        Cache: game.Cache,
        HandleKeys: func(keys []ebiten.Key) {
            for _, key := range keys {
                switch key {
                    case ebiten.KeyF1:
                        quit = true
                }
            }
        },
        Draw: func(ui *uilib.UI, screen *ebiten.Image){
            var options ebiten.DrawImageOptions
            mainHud, _ := game.ImageCache.GetImage("main.lbx", 0, 0)
            scale.DrawScaled(screen, mainHud, &options)

            landImage, _ := game.ImageCache.GetImage("main.lbx", 57, 0)
            options.GeoM.Translate(float64(240), float64(77))
            scale.DrawScaled(screen, landImage, &options)

            options.GeoM.Reset()
            options.GeoM.Translate(float64(240), float64(174))
            scale.DrawScaled(screen, cancelBackground, &options)

            ui.StandardDraw(screen)

            player := game.Model.GetHumanPlayer()

            // outlined like the world map panel, see panel.go
            game.drawPanelReserves(screen)

            fonts.SurveyorFont.PrintCenter(screen, float64(280), float64(81), scale.ScaleAmount, ebiten.ColorScale{}, "Surveyor")

            // everything below the title is Compact text, see lib/font/textsize.go
            defer font.UseTextSize(font.TextSizeCompact)()

            if selectedPoint.X >= 0 && selectedPoint.X < game.Model.CurrentMap().Width() && selectedPoint.Y >= 0 && selectedPoint.Y < game.Model.CurrentMap().Height() {
                if overworld.Fog[selectedPoint.X][selectedPoint.Y] != data.FogTypeUnexplored {
                    mapObject := game.Model.CurrentMap()
                    tile := mapObject.GetTile(selectedPoint.X, selectedPoint.Y)
                    node := mapObject.GetMagicNode(selectedPoint.X, selectedPoint.Y)

                    y := float64(93)

                    // Terrain
                    name := tile.Name(mapObject)
                    if node != nil {
                        switch node.Kind {
                            case maplib.MagicNodeNature: name = "Forest"
                            case maplib.MagicNodeSorcery: name = "Grasslands"
                            case maplib.MagicNodeChaos: name = "Mountain"
                        }
                    }
                    fonts.YellowFont.PrintCenter(screen, 280, y, scale.ScaleAmount, ebiten.ColorScale{}, name)
                    y += fonts.YellowFont.HeightAt(scale.ScaleAmount)

                    // Terrain bonuses
                    if tile.Corrupted() {
                        fonts.WhiteFont.PrintCenter(screen, float64(280), y, scale.ScaleAmount, ebiten.ColorScale{}, "Corruption")
                        y += fonts.WhiteFont.HeightAt(scale.ScaleAmount)
                    }

                    foodBonus := tile.FoodBonus()
                    if !foodBonus.IsZero() {
                        fonts.WhiteFont.PrintCenter(screen, float64(280), y, scale.ScaleAmount, ebiten.ColorScale{}, fmt.Sprintf("%v food", foodBonus.NormalString()))
                        y += fonts.WhiteFont.HeightAt(scale.ScaleAmount)
                    }

                    productionBonus := tile.ProductionBonus(false)
                    if productionBonus != 0 {
                        fonts.WhiteFont.PrintCenter(screen, float64(280), y, scale.ScaleAmount, ebiten.ColorScale{}, fmt.Sprintf("+%v%% production", productionBonus))
                        y += fonts.WhiteFont.HeightAt(scale.ScaleAmount)
                    }

                    goldBonus := tile.GoldBonus(mapObject)
                    if goldBonus != 0 {
                        fonts.WhiteFont.PrintCenter(screen, float64(280), y, scale.ScaleAmount, ebiten.ColorScale{}, fmt.Sprintf("+%v%% gold", goldBonus))
                        y += fonts.WhiteFont.HeightAt(scale.ScaleAmount)
                    }

                    y += fonts.WhiteFont.HeightAt(scale.ScaleAmount)

                    // Bonuses
                    bonus := tile.GetBonus()
                    if bonus != data.BonusNone {
                        fonts.YellowFont.PrintCenter(screen, float64(280), y, scale.ScaleAmount, ebiten.ColorScale{}, bonus.String())
                        y += fonts.YellowFont.HeightAt(scale.ScaleAmount)

                        food := bonus.FoodBonus()
                        if food != 0 {
                            fonts.WhiteFont.PrintWrapCenter(screen, float64(280), y, float64(cancelBackground.Bounds().Dx() - 5), scale.ScaleAmount, ebiten.ColorScale{}, fmt.Sprintf("+%v food", food))
                            y += fonts.WhiteFont.HeightAt(scale.ScaleAmount)
                        }

                        gold := bonus.GoldBonus()
                        if gold != 0 {
                            fonts.WhiteFont.PrintWrapCenter(screen, float64(280), y, float64(cancelBackground.Bounds().Dx() - 5), scale.ScaleAmount, ebiten.ColorScale{}, fmt.Sprintf("+%v gold", gold))
                            y += fonts.WhiteFont.HeightAt(scale.ScaleAmount)
                        }

                        power := bonus.PowerBonus()
                        if power != 0 {
                            fonts.WhiteFont.PrintWrapCenter(screen, float64(280), y, float64(cancelBackground.Bounds().Dx() - 5), scale.ScaleAmount, ebiten.ColorScale{}, fmt.Sprintf("+%v power", power))
                            y += fonts.WhiteFont.HeightAt(scale.ScaleAmount)
                        }

                        reduction := bonus.UnitReductionBonus()
                        if reduction != 0 {
                            fonts.WhiteFont.PrintWrapCenter(screen, float64(280), y, float64(cancelBackground.Bounds().Dx() - 5), scale.ScaleAmount, ebiten.ColorScale{}, fmt.Sprintf("Reduces normal unit cost by %v%%", reduction))
                            y += fonts.WhiteFont.HeightAt(scale.ScaleAmount)
                        }
                    }

                    // Nodes
                    if node != nil {
                        fonts.YellowFont.PrintWrapCenter(screen, float64(280), y, float64(cancelBackground.Bounds().Dx() - 5), scale.ScaleAmount, ebiten.ColorScale{}, node.Kind.Name())
                        y += fonts.YellowFont.HeightAt(scale.ScaleAmount)

                        if node.Warped {
                            fonts.WhiteFont.PrintCenter(screen, float64(280), y, scale.ScaleAmount, ebiten.ColorScale{}, "Warped")
                            y += fonts.WhiteFont.HeightAt(scale.ScaleAmount)
                        } else if node.MeldingWizard != nil && node.GuardianSpiritMeld {
                            fonts.WhiteFont.PrintCenter(screen, float64(280), y, scale.ScaleAmount, ebiten.ColorScale{}, "Guardian Spirit")
                            y += fonts.WhiteFont.HeightAt(scale.ScaleAmount)
                        } else if node.MeldingWizard != nil {
                            fonts.WhiteFont.PrintCenter(screen, float64(280), y, scale.ScaleAmount, ebiten.ColorScale{}, "Magic Spirit")
                            y += fonts.WhiteFont.HeightAt(scale.ScaleAmount)
                        }
                    }

                    // Lairs
                    encounter := mapObject.GetEncounter(selectedPoint.X, selectedPoint.Y)
                    if encounter != nil && encounter.Type != maplib.EncounterTypeChaosNode && encounter.Type != maplib.EncounterTypeNatureNode && encounter.Type != maplib.EncounterTypeSorceryNode {
                        fonts.YellowFont.PrintCenter(screen, float64(280), y, scale.ScaleAmount, ebiten.ColorScale{}, encounter.Type.Name())
                        y += fonts.YellowFont.HeightAt(scale.ScaleAmount)
                    }

                    // Enemies
                    if encounter != nil {
                        text := "Unexplored"
                        if encounter.ExploredBy.Contains(player) {
                            text = "Empty"
                            if len(encounter.Units) > 0 {
                                text = encounter.Units[0].Name
                            }
                        }
                        fonts.WhiteFont.PrintCenter(screen, float64(280), y, scale.ScaleAmount, ebiten.ColorScale{}, text)
                        y += fonts.WhiteFont.HeightAt(scale.ScaleAmount)
                    }

                    // FIXME: how should this behave for different fog types?
                    if cityMap[selectedPoint] != nil {
                        city := cityMap[selectedPoint]
                        fonts.YellowFont.PrintWrapCenter(screen, float64(280), y, float64(cancelBackground.Bounds().Dx() - 5), scale.ScaleAmount, ebiten.ColorScale{}, city.String())
                    }

                    y = float64(170) - cityInfoText.TotalHeight

                    if resources.Enabled {
                        y = float64(170) - fonts.WhiteFont.HeightAt(scale.ScaleAmount) * 3 - cityInfoText.TotalHeight
                    }

                    fonts.YellowFont.RenderWrapped(screen, float64(245), y, cityInfoText, font.FontOptions{Scale: scale.ScaleAmount})
                    y += cityInfoText.TotalHeight

                    if resources.Enabled {
                        fonts.WhiteFont.Print(screen, float64(245), y, scale.ScaleAmount, ebiten.ColorScale{}, "Maximum Pop")
                        fonts.WhiteFont.PrintRight(screen, float64(308), y, scale.ScaleAmount, ebiten.ColorScale{}, fmt.Sprintf("%v", resources.MaximumPopulation))
                        y += fonts.WhiteFont.HeightAt(scale.ScaleAmount)
                        fonts.WhiteFont.Print(screen, float64(245), y, scale.ScaleAmount, ebiten.ColorScale{}, "Prod Bonus")
                        fonts.WhiteFont.PrintRight(screen, float64(314), y, scale.ScaleAmount, ebiten.ColorScale{}, fmt.Sprintf("+%v%%", resources.ProductionBonus))
                        y += fonts.WhiteFont.HeightAt(scale.ScaleAmount)
                        fonts.WhiteFont.Print(screen, float64(245), y, scale.ScaleAmount, ebiten.ColorScale{}, "Gold Bonus")
                        fonts.WhiteFont.PrintRight(screen, float64(314), y, scale.ScaleAmount, ebiten.ColorScale{}, fmt.Sprintf("+%v%%", resources.GoldBonus))
                    }
                }
            }
        },
    }

    ui.SetElementsFromArray(nil)

    makeButton := func(lbxIndex int, x int, y int) *uilib.UIElement {
        button, _ := game.ImageCache.GetImage("main.lbx", lbxIndex, 0)
        var options ebiten.DrawImageOptions
        options.GeoM.Translate(float64(x), float64(y))
        return &uilib.UIElement{
            Draw: func(element *uilib.UIElement, screen *ebiten.Image){
                scale.DrawScaled(screen, button, &options)
            },
        }
    }

    // game
    ui.AddElement(makeButton(1, 7, 4))

    // spells
    ui.AddElement(makeButton(2, 47, 4))

    // army button
    ui.AddElement(makeButton(3, 89, 4))

    // cities button
    ui.AddElement(makeButton(4, 140, 4))

    // magic button
    ui.AddElement(makeButton(5, 184, 4))

    // info button
    ui.AddElement(makeButton(6, 226, 4))

    // plane button
    ui.AddElement((func () *uilib.UIElement {
        buttons, _ := game.ImageCache.GetImages("main.lbx", 7)
        x := 270
        y := 4

        clicked := false

        var options ebiten.DrawImageOptions
        options.GeoM.Translate(float64(x), float64(y))

        return &uilib.UIElement{
            Rect: util.ImageRect(x, y, buttons[0]),
            PlaySoundLeftClick: true,
            LeftClick: func(element *uilib.UIElement){
                clicked = true
            },
            LeftClickRelease: func(element *uilib.UIElement){
                clicked = false
                game.Model.SwitchPlane()
                overworld = makeOverworld()
            },
            Draw: func(element *uilib.UIElement, screen *ebiten.Image){
                if clicked {
                    scale.DrawScaled(screen, buttons[1], &options)
                } else {
                    scale.DrawScaled(screen, buttons[0], &options)
                }
            },
        }
    })())

    // cancel button at bottom
    cancel, _ := game.ImageCache.GetImages("main.lbx", 41)
    cancelIndex := 0
    cancelRect := util.ImageRect(263, 182, cancel[0])
    ui.AddElement(&uilib.UIElement{
        Rect: cancelRect,
        LeftClick: func(element *uilib.UIElement){
            cancelIndex = 1
        },
        LeftClickRelease: func(element *uilib.UIElement){
            cancelIndex = 0
            quit = true
        },
        Draw: func(element *uilib.UIElement, screen *ebiten.Image){
            var options ebiten.DrawImageOptions
            options.GeoM.Translate(float64(cancelRect.Min.X), float64(cancelRect.Min.Y))
            scale.DrawScaled(screen, cancel[cancelIndex], &options)
        },
    })

    game.PushDrawer(func(screen *ebiten.Image){
        // the wide map layout, like the world map. see wide.go
        game.updateCameraSize()
        overworld.Camera = game.displayCamera()

        overworld.DrawOverworld(screen, ebiten.GeoM{})

        mini := screen.SubImage(game.GetMinimapScreenRect()).(*ebiten.Image)
        overworld.DrawMinimap(mini)

        game.drawWideHud(ui, screen)
    })
    defer game.PopDrawer()
    game.markOverlayDrawer()
    ui.CursorTransform = game.shiftedCursorTransform

    for !quit {
        overworld.Counter += 1
        zoomed := game.doInputZoom(yield)
        _ = zoomed

        ui.StandardUpdate()

        x, y := inputmanager.MousePosition()
        // development: a capture points the surveyor at a tile without a mouse, see capturescreens.go
        if CaptureSurveyorTile != nil {
            screenX, screenY := game.TileToScreen(CaptureSurveyorTile.X, CaptureSurveyorTile.Y)
            x, y = scale.Scale2(screenX, screenY)
        }

        // within the viewable area
        if game.InOverworldArea(x, y) {
            newX, newY := game.ScreenToTile(float64(x), float64(y))
            newPoint := image.Pt(newX, newY)

            // right click should move the camera
            rightClick := inputmanager.RightClick()
            if rightClick /*|| zoomed*/ {
                game.doMoveCamera(yield, newX, newY)
            }

            if selectedPoint != newPoint {
                selectedPoint = newPoint
                resources.Enabled = false

                text := ""

                tile := game.Model.CurrentMap().GetTile(newX, newY)

                if !tile.Tile.IsLand() {
                    text = "Cannot build cities on water."
                } else if cityMap[newPoint] == nil && game.Model.NearCity(newPoint, 3, game.Model.Plane) {
                    text = "Cities cannot be built less than 3 squares from any other city."
                } else {
                    text = "City Resources"
                    resources.Enabled = true
                    // FIXME: compute proper values for these
                    resources.MaximumPopulation = game.Model.ComputeMaximumPopulation(newX, newY, game.Model.Plane)
                    resources.ProductionBonus = game.CityProductionBonus(newX, newY, game.Model.Plane)
                    resources.GoldBonus = game.CityGoldBonus(newX, newY, game.Model.Plane)
                }

                cityInfoText = fonts.YellowFont.CreateWrappedText(float64(cancelBackground.Bounds().Dx() - 9), font.RelativeTextSize(scale.ScaleAmount, font.TextSizeCompact), text)
            }
        } else {
            cityInfoText.Clear()
            resources.Enabled = false
        }

        yield()
    }
}
