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
| 3 | Camera: whole pixel zoom levels, smooth panning with the middle mouse button (Modern controls) and the arrow keys, release glide. Home returns to the original view (space until it became auto) | `combat/camera.go` |
| 4 | Deployment: melee units in front, ranged behind, wall corners and the structure's place skipped | `combat/deploy.go` |
| 4 | Figure frames: standing is frame 1, walking 1 2 1 0, flying units cycle 0 1 2, strikes alternate 1 and 3, all at 18.2 steps a second | `combat/animation.go` |
| 4 | A step to the next cell takes 8 of the original's ticks, straight or diagonal (now 10 on purpose, see below) | `combat/animation.go` |
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
| 9 | Spells: bolts fly the original's paths, effects play once on their target at the original's place and pace, Lightning Bolt comes from the top of the screen, Cracks Call and the summoning circle lie on the ground | `combat/spellanim.go` |
| 9 | The message "X has cast Y" at the top of the screen while a spell plays | `combat/spellanim.go` |
| 9 | Spells on all units of a side start on each unit a little later or earlier | `combat/spellanim.go` |
| 9 | Pictures: Weakness and Black Sleep use the death effect, Shatter and Warp Creature the chaos effect, Word of Death and Death Spell picture 13 | `combat/combat-screen.go` |
| 9 | Spells on the whole battlefield: the screen goes 40 of 100 to the color of the realm, life and arcane white, death black | `combat/spellanim.go` |
| 9 | A summoned unit: the circle alone first, then the unit rises and gets solid | `combat/summon.go` |
| 10 | Ground: every cell is grass, rough or dirt. Rough and dirt come in wandering patches (counts per landscape), rough never next to dirt, on a road or in a town or lair, grass between dirt becomes dirt. Pictures by the kinds of the neighbors: grass shows the edge of dirt next to it, rough is autotiled by its four sides. Made over the field and its border in the original's grid, the original's counts on its grid and as many for the size beyond | `combat/terrain.go` |
| 10 | Forest and hills from the world map: forest (and nature nodes) 30 to 60 trees and few rough patches, hills 20 rough patches. Trees and rocks only on grass without a road | `combat/terrain.go`, `combat/scenery.go`, `game/battleground.go` |
| 10 | Roads: when the battle's tile of the world map has a road, roads run from the middle (from the town's sides in a town) toward each neighbor with a road, wandering as the original's, to the edge of the border. Road pieces by neighbors, two sets of pictures, enchanted roads golden | `combat/terrain.go`, `combat/terraindraw.go` |
| 10 | Movement costs of the ground, in halves of a move: grass and dirt 2, rough 4, a road 1, each tree in the cell 1 more up to 4, mud 12, a diagonal step 1 more. Flying units 2 everywhere. Earth to Mud leaves rough alone. A step is allowed with any movement left before it (was already so) | `combat/movecost.go` |

## Spells: what was not done in batch 9
- Dispel Magic and Dispel Magic True share one function in our code and both show CMBTFX 26. The
  original shows SPECFX 13 for the plain one.
- How long the color of a spell on the whole battlefield takes. The original steps its palette 41
  times without a wait of its own, so the time depends on the machine. Ours keeps 1.5 seconds.
- Call Chaos, Raise Dead and Animate Dead, Magic Vortex, Earth to Mud, the walls rising.
- Teleport and tunneling (Battle_Unit_Teleport, Battle_Unit_Tunnel): ours fade and sink at their own pace.
- Missiles of units (arrows, rocks, magic) are not spells and were not touched (Make_Missiles).
- The marks of curses over a unit (RESOURCE 76 to 82) were not checked.
- Sounds were not checked.

## Different from the original on purpose (user requests)
- Spellbooks, for casting and of research: the leaf that turns is made by the game and moves with every tick, in
  place of the original's 4 pictures (`spellbook/pageturn.go`, `ProceduralPageTurn`).
- Debug battles cast spells that can take more power at full power and do not ask
  (`spellbook.FullPowerWithoutAsking`, set in `fastplay.go` only).
- Where the unit of the player can go shows on the ground, F6 goes through the looks
  (`combat/movearea.go`). Over a cell out of reach: a black outline and the plain cursor in place
  of the blue outline and the red X.
- Spells: their picture lights itself up; particles of single art pixels come off bolts in
  flight; where a spell hits, particles are thrown up, the unit shows in one color, the battle
  stands still for a moment, the view shakes and light runs over the ground
  (`combat/spelleffects.go`, `particles.go`; values per spell in `spellvalues.go` and
  `mod/effects.txt`; F7 turns it off and on). A spell leaves a mark on the ground that stays for
  the battle (`combat/decals.go`; first pictures in `combat/decals/`, made by
  `util/decals/make.py`, replaced by `mod/spells/<name>/decal_NN.png`). The halo was tried and taken out: it made the
  pictures look blurred.
- The light of a spell: the field gets dark while a spell plays, the spell lights what is near
  it, the shadows of the day go out and the units near the spell get shadows that lie away from it, and all of it goes back
  when the spell is over (`combat/spelllight.go`).
