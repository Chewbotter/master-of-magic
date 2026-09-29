package game

// Fog of war that fades.
//
// DrawFog draws the fog of a tile from the fog map as it is, so a tile that is revealed changes
// in one frame. Here the change is spread over time, one art pixel at a time: every art pixel of
// a tile waits for a time of its own (by chance, up to FogFadeSpread) and then fades from what it
// showed to what it shows now (in FogFadePixelTime).
//
// How: the fog map is compared with a copy of the last frame. For a tile that changed, and its 8
// neighbors (their edges toward it change too), the fog is put together here as DrawFog would draw
// it, once for the map of before and once for the map of now. While a tile fades it is drawn from
// these two pictures; when it is done DrawFog draws it again as always.
//
// KEEP IN STEP with DrawFog in game.go: fogLook is the same drawing in memory.

import (
    "bytes"
    "log"
    "image"
    "time"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
)

// false: the fog changes at once, as before
var FogFade = true
// seconds over which the art pixels of a tile start to fade, each at a time of its own.
// 0: all pixels of a tile fade together
const FogFadeSpread = 0.22
// seconds one art pixel takes to fade. 0: a pixel changes at once
const FogFadePixelTime = 0.12
// false: only fog that gets lighter fades, fog that gets darker (a tile that is no longer seen) changes at once
const FogFadeWhenCovered = true
// development: 0 or more holds every fade at this many seconds after its start, and the first
// look at a map counts as revealed from nothing, so a capture shows the fade. see -capture-fog-fade
var FogFadeProbe float64 = -1

// development: logs the number of fading tiles every frame they are drawn
var FogFadeTrace = false

// how much fog of the last frame and of tiles that fade is kept before it is thrown away
const fogFadeMapsKept = 8
// tiles in a row of the picture the fading tiles are drawn from
const fogFadeAtlasColumns = 32

// a picture in memory, 4 bytes per pixel, colors multiplied by alpha
type fogPicture struct {
    Pixels []uint8
    Width int
    Height int
}

// the pictures DrawFog uses
type fogPictures struct {
    // size of a tile
    Width int
    Height int
    Black fogPicture
    EdgeNE, EdgeSE, EdgeE, EdgeSW, EdgeS, EdgeNW, EdgeN, EdgeW fogPicture
    CornerNE, CornerSW, CornerSE, CornerNW fogPicture
}

// draws a picture over the look of a tile, with the alpha scaled as ColorScale.Scale(1, 1, 1, alpha) does
func (picture *fogPicture) over(look []uint8, width int, height int, alpha float64) {
    for y := 0; y < min(height, picture.Height); y++ {
        for x := 0; x < min(width, picture.Width); x++ {
            from := (y * picture.Width + x) * 4
            to := (y * width + x) * 4

            sourceAlpha := float64(picture.Pixels[from + 3]) * alpha
            if sourceAlpha <= 0 {
                continue
            }
            keep := 1 - sourceAlpha / 255

            for channel := 0; channel < 3; channel++ {
                look[to + channel] = uint8(min(255, float64(picture.Pixels[from + channel]) + float64(look[to + channel]) * keep + 0.5))
            }
            look[to + 3] = uint8(min(255, sourceAlpha + float64(look[to + 3]) * keep + 0.5))
        }
    }
}

