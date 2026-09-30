package inputmanager

import (
    "log"

    "github.com/kazzmir/master-of-magic/game/magic/display"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/inpututil"
)

type InputManager struct {
    lastTouchX int
    lastTouchY int
    mouseX int
    mouseY int
    // how far right and down the screen on top was drawn when the mouse was read, see
    // MouseOnScreen
    offsetX int
    offsetY int
    Counter uint64
    touchStartTime uint64

    leftClick bool
    leftClickReleased bool
    rightClick bool
}

var theInputManager *InputManager
var updated bool

func NewInputManager() *InputManager {
    return &InputManager{}
}

func init(){
    theInputManager = NewInputManager()
}

func (manager *InputManager) Update() {
    manager.Counter += 1

    manager.leftClick = false
    manager.leftClickReleased = false
    manager.rightClick = false

    manager.offsetX, manager.offsetY = display.ContentOffsetX(), display.ContentOffsetY()
    if manager.lastTouchX == 0 && manager.lastTouchY == 0 {
        manager.mouseX, manager.mouseY = display.CursorPosition()
    }

    if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
        manager.leftClick = true
    }

    if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
        manager.leftClickReleased = true
    }

    if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
        manager.rightClick = true
    }

    pressedTouchIds := inpututil.AppendJustPressedTouchIDs(nil)
    if len(pressedTouchIds) > 0 {
        touchId := pressedTouchIds[0]
        manager.lastTouchX, manager.lastTouchY = display.AdjustPosition(ebiten.TouchPosition(touchId))
        manager.touchStartTime = manager.Counter
    }

    touchIds := inpututil.AppendJustReleasedTouchIDs(nil)
    if len(touchIds) > 0 {
        // touchId := touchIds[0]

        duration := manager.Counter - manager.touchStartTime

        // log.Printf("Touch %v duration %v", touchId, duration)

        if duration < 40 {
            manager.leftClick = true
            manager.leftClickReleased = true
        } else {
            manager.rightClick = true
        }

        manager.mouseX, manager.mouseY = manager.lastTouchX, manager.lastTouchY
        // log.Printf("Touch %v %v %v", touchId, mouseX, mouseY)
        // log.Printf("Touches %v", touchIds)
    }

}

func Update() {
    updated = true
    theInputManager.Update()
}

// development: a game without a window (-sim) has no input: nothing is pressed and the mouse is
// far off the screen
func NoInput() {
    updated = true
    FixedMouse = true
    FixedMouseX = -10000
    FixedMouseY = -10000
}

func LeftClick() bool {
    if !updated {
        log.Fatal("InputManager.Update() not called")
    }

    return theInputManager.leftClick
}

func LeftClickReleased() bool {
    if !updated {
        log.Fatal("InputManager.Update() not called")
    }

    return theInputManager.leftClickReleased
}

func RightClick() bool {
    if !updated {
        log.Fatal("InputManager.Update() not called")
    }

    return theInputManager.rightClick
}

func MousePosition() (int, int) {
    if !updated {
        log.Fatal("InputManager.Update() not called")
    }

    if FixedMouse {
        return FixedMouseX, FixedMouseY
    }

    return theInputManager.mouseX, theInputManager.mouseY
}

// where the mouse is in the window, for drawing the cursor. the position of MousePosition is
// the one the screen on top understands, which is without the black bars of a screen that has no
// wide layout; it was read at the last tick. the cursor was drawn at that position plus the bars
// of the moment of the DRAW, so when a screen with bars closed over the wide map (or opened) the
// cursor was a bar's width off until the next tick (user, 2026-09-29: "the cursor changes
// position suddenly after closing the research book")
func MouseOffset() (int, int) {
    if !updated || FixedMouse {
        return display.ContentOffsetX(), display.ContentOffsetY()
    }

    return theInputManager.offsetX, theInputManager.offsetY
}

// return normalized wheel values, which are always -1,0,1 for both x and y, no matter how far the wheel has moved
func Wheel() (float64, float64) {
    x, y := ebiten.Wheel()
    if x > 0 {
        x = min(x, 1)
    }
    if x < 0 {
        x = max(x, -1)
    }
    if y > 0 {
        y = min(y, 1)
    }
    if y < 0 {
        y = max(y, -1)
    }

    return x, y
}
