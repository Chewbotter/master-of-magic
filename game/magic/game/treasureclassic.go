package game

// The treasure of the original's lairs, towers and nodes (ReMoM MoM/src/MAPGEN.c: Create_Lair, the
// hoard rolled from the budget of the treasure; MoM/src/Lair.c: Lair_Generate_Treasure,
// EZ_SpecialTreasure, Lair_Magic_Realm, what the hoard gives). The original rolls the hoard when the
// world is made; nobody sees it before the place is taken, so it is rolled here when it is taken,
// from the budget the world gave the place (maplib/classiclairs.go). The rules in words:
// docs/mod/worlds.md. The code is ours.

import (
    rand "github.com/kazzmir/master-of-magic/lib/chance"
    "slices"

    "github.com/kazzmir/master-of-magic/game/magic/artifact"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/maplib"
    "github.com/kazzmir/master-of-magic/game/magic/setup"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    "github.com/kazzmir/master-of-magic/lib/lbx"
    herolib "github.com/kazzmir/master-of-magic/game/magic/hero"
)

const (
    // Lair_Magic_Realm has no case for a tower, so a tower's book is of nature; kept
    quirkTowerBooksNature = true
    // Create_Lair: a second spell adds its rarity to the first, so a rare one comes cheap; kept
    quirkSpellRaritiesAdd = true
    // EZ_SpecialTreasure: a book of arcane (a cave's) lands on the Alchemy retort
    quirkArcaneBookAlchemy = true

    classicTreasureStop = 50
    classicGoldOrManaSpend = 200
    classicItemsMost = 3
    classicItemLeast = 300
    classicPrisonerLeast = 400
    classicPrisonerSpend = 1000
    classicSpecialSpend = 3000
    // EZ_SpecialTreasure: an item of this value, or this much mana with 3 items, when no book or
    // retort could be given
    classicFallbackItem = 1200
    classicFallbackMana = 1000
)

// the original's Random(n): 1 to n
func classicTreasureRoll(n int) int {
    return rand.N(max(n, 1)) + 1
}

// the hoard as Create_Lair leaves it
type classicHoard struct {
    Gold int
    Mana int
    Items []int
    Prisoner bool
    // 1 to 4 a spell of that rarity, 5 one book or retort, 6 two
    SpellOrSpecial int
}

// Create_Lair: the hoard of a budget; a tower first has a spell of rarity 1 to 4, which changes the
// budget by 100 less 50 times the rarity squared
func classicRollHoard(budget int, tower bool) classicHoard {
    var hoard classicHoard
    if tower {
        rarity := classicTreasureRoll(4)
        budget -= rarity * rarity * 50 - 100
        hoard.SpellOrSpecial = rarity
    }
    for budget >= classicTreasureStop {
        switch classicTreasureRoll(15) - 1 {
            case 0, 1:
                hoard.Gold += min(classicTreasureRoll(20) * 10, budget) / 10 * 10
                budget -= classicGoldOrManaSpend
            case 2, 3:
                hoard.Mana += min(classicTreasureRoll(20) * 10, budget) / 10 * 10
                budget -= classicGoldOrManaSpend
            case 4, 5, 6, 7, 8:
                if len(hoard.Items) < classicItemsMost && budget >= classicItemLeast {
                    value := min(300 + classicTreasureRoll(27) * 100, budget) / 10 * 10
                    budget -= value
                    hoard.Items = append(hoard.Items, value)
                }
            case 9:
                if budget >= classicPrisonerLeast && !hoard.Prisoner {
                    hoard.Prisoner = true
                    budget -= classicPrisonerSpend
                }
            case 10, 11, 12:
                rarity := classicTreasureRoll(4)
                if rarity * rarity * 50 <= budget && hoard.SpellOrSpecial < 4 {
                    budget -= rarity * rarity * 50
                    if quirkSpellRaritiesAdd {
                        hoard.SpellOrSpecial = min(hoard.SpellOrSpecial + rarity, 4)
                    } else {
                        hoard.SpellOrSpecial = max(hoard.SpellOrSpecial, rarity)
                    }
                }
            case 13, 14:
                roll := classicTreasureRoll(4)
                switch {
                    case roll == 1 && budget >= 3000: hoard.SpellOrSpecial = 6
                    case budget >= 2000: hoard.SpellOrSpecial = 6
                    case budget >= 1000: hoard.SpellOrSpecial = 5
                    default: continue
                }
                budget -= classicSpecialSpend
        }
    }
    // a book or retort takes the place of everything else
    if hoard.SpellOrSpecial > 4 {
        hoard.Gold, hoard.Mana, hoard.Items, hoard.Prisoner = 0, 0, nil, false
    }
    return hoard
}

