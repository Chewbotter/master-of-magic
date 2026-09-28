package main

// Writes the pictures of the battles out of the game's own data as png files, to look at, change
// and add to.
//
// The pictures are the property of the game's rights holder. They are written to a folder OUTSIDE
// of this repository, for the use of the owner of the game. Never commit them, never pass them on.
//
//   go run ./util/export-sprites -data <folder of the game's LBX files> -out <folder>
//
// What is written:
//   units/<race> <name>/<facing>_<frame>.png   the figure of every unit, 8 facings of 4 frames
//   units/<race> <name>/_sheet.png             all of them on one picture, facings down, frames across
//   archives/<file>/<entry>_<frame>.png        every picture of the archives battles draw from
//   environment/<set>/<name>_<frame>.png       ground, trees, towns, walls, lairs, by name
//   palette.png, palette.gpl                   the colors of the game
//   README.txt
//
// The png files keep the numbers of the colors (indexed color), which the game needs: a picture that
// is changed has to stay in indexed color with the same palette.

import (
    "flag"
    "fmt"
    "image"
    "image/color"
    "image/png"
    "log"
    "os"
    "path/filepath"
    "strings"

    "github.com/kazzmir/master-of-magic/game/magic/combat"
    "github.com/kazzmir/master-of-magic/game/magic/mod"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    "github.com/kazzmir/master-of-magic/lib/lbx"
    "github.com/kazzmir/master-of-magic/lib/mouse"
)

// the archives battles draw from, besides the figures of the units
var battleArchives = []string{
    "cmbgrass.lbx", "cmbgrasc.lbx", "cmbdesrt.lbx", "cmbdesrc.lbx", "cmbmount.lbx", "cmbmounc.lbx",
    "cmbtundr.lbx", "cmbtundc.lbx", "cmbtcity.lbx", "citywall.lbx", "wallrise.lbx", "cmbtwall.lbx",
    "cmbtfx.lbx", "cmbmagic.lbx", "chriver.lbx", "compix.lbx", "specfx.lbx", "resource.lbx",
}

var facingNames = mod.FacingNames

const facings = 8
// squares of the picture of the palette, and their size in pixels
const paletteColumns = 16
const paletteSquare = 12

var archives = make(map[string]*lbx.LbxFile)

func openArchive(dataPath string, name string) (*lbx.LbxFile, error) {
    if file, ok := archives[name]; ok {
        return file, nil
    }

    entries, err := os.ReadDir(dataPath)
    if err != nil {
        return nil, err
    }

    for _, entry := range entries {
        if strings.EqualFold(entry.Name(), name) {
            reader, err := os.Open(filepath.Join(dataPath, entry.Name()))
            if err != nil {
                return nil, err
            }
            defer reader.Close()

            file, err := lbx.ReadLbx(reader)
            if err != nil {
                return nil, err
            }

            archives[name] = &file
            return &file, nil
        }
    }

    return nil, fmt.Errorf("no file %v in %v", name, dataPath)
}

func writePng(path string, picture image.Image) error {
    err := os.MkdirAll(filepath.Dir(path), 0755)
    if err != nil {
        return err
    }

    out, err := os.Create(path)
    if err != nil {
        return err
    }
    defer out.Close()

    return png.Encode(out, picture)
}

// all pictures on one, rows of frames
func makeSheet(rows [][]*image.Paletted) *image.Paletted {
    width, height, columns := 0, 0, 0
    var palette color.Palette

    for _, row := range rows {
        columns = max(columns, len(row))
        for _, picture := range row {
            width = max(width, picture.Bounds().Dx())
            height = max(height, picture.Bounds().Dy())
            palette = picture.Palette
        }
    }

    if palette == nil {
        return nil
    }

    sheet := image.NewPaletted(image.Rect(0, 0, width * columns, height * len(rows)), palette)
    for rowIndex, row := range rows {
        for column, picture := range row {
            bounds := picture.Bounds()
            for y := 0; y < bounds.Dy(); y++ {
                for x := 0; x < bounds.Dx(); x++ {
                    sheet.SetColorIndex(column * width + x, rowIndex * height + y, picture.ColorIndexAt(bounds.Min.X + x, bounds.Min.Y + y))
                }
            }
        }
    }

    return sheet
}

