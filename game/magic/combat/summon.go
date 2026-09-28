package combat

// A unit that is summoned, matched to the original game.
//
// The original shows the circle of the summoning on the ground for 16 steps of 2 redraws. For the
// first 7 the unit is not there. Then it rises out of the ground, 3 pixels a step from 18 below
// where it stands, and gets more solid with every step. For the last 2 steps it stands as it is.
// (ReMoM: MoM/src/Spells133.c Battle_Unit_Summon_Animation. The code here is ours.)

import (
    "github.com/kazzmir/master-of-magic/lib/coroutine"

    "github.com/hajimehoshi/ebiten/v2"
)

const summonSteps = 16
const summonStepTicks = 2
// the unit rises from after this step until this step
const summonRiseFrom = 6
const summonRiseUntil = 14
// art pixels a step
const summonRise = 3
// how solid the unit gets with every step, of 100
const summonSolid = 14

func (combat *CombatScreen) showSummon(yield coroutine.YieldFunc, unit *ArmyUnit) {
    start := combat.Counter

    for {
        // so that the circle of the summoning shows
        combat.Model.UpdateProjectiles(combat.Counter)
        combat.Counter += 1

        ticks := int(float64(combat.Counter - start) * OriginalTicksPerSecond / float64(max(1, ebiten.TPS())))
        step := ticks / summonStepTicks

        switch {
            case step <= summonRiseFrom:
                // only the circle
                unit.SetHeight(-(summonRiseUntil - summonRiseFrom - 1) * summonRise)
                unit.SetFade(1)
            case step < summonRiseUntil:
                risen := step - summonRiseFrom
                unit.SetHeight(-(summonRiseUntil - summonRiseFrom - 1 - risen) * summonRise)
                unit.SetFade(1 - float32(risen * summonSolid) / 100)
            default:
                unit.SetHeight(0)
                unit.SetFade(0)
        }

        if step >= summonSteps {
            break
        }

        if yield() != nil {
            break
        }
    }

    unit.SetHeight(0)
    unit.SetFade(0)
}
