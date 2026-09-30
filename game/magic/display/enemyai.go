package display

// The setting "Enemy AI": which AI plays the computer players (user, 2026-09-29).
//   Clone original  the AI the clone came with (upstream's, which its author made up)
//   Chewbot         the user's: first as close to the original game's AI as it can be ported from
//                   the ReMoM reconstruction, then changed from there
// Only the combat AI of Chewbot is built so far (game/magic/combat/aichewbot.go); everything
// else of Chewbot is still the clone's.
//
// Chewbot unless another is saved: the file of settings keeps "enemy-ai", and a file that says
// nothing of it means Chewbot.

type EnemyAI string

const EnemyAIClone EnemyAI = "clone"
const EnemyAIChewbot EnemyAI = "chewbot"

const DefaultEnemyAI = EnemyAIChewbot

// true when Chewbot plays the computer players
func ChewbotAI() bool {
    return Current != nil && Current.EnemyAI() == EnemyAIChewbot
}

// the AI in use. anything unrecognized counts as the default
func (settings *Settings) EnemyAI() EnemyAI {
    switch settings.EnemyAIName {
        case EnemyAIClone, EnemyAIChewbot:
            return settings.EnemyAIName
    }
    return DefaultEnemyAI
}

func (settings *Settings) SetEnemyAI(ai EnemyAI) {
    settings.EnemyAIName = ai
    settings.Save()
}

// the name shown in the settings screen
func (ai EnemyAI) Name() string {
    switch ai {
        case EnemyAIClone: return "Clone original"
        case EnemyAIChewbot: return "Chewbot"
    }
    return "Chewbot"
}
