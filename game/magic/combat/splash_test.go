package combat

import (
    "testing"
)

// one figure makes a splash that is seen, many figures make more of it but far less than as many
// times as much, and wading on makes less than walking in
func TestSplashCount(test *testing.T) {
    single := splashCount(1, 1)
    group := splashCount(8, 1)

    if single < 4 {
        test.Fatalf("a single figure throws %v drops", single)
    }
    if group <= single || group > single * 3 {
        test.Fatalf("8 figures throw %v drops, one throws %v", group, single)
    }
    for figures := 1; figures < 8; figures++ {
        if splashCount(figures + 1, 1) < splashCount(figures, 1) {
            test.Fatalf("%v figures throw less than %v", figures + 1, figures)
        }
    }
    if wade := splashCount(1, splashWade); wade >= single || wade < 1 {
        test.Fatalf("wading throws %v drops, walking in %v", wade, single)
    }
    if splashCount(0, 1) != 0 {
        test.Fatalf("no figures throw drops")
    }
}

// the drops of a splash go up, come down and are gone
func TestSplashDrops(test *testing.T) {
    var system particleSystem
    system.emitSplash(100, 100, 20, splashColors)
    if len(system.Particles) != 20 {
        test.Fatalf("%v drops", len(system.Particles))
    }

    highest := 100.0
    for range 120 {
        system.step(1.0 / 60)
        for _, each := range system.Particles {
            highest = min(highest, each.Y)
            if each.Y > 100 + splashSpreadY + splashSpeed {
                test.Fatalf("a drop falls to %v", each.Y)
            }
        }
    }
    if highest > 98 {
        test.Fatalf("the drops rise to %v", highest)
    }
    if len(system.Particles) != 0 {
        test.Fatalf("%v drops are left", len(system.Particles))
    }
}
