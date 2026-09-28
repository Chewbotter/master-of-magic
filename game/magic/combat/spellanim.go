package combat

// How spells look on the battlefield, matched to the original game.
//
// The original shows a spell as a picture that goes through its frames at a place of its screen,
// one step after the other, while a message at the top says who has cast what. Here a spell is a
// projectile with steps (Projectile.Steps): each step is a frame of the picture, the place of its
// top left corner on the original's screen, and how many of the original's redraws it shows. What
// the spell does happens after the last step.
//
// - bolts (Fire Bolt, Ice Bolt, Fireball, Doom Bolt) fly in from a fixed place next to their target
// - everything else plays its frames once on its target
// - Lightning Bolt shows two flashes of a bolt that comes from the top of the screen
// - Cracks Call and the circle of a summoning lie on the ground, under the units
// - spells that hit all units of a side start on each unit a little later or earlier
//
// (ReMoM: MoM/src/Spells131.c Combat_Spell_Animation, Combat_Battlefield_Instant; Spells133.c
// Combat_Spell_Animation_Bolt, Combat_Spell_Animation_Default, Animate_Lightning_Bolt,
// Animate_Cracks_Call, Battle_Unit_Summon_Animation; SBookScr.c Spell_Animation_Load_Graphics;
// Combat.c Combat_Cast_Spell_Message and the cell effect of the map draw. The code here is ours.)
//
// NOT FROM THE ORIGINAL, judgment calls:
// - the reference names SPECFX 11 and 12 for Ice Bolt and Petrify. those are pictures of spells of
//   the world map. CMBTFX 11 and 12 are named Ice Bolt and Petrify in the archive, and are used
// - the reference holds a frame of an effect on the ground for 7 redraws and calls that a hack of
//   its own. they show 2 redraws here, like every other effect
// - spells on all units: the reference places all of them like small pictures, which it marks as a
//   bug of the original. large pictures are placed like large pictures here

import (
    "fmt"
    "image/color"
    "math/rand/v2"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/game/magic/spellbook"
    "github.com/kazzmir/master-of-magic/lib/font"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/vector"
)

// one step of a spell: a frame of its picture at a place, for a time
type ProjectileStep struct {
    // the top left corner of the picture on the original's screen
    X int
    Y int
    // the frame of the picture, below 0 for nothing
    Frame int
    // redraws of the original the step shows
    Ticks int
}

const spellFrameNone = -1

// redraws a frame of an effect shows
const spellFrameTicks = 2

// pictures this wide and wider are the large effects of the original
const spellLargeWidth = 60

// from the place of a unit to the top left corner of an effect on it
const spellSmallX = -14
const spellSmallY = -25
const spellLargeX = -28
const spellLargeY = -30
// the same for a spell that hits all units of a side
const spellAllSmallY = -21
// a unit waits up to this many frames before its effect of such a spell starts
const spellAllDelayMax = 4

// effects on the ground: from the top corner of the tile to the top left corner of the picture
const spellGroundX = -14
const spellGroundY = -18

// the place of a unit on the original's screen: the middle of its tile, as the original counts it
const spellPlaceSubcell = 4

// Lightning Bolt: redraws without the bolt before and after it, flashes of the bolt, its frames to
// pick from, and how far above the place of the unit the top of its picture is
const lightningWaitTicks = 5
const lightningFlashes = 2
const lightningFrames = 4
const lightningAbove = 199

type boltPath struct {
    // from the place of the target to the top left corner of the picture when it hits
    HitX int
    HitY int
    // from there to where it starts
    StartX int
    StartY int
    // how far it goes each redraw
    StepX int
    StepY int
    // redraws in all
    Steps int
}

// after this many redraws a bolt is at its target, and each frame shows two redraws
const boltFlightSteps = 11
// the frames of a bolt in flight, and the frame of its hit
const boltFlightFrames = 3
const boltHitFrame = 3
// Doom Bolt stays on each frame for this many redraws
const doomBoltFrameSteps = 4

type BoltKind int
const (
    BoltFire BoltKind = iota
    BoltFireball
    BoltIce
    BoltDoom
)

