package combat

import (
    "math"
    "testing"
)

// how far apart two units at two tiles are on the screen once they have closed in
func apartAfter(x int, y int, otherX int, otherY int) float64 {
    field := MakeBattlefieldMatrix()
    fromX, fromY := field.Apply(float64(x), float64(y))
    toX, toY := field.Apply(float64(otherX), float64(otherY))

    moveX, moveY := approachFor(x, y, otherX, otherY)
    otherMoveX, otherMoveY := approachFor(otherX, otherY, x, y)

    return math.Hypot((toX + otherMoveX) - (fromX + moveX), (toY + otherMoveY) - (fromY + moveY))
}

// units across a corner to the left and right of each other close in, the others stay
func TestApproach(test *testing.T) {
    // tiles that touch with a side
    for _, other := range [][2]int{{11, 10}, {9, 10}, {10, 11}, {10, 9}} {
        moveX, moveY := approachFor(10, 10, other[0], other[1])
        if moveX != 0 || moveY != 0 {
            test.Errorf("toward %v: %v, %v", other, moveX, moveY)
        }
    }

    // with a corner, above and below each other on the screen: near enough
    for _, other := range [][2]int{{11, 9}, {9, 11}} {
        moveX, moveY := approachFor(10, 10, other[0], other[1])
        if moveX != 0 || moveY != 0 {
            test.Errorf("toward %v: %v, %v", other, moveX, moveY)
        }
    }

    // with a corner, left and right of each other: 32 apart, they close in to 18
    for _, other := range [][2]int{{11, 11}, {9, 9}} {
        moveX, moveY := approachFor(10, 10, other[0], other[1])
        if math.Abs(math.Abs(moveX) - 7) > 0.01 || math.Abs(moveY) > 0.01 {
            test.Errorf("toward %v: %v, %v, should be 7 across", other, moveX, moveY)
        }
        if apart := apartAfter(10, 10, other[0], other[1]); math.Abs(apart - approachDistance) > 0.01 {
            test.Errorf("with %v: %v apart once they have closed in", other, apart)
        }
    }

    // toward the other: the one on the left goes right
    moveX, _ := approachFor(10, 10, 11, 11)
    otherX, _ := approachFor(11, 11, 10, 10)
    if moveX * otherX >= 0 {
        test.Errorf("they go %v and %v", moveX, otherX)
    }
}

// a unit closes in as the attack starts, stays, and goes back as it ends
func TestApproachPart(test *testing.T) {
    if approachPart(0, 60, 10) != 0 || approachPart(60, 60, 10) != 0 || approachPart(-1, 60, 10) != 0 {
        test.Errorf("at the ends of the attack")
    }
    if approachPart(10, 60, 10) != 1 || approachPart(30, 60, 10) != 1 || approachPart(50, 60, 10) != 1 {
        test.Errorf("in the attack: %v %v %v", approachPart(10, 60, 10), approachPart(30, 60, 10), approachPart(50, 60, 10))
    }

    last := 0.0
    for tick := 0.0; tick <= 10; tick++ {
        part := approachPart(tick, 60, 10)
        if part < last {
            test.Errorf("closing in it goes back at %v", tick)
        }
        last = part
    }
    for tick := 50.0; tick < 60; tick++ {
        part := approachPart(tick, 60, 10)
        if part > last {
            test.Errorf("going back it closes in at %v", tick)
        }
        last = part
    }

    // an attack that is shorter than the two ways
    if part := approachPart(5, 10, 10); part <= 0 || part > 1 {
        test.Errorf("in a short attack: %v", part)
    }
}
