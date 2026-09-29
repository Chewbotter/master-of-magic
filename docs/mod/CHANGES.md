# What this mod changes

A list of what is different from the game this fork is built on (kazzmir's Master of Magic remake), written for players. One line per feature, as it is now: a feature that was changed later has its line rewritten, a feature that was taken out has its line removed. New lines go at the end of their section.

## Display
- Widescreen: the world map and the battlefield fill a wide window, other screens sit in the middle with black bars. A setting, on by default.
- The game is drawn at the real pixel size of the window, so every art pixel is the same whole number of screen pixels.
- Zoom of the world map in whole steps only, so the art is never blurred or uneven.
- Resolution list and Fullscreen in the settings, kept between runs.
- A frame rate counter in the upper right.

## Start screen and settings
- The intro is skipped at the start. The main menu has Quick Start, Continue, Load, New Game, Settings, Credits, Quit.
- One settings screen for the whole game, with small text and room for many settings, and tips for them.
- Controls setting: Modern (default) or Classic. Classic is the controls of the original; everything listed under "modern controls" below is off there.
- Setting "Single strikes": the look of strikes in battle, see Battles.

## World map
- Smooth panning with the middle mouse button (modern controls): the map glides, keeps moving for a moment after release, and the cursor glides with it.
- Camera moves are eased: a right click, the jump to the next unit, and following a unit that walks.
- When you send a stack walking, the camera makes one move to where the stack will stop and waits for it there (modern controls).
- Fog as in the original: land is unexplored (black) or shown as it is. Explored land that no unit or city sees at the moment is not darkened.
- The start of a game reveals the original's area around the first city, 5 by 5 tiles without the corners.
- Newly revealed land fades in, one art pixel at a time, and starts to when a unit sets out for the next tile, not when it has arrived.
- A left click on one of your stacks selects it (modern controls). Shift and a left click sends the selected stack there.
- A move of more than one tile takes two clicks (modern controls): the first shows the path, the second on the same tile sends the stack. A tile next to the stack takes one click.
- The cursor is a red X over a tile the selected stack can not go to, and a click there does nothing (modern controls). No red flash of the screen.
- Of the selected stack only the colored square blinks, the unit stays visible (modern controls). The blinking starts anew with every selection.
- Text of the panel (gold, mana, income) with the original's black outlines.
- Surveyor, cartographer and chancellor's scroll keep the wide map beneath them.

## Battles
- The battlefield, the places of figures, deployment, timing, trees, rocks, houses and walls follow the original.
- The bar at the bottom matches the original: text, names, health bars.
- Zoom in whole steps and smooth panning. The field reaches across a wide window, with darkening ground around it.
- Ground made as the original makes it: grass and dirt in patches, the roads of the world map crossing the field (enchanted roads in gold), a forest with many more trees.
- Movement costs of the original: trees and mud slow a unit down, a road is faster than grass.
- Raised ground forms plateaus and small hills with shaded slopes and rounded outlines, most of them in hills and mountains. Only going up or down a slope costs more, the top is ground as any other. The original's raised ground is left as short mounds, for the look only.
- Biomes: a battle in a forest, in a swamp, in hills, on a volcano or on a mountain beside tundra can have pictures of its own. The game has none yet, so until an artist adds some they look as their landscape does. A volcano is fought on the ground of the mountains.
- A coast on the side of the field where the sea lies on the world map: a beach, then water. The armies stand on land. The beach is rough ground and costs twice as much to cross; only units that fly, swim or sail can go into the water.
- Large pieces of ground over 2 by 2 tiles, such as a cluster of rocks or a patch of dirt, each once per battle at most. For the look only.
- Units the computer controls act at the same time, and decide as they would in turn.
- Space toggles auto combat; the AUTO button is lit while it is on.
- The area the selected unit can move to is shown. A click on a unit picks it; WAIT goes through the units by where they stand.
- A chevron marks the unit whose turn it is.
- Cursors sit on the tile they act on and are drawn on the pixels of the field.
- Figures of a unit move and strike each with timing of their own.
- A strike is a swing of steps: back, forward, the blow where the swing lands. Units that fight across a corner close in. With the setting "Single strikes" off, the simpler strike of before.
- Killed figures are thrown back and fall; corpses stay on the field, gray, and the oldest fade when a tile has more than 4.
- Figures cast shadows.
- Blood made of particles, dust for the undead and mist for spirits; it stains the ground and the corpses it lands on. Sparks for a blow that does no damage.
- Damage numbers in the original's style, as large as the field is drawn.
- Spells play as in the original, and add to it: particles where they hit, marks on the ground, light that darkens the field around it, a lit rim on the figures that face it, corpses in the color of what killed them.
- A battle that is won or lost is shown for a moment before it ends.

## Spellbook
- Descriptions and the info window of a spell in smaller, sharp text.
- The page turn is made by the game from the pages as they are, forward and back.
- In battle the realms that filter the spells are in a row over the book.

## For artists
- A replacement folder (`mod/`): a picture in it takes the place of the game's own. Figures, cursors, spells, the surroundings of battles, and any picture of the game's archives.
- The folder can ADD pictures: more ground tiles, trees, rocks, houses, large pieces, and props that lie or stand. Added ground tiles are sprinkled in among the game's own, which stay the most, and the same one does not show twice close together; of the others every picture is used once before any comes again.
- Biomes and the coast have folders of their own in the replacement folder. A biome only needs the pictures that differ from its landscape.
- Figures can have more frames: a longer strike, dying and lying.
- Values of spell effects and blood in a text file that is read again while the game runs.
- A tool that writes the game's pictures as png files, the surroundings of battles under names that say what each picture is for, and Aseprite files for figures and spells with an export back into the replacement folder.

## For testing (debug)
- A Debug list on the start screen: World Map, Random Battle, Random City Battle, Test Battle of a unit picked from a list (with its biome and coast), Army Size. Random battles have a landscape, a biome, a coast and roads by chance.
- World Map: a quick game with unlimited moves, no greetings of rival wizards, 3 more units around the city, and units that are never disbanded.
- In debug battles the player knows every spell and does not run out of mana.
- Escape in anything started from the Debug list goes back to the start screen.
- A Debug menu on the world map: Reveal All, Unlimited Moves, No Greetings.
- Under the Debug list: which build this is (lane, commit, time).
