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
| 6 | The roads of a town, the clouds of a flying fortress and the ground under an outpost on the original's place. They now scale with the zoom | `combat/scenerydraw.go` |
| 6 | Cursor pictures are drawn with their middle or tip on the mouse position, which is the point that picks the tile | `combat/cursor.go` |
| 6 | The red X has arms of 4 instead of 7, at the same size of pixel (user request) | `combat/cursor.go` |
| 7 | Widescreen: the field spans the width of the window, the combat bar and windows stay in the middle | `combat/widefield.go`, `display/backdrop.go` |
| 7 | Damage numbers: the original's font 2 in its reds with a black border, whole screen pixels. Not in the original | `combat/damagenumbers.go` |
| 7 | Projectiles and damage numbers are drawn under the combat bar, were over it | `combat/combat-screen.go` |
| 8 | Ground around the field: 16 rows of tiles nobody can enter, each darker, black at the last. Trees and rocks continue on it. The camera stops where its view would leave it | `combat/fieldedge.go` |
| 8 | Projectiles keep positions of the field, so they stay on their way while the camera moves or zooms | `combat/projectilespace.go` |
| 8 | Cursor pictures of the field are drawn as large as the tiles, so they shrink when zoomed out | `combat/cursor.go` |
| 8 | Damage numbers in the original's smallest font, in a loose cluster around the unit. One that would touch another moves away the shorter way. Each fades as one picture | `combat/damagenumbers.go` |

## Different from the original on purpose (user requests)
- The figures of a unit do not move as one: each trails its unit by up to 0.09 seconds while it
  walks, and its walking and striking frames are out of step with the others by up to 3 redraws.
  `combat/figurevariety.go`, `FigureVariety` turns it off.
- The outlines of the tiles, the red pulse of the unit under the cursor and the frames of a strike
  step at half the original's speed: every second redraw of its 18.2 a second. `combat/animation.go`.

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
- Roads that lead out of a town, and enchanted roads.
- The sorcery and chaos nodes were not checked.
- A wall of stone rising (the Wall of Stone spell) has no animation.
- The cell outline under the cursor and under the selected unit (doc/Combat/MoX-Combat-Draw-SquareOutline.md).
- Mouse cursor pictures per action (doc/Combat/MoM-CombatScreen-Mouse.md). ON PURPOSE different:
  the original picks the cell 4 right and 4 below the corner of every cursor picture. Ours picks at
  the mouse position and centers the pictures there, which stays exact when zoomed out.
- Projectiles and spell effects (Make_Missiles and the spell animation code).
- A window wider than 16 to 9 can still show black past the last row at the farthest zoom level.
  The rows fade to black, so there is no cut edge.
- The unit information box and the spell announcement stay in the middle 320 columns.

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
