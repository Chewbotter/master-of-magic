package camera

import (
    "math"
    "testing"
)

// the columns the camera says to draw must cover every column the view shows, also when the view
// reaches past column 0 on its way around the world and sits part way into a tile
func TestTileBoundsCoverTheLeftEdge(test *testing.T) {
    for _, cameraX := range []int{0, 1, 3, 7, 12, 40} {
        for _, fraction := range []float64{0, 0.1, 0.3, 0.5, 0.7, 0.9} {
            for _, zoom := range []int{ZoomMax, ZoomMax * 4 / 5, ZoomMax * 3 / 5} {
                for _, sizeX := range []int{12, 14, 18} {
                    camera := MakeCameraAt(cameraX, 10)
                    camera.Zoom = zoom
                    camera.SizeX = sizeX
                    camera.SetOffset(fraction, 0)

                    minX, _, _, _ := camera.GetTileBounds()
                    leftmostShown := int(math.Floor(camera.GetZoomedX()))
                    if minX > leftmostShown {
                        test.Fatalf("camera %v+%v zoom %v size %v: drawing starts at column %v, the view shows column %v",
                            cameraX, fraction, zoom, sizeX, minX, leftmostShown)
                    }
                }
            }
        }
    }
}
