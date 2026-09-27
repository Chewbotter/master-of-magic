package game

// The map keeps moving after a pan is released, at the speed of the drag, and slows to a stop.
// Letting go of a still mouse does not coast: the measured speed has already fallen to zero.

import (
    "log"
    "math"

    "github.com/hajimehoshi/ebiten/v2"
)

// seconds for the coasting speed to fall to about a third. the distance coasted is the release speed times this
const PanCoastTime = 0.15
// slowest release, in screen pixels per second, that starts a coast
const PanCoastStartSpeed = 60.0
// the coast ends below this speed, in screen pixels per second
const PanCoastStopSpeed = 15.0
// seconds of drag the release speed is measured over. shorter reacts faster to a flick
const PanVelocitySmoothTime = 0.05

var coasting bool
// map movement in screen pixels per second
var panVelocityX, panVelocityY float64
// where the last coast step left the camera. any other value means something else moved it
var coastOffsetX, coastOffsetY float64

func panTickSeconds() float64 {
    return 1 / float64(max(1, ebiten.TPS()))
}

func (game *Game) startCoast() {
    speed := math.Hypot(panVelocityX, panVelocityY)
    coasting = speed >= PanCoastStartSpeed
    coastOffsetX = game.Camera.GetOffsetX()
    coastOffsetY = game.Camera.GetOffsetY()
    if PanTrace {
        log.Printf("pan released at %.0f px/s, coast %v", speed, coasting)
    }
}

// one tick of coasting. returns false when there is no coast going on
func (game *Game) coastTick() bool {
    if !coasting {
        return false
    }

    width := float64(game.Model.CurrentMap().Width())
    movedElsewhere := math.Abs(wrapDelta(game.Camera.GetOffsetX() - coastOffsetX, width)) > 1e-6 ||
                      math.Abs(game.Camera.GetOffsetY() - coastOffsetY) > 1e-6
    // a zoom or a jump to a unit ends the coast
    if movedElsewhere || game.Camera.AnimatedZoom != 0 {
        coasting = false
        return false
    }

    tick := panTickSeconds()
    beforeY := game.Camera.GetOffsetY()
    _, tileHeight := game.tilePixels()

    game.PanBy(panVelocityX * tick, panVelocityY * tick)

    // stopped by the top or bottom of the map: no more vertical speed
    if tileHeight > 0 && math.Abs((beforeY - game.Camera.GetOffsetY()) * tileHeight - panVelocityY * tick) > 0.01 {
        panVelocityY = 0
    }

    decay := math.Exp(-tick / PanCoastTime)
    panVelocityX *= decay
    panVelocityY *= decay

    coastOffsetX = game.Camera.GetOffsetX()
    coastOffsetY = game.Camera.GetOffsetY()

    speed := math.Hypot(panVelocityX, panVelocityY)
    if PanTrace {
        log.Printf("coast at %.0f px/s", speed)
    }
    if speed < PanCoastStopSpeed {
        coasting = false
        return false
    }

    return true
}
