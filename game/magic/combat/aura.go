package combat

// The outline of the figures of a unit that has a spell on it: the original's, see the package
// aura. In a battle it is painted INTO the picture of the figure, as the original does it and as
// the pulse of the outline of the unit under the mouse is (animation.go), so shadows and the rim
// light see the figure with its outline.

import (
    "fmt"
    "image"
    "image/draw"

    "github.com/kazzmir/master-of-magic/game/magic/aura"
    "github.com/kazzmir/master-of-magic/game/magic/util"
)

// false: the glow of upstream
const OriginalAura = true

// the realm of the outline of a unit, aura.None for no outline
func unitAura(unit *ArmyUnit) aura.Realm {
    if unit.IsInvisible() {
        return aura.None
    }

    return aura.Of(unit.GetEnchantments())
}

func (combat *CombatScreen) auraStage() int {
    return aura.StageAt(combat.originalTick())
}

// paints the outline into the picture of a figure, after everything else was done to it
func withAura(base util.ImageTransformFunc, realm aura.Realm, stage int) util.ImageTransformFunc {
    return func(original *image.Paletted) image.Image {
        // where the outline goes is a matter of the picture as the game has it
        pixels := aura.Pixels(original)
        bounds := original.Bounds()

        changed := base(original)
        out := image.NewNRGBA(bounds)
        draw.Draw(out, bounds, changed, bounds.Min, draw.Src)
        aura.Paint(out, pixels, realm, stage)

        return out
    }
}

func auraKey(key string, realm aura.Realm, stage int) string {
    return fmt.Sprintf("%v-aura%v-%v", key, int(realm), stage)
}
