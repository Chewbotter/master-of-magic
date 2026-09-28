package spellbook

import (
    "github.com/kazzmir/master-of-magic/lib/font"
)

// Size of the spell descriptions in the research spellbook, relative to the art. At full size the
// longer descriptions run into the next spell's name. See lib/font/textsize.go: the size is rounded
// to whole screen pixels so the letters stay sharp.
const SpellDescriptionTextSize = font.TextSizeCompact
// what a right click on a spell shows about it. its headline stays at full size
const SpellInfoTextSize = font.TextSizeCompact
// spells that can be given more power are cast with all the power the caster has for them, and
// the window that asks how much does not open. false in the game; the debug battles of the start
// screen set it for as long as they run (user, 2026-09-28)
var FullPowerWithoutAsking bool

// development: the spellbook for casting shows the turn of its first page, held at this step
// of the turn, of captureFlipSteps (of the 4 pictures of the old turn). below 0: not
var CaptureFlip = -1
const captureFlipSteps = 10.0
// the turn that is held is one back, from the left to the right
var CaptureFlipBack bool
// the turn of which page: 0 the first, 1 the one after it
var CaptureFlipPage = 0

// development: the name of a spell whose info the research spellbook opens by itself, once
var CaptureSpellInfo string
// the Research Cost and Casting cost lines under each spell name
const SpellCostTextSize = font.TextSizeCompact