func exportUnits(dataPath string, outPath string) (int, int) {
    unitCount := 0
    pictureCount := 0
    seen := make(map[string]bool)

    for _, unit := range units.AllUnits {
        if unit.CombatLbxFile == "" {
            continue
        }

        key := fmt.Sprintf("%v/%v", unit.CombatLbxFile, unit.CombatIndex)
        if seen[key] {
            continue
        }
        seen[key] = true

        archive, err := openArchive(dataPath, unit.CombatLbxFile)
        if err != nil {
            log.Printf("%v: %v", unit.Name, err)
            continue
        }

        // the name the game looks for in the replacement folder, see game/magic/mod
        folder := filepath.Join(outPath, "units", mod.FolderName(fmt.Sprintf("%v %v", unit.Race, unit.Name)))

        var rows [][]*image.Paletted
        for facing := range facings {
            frames, err := archive.ReadImages(unit.CombatIndex + facing)
            if err != nil {
                log.Printf("%v facing %v: %v", unit.Name, facingNames[facing], err)
                continue
            }

            rows = append(rows, frames)
            for frame, picture := range frames {
                err := writePng(filepath.Join(folder, fmt.Sprintf("%v_%v.png", facingNames[facing], frame)), picture)
                if err != nil {
                    log.Printf("%v: %v", unit.Name, err)
                    continue
                }
                pictureCount += 1
            }
        }

        sheet := makeSheet(rows)
        if sheet != nil {
            writePng(filepath.Join(folder, "_sheet.png"), sheet)
        }

        // where the figure comes from, for putting a changed one back
        note := fmt.Sprintf("%v\nrace: %v\narchive: %v\nentries: %v to %v, one per facing in the order %v\n", unit.Name, unit.Race, unit.CombatLbxFile, unit.CombatIndex, unit.CombatIndex + facings - 1, strings.Join(facingNames, ", "))
        os.WriteFile(filepath.Join(folder, "_source.txt"), []byte(note), 0644)

        unitCount += 1
    }

    return unitCount, pictureCount
}

func exportArchives(dataPath string, outPath string) int {
    pictureCount := 0

    for _, name := range battleArchives {
        archive, err := openArchive(dataPath, name)
        if err != nil {
            log.Printf("%v", err)
            continue
        }

        folder := filepath.Join(outPath, "archives", strings.TrimSuffix(name, ".lbx"))

        for entry := range archive.TotalEntries() {
            frames, err := readPictures(archive, entry)
            if err != nil {
                // not every entry of an archive is a picture
                continue
            }

            for frame, picture := range frames {
                err := writePng(filepath.Join(folder, fmt.Sprintf("%03d_%v.png", entry, frame)), picture)
                if err == nil {
                    pictureCount += 1
                }
            }
        }
    }

    return pictureCount
}

// an entry that is not a picture can make the reader give up badly
func readPictures(archive *lbx.LbxFile, entry int) (frames []*image.Paletted, err error) {
    defer func() {
        if recover() != nil {
            frames = nil
            err = fmt.Errorf("not a picture")
        }
    }()

    return archive.ReadImages(entry)
}

