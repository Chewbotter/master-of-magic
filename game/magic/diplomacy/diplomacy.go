package diplomacy

// The screen of diplomacy as the original game has it: the wizard in the mirror between the
// gargoyles, what it says in its own words (messages.go), and the talk of the human, or what the
// wizard has to say at the end of a turn. What the talk changes is decided by the original's rules
// (relations/). Facts from the ReMoM project's reconstruction (MoM/src/DIPLOMAC.c:
// Diplomacy_Screen__WIP, Diplomacy_Screen_Draw__WIP, _sub_6ED5D_Draw, Get_Main_Diplomacy_Choices,
// Diplomacy_Propose_Treaty__WIP, Diplomacy_Display_Response, Diplomacy_Break_Treaty,
// Diplomacy_Offer_Tribute, Diplomacy_Exchange_Spell__WIP, Npc_Diplomacy_Screen, Npc_Proposal__WIP,
// DIPL_sub_72DB6__STUB, Start_Diplomacy_Music, Diplomacy_Screen_Draw_Portrait_Mouth_Animation,
// Diplomacy_Screen_Load__WIP, DIPL_LoadTalkGFX, Diplomacy_Screen_Draw_Gargoyle_Eyes,
// Diplomacy_Screen_Fade_In; MoX/src/Fields.c: Get_List_Field). The code is ours.
//
//   the picture diplomac 0; the eyes of the gargoyles diplomac 2 + level at 64,58 and 13 + level at
//     233,58, level (relation + 100) / 20
//   the wizard at 107,13: first its fade in (diplomac 38 + wizard), while it speaks diplomac 24 +
//     wizard, then its face moodwiz (wizard), pleased or angry by the group of the message; the
//     mirror backgrnd 18 at 95,1 over it
//   what is said: font 4 in color set 4 with soft edge 187, a shadow below and right in black,
//     from 38,140 in lines of 245 one pixel apart
//   a choice: the heading as what is said, at 38,138 (lines 0 apart when it is taller than 39); the
//     items font 1 in color set 4 (set 3 when they can not be picked) with soft edge 187, from 38,
//     the first a line under 142; the item under the mouse on a band from 37 to 283
//
// NOT THE ORIGINAL'S, and why:
//   - the band of the item under the mouse is dark BY EYE: the original remaps the colors behind it
//     through remap block 11 of its data, which was not read
//   - the answer to a proposal of the wizard (Accept, Reject) is listed under what it says: the
//     reconstruction has not reconstructed that list (DIPL_sub_72DB6__STUB)
//   - a proposal with gold or a spell adds the original's words "What if we were to also offer ...
//     as an incentive?": how the original puts them together is not reconstructed
//   - the lists of spells of a tribute and an exchange end with "Forget It" (the reconstruction's
//     lists lose their items there)

import (
    "image"
    "image/color"
    "log"
    "strconv"
    "strings"

    "github.com/kazzmir/master-of-magic/lib/coroutine"
    "github.com/kazzmir/master-of-magic/lib/lbx"
    "github.com/kazzmir/master-of-magic/lib/font"
    "github.com/kazzmir/master-of-magic/game/magic/util"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/game/magic/music"
    fontslib "github.com/kazzmir/master-of-magic/game/magic/fonts"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
    "github.com/kazzmir/master-of-magic/game/magic/relations"
    uilib "github.com/kazzmir/master-of-magic/game/magic/ui"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/vector"
)

const (
    textX = 38
    textY = 140
    headingY = 138
    textWidth = 245
    listY = 142
    bandLeft = 37
    bandRight = 283
    headingTallest = 39
    // where a message with a list under it may go, under the rail of the picture
    screenBottom = 198
    textTopmost = 113
    portraitX = 107
    portraitY = 13
    mirrorX = 95
    mirrorY = 1
    leftEyeX = 64
    rightEyeX = 233
    eyesY = 58
    // the original draws a frame of the wizard every 3 of its redraws, 18.2 a second
    talkFrameTicks = 10
    fadeInFrames = 5
    talkFramesMost = 24
)

// by eye, see above
var bandColor = color.RGBA{A: 90}

