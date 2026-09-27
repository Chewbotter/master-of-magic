package spellbook

import (
    "github.com/kazzmir/master-of-magic/lib/font"
)

// Size of the spell descriptions in the research spellbook, relative to the art. At full size the
// longer descriptions run into the next spell's name. See lib/font/textsize.go: the size is rounded
// to whole screen pixels so the letters stay sharp.
const SpellDescriptionTextSize = font.TextSizeCompact
// the Research Cost and Casting cost lines under each spell name
const SpellCostTextSize = font.TextSizeCompact
