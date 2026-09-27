package ui

// Support for drawing a ui in separate passes, used by the widescreen world map
// to place the base hud and the popups on top of it at different positions.

import (
    "image"

    "github.com/hajimehoshi/ebiten/v2"
)

// same as StandardDraw, but only for the elements on the layers accepted by include, and without the tooltip
func (ui *UI) DrawLayers(screen *ebiten.Image, include func(layer UILayer) bool) {
    ui.IterateElementsByLayer(func (element *UIElement){
        if element.Draw != nil && include(element.Layer) {
            var offsetPoint image.Point
            if element.IsOffsetWhenPressed && element.isPressed {
                offsetPoint = image.Pt(1, 1)
            }

            element.Rect = element.Rect.Add(offsetPoint)
            element.Draw(element, screen)
            element.Rect = element.Rect.Sub(offsetPoint)
        }
    })
}