// what the game gives the screen
type Setup struct {
    Cache *lbx.LbxCache
    Human *playerlib.Player
    Other *playerlib.Player
    Rules *relations.Rules
    // what the wizard says at the end of a turn; nil when the human opens the talk
    Message *relations.Message
    Year int
    // plays the music of the mood of what is said; may be nil
    PlaySong func(music.Song)
}

type listItem struct {
    Text string
    Active bool
}

type screen struct {
    Setup
    imageCache util.ImageCache
    ui *uilib.UI
    messages *Messages
    text *font.StyledFont
    item *font.StyledFont
    itemOff *font.StyledFont
    wizard int
    cutout util.ImageTransformFunc

    // what is on screen: the lines said, and a list under them
    lines []string
    linesY int
    lineStep int
    items []listItem
    itemsY int
    hover int

    // the face: 0 and 1 pleased, 2 angry, below 0 no wizard in the mirror
    group int
    fading bool
    fadeFrame int
    talking bool
    talkFrame int
    counter uint64
    // when the last picture of the fade in came, and when the fade in ended (portraitBefore)
    fadeStepAt uint64
    fadeEndedAt uint64
    song music.Song

    clicked bool
    chosen int
    quit bool
    fadeOut float64
}

// the wizard pictures of a wizard: the original's 14 in its order
func wizardIndex(player *playerlib.Player) int {
    switch player.Wizard.Base {
        case data.WizardMerlin: return 0
        case data.WizardRaven: return 1
        case data.WizardSharee: return 2
        case data.WizardLoPan: return 3
        case data.WizardJafar: return 4
        case data.WizardOberic: return 5
        case data.WizardRjak: return 6
        case data.WizardSssra: return 7
        case data.WizardTauron: return 8
        case data.WizardFreya: return 9
        case data.WizardHorus: return 10
        case data.WizardAriel: return 11
        case data.WizardTlaloc: return 12
        case data.WizardKali: return 13
    }
    return 0
}

func (screen *screen) wizards() int {
    count := 0
    if screen.Rules != nil {
        for _, player := range screen.Rules.Players {
            if !player.IsNeutral() {
                count += 1
            }
        }
    }
    return count
}

// the human's view of the wizard
func (screen *screen) relation() *playerlib.Relationship {
    relation, ok := screen.Human.GetDiplomaticRelation(screen.Other)
    if !ok {
        return nil
    }
    return relation
}

func (screen *screen) words() Words {
    words := Words{Human: screen.Human.Wizard.Name, Wizard: screen.Other.Wizard.Name, Year: screen.Year}
    if relation := screen.relation(); relation != nil {
        words.Treaty = relation.Treaty
        words.BrokenTreaty = relation.BrokenTreaty
    }
    if message := screen.Message; message != nil {
        words.City = message.City
        words.CitySize = message.CitySize
        words.Spell = message.Spell
        words.WantSpell = message.WantSpell
        words.Gold = message.OfferGold
        words.BrokenTreaty = message.BrokenTreaty
        words.Treaty = message.Treaty
        if words.Treaty == data.TreatyNone || words.Treaty == data.TreatyWar {
            // a treaty that was broken by what is said
            words.Treaty = message.BrokenTreaty
        }
    }
    return words
}

// breaks a text into lines of the width
func wrapLines(style *font.StyledFont, text string, width int) []string {
    var lines []string
    for _, paragraph := range strings.Split(text, "\n") {
        line := ""
        for _, word := range strings.Split(paragraph, " ") {
            longer := word
            if line != "" {
                longer = line + " " + word
            }
            if line != "" && style.Width(longer) > width {
                lines = append(lines, line)
                line = word
            } else {
                line = longer
            }
        }
        lines = append(lines, line)
    }
    return lines
}

// the music of a mood: 2 the angry one
func (screen *screen) playMood(mood int) {
    song := GetSong(screen.Other, mood == 2)
    if song != screen.song && screen.PlaySong != nil {
        screen.PlaySong(song)
    }
    screen.song = song
}

