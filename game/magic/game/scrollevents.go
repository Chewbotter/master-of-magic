package game

// ONE SCROLL FOR ALL EVENTS OF A TURN, as the original has it (user, 2026-09-29: "make the
// Chancellor show one scroll with all events"). Upstream showed a scroll for every event, one
// after the other, at the start of a turn and again when the chancellor was asked.
//
// The events are kept as they come (GameEventScroll, a heading and lines). At the start of a
// turn the scroll is shown when no more events wait, with the events that were not shown yet;
// the chancellor shows all of the turn. They are put under their headings in the order of the
// original (ReMoM MoM/src/REPORT.c Chancellor_Screen_Scroll_Draw__WIP), events of one heading
// together. Text that is higher than the scroll can be moved: the wheel of the mouse, the arrow
// keys, U and D, or a click on the arrows the scroll shows then.
//
// How the scroll opens and closes is still the fork's.

import (
    "image"

    "github.com/kazzmir/master-of-magic/game/magic/inputmanager"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/game/magic/util"
    "github.com/kazzmir/master-of-magic/lib/coroutine"
    "github.com/kazzmir/master-of-magic/lib/font"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/inpututil"
)

// false: a scroll for every event
var OneScrollForAllEvents = true

// the headings in the order of the original. others follow in the order they came
var scrollHeadingOrder = []string{
    "RANDOM EVENTS",
    "UNITS DISBANDED",
    "UNITS KILLED",
    "CITY GROWTH",
    "CITY DEATHS",
    "UNIT SPELLS DISSIPATED",
    "CITY SPELLS DISSIPATED",
    "GLOBAL SPELLS DISSIPATED",
}

const (
    // under the last line of a heading
    scrollGroupGap = 2
    // the arrows of a text that is higher than the scroll: pictures of scroll.lbx and places
    scrollArrowUpIndex = 0
    scrollArrowDownIndex = 1
    scrollArrowX = 156
    scrollArrowUpY = 13
    scrollArrowDownY = 172
    // pixels the text moves at once
    scrollMoveStep = scrollLineStep
    // the rows of the screen the text shows in
    scrollTextTop = scrollMiddleY - scrollTextMost / 2
    scrollTextBottom = scrollTextTop + scrollTextMost
)

type scrollGroup struct {
    Title string
    Lines []string
}

// the events under their headings
func (style *scrollStyle) groups(events []*GameEventScroll) []scrollGroup {
    var order []string
    lines := make(map[string][]string)

    add := func(title string) {
        for _, event := range events {
            if event.Title != title {
                continue
            }
            if _, ok := lines[title]; !ok {
                order = append(order, title)
                lines[title] = nil
            }
            lines[title] = append(lines[title], style.lines(event.Text)...)
        }
    }

    for _, title := range scrollHeadingOrder {
        add(title)
    }
    for _, event := range events {
        if _, ok := lines[event.Title]; !ok {
            add(event.Title)
        }
    }

    var out []scrollGroup
    for _, title := range order {
        out = append(out, scrollGroup{Title: title, Lines: lines[title]})
    }
    return out
}

// how high the text of the groups is
func scrollGroupsHeight(groups []scrollGroup) int {
    height := 0
    for _, group := range groups {
        if group.Title != "" {
            height += scrollHeadingHeight
        }
        height += len(group.Lines) * scrollLineStep
    }
    return height
}

// draws the groups, the first at y. what is out of the rows of the text, or out of the rows
// from paperTop to paperBottom the scroll has opened so far, is not drawn
func (style *scrollStyle) drawGroups(screen *ebiten.Image, colors ebiten.ColorScale, groups []scrollGroup, y int, paperTop int, paperBottom int) {
    bounds := screen.Bounds()
    top := max(scrollTextTop, paperTop)
    bottom := min(scrollTextBottom, paperBottom)
    rows := image.Rect(bounds.Min.X, bounds.Min.Y + scale.Scale(top), bounds.Max.X, bounds.Min.Y + scale.Scale(bottom)).Intersect(bounds)
    if rows.Empty() {
        return
    }
    part := screen.SubImage(rows).(*ebiten.Image)

    var options ebiten.DrawImageOptions
    options.ColorScale = colors
    for _, group := range groups {
        if group.Title != "" {
            x := scrollHeadingX
            if group.Title == scrollNoEvents || group.Title == "GLOBAL SPELLS DISSIPATED" {
                x = scrollNoEventsX
            }
            style.Heading.Print(part, x, y, font.FontOptions{Options: &options, Scale: scale.ScaleAmount, Justify: font.FontJustifyCenter}, group.Title)
            y += scrollHeadingStep
        }

        for _, line := range group.Lines {
            style.Line.Print(part, scrollLineX, y, font.FontOptions{Options: &options, Scale: scale.ScaleAmount}, line)
            y += scrollLineStep
        }
        y += scrollGroupGap
    }
}