// the pictures of the spells of battles, in folders with their names. the same pictures as in
// archives/, easier to find
func exportSpells(dataPath string, outPath string) int {
    count := 0
    archives := make(map[string]*lbx.LbxFile)

    for _, spell := range mod.SpellPictures {
        archive, ok := archives[spell.Archive]
        if !ok {
            opened, err := openArchive(dataPath, spell.Archive)
            if err != nil {
                log.Printf("%v", err)
                continue
            }
            archive = opened
            archives[spell.Archive] = archive
        }

        frames, err := readPictures(archive, spell.Entry)
        if err != nil {
            log.Printf("No pictures of %v in %v entry %v", spell.Name, spell.Archive, spell.Entry)
            continue
        }

        folder := filepath.Join(outPath, "spells", spell.Name)
        for frame, picture := range frames {
            if writePng(filepath.Join(folder, mod.SpellFrameFile(frame)), picture) == nil {
                count += 1
            }
        }

        writePng(filepath.Join(folder, "_sheet.png"), makeSheet([][]*image.Paletted{frames}))

        // the marks the spell leaves on the ground: the first ones, of the game, to be changed
        for number, picture := range combat.DecalPictures(combat.GameDecal(spell.Name)) {
            writePng(filepath.Join(folder, mod.SpellExtraFile(combat.DecalPrefix, number)), picture)
        }

        width, height := 0, 0
        if len(frames) > 0 {
            width = frames[0].Bounds().Dx()
            height = frames[0].Bounds().Dy()
        }
        text := fmt.Sprintf("%v\r\n\r\nShown by: %v\r\nFrames: %v, %v by %v pixels\r\nFrom: %v entry %v\r\n", spell.Name, spell.Note, len(frames), width, height, spell.Archive, spell.Entry)
        os.WriteFile(filepath.Join(folder, "_source.txt"), []byte(text), 0644)
    }

    return count
}

// the pictures of the places battles are fought in, in folders by set, with names. the same
// pictures as in archives/, easier to find
func exportEnvironment(dataPath string, outPath string) int {
    count := 0
    notes := make(map[string]*strings.Builder)
    var sets []string

    for _, picture := range mod.EnvironmentPictures {
        archive, err := openArchive(dataPath, picture.Archive)
        if err != nil {
            log.Printf("%v", err)
            continue
        }

        frames, err := readPictures(archive, picture.Entry)
        if err != nil || len(frames) == 0 {
            log.Printf("No pictures of %v/%v in %v entry %v", picture.Set, picture.Name, picture.Archive, picture.Entry)
            continue
        }

        folder := filepath.Join(outPath, "environment", picture.Set)
        for number, frame := range frames {
            if writePng(filepath.Join(folder, mod.EnvironmentFrameFile(picture.Name, number)), frame) == nil {
                count += 1
            }
        }

        note, ok := notes[picture.Set]
        if !ok {
            note = &strings.Builder{}
            notes[picture.Set] = note
            sets = append(sets, picture.Set)
            note.WriteString(fmt.Sprintf("%v\r\n\r\n", picture.Set))
        }
        bounds := frames[0].Bounds()
        note.WriteString(fmt.Sprintf("%-20v %v frames, %v by %v, %v entry %v: %v\r\n", picture.Name, len(frames), bounds.Dx(), bounds.Dy(), picture.Archive, picture.Entry, picture.Note))
    }

    // the large pieces of 2 by 2 tiles (combat/large.go): the one the game makes of the dirt of
    // every landscape, and the four tiles as grass to paint a new one over
    for _, picture := range mod.EnvironmentPictures {
        if picture.Entry != 0 || notes[picture.Set] == nil || !strings.HasPrefix(picture.Name, "grass") {
            continue
        }
        archive, err := openArchive(dataPath, picture.Archive)
        if err != nil {
            continue
        }
        tiles := func(index int) *image.Paletted {
            frames, err := readPictures(archive, index)
            if err != nil || len(frames) == 0 {
                return nil
            }
            return frames[0]
        }

        folder := filepath.Join(outPath, "environment", picture.Set)
        largeNotes := []string{"a cluster of rocks", "a bare patch of dirt"}
        for index, large := range combat.MakeLargePieces(tiles) {
            name := mod.NumberedName(combat.LargeName(), index + 1)
            if large != nil && writePng(filepath.Join(folder, mod.EnvironmentFrameFile(name, 0)), large) == nil {
                count += 1
                notes[picture.Set].WriteString(fmt.Sprintf("%-20v 1 frames, %v by %v, made by this game, not the original's: %v over 2 by 2 tiles\r\n", name, large.Bounds().Dx(), large.Bounds().Dy(), largeNotes[index]))
            }
        }
        template := combat.MakeLargeTemplate(tiles)
        if template != nil {
            writePng(filepath.Join(folder, "_large template.png"), template)
        }
    }

    for _, set := range sets {
        os.WriteFile(filepath.Join(outPath, "environment", set, "_source.txt"), []byte(notes[set].String()), 0644)
    }

    return count
}