var boltPaths = map[BoltKind]boltPath{
    BoltFire: {HitX: -16, HitY: -20, StartX: 110, StartY: -66, StepX: -10, StepY: 6, Steps: 11},
    BoltFireball: {HitX: -14, HitY: -21, StartX: 110, StartY: -66, StepX: -10, StepY: 6, Steps: 16},
    BoltIce: {HitX: -17, HitY: -20, StartX: 110, StartY: -110, StepX: -10, StepY: 10, Steps: 11},
    BoltDoom: {HitX: -14, HitY: -25, StartX: 0, StartY: -72, StepX: 0, StepY: 6, Steps: 13},
}

// the place of a tile on the original's screen, where the original puts the effects of a unit
func spellPlace(x int, y int) (int, int) {
    cgx, cgy := TileToCell(x, y)
    return cellScreen(cgx, cgy, spellPlaceSubcell, spellPlaceSubcell)
}

func newSpellProjectile(target *ArmyUnit, pictures []*ebiten.Image, steps []ProjectileStep, effect ProjectileEffect) *Projectile {
    return &Projectile{
        Target: target,
        Effect: effect,
        Scripted: true,
        Pictures: pictures,
        Steps: steps,
    }
}

// the frames of a picture, once, on a unit. wait is the number of frames before it starts
func effectSteps(pictures []*ebiten.Image, x int, y int, wait int, all bool) []ProjectileStep {
    if len(pictures) == 0 {
        return nil
    }

    placeX, placeY := spellPlace(x, y)
    if pictures[0].Bounds().Dx() >= spellLargeWidth {
        placeX += spellLargeX
        placeY += spellLargeY
    } else {
        placeX += spellSmallX
        if all {
            placeY += spellAllSmallY
        } else {
            placeY += spellSmallY
        }
    }

    var steps []ProjectileStep
    if wait > 0 {
        steps = append(steps, ProjectileStep{Frame: spellFrameNone, Ticks: wait * spellFrameTicks})
    }
    for frame := range pictures {
        steps = append(steps, ProjectileStep{X: placeX, Y: placeY, Frame: frame, Ticks: spellFrameTicks})
    }

    return steps
}

// an effect on a unit
func (combat *CombatScreen) createEffect(target *ArmyUnit, pictures []*ebiten.Image, effect ProjectileEffect) *Projectile {
    return newSpellProjectile(target, pictures, effectSteps(pictures, target.X, target.Y, 0, false), effect)
}

// the effect of a spell that hits all units of a side, on one of them
func (combat *CombatScreen) createEffectOfAll(target *ArmyUnit, pictures []*ebiten.Image, effect ProjectileEffect) *Projectile {
    wait := rand.N(spellAllDelayMax + 1)
    return newSpellProjectile(target, pictures, effectSteps(pictures, target.X, target.Y, wait, true), effect)
}

// an effect that lies on the ground of a tile, under the units
func (combat *CombatScreen) createGroundEffect(target *ArmyUnit, pictures []*ebiten.Image, effect ProjectileEffect) *Projectile {
    cgx, cgy := TileToCell(target.X, target.Y)
    x, y := cellScreen(cgx, cgy, 0, 0)

    var steps []ProjectileStep
    for frame := range pictures {
        steps = append(steps, ProjectileStep{X: x + spellGroundX, Y: y + spellGroundY, Frame: frame, Ticks: spellFrameTicks})
    }

    projectile := newSpellProjectile(target, pictures, steps, effect)
    projectile.Ground = true
    return projectile
}

