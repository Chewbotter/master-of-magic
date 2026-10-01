package combat

// Draws what stands on the battlefield in the original's order. See scenery.go.

import (
    "cmp"
    "math"
    "slices"

    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    "github.com/kazzmir/master-of-magic/game/magic/unitview"

    "github.com/hajimehoshi/ebiten/v2"
)

// one thing on the battlefield: a figure, a tree, a rock, a house, a piece of wall
type fieldDrawable struct {
    // from its position on the original's screen, see DrawOrder
    Order int
    // decides between things at one position
    Layer int
    Render func()
}

func sortFieldDrawables(drawables []fieldDrawable) {
    slices.SortStableFunc(drawables, func(a fieldDrawable, b fieldDrawable) int {
        return cmp.Or(cmp.Compare(a.Order, b.Order), cmp.Compare(a.Layer, b.Layer))
    })
}

// draws a picture with its top left corner at a position of the original's screen
func (combat *CombatScreen) drawOnField(screen *ebiten.Image, picture *ebiten.Image, x int, y int) {
    combat.drawOnFieldDimmed(screen, picture, x, y, 1)
}

// the same, darker: for what stands on the ground around the field, see fieldedge.go
func (combat *CombatScreen) drawOnFieldDimmed(screen *ebiten.Image, picture *ebiten.Image, x int, y int, brightness float32) {
    matrix := originalScreenMatrix(combat.GetCameraMatrix())

    var options ebiten.DrawImageOptions
    options.GeoM.Translate(float64(x), float64(y))
    options.GeoM.Concat(matrix)
    options.ColorScale.Scale(brightness, brightness, brightness, 1)
    scale.DrawScaled(screen, picture, &options)
}

// trees, rocks, houses, the fortress
func (combat *CombatScreen) sceneryDrawables(screen *ebiten.Image) []fieldDrawable {
    var out []fieldDrawable

    for index := range combat.Model.Scenery {
        piece := &combat.Model.Scenery[index]

        layer := layerStructure
        switch piece.Kind {
            case SceneryTree: layer = layerTree
            case SceneryRock: layer = layerRock
        }

        // darker on the border
        brightness := fieldBrightness(piece.ScreenX, piece.ScreenY)
        if brightness <= 0 {
            continue
        }

        // by depth, see depthsort.go: a house and the structure stand in their cell as a whole
        order := pointOrder(piece.ScreenX, piece.ScreenY)
        switch piece.Kind {
            case SceneryHouse:
                cellX, cellY := screenCell(piece.ScreenX, piece.ScreenY)
                order = cellOrder(cellX, cellY, depthFarEdge, piece.ScreenX, piece.ScreenY)
            case SceneryStructure:
                order = cellOrder(lairCellX, lairCellY, depthFarEdge, piece.ScreenX, piece.ScreenY)
        }

        out = append(out, fieldDrawable{
            Order: order,
            Layer: layer,
            Render: func() {
                pictures := combat.sceneryPictures(piece)
                if len(pictures) == 0 {
                    return
                }
                picture := pictures[0]
                anchorX, anchorY := piece.anchor(picture.Bounds().Dx(), picture.Bounds().Dy())
                combat.drawOnFieldDimmed(screen, picture, piece.ScreenX - anchorX, piece.ScreenY - anchorY, brightness)
            },
        })
    }

    return out
}