// takes the pictures of the cursors as they are read
type cursorKeeper struct {
    Pictures []*image.Paletted
}

func (keeper *cursorKeeper) ApplyScale(picture image.Image) image.Image {
    paletted, ok := picture.(*image.Paletted)
    if ok {
        keeper.Pictures = append(keeper.Pictures, paletted)
    }
    return picture
}

// the cursors of the mouse. they are not pictures of an archive as the others are
func exportCursors(dataPath string, outPath string) int {
    archive, err := openArchive(dataPath, "fonts.lbx")
    if err != nil {
        log.Printf("%v", err)
        return 0
    }

    var keeper cursorKeeper
    _, err = mouse.ReadMouseImages(archive, &keeper, mouse.CursorEntry)
    if err != nil {
        log.Printf("Unable to read the cursors: %v", err)
        return 0
    }

    count := 0
    for index, picture := range keeper.Pictures {
        name := mod.CursorFile(index)
        if name == "" {
            continue
        }
        if writePng(filepath.Join(outPath, "cursors", name), picture) == nil {
            count += 1
        }
    }

    return count
}

func exportPalette(outPath string) error {
    palette := lbx.GetDefaultPalette()

    rows := (len(palette) + paletteColumns - 1) / paletteColumns
    picture := image.NewRGBA(image.Rect(0, 0, paletteColumns * paletteSquare, rows * paletteSquare))
    for index, entry := range palette {
        left := index % paletteColumns * paletteSquare
        top := index / paletteColumns * paletteSquare
        red, green, blue, _ := entry.RGBA()
        for y := range paletteSquare {
            for x := range paletteSquare {
                picture.Set(left + x, top + y, color.RGBA{R: uint8(red >> 8), G: uint8(green >> 8), B: uint8(blue >> 8), A: 0xff})
            }
        }
    }

    err := writePng(filepath.Join(outPath, "palette.png"), picture)
    if err != nil {
        return err
    }

    // the palette as GIMP, Aseprite and others read it
    var text strings.Builder
    text.WriteString("GIMP Palette\nName: Master of Magic\nColumns: 16\n#\n")
    for index, entry := range palette {
        red, green, blue, _ := entry.RGBA()
        text.WriteString(fmt.Sprintf("%3d %3d %3d\tIndex %v\n", red >> 8, green >> 8, blue >> 8, index))
    }

    return os.WriteFile(filepath.Join(outPath, "palette.gpl"), []byte(text.String()), 0644)
}