// Lair_Magic_Realm: the realm of a book of a place
func classicBookRealm(kind maplib.EncounterType) data.MagicType {
    switch kind {
        case maplib.EncounterTypeChaosNode: return data.ChaosMagic
        case maplib.EncounterTypeNatureNode: return data.NatureMagic
        case maplib.EncounterTypeSorceryNode: return data.SorceryMagic
        case maplib.EncounterTypeDungeon, maplib.EncounterTypeAbandonedKeep, maplib.EncounterTypeRuins: return data.DeathMagic
        case maplib.EncounterTypeAncientTemple, maplib.EncounterTypeFallenTemple: return data.LifeMagic
        case maplib.EncounterTypePlaneTower:
            if quirkTowerBooksNature {
                return data.NatureMagic
            }
    }
    // a cave or monster lair: the original rolls 1 to 5 where it means 0 to 4, so sorcery, chaos,
    // life, death or arcane, never nature (kept)
    realms := []data.MagicType{data.SorceryMagic, data.ChaosMagic, data.LifeMagic, data.DeathMagic, data.ArcaneMagic}
    return realms[rand.N(len(realms))]
}

// the retorts a treasure gives: of one pick, and of two (EZ_SpecialTreasure); never Myrran
var classicRetortsOne = []data.Retort{data.RetortAlchemy, data.RetortArchmage, data.RetortArtificer, data.RetortConjurer,
    data.RetortSageMaster, data.RetortCharismatic, data.RetortChaosMastery, data.RetortNatureMastery, data.RetortSorceryMastery,
    data.RetortManaFocusing, data.RetortNodeMastery, data.RetortRunemaster}
var classicRetortsTwo = []data.Retort{data.RetortWarlord, data.RetortChanneler, data.RetortDivinePower, data.RetortFamous, data.RetortInfernalPower}

