package display

// The setting "Enemy AI": which AI plays the computer players (user, 2026-09-29).
//   Clone original  the AI the clone came with (upstream's, which its author made up)
//   Classic         the original game's AI, ported from the ReMoM reconstruction. In the code it is
//                   called Chewbot (game/magic/ai/chewbot*.go, combat/aichewbot*.go); the user named
//                   it Classic for the screen (2026-10-01)
//
// Classic unless another is saved: the file of settings keeps "enemy-ai" ("chewbot" for Classic),
// and a file that says nothing of it means Classic.

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
        case EnemyAIChewbot: return "Classic"
    }
    return "Classic"
}
