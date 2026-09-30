package mod

// The replacement folder: pictures in it take the place of the game's own.
//
// The folder has the layout util/export-sprites writes, so a picture that was changed is copied to
// the same place in the replacement folder:
//
//   units/<race> <name>/<facing>_<frame>.png     the figures of a unit
//   archives/<file>/<entry>_<frame>.png          any picture of any archive, <entry> with 3 digits
//   cursors/<number>_<name>.png                  a cursor of the mouse, see CursorNames
//   spells/<name>/<frame>.png                    a spell of the battles, see spells.go
//   environment/<set>/<name>_<frame>.png         the ground, walls, towns of battles, see environment.go
//
// A picture that is not there is the game's own. Every frame is a file of its own: the sheets the
// export writes are for looking at and are not read.
//
// NEW FRAMES (user, 2026-09-27). The figures of the game have the frames 0 to 3. The replacement
// folder can give a figure more, per facing:
//   4, 5, 6   more frames of the strike: the figure strikes through 3 and the ones that are
//             there and around. 4 alone gives 3 4, 4 and 5 give 3 4 5
//   7, 8      death: 7 on the way down, 8 lying, where the figure stays
// A figure can have some of them and not others. HasFrame says which are there.
//
// Pictures are read when they are first needed by a screen, and a battle asks again when it
// starts: a picture that was changed shows in the next battle, without starting the game again.
//
// The game's pictures are numbers of colors of its palette. A png in indexed color with the game's
// palette is taken as it is. Any other png is matched to the palette color by color: the same color
// where the palette has it, the closest where it has not. See-through pixels become color 0.

import (
    "fmt"
    "image"
    "image/color"
    "image/png"
    "log"
    "os"
    "path/filepath"
    "strings"
    "time"
)

// the folders that are looked at when none was given, from where the game runs
var DefaultFolders = []string{"mod", "../mod"}

const unitsFolder = "units"
const archivesFolder = "archives"
const cursorsFolder = "cursors"

// the cursors of the game in the order it keeps them, as the files are named. the export writes
// the same names
var CursorNames = []string{
    "00_normal", "01_magic", "02_cross", "03_arrow", "04_attack", "05_wait", "06_move", "07",
    "08_cast0", "09_cast1", "10_cast2", "11_cast3", "12_cast4", "13", "14", "15",
}

const CursorCross = 2

// the cursors that came from the replacement folder
var replacedCursors = make(map[int]bool)

func CursorFile(index int) string {
    if index < 0 || index >= len(CursorNames) {
        return ""
    }
    return CursorNames[index] + ".png"
}

// true if the cursor came from the replacement folder
func HasCursor(index int) bool {
    return replacedCursors[index]
}

// the picture of a cursor, the one of the replacement folder if there is one. a cursor has to keep
// its size
func ReplaceCursor(index int, picture *image.Paletted) *image.Paletted {
    delete(replacedCursors, index)

    name := CursorFile(index)
    if folder == "" || name == "" || picture == nil {
        return picture
    }

    path := filepath.Join(folder, cursorsFolder, name)
    replacement := readFrame([]string{path}, picture.Palette)
    if replacement == nil {
        return picture
    }

    if replacement.Bounds().Dx() != picture.Bounds().Dx() || replacement.Bounds().Dy() != picture.Bounds().Dy() {
        reportOnce(fmt.Sprintf("Replacement cursor %v is %v by %v, it has to be %v by %v. The cursor of the game is used", path, replacement.Bounds().Dx(), replacement.Bounds().Dy(), picture.Bounds().Dx(), picture.Bounds().Dy()))
        return picture
    }

    replacedCursors[index] = true
    return replacement
}

// pixels less solid than this, of 255, are see-through
const solidFrom = 128
// frames are looked for up to this number
const maxFrames = 16
// the files of a folder are looked up again after this long, so a picture that was added is found
const folderListTime = 2 * time.Second

// the frames of a figure the replacement folder can add, see the top of the file
const FrameStrike = 3
const FrameStrikeLast = 6
const FrameDying = 7
const FrameDead = 8

// the first color of the palette is see-through
const clearColor = 0

var FacingNames = []string{"up", "upright", "right", "downright", "down", "downleft", "left", "upleft"}

// the replacement folder, empty when there is none
var folder string
// by archive and entry: the folder of the unit and the facing the entry is
var unitEntries = make(map[string]unitEntry)
// what was said in the log already
var reported = make(map[string]bool)
// by archive and entry: the frames that came from the replacement folder
var replaced = make(map[string]map[int]bool)
// the entries whose number of frames the replacement folder has changed, with how many the game has
var framesChanged = make(map[string]int)

type folderList struct {
    Files map[string]bool
    Read time.Time
}

var folderLists = make(map[string]*folderList)

