package halloffame

// The score at the end of a game and the Hall of Fame, as the original game has them.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/SCORE.c:
// End_Of_Game_Score, End_Of_Game_Score_Draw, Hall_Of_Fame_Screen, Hall_Of_Fame_Screen_Draw).
// The code is ours. Upstream had neither (a FIXME for a Hall of Fame button on the main menu).
//
// THE SCORE, all of the human player:
//   a point for every spell it knows
//   half the people of its cities, in thousands, rounded down (the text shows the whole number)
//   50 for every wizard it banished or defeated, each wizard once; a win by the Spell of Mastery
//     counts every wizard whose fortress still stands
//   its fame twice (with Just Cause)
//   2000 less 2 a turn, only while its own fortress stands and nobody else cast the Spell of
//     Mastery; no floor at 0
//   250 for its own Spell of Mastery
//   the sum is halved at Intro, three quarters at Easy, doubled at Hard, three times at
//     Impossible (Extreme counts as Impossible), rounded down
//
// THE SCORE SCREEN: halofam.lbx 0, the name in font 3 color set 4 in the middle at 160,3; the
// lines in font 2 color set 4 from x 47; both with soft edge 246 and a shadow below and right in
// 250. "N Spells: N" at 15 with a book (halofam 30) for each, at most 50, 8 apart from y 25, the
// 26th and on 4 further right and 3 lower; "N Townsfolk: N/2" at 37 with the picture of the
// race (halofam 15 + race) for each point, at most 100, 9 apart from y 46, every 25 a row 5
// lower that starts 4 further right in its 2nd and 4th row; "No Banished Wizards: 0" or
// "N Banished Wizards: 50N" at 78 with the faces (halofam 1 + wizard) 35 apart at y 87;
// fame (halofam 31) at 47,123 and its text at 65,128; the hourglass (29) at 47,142 with
// "Year <1400 + turn / 12>: +N" at 65,147; the Spell of Mastery (32) at 47,161 with its text at
// 65,166; "Final Score: S" at 47,180; under it a bar in a frame from 47,189 to 251,197 whose
// fill in 77 is score / 40 long, at most 200. A click or Escape goes on.
//   BY EYE: the frame is drawn in the original through remap blocks (dark above and left, light
//   below and right), which were not read; here black and white at half and a quarter.
//
// THE HALL OF FAME: 10 places of name, score and race; a new score goes in at the first place
// with a lower score, the rest move down and the last is dropped. The original keeps them in
// MAGIC.SET; here in hall-of-fame.json beside the settings (hall-of-fame-dev.json in runs of
// Claude's own). Screen halofam.lbx 0: "Hall Of Fame" in font 5 in its own colors in the middle
// at 160,7, shadow 250; the places in font 2, all 179 with soft edge 246, shadow 250, from y 42,
// 13 apart, only places with a score: "Master" for the first and "2)" to "10)" ending on 72,
// "<name> of the <race>" from 81, the score ending on 242, "(<score * 100 / 8000>%)" ending on
// 270.

import (
    "encoding/json"
    "fmt"
    "image/color"
    "log"
    "os"
    "strings"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    fontslib "github.com/kazzmir/master-of-magic/game/magic/fonts"
    "github.com/kazzmir/master-of-magic/game/magic/inputmanager"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/game/magic/util"
    "github.com/kazzmir/master-of-magic/lib/coroutine"
    "github.com/kazzmir/master-of-magic/lib/font"
    "github.com/kazzmir/master-of-magic/lib/lbx"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/inpututil"
    "github.com/hajimehoshi/ebiten/v2/vector"
)

const lbxFile = "halofam.lbx"

const (
    pictureBackground = 0
    pictureFirstWizard = 1
    pictureFirstRace = 15
    pictureTime = 29
    pictureBook = 30
    pictureFame = 31
    pictureMastery = 32
)

const (
    scoreLeft = 47
    scoreEdge = 246
    scoreShadow = 250
    scoreBarColor = 77
    // the score that fills the bar and the one that is 100 percent in the Hall of Fame
    scorePerBarPixel = 40
    scoreBarMost = 200
    scoreFull = 8000

    booksMost = 50
    townsfolkMost = 100
)

