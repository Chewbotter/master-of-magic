package game

import (
    "bytes"
    "image"
    "testing"
    "time"

    "github.com/kazzmir/master-of-magic/game/magic/data"
)

func testFogPictures() *fogPictures {
    fill := func(red uint8, alpha uint8) fogPicture {
        picture := fogPicture{Pixels: make([]uint8, 4 * 3 * 4), Width: 4, Height: 3}
        for at := 0; at < len(picture.Pixels); at += 4 {
            picture.Pixels[at] = red
            picture.Pixels[at + 3] = alpha
        }
        return picture
    }

    // an edge covers the left column only
    edge := fogPicture{Pixels: make([]uint8, 4 * 3 * 4), Width: 4, Height: 3}
    for y := 0; y < 3; y++ {
        edge.Pixels[y * 4 * 4 + 3] = 255
    }

    // the ones toward the east the right column too
    east := fogPicture{Pixels: bytes.Clone(edge.Pixels), Width: 4, Height: 3}
    for y := 0; y < 3; y++ {
        east.Pixels[(y * 4 + 3) * 4 + 3] = 255
    }

    return &fogPictures{
        Width: 4, Height: 3,
        Black: fill(8, 255),
        EdgeNE: east, EdgeSE: east, EdgeE: east, EdgeSW: edge, EdgeS: edge, EdgeNW: edge, EdgeN: edge, EdgeW: edge,
        CornerNE: edge, CornerSW: edge, CornerSE: edge, CornerNW: edge,
    }
}

func testFogMap(width int, height int) data.FogMap {
    fog := make(data.FogMap, width)
    for x := range fog {
        fog[x] = make([]data.FogType, height)
    }
    return fog
}

func TestFogFade(test *testing.T) {
    pictures := testFogPictures()
    wrapX := func(x int) int {
        return (x % 8 + 8) % 8
    }

    fog := testFogMap(8, 6)
    fog[2][2] = data.FogTypeVisible

    state := makeFogFadeState(fog, true)
    start := time.Now()
    state.update(fog, pictures, wrapX, start)
    if len(state.Tiles) != 0 {
        test.Fatalf("nothing changed, but %v tiles fade", len(state.Tiles))
    }

    // reveal the tile to the right
    before := fogLook(fog, pictures, wrapX, 3, 2)
    beforeNeighbor := fogLook(fog, pictures, wrapX, 2, 2)
    fog[3][2] = data.FogTypeVisible
    state.update(fog, pictures, wrapX, start)

    revealed, ok := state.Tiles[image.Point{X: 3, Y: 2}]
    if !ok {
        test.Fatalf("the revealed tile does not fade")
    }

    // the tile next to it loses its edge toward the revealed tile
    neighbor, ok := state.Tiles[image.Point{X: 2, Y: 2}]
    if !ok {
        test.Fatalf("the tile next to the revealed tile does not fade")
    }
    if !bytes.Equal(neighbor.From, beforeNeighbor) {
        test.Errorf("the tile next to the revealed tile does not start from what it showed")
    }

    // a tile that shows the same as before does not fade
    if _, ok := state.Tiles[image.Point{X: 6, Y: 4}]; ok {
        test.Errorf("a tile far away fades")
    }

    out := make([]uint8, len(revealed.To))
    revealed.shown(0, out)
    if !bytes.Equal(out, before) {
        test.Errorf("the fade does not start from what the tile showed")
    }

    revealed.shown(fogFadeLength(), out)
    if !bytes.Equal(out, fogLook(fog, pictures, wrapX, 3, 2)) {
        test.Errorf("the fade does not end at what the tile shows")
    }

    // in between, pixels are at different places of their fade
    revealed.shown(fogFadeLength() / 2, out)
    values := make(map[uint8]bool)
    for at := 3; at < len(out); at += 4 {
        values[out[at]] = true
    }
    if FogFadeSpread > 0 && len(values) < 2 {
        test.Errorf("all pixels are equally far half way through: %v", values)
    }

    // a change next to a fading tile that does not change what it fades to leaves it alone
    fog[5][2] = data.FogTypeVisible
    state.update(fog, pictures, wrapX, start.Add(time.Millisecond))
    if state.Tiles[image.Point{X: 3, Y: 2}] != revealed {
        test.Errorf("the fade of a tile started again without a change of what it shows")
    }

    // done
    state.update(fog, pictures, wrapX, start.Add(time.Duration((fogFadeLength() + 0.1) * float64(time.Second))))
    if len(state.Tiles) != 0 {
        test.Errorf("%v tiles still fade after the end", len(state.Tiles))
    }
}