// the scroll with these events. with none it says so
func (game *Game) showScrollEvents(yield coroutine.YieldFunc, events []*GameEventScroll) {
    style := game.getScrollStyle()
    if style == nil || !OneScrollForAllEvents {
        if len(events) == 0 {
            game.showScroll(yield, scrollNoEvents, "")
        }
        for _, event := range events {
            game.showScroll(yield, event.Title, event.Text)
        }
        return
    }

    groups := style.groups(events)
    if len(groups) == 0 {
        groups = []scrollGroup{{Title: scrollNoEvents}}
    }

    textHeight := scrollGroupsHeight(groups)
    shown := min(textHeight, scrollTextMost)
    // how far the text is moved up, 0 or less
    top := 0
    lowest := min(0, shown - textHeight)

    scrollImages, _ := game.ImageCache.GetImages("scroll.lbx", 2)
    pageBackground, _ := game.ImageCache.GetImage("scroll.lbx", 5, 0)
    arrowUp, _ := game.ImageCache.GetImage("scroll.lbx", scrollArrowUpIndex, 0)
    arrowDown, _ := game.ImageCache.GetImage("scroll.lbx", scrollArrowDownIndex, 0)
    if len(scrollImages) == 0 || pageBackground == nil {
        return
    }

    totalImages := max(3, min(len(scrollImages), (shown + scrollHeadingStep) / 5 + 1))
    scrollAnimation := util.MakeAnimation(scrollImages[:totalImages], false)

    scrollLength := 30
    opened := false
    getAlpha := util.MakeFadeIn(7, &game.Counter)

    drawer := game.LastDrawer()
    game.PushDrawer(func (screen *ebiten.Image){
        drawer(screen)

        // in the middle of the wide map, see wide.go
        game.drawCentered(screen, func (screen *ebiten.Image){
            var options ebiten.DrawImageOptions
            options.ColorScale.ScaleAlpha(getAlpha())
            options.GeoM.Translate(float64(65), float64(25))

            middleY := pageBackground.Bounds().Dy() / 2
            length := min(scrollLength / 2, middleY)
            pagePart := pageBackground.SubImage(image.Rect(0, middleY - length, pageBackground.Bounds().Dx(), middleY + length)).(*ebiten.Image)

            pageOptions := options
            pageOptions.GeoM.Translate(0, float64(middleY - length) + float64(5))
            scale.DrawScaled(screen, pagePart, &pageOptions)

            // the text fades a little more than the rest of the scroll, as upstream has it
            textScale := options.ColorScale
            textScale.ScaleAlpha(getAlpha())
            paperTop := 25 + 5 + middleY - length
            style.drawGroups(screen, textScale, groups, scrollMiddleY - shown / 2 + top, paperTop, paperTop + 2 * length)

            scrollOptions := options
            scrollOptions.GeoM.Translate(float64(-63), float64(-20))
            scale.DrawScaled(screen, scrollAnimation.Frame(), &scrollOptions)

            if opened && arrowUp != nil && top < 0 {
                var arrow ebiten.DrawImageOptions
                arrow.ColorScale.ScaleAlpha(getAlpha())
                arrow.GeoM.Translate(scrollArrowX, scrollArrowUpY)
                scale.DrawScaled(screen, arrowUp, &arrow)
            }
            if opened && arrowDown != nil && top > lowest {
                var arrow ebiten.DrawImageOptions
                arrow.ColorScale.ScaleAlpha(getAlpha())
                arrow.GeoM.Translate(scrollArrowX, scrollArrowDownY)
                scale.DrawScaled(screen, arrowDown, &arrow)
            }
        })
    })
    defer game.PopDrawer()
    game.markOverlayDrawer()

    move := func(by int) {
        top = max(lowest, min(0, top + by))
    }

    // a click on an arrow. the scroll is in the middle of the wide map
    onArrow := func() int {
        if lowest == 0 || arrowUp == nil || arrowDown == nil {
            return 0
        }
        x, y := game.overlayCursorTransform(inputmanager.MousePosition())
        point := image.Pt(scale.Unscale(x), scale.Unscale(y))
        if point.In(util.ImageRect(scrollArrowX, scrollArrowUpY, arrowUp).Inset(-2)) {
            return 1
        }
        if point.In(util.ImageRect(scrollArrowX, scrollArrowDownY, arrowDown).Inset(-2)) {
            return -1
        }
        return 0
    }

    animationSpeed := uint64(6)

    // absorb clicks and key presses
    yield()

    quit := false
    for !quit {
        game.Counter += 1

        _, wheel := inputmanager.Wheel()
        switch {
            case wheel > 0 || inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyU):
                move(scrollMoveStep)
            case wheel < 0 || inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyD):
                move(-scrollMoveStep)
        }

        // while it unrolls a click or a key opens it at once, see scrollopen.go
        closeOrOpen := func() {
            if opened || !ScrollOpensOnClick {
                quit = true
            } else {
                finishScrollOpening(scrollAnimation, &scrollLength)
                opened = true
            }
        }
        if inputmanager.LeftClick() {
            arrow := onArrow()
            if arrow != 0 {
                move(arrow * scrollMoveStep)
            } else {
                closeOrOpen()
            }
        }
        if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
            closeOrOpen()
        }

        if game.Counter % animationSpeed == 0 {
            if scrollAnimation.Next() {
                scrollLength += 10
            } else {
                opened = true
            }
        }

        yield()
    }

    // the scroll closes
    opened = false
    scrollAnimation = util.MakeReverseAnimation(scrollImages[:totalImages], false)
    quit = false
    for !quit {
        game.Counter += 1

        if game.Counter % animationSpeed == 0 {
            if scrollAnimation.Next() {
                scrollLength -= 10
            } else {
                quit = true
            }
        }

        yield()
    }

    scrollLength += 10

    getAlpha = util.MakeFadeOut(7, &game.Counter)
    for range 7 {
        game.Counter += 1
        yield()
    }
}