// true if the folder has the file. looks the folder up once in a while, not for every file
func hasFile(path string) bool {
    directory := filepath.Dir(path)

    list, ok := folderLists[directory]
    if !ok || time.Since(list.Read) > folderListTime {
        list = &folderList{Files: make(map[string]bool), Read: time.Now()}
        entries, err := os.ReadDir(directory)
        if err == nil {
            for _, entry := range entries {
                list.Files[strings.ToLower(entry.Name())] = true
            }
        }
        folderLists[directory] = list
    }

    return list.Files[strings.ToLower(filepath.Base(path))]
}

// true if the frame of the entry came from the replacement folder
func HasFrame(archive string, entry int, frame int) bool {
    frames, ok := replaced[entryKey(archive, entry)]
    return ok && frames[frame]
}

// true if the replacement folder gave the entry another number of frames than the game has: more,
// or fewer with its last frames missing. a spell with such frames plays all of them in their order
// (user, 2026-09-30: "any spell with modified frame numbers will just play the whole series,
// regardless of how many are there"). the entry has to have been read before
func FrameCountChanged(archive string, entry int) bool {
    _, ok := framesChanged[entryKey(archive, entry)]
    return ok
}

// true if all frames from first to last came from the replacement folder
func HasFrames(archive string, entry int, first int, last int) bool {
    for frame := first; frame <= last; frame++ {
        if !HasFrame(archive, entry, frame) {
            return false
        }
    }
    return true
}

type unitEntry struct {
    Folder string
    Facing string
}

func entryKey(archive string, entry int) string {
    return fmt.Sprintf("%v:%v", strings.ToLower(archive), entry)
}

// letters, digits, spaces and dashes only. the same as the export
func FolderName(name string) string {
    var out strings.Builder
    for _, letter := range name {
        switch {
            case letter >= 'a' && letter <= 'z', letter >= 'A' && letter <= 'Z', letter >= '0' && letter <= '9', letter == ' ', letter == '-':
                out.WriteRune(letter)
        }
    }
    return strings.TrimSpace(out.String())
}

// the figures of a unit are 8 entries of an archive, one per facing, from entry on
func RegisterUnit(name string, archive string, entry int) {
    for facing, facingName := range FacingNames {
        key := entryKey(archive, entry + facing)
        if _, ok := unitEntries[key]; !ok {
            unitEntries[key] = unitEntry{Folder: FolderName(name), Facing: facingName}
        }
    }
}

// picks the replacement folder. path is the one that was asked for, or empty. returns the folder in use
func Setup(path string) string {
    folder = ""

    candidates := DefaultFolders
    if path != "" {
        candidates = []string{path}
    }

    for _, candidate := range candidates {
        info, err := os.Stat(candidate)
        if err == nil && info.IsDir() {
            folder = candidate
            break
        }
    }

    if folder != "" {
        absolute, err := filepath.Abs(folder)
        if err == nil {
            folder = absolute
        }
        log.Printf("Replacement pictures are read from %v", folder)
    } else if path != "" {
        log.Printf("No replacement folder at %v", path)
    }

    return folder
}

func Active() bool {
    return folder != ""
}

func reportOnce(text string) {
    if !reported[text] {
        reported[text] = true
        log.Print(text)
    }
}

// the files a frame of an entry can be, the one of the unit first
func framePaths(archive string, entry int, frame int) []string {
    var out []string

    unit, ok := unitEntries[entryKey(archive, entry)]
    if ok {
        out = append(out, filepath.Join(folder, unitsFolder, unit.Folder, fmt.Sprintf("%v_%v.png", unit.Facing, frame)))
    }

    // a spell by its name, see spells.go
    spell := spellFramePath(archive, entry, frame)
    if spell != "" {
        out = append(out, spell)
    }

    // the biome of the battle before its landscape, see biomes.go
    ofBiome := biomeFramePath(archive, entry, frame)
    if ofBiome != "" {
        out = append(out, ofBiome)
    }

    // the places of battles by name, see environment.go
    environment := environmentFramePath(archive, entry, frame)
    if environment != "" {
        out = append(out, environment)
    }

    base := strings.TrimSuffix(strings.ToLower(archive), ".lbx")
    out = append(out, filepath.Join(folder, archivesFolder, base, fmt.Sprintf("%03d_%v.png", entry, frame)))

    return out
}

func sameColor(a color.Color, b color.Color) bool {
    red1, green1, blue1, _ := a.RGBA()
    red2, green2, blue2, _ := b.RGBA()
    return red1 >> 8 == red2 >> 8 && green1 >> 8 == green2 >> 8 && blue1 >> 8 == blue2 >> 8
}

// the number of the palette's color that is the same as, or the closest to, a color. never the
// see-through one
func closestColor(palette color.Palette, use color.Color) uint8 {
    red, green, blue, _ := use.RGBA()
    red, green, blue = red >> 8, green >> 8, blue >> 8

    best := 1
    bestDistance := -1

    for index := 1; index < len(palette); index++ {
        otherRed, otherGreen, otherBlue, _ := palette[index].RGBA()
        dr := int(red) - int(otherRed >> 8)
        dg := int(green) - int(otherGreen >> 8)
        db := int(blue) - int(otherBlue >> 8)
        distance := dr * dr + dg * dg + db * db

        if bestDistance < 0 || distance < bestDistance {
            best = index
            bestDistance = distance
            if distance == 0 {
                break
            }
        }
    }

    return uint8(best)
}