// the wizard says a message of the original; the face and music follow its record
func (screen *screen) show(record int, words Words, extra string) {
    pick := 0
    if screen.messages != nil {
        pick = screen.messages.Pick(record, screen.wizards(), relations.Roll)
    }
    text := screen.messages.Text(record, pick, words)
    if extra != "" {
        text += " " + extra
    }
    line := screen.messages.record(record)
    screen.playMood(line.Mood)
    switch line.Group {
        case 0, 1: screen.group = 1
        case 2: screen.group = 2
        default: screen.group = -1
    }
    screen.talking = screen.group >= 0
    screen.talkFrame = 0
    screen.setText(text, textY, 1)
    screen.items = nil
    log.Printf("Diplomacy: %v says (%v): %v", screen.Other.Wizard.Name, record, text)
}

func (screen *screen) setText(text string, y int, gap int) {
    screen.lines = wrapLines(screen.text, text, textWidth)
    screen.linesY = y
    screen.lineStep = screen.text.Height() + gap
}

// development: a capture clicks this many messages away, a second after each shows
var CaptureAdvance = 0

func (screen *screen) waitClick(yield coroutine.YieldFunc) {
    screen.clicked = false
    screen.setElements()
    waited := 0
    for !screen.clicked {
        screen.update()
        waited += 1
        if CaptureAdvance > 0 && waited > 60 {
            CaptureAdvance -= 1
            screen.clicked = true
        }
        yield()
    }
}

// says a message and waits for a click
func (screen *screen) say(yield coroutine.YieldFunc, record int, words Words) {
    screen.show(record, words, "")
    screen.waitClick(yield)
}

// Diplomacy_Display_Response for most answers: one time in 4 the thanks for a spell given before
func (screen *screen) respond(yield coroutine.YieldFunc, record int, words Words) {
    if screen.Rules != nil {
        if spell, ok := screen.Rules.GiftAnswer(screen.Human, screen.Other); ok {
            record = relations.RecordGiftRemembered
            words.GivenSpell = spell
        }
    }
    screen.say(yield, record, words)
}

// a list to pick from under a heading; -1 for none (a right click)
func (screen *screen) ask(yield coroutine.YieldFunc, heading string, items []listItem) int {
    screen.setText(heading, headingY, 1)
    if len(screen.lines) * screen.lineStep > headingTallest {
        screen.setText(heading, headingY, 0)
    }
    return screen.pick(yield, items, listY + screen.item.LineHeight())
}

// a message of the original with a list under it
func (screen *screen) sayAndAsk(yield coroutine.YieldFunc, record int, words Words, extra string, items []listItem) int {
    screen.show(record, words, extra)
    // a long message moves up so the list under it stays on the screen
    bottom := screen.linesY + len(screen.lines) * screen.lineStep + 2 + len(items) * screen.item.LineHeight()
    if bottom > screenBottom {
        screen.linesY = max(screen.linesY - (bottom - screenBottom), textTopmost)
    }
    return screen.pick(yield, items, screen.linesY + len(screen.lines) * screen.lineStep + 2)
}

func (screen *screen) pick(yield coroutine.YieldFunc, items []listItem, firstY int) int {
    screen.items = items
    screen.itemsY = firstY
    screen.hover = -1
    screen.chosen = -2
    screen.setElements()
    for screen.chosen == -2 {
        screen.update()
        yield()
    }
    screen.items = nil
    return screen.chosen
}

func (screen *screen) itemRect(index int) image.Rectangle {
    y := screen.itemsY + index * screen.item.LineHeight()
    return image.Rect(bandLeft, y, bandRight, y + screen.item.Height() + 1)
}

func (screen *screen) setElements() {
    // a right click anywhere: no choice in a list, else the same as a left click
    elements := []*uilib.UIElement{&uilib.UIElement{
        Rect: image.Rect(0, 0, 320, 200),
        RightClick: func(element *uilib.UIElement) {
            if len(screen.items) > 0 {
                screen.chosen = -1
            } else {
                screen.clicked = true
            }
        },
    }}
    for index, item := range screen.items {
        elements = append(elements, &uilib.UIElement{
            Rect: screen.itemRect(index),
            LeftClick: func(element *uilib.UIElement) {
                if item.Active {
                    screen.chosen = index
                }
            },
            RightClick: func(element *uilib.UIElement) {
                screen.chosen = -1
            },
            Inside: func(element *uilib.UIElement, x int, y int) {
                if item.Active {
                    screen.hover = index
                }
            },
            NotInside: func(element *uilib.UIElement) {
                if screen.hover == index {
                    screen.hover = -1
                }
            },
        })
    }
    screen.ui.SetElementsFromArray(elements)
}

