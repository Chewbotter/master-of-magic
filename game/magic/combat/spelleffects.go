package combat

// What goes with the pictures of a spell in a battle. Not in the original (user, 2026-09-28).
//
// - the picture of the spell lights itself up: it is drawn a second time so that its light is
//   added to itself
// - particles come off a bolt in flight
// - where a spell hits: particles are thrown up, the unit that is hit shows in one color, the
//   battle stands still for a moment, the view shakes, light runs over the ground
//
// How much of each a spell has are its values, see spellvalues.go. All of it is made of art
// pixels of the field.
//
// SpellEffectsKey turns all of it off and on in a battle, to compare.

import (
    "image"
    rand "github.com/kazzmir/master-of-magic/lib/chance"
    "image/color"
    "log"
    "math"
    "strings"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/mod"
    "github.com/kazzmir/master-of-magic/game/magic/scale"
    "github.com/kazzmir/master-of-magic/game/magic/unitview"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/inpututil"
)

var SpellEffects = true
var SpellEffectsKey = ebiten.KeyF7

// the steps the light on the ground has
const pulseBands = 3
// the ring of light is this many tiles wide
const pulseWidth = 1.2

// what has hit a unit
type deathCause struct {
    Tick uint64
    HasColor bool
    Color color.RGBA
    // where it hit, on the original's screen, and how far it throws
    X float64
    Y float64
    Throw float64
}

// what a unit was hit by a moment ago, if it was a spell
func (combat *CombatScreen) deathCauseOf(unit *ArmyUnit) (deathCause, bool) {
    effects := &combat.effects
    cause, ok := effects.Causes[unit]
    if ok && effects.Tick - cause.Tick <= effectTicks(deathCauseTime) {
        return cause, true
    }
    return deathCause{}, false
}

// what a unit was hit by counts for the figures it loses for this long, in seconds
const deathCauseTime = 4.0

// the color the figures a unit loses now take on as they lie, if any
func (combat *CombatScreen) corpseColor(unit *ArmyUnit) (color.RGBA, bool) {
    effects := &combat.effects

    cause, ok := effects.Causes[unit]
    if ok && effects.Tick - cause.Tick <= effectTicks(deathCauseTime) {
        return cause.Color, cause.HasColor
    }

    // killed in a fight
    file, _ := mod.Effects()
    text, given := file[combatPart]["corpse-color"]
    if given {
        if strings.ToLower(strings.TrimSpace(text)) == noColor {
            return color.RGBA{}, false
        }
        value, ok := parseColor(text)
        if ok {
            return value, true
        }
    }
    return combatCorpseColor, combatCorpseTinted
}

// a unit that shows in one color
type unitFlash struct {
    Start uint64
    Ticks uint64
    Strength float64
    Color [3]float32
}

// light that runs over the ground
type groundPulse struct {
    X int
    Y int
    Start uint64
    Ticks uint64
    Radius float64
    Strength float64
    Color [3]float32
}

// what the effects of a battle keep
type spellEffects struct {
    Particles particleSystem
    // ticks of the effects. they go on while the battle stands still
    Tick uint64
    // the tick of the battle the effects were last moved on at
    Counter uint64
    Moved bool

    // by the first picture of a spell: its name
    Names map[*ebiten.Image]string
    // the values of the spells, and the state of the file they are from
    Values map[string]SpellValues
    Version int

    Flashes map[*ArmyUnit]unitFlash
    // the rims of the pictures of figures by the side the light is on, see rimlight.go
    Rims map[rimKey]*ebiten.Image
    // the pictures of figures in white, by their picture
    Masks map[*ebiten.Image]*ebiten.Image
    Pulses []groundPulse

    // ticks the battle still stands still
    HitStop int
    // the view shakes: since when, for how long, how far, and where it is now in art pixels
    ShakeStart uint64
    ShakeTicks uint64
    ShakeSize float64
    ShakeX int
    ShakeY int

    // what is left of the particles a bolt is to give off, by bolt
    Owed map[*Projectile]float64
    // the same for the sparks off the picture of a spell, and the pixels of a picture sparks come
    // off, by picture
    SparksOwed map[*Projectile]float64
    Points map[*ebiten.Image][]image.Point
    // the tile every unit that walks was last over, see splash.go
    Wading map[*ArmyUnit]image.Point

    Pixel *ebiten.Image

    // the light of spells, see spelllight.go
    Lighting spellLighting
    // what a unit was last hit by, for the color of its corpses. see figurefall.go
    Causes map[*ArmyUnit]deathCause

    // which pixels of the pictures of corpses are there, see corpsesplat.go
    Solid map[*ebiten.Image][]bool
    // the figures the units had at the start of the tick, see blood.go
    Figures map[*ArmyUnit]int
    // the stains of blood on the ground, see blood.go
    Stains bloodStains
    StainStrength float64

    // the marks on the ground and their pictures, see decals.go
    Decals []decal
    DecalPictures map[string][]*ebiten.Image
}

