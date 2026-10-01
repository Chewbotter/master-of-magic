package combat

// The marks spells leave on the ground where they hit. Not in the original (user, 2026-09-28).
//
// A mark lies on the ground, under everything that stands on it, and stays for the whole battle.
//
// The pictures: the game has a first set of its own (the folder decals, made by
// util/decals/make.py), by kind: scorch, crater, frost, spark. The values of a spell name the
// kind (spellvalues.go). Pictures in the replacement folder go before them, by spell:
//
//   spells/<name>/decal_00.png, decal_01.png, ...
//
// One of the pictures of a spell is taken by chance for every mark, and by chance it is turned
// left to right. Pictures of marks can have any size. Their middle lies on the middle of the tile.

import (
    "embed"
    "fmt"
    "image"
    "image/png"
    rand "github.com/kazzmir/master-of-magic/lib/chance"
    "sort"
    "strings"

    "github.com/kazzmir/master-of-magic/game/magic/mod"
    "github.com/kazzmir/master-of-magic/game/magic/scale"

    "github.com/hajimehoshi/ebiten/v2"
)

//go:embed decals/*.png
var decalFiles embed.FS

const decalFolder = "decals"
const DecalNone = "none"
// the name of the files of marks in the folder of a spell
const DecalPrefix = "decal"
// no more marks than this. the oldest go first
const decalMax = 300
// a mark lies this many art pixels from the middle of its tile at most, each way
const decalScatterX = 3
const decalScatterY = 1

type decal struct {
    Picture *ebiten.Image
    // the top left corner on the original's screen
    X int
    Y int
    Flip bool
    Strength float32
}

// the kinds of marks the game has pictures of
func DecalKinds() []string {
    seen := make(map[string]bool)
    entries, _ := decalFiles.ReadDir(decalFolder)
    for _, entry := range entries {
        kind, _, ok := strings.Cut(entry.Name(), "_")
        if ok {
            seen[kind] = true
        }
    }

    var out []string
    for kind := range seen {
        out = append(out, kind)
    }
    sort.Strings(out)
    return out
}

// the pictures the game has of a kind of mark
func DecalPictures(kind string) []image.Image {
    var out []image.Image

    entries, _ := decalFiles.ReadDir(decalFolder)
    for _, entry := range entries {
        if !strings.HasPrefix(entry.Name(), kind + "_") {
            continue
        }

        file, err := decalFiles.Open(decalFolder + "/" + entry.Name())
        if err != nil {
            continue
        }
        picture, err := png.Decode(file)
        file.Close()
        if err == nil {
            out = append(out, picture)
        }
    }

    return out
}

// the pictures of the marks of a spell: the ones of the replacement folder, or the ones of the game
func (combat *CombatScreen) decalPictures(name string, kind string) []*ebiten.Image {
    key := fmt.Sprintf("%v/%v", name, kind)
    if pictures, ok := combat.effects.DecalPictures[key]; ok {
        return pictures
    }

    sources := mod.ReadSpellExtras(name, DecalPrefix)
    if len(sources) == 0 && kind != "" && kind != DecalNone {
        sources = DecalPictures(kind)
    }

    var pictures []*ebiten.Image
    for _, source := range sources {
        pictures = append(pictures, ebiten.NewImageFromImage(source))
    }

    if combat.effects.DecalPictures == nil {
        combat.effects.DecalPictures = make(map[string][]*ebiten.Image)
    }
    combat.effects.DecalPictures[key] = pictures
    return pictures
}

// a spell leaves its mark on a tile
func (combat *CombatScreen) addDecal(name string, values SpellValues, x int, y int) {
    if values.DecalStrength <= 0 {
        return
    }

    pictures := combat.decalPictures(name, values.Decal)
    if len(pictures) == 0 {
        return
    }

    picture := pictures[rand.N(len(pictures))]
    placeX, placeY := spellPlace(x, y)

    effects := &combat.effects
    if len(effects.Decals) >= decalMax {
        effects.Decals = effects.Decals[1:]
    }

    effects.Decals = append(effects.Decals, decal{
        Picture: picture,
        X: placeX - picture.Bounds().Dx() / 2 + rand.N(decalScatterX * 2 + 1) - decalScatterX,
        Y: placeY - picture.Bounds().Dy() / 2 + rand.N(decalScatterY * 2 + 1) - decalScatterY,
        Flip: rand.N(2) == 0,
        Strength: float32(min(1, values.DecalStrength)),
    })
}

// the marks, on the ground and under what stands on it
func (combat *CombatScreen) drawDecals(screen *ebiten.Image) {
    if len(combat.effects.Decals) == 0 {
        return
    }

    matrix := originalScreenMatrix(combat.GetCameraMatrix())

    for _, each := range combat.effects.Decals {
        var options ebiten.DrawImageOptions
        if each.Flip {
            options.GeoM.Scale(-1, 1)
            options.GeoM.Translate(float64(each.Picture.Bounds().Dx()), 0)
        }
        options.GeoM.Translate(float64(each.X), float64(each.Y))
        options.GeoM.Concat(matrix)
        // darker on the ground around the field, as the ground is
        brightness := fieldBrightness(each.X + each.Picture.Bounds().Dx() / 2, each.Y + each.Picture.Bounds().Dy() / 2)
        options.ColorScale.Scale(brightness, brightness, brightness, 1)
        options.ColorScale.ScaleAlpha(each.Strength)
        scale.DrawScaled(screen, each.Picture, &options)
    }
}

// the kind of mark the game has for a spell, empty for none
func GameDecal(name string) string {
    kind := gameValues(name).Decal
    if kind == DecalNone {
        return ""
    }
    return kind
}