- Corpses take on a color by what killed the figure: brown for fire, blue for ice, gray for
  lightning and other spells, red for a fight (`corpse-color` in the values of spells).
- A battle that is won or lost is shown for a moment longer before its result comes up
  (`combat/combatend.go`).
- Spells that hurt do what they do when they hit, not when their pictures have played to their
  end. The figures they kill are thrown away from where the spell hit, by how far the spell
  throws, and lie facing where it came from.
- Blood: the original plays pictures of blood over the figures of a unit that is hurt in a fight
  or by a missile. Here drops of single art pixels fly off the figures and leave stains on the
  ground that stay (`combat/blood.go`). How much a unit bleeds is the original's rule.
- Bolts of spells go on with every tick of the game in place of 10 pixels with every redraw of
  the original (`SmoothBolts` in `combat/spellanim.go`). Way, time and frames are the original's.
- Cursors over the field are drawn on the art pixels of the field (`CursorOnFieldPixels`), the
  hand with the wand is as large as the field is drawn there.
- Damage numbers are as large as the field is drawn (`DamageNumbersZoom`).
- A click on a unit of the player that still has its turn picks it. WAIT and the end of a unit's
  turn go to the next unit on the field, along a row and then the row below, in place of the
  order the units joined the army in (`combat/unitorder.go`). Armies of the computer and auto
  keep the game's order.
- Tab: a double chevron of art pixels over the unit whose turn it is, going up and down, in place
  of the fork's column of light (`combat/unitmarker.go`).
- A magic vortex the player sends on its way: the red outline of its cell, the four tiles it can
  go to as the move area, the black outline elsewhere, in place of the fork's red square.
- No boots on the tiles a unit would walk over (`ShowMovePath` in `combat/autotoggle.go`).
- The AUTO button toggles auto and is lit while auto is on.
- Space sets the player's army to auto and takes it off again (`combat/autotoggle.go`). Taken off
  while units act: units that started their turn finish it, the others keep theirs for the player.
- A figure that attacks steps back and then forward: back on 3, held on 4 and forward on 5 and 6
  with a long strike, back on 1 and forward on 3 with the frames of the game. The figures that are left of a unit slide to their new places in the tile
  when others were lost. `combat/figureslide.go`, `StrikeSlide` and `RegroupSlide` turn them off.
- Figures can have more frames than the original's four, from the replacement folder
  (`game/magic/mod`): 4, 5, 6 make the strike go through 3, 4, 5, 6 and around, 7 and 8 are the
  figure dying and lying. Without them a figure acts as before.
- Units the computer controls act together, not one after the other: the enemy's army, and the
  player's army when it is set to auto. `combat/together.go`, `ActTogether` turns it off. Every
  unit decides on the battle it would have found in its turn: it waits until the unit before it has
  done all its turn does to the battle. Walking and the rest of an attack after the blow overlap.
  Measured on the same battle run both ways, 24 units: every unit stood on the same tile at the
  start of turns 2 and 3, and a turn took about 4 seconds in place of about 19.
- A figure that is killed is pushed back from what its unit faces and tips over around its feet,
  each at a time of its own. It stays as a corpse for the rest of the battle: the picture it had,
  a little darker and with little of its color, in the draw order of everything else. The user
  plans pictures of death poses to replace the tipped over picture. `combat/figurefall.go`, `FigureFall` turns
  it off. The original shows blood on the figure ("gibs"), the fork showed a colored shape that faded.
- Figures cast shadows on the ground, to the lower right, made of pixels of the art.
  `combat/shadows.go`, `FigureShadows` turns them off.
- Units walk slower than in the original: a step to the next cell takes 10 of its redraws, not 8,
  and the walking frames step every second redraw.
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
- Armies of more than 12 units do not exist in the original. Ours stand in rows of 8, five rows
  deep (`combat/deploy.go`). The debug setting Army Size x3 makes such armies.
- Unit enchantment outlines, invisibility and other figure effects (Combat_Figure_Effect,
  Combat_Unit_Enchantment_Outline_Draw).
- Death: the original plays "gibs" frames per lost figure. Ours lets the figure fall, see above.
- Missiles, vortexes and curse pictures are not part of the draw order yet: drawn after everything.

### Battlefield
- The original makes the same battlefield every time at the same place of the world map: it seeds
  its random numbers with the place (wx * wy * (wp + 5) + 10039). Ours makes a new one every battle.
- Rivers: not in the original either. It has the code (Carve_River_Terrain) but never passes it a
  river, so the river pictures of cmbtcity 103 to 108 never show.
- The sorcery and chaos nodes were not checked.
- A wall of stone rising (the Wall of Stone spell) has no animation.
- The cell outline under the cursor and under the selected unit (doc/Combat/MoX-Combat-Draw-SquareOutline.md).
- Mouse cursor pictures per action (doc/Combat/MoM-CombatScreen-Mouse.md). ON PURPOSE different:
  the original picks the cell 4 right and 4 below the corner of every cursor picture. Ours picks at
  the mouse position and centers the pictures there, which stays exact when zoomed out.
- Missiles of units (Make_Missiles). Spells: see batch 9 and what was not done in it.
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