func effectTicks(seconds float64) uint64 {
    if seconds <= 0 {
        return 0
    }
    return uint64(max(1, int(math.Round(seconds * float64(max(1, ebiten.TPS()))))))
}

func colorParts(red uint8, green uint8, blue uint8) [3]float32 {
    return [3]float32{float32(red) / 255, float32(green) / 255, float32(blue) / 255}
}

func (combat *CombatScreen) updateSpellEffectsKey() {
    if inpututil.IsKeyJustPressed(SpellEffectsKey) {
        SpellEffects = !SpellEffects
        if SpellEffects {
            combat.CastMessage("Spell effects: on")
        } else {
            combat.CastMessage("Spell effects: off")
        }
    }
}

// the name of the pictures of a spell, empty if they are none of a spell
func (combat *CombatScreen) spellName(pictures []*ebiten.Image) string {
    if len(pictures) == 0 {
        return ""
    }

    effects := &combat.effects
    if effects.Names == nil && combat.Cache == nil {
        // no pictures of the game, as in a test
        return ""
    }
    if effects.Names == nil {
        effects.Names = make(map[*ebiten.Image]string)
        for _, spell := range mod.SpellPictures {
            images, err := combat.ImageCache.GetImages(spell.Archive, spell.Entry)
            if err == nil && len(images) > 0 {
                effects.Names[images[0]] = spell.Name
            }
        }
    }

    return effects.Names[pictures[0]]
}

func (combat *CombatScreen) valuesOf(name string) SpellValues {
    effects := &combat.effects

    file, version := mod.Effects()
    if effects.Values == nil || version != effects.Version {
        effects.Values = make(map[string]SpellValues)
        effects.Version = version
    }

    values, ok := effects.Values[name]
    if !ok {
        var notes []string
        values, notes = spellValues(name, file)
        for _, note := range notes {
            log.Print(note)
        }
        effects.Values[name] = values
    }

    return values
}

// gives a spell its name and says at which of its steps it hits
func (combat *CombatScreen) dress(projectile *Projectile, impactStep int) *Projectile {
    if projectile == nil {
        return nil
    }

    projectile.Name = combat.spellName(projectile.Pictures)
    projectile.ImpactStep = impactStep
    projectile.OnImpact = func() {
        combat.spellHits(projectile)
    }
    return projectile
}

// the first step of a spell that shows a picture
func firstShownStep(steps []ProjectileStep) int {
    for index, step := range steps {
        if step.Frame >= 0 {
            return index
        }
    }
    return 0
}