var hallTitleColors = []int{246, 49, 176, 162, 178, 179, 51, 180, 184, 185, 213, 186, 246, 246, 246, 246}

// the Hall of Fame file. runs of Claude's own set DevelopmentFile
var File = "hall-of-fame.json"
const DevelopmentFile = "hall-of-fame-dev.json"

const Places = 10

type Entry struct {
    Name string `json:"name"`
    Score int `json:"score"`
    Race data.Race `json:"race"`
}

type saved struct {
    Entries []Entry `json:"entries"`
}

// the places of the Hall of Fame, best first
func Load() []Entry {
    data, err := os.ReadFile(File)
    if err != nil {
        return nil
    }
    var out saved
    if err := json.Unmarshal(data, &out); err != nil {
        log.Printf("Unable to read %v: %v", File, err)
        return nil
    }
    if len(out.Entries) > Places {
        out.Entries = out.Entries[:Places]
    }
    return out.Entries
}

func store(entries []Entry) {
    data, err := json.MarshalIndent(saved{Entries: entries}, "", "  ")
    if err != nil {
        return
    }
    if err := os.WriteFile(File, data, 0644); err != nil {
        log.Printf("Unable to write %v: %v", File, err)
    }
}

// where a score goes in a list of places: the first place with a lower score, -1 when none
func placeOf(entries []Entry, score int) int {
    for i := 0; i < Places; i++ {
        if i >= len(entries) || entries[i].Score < score {
            return i
        }
    }
    return -1
}

// the list with the new score in its place and the last one dropped; the same list when it does
// not get a place
func insert(entries []Entry, entry Entry) ([]Entry, bool) {
    place := placeOf(entries, entry.Score)
    if place < 0 || entry.Score <= 0 {
        return entries, false
    }
    out := append([]Entry{}, entries[:min(place, len(entries))]...)
    out = append(out, entry)
    if place < len(entries) {
        out = append(out, entries[place:]...)
    }
    if len(out) > Places {
        out = out[:Places]
    }
    return out, true
}

// puts the score in the Hall of Fame file when it gets a place
func Insert(entry Entry) bool {
    entries, ok := insert(Load(), entry)
    if ok {
        store(entries)
    }
    return ok
}

// what the score is made of
type Score struct {
    Name string
    // the race of the capital
    Race data.Race
    Spells int
    // the people of its cities, in thousands
    Townsfolk int
    Banished []data.WizardBase
    // with Just Cause
    Fame int
    Turn int
    FortressStands bool
    // the Spell of Mastery: cast by the player, or by another wizard
    Mastery bool
    OtherMastery bool
    Difficulty data.DifficultySetting
}

func (score Score) TownsfolkPoints() int {
    return score.Townsfolk / 2
}

func (score Score) TimePoints() int {
    if score.FortressStands && !score.OtherMastery {
        return 2000 - 2 * score.Turn
    }
    return 0
}

func (score Score) MasteryPoints() int {
    if score.Mastery {
        return 250
    }
    return 0
}

func (score Score) Year() int {
    return 1400 + score.Turn / 12
}

func (score Score) Final() int {
    sum := score.Spells + score.TownsfolkPoints() + 50 * len(score.Banished) + 2 * score.Fame + score.TimePoints() + score.MasteryPoints()
    switch score.Difficulty {
        case data.DifficultyIntro: return sum / 2
        case data.DifficultyEasy: return sum * 3 / 4
        case data.DifficultyHard: return sum * 2
        case data.DifficultyImpossible, data.DifficultyExtreme: return sum * 3
    }
    return sum
}

// the index of a race in the original's list (halofam pictures, the names of the Hall of Fame)
var raceOrder = []data.Race{
    data.RaceBarbarian, data.RaceBeastmen, data.RaceDarkElf, data.RaceDraconian, data.RaceDwarf,
    data.RaceGnoll, data.RaceHalfling, data.RaceHighElf, data.RaceHighMen, data.RaceKlackon,
    data.RaceLizard, data.RaceNomad, data.RaceOrc, data.RaceTroll,
}