const readme = `Pictures of the battles of Master of Magic, written by util/export-sprites.

THESE PICTURES BELONG TO THE RIGHTS HOLDER OF THE GAME. They are here for the owner of the game to
look at and to change for their own use. Do not put them in the repository, do not pass them on.

units/<race> <name>/
    <facing>_<frame>.png    one picture of the figure, 28 by 30 pixels for most units
    _sheet.png              all of them on one picture: a row per facing, a column per frame
    _source.txt             the archive and the entries the figure comes from

    Facings, in the order of the rows of the sheet:
        up, upright, right, downright, down, downleft, left, upleft
    Frames:
        0, 1, 2   walking. 1 is also how the figure stands
        3         striking
    The feet of a figure are at 13 across and 23 down in its picture. The game puts that point on
    the place of the figure, so a changed figure has to keep its feet there.

archives/<file>/<entry>_<frame>.png
    every picture of the archives battles draw from:
        cmbgrass, cmbdesrt, cmbmount, cmbtundr   the ground, trees (48 to 52) and rocks (53 to 57)
        cmbgrasc, cmbdesrc, cmbmounc, cmbtundc   the same on Myrror
        cmbtcity    houses (2 to 16), fortress (17), outpost (18), cave, tower, temple, keep, ruins,
                    nodes, the outlines of tiles (67, 68), mud, clouds
        citywall    walls of stone (0 to 35), of fire (36 to 49), of darkness (50 to 63)
        wallrise    walls rising
        cmbtfx      spells and projectiles
        cmbmagic    more spells
        chriver     rivers, the chaos node
        compix      the combat bar and its buttons

environment/<set>/<name>_<frame>.png
    the pictures of the places battles are fought in, by name. They are the same pictures as in
    archives/, easier to find. _source.txt in each folder lists them with their sizes, where they
    come from and where the game shows them.
        Grass, Desert, Mountain, Tundra, and each on Myrror
                    grass 1 to 4, dirt 1 to 4, grass with dirt next to it (dirt edge,
                    dirt edges, dirt corner), raised ground (rough), tree 1 to 5, rock 1 to 5.
                    The directions are the ones of the tile on the screen: its edges NE (upper
                    right), SE, SW, NW and its corners N (top), E, S (bottom), W. "dirt edge SE"
                    is grass with dirt beyond its lower right edge, "rough NE SW" raised ground
                    that goes on across its upper right and lower left edges.
                    Pictures of the ground, trees, rocks and houses can be added in the
                    replacement folder, their names counting on: grass 5, dirt edge SE 3,
                    rough NE SW 2, tree 6, rock 6, house 6. Its README says how.
                    large 1 and 2 are pieces of 2 by 2 tiles this game makes of the rocks and
                    the dirt of the landscape, not pictures of the original. _large template
                    shows the four tiles to paint a new one over
        Water, Water Myrror    the ground of a battle on the water
        Town        roads, houses, huts, tree houses, fortress, outpost, clouds under a flying
                    fortress, the pieces of the roads that lead out of the field ("road NE SW 1"
                    joins the upper right and lower left edges, 1 and 2 are two sets)
        Lairs       cave, tower, ruins, keep, temple, the nodes
        Walls       stone (frame 0 standing, 1 broken), fire, darkness, and fire and darkness
                    while they rise. The numbers are the pieces around the town: 00 the far
                    corner, 01 to 03 the far left side, 04 to 06 the far right side, 07 to 09 the
                    near left side, 10 and 11 the near right side (11 the gate), 12 and 13 the
                    left and right corners of fire and darkness
        Other       mud, and pieces the game does not use
    Keep the size of a changed picture: most are put in place by a fixed point in them.

spells/<name>/<frame>.png
    the pictures of the spells of battles, in folders with their names. They are the same pictures
    as in archives/cmbtfx, specfx and cmbmagic, easier to find. _source.txt says which spells show
    the picture and where it comes from, _sheet.png shows all frames in a row.
    A changed frame has to keep the size of the picture. Frames can be added after the last one
    for the spells that play all their frames once (not for the bolts and the lightning, which
    use their frames by number).
    Realm Nature, Sorcery, Chaos, Life, Death and Arcane are the pictures most spells on a unit
    show: one picture for all such spells of the realm.
    In a battle the game adds light to these pictures: what is drawn bright glows, black gives no
    light. It also adds particles, and more where a spell hits. How much is in effects.txt.
    Fire Bolt and Ice Bolt: the last frame is where it hits, all frames before it go around while
    it flies. The game has 3 of those, more can be added: with 00 to 04 in flight the hit is 05.

spells/<name>/decal_00.png, decal_01.png, ...
    the marks a spell leaves on the ground where it hits. They stay for the whole battle. These
    are not pictures of the original game, it has none: they are a first set made by this game.
    One of them is taken by chance for every mark, and by chance turned left to right.
    In the replacement folder there can be any number of them from 00 on, in any size and with
    any colors. A tile of the ground is 30 wide and 16 high, and the ground is seen from above
    at an angle, so a round mark is twice as wide as it is high. The middle of the picture lies
    on the middle of the tile.

effects.txt
    the values of what the game adds to the pictures of spells: light, particles, and where a spell
    hits the color of the unit, the battle standing still, the view shaking, light on the ground.
    The file says what each value is. Copied into the replacement folder it is read by the game,
    again and again while it runs.

aseprite/
    not written by this export: the Aseprite files made from these pictures. One file per race,
    a folder per unit, a layer per facing. Spells <width>x<height>.aseprite: one file for each
    size of the pictures of spells, a layer per spell. aseprite-export.bat writes what was changed
    in them into the replacement folder.

cursors/<number>_<name>.png
    the cursors of the mouse, 16 by 16 pixels. A changed cursor has to keep that size.
        00_normal   the pointer
        01_magic    the pointer over the world map while a spell is aimed
        02_cross    the red X
        03_arrow    a shot
        04_attack   a strike
        05_wait     the hourglass
        06_move     the boot
        08_cast0 to 12_cast4   aiming a spell in a battle, 5 frames
        07, 13, 14, 15         not used by the game
    The point of a cursor that counts as the place of the mouse is fixed in the game. In battles:
    the boot 5 across and 3 down, the strike and the X 7 across and 7 down, the shot 1 across and
    1 down. Everywhere
    else the upper left corner.

COLORS
    The png files are in indexed color: every pixel is the number of a color of the game's palette
    of 256. The game needs those numbers. Keep a changed picture in indexed color with the same
    palette, and pick colors from the palette only. palette.gpl loads into GIMP, Aseprite,
    LibreSprite and others. palette.png shows the colors in the order of their numbers, 16 a row.

    Color 0 is see-through.
    Colors 215, 216, 217, 218 are the colors of the banner, lightest first. The game replaces them
    with the color of the wizard the unit belongs to. In the pictures they are green.
`