func (screen *screen) update() {
    screen.ui.StandardUpdate()
    screen.counter += 1
    if screen.fading {
        // a picture of the fade in every talkFrameTicks from its start, so the blend of one into
        // the next (portraitBefore) runs evenly
        if screen.counter - screen.fadeStepAt >= talkFrameTicks {
            screen.fadeFrame += 1
            screen.fadeStepAt = screen.counter
            if screen.fadeFrame >= fadeInFrames {
                screen.fading = false
                screen.fadeEndedAt = screen.counter
            }
        }
    } else if screen.counter % talkFrameTicks == 0 {
        if screen.talking {
            screen.talkFrame += 1
            images, _ := screen.imageCache.GetImagesTransform("diplomac.lbx", 24 + screen.wizard, "cutout", screen.cutout)
            // the original stops the talk at its frame 24
            if screen.talkFrame >= min(len(images), talkFramesMost) {
                screen.talking = false
            }
        }
    }
}

// Diplomacy_Screen_Fade_In
func (screen *screen) fadeIn(yield coroutine.YieldFunc) {
    screen.fading = true
    screen.fadeFrame = 0
    screen.fadeStepAt = screen.counter
    screen.fadeEndedAt = 0
    screen.lines = nil
    for {
        screen.update()
        // (user 2026-10-03: "right at the end of the fade-up, for just a frame ... the portrait
        // popping out for a moment, exposing the background image"): the update that ends the fade is
        // not followed by a picture; the one after it already has the face of the message that comes
        // next. Before, the picture between had no fade and no face (group -1), so the mirror was empty
        if !screen.fading {
            return
        }
        yield()
    }
}

// the talk the human opens (Diplomacy_Screen__WIP, Diplomacy_Greeting, Get_Main_Diplomacy_Choices)
func (screen *screen) humanTalk(yield coroutine.YieldFunc) {
    rules := screen.Rules
    greeting := rules.Greeting(screen.Human, screen.Other)
    if greeting == relations.RecordNoAudience {
        screen.say(yield, greeting, screen.words())
        return
    }
    screen.fadeIn(yield)
    screen.say(yield, greeting, screen.words())

    for {
        if rules.Impatient(screen.Human, screen.Other) {
            screen.respond(yield, relations.RecordImpatient, screen.words())
            return
        }
        atWar := false
        if relation := screen.relation(); relation != nil {
            atWar = relation.Treaty == data.TreatyWar
        }
        choice := screen.ask(yield, "How may I serve you:", []listItem{
            {Text: "[ Propose Treaty", Active: true},
            {Text: "[ Threaten/Break Treaty", Active: !atWar},
            {Text: "[ Offer Tribute", Active: true},
            {Text: "[ Exchange Spells", Active: !atWar},
            {Text: "[ Good Bye", Active: true},
        })
        switch choice {
            case 0: screen.proposeTreaty(yield)
            case 1: screen.threaten(yield)
            case 2:
                if screen.tribute(yield) {
                    return
                }
            case 3: screen.exchange(yield)
            default: return
        }
    }
}