// Lair_Generate_Treasure: what the hoard gives the wizard that takes the place
func makeClassicTreasure(cache *lbx.LbxCache, encounterType maplib.EncounterType, budget int, point data.PlanePoint, wizard setup.WizardCustom, knownSpells spellbook.Spells, allSpells spellbook.Spells, heroes []*herolib.Hero) Treasure {
    hoard := classicRollHoard(budget, encounterType == maplib.EncounterTypePlaneTower)
    var items []TreasureItem

    // an item: Lair_Generate_Treasure calls Make_Item with its value where the "power" goes and no
    // worth, so every item of a hoard is a random one of 800 to 1700 (artifact.MakeClassicRandomItem)
    giveItem := func(power int, worth int) bool {
        made, ok := artifact.MakeClassicRandomItem(cache, power, worth)
        if !ok {
            return false
        }
        items = append(items, &TreasureMagicalItem{Artifact: &made})
        return true
    }

    if hoard.Gold > 0 {
        items = append(items, &TreasureGold{Amount: hoard.Gold})
    }
    mana := hoard.Mana
    for _, value := range hoard.Items {
        giveItem(value, 0)
    }
    if hoard.Prisoner && len(heroes) > 0 {
        items = append(items, &TreasurePrisonerHero{Hero: heroes[rand.N(len(heroes))]})
    }

    switch {
        case hoard.SpellOrSpecial >= 1 && hoard.SpellOrSpecial <= 4:
            rarities := []spellbook.SpellRarity{spellbook.SpellRarityCommon, spellbook.SpellRarityUncommon, spellbook.SpellRarityRare, spellbook.SpellRarityVeryRare}
            if spell, ok := classicTreasureSpell(rarities[hoard.SpellOrSpecial - 1], wizard, knownSpells, allSpells); ok {
                items = append(items, &TreasureSpell{Spell: spell})
            }
        case hoard.SpellOrSpecial > 4:
            picks := hoard.SpellOrSpecial - 4
            given := false
            books := make(map[data.MagicType]int)
            for picks > 0 {
                if classicTreasureRoll(4) <= 3 {
                    realm := classicBookRealm(encounterType)
                    // an arcane book is not refused; the original adds it past the end of its list of
                    // books, onto the retort of Alchemy (kept: quirkArcaneBookAlchemy)
                    if realm == data.ArcaneMagic {
                        picks -= 1
                        given = true
                        if quirkArcaneBookAlchemy && !wizard.RetortEnabled(data.RetortAlchemy) {
                            items = append(items, &TreasureRetort{Retort: data.RetortAlchemy})
                        }
                        continue
                    }
                    // a wizard of life gets no book of death, and the other way round
                    if (realm == data.DeathMagic && wizard.MagicLevel(data.LifeMagic) > 0) || (realm == data.LifeMagic && wizard.MagicLevel(data.DeathMagic) > 0) {
                        picks -= 1
                        continue
                    }
                    books[realm] += 1
                    picks -= 1
                    given = true
                    continue
                }
                choices := classicRetortsOne
                if picks >= 2 {
                    choices = append(slices.Clone(classicRetortsOne), classicRetortsTwo...)
                }
                choices = slices.DeleteFunc(slices.Clone(choices), func(retort data.Retort) bool {
                    return wizard.RetortEnabled(retort)
                })
                if len(choices) == 0 {
                    picks -= 1
                    continue
                }
                retort := choices[rand.N(len(choices))]
                items = append(items, &TreasureRetort{Retort: retort})
                given = true
                if slices.Contains(classicRetortsTwo, retort) {
                    picks -= 2
                } else {
                    picks -= 1
                }
            }
            for _, realm := range []data.MagicType{data.NatureMagic, data.SorceryMagic, data.ChaosMagic, data.LifeMagic, data.DeathMagic} {
                if books[realm] > 0 {
                    items = append(items, &TreasureSpellbook{Magic: realm, Count: books[realm]})
                }
            }
            if !given {
                if len(hoard.Items) >= classicItemsMost || !giveItem(0, classicFallbackItem) {
                    mana += classicFallbackMana
                }
            }
    }
    if mana > 0 {
        items = append(items, &TreasureMana{Amount: mana})
    }

    return Treasure{Treasures: items, Point: point}
}

// a spell of the rarity the wizard does not know, of its realms or arcane (as the fork's treasure
// picks one)
func classicTreasureSpell(rarity spellbook.SpellRarity, wizard setup.WizardCustom, knownSpells spellbook.Spells, allSpells spellbook.Spells) (spellbook.Spell, bool) {
    spells := allSpells.GetSpellsByRarity(rarity)
    spells.RemoveSpells(knownSpells)
    var possible spellbook.Spells
    for _, book := range wizard.Books {
        possible.AddAllSpells(spells.GetSpellsByMagic(book.Magic))
    }
    possible.AddAllSpells(spells.GetSpellsByMagic(data.ArcaneMagic))
    if len(possible.Spells) == 0 {
        return spellbook.Spell{}, false
    }
    return possible.Spells[rand.N(len(possible.Spells))], true
}
