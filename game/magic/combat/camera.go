package combat

// The camera of the battlefield: whole pixel zoom levels and smooth panning, like the world map.
//
// The default view is the original's: the battlefield at the size of the art, not moved. Zooming out
// shows more of the field at a smaller, still whole, number of screen pixels per art pixel. The
// battlefield is always drawn on whole screen pixels, so it stays crisp while it moves.
//
// KEEP IN STEP with the world map's camera (game/magic/game/pan.go, pancoast.go, camera/levels.go):
// the same gain, glide and coast, so both scenes feel the same.

import (
    "math"
    "time"

    "github.com/kazzmir/master-of-magic/game/magic/camera"
    "github.com/kazzmir/master-of-magic/game/magic/display"
    "github.com/kazzmir/master-of-magic/game/magic/inputmanager"
    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/inpututil"
)

const BattlePanButton = ebiten.MouseButtonMiddle
// how far the battlefield moves for each pixel the mouse moves
const BattlePanGain = 1.25
// seconds for the view to cover about two thirds of the distance to where it is going
const BattlePanSmoothTime = 0.03
// seconds for the speed of a released drag to fall to about a third
const BattleCoastTime = 0.15
// slowest release, in screen pixels per second, that keeps moving
const BattleCoastStartSpeed = 60.0
const BattleCoastStopSpeed = 15.0
// seconds of drag the release speed is measured over
const BattleVelocitySmoothTime = 0.05
// how fast the arrow keys move the battlefield, in art pixels per tick
const BattleKeyPanSpeed = 2.0
// ticks a change of zoom level takes
const BattleZoomTicks = 8
// the middle of the battlefield view, in art pixels. zooming keeps this point in place
const BattleViewCenterX = 160
const BattleViewCenterY = 82

type BattleCamera struct {
    // screen pixels per art pixel. 0 until the first update, then a whole number
    Level int
    // the size the battlefield is drawn at while a change of level animates, in screen pixels per art pixel
    shownLevel float64
    zoomFrom float64
    zoomTick int

    // where the battlefield is going, and where it is drawn, in screen pixels from its original position
    targetX float64
    targetY float64
    viewX float64
    viewY float64

    dragging bool
    lastMouseX int
    lastMouseY int
    // screen pixels per second
    velocityX float64
    velocityY float64
    coasting bool
    lastTime time.Time
}

func battleTickSeconds() float64 {
    return 1 / float64(max(1, ebiten.TPS()))
}

// the zoom levels for the current draw scale, closest first, in screen pixels per art pixel
func battleLevels() []int {
    var out []int
    for _, zoom := range camera.ZoomLevels(scale.ScaleAmount) {
        out = append(out, int(math.Round(float64(zoom) * scale.ScaleAmount / float64(camera.ZoomMax))))
    }
    return out
}

func (battle *BattleCamera) clampLevel() {
    levels := battleLevels()
    closest := levels[0]
    farthest := levels[len(levels) - 1]

    if battle.Level == 0 {
        battle.Level = closest
        battle.shownLevel = float64(closest)
    }

    // the draw scale can change with the window
    if battle.Level > closest {
        battle.Level = closest
        battle.shownLevel = float64(closest)
    }
    if battle.Level < farthest {
        battle.Level = farthest
        battle.shownLevel = float64(farthest)
    }
}

func (battle *BattleCamera) zoom(direction int) {
    levels := battleLevels()
    next := battle.Level
    for index, level := range levels {
        if level != battle.Level {
            continue
        }
        if direction > 0 && index > 0 {
            next = levels[index - 1]
        }
        if direction < 0 && index < len(levels) - 1 {
            next = levels[index + 1]
        }
    }

    if next != battle.Level {
        battle.zoomFrom = battle.shownLevel
        battle.zoomTick = BattleZoomTicks
        battle.Level = next
    }
}

func (battle *BattleCamera) reset() {
    levels := battleLevels()
    if battle.Level != levels[0] {
        battle.zoomFrom = battle.shownLevel
        battle.zoomTick = BattleZoomTicks
        battle.Level = levels[0]
    }
    battle.targetX = 0
    battle.targetY = 0
    battle.coasting = false
}

func (battle *BattleCamera) moveBy(deltaX float64, deltaY float64) {
    battle.targetX += deltaX
    battle.targetY += deltaY
    // the view stops at the edge of the ground around the field, see fieldedge.go
    battle.clampToField()
}

