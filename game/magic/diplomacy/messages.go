package diplomacy

// The words of the computer wizards: the original's messages (DIPLOMSG.LBX), picked and filled in
// as the original does. Facts from the ReMoM project's reconstruction (MoM/src/DIPLOMAC.c:
// Get_Diplomacy_Statement). The code is ours; the texts are read from the game's data.
//
//   entry 0: a record of three numbers per message (mood, group, count of texts)
//   entry 1: 15 texts of 200 bytes per message, text = message * 15 + pick
//   the pick: a roll of 1 to the count, less 1 (less 2 in a game of two wizards), at least 0
//   a byte of 128 or more whose lower 7 bits are 20 or less is a word filled in: 0 the human's
//     name, 1 the wizard's, 2 a third wizard, 3 a city, 4 its size, 5 the wizard the human should
//     break with, 7 the wizard of a reward, 8 a number, 9 the spell wanted, 10 "N gold", 11 the
//     treaty broken, 12 the spell of a grievance, 13 a unit, 14 "a"/"an" and the unit, 15 the
//     treaty, 16 the year, 17 the spell given, 18 the wizard of a war, 19 the third wizard, 20 a
//     number
//   the mood is the music (2 the angry one), the group the face: 0 and 1 pleased, 2 angry

import (
    "encoding/binary"
    "fmt"
    "log"
    "strings"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/lib/lbx"
)

const (
    messageTextsPerRecord = 15
    messageTextSize = 200
    messageFirstCode = 128
    messageLastCode = 20
)

type messageRecord struct {
    Mood int
    Group int
    Count int
}

type Messages struct {
    records []messageRecord
    texts []byte
}

// the words filled into a text
type Words struct {
    Human string
    Wizard string
    Third string
    City string
    CitySize string
    BreakWith string
    Number int
    WantSpell string
    Gold int
    BrokenTreaty data.TreatyType
    Spell string
    Unit string
    Treaty data.TreatyType
    Year int
    GivenSpell string
    Target string
}

var loadedMessages = make(map[*lbx.LbxCache]*Messages)

// the messages of the game's data; nil when they can not be read
func LoadMessages(cache *lbx.LbxCache) *Messages {
    if messages, ok := loadedMessages[cache]; ok {
        return messages
    }
    loadedMessages[cache] = nil
    file, err := cache.GetLbxFile("diplomsg.lbx")
    if err != nil {
        log.Printf("Unable to read diplomsg.lbx: %v", err)
        return nil
    }
    recordData, err := file.RawData(0)
    if err != nil || len(recordData) < 4 {
        log.Printf("Unable to read the records of diplomsg.lbx: %v", err)
        return nil
    }
    textData, err := file.RawData(1)
    if err != nil || len(textData) < 4 {
        log.Printf("Unable to read the texts of diplomsg.lbx: %v", err)
        return nil
    }
    messages := &Messages{texts: textData[4:]}
    count := int(binary.LittleEndian.Uint16(recordData[0:]))
    for index := range count {
        start := 4 + index * 6
        if start + 6 > len(recordData) {
            break
        }
        messages.records = append(messages.records, messageRecord{
            Mood: int(int16(binary.LittleEndian.Uint16(recordData[start:]))),
            Group: int(int16(binary.LittleEndian.Uint16(recordData[start + 2:]))),
            Count: int(int16(binary.LittleEndian.Uint16(recordData[start + 4:]))),
        })
    }
    loadedMessages[cache] = messages
    return messages
}

// the mood and group of a message
func (messages *Messages) record(message int) messageRecord {
    if messages == nil || message < 0 || message >= len(messages.records) {
        return messageRecord{Group: 1}
    }
    return messages.records[message]
}

// which of the texts of a message is said: roll gives 1 to n
func (messages *Messages) Pick(message int, wizards int, roll func(int) int) int {
    count := messages.record(message).Count
    if count <= 0 {
        return 0
    }
    pick := roll(count) - 1
    if wizards == 2 {
        pick -= 1
    }
    return max(pick, 0)
}

func treatyWords(treaty data.TreatyType) string {
    switch treaty {
        case data.TreatyAlliance: return "Alliance"
        case data.TreatyPact: return "Wizard Pact"
    }
    return "treaty"
}

func article(name string) string {
    if name != "" && strings.ContainsAny(name[:1], "AEIOUaeiou") {
        return "an " + name
    }
    return "a " + name
}

// the text of a message with its words
func (messages *Messages) Text(message int, pick int, words Words) string {
    if messages == nil || message < 0 {
        return ""
    }
    start := (message * messageTextsPerRecord + pick) * messageTextSize
    if start + messageTextSize > len(messages.texts) {
        return ""
    }
    var out strings.Builder
    text := messages.texts[start:start + messageTextSize]
    for index, value := range text {
        if value == 0 {
            break
        }
        code := int(value & 0x7f)
        if value < messageFirstCode || code > messageLastCode {
            out.WriteByte(value)
            continue
        }
        switch code {
            case 0: out.WriteString(words.Human)
            case 1: out.WriteString(words.Wizard)
            case 2, 19: out.WriteString(words.Third)
            case 3: out.WriteString(words.City)
            case 4: out.WriteString(words.CitySize)
            case 5: out.WriteString(words.BreakWith)
            case 7: out.WriteString(words.Target)
            case 8, 20: out.WriteString(fmt.Sprintf("%v", words.Number))
            case 9: out.WriteString(words.WantSpell)
            case 10: out.WriteString(fmt.Sprintf("%v gold", words.Gold))
            case 11: out.WriteString(treatyWords(words.BrokenTreaty))
            case 12: out.WriteString(words.Spell)
            case 13: out.WriteString(words.Unit)
            case 14:
                // "a" or "an" for the unit that follows; some texts have a space after it, some not
                word := strings.TrimSuffix(article(words.Unit), words.Unit)
                if index + 1 < len(text) && text[index + 1] == ' ' {
                    word = strings.TrimSuffix(word, " ")
                }
                out.WriteString(word)
            case 15: out.WriteString(treatyWords(words.Treaty))
            case 16: out.WriteString(fmt.Sprintf("%v", words.Year))
            case 17: out.WriteString(words.GivenSpell)
            case 18: out.WriteString(words.Target)
        }
    }
    return out.String()
}