// Diplomacy_Propose_Treaty__WIP
func (screen *screen) proposeTreaty(yield coroutine.YieldFunc) {
    treaty := data.TreatyNone
    if relation := screen.relation(); relation != nil {
        treaty = relation.Treaty
    }
    // the original never enables the war on another wizard, and the wizard to break an alliance
    // with is not reconstructed (IDK_Npc_Target_Player__STUB)
    choice := screen.ask(yield, "You propose a treaty: ", []listItem{
        {Text: "[ Wizard Pact", Active: treaty == data.TreatyNone},
        {Text: "[ Alliance", Active: treaty < data.TreatyAlliance},
        {Text: "[ Peace Treaty", Active: treaty >= data.TreatyWar},
        {Text: "[ Declaration of War on Another Wizard", Active: false},
        {Text: "[ Break Alliance With Another Wizard", Active: false},
        {Text: "[ Forget It", Active: true},
    })
    kinds := map[int]data.TreatyType{0: data.TreatyPact, 1: data.TreatyAlliance, 2: data.TreatyNone}
    accepted := map[int]int{0: relations.RecordPactAccepted, 1: relations.RecordAllianceAccepted, 2: relations.RecordPeaceAccepted}
    kind, ok := kinds[choice]
    if !ok {
        return
    }
    if screen.Rules.ProposeTreaty(screen.Human, screen.Other, kind) == relations.AnswerAccepted {
        screen.respond(yield, accepted[choice], screen.words())
    } else {
        screen.say(yield, screen.Rules.RefusalRecord(screen.Human, screen.Other), screen.words())
    }
}

// Diplomacy_Break_Treaty
func (screen *screen) threaten(yield coroutine.YieldFunc) {
    rules := screen.Rules
    treaty := data.TreatyNone
    if relation := screen.relation(); relation != nil {
        treaty = relation.Treaty
    }
    choice := screen.ask(yield, "Your actions: ", []listItem{
        {Text: "[ Break Wizard Pact", Active: treaty == data.TreatyPact},
        {Text: "[ Break Alliance", Active: treaty == data.TreatyAlliance},
        {Text: "[ Threaten To Attack", Active: true},
        {Text: "[ Forget It", Active: true},
    })
    words := screen.words()
    record := -1
    switch choice {
        case 0, 1:
            words.BrokenTreaty = treaty
            rules.HumanBreaksTreaty(screen.Human, screen.Other)
            record = relations.RecordTreatyBrokenByHuman
        case 2:
            if answer, ok := rules.Threaten(screen.Human, screen.Other); ok {
                record = answer.Record
                words.Number = answer.Gold
                words.GivenSpell = answer.Spell
            }
    }
    // every visit of this list wears the patience down, even a Forget It
    rules.AfterThreats(screen.Human, screen.Other)
    if record >= 0 {
        screen.respond(yield, record, words)
    }
}

// Diplomacy_Offer_Tribute. true: the wizard had enough of talk
func (screen *screen) tribute(yield coroutine.YieldFunc) bool {
    rules := screen.Rules
    amounts := relations.TributeGoldAmounts(screen.Human.Gold)
    var items []listItem
    for _, amount := range amounts {
        items = append(items, listItem{Text: "[ " + itoa(amount) + " gold", Active: true})
    }
    items = append(items, listItem{Text: "[ Spells", Active: true}, listItem{Text: "[ Forget It", Active: true})
    choice := screen.ask(yield, "What do you offer as tribute?", items)
    switch {
        case choice < 0 || choice == len(amounts) + 1:
            return false
        case choice < len(amounts):
            rules.TributeGold(screen.Human, screen.Other, choice, amounts[choice])
            screen.respond(yield, relations.RecordTributeThanks, screen.words())
            return false
    }
    spells := rules.TributeSpells(screen.Human, screen.Other)
    if len(spells) == 0 {
        screen.respond(yield, relations.RecordImpatient, screen.words())
        return true
    }
    items = nil
    for _, spell := range spells {
        items = append(items, listItem{Text: "[ " + spell.Name, Active: true})
    }
    items = append(items, listItem{Text: "[ Forget It", Active: true})
    choice = screen.ask(yield, "What do you offer as tribute?", items)
    if choice >= 0 && choice < len(spells) {
        rules.TributeSpell(screen.Human, screen.Other, spells[choice])
        screen.respond(yield, relations.RecordTributeThanks, screen.words())
    }
    return false
}

