package combat

import (
    "testing"

    "github.com/kazzmir/master-of-magic/game/magic/mod"

    "github.com/hajimehoshi/ebiten/v2"
)

func testPictures(count int, width int, height int) []*ebiten.Image {
    var out []*ebiten.Image
    for range count {
        out = append(out, ebiten.NewImage(width, height))
    }
    return out
}

// the place the original puts the effects of a unit at: the top corner of its tile, 8 down
func TestSpellPlace(test *testing.T) {
    x, y := CellToTile(10, 12)
    placeX, placeY := spellPlace(x, y)
    // (10 - 12) * 16 + 158 and (10 + 12) * 8 - 80 + 8
    if placeX != 126 || placeY != 104 {
        test.Fatalf("place %v,%v, should be 126,104", placeX, placeY)
    }
}

// Fire Bolt: 11 redraws from 110 right and 66 above of where it hits, 10 left and 6 down each
func TestFireBoltPath(test *testing.T) {
    combat := &CombatScreen{}
    x, y := CellToTile(10, 12)
    target := &ArmyUnit{X: x, Y: y}

    bolt := combat.createBolt(BoltFire, target, testPictures(4, 28, 30), nil)
    if len(bolt.Steps) != 11 {
        test.Fatalf("%v steps, should be 11", len(bolt.Steps))
    }

    hitX, hitY := 126 - 16, 104 - 20
    first := bolt.Steps[0]
    if first.X != hitX + 110 || first.Y != hitY - 66 || first.Frame != 0 || first.Ticks != 1 {
        test.Errorf("first step %v", describeStep(first))
    }

    last := bolt.Steps[10]
    if last.X != hitX + 110 - 100 || last.Y != hitY - 66 + 60 || last.Frame != 3 {
        test.Errorf("last step %v", describeStep(last))
    }
}

// a bolt goes on between the redraws of the original, by whole pixels, and is where the original
// has it at every redraw
func TestBoltGlides(test *testing.T) {
    combat := &CombatScreen{}
    x, y := CellToTile(10, 12)
    target := &ArmyUnit{X: x, Y: y}

    bolt := combat.createBolt(BoltFire, target, testPictures(4, 28, 30), nil)
    bolt.Started = true
    bolt.Start = 100

    tps := uint64(ebiten.TPS())
    places := make(map[int]bool)
    lastX := 0
    for tick := uint64(0); tick * 182 < tps * 110; tick++ {
        placeX, placeY, _, ok := bolt.placeAt(bolt.Start + tick)
        if !ok {
            test.Fatalf("nothing to show at tick %v", tick)
        }
        if tick > 0 && (placeX > lastX || lastX - placeX > 5) {
            test.Errorf("tick %v: from %v to %v", tick, lastX, placeX)
        }
        lastX = placeX
        places[placeX] = true

        // 10 across for 6 down all the way
        fromStartX := bolt.Steps[0].X - placeX
        fromStartY := placeY - bolt.Steps[0].Y
        if fromStartX * 6 - fromStartY * 10 > 10 || fromStartY * 10 - fromStartX * 6 > 10 {
            test.Errorf("tick %v: %v across and %v down is off the way", tick, fromStartX, fromStartY)
        }
    }

    if len(places) < 30 {
        test.Errorf("the bolt was at %v places only", len(places))
    }

    hitX := 126 - 16
    if lastX < hitX || lastX > hitX + 4 {
        test.Errorf("the bolt ends at %v, it hits at %v", lastX, hitX)
    }
}

// Fireball: 11 redraws of flight with a frame each, then 5 frames where it hits, 2 redraws each
func TestFireballPath(test *testing.T) {
    combat := &CombatScreen{}
    x, y := CellToTile(10, 12)
    target := &ArmyUnit{X: x, Y: y}

    bolt := combat.createBolt(BoltFireball, target, testPictures(16, 28, 30), nil)
    if len(bolt.Steps) != 16 {
        test.Fatalf("%v steps, should be 16", len(bolt.Steps))
    }

    for index, step := range bolt.Steps {
        if step.Frame != index {
            test.Errorf("step %v shows frame %v", index, step.Frame)
        }
        if index >= 11 && (step.X != 126 - 14 || step.Y != 104 - 21 || step.Ticks != 2) {
            test.Errorf("step %v where it hits: %v", index, describeStep(step))
        }
    }
}

