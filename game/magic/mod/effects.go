package mod

// The values of the effects of spells, from a text file in the replacement folder.
//
//   effects.txt
//
//   # a note
//   [Fire Bolt]
//   burst-count = 40
//   trail-colors = fff0a0 ffc020 ff7010
//
// A part in brackets is the name of a spell as its folder in spells/ has it, or default for all
// spells. What the file does not name stays as the game has it. The game makes of the values what
// it needs, see game/magic/combat/spelleffects.go.
//
// The file is looked at again once in a while: a value that was changed shows with the next spell,
// without starting the game or the battle again.

import (
    "bufio"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "time"
)

const EffectsFile = "effects.txt"
const EffectsDefault = "default"

// the file is looked at again after this long
const effectsCheckTime = time.Second

var effectsValues map[string]map[string]string
var effectsRead time.Time
var effectsChanged time.Time
var effectsVersion int

// the values of a text in the form of the file: by part, in small letters, and by name
func ParseEffects(text string) map[string]map[string]string {
    out := make(map[string]map[string]string)
    part := EffectsDefault

    scanner := bufio.NewScanner(strings.NewReader(text))
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if at := strings.Index(line, "#"); at >= 0 {
            line = strings.TrimSpace(line[:at])
        }
        if line == "" {
            continue
        }

        if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
            part = strings.ToLower(strings.TrimSpace(line[1:len(line) - 1]))
            continue
        }

        name, value, ok := strings.Cut(line, "=")
        if !ok {
            continue
        }

        if out[part] == nil {
            out[part] = make(map[string]string)
        }
        out[part][strings.ToLower(strings.TrimSpace(name))] = strings.TrimSpace(value)
    }

    return out
}

// the values of the file, and a number that is another one whenever they have changed
func Effects() (map[string]map[string]string, int) {
    if time.Since(effectsRead) < effectsCheckTime {
        return effectsValues, effectsVersion
    }
    effectsRead = time.Now()

    if folder == "" {
        if effectsValues != nil {
            effectsValues = nil
            effectsVersion += 1
        }
        return effectsValues, effectsVersion
    }

    path := filepath.Join(folder, EffectsFile)
    info, err := os.Stat(path)
    if err != nil {
        if effectsValues != nil {
            effectsValues = nil
            effectsVersion += 1
            effectsChanged = time.Time{}
        }
        return effectsValues, effectsVersion
    }

    if effectsValues != nil && info.ModTime().Equal(effectsChanged) {
        return effectsValues, effectsVersion
    }

    content, err := os.ReadFile(path)
    if err != nil {
        return effectsValues, effectsVersion
    }

    effectsValues = ParseEffects(string(content))
    effectsChanged = info.ModTime()
    effectsVersion += 1
    reported[path] = false
    reportOnce(fmt.Sprintf("Effects of spells are read from %v", path))

    return effectsValues, effectsVersion
}
