package game

// Random events as the original game has them. Rules in words: docs/mod/events.md.
//
// The facts were read from the ReMoM project's reconstruction (MoM/src/EVENTS.c: Determine_Event,
// Event_Twiddle, Get_Event_Victim, Pick_Random_City; INITGAME.c: the last event at turn 50).
// Determine_Event and Event_Twiddle are marked there as checked against the program. The code is
// ours.

import (
    "math/rand/v2"

    "github.com/kazzmir/master-of-magic/game/magic/artifact"
    citylib "github.com/kazzmir/master-of-magic/game/magic/city"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

// false: upstream's random events and conquest details
var ClassicEvents = true

// the turn the original counts the first event from
const classicFirstEventTurn = 50

// the original's order of the 18 events (Random(18))
var classicEventOrder = []RandomEventType{
    RandomEventGreatMeteor, RandomEventGift, RandomEventDisjunction, RandomEventDiplomaticMarriage,
    RandomEventEarthquake, RandomEventPiracy, RandomEventPlague, RandomEventRebellion,
    RandomEventDonation, RandomEventDepletion, RandomEventNewMinerals, RandomEventPopulationBoom,
    RandomEventGoodMoon, RandomEventBadMoon, RandomEventConjunctionChaos, RandomEventConjunctionNature,
    RandomEventConjunctionSorcery, RandomEventManaShort,
}

// m_event_good_array: the events that are good for their victim
func classicGoodEvent(kind RandomEventType) bool {
    switch kind {
        case RandomEventGreatMeteor, RandomEventEarthquake, RandomEventPiracy, RandomEventPlague,
             RandomEventRebellion, RandomEventDepletion, RandomEventManaShort:
            return false
    }
    return true
}

func classicMoonOrConjunction(kind RandomEventType) bool {
    switch kind {
        case RandomEventGoodMoon, RandomEventBadMoon, RandomEventConjunctionChaos, RandomEventConjunctionNature,
             RandomEventConjunctionSorcery, RandomEventManaShort:
            return true
    }
    return false
}

// Determine_Event: the pressure is the turns since the last event (the first is counted from
// turn 50), nothing for 5 turns after an event or the start of a session, times the difficulty;
// an event when a roll of 1 to 512 is not above it
func (model *GameModel) classicEventFires() bool {
    last := max(model.LastEventTurn, classicFirstEventTurn)
    pressure := 0
    if turn := model.classicEventTurn(); turn > last {
        pressure = int(turn - last)
    }
    if model.eventDelay < 5 {
        model.eventDelay += 1
        pressure = 0
    }
    switch model.Settings.Difficulty {
        case data.DifficultyIntro: pressure = pressure / 2
        case data.DifficultyEasy: pressure = pressure * 2 / 3
        case data.DifficultyAverage: pressure = pressure * 3 / 4
        case data.DifficultyHard: pressure = pressure * 4 / 5
    }
    return rand.N(512) + 1 <= pressure
}

// Get_Event_Victim: by the astrologer's numbers, the strong for bad events, the weak (600 less)
// for good ones; halved until they sum to 500 or less; nil when they sum to nothing
func (model *GameModel) classicEventVictim(good bool) *playerlib.Player {
    var wizards []*playerlib.Player
    var weights []int
    for _, player := range model.Players {
        // Skip: a player that sits out (the human of a headless run), as upstream's targets
        if player.IsNeutral() || player.Defeated || player.Skip {
            continue
        }
        power := player.LatestWizardPower()
        weight := power.Army + power.Magic + power.SpellResearch
        if good {
            weight = max(0, 600 - weight)
        }
        wizards = append(wizards, player)
        weights = append(weights, weight)
    }
    total := 0
    for {
        total = 0
        for _, weight := range weights {
            total += weight
        }
        if total <= 500 {
            break
        }
        for i := range weights {
            weights[i] /= 2
        }
    }
    if total < 1 {
        return nil
    }
    roll := rand.N(total)
    for i, weight := range weights {
        roll -= weight
        if roll < 0 {
            return wizards[i]
        }
    }
    return wizards[len(wizards) - 1]
}

// Pick_Random_City: a city of the player whose people are content under the race of its capital
// (the unrest of the races 1 or less); MY CALL, the meaning (see improvements.md)
func classicEventCity(player *playerlib.Player) *citylib.City {
    var choices []*citylib.City
    capital := citylib.ClassicRaceIndexOf(player.Wizard.Race)
    for _, city := range player.Cities {
        if city.Outpost {
            continue
        }
        own := citylib.ClassicRaceIndexOf(city.Race)
        if capital >= 0 && own >= 0 && citylib.ClassicRaceUnrest(capital, own) > 1 {
            continue
        }
        choices = append(choices, city)
    }
    if len(choices) == 0 {
        return nil
    }
    return choices[rand.N(len(choices))]
}

// what a classic event is aimed at
type classicEventPick struct {
    Kind RandomEventType
    Victim *playerlib.Player
    City *citylib.City
    NeutralCity *citylib.City
}

// Determine_Event: up to 5 tries of an event and its victim through the original's vetoes
func (model *GameModel) classicPickEvent() (classicEventPick, bool) {
    active := func(kind RandomEventType) *RandomEvent {
        for _, event := range model.RandomEvents {
            if event.Type == kind {
                return event
            }
        }
        return nil
    }
    anyMoon := false
    for _, event := range model.RandomEvents {
        if classicMoonOrConjunction(event.Type) {
            anyMoon = true
        }
    }

    for range 5 {
        kind := classicEventOrder[rand.N(len(classicEventOrder))]
        victim := model.classicEventVictim(classicGoodEvent(kind))
        if victim == nil || victim.Banished || active(kind) != nil {
            continue
        }
        if anyMoon && classicMoonOrConjunction(kind) {
            continue
        }
        if kind == RandomEventPiracy && victim.Gold < 100 {
            continue
        }
        pick := classicEventPick{Kind: kind, Victim: victim}
        switch kind {
            case RandomEventGreatMeteor, RandomEventDiplomaticMarriage, RandomEventEarthquake, RandomEventPlague, RandomEventRebellion, RandomEventPopulationBoom:
                pick.City = classicEventCity(victim)
                if pick.City == nil {
                    continue
                }
        }
        if kind == RandomEventDiplomaticMarriage {
            neutral := model.GetNeutralPlayer()
            if neutral == nil {
                continue
            }
            pick.NeutralCity = classicEventCity(neutral)
            if pick.NeutralCity == nil {
                continue
            }
        }
        if (kind == RandomEventDiplomaticMarriage || kind == RandomEventGreatMeteor) && model.classicEventTurn() < 150 {
            continue
        }
        if kind == RandomEventRebellion {
            if pick.City.HasFortress() {
                continue
            }
            normal, summoned, hero := 0, 0, false
            for _, unit := range victim.GetUnits(pick.City.X, pick.City.Y, pick.City.Plane) {
                if unit.IsHero() {
                    hero = true
                }
                if unit.GetRace() == data.RaceFantastic {
                    summoned += 1
                } else {
                    normal += 1
                }
            }
            if hero || summoned > normal {
                continue
            }
        }
        if kind == RandomEventDisjunction {
            any := false
            for _, player := range model.Players {
                if player.GlobalEnchantments.Size() > 0 {
                    any = true
                }
            }
            if !any {
                continue
            }
        }
        if kind == RandomEventGoodMoon && active(RandomEventBadMoon) != nil || kind == RandomEventBadMoon && active(RandomEventGoodMoon) != nil {
            continue
        }
        if kind == RandomEventPlague {
            if boom := active(RandomEventPopulationBoom); boom != nil && boom.TargetCity == pick.City {
                continue
            }
        }
        if kind == RandomEventPopulationBoom {
            if plague := active(RandomEventPlague); plague != nil && plague.TargetCity == pick.City {
                continue
            }
        }
        return pick, true
    }
    return classicEventPick{}, false
}

// Event_Twiddle: a moon, a conjunction, a mana short ends from its 5th turn with a chance that
// grows: (turns - 3) in 20
func classicMoonEnds(turns uint64) bool {
    if turns < 5 {
        return false
    }
    return uint64(rand.N(20) + 1) <= turns - 3
}

// Determine_Event: a donation of 105 to 600
func classicDonation() int {
    return 100 + (rand.N(100) + 1) * 5
}

// the item of a Gift: a new random item of the original (Make_Item(2, ...)); set by the game,
// which has the archive
var classicGiftItem func() (artifact.Artifact, bool)

// the turn the events are rolled for: the turn just played (EndOfTurn has counted on already,
// the original rolls before its counter moves)
func (model *GameModel) classicEventTurn() uint64 {
    if model.TurnNumber == 0 {
        return 0
    }
    return model.TurnNumber - 1
}