func (combat *CombatScreen) createBolt(kind BoltKind, target *ArmyUnit, pictures []*ebiten.Image, effect ProjectileEffect) *Projectile {
    path := boltPaths[kind]
    placeX, placeY := spellPlace(target.X, target.Y)
    hitX := placeX + path.HitX
    hitY := placeY + path.HitY
    x := hitX + path.StartX
    y := hitY + path.StartY

    var steps []ProjectileStep
    for step := range path.Steps {
        frame := 0
        switch kind {
            case BoltFireball: frame = step
            case BoltDoom: frame = step / doomBoltFrameSteps
            default:
                frame = step % boltFlightFrames
                if step == path.Steps - 1 {
                    frame = boltHitFrame
                }
        }
        frame = min(frame, len(pictures) - 1)

        if step >= boltFlightSteps {
            // at its target
            steps = append(steps, ProjectileStep{X: hitX, Y: hitY, Frame: frame, Ticks: spellFrameTicks})
        } else {
            steps = append(steps, ProjectileStep{X: x, Y: y, Frame: frame, Ticks: 1})
            x += path.StepX
            y += path.StepY
        }
    }

    if len(pictures) == 0 {
        steps = nil
    }

    return newSpellProjectile(target, pictures, steps, effect)
}

func (combat *CombatScreen) createLightning(target *ArmyUnit, pictures []*ebiten.Image, effect ProjectileEffect) *Projectile {
    placeX, placeY := spellPlace(target.X, target.Y)

    steps := []ProjectileStep{{Frame: spellFrameNone, Ticks: lightningWaitTicks}}
    for range lightningFlashes {
        frame := rand.N(lightningFrames)
        steps = append(steps, ProjectileStep{X: placeX, Y: placeY - lightningAbove, Frame: min(frame, len(pictures) - 1), Ticks: 1})
    }
    steps = append(steps, ProjectileStep{Frame: spellFrameNone, Ticks: lightningWaitTicks})

    if len(pictures) == 0 {
        steps = nil
    }

    return newSpellProjectile(target, pictures, steps, effect)
}

// the step a spell is at, or the number of its steps when it is over
func (projectile *Projectile) stepAt(counter uint64) int {
    if !projectile.Started {
        return 0
    }

    ticks := int(float64(counter - projectile.Start) * OriginalTicksPerSecond / float64(max(1, ebiten.TPS())))
    for index, step := range projectile.Steps {
        if ticks < step.Ticks {
            return index
        }
        ticks -= step.Ticks
    }

    return len(projectile.Steps)
}

// one tick of a spell with steps. false when it is over
func (projectile *Projectile) updateSteps(counter uint64) bool {
    if !projectile.Started {
        projectile.Started = true
        projectile.Start = counter
    }

    projectile.Step = projectile.stepAt(counter)
    return projectile.Step < len(projectile.Steps)
}

func (combat *CombatScreen) drawSpell(screen *ebiten.Image, projectile *Projectile) {
    if projectile.Step < 0 || projectile.Step >= len(projectile.Steps) {
        return
    }

    step := projectile.Steps[projectile.Step]
    if step.Frame < 0 || step.Frame >= len(projectile.Pictures) {
        return
    }

    combat.drawOnField(screen, projectile.Pictures[step.Frame], step.X, step.Y)
}

// the spells that lie on the ground. drawn after the ground and before anything that stands on it
func (combat *CombatScreen) drawGroundSpells(screen *ebiten.Image) {
    for _, projectile := range combat.Model.Projectiles {
        if projectile.Ground && len(projectile.Steps) > 0 {
            combat.drawSpell(screen, projectile)
        }
    }
}

// ---- the message at the top of the screen that says who has cast what

// palette indexes and places, in art pixels
const castMessageTextColor = 177
const castMessageEdgeColor = 182
const castMessageShadowColor = 2
const castMessageLightColor = 230
const castMessageDarkColor = 237
const castMessageFontIndex = 1
const castMessageCenterX = 160
const castMessageTop = 5
const castMessageBottom = 15
const castMessageTextY = 8
const castMessageMargin = 5
// how dark the box is, of 255. the original darkens what is behind it
const castMessageShade = 140
// the message stays at least this long, in seconds
const castMessageTime = 1.0

// the words of the message
const castMessageHas = " has cast "
const castMessageHave = " have cast "
const castMessageThe = "The "