var raceNames = []string{
    "Barbarians", "Beastmen", "Dark Elves", "Draconians", "Dwarves", "Gnolls", "Halflings",
    "High Elves", "High Men", "Klackons", "Lizardmen", "Nomads", "Orcs", "Trolls",
}

func raceIndex(race data.Race) int {
    for i, check := range raceOrder {
        if check == race {
            return i
        }
    }
    return 0
}

// a click anywhere or Escape
func leaves() bool {
    return inputmanager.LeftClick() || inpututil.IsKeyJustPressed(ebiten.KeyEscape)
}

// shows a picture of art pixels until a click or Escape
func showPicture(picture func() *ebiten.Image) (coroutine.AcceptYieldFunc, func (*ebiten.Image)) {
    var shown *ebiten.Image
    logic := func (yield coroutine.YieldFunc) error {
        // the click or key that opened the screen is not the one that closes it
        if yield() != nil {
            return nil
        }
        for !leaves() {
            if yield() != nil {
                return nil
            }
        }
        return nil
    }
    draw := func (screen *ebiten.Image) {
        if shown == nil {
            shown = picture()
        }
        if shown != nil {
            scale.DrawScaled(screen, shown, &ebiten.DrawImageOptions{})
        }
    }
    return logic, draw
}

func fill(picture *ebiten.Image, x int, y int, width int, height int, use color.Color) {
    vector.FillRect(picture, float32(x), float32(y), float32(width), float32(height), use, false)
}

// the score screen
func ScoreScreen(cache *lbx.LbxCache, score Score) (coroutine.AcceptYieldFunc, func (*ebiten.Image)) {
    return showPicture(func() *ebiten.Image {
        return drawScore(cache, score)
    })
}

func drawScore(cache *lbx.LbxCache, score Score) *ebiten.Image {
    imageCache := util.MakeImageCache(cache)
    picture := ebiten.NewImage(data.ScreenWidth, data.ScreenHeight)
    picture.Fill(color.Black)

    draw := func(index int, x int, y int) {
        image, err := imageCache.GetImage(lbxFile, index, 0)
        if err != nil {
            return
        }
        var options ebiten.DrawImageOptions
        options.GeoM.Translate(float64(x), float64(y))
        picture.DrawImage(image, &options)
    }

    draw(pictureBackground, 0, 0)

    original := fontslib.LoadOriginal(cache, lbxFile, pictureBackground)
    if original == nil {
        return picture
    }
    nameStyle := original.Style(3, original.Set(4, scoreEdge), font.ShadowDown, scoreShadow)
    lineStyle := original.Style(2, original.Set(4, scoreEdge), font.ShadowDown, scoreShadow)
    if nameStyle == nil || lineStyle == nil {
        return picture
    }

    left := font.FontOptions{Scale: 1}
    center := font.FontOptions{Scale: 1, Justify: font.FontJustifyCenter}

    nameStyle.Print(picture, 160, 3, center, score.Name)

    // spells
    lineStyle.Print(picture, scoreLeft, 15, left, fmt.Sprintf("%v Spells: %v", score.Spells, score.Spells))
    x, y := scoreLeft, 25
    for i := 0; i < min(score.Spells, booksMost); i++ {
        draw(pictureBook, x, y)
        x += 8
        if i == 24 {
            x = scoreLeft + 4
            y += 3
        }
    }

    // townsfolk
    points := score.TownsfolkPoints()
    lineStyle.Print(picture, scoreLeft, 37, left, fmt.Sprintf("%v Townsfolk: %v", score.Townsfolk, points))
    x, y = scoreLeft, 46
    for i := 0; i < min(points, townsfolkMost); i++ {
        draw(pictureFirstRace + raceIndex(score.Race), x, y)
        x += 9
        if (i + 1) % 25 == 0 {
            quotient := (i + 1) / 25
            if quotient == 1 || quotient == 3 {
                x = scoreLeft + 4
            } else {
                x = scoreLeft
            }
            y += 5
        }
    }

    // banished wizards
    count := "No"
    if len(score.Banished) > 0 {
        count = fmt.Sprintf("%v", len(score.Banished))
    }
    lineStyle.Print(picture, scoreLeft, 78, left, fmt.Sprintf("%v Banished Wizards: %v", count, 50 * len(score.Banished)))
    for i, wizard := range score.Banished {
        draw(pictureFirstWizard + int(wizard), scoreLeft + 35 * i, 87)
    }

    draw(pictureFame, scoreLeft, 123)
    lineStyle.Print(picture, scoreLeft + 18, 128, left, fmt.Sprintf("%v Fame: %v", score.Fame, 2 * score.Fame))

    time := score.TimePoints()
    sign := ""
    if time >= 0 {
        sign = "+"
    }
    draw(pictureTime, scoreLeft, 142)
    lineStyle.Print(picture, scoreLeft + 18, 147, left, fmt.Sprintf("Year %v: %v%v", score.Year(), sign, time))

    draw(pictureMastery, scoreLeft, 161)
    lineStyle.Print(picture, scoreLeft + 18, 166, left, fmt.Sprintf("Spell Of Mastery: %v", score.MasteryPoints()))

    final := score.Final()
    lineStyle.Print(picture, scoreLeft, 180, left, fmt.Sprintf("Final Score: %v", final))

    // the frame of the bar, by eye (see the top of the file)
    dark := color.NRGBA{A: 128}
    light := color.NRGBA{R: 255, G: 255, B: 255, A: 64}
    fill(picture, 47, 189, 205, 1, dark)
    fill(picture, 47, 190, 1, 8, dark)
    fill(picture, 48, 197, 204, 1, light)
    fill(picture, 251, 190, 1, 7, light)
    fill(picture, 48, 190, 203, 7, color.Black)
    length := min(max(final / scorePerBarPixel, 0), scoreBarMost)
    if length > 0 {
        fill(picture, 49, 191, length + 1, 5, original.Palette[scoreBarColor])
    }

    return picture
}

