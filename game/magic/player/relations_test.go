package player

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/setup"
    "github.com/kazzmir/master-of-magic/game/magic/data"
)

// the original's Init_Diplomatic_Relations: 10 life against 10 chaos: 20 - 20; 10 life against
// 5 death: (10 + 5) * -5; 10 chaos against 5 death: -15 - 20
func TestStartingRelation(test *testing.T) {
    ariel := setup.WizardCustom{
        Books: []data.WizardBook{
            data.WizardBook{
                Magic: data.LifeMagic,
                Count: 10,
            },
        },
    }

    tauron := setup.WizardCustom{
        Books: []data.WizardBook{
            data.WizardBook{
                Magic: data.ChaosMagic,
                Count: 10,
            },
        },
    }

    tlaloc := setup.WizardCustom{
        Books: []data.WizardBook{
            data.WizardBook{
                Magic: data.NatureMagic,
                Count: 4,
            },
            data.WizardBook{
                Magic: data.DeathMagic,
                Count: 5,
            },
        },
    }

    relation := computeStartingRelation(ariel, tauron)
    if relation != 0 {
        test.Errorf("expected 0, got %d", relation)
    }

    relation = computeStartingRelation(ariel, tlaloc)
    if relation != -75 {
        test.Errorf("expected -75, got %d", relation)
    }

    relation = computeStartingRelation(tauron, tlaloc)
    if relation != -35 {
        test.Errorf("expected -35, got %d", relation)
    }
}