// one tick of input. inBattlefield says whether the cursor is over the battlefield, where a drag can start
func (battle *BattleCamera) Update(keys []ebiten.Key, inBattlefield bool) {
    battle.clampLevel()

    for _, key := range keys {
        step := BattleKeyPanSpeed * scale.ScaleAmount
        switch key {
            case ebiten.KeyDown: battle.moveBy(0, -step)
            case ebiten.KeyUp: battle.moveBy(0, step)
            case ebiten.KeyLeft: battle.moveBy(step, 0)
            case ebiten.KeyRight: battle.moveBy(-step, 0)
            case ebiten.KeySpace: battle.reset()
        }
    }

    if inpututil.IsKeyJustPressed(ebiten.KeyEqual) {
        battle.zoom(1)
    }
    if inpututil.IsKeyJustPressed(ebiten.KeyMinus) {
        battle.zoom(-1)
    }

    _, wheelY := inputmanager.Wheel()
    if wheelY > 0 {
        battle.zoom(1)
    } else if wheelY < 0 {
        battle.zoom(-1)
    }

    if battle.zoomTick > 0 {
        battle.zoomTick -= 1
        done := 1 - float64(battle.zoomTick) / BattleZoomTicks
        eased := done * done * (3 - 2 * done)
        battle.shownLevel = battle.zoomFrom + (float64(battle.Level) - battle.zoomFrom) * eased
    } else {
        battle.shownLevel = float64(battle.Level)
    }

    battle.updateDrag(inBattlefield)
    // a change of zoom level or of the window can put the view over the edge
    battle.clampToField()
}

func (battle *BattleCamera) updateDrag(inBattlefield bool) {
    tick := battleTickSeconds()
    mouseX, mouseY := inputmanager.MousePosition()

    pressed := display.ModernControls() && ebiten.IsMouseButtonPressed(BattlePanButton)
    if !pressed {
        if battle.dragging {
            battle.dragging = false
            battle.coasting = math.Hypot(battle.velocityX, battle.velocityY) >= BattleCoastStartSpeed
        }

        if battle.coasting {
            battle.moveBy(battle.velocityX * tick, battle.velocityY * tick)
            decay := math.Exp(-tick / BattleCoastTime)
            battle.velocityX *= decay
            battle.velocityY *= decay
            if math.Hypot(battle.velocityX, battle.velocityY) < BattleCoastStopSpeed {
                battle.coasting = false
            }
        } else {
            // at rest on whole pixels
            battle.targetX = math.Round(battle.targetX)
            battle.targetY = math.Round(battle.targetY)
        }
        return
    }

    if !battle.dragging {
        if inpututil.IsMouseButtonJustPressed(BattlePanButton) && inBattlefield {
            battle.dragging = true
            battle.coasting = false
            battle.velocityX, battle.velocityY = 0, 0
            battle.lastMouseX, battle.lastMouseY = mouseX, mouseY
        }
        return
    }

    deltaX := float64(mouseX - battle.lastMouseX) * BattlePanGain
    deltaY := float64(mouseY - battle.lastMouseY) * BattlePanGain
    battle.lastMouseX, battle.lastMouseY = mouseX, mouseY

    step := 1 - math.Exp(-tick / BattleVelocitySmoothTime)
    battle.velocityX += (deltaX / tick - battle.velocityX) * step
    battle.velocityY += (deltaY / tick - battle.velocityY) * step

    battle.moveBy(deltaX, deltaY)
}

// moves the shown position toward the target. called once per drawn frame
func (battle *BattleCamera) glide() {
    now := time.Now()
    frameTime := 0.0
    if !battle.lastTime.IsZero() {
        frameTime = min(0.1, max(0, now.Sub(battle.lastTime).Seconds()))
    }
    battle.lastTime = now

    step := 1 - math.Exp(-frameTime / BattlePanSmoothTime)
    battle.viewX += (battle.targetX - battle.viewX) * step
    battle.viewY += (battle.targetY - battle.viewY) * step

    if !battle.dragging && !battle.coasting && math.Abs(battle.targetX - battle.viewX) < 0.05 && math.Abs(battle.targetY - battle.viewY) < 0.05 {
        battle.viewX = battle.targetX
        battle.viewY = battle.targetY
    }
}

// the size the battlefield is drawn at, relative to the art
func (battle *BattleCamera) Scale() float64 {
    if battle.shownLevel <= 0 || scale.ScaleAmount <= 0 {
        return 1
    }
    return battle.shownLevel / scale.ScaleAmount
}

// maps a tile to the middle of its diamond on the screen, in art pixels of the screen
func (battle *BattleCamera) Matrix() ebiten.GeoM {
    matrix := MakeBattlefieldMatrix()
    zoom := battle.Scale()

    // zoom around the middle of the view
    matrix.Translate(-BattleViewCenterX, -BattleViewCenterY)
    matrix.Scale(zoom, zoom)
    matrix.Translate(BattleViewCenterX, BattleViewCenterY)

    // the shown position, on whole screen pixels
    matrix.Translate(math.Round(battle.viewX) / scale.ScaleAmount, math.Round(battle.viewY) / scale.ScaleAmount)
    return matrix
}

// development: puts the camera at a zoom level and position at once, for a capture
func (battle *BattleCamera) SetView(level int, panX float64, panY float64) {
    battle.Level = level
    battle.shownLevel = float64(level)
    battle.zoomTick = 0
    battle.clampLevel()
    battle.targetX, battle.targetY = panX, panY
    battle.clampToField()
    battle.viewX, battle.viewY = battle.targetX, battle.targetY
}