// the fog of a tile as DrawFog draws it
func fogLook(fog data.FogMap, pictures *fogPictures, wrapX func(int) int, tileX int, tileY int) []uint8 {
    look := make([]uint8, pictures.Width * pictures.Height * 4)

    if tileX < 0 || tileX >= len(fog) || tileY < 0 || tileY >= len(fog[tileX]) {
        return look
    }

    is := func(deltaX int, deltaY int, fogType data.FogType) bool {
        x := wrapX(tileX + deltaX)
        y := tileY + deltaY
        if x < 0 || x >= len(fog) || y < 0 || y >= len(fog[x]) {
            return false
        }

        return fog[x][y] == fogType
    }

    draw := func(picture *fogPicture, alpha float64) {
        picture.over(look, pictures.Width, pictures.Height, alpha)
    }

    border := func(fogType data.FogType, alpha float64) {
        n := is(0, -1, fogType)
        e := is(1, 0, fogType)
        s := is(0, 1, fogType)
        w := is(-1, 0, fogType)
        ne := is(1, -1, fogType)
        se := is(1, 1, fogType)
        nw := is(-1, -1, fogType)
        sw := is(-1, 1, fogType)

        if n && e {
            draw(&pictures.EdgeNE, alpha)
        } else if n {
            draw(&pictures.EdgeN, alpha)
        } else if e {
            draw(&pictures.EdgeE, alpha)
        } else if ne {
            draw(&pictures.CornerNE, alpha)
        }

        if s && e {
            draw(&pictures.EdgeSE, alpha)
        } else if s {
            draw(&pictures.EdgeS, alpha)
        } else if se {
            draw(&pictures.CornerSE, alpha)
        }

        if n && w {
            draw(&pictures.EdgeNW, alpha)
        } else if w {
            draw(&pictures.EdgeW, alpha)
        } else if nw {
            draw(&pictures.CornerNW, alpha)
        }

        if s && w {
            draw(&pictures.EdgeSW, alpha)
        } else if sw {
            draw(&pictures.CornerSW, alpha)
        }
    }

    switch fog[tileX][tileY] {
        case data.FogTypeUnexplored:
            draw(&pictures.Black, 1)
        case data.FogTypeExplored:
            if !data.ClassicFog {
                draw(&pictures.Black, 0.5)
            }
            border(data.FogTypeUnexplored, 1)
        case data.FogTypeVisible:
            if !data.ClassicFog {
                border(data.FogTypeExplored, 0.3)
            }
            border(data.FogTypeUnexplored, 1)
    }

    return look
}

// a tile whose fog is changing
type fogFadeTile struct {
    From []uint8
    To []uint8
    Start time.Time
    // makes the order of the pixels different for every tile and every change
    Salt uint32
}

// a number from 0 up to 1 by chance, the same for the same pixel of the same change
func fogFadeChance(pixel int, salt uint32) float64 {
    value := uint32(pixel) * 2654435761 + salt
    value ^= value >> 15
    value *= 2246822519
    value ^= value >> 13
    value *= 3266489917
    value ^= value >> 16
    return float64(value >> 8) / float64(1 << 24)
}

// seconds a change takes from its start to its last pixel
func fogFadeLength() float64 {
    return FogFadeSpread + FogFadePixelTime
}

// how far the given pixel is, 0 to 1, this many seconds after the start
func fogFadeProgress(pixel int, salt uint32, elapsed float64) float64 {
    since := elapsed - fogFadeChance(pixel, salt) * FogFadeSpread
    if since < 0 {
        return 0
    }
    if since >= FogFadePixelTime {
        return 1
    }

    progress := since / FogFadePixelTime
    // eased in and out
    return progress * progress * (3 - 2 * progress)
}

// what the tile shows this many seconds after the start, written to out
func (tile *fogFadeTile) shown(elapsed float64, out []uint8) {
    for pixel := 0; pixel < len(tile.To) / 4; pixel++ {
        at := pixel * 4
        from := tile.From[at:at + 4]
        to := tile.To[at:at + 4]

        if from[0] == to[0] && from[1] == to[1] && from[2] == to[2] && from[3] == to[3] {
            copy(out[at:at + 4], to)
            continue
        }

        progress := fogFadeProgress(pixel, tile.Salt, elapsed)
        for channel := 0; channel < 4; channel++ {
            out[at + channel] = uint8(float64(from[channel]) + (float64(to[channel]) - float64(from[channel])) * progress + 0.5)
        }
    }
}

// the fades of one fog map
type fogFadeState struct {
    // the map of the last frame
    Seen data.FogMap
    Tiles map[image.Point]*fogFadeTile
    changes uint32
}

func copyFogMap(fog data.FogMap) data.FogMap {
    out := make(data.FogMap, len(fog))
    for x := range fog {
        out[x] = make([]data.FogType, len(fog[x]))
        copy(out[x], fog[x])
    }
    return out
}

// revealed is false for a map that counts as unexplored all over before its first frame
func makeFogFadeState(fog data.FogMap, revealed bool) *fogFadeState {
    state := &fogFadeState{
        Seen: copyFogMap(fog),
        Tiles: make(map[image.Point]*fogFadeTile),
    }

    if !revealed {
        for x := range state.Seen {
            for y := range state.Seen[x] {
                state.Seen[x][y] = data.FogTypeUnexplored
            }
        }
    }

    return state
}

func (state *fogFadeState) elapsed(tile *fogFadeTile, now time.Time) float64 {
    if FogFadeProbe >= 0 {
        return FogFadeProbe
    }

    return now.Sub(tile.Start).Seconds()
}

