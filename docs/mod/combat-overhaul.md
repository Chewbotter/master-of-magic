# Combat scene overhaul

Goal: the combat scene looks and reads like the original game. The reference is the ReMoM
reconstruction (see CLAUDE.md, Reference workflow): we read facts from it and write our own code.

Work goes in batches. After each batch the user compares the result with the original.

## How our battlefield maps to the original's

The original's grid is 21 by 22 cells, (cgx, cgy). Ours is 30 by 30 tiles, (x, y), and already lays
out armies, the city walls and the gate like the original under this mapping:

    cgx = y - 3
    cgy = 22 - x

The anchor of a cell on the screen is x = (cgx - cgy) * 16 + 158, y = (cgx + cgy) * 8 - 80, the top
corner of its diamond. The middle of the diamond is 1 right and 8 below the anchor. Our camera matrix
maps a tile to the middle of its diamond. `game/magic/combat/battlefield.go` holds all of this.

## Done

| Batch | What | Where |
|---|---|---|
| 1 | Combat bar: chiseled text, banner shaded names, exact positions, health bar on its track | `combat/hudstyle.go`, `lib/font/styled.go` |
| 2 | Projection: 16 across and 8 down per cell, was about 14.5 and 7.8 | `combat/battlefield.go` |
| 2 | Terrain pictures sit on their own diamonds, were shifted half a tile | `combat/combat-screen.go` |
| 2 | Figure positions for units of 1 to 8 figures, were "copied from case 8" guesses for 2 to 7 | `unitview/combat.go` |
| 2 | Figure pictures anchored by their feet at 13, 23 | `unitview/combat.go` |
| 2 | Cave, tower, temple, keep, ruins, node pictures on the original's cell (6, 11) with its anchor | `combat/model.go`, `combat/battlefield.go` |
| 3 | Camera: whole pixel zoom levels, smooth panning with the middle mouse button (Modern controls) and the arrow keys, release glide. Space returns to the original view | `combat/camera.go` |
| 4 | Deployment: melee units in front, ranged behind, wall corners and the structure's place skipped | `combat/deploy.go` |
| 4 | Figure frames: standing is frame 1, walking 1 2 1 0, flying units cycle 0 1 2, strikes alternate 1 and 3, all at 18.2 steps a second | `combat/animation.go` |
| 4 | A step to the next cell takes 8 of the original's ticks, straight or diagonal | `combat/animation.go` |
| 4 | Cell outlines are the original's pictures: blue under the cursor, red under the selected unit | `combat/animation.go` |
| 4 | The unit under the cursor: the outline of its figures pulses black to red. The selected unit no longer pulses in brightness | `combat/animation.go` |
| 5 | Trees and rocks: the original's counts per landscape, trees in patches, its five tree and five rock pictures, its anchors | `combat/scenery.go` |
| 5 | City: one house per citizen up to a full town, house style by race, fortress and outpost on cell (6, 11) | `combat/scenery.go` |
| 5 | Walls of stone, fire and darkness: the original's 12 and 14 pieces with their cells, shifts and anchors. Myrror has its own stone | `combat/scenery.go`, `combat/scenerydraw.go` |
| 5 | Draw order: figures one by one, trees, rocks, houses, walls and the structure all in the original's order | `combat/scenerydraw.go`, `unitview/figure.go` |
| 5 | The fortress takes its cell. A city without one has no blocked cell | `combat/model.go` |

## Judgment calls of batch 5
- Beyond the original's screen (our field is larger and the camera zooms out) scenery continues at
  the same density: three times the count again, `sceneryBeyondScreen`.
- The original draws rocks with the count and the picture numbers of the trees (its bug). Ours draws
  every rock with its own picture.
- The original reads the broken state of the near walls from the cells of the far walls (its bug or
  the reconstruction's). Ours uses each piece's own cell.
- Rising walls of fire and darkness keep the fork's timing, 8 of our ticks per frame.

## Still different, by area

Found while reading the reference. Not started unless noted.

### Units on the field
- When the original shows each cell outline depends on the cursor's action (Assign_Mouse_Image). Ours
  always shows the blue one under the cursor and the red one under a selected unit that stands still.
- Armies of more than 12 units do not exist in the original. Ours places the extra units with the
  old search.
- Unit enchantment outlines, invisibility and other figure effects (Combat_Figure_Effect,
  Combat_Unit_Enchantment_Outline_Draw).
- Death: the original plays "gibs" frames per lost figure. Ours fades the figure to a color.
- Missiles, vortexes and curse pictures are not part of the draw order yet: drawn after everything.

### Battlefield
- Terrain choice per cell (grass, rough and dirt patches), rivers, roads, mud. Trees and rocks only
  stand on grass cells in the original, and add 1 to the cost of moving through their cell. Ours has
  neither.
- Forest squares: the original has 31 to 60 trees there. Our landscapes have no forest, it counts as grass.
- The road and the dirt paths of a town are still the fork's one picture.
- The sorcery and chaos nodes were not checked.
- A wall of stone rising (the Wall of Stone spell) has no animation.
- The cell outline under the cursor and under the selected unit (doc/Combat/MoX-Combat-Draw-SquareOutline.md).
- Mouse cursor pictures per action (doc/Combat/MoM-CombatScreen-Mouse.md).
- Projectiles and spell effects (Make_Missiles and the spell animation code).
- Zoomed out or moved far, the area beyond our 30 by 30 grid shows black. The original's view never
  leaves its grid.
- Combat is not widescreen yet: it is drawn in the middle with black bars.

### Combat bar
- The selected unit's figure is not centered in its 32 by 25 box at (84, 173) by the size of what is
  actually drawn.
- The INFO and SPELL buttons are never shown locked.
- The original always shows the human player's name on the right. Ours shows the attacker there.
- For a lair or a neutral defender the original prints the lair type, "Monsters" or "Raiders" on the
  left. Ours prints the defending player's name.
- Enchantment icon rows.

### Windows
- Unit information popup, combat information window, end of combat scroll.
