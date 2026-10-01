package player

// Development: the journal of key decisions and happenings of a game without a window (-sim
// -sim-journal, see sim.go of the main package and game/simjournal.go). Journal is nil in every
// other game, and then nothing is noted. The AI notes what it decided and why; the game notes what
// happened (battles, events, conquests).

// the sink of the journal: who, the kind of decision ("build", "target", "cast", "war", ...), what
// was decided and why. It may be called from the AI's goroutine.
var Journal func(player *Player, kind string, what string, why string)

// notes a decision or happening when a journal is kept
func Note(player *Player, kind string, what string, why string) {
    if Journal != nil {
        Journal(player, kind, what, why)
    }
}

// true when a journal is kept: build the texts of a note only then
func Noting() bool {
    return Journal != nil
}
