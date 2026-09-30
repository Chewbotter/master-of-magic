package diplomacy

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/data"
)

// a message as the data has it: words filled in where the codes are, "a"/"an" before a unit
func TestMessageText(test *testing.T) {
    texts := make([]byte, 2 * messageTextsPerRecord * messageTextSize)
    write := func(message int, pick int, parts ...[]byte) {
        start := (message * messageTextsPerRecord + pick) * messageTextSize
        for _, part := range parts {
            copy(texts[start:], part)
            start += len(part)
        }
    }
    code := func(value int) []byte {
        return []byte{byte(messageFirstCode + value)}
    }
    write(1, 0, []byte("In the year "), code(16), []byte(", "), code(1), []byte(" and "), code(0), []byte(" keep a "), code(15), []byte("."))
    write(1, 1, []byte("The summoning of "), code(14), code(13), []byte(" and "), code(14), []byte(" "), code(13), []byte("."))
    messages := &Messages{records: []messageRecord{{}, {Mood: 1, Group: 1, Count: 2}}, texts: texts}

    words := Words{Human: "Merlin", Wizard: "Kali", Year: 1405, Treaty: data.TreatyPact, Unit: "Great Drake"}
    if got, want := messages.Text(1, 0, words), "In the year 1405, Kali and Merlin keep a Wizard Pact."; got != want {
        test.Errorf("got %q, want %q", got, want)
    }
    if got, want := messages.Text(1, 1, words), "The summoning of a Great Drake and a Great Drake."; got != want {
        test.Errorf("got %q, want %q", got, want)
    }

    // the pick: a roll of 1 to 2, less 1; in a game of two wizards less 2, at least 0
    if got := messages.Pick(1, 4, func(n int) int { return n }); got != 1 {
        test.Errorf("pick of the highest roll: %v, want 1", got)
    }
    if got := messages.Pick(1, 2, func(n int) int { return n }); got != 0 {
        test.Errorf("pick in a game of two wizards: %v, want 0", got)
    }
}