// the pieces of the walls around a town: stone, fire, darkness
func (combat *CombatScreen) wallDrawables(screen *ebiten.Image) []fieldDrawable {
    var out []fieldDrawable

    if combat.Model.Zone.City == nil {
        return out
    }

    magicFrame := int(combat.originalTick() / wallMagicTicksPerFrame % wallMagicFrames)

    drawMagic := func(piece wallPiece, x int, y int, index int, activeCounter uint64) func() {
        return func() {
            anchorX := piece.AnchorX

            var picture *ebiten.Image
            rising, _ := combat.ImageCache.GetImages(wallRiseLbx, index + piece.Piece)
            riseFrame := (combat.Counter - activeCounter) / wallRiseTicksPerFrame
            if riseFrame < uint64(len(rising)) {
                picture = rising[riseFrame]
                if anchorX == wallHalfWidth {
                    anchorX = wallRiseCornerAnchorX
                }
            } else {
                pictures, err := combat.ImageCache.GetImages(wallLbx, index + piece.Piece)
                if err != nil || len(pictures) == 0 {
                    return
                }
                picture = pictures[magicFrame % len(pictures)]
            }

            if anchorX == wallHalfWidth {
                anchorX = picture.Bounds().Dx() / 2
            }

            combat.drawOnField(screen, picture, x - anchorX, y - piece.MagicAnchorY)
        }
    }

    stoneIndex := wallStoneIndex(combat.Model.Plane)

    for _, piece := range wallPieces {
        tileX, tileY := CellToTile(piece.CellX, piece.CellY)
        tile := combat.Model.GetTile(tileX, tileY)
        if tile == nil {
            continue
        }

        x, y := cellScreen(piece.CellX, piece.CellY, 0, 0)
        x += piece.ShiftX
        y += piece.ShiftY
        // a piece of wall stands at an edge of its cell, see depthsort.go
        part := depthFarEdge
        if piece.Near {
            part = depthNearEdge
        }
        order := cellOrder(piece.CellX, piece.CellY, part, x, y)

        if tile.Wall != nil && !tile.Wall.IsEmpty() && !piece.MagicOnly {
            layer := layerWallFarStone
            if piece.Near {
                layer = layerWallNearStone
            }

            out = append(out, fieldDrawable{
                Order: order,
                Layer: layer,
                Render: func() {
                    pictures, err := combat.ImageCache.GetImages(wallLbx, stoneIndex + piece.Piece)
                    if err != nil || len(pictures) == 0 {
                        return
                    }

                    frame := wallFrameStanding
                    if tile.WallDestroyed {
                        frame = wallFrameBroken
                    }
                    picture := pictures[min(frame, len(pictures) - 1)]

                    anchorX := piece.AnchorX
                    if anchorX == wallHalfWidth {
                        anchorX = picture.Bounds().Dx() / 2
                    }

                    combat.drawOnField(screen, picture, x - anchorX, y - piece.AnchorY)
                },
            })
        }

        if tile.Darkness != nil && !tile.Darkness.IsEmpty() {
            layer := layerWallFarDarkness
            if piece.Near {
                layer = layerWallNearDarkness
            }

            out = append(out, fieldDrawable{Order: order, Layer: layer, Render: drawMagic(piece, x, y, wallDarknessIndex, tile.DarknessActive)})
        }

        if tile.Fire != nil && !tile.Fire.IsEmpty() {
            layer := layerWallFarFire
            if piece.Near {
                layer = layerWallNearFire
            }

            out = append(out, fieldDrawable{Order: order, Layer: layer, Render: drawMagic(piece, x, y, wallFireIndex, tile.FireActive)})
        }
    }

    return out
}

// the cave, tower, temple, keep, ruins or nature node in the middle of the battlefield
func (combat *CombatScreen) structureDrawables(render func(x int, y int, extra TileTop)) []fieldDrawable {
    var out []fieldDrawable

    x, y := LairTile()
    tile := combat.Model.GetTile(x, y)
    if tile == nil || !tile.ExtraObject.standsOnField() {
        return out
    }

    screenX, screenY := cellScreen(lairCellX, lairCellY, 0, 0)

    return append(out, fieldDrawable{
        Order: cellOrder(lairCellX, lairCellY, depthFarEdge, screenX, screenY),
        Layer: layerStructure,
        Render: func() {
            render(x, y, tile.ExtraObject)
        },
    })
}

// the figures of a unit, each in its own place of the order. render draws one figure of the unit,
// and with it what is shown over the whole unit if overlays is set
// A UNIT THAT WALKS IS SORTED BY A CELL, NOT BY WHERE IT IS (user, 2026-09-30: "long characters
// like horse riders will pop back and forth when trying to determine who should be in front").
// The order is by the row of cells, then the cell (DrawOrder). Sorted by where they were, the
// figures of a unit that walks crossed into the next row of cells one after the other, so two
// horses of one unit, or a horse and the figure beside it, went in front of each other and back
// again during a step. Now all figures of a unit are sorted by one cell, as a unit that stands is,
// with their places in it: while it walks, the one of the cells it is between that is drawn last,
// so it goes over what it passes in front of. The order changes once a step, not every pixel.
// STILL POPPED (user, same day), so the field is sorted by depth now (depthsort.go): this cell
// only decides among things of the same depth, and the whole order when DepthSort is off.