// the Hall of Fame screen
func HallOfFameScreen(cache *lbx.LbxCache) (coroutine.AcceptYieldFunc, func (*ebiten.Image)) {
    return HallOfFameScreenOf(cache, Load())
}

// the Hall of Fame screen with these places
func HallOfFameScreenOf(cache *lbx.LbxCache, entries []Entry) (coroutine.AcceptYieldFunc, func (*ebiten.Image)) {
    return showPicture(func() *ebiten.Image {
        return drawHallOfFame(cache, entries)
    })
}

func drawHallOfFame(cache *lbx.LbxCache, entries []Entry) *ebiten.Image {
    imageCache := util.MakeImageCache(cache)
    picture := ebiten.NewImage(data.ScreenWidth, data.ScreenHeight)
    picture.Fill(color.Black)

    background, err := imageCache.GetImage(lbxFile, pictureBackground, 0)
    if err == nil {
        picture.DrawImage(background, &ebiten.DrawImageOptions{})
    }

    original := fontslib.LoadOriginal(cache, lbxFile, pictureBackground)
    if original == nil {
        return picture
    }
    title := original.Style(5, original.Colors(hallTitleColors...), font.ShadowDown, scoreShadow)
    rows := original.Style(2, original.Flat(scoreEdge, 179), font.ShadowDown, scoreShadow)
    if title == nil || rows == nil {
        return picture
    }

    left := font.FontOptions{Scale: 1}
    right := font.FontOptions{Scale: 1, Justify: font.FontJustifyRight}
    center := font.FontOptions{Scale: 1, Justify: font.FontJustifyCenter}

    title.Print(picture, 160, 7, center, "Hall Of Fame")

    x, y := 72, 42
    for i, entry := range entries {
        if entry.Score == 0 {
            continue
        }
        rank := "Master"
        if i > 0 {
            rank = fmt.Sprintf("%v)", i + 1)
        }
        rows.Print(picture, x, y, right, rank)
        rows.Print(picture, x + 9, y, left, strings.Join([]string{entry.Name, "of the", raceNames[raceIndex(entry.Race)]}, " "))
        rows.Print(picture, x + 170, y, right, fmt.Sprintf("%v", entry.Score))
        rows.Print(picture, x + 198, y, right, fmt.Sprintf("(%v%%)", entry.Score * 100 / scoreFull))
        y += 13
    }

    return picture
}
