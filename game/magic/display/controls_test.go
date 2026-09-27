package display

import (
    "testing"
)

// a settings file without the key, or with a value this build does not know, gets the modern controls
func TestControlTypeDefaultsToModern(test *testing.T) {
    settings := MakeDefault()
    if settings.Controls() != ControlsModern {
        test.Fatalf("new settings: %v, expected modern", settings.Controls())
    }

    settings.ControlType = ""
    if settings.Controls() != ControlsModern {
        test.Fatalf("missing value: %v, expected modern", settings.Controls())
    }

    settings.ControlType = "something else"
    if settings.Controls() != ControlsModern {
        test.Fatalf("unknown value: %v, expected modern", settings.Controls())
    }

    settings.ControlType = ControlsClassic
    if settings.Controls() != ControlsClassic {
        test.Fatalf("classic: %v", settings.Controls())
    }
}