// small pictures are 14 left and 25 above of the place, large ones 28 and 30
func TestEffectPlaces(test *testing.T) {
    x, y := CellToTile(10, 12)

    small := effectSteps(testPictures(8, 28, 34), x, y, 0, false)
    if len(small) != 8 || small[0].X != 126 - 14 || small[0].Y != 104 - 25 || small[0].Ticks != 2 {
        test.Errorf("small: %v steps, first %v", len(small), describeStep(small[0]))
    }

    large := effectSteps(testPictures(12, 60, 54), x, y, 0, false)
    if len(large) != 12 || large[0].X != 126 - 28 || large[0].Y != 104 - 30 {
        test.Errorf("large: %v steps, first %v", len(large), describeStep(large[0]))
    }

    // a spell on all units: a wait of whole frames first, and small pictures 21 above
    all := effectSteps(testPictures(16, 28, 34), x, y, 3, true)
    if len(all) != 17 || all[0].Frame != spellFrameNone || all[0].Ticks != 6 || all[1].Y != 104 - 21 {
        test.Errorf("all: %v steps, first %v, second %v", len(all), describeStep(all[0]), describeStep(all[1]))
    }
}

// Lightning Bolt: 5 redraws, 2 flashes of a redraw from 199 above the place, 5 redraws
func TestLightning(test *testing.T) {
    combat := &CombatScreen{}
    x, y := CellToTile(10, 12)
    target := &ArmyUnit{X: x, Y: y}

    bolt := combat.createLightning(target, testPictures(4, 184, 200), nil)
    if len(bolt.Steps) != 4 {
        test.Fatalf("%v steps, should be 4", len(bolt.Steps))
    }
    if bolt.Steps[0].Frame != spellFrameNone || bolt.Steps[0].Ticks != 5 || bolt.Steps[3].Frame != spellFrameNone || bolt.Steps[3].Ticks != 5 {
        test.Errorf("waits: %v and %v", describeStep(bolt.Steps[0]), describeStep(bolt.Steps[3]))
    }
    for _, step := range bolt.Steps[1:3] {
        if step.X != 126 || step.Y != 104 - 199 || step.Ticks != 1 || step.Frame < 0 || step.Frame > 3 {
            test.Errorf("flash: %v", describeStep(step))
        }
    }
}

// a spell goes through its steps by the clock of the original and is over after the last
func TestSpellSteps(test *testing.T) {
    ticks := uint64(ebiten.TPS())
    projectile := &Projectile{Scripted: true, Steps: []ProjectileStep{{Ticks: 2}, {Ticks: 2}, {Ticks: 1}}}

    if !projectile.updateSteps(1000) || projectile.Step != 0 {
        test.Fatalf("at its start: step %v", projectile.Step)
    }

    // after a second the original has redrawn 18 times: all 5 redraws of the spell are over
    if projectile.updateSteps(1000 + ticks) {
        test.Errorf("after a second it should be over, it is at step %v", projectile.Step)
    }

    // 3 redraws of the original are 3 / 18.2 seconds: the second step
    at := uint64(float64(ticks) * 3.2 / OriginalTicksPerSecond)
    projectile.updateSteps(1000 + at)
    if projectile.Step != 1 {
        test.Errorf("after 3 redraws: step %v, should be 1", projectile.Step)
    }
}

// a lightning bolt with a series of frames of the replacement folder plays all of them in order
func TestLightningSeries(test *testing.T) {
    combat := &CombatScreen{}
    target := &ArmyUnit{X: 5, Y: 5}
    pictures := testPictures(6, 184, 200)

    // the game's frames: two flashes by chance between the waits
    projectile := combat.createLightning(target, pictures, nil)
    if len(projectile.Steps) != lightningFlashes + 2 {
        test.Fatalf("%v steps with the game's frames", len(projectile.Steps))
    }

    // a series: every frame once, in its order
    mod.MarkFrameCountChangedForTest(lightningLbx, lightningEntry, 5)
    defer mod.MarkFrameCountChangedForTest(lightningLbx, lightningEntry, 0)
    projectile = combat.createLightning(target, pictures, nil)
    if len(projectile.Steps) != len(pictures) + 2 {
        test.Fatalf("%v steps with a series of %v", len(projectile.Steps), len(pictures))
    }
    for index, step := range projectile.Steps[1:len(projectile.Steps) - 1] {
        if step.Frame != index || step.Ticks != lightningSeriesTicks {
            test.Fatalf("step %v: %v", index, describeStep(step))
        }
    }
}
