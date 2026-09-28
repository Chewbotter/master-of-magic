package combat

// Particles: single art pixels that fly, fall and go out. Not in the original (user, 2026-09-28).
//
// The pictures of a spell are drawn by hand. Particles go with them: embers that come off a bolt
// in flight, and what is thrown up where a spell hits. A particle is one pixel of the art, at a
// place on the original's screen, so it is as large as the pixels of everything else at every zoom.
// Its color goes through a row of colors while it lives, in steps.

import (
    "image/color"
    "math"
    "math/rand/v2"
)

// no more than this many at a time. the oldest go first
const particleMax = 3000
// of the speed it comes down with a particle keeps this much when it bounces, and of its speed
// along the ground
const particleBounce = 0.45
const particleSlide = 0.6
// slower than this, in art pixels a second, a particle on the ground lies still
const particleRest = 12.0
// how much lower on the screen a way along the ground is than it is wide: the ground is seen from above
const particleGroundSquash = 0.5

type particle struct {
    // on the original's screen, in art pixels
    X float64
    Y float64
    // art pixels a second
    SpeedX float64
    SpeedY float64
    // art pixels a second more with every second, downward
    Gravity float64
    // seconds
    Age float64
    Life float64
    // the colors it goes through while it lives
    Colors []color.RGBA
    // below this it bounces, if it has a ground
    HasGround bool
    Ground float64
}

type particleSystem struct {
    Particles []particle
}

func (system *particleSystem) add(each particle) {
    if each.Life <= 0 || len(each.Colors) == 0 {
        return
    }

    if len(system.Particles) >= particleMax {
        system.Particles = system.Particles[1:]
    }
    system.Particles = append(system.Particles, each)
}

// what happens in a part of a second
func (system *particleSystem) step(seconds float64) {
    kept := system.Particles[:0]

    for _, each := range system.Particles {
        each.Age += seconds
        if each.Age >= each.Life {
            continue
        }

        each.SpeedY += each.Gravity * seconds
        each.X += each.SpeedX * seconds
        each.Y += each.SpeedY * seconds

        if each.HasGround && each.Y > each.Ground && each.SpeedY > 0 {
            each.Y = each.Ground
            if each.SpeedY < particleRest {
                // it lies
                each.SpeedX = 0
                each.SpeedY = 0
                each.Gravity = 0
            } else {
                each.SpeedY = -each.SpeedY * particleBounce
                each.SpeedX *= particleSlide
            }
        }

        kept = append(kept, each)
    }

    system.Particles = kept
}

// the color a particle has now
func (each *particle) color() color.RGBA {
    if len(each.Colors) == 0 || each.Life <= 0 {
        return color.RGBA{}
    }

    index := int(each.Age / each.Life * float64(len(each.Colors)))
    index = max(0, min(index, len(each.Colors) - 1))
    return each.Colors[index]
}

// the place it is drawn at: a whole art pixel
func (each *particle) place() (int, int) {
    return int(math.Floor(each.X)), int(math.Floor(each.Y))
}

// between low and high of a value
func randomPart(low float64, high float64) float64 {
    return low + (high - low) * rand.Float64()
}

// what comes off something in flight: particles at a place with a little speed of their own any way
func (system *particleSystem) emitTrail(x float64, y float64, spread float64, count int, speed float64, gravity float64, life float64, colors []color.RGBA) {
    for range count {
        angle := rand.Float64() * 2 * math.Pi
        own := rand.Float64() * speed

        system.add(particle{
            X: x + randomPart(-spread, spread),
            Y: y + randomPart(-spread, spread),
            SpeedX: math.Cos(angle) * own,
            SpeedY: math.Sin(angle) * own,
            Gravity: gravity,
            Life: life * randomPart(0.6, 1),
            Colors: colors,
        })
    }
}

// what is thrown up where something hits: particles that fly apart over the ground, up first, and
// come down on it. height is how far over the ground they start
func (system *particleSystem) emitBurst(x float64, ground float64, height float64, count int, speed float64, lift float64, gravity float64, life float64, colors []color.RGBA) {
    for range count {
        angle := rand.Float64() * 2 * math.Pi
        own := speed * randomPart(0.3, 1)

        system.add(particle{
            X: x,
            Y: ground - height,
            SpeedX: math.Cos(angle) * own,
            SpeedY: math.Sin(angle) * own * particleGroundSquash - lift * randomPart(0.4, 1),
            Gravity: gravity,
            Life: life * randomPart(0.5, 1),
            Colors: colors,
            HasGround: true,
            // where it comes down lies further up or down the screen by where it flew to
            Ground: ground + math.Sin(angle) * own * particleGroundSquash * life * 0.5,
        })
    }
}
