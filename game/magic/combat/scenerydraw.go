package combat

// Draws what stands on the battlefield in the original's order. See scenery.go.

import (
    "cmp"
    "math"
    "slices"

    "github.com/kazzmir/master-of-magic/game/magic/scale"
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
    matrix := originalScreenMatrix(combat.GetCameraMatrix())

    var options ebiten.DrawImageOptions
    options.GeoM.Translate(float64(x), float64(y))
    options.GeoM.Concat(matrix)
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

        out = append(out, fieldDrawable{
            Order: DrawOrder(piece.ScreenX, piece.ScreenY),
            Layer: layer,
            Render: func() {
                pictures, err := combat.ImageCache.GetImages(piece.Lbx, piece.Index)
                if err != nil || len(pictures) == 0 {
                    return
                }
                picture := pictures[0]
                anchorX, anchorY := piece.anchor(picture.Bounds().Dx(), picture.Bounds().Dy())
                combat.drawOnField(screen, picture, piece.ScreenX - anchorX, piece.ScreenY - anchorY)
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
        order := DrawOrder(x, y)

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
        Order: DrawOrder(screenX, screenY),
        Layer: layerStructure,
        Render: func() {
            render(x, y, tile.ExtraObject)
        },
    })
}

// the figures of a unit, each in its own place of the order. render draws one figure of the unit,
// and with it what is shown over the whole unit if overlays is set
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

    // the middle of the unit's cell on the original's screen
    field := MakeBattlefieldMatrix()
    middleX, middleY := field.Apply(x, y)

    last := 0
    for index, point := range points {
        order := DrawOrder(int(math.Floor(middleX)) + point.X, int(math.Floor(middleY)) + point.Y)
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