// the middle on the original's screen of the cell a unit at a place is sorted by. facing is the way
// it walks
func sortCellMiddle(x float64, y float64, facing units.Facing) (float64, float64) {
    field := MakeBattlefieldMatrix()

    // the two cells it is between. a unit walks straight from the middle of a cell to the middle
    // of a cell next to it. on a diagonal of the grid it is between the corners of a square of
    // four cells: the way it faces says which two. a step to more x and more y faces right on the
    // screen, to less of both left; the other diagonal faces up or down (faceTowards)
    lowX, highX := math.Floor(x), math.Ceil(x)
    lowY, highY := math.Floor(y), math.Ceil(y)
    cells := [2][2]float64{{lowX, lowY}, {highX, highY}}
    if lowX != highX && lowY != highY && (facing == units.FacingUp || facing == units.FacingDown) {
        cells = [2][2]float64{{lowX, highY}, {highX, lowY}}
    }

    best := 0
    var bestX, bestY float64
    for index, cell := range cells {
        middleX, middleY := field.Apply(cell[0], cell[1])
        order := DrawOrder(int(math.Floor(middleX)), int(math.Floor(middleY)))
        if index == 0 || order > best {
            best, bestX, bestY = order, middleX, middleY
        }
    }
    return bestX, bestY
}

func (combat *CombatScreen) unitDrawables(unit *ArmyUnit, render func(unit *ArmyUnit, figure int, overlays bool)) []fieldDrawable {
    var out []fieldDrawable

    points := unitview.CombatPoints(unit.VisibleFigures() + unit.LostUnits)
    if len(points) == 0 {
        return out
    }

    x, y := float64(unit.X), float64(unit.Y)
    if unit.Moving {
        x, y = unit.MoveX, unit.MoveY
    }

    // the middle of the cell the unit is sorted by on the original's screen: the order of the
    // original, which decides among figures of the same depth
    middleX, middleY := sortCellMiddle(x, y, unit.Facing)

    field := MakeBattlefieldMatrix()

    last := 0
    for index, point := range points {
        tieX, tieY := int(math.Floor(middleX)) + point.X, int(math.Floor(middleY)) + point.Y

        // where its feet are drawn, as renderUnit puts it: the unit's place and the figure's place
        // in the cell
        feetX, feetY := field.Apply(x, y)
        feetX += float64(point.X)
        feetY += float64(point.Y)

        order := depthOrder(feetX, feetY, tieX, tieY)
        out = append(out, fieldDrawable{Order: order, Layer: layerFigure})
        if order >= out[last].Order {
            last = index
        }
    }

    for index := range out {
        out[index].Render = func() {
            render(unit, index, index == last)
        }
    }

    return out
}

// the roads of a town, the clouds of a flying fortress, the ground under an outpost. part of the
// ground: drawn before anything that stands on the field
// (ReMoM: MoM/src/Combat.c, the end of the ground loop of the map draw)
func (combat *CombatScreen) drawTownGround(screen *ebiten.Image) {
    city := combat.Model.Zone.City
    if city == nil {
        return
    }

    draw := func(index int, cgx int, cgy int, shift int) {
        pictures, err := combat.ImageCache.GetImages(townGroundLbx, index)
        if err != nil || len(pictures) == 0 {
            return
        }
        x, y := cellScreen(cgx, cgy, 0, 0)
        combat.drawOnField(screen, pictures[0], x + cellTerrainX - shift, y - shift)
    }

    if city.Outpost {
        draw(outpostGroundIndex, lairCellX, lairCellY, 0)
        return
    }

    if combat.DrawClouds {
        draw(townCloudsIndex, townCellMaxX, townCellMaxY, townGroundShift)
    } else {
        draw(townRoadsIndex, townCellMaxX, townCellMaxY, townGroundShift)
    }
}