// Diplomacy_Exchange_Spell__WIP
func (screen *screen) exchange(yield coroutine.YieldFunc) {
    rules := screen.Rules
    defer rules.EndExchange(screen.Human, screen.Other)
    offers := rules.ExchangeOffers(screen.Human, screen.Other)
    if len(offers) == 0 {
        screen.respond(yield, relations.RecordNothingToTrade, screen.words())
        return
    }
    var items []listItem
    for _, offer := range offers {
        items = append(items, listItem{Text: "[ " + offer.Spell.Name, Active: true})
    }
    items = append(items, listItem{Text: "[ Forget It", Active: true})
    choice := screen.ask(yield, "What type of spell interests you?", items)
    if choice < 0 || choice >= len(offers) {
        return
    }
    offer := offers[choice]
    items = nil
    for _, spell := range offer.For {
        items = append(items, listItem{Text: "[ " + spell.Name, Active: true})
    }
    items = append(items, listItem{Text: "[ Forget It", Active: true})
    mine := screen.ask(yield, "What will you trade for it?", items)
    if mine >= 0 && mine < len(offer.For) {
        rules.Exchange(screen.Human, screen.Other, offer.Spell, offer.For[mine])
    }
}

// the original's words for an offer that comes with a proposal
func offerWords(message *relations.Message) string {
    offer := ""
    if message.OfferGold > 0 {
        offer = itoa(message.OfferGold) + " gold"
    } else if message.OfferSpell != "" {
        offer = message.OfferSpell
    }
    if offer == "" {
        return ""
    }
    return "What if we were to also offer " + offer + " as an incentive?"
}

// what the wizard says at the end of a turn (Npc_Diplomacy_Screen)
func (screen *screen) computerTalk(yield coroutine.YieldFunc) {
    rules := screen.Rules
    message := screen.Message
    order := message.Action
    words := screen.words()
    proposal := order == relations.ActionProposePact || order == relations.ActionProposeAlliance || order == relations.ActionProposePeace

    screen.fadeIn(yield)
    if !proposal {
        screen.respond(yield, order, words)
        return
    }

    screen.say(yield, relations.RecordGreetingGood, words)
    // Npc_Proposal__WIP: first the ally of the human the wizard is at war with
    if ally := rules.AllyToAbandon(screen.Human, screen.Other); ally != nil {
        words.BreakWith = ally.Wizard.Name
        answer := screen.sayAndAsk(yield, relations.RecordBreakYourTreaty, words, "", []listItem{
            {Text: "[ Agree", Active: true},
            {Text: "[ Forget It", Active: true},
        })
        rules.AnswerAbandon(screen.Human, screen.Other, ally, answer == 0)
        if answer != 0 {
            rules.AnswerProposal(screen.Human, screen.Other, *message, false)
            return
        }
    }
    answer := screen.sayAndAsk(yield, order, words, offerWords(message), []listItem{
        {Text: "[ Accept", Active: true},
        {Text: "[ Reject", Active: true},
    })
    rules.AnswerProposal(screen.Human, screen.Other, *message, answer == 0)
}

func itoa(value int) string {
    return strconv.Itoa(value)
}

