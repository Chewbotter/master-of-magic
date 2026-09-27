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

    "github.com/kazzmir/master-of-magic/game/magic/mod"
    "github.com/kazzmir/master-of-magic/game/magic/units"
    "github.com/kazzmir/master-of-magic/lib/lbx"
)

// the archives battles draw from, besides the figures of the units
var battleArchives = []string{
    "cmbgrass.lbx", "cmbgrasc.lbx", "cmbdesrt.lbx", "cmbdesrc.lbx", "cmbmount.lbx", "cmbmounc.lbx",
    "cmbtundr.lbx", "cmbtundc.lbx", "cmbtcity.lbx", "citywall.lbx", "wallrise.lbx", "cmbtwall.lbx",
    "cmbtfx.lbx", "cmbmagic.lbx", "chriver.lbx", "compix.lbx",
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

    err = exportPalette(*outPath)
    if err != nil {
        log.Printf("Unable to write the palette: %v", err)
    }

    os.WriteFile(filepath.Join(*outPath, "README.txt"), []byte(readme), 0644)

    fmt.Printf("units: %v, pictures of units: %v, pictures of archives: %v\n", unitCount, unitPictures, archivePictures)
    fmt.Printf("written to %v\n", *outPath)
}