// a spell hits
func (combat *CombatScreen) spellHits(projectile *Projectile) {
    if projectile.Target == nil {
        return
    }

    // what it does to its target is done now, so the units answer to the hit and not to the end
    // of its pictures. after the rest of this function: the figures it kills are thrown by it
    if combat.valuesOf(projectile.Name).ResolveAtImpact && projectile.Effect != nil && !projectile.EffectDone {
        projectile.EffectDone = true
        defer projectile.Effect(projectile.Target)
    }

    if !SpellEffects {
        return
    }

    values := combat.valuesOf(projectile.Name)
    effects := &combat.effects
    target := projectile.Target

    placeX, placeY := spellPlace(target.X, target.Y)

    if values.BurstCount > 0 {
        effects.Particles.emitBurst(float64(placeX), float64(placeY), values.ImpactHeight, values.BurstCount, values.BurstSpeed, values.BurstLift, values.BurstGravity, values.BurstLife, values.BurstColors)
    }
    // dirt from the ground
    if values.DebrisCount > 0 && len(values.DebrisColors) > 0 {
        effects.Particles.emitBurst(float64(placeX), float64(placeY), 0, values.DebrisCount, values.DebrisSpeed, values.DebrisLift, values.DebrisGravity, values.DebrisLife, values.DebrisColors)
    }

    // the figures it kills, see figurefall.go
    if effects.Causes == nil {
        effects.Causes = make(map[*ArmyUnit]deathCause)
    }
    effects.Causes[target] = deathCause{
        Tick: effects.Tick,
        HasColor: values.HasCorpseColor,
        Color: values.CorpseColor,
        X: float64(placeX),
        Y: float64(placeY),
        Throw: values.Throw,
    }

    // its light stays for a while, see spelllight.go
    combat.addGlow(projectile, values)

    // the mark it leaves on the ground, see decals.go
    if values.Decal != DecalNone {
        combat.addDecal(projectile.Name, values, target.X, target.Y)
    }

    if values.FlashTime > 0 && values.FlashStrength > 0 {
        if effects.Flashes == nil {
            effects.Flashes = make(map[*ArmyUnit]unitFlash)
        }
        effects.Flashes[target] = unitFlash{
            Start: effects.Tick,
            Ticks: effectTicks(values.FlashTime),
            Strength: values.FlashStrength,
            Color: colorParts(values.FlashColor.R, values.FlashColor.G, values.FlashColor.B),
        }
    }

    if values.HitStop > 0 && !combat.together {
        effects.HitStop = max(effects.HitStop, int(effectTicks(values.HitStop)))
    }

    if values.Shake > 0 && values.ShakeTime > 0 {
        effects.ShakeStart = effects.Tick
        effects.ShakeTicks = effectTicks(values.ShakeTime)
        effects.ShakeSize = values.Shake
    }

    if values.PulseRadius > 0 && values.PulseTime > 0 && values.PulseStrength > 0 {
        effects.Pulses = append(effects.Pulses, groundPulse{
            X: target.X,
            Y: target.Y,
            Start: effects.Tick,
            Ticks: effectTicks(values.PulseTime),
            Radius: values.PulseRadius,
            Strength: values.PulseStrength,
            Color: colorParts(values.PulseColor.R, values.PulseColor.G, values.PulseColor.B),
        })
    }
}

// one tick of what does not stand still with the battle: the shaking, the color of a unit that
// was hit, the light on the ground
func (combat *CombatScreen) effectsTick() {
    effects := &combat.effects
    effects.Tick += 1

    effects.ShakeX = 0
    effects.ShakeY = 0
    if effects.ShakeTicks > 0 {
        gone := effects.Tick - effects.ShakeStart
        if gone >= effects.ShakeTicks {
            effects.ShakeTicks = 0
        } else {
            // less and less
            size := effects.ShakeSize * (1 - float64(gone) / float64(effects.ShakeTicks))
            effects.ShakeX = int(math.Round(randomPart(-size, size)))
            effects.ShakeY = int(math.Round(randomPart(-size, size)))
        }
    }

    for unit, flash := range effects.Flashes {
        if effects.Tick - flash.Start >= flash.Ticks {
            delete(effects.Flashes, unit)
        }
    }

    combat.lightTick()

    kept := effects.Pulses[:0]
    for _, pulse := range effects.Pulses {
        if effects.Tick - pulse.Start < pulse.Ticks {
            kept = append(kept, pulse)
        }
    }
    effects.Pulses = kept
}

