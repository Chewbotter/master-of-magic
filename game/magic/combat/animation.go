package combat

// How the figures on the battlefield animate, matched to the original game.
//
// The original redraws the battlefield on every tick of its timer, 18.2 times a second, and steps its
// animations once per redraw. Figure pictures have four frames: 0, 1 and 2 are the walk, 1 is also
// the standing pose, 3 is the strike.
// (ReMoM: MoM/src/Combat.c Combat_Compose_Background, Battle_Unit_Set_Animation_Flags, the frame_num
// blocks of the figure compose functions, Combat_Figure_Active_Red_Outline, the outline drawing in the
// map draw. The code here is ours.)

import (
    "fmt"
    "image"
    "image/color"
    "math"

    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/game/magic/util"

    "github.com/hajimehoshi/ebiten/v2"
)

// the original's redraws per second
const OriginalTicksPerSecond = 18.2

// redraws of the original one step from a cell to the next takes
const MoveTicksPerCell = 8

const figureStandFrame = 1
// walking goes forward and back through the first three frames
var figureWalkFrames = []int{1, 2, 1, 0}
// flying units, and a few that never stand still, cycle through them
const figureIdleFrames = 3
// a strike alternates between the pose and the strike frame. the defender is one step out of phase
var figureAttackFrames = []int{3, 1}
var figureDefendFrames = []int{1, 3}

// units that animate while standing although they do not fly
var alwaysAnimated = map[string]bool{
    "Great Wyrm": true,
    "Fire Elemental": true,
}

// the outlines of the cells under the cursor and under the selected unit are pictures
const cellOutlineLbx = "cmbtcity.lbx"
const cellOutlineScannedIndex = 67
const cellOutlineActiveIndex = 68
const cellOutlineFrames = 3

// the outline of the figures of the unit under the cursor pulses from black to this red and back.
// 55 of 63, the original's value, over 8 steps each way
const scannedOutlinePaletteIndex = 1
const scannedOutlineRed = 55.0 / 63.0
const scannedOutlineSteps = 8

// the original's redraw counter, from our ticks
func (combat *CombatScreen) originalTick() uint64 {
    tps := float64(max(1, ebiten.TPS()))
    return uint64(float64(combat.Counter) * OriginalTicksPerSecond / tps)
}

// our ticks a step from one cell to the next takes
func moveTicksPerCell() float64 {
    return MoveTicksPerCell * float64(max(1, ebiten.TPS())) / OriginalTicksPerSecond
}

// the frame of its picture a unit shows
func (combat *CombatScreen) figureFrame(unit *ArmyUnit, frameCount int) int {
    tick := combat.originalTick()

    flyingKind := unit.Unit.IsFlying()
    animateIdle := (flyingKind && unit.IsFlying()) || alwaysAnimated[unit.Unit.GetName()]

    frame := figureStandFrame
    if animateIdle {
        frame = int(tick % figureIdleFrames)
    } else if unit.Moving {
        if flyingKind {
            frame = int(tick % figureIdleFrames)
        } else {
            frame = figureWalkFrames[tick % uint64(len(figureWalkFrames))]
        }
    }

    if unit.Attacking {
        frame = figureAttackFrames[tick % uint64(len(figureAttackFrames))]
    } else if unit.Defending {
        frame = figureDefendFrames[tick % uint64(len(figureDefendFrames))]
    }

    if frameCount <= 0 {
        return 0
    }

    return min(frame, frameCount - 1)
}

// how far the outline of the unit under the cursor is from black to red, 0 to scannedOutlineSteps
func (combat *CombatScreen) scannedOutlineStep() int {
    position := int(combat.originalTick() % (scannedOutlineSteps * 2))
    if position > scannedOutlineSteps {
        position = scannedOutlineSteps * 2 - position
    }
    return position
}

// recolors the outline of a figure picture, after the banner colors have been applied
func withScannedOutline(base util.ImageTransformFunc, step int) util.ImageTransformFunc {
    red := uint8(math.Round(255 * scannedOutlineRed * float64(step) / scannedOutlineSteps))

    return func(original *image.Paletted) image.Image {
        out := base(original)
        paletted, ok := out.(*image.Paletted)
        if ok && len(paletted.Palette) > scannedOutlinePaletteIndex {
            paletted.Palette = util.ClonePalette(paletted.Palette)
            paletted.Palette[scannedOutlinePaletteIndex] = color.RGBA{R: red, G: 0, B: 0, A: 0xff}
        }
        return out
    }
}

func scannedOutlineKey(bannerKey string, step int) string {
    return fmt.Sprintf("%v-scanned%v", bannerKey, step)
}

// draws the outline of a cell: under the cursor, or under the selected unit
func (combat *CombatScreen) drawCellOutline(screen *ebiten.Image, x int, y int, active bool) {
    index := cellOutlineScannedIndex
    if active {
        index = cellOutlineActiveIndex
    }

    images, err := combat.ImageCache.GetImages(cellOutlineLbx, index)
    if err != nil || len(images) == 0 {
        return
    }

    frame := int(combat.originalTick() % cellOutlineFrames)
    frame = min(frame, len(images) - 1)

    matrix := combat.GetCameraMatrix()
    tx, ty := matrix.Apply(float64(x), float64(y))

    var options ebiten.DrawImageOptions
    options.GeoM.Translate(TerrainOffsetX, TerrainOffsetY)
    options.GeoM.Scale(combat.CameraScale, combat.CameraScale)
    options.GeoM.Translate(tx, ty)
    scale.DrawScaled(screen, images[frame], &options)
}