// the screen of diplomacy: the talk the human opens, or what a wizard says at the end of a turn
func ShowDiplomacyScreen(setup Setup) (func (coroutine.YieldFunc), func (*ebiten.Image)) {
    original := fontslib.LoadOriginal(setup.Cache, "diplomac.lbx", 0)
    if original == nil || setup.Rules == nil {
        log.Printf("Unable to make the screen of diplomacy")
        return func (coroutine.YieldFunc) {}, func (*ebiten.Image) {}
    }

    screen := &screen{
        Setup: setup,
        imageCache: util.MakeImageCache(setup.Cache),
        messages: LoadMessages(setup.Cache),
        text: font.MakeStyledFont(original.Fonts[4], original.Set(4, 187), font.ShadowDown, color.Black),
        item: original.Style(1, original.Set(4, 187), font.ShadowNone, -1),
        itemOff: original.Style(1, original.Set(3, 187), font.ShadowNone, -1),
        wizard: wizardIndex(setup.Other),
        group: -1,
        song: music.SongNone,
    }
    screen.ui = &uilib.UI{
        Draw: func (ui *uilib.UI, image *ebiten.Image) {
            ui.StandardDraw(image)
        },
        LeftClick: func() {
            screen.clicked = true
        },
    }

    // the portrait is cut to the shape of the mirror (any fade in picture will do)
    diplomacLbx, err := setup.Cache.GetLbxFile("diplomac.lbx")
    if err == nil {
        maskSprites, err := diplomacLbx.ReadImages(46)
        if err == nil && len(maskSprites) > 0 {
            mask := maskSprites[0]
            screen.cutout = func (img *image.Paletted) image.Image {
                bounds := img.Bounds().Intersect(mask.Bounds())
                out := image.NewPaletted(bounds, img.Palette)
                for x := bounds.Min.X; x < bounds.Max.X; x++ {
                    for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
                        _, _, _, alpha := mask.At(x, y).RGBA()
                        if alpha > 0 {
                            out.SetColorIndex(x, y, img.ColorIndexAt(x, y))
                        }
                    }
                }
                return out
            }
        }
    }
    if screen.cutout == nil {
        screen.cutout = func (img *image.Paletted) image.Image { return img }
    }

    logic := func (yield coroutine.YieldFunc) {
        if setup.Message == nil {
            screen.humanTalk(yield)
        } else {
            screen.computerTalk(yield)
        }
        screen.quit = true
        screen.lines = nil
        for screen.fadeOut < 1 {
            screen.fadeOut += 0.07
            yield()
        }
    }

    return logic, screen.draw
}

func (screen *screen) draw(destination *ebiten.Image) {
    var options ebiten.DrawImageOptions
    background, _ := screen.imageCache.GetImage("diplomac.lbx", 0, 0)
    if background != nil {
        scale.DrawScaled(destination, background, &options)
    }

    level := 5
    if relation := screen.relation(); relation != nil {
        level = playerlib.RelationLevel(relation.VisibleRelation)
    }
    leftEye, _ := screen.imageCache.GetImage("diplomac.lbx", 2 + level, 0)
    rightEye, _ := screen.imageCache.GetImage("diplomac.lbx", 13 + level, 0)
    if leftEye != nil {
        options.GeoM.Reset()
        options.GeoM.Translate(leftEyeX, eyesY)
        scale.DrawScaled(destination, leftEye, &options)
    }
    if rightEye != nil {
        options.GeoM.Reset()
        options.GeoM.Translate(rightEyeX, eyesY)
        scale.DrawScaled(destination, rightEye, &options)
    }

    if portrait := screen.portrait(); portrait != nil {
        options.GeoM.Reset()
        options.GeoM.Translate(portraitX, portraitY)
        // the picture before under it while this one comes up (portraitBefore)
        previous, part := screen.portraitBefore()
        if previous != nil {
            scale.DrawScaled(destination, previous, &options)
        }
        portraitOptions := options
        portraitOptions.ColorScale.ScaleAlpha(part)
        scale.DrawScaled(destination, portrait, &portraitOptions)
    }
    mirror, _ := screen.imageCache.GetImage("backgrnd.lbx", 18, 0)
    if mirror != nil {
        options.GeoM.Reset()
        options.GeoM.Translate(mirrorX, mirrorY)
        scale.DrawScaled(destination, mirror, &options)
    }

    fontOptions := font.FontOptions{Scale: scale.ScaleAmount}
    for index, line := range screen.lines {
        screen.text.Print(destination, textX, screen.linesY + index * screen.lineStep, fontOptions, line)
    }
    for index, item := range screen.items {
        rect := screen.itemRect(index)
        if index == screen.hover {
            // Gradient_Fill of the list: x - 1 to the end of the field, from half the gap under the
            // top of the line to its bottom
            top := rect.Min.Y + max(1, (screen.item.LineHeight() - screen.item.Height()) / 2) + 1
            vector.FillRect(destination, scale.Scale(float32(rect.Min.X)), scale.Scale(float32(top)), scale.Scale(float32(rect.Dx())), scale.Scale(float32(rect.Max.Y - top)), bandColor, false)
        }
        style := screen.item
        if !item.Active {
            style = screen.itemOff
        }
        style.Print(destination, textX, rect.Min.Y, fontOptions, item.Text)
    }

    if screen.quit {
        alpha := min(max(255 * screen.fadeOut, 0), 255)
        vector.FillRect(destination, 0, 0, float32(destination.Bounds().Dx()), float32(destination.Bounds().Dy()), color.RGBA{A: uint8(alpha)}, false)
    }
}

