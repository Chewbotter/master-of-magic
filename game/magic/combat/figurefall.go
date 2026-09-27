package combat

// Figures that are killed fall over: each is pushed back from where the blow came from and tips
// over backward around its feet, a little later or earlier than the others, lies there, and fades.
// Not in the original, which shows a splash of blood on a lost figure. Before this, lost figures
// turned into a colored shape that faded where it stood.

import (
    "math"
    "math/rand/v2"

    "github.com/kazzmir/master-of-magic/game/magic/units"

    "github.com/hajimehoshi/ebiten/v2"
)

// turns it off: lost figures fade where they stand, in a color
const FigureFall = true

// seconds a figure takes to fall
const figureFallTime = 0.3
// the latest a figure starts to fall after it was lost, in seconds
const figureFallDelayMax = 0.12
// how far a figure is pushed back, in art pixels
const figureFallPushMin = 4.0
const figureFallPushMax = 9.0
// how far it tips over, in degrees. 90 is flat on its back
const figureFallAngleMin = 75.0
const figureFallAngleMax = 100.0
// the last seconds of the time a lost figure is shown, in which it fades
const figureFallFadeTime = 0.25

// the way a unit faces on the screen, across and down
func facingOnScreen(facing units.Facing) (float64, float64) {
    // the sides of a tile run 2 across for 1 down
    diagonalX := 2 / math.Sqrt(5)
    diagonalY := 1 / math.Sqrt(5)

    switch facing {
        case units.FacingUp: return 0, -1
        case units.FacingUpRight: return diagonalX, -diagonalY
        case units.FacingRight: return 1, 0
        case units.FacingDownRight: return diagonalX, diagonalY
        case units.FacingDown: return 0, 1
        case units.FacingDownLeft: return -diagonalX, diagonalY
        case units.FacingLeft: return -1, 0
        case units.FacingUpLeft: return -diagonalX, -diagonalY
    }

    return 1, 0
}

type figureFall struct {
    // the tick the figure was lost, 0 while it stands
    Start uint64
    // seconds
    Delay float64
    // art pixels
    PushX float64
    PushY float64
    // radians, to the side it falls to
    Angle float64
}

// called for every figure that is drawn. lost says whether it is one of the lost figures
func (combat *CombatScreen) figureFallOf(unit *ArmyUnit, figure int, count int, lost bool) *figureFall {
    state := combat.figuresOf(unit, count, float64(unit.X), float64(unit.Y))
    fall := &state.Figures[figure].Fall

    if !lost {
        fall.Start = 0
        return fall
    }

    if fall.Start == 0 {
        fall.Start = combat.Counter
        fall.Delay = rand.Float64() * figureFallDelayMax

        // away from what the unit faces, which is what hit it
        faceX, faceY := facingOnScreen(unit.Facing)
        push := figureFallPushMin + rand.Float64() * (figureFallPushMax - figureFallPushMin)
        fall.PushX = -faceX * push
        fall.PushY = -faceY * push

        angle := (figureFallAngleMin + rand.Float64() * (figureFallAngleMax - figureFallAngleMin)) * math.Pi / 180
        // backward: to the side it is pushed to. straight up or down it can be either side
        if fall.PushX < 0 || (fall.PushX == 0 && rand.N(2) == 0) {
            angle = -angle
        }
        fall.Angle = angle
    }

    return fall
}

// how far a figure has fallen, 0 to 1
func (combat *CombatScreen) fallProgress(fall *figureFall) float64 {
    seconds := float64(combat.Counter - fall.Start) / float64(max(1, ebiten.TPS())) - fall.Delay
    return max(0, min(1, seconds / figureFallTime))
}

// how much of a lost figure shows, 1 to 0 over the end of the time it is shown
func lostFigureAlpha(unit *ArmyUnit) float32 {
    seconds := float64(unit.LostUnitsTime) / float64(max(1, ebiten.TPS()))
    return float32(max(0, min(1, seconds / figureFallFadeTime)))
}