// what the message says. unitCaster is the unit that cast, or nil for the wizard
func castMessageText(army *Army, unitCaster *ArmyUnit, spell spellbook.Spell) string {
    if unitCaster == nil {
        return army.Player.GetWizard().Name + castMessageHas + spell.Name
    }

    if unitCaster.Unit.IsHero() {
        return unitCaster.Unit.GetName() + castMessageHas + spell.Name
    }

    if unitCaster.Figures() > 1 {
        return castMessageThe + unitCaster.Unit.GetName() + castMessageHave + spell.Name
    }

    return castMessageThe + unitCaster.Unit.GetName() + castMessageHas + spell.Name
}

// shows the message from now on, see castMessageShown
func (combat *CombatScreen) CastMessage(text string) {
    combat.castMessage = text
    combat.castMessageUntil = combat.Counter + uint64(castMessageTime * float64(max(1, ebiten.TPS())))
}

// the message shows while the spell it is about plays, and for a moment at least
func (combat *CombatScreen) castMessageShown() bool {
    if combat.castMessage == "" {
        return false
    }

    if combat.Counter < combat.castMessageUntil {
        return true
    }

    for _, projectile := range combat.Model.Projectiles {
        if len(projectile.Steps) > 0 {
            return true
        }
    }

    combat.castMessage = ""
    return false
}

func (combat *CombatScreen) castMessageFont() *font.StyledFont {
    if combat.castMessageStyle != nil {
        return combat.castMessageStyle
    }

    hud := combat.Fonts.Hud
    if hud == nil || len(hud.LbxFonts) <= castMessageFontIndex {
        return nil
    }

    // pixel value 0 is the soft edge of the letters, the rest are the letters
    colors := []color.Color{hud.Palette[castMessageEdgeColor]}
    for range 15 {
        colors = append(colors, hud.Palette[castMessageTextColor])
    }

    combat.castMessageStyle = font.MakeStyledFont(hud.LbxFonts[castMessageFontIndex], colors, font.ShadowDown, hud.Palette[castMessageShadowColor])
    return combat.castMessageStyle
}

func (combat *CombatScreen) drawCastMessage(screen *ebiten.Image) {
    if !combat.castMessageShown() {
        return
    }

    style := combat.castMessageFont()
    if style == nil {
        return
    }

    hud := combat.Fonts.Hud
    text := combat.castMessage
    width := style.Width(text)

    x1 := castMessageCenterX - width / 2 - castMessageMargin
    x2 := castMessageCenterX + castMessageMargin + width / 2

    fill := func(x int, y int, wide int, high int, use color.Color) {
        vector.FillRect(screen, float32(scale.Scale(x)), float32(scale.Scale(y)), float32(scale.Scale(wide)), float32(scale.Scale(high)), use, false)
    }

    // lines run from their first pixel to their last
    fill(x1, castMessageTop, x2 - x1 + 1, castMessageBottom - castMessageTop + 1, color.RGBA{A: castMessageShade})
    fill(x1, castMessageTop, 1, castMessageBottom - castMessageTop + 1, hud.Palette[castMessageLightColor])
    fill(x1, castMessageTop, x2 - x1, 1, hud.Palette[castMessageLightColor])
    fill(x2, castMessageTop, 1, castMessageBottom - castMessageTop + 1, hud.Palette[castMessageDarkColor])
    fill(x1 + 1, castMessageBottom, x2 - x1, 1, hud.Palette[castMessageDarkColor])

    style.Print(screen, castMessageCenterX, castMessageTextY, font.FontOptions{Scale: scale.ScaleAmount, Justify: font.FontJustifyCenter}, text)
}

// ---- the color the screen takes while a spell on the whole battlefield is cast

// how far the screen goes to the color, of 255: 40 of 100 in the original
const castTintMax = 102

// the color by the realm of the spell
func castTintColor(magic data.MagicType) (uint8, uint8, uint8) {
    switch magic {
        case data.NatureMagic: return 0, 255, 0
        case data.SorceryMagic: return 0, 0, 255
        case data.ChaosMagic: return 255, 0, 0
        case data.DeathMagic: return 0, 0, 0
    }
    // life and arcane
    return 255, 255, 255
}

func describeStep(step ProjectileStep) string {
    return fmt.Sprintf("%v,%v frame %v for %v", step.X, step.Y, step.Frame, step.Ticks)
}