// one tick of the effects with the clock of the battle. called wherever the clock moves, counts
// once per tick of the battle
func (combat *CombatScreen) updateSpellEffects() {
    effects := &combat.effects
    if effects.Moved && effects.Counter == combat.Counter {
        return
    }
    effects.Moved = true
    effects.Counter = combat.Counter

    combat.effectsTick()
    // how many figures the units have before anything is hurt, see blood.go
    combat.noteFigures()
    // nothing is owed for standing still when units act together
    effects.HitStop = min(effects.HitStop, hitStopMost())
    if combat.together {
        effects.HitStop = 0
    }

    seconds := 1 / float64(max(1, ebiten.TPS()))

    // what comes off the bolts in flight
    if SpellEffects {
        for _, projectile := range combat.Model.Projectiles {
            if !projectile.Scripted || !projectile.Started || projectile.Impacted {
                continue
            }
            if projectile.Step < 0 || projectile.Step >= len(projectile.Steps) {
                continue
            }
            step := projectile.Steps[projectile.Step]
            if step.GlideX == 0 && step.GlideY == 0 {
                continue
            }

            values := combat.valuesOf(projectile.Name)
            if values.TrailRate <= 0 || len(projectile.Pictures) == 0 {
                continue
            }

            x, y, frame, ok := projectile.placeAt(combat.Counter)
            if !ok {
                continue
            }

            if effects.Owed == nil {
                effects.Owed = make(map[*Projectile]float64)
            }
            owed := effects.Owed[projectile] + values.TrailRate * seconds
            count := int(owed)
            effects.Owed[projectile] = owed - float64(count)

            bounds := projectile.Pictures[frame].Bounds()
            middleX := float64(x) + float64(bounds.Dx()) / 2
            middleY := float64(y) + float64(bounds.Dy()) / 2
            effects.Particles.emitTrail(middleX, middleY, trailSpread, count, values.TrailSpeed, values.TrailGravity, values.TrailLife, values.TrailColors)
        }
    }

    for projectile := range effects.Owed {
        if projectile.Impacted || projectile.Step >= len(projectile.Steps) {
            delete(effects.Owed, projectile)
        }
    }

    // sparks off the pixels of the picture of a spell while it shows
    if SpellEffects {
        for _, projectile := range combat.Model.Projectiles {
            if !projectile.Scripted || !projectile.Started {
                continue
            }
            values := combat.valuesOf(projectile.Name)
            if values.SparkRate <= 0 || len(values.SparkColors) == 0 {
                continue
            }
            x, y, frame, ok := projectile.placeAt(combat.Counter)
            if !ok {
                continue
            }
            points := combat.picturePoints(projectile.Pictures[frame])
            if len(points) == 0 {
                continue
            }

            if effects.SparksOwed == nil {
                effects.SparksOwed = make(map[*Projectile]float64)
            }
            owed := effects.SparksOwed[projectile] + values.SparkRate * seconds
            count := int(owed)
            effects.SparksOwed[projectile] = owed - float64(count)

            for range count {
                point := points[rand.N(len(points))]
                effects.Particles.emitTrail(float64(x + point.X) + 0.5, float64(y + point.Y) + 0.5, 0.5, 1, values.SparkSpeed, values.SparkGravity, values.SparkLife, values.SparkColors)
            }
        }
    }
    for projectile := range effects.SparksOwed {
        if projectile.Step >= len(projectile.Steps) {
            delete(effects.SparksOwed, projectile)
        }
    }

    // the water of a river under the units that walk into it, see splash.go
    combat.splashTick()

    effects.Particles.step(seconds)

    // drops of blood that have come down, see blood.go
    for _, each := range effects.Particles.Landed {
        combat.stain(each.X, each.Y, each.Color)
        // on a corpse that lies there it stays on top of it, see corpsesplat.go
        combat.splatCorpse(each.X, each.Y, each.Color)
    }
    effects.Particles.Landed = effects.Particles.Landed[:0]
}

// the pixels of a picture that are not see-through, from its top left corner. read once
func (combat *CombatScreen) picturePoints(picture *ebiten.Image) []image.Point {
    effects := &combat.effects
    if points, ok := effects.Points[picture]; ok {
        return points
    }

    bounds := picture.Bounds()
    pixels := make([]byte, 4 * bounds.Dx() * bounds.Dy())
    picture.ReadPixels(pixels)

    var points []image.Point
    for y := range bounds.Dy() {
        for x := range bounds.Dx() {
            if pixels[(y * bounds.Dx() + x) * 4 + 3] >= 0x80 {
                points = append(points, image.Pt(x, y))
            }
        }
    }

    if effects.Points == nil {
        effects.Points = make(map[*ebiten.Image][]image.Point)
    }
    effects.Points[picture] = points
    return points
}

