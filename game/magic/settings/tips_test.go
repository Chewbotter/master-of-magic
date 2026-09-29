package settings

import (
    "image"
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
)

// the panel shows after the mouse has rested, and not after it has left
func TestSettingsTipShows(test *testing.T) {
    var tip settingsTipState

    for tick := int64(100); tick <= 100 + SettingsTipDelay; tick++ {
        tip.hover("Widescreen", image.Pt(10, 10), tick)
        if tip.shows(tick) != 0 {
            test.Errorf("it shows %v after %v ticks", tip.shows(tick), tick - 100)
        }
    }

    last := int64(100 + SettingsTipDelay + SettingsTipFade)
    for tick := int64(100 + SettingsTipDelay + 1); tick <= last; tick++ {
        tip.hover("Widescreen", image.Pt(10, 10), tick)
    }
    if tip.shows(last) != 1 {
        test.Errorf("it shows %v after the wait and the fade", tip.shows(last))
    }

    // the mouse is gone
    if tip.shows(last + 2) != 0 {
        test.Errorf("it shows %v without the mouse", tip.shows(last + 2))
    }

    // it comes back: the wait starts again
    tip.hover("Widescreen", image.Pt(10, 10), last + 10)
    if tip.shows(last + 10) != 0 {
        test.Errorf("it shows at once when the mouse comes back")
    }

    // another setting: the wait starts again
    tip.hover("Fullscreen", image.Pt(10, 10), last + 11)
    tip.hover("Fullscreen", image.Pt(10, 10), last + 12)
    if tip.shows(last + 12) != 0 {
        test.Errorf("it shows at once on another setting")
    }
}

// the panel stays on the screen
func TestSettingsTipPlace(test *testing.T) {
    for _, mouse := range []image.Point{image.Pt(0, 0), image.Pt(319, 0), image.Pt(0, 199), image.Pt(319, 199), image.Pt(160, 100), image.Pt(250, 190)} {
        place := settingsTipPlace(mouse, 116, 40)
        panel := image.Rect(place.X, place.Y, place.X + 116, place.Y + 40)
        if !panel.In(image.Rect(0, 0, data.ScreenWidth, data.ScreenHeight)) {
            test.Errorf("mouse at %v: the panel %v leaves the screen", mouse, panel)
        }
        if mouse.In(panel) && mouse.Y > 60 && mouse.Y < 140 {
            test.Errorf("mouse at %v is under the panel %v", mouse, panel)
        }
    }
}

// every text belongs to a setting of the screen
func TestSettingTipsHaveText(test *testing.T) {
    for name, text := range settingTips {
        if name == "" || text == "" {
            test.Errorf("an empty entry: '%v' '%v'", name, text)
        }
    }
}