// starts a fade for every tile that shows something else than in the last frame, and ends the fades that are done
func (state *fogFadeState) update(fog data.FogMap, pictures *fogPictures, wrapX func(int) int, now time.Time) {
    for key, tile := range state.Tiles {
        if state.elapsed(tile, now) >= fogFadeLength() {
            delete(state.Tiles, key)
        }
    }

    if len(fog) != len(state.Seen) {
        state.Seen = copyFogMap(fog)
        state.Tiles = make(map[image.Point]*fogFadeTile)
        return
    }

    // the tiles that changed and the tiles around them
    var touched map[image.Point]struct{}
    for x := range fog {
        if len(fog[x]) != len(state.Seen[x]) {
            state.Seen = copyFogMap(fog)
            state.Tiles = make(map[image.Point]*fogFadeTile)
            return
        }

        for y := range fog[x] {
            if fog[x][y] == state.Seen[x][y] {
                continue
            }

            if touched == nil {
                touched = make(map[image.Point]struct{})
            }

            for deltaX := -1; deltaX <= 1; deltaX++ {
                for deltaY := -1; deltaY <= 1; deltaY++ {
                    nearX := wrapX(x + deltaX)
                    nearY := y + deltaY
                    if nearX >= 0 && nearX < len(fog) && nearY >= 0 && nearY < len(fog[nearX]) {
                        touched[image.Point{X: nearX, Y: nearY}] = struct{}{}
                    }
                }
            }
        }
    }

    if len(touched) == 0 {
        return
    }

    if FogFadeTrace {
        log.Printf("fog fade: the fog changed, %v tiles touched", len(touched))
    }

    // what they show now, by the map of before
    shown := make(map[image.Point][]uint8, len(touched))
    for key := range touched {
        tile, fading := state.Tiles[key]
        if fading {
            out := make([]uint8, len(tile.To))
            tile.shown(state.elapsed(tile, now), out)
            shown[key] = out
        } else {
            shown[key] = fogLook(state.Seen, pictures, wrapX, key.X, key.Y)
        }
    }

    for x := range fog {
        copy(state.Seen[x], fog[x])
    }

    for key := range touched {
        to := fogLook(state.Seen, pictures, wrapX, key.X, key.Y)

        tile, fading := state.Tiles[key]
        if fading && bytes.Equal(tile.To, to) {
            // already on its way to this
            continue
        }

        from := shown[key]

        if !FogFadeWhenCovered {
            for at := 0; at < len(to); at += 4 {
                if to[at + 3] > from[at + 3] {
                    copy(from[at:at + 4], to[at:at + 4])
                }
            }
        }

        if bytes.Equal(from, to) {
            delete(state.Tiles, key)
            continue
        }

        state.changes += 1
        state.Tiles[key] = &fogFadeTile{
            From: from,
            To: to,
            Start: now,
            Salt: state.changes * 7919 + uint32(key.X) * 131 + uint32(key.Y) * 8191,
        }
    }
}

// the fades of every fog map that was drawn, by the first tile of the map
var fogFades = make(map[*data.FogType]*fogFadeState)

// the pixels of the pictures of the fog, by picture
var fogPixels = make(map[*ebiten.Image]fogPicture)

var fogFadeAtlas *ebiten.Image
var fogFadeBuffer []uint8

func readFogPicture(picture *ebiten.Image) fogPicture {
    if picture == nil {
        return fogPicture{}
    }

    read, ok := fogPixels[picture]
    if ok {
        return read
    }

    if len(fogPixels) > 64 {
        // pictures of image caches that are gone
        clear(fogPixels)
    }

    bounds := picture.Bounds()
    read = fogPicture{
        Pixels: make([]uint8, bounds.Dx() * bounds.Dy() * 4),
        Width: bounds.Dx(),
        Height: bounds.Dy(),
    }
    picture.ReadPixels(read.Pixels)
    fogPixels[picture] = read
    return read
}

func (overworld *Overworld) fogPictures() *fogPictures {
    edge := func(index int) fogPicture {
        picture, err := overworld.ImageCache.GetImage("mapback.lbx", index, 0)
        if err != nil {
            return fogPicture{}
        }
        return readFogPicture(picture)
    }

    // the numbers are the ones of DrawFog
    return &fogPictures{
        Width: overworld.Map.TileWidth(),
        Height: overworld.Map.TileHeight(),
        Black: readFogPicture(overworld.FogBlack),
        EdgeNE: edge(0),
        EdgeSE: edge(1),
        EdgeE: edge(2),
        EdgeSW: edge(3),
        EdgeS: edge(5),
        EdgeNW: edge(7),
        EdgeN: edge(8),
        EdgeW: edge(11),
        CornerNE: edge(10),
        CornerSW: edge(13),
        CornerSE: edge(6),
        CornerNW: edge(12),
    }
}