// art pixels around the middle of a bolt its particles start in
const trailSpread = 3.0
// the battle never stands still for longer than this, in seconds, whatever the values say
const hitStopLongest = 0.5

func hitStopMost() int {
    return int(effectTicks(hitStopLongest))
}

// the battle stands still for as long as what has hit asks for. the view shakes on
func (combat *CombatScreen) holdHit(yield func() error) {
    effects := &combat.effects
    effects.HitStop = min(effects.HitStop, hitStopMost())

    for effects.HitStop > 0 {
        effects.HitStop -= 1
        combat.effectsTick()
        combat.syncCamera()
        if yield() != nil {
            effects.HitStop = 0
            return
        }
    }
}

// how far the view is shaken now, in the measure of the matrix of the camera
func (combat *CombatScreen) shakeShift() (float64, float64) {
    if !SpellEffects {
        return 0, 0
    }
    scaleNow := combat.Camera.Scale()
    return float64(combat.effects.ShakeX) * scaleNow, float64(combat.effects.ShakeY) * scaleNow
}

// ---- drawing

// adds the light of a picture to what is on the field. x and y are on the original's screen
func (combat *CombatScreen) addLightOnField(screen *ebiten.Image, picture *ebiten.Image, x int, y int, strength float32) {
    matrix := originalScreenMatrix(combat.GetCameraMatrix())

    var options ebiten.DrawImageOptions
    options.GeoM.Translate(float64(x), float64(y))
    options.GeoM.Concat(matrix)
    options.ColorScale.ScaleAlpha(strength)
    options.Blend = ebiten.BlendLighter
    scale.DrawScaled(screen, picture, &options)
}

// a frame of a spell with its light
func (combat *CombatScreen) drawSpellPicture(screen *ebiten.Image, name string, picture *ebiten.Image, x int, y int) {
    combat.drawOnField(screen, picture, x, y)

    if SpellEffects {
        emissive := combat.valuesOf(name).Emissive
        if emissive > 0 {
            combat.addLightOnField(screen, picture, x, y, float32(min(1, emissive)))
        }
    }
}

func (combat *CombatScreen) effectPixel() *ebiten.Image {
    if combat.effects.Pixel == nil {
        combat.effects.Pixel = ebiten.NewImage(1, 1)
        combat.effects.Pixel.Fill(white)
    }
    return combat.effects.Pixel
}

// the particles, on top of what stands on the field
func (combat *CombatScreen) drawParticles(screen *ebiten.Image) {
    particles := combat.effects.Particles.Particles
    if len(particles) == 0 {
        return
    }

    pixel := combat.effectPixel()
    matrix := originalScreenMatrix(combat.GetCameraMatrix())

    for index := range particles {
        each := &particles[index]
        x, y := each.place()
        tint := each.color()

        var options ebiten.DrawImageOptions
        options.GeoM.Translate(float64(x), float64(y))
        options.GeoM.Concat(matrix)
        options.ColorScale.Scale(float32(tint.R) / 255, float32(tint.G) / 255, float32(tint.B) / 255, 1)
        scale.DrawScaled(screen, pixel, &options)
    }
}

// how much of the light of a pulse a tile has now, in steps. 0 for none
func pulseBand(pulse groundPulse, tick uint64, x int, y int) int {
    if pulse.Ticks == 0 || tick < pulse.Start || tick - pulse.Start >= pulse.Ticks {
        return 0
    }

    gone := float64(tick - pulse.Start) / float64(pulse.Ticks)
    // the ring runs from the middle to the rim and gets weaker on its way
    front := gone * (pulse.Radius + pulseWidth)
    distance := math.Hypot(float64(x - pulse.X), float64(y - pulse.Y))
    if distance > pulse.Radius + 0.5 {
        return 0
    }

    near := 1 - math.Abs(distance - front) / pulseWidth
    if near <= 0 {
        return 0
    }

    return int(math.Ceil(near * (1 - gone) * pulseBands))
}

