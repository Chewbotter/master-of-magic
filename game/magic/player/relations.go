package player

import (
    "github.com/kazzmir/master-of-magic/game/magic/setup"
    "github.com/kazzmir/master-of-magic/game/magic/data"
)

// the relation two wizards start with, by their books: Init_Diplomatic_Relations of the original
// (the ReMoM project's reconstruction, MoM/src/INITGAME.c). Life against death costs the most, chaos
// costs, books of sorcery, chaos, nature and life both have bring them closer; at least -90
func computeStartingRelation(wizard1 setup.WizardCustom, wizard2 setup.WizardCustom) int {
    books1 := make(map[data.MagicType]int)
    books2 := make(map[data.MagicType]int)
    for _, book := range wizard1.Books {
        books1[book.Magic] += book.Count
    }
    for _, book := range wizard2.Books {
        books2[book.Magic] += book.Count
    }

    maxDeath := max(books1[data.DeathMagic], books2[data.DeathMagic])
    sumLife := books1[data.LifeMagic] + books2[data.LifeMagic]
    maxChaos := max(books1[data.ChaosMagic], books2[data.ChaosMagic])

    score := 0
    if sumLife > 0 && maxDeath > 0 {
        score -= (sumLife + maxDeath) * 5
    } else {
        score += sumLife * 2
        score -= maxDeath * 3
    }
    score -= maxChaos * 2
    for _, magic := range []data.MagicType{data.SorceryMagic, data.ChaosMagic, data.NatureMagic, data.LifeMagic} {
        score += min(books1[magic], books2[magic]) * 2
    }
    return max(score, -90)
}