// a picture of the game made from one that was read. palette is the one of the picture it replaces
func toPaletted(source image.Image, palette color.Palette, path string) *image.Paletted {
    bounds := source.Bounds()
    out := image.NewPaletted(image.Rect(0, 0, bounds.Dx(), bounds.Dy()), palette)

    indexed, ok := source.(*image.Paletted)
    if ok {
        // taken as it is when every color it uses is the palette's color of the same number
        same := true
        used := make(map[uint8]bool)
        for _, index := range indexed.Pix {
            used[index] = true
        }
        for index := range used {
            if int(index) >= len(palette) || int(index) >= len(indexed.Palette) {
                same = false
                break
            }
            _, _, _, alpha := indexed.Palette[index].RGBA()
            if index == clearColor || alpha >> 8 < solidFrom {
                continue
            }
            if !sameColor(indexed.Palette[index], palette[index]) {
                same = false
                break
            }
        }

        if same {
            for y := 0; y < bounds.Dy(); y++ {
                for x := 0; x < bounds.Dx(); x++ {
                    index := indexed.ColorIndexAt(bounds.Min.X + x, bounds.Min.Y + y)
                    _, _, _, alpha := indexed.Palette[index].RGBA()
                    if alpha >> 8 < solidFrom {
                        index = clearColor
                    }
                    out.SetColorIndex(x, y, index)
                }
            }
            return out
        }
    }

    // matched to the palette color by color
    matches := make(map[color.RGBA]uint8)
    inexact := 0

    for y := 0; y < bounds.Dy(); y++ {
        for x := 0; x < bounds.Dx(); x++ {
            red, green, blue, alpha := source.At(bounds.Min.X + x, bounds.Min.Y + y).RGBA()
            if alpha >> 8 < solidFrom {
                out.SetColorIndex(x, y, clearColor)
                continue
            }

            // the color without its see-through part
            use := color.RGBA{R: uint8(red * 0xff / alpha), G: uint8(green * 0xff / alpha), B: uint8(blue * 0xff / alpha), A: 0xff}

            index, ok := matches[use]
            if !ok {
                index = closestColor(palette, use)
                matches[use] = index
                if !sameColor(palette[index], use) {
                    inexact += 1
                }
            }
            out.SetColorIndex(x, y, index)
        }
    }

    if inexact > 0 {
        reportOnce(fmt.Sprintf("Replacement picture %v: %v of its colors are not in the palette, the closest are used", path, inexact))
    }

    return out
}

func readFrame(paths []string, palette color.Palette) *image.Paletted {
    for _, path := range paths {
        if !hasFile(path) {
            continue
        }

        file, err := os.Open(path)
        if err != nil {
            continue
        }

        picture, err := png.Decode(file)
        file.Close()
        if err != nil {
            reportOnce(fmt.Sprintf("Replacement picture %v can not be read: %v", path, err))
            continue
        }

        reportOnce(fmt.Sprintf("Replacement picture %v", path))
        return toPaletted(picture, palette, path)
    }

    return nil
}

// the pictures of an entry of an archive, with the ones of the replacement folder in the place of
// the game's. pictures are the game's own
func Replace(archive string, entry int, pictures []*image.Paletted) []*image.Paletted {
    if folder == "" || len(pictures) == 0 {
        return pictures
    }

    palette := pictures[0].Palette
    key := entryKey(archive, entry)
    delete(replaced, key)

    found := make(map[int]*image.Paletted)
    last := len(pictures) - 1
    for frame := 0; frame < maxFrames; frame++ {
        replacement := readFrame(framePaths(archive, entry, frame), palette)
        if replacement != nil {
            found[frame] = replacement
            last = max(last, frame)
        }
    }

    if len(found) == 0 {
        return pictures
    }

    replaced[key] = make(map[int]bool)
    delete(framesChanged, key)
    if last + 1 != len(pictures) {
        framesChanged[key] = len(pictures)
    }

    var out []*image.Paletted
    for frame := 0; frame <= last; frame++ {
        replacement, ok := found[frame]
        switch {
            case ok:
                out = append(out, replacement)
                replaced[key][frame] = true
            case frame < len(pictures):
                out = append(out, pictures[frame])
            default:
                // a frame that is not there, before one that is: never shown, HasFrame says it is missing
                out = append(out, pictures[min(1, len(pictures) - 1)])
        }
    }

    return out
}

// for tests: notes that the replacement folder changed the number of frames of an entry, or with
// 0 that it did not
func MarkFrameCountChangedForTest(archive string, entry int, gameFrames int) {
    if gameFrames == 0 {
        delete(framesChanged, entryKey(archive, entry))
        return
    }
    framesChanged[entryKey(archive, entry)] = gameFrames
}