func main() {
    dataPath := flag.String("data", "", "the folder of the game's LBX files")
    outPath := flag.String("out", "", "the folder to write to. outside of the repository")
    flag.Parse()

    if *dataPath == "" || *outPath == "" {
        flag.Usage()
        os.Exit(1)
    }

    err := os.MkdirAll(*outPath, 0755)
    if err != nil {
        log.Fatalf("Unable to make %v: %v", *outPath, err)
    }

    unitCount, unitPictures := exportUnits(*dataPath, *outPath)
    archivePictures := exportArchives(*dataPath, *outPath)
    cursorPictures := exportCursors(*dataPath, *outPath)
    spellPictures := exportSpells(*dataPath, *outPath)
    environmentPictures := exportEnvironment(*dataPath, *outPath)

    err = exportPalette(*outPath)
    if err != nil {
        log.Printf("Unable to write the palette: %v", err)
    }

    os.WriteFile(filepath.Join(*outPath, "README.txt"), []byte(readme), 0644)
    // the values of the effects of spells, to be changed and put into the replacement folder
    os.WriteFile(filepath.Join(*outPath, mod.EffectsFile), []byte(combat.EffectsTemplate()), 0644)

    fmt.Printf("units: %v, pictures of units: %v, pictures of archives: %v, cursors: %v, pictures of spells: %v, pictures of the environment: %v\n", unitCount, unitPictures, archivePictures, cursorPictures, spellPictures, environmentPictures)
    fmt.Printf("written to %v\n", *outPath)
}