// (user 2026-10-02: the reflection lines "pop out as soon as the wizard start talking"): the
// original's fade in is 5 pictures (diplomac 38 + wizard), the wizard behind glass streaks that
// fade from picture to picture; its last still has faint streaks and the face that follows none,
// and every picture came at once. Now each picture comes up over the one before in talkFrameTicks
// (the first over the empty mirror) and the face over the last picture of the fade in. Returns the
// picture under the one of now (nil: none) and how much of the one of now shows
func (screen *screen) portraitBefore() (*ebiten.Image, float32) {
    fadeImages, _ := screen.imageCache.GetImagesTransform("diplomac.lbx", 38 + screen.wizard, "cutout", screen.cutout)
    if len(fadeImages) == 0 {
        return nil, 1
    }
    if screen.fading {
        part := min(1, float32(screen.counter - screen.fadeStepAt) / talkFrameTicks)
        if screen.fadeFrame == 0 {
            return nil, part
        }
        return fadeImages[min(screen.fadeFrame - 1, len(fadeImages) - 1)], part
    }
    if screen.fadeEndedAt > 0 && screen.counter - screen.fadeEndedAt < talkFrameTicks {
        return fadeImages[len(fadeImages) - 1], float32(screen.counter - screen.fadeEndedAt) / talkFrameTicks
    }
    return nil, 1
}

// the wizard in the mirror as it is now
func (screen *screen) portrait() *ebiten.Image {
    if screen.group < 0 && !screen.fading {
        return nil
    }
    if screen.fading {
        images, _ := screen.imageCache.GetImagesTransform("diplomac.lbx", 38 + screen.wizard, "cutout", screen.cutout)
        if len(images) == 0 {
            return nil
        }
        return images[min(screen.fadeFrame, len(images) - 1)]
    }
    if screen.talking {
        images, _ := screen.imageCache.GetImagesTransform("diplomac.lbx", 24 + screen.wizard, "cutout", screen.cutout)
        if len(images) > 0 {
            return images[min(screen.talkFrame, len(images) - 1)]
        }
    }
    // moodwiz: 0 pleased, 1 angry
    mood := 0
    if screen.group == 2 {
        mood = 1
    }
    face, _ := screen.imageCache.GetImageTransform("moodwiz.lbx", screen.wizard, mood, "cutout", screen.cutout)
    return face
}

// the music of the wizard, the angry one or not
func GetSong(enemy *playerlib.Player, mad bool) music.Song {
    songs := map[data.WizardBase][2]music.Song{
        data.WizardMerlin: {music.SongMerlin, music.SongMerlinMad},
        data.WizardRaven: {music.SongRaven, music.SongRavenMad},
        data.WizardSharee: {music.SongSharee, music.SongShareeMad},
        data.WizardLoPan: {music.SongLoPan, music.SongLoPanMad},
        data.WizardJafar: {music.SongJafar, music.SongJafarMad},
        data.WizardOberic: {music.SongOberic, music.SongObericMad},
        data.WizardRjak: {music.SongRjak, music.SongRjakMad},
        data.WizardSssra: {music.SongSssra, music.SongSssraMad},
        data.WizardTauron: {music.SongTauron, music.SongTauronMad},
        data.WizardFreya: {music.SongFreya, music.SongFreyaMad},
        data.WizardHorus: {music.SongHorus, music.SongHorusMad},
        data.WizardAriel: {music.SongAriel, music.SongArielMad},
        data.WizardTlaloc: {music.SongTlaloc, music.SongTlalocMad},
        data.WizardKali: {music.SongKali, music.SongKaliMad},
    }
    pair, ok := songs[enemy.Wizard.Base]
    if !ok {
        return music.SongNone
    }
    if mad {
        return pair[1]
    }
    return pair[0]
}
