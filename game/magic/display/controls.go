package display

// Control type: Classic is the original game's controls, left untouched. Modern is where new
// control changes go. Saved with the other machine preferences in the settings file.

type ControlType string

const ControlsModern ControlType = "modern"
const ControlsClassic ControlType = "classic"

const DefaultControlType = ControlsModern

// true when the modern controls are in use. new control behavior checks this
func ModernControls() bool {
    return Current.Controls() == ControlsModern
}

// the control type in use. anything unrecognized counts as the default
func (settings *Settings) Controls() ControlType {
    switch settings.ControlType {
        case ControlsModern, ControlsClassic:
            return settings.ControlType
    }
    return DefaultControlType
}

func (settings *Settings) SetControlType(controlType ControlType) {
    settings.ControlType = controlType
    settings.Save()
}

// the name shown in the settings screen
func (controlType ControlType) Name() string {
    switch controlType {
        case ControlsClassic: return "Classic"
        case ControlsModern: return "Modern"
    }
    return "Modern"
}