// the light that runs over the ground. drawn after the ground and before what stands on it
func (combat *CombatScreen) drawGroundPulses(screen *ebiten.Image) {
    effects := &combat.effects
    if !SpellEffects || len(effects.Pulses) == 0 {
        return
    }

    shapes := combat.moveAreaShapes()
    if shapes == nil {
        return
    }

    matrix := combat.GetCameraMatrix()

    for _, pulse := range effects.Pulses {
        reach := int(math.Ceil(pulse.Radius))
        for y := pulse.Y - reach; y <= pulse.Y + reach; y++ {
            for x := pulse.X - reach; x <= pulse.X + reach; x++ {
                band := pulseBand(pulse, effects.Tick, x, y)
                if band <= 0 {
                    continue
                }

                strength := float32(pulse.Strength * float64(band) / pulseBands)
                tx, ty := matrix.Apply(float64(x), float64(y))

                var options ebiten.DrawImageOptions
                options.GeoM.Translate(TerrainOffsetX, TerrainOffsetY)
                options.GeoM.Scale(combat.CameraScale, combat.CameraScale)
                options.GeoM.Translate(tx, ty)
                options.ColorScale.Scale(pulse.Color[0], pulse.Color[1], pulse.Color[2], 1)
                options.ColorScale.ScaleAlpha(strength)
                options.Blend = ebiten.BlendLighter
                scale.DrawScaled(screen, shapes.Fill, &options)
            }
        }
    }
}

// the picture of a figure in white
func (combat *CombatScreen) figureMask(picture *ebiten.Image) *ebiten.Image {
    effects := &combat.effects
    if mask, ok := effects.Masks[picture]; ok {
        return mask
    }

    bounds := picture.Bounds()
    pixels := make([]byte, 4 * bounds.Dx() * bounds.Dy())
    picture.ReadPixels(pixels)
    for index := 0; index < len(pixels); index += 4 {
        alpha := pixels[index + 3]
        pixels[index] = alpha
        pixels[index + 1] = alpha
        pixels[index + 2] = alpha
    }

    mask := ebiten.NewImage(bounds.Dx(), bounds.Dy())
    mask.WritePixels(pixels)

    if effects.Masks == nil {
        effects.Masks = make(map[*ebiten.Image]*ebiten.Image)
    }
    effects.Masks[picture] = mask
    return mask
}

// how much of its one color a unit that was hit shows now, 0 for none
func flashPart(flash unitFlash, tick uint64) float64 {
    if flash.Ticks == 0 || tick < flash.Start || tick - flash.Start >= flash.Ticks {
        return 0
    }
    // all of it for the first half, then less
    gone := float64(tick - flash.Start) / float64(flash.Ticks)
    if gone < 0.5 {
        return flash.Strength
    }
    return flash.Strength * (1 - gone) * 2
}

// a figure of a unit that was hit, in its one color over its picture
func (combat *CombatScreen) drawFigureFlash(screen *ebiten.Image, unit *ArmyUnit, picture *ebiten.Image, options ebiten.DrawImageOptions, figure int) {
    if !SpellEffects {
        return
    }

    flash, ok := combat.effects.Flashes[unit]
    if !ok {
        return
    }

    part := flashPart(flash, combat.effects.Tick)
    if part <= 0 {
        return
    }

    options.ColorScale.Scale(flash.Color[0], flash.Color[1], flash.Color[2], 1)
    options.ColorScale.ScaleAlpha(float32(part))
    unitview.RenderCombatFigure(screen, combat.figureMask(picture), options, unit.VisibleFigures(), unit.LostUnits, nil, data.UnitEnchantmentNone, combat.Counter, &combat.ImageCache, figure)
}

// what the effects use of the tiles around a place, for a test
func pulseTiles(pulse groundPulse, tick uint64) map[image.Point]int {
    out := make(map[image.Point]int)
    reach := int(math.Ceil(pulse.Radius))
    for y := pulse.Y - reach; y <= pulse.Y + reach; y++ {
        for x := pulse.X - reach; x <= pulse.X + reach; x++ {
            band := pulseBand(pulse, tick, x, y)
            if band > 0 {
                out[image.Pt(x, y)] = band
            }
        }
    }
    return out
}
