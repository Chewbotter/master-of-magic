package player

// The personality and objective of a computer wizard, as the original game has them (the ReMoM
// project's reconstruction, MoX/src/MOM_DAT.h: PRS_*, OBJ_*; MoX/src/MOM_DAT.c: the names). What
// they change is in diplomacy/rules*.go and ai/chewbot*.go. The human's stay the first of each, as
// in the original.

type Personality int

const (
    PersonalityManiacal Personality = iota
    PersonalityRuthless
    PersonalityAggressive
    PersonalityChaotic
    PersonalityLawful
    PersonalityPeaceful
)

func (personality Personality) String() string {
    switch personality {
        case PersonalityManiacal: return "Maniacal"
        case PersonalityRuthless: return "Ruthless"
        case PersonalityAggressive: return "Aggressive"
        case PersonalityChaotic: return "Chaotic"
        case PersonalityLawful: return "Lawful"
        case PersonalityPeaceful: return "Peaceful"
    }
    return "Unknown"
}

type Objective int

const (
    ObjectivePragmatist Objective = iota
    ObjectiveMilitarist
    ObjectiveTheurgist
    ObjectivePerfectionist
    ObjectiveExpansionist
)

func (objective Objective) String() string {
    switch objective {
        case ObjectivePragmatist: return "Pragmatist"
        case ObjectiveMilitarist: return "Militarist"
        case ObjectiveTheurgist: return "Theurgist"
        case ObjectivePerfectionist: return "Perfectionist"
        case ObjectiveExpansionist: return "Expansionist"
    }
    return "Unknown"
}