// the fading tiles of one frame, ready to be drawn
type fogFadeFrame struct {
    Pictures map[image.Point]*ebiten.Image
}

// called by DrawFog before it draws. gives the tiles of the view that are fading
func (overworld *Overworld) fogFadeFrame() fogFadeFrame {
    fog := overworld.Fog
    if !FogFade || len(fog) == 0 || len(fog[0]) == 0 || overworld.ImageCache == nil {
        return fogFadeFrame{}
    }

    pictures := overworld.fogPictures()
    now := time.Now()

    state, ok := fogFades[&fog[0][0]]
    if !ok {
        if len(fogFades) >= fogFadeMapsKept {
            // maps of games that are over
            clear(fogFades)
        }
        state = makeFogFadeState(fog, FogFadeProbe < 0)
        fogFades[&fog[0][0]] = state
    }

    state.update(fog, pictures, overworld.Map.WrapX, now)

    if len(state.Tiles) == 0 {
        return fogFadeFrame{}
    }

    // the fading tiles of the view
    minX, minY, maxX, maxY := overworld.Camera.GetTileBounds()
    var keys []image.Point
    placed := make(map[image.Point]struct{})
    for x := minX; x < maxX; x++ {
        for y := minY; y < maxY; y++ {
            key := image.Point{X: overworld.Map.WrapX(x), Y: y}
            _, fading := state.Tiles[key]
            _, done := placed[key]
            if fading && !done {
                placed[key] = struct{}{}
                keys = append(keys, key)
            }
        }
    }

    if len(keys) == 0 {
        return fogFadeFrame{}
    }
    if FogFadeTrace {
        log.Printf("fog fade: %v tiles fading in view, %v in all", len(keys), len(state.Tiles))
    }

    // all of them next to each other on one picture, one pixel per art pixel
    width := pictures.Width
    height := pictures.Height
    columns := min(len(keys), fogFadeAtlasColumns)
    rows := (len(keys) + fogFadeAtlasColumns - 1) / fogFadeAtlasColumns
    atlasWidth := columns * width
    atlasHeight := rows * height

    if fogFadeAtlas == nil || fogFadeAtlas.Bounds().Dx() < fogFadeAtlasColumns * width || fogFadeAtlas.Bounds().Dy() < atlasHeight {
        if fogFadeAtlas != nil {
            fogFadeAtlas.Deallocate()
        }
        fogFadeAtlas = ebiten.NewImage(fogFadeAtlasColumns * width, atlasHeight)
    }

    if len(fogFadeBuffer) < atlasWidth * atlasHeight * 4 {
        fogFadeBuffer = make([]uint8, atlasWidth * atlasHeight * 4)
    }
    buffer := fogFadeBuffer[:atlasWidth * atlasHeight * 4]
    clear(buffer)

    frame := fogFadeFrame{
        Pictures: make(map[image.Point]*ebiten.Image, len(keys)),
    }

    tilePixels := make([]uint8, width * height * 4)
    for index, key := range keys {
        tile := state.Tiles[key]
        tile.shown(state.elapsed(tile, now), tilePixels)

        left := (index % fogFadeAtlasColumns) * width
        top := (index / fogFadeAtlasColumns) * height
        for y := 0; y < height; y++ {
            copy(buffer[((top + y) * atlasWidth + left) * 4:], tilePixels[y * width * 4:(y + 1) * width * 4])
        }

        frame.Pictures[key] = fogFadeAtlas.SubImage(image.Rect(left, top, left + width, top + height)).(*ebiten.Image)
    }

    fogFadeAtlas.SubImage(image.Rect(0, 0, atlasWidth, atlasHeight)).(*ebiten.Image).WritePixels(buffer)

    return frame
}

// draws the tile if its fog is fading. options hold the place of the tile, as for the pictures of DrawFog
func (frame *fogFadeFrame) DrawTile(screen *ebiten.Image, tileX int, tileY int, options *ebiten.DrawImageOptions) bool {
    picture, ok := frame.Pictures[image.Point{X: tileX, Y: tileY}]
    if !ok {
        return false
    }

    options.ColorScale = ebiten.ColorScale{}
    scale.DrawScaled(screen, picture, options)
    return true
}
