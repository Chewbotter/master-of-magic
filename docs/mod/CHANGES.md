# What this mod changes

A list of what is different from the game this fork is built on (kazzmir's Master of Magic remake), written for players. One line per feature, as it is now: a feature that was changed later has its line rewritten, a feature that was taken out has its line removed. New lines go at the end of their section.

## Display
- Widescreen: the world map and the battlefield fill a wide window, other screens sit in the middle with black bars. A setting, on by default.
- The game is drawn at the real pixel size of the window, so every art pixel is the same whole number of screen pixels.
- Zoom of the world map in whole steps only, so the art is never blurred or uneven.
- Resolution list and Fullscreen in the settings, kept between runs.
- F10 shows and hides a counter of frames per second in the upper right corner.

## Start screen and settings
- The intro is skipped at the start. The main menu has Quick Start, Continue, Load, New Game, Settings, Credits, Quit.
- One settings screen for the whole game, with small text and room for many settings, and tips for them.
- Controls setting: Modern (default) or Classic. Classic is the controls of the original; everything listed under "modern controls" below is off there.
- Setting "Single strikes": the look of strikes in battle, see Battles.
- Setting "Hide cursor on attack", on at first: see the cursors of Battles.
- Setting "Pulsing spellbook text", on at first: off, nothing in the spellbooks pulses, as in the original.
- Setting "Damage numbers", on at first: off, no numbers rise from units that are hit, as in the original.
- Setting "Enemy AI": Chewbot (default) or Clone original. Chewbot is the original game's AI rebuilt, to be changed from there; so far its battles (see Battles), what its cities build and buy, and where its armies, settlers, engineers and ships go on the world map; the rest is still the remake's.

## World map
- The menus of choices (Info, Game) and the advisors of Info have the original's text: its letters, colors, edges and places.
- Messages and popups have the original's text: the box of a message, what a stack has found (lair, node, tower), the help scroll, the box of a finished building and of an event, the windows of a hero or mercenaries for hire and of the merchant, the box a name is typed in, what was found in a lair, a hero that has made a level, the window of an outpost, the pictures of a summoning and of a spell for the whole world, the words over a banished wizard, the window of a city of another wizard, the screen of the items.
- The picture of a summoning has the original's colors of the realm: the ring on the floor and the edges of the table are in the color of the spell too, not only the flames.
- The events of a turn are on one scroll, under their headings, as in the original; a text that is too long for it can be moved with the wheel, the arrow keys or the arrows on the scroll.
- With the Enemy AI Chewbot, the cities of computer wizards build and buy as in the original: by the wizard's personality (Militarist, Theurgist, Perfectionist, Expansionist), one settler at a time, army buildings first away from you, often Trade Goods; neutral cities build Barracks first.
- With the Enemy AI Chewbot, computer wizards move on the world map as in the original: settlers look for the best site near home, garrisons grow with their cities, units beyond what their cities need gather into expeditions sized by the empire (a small empire sends small ones) that march on neutral cities, lairs, nodes and the wizards they are hostile to, engineers link cities with roads, and armies that run out of targets wait at the coast for ships. Before turn 100 they leave other wizards alone unless attacked.
- With the Enemy AI Chewbot, computer wizards cast spells on the world map as in the original: summons (the costliest creature they know), enchantments of their units and cities, curses and attacks on the cities and armies of wizards they are hostile to, spells against your mana and casting, global enchantments and Disjunction against yours; they research and split their power as the original does. Their spells that need a target now work (before, any spell of a computer wizard aimed at a square did nothing), and are shown only where you can see them.
- With the Enemy AI Chewbot, computer wizards keep their books as the original does: they shift gold and mana between their reserves, raise and lower taxes, add farmers when food runs short, disband their weakest units (and summons) when they cannot pay for them, and keep one settler per landmass.
- Difficulty works as in the original: above Intro, computer wizards' cities grow faster and make more food, production, gold, mana and research, their nodes give more power and they pay less upkeep (at Average half again as much, at Hard twice, at Impossible four times; Extreme counts as Impossible). Neutral towns grow slowly and stop at a size set by the difficulty, outposts grow faster at higher difficulty, and gold and mana are capped at 30000.
- Smooth panning with the middle mouse button (modern controls): the map glides, keeps moving for a moment after release, and the cursor glides with it.
- Camera moves are eased: a right click, the jump to the next unit, and following a unit that walks.
- When you send a stack walking, the camera makes one move to where the stack will stop and waits for it there (modern controls). Toward a lair, a ruin or a node it goes to the tile before it, where the stack waits for your answer, so it does not move there and back when you do not enter.
- Fog as in the original: land is unexplored (black) or shown as it is. Explored land that no unit or city sees at the moment is not darkened.
- The start of a game reveals the original's area around the first city, 5 by 5 tiles without the corners.
- Newly revealed land fades in, one art pixel at a time, and starts to when a unit sets out for the next tile, not when it has arrived.
- A left click on one of your stacks selects it (modern controls). Shift and a left click sends the selected stack there.
- A move of more than one tile takes two clicks (modern controls): the first shows the path, the second on the same tile sends the stack. A tile next to the stack takes one click.
- The path of a stack shows boots as far as it gets in this turn, and flags in the color of your wizard on the tiles it will walk in later turns.
- A stack that goes on along its path when a turn starts is shown for a moment before it walks.
- Space ends the turn (modern controls), as N does. Home puts the camera on the selected stack.
- Over the map the cursor is as large as the map is drawn, at every zoom level (modern controls).
- The cursor is a red X over a tile the selected stack can not go to, and a click there does nothing (modern controls). No red flash of the screen.
- Of the selected stack only the colored square blinks, the unit stays visible (modern controls). The blinking starts anew with every selection.
- Text of the panel (gold, mana, income, moves) with the original's black outlines.
- The screen that picks what a city builds has the original's look: the names in the lists, the text and the places in the window. 
- The screen of a city has the original's text and places for the name of the city, the race, the population, and in the field of what it builds the turns, the name and the description.
- The window of a unit has the original's text: outlined letters in its colors, at its places, on the world map and in battles. The list of abilities is in smaller letters.
- Surveyor, cartographer and chancellor's scroll keep the wide map beneath them.
- Diplomacy is the original game's: every computer wizard has a personality and an objective, relations rise and fall with what you do (battles, taken cities, curses, enchantments, treaties kept or broken), the wizards make pacts, alliances and wars among themselves, and they speak to you at the start of your turn in the original's words: a greeting when you meet, warnings, broken treaties, declarations of war, offers of peace and proposals you can accept or reject. Talking to a wizard (magic screen) works as in the original: treaties, threats, tribute, exchange of spells, and a wizard that has had enough of you will not talk. Attacking a wizard you have a treaty with asks first.
- The worlds are the original game's: 60 by 40 squares at every Land Size, which says how much of it is land (small, medium, large), grown into continents and islands as the original grows them, with its tundra poles, deserts and swamps, 16 nodes on Arcanus and 14 on Myrror, 6 towers, about 57 lairs, 15 neutral cities on each plane, and the wizards' homes spaced as in the original. You meet another wizard when you see its city or units; computer wizards meet each other only through Nature's Awareness, as in the original.

## Battles
- The battlefield, the places of figures, deployment, timing, trees, rocks, houses and walls follow the original.
- The bar at the bottom matches the original in its text and names. The panel of the selected unit is arranged anew: the name over the figure, the health bar under the name, the numbers in a column with more room.
- What is known of the unit under the mouse stands in the upper right corner of the window, also in widescreen, in the original's letters and pictures without a panel: the name, the health bar under it and one column of attack, defense, resistance and moves, all ending on one edge.
- The outline of the unit under the mouse pulses to a soft gray for your units and to red for the enemy's, starting from black each time the mouse comes over a unit.
- A unit with a spell on it, and an item with powers, has the original's outline everywhere: a rim of one pixel in the shades of the realm of the spell, which run around the figure.
- While Tab is held, units whose turn is over are gray.
- Zoom in whole steps and smooth panning. The field reaches across a wide window, with darkening ground around it.
- Ground made as the original makes it: grass and dirt in patches, the roads of the world map crossing the field (enchanted roads in gold), a forest with many more trees.
- Movement costs of the original: trees and mud slow a unit down, a road is faster than grass.
- Raised ground forms plateaus and small hills with shaded slopes and rounded outlines, most of them in hills and mountains. Only going up or down a slope costs more, the top is ground as any other. The original's raised ground is left as short mounds, for the look only.
- Biomes: a battle in a forest, in a swamp, in hills, on a volcano or on a mountain beside tundra can have pictures of its own. The game has none yet, so until an artist adds some they look as their landscape does. A volcano is fought on the ground of the mountains.
- A coast on the side of the field where the sea or a lake lies on the world map: a wide beach that is narrow in places, then water, with ragged edges where grass runs into sand and sand into water. The armies stand on land. The beach is rough ground and costs twice as much to cross; only units that fly, swim or sail can go into the water. A tile on an edge counts as what lies under the sand: ground or water. On tundra the sea is frozen.
- Farmland: a battle one or two tiles from a town has houses of the race of the town here and there, more of them right beside the town. On plain grass land it is fought on the fields of the town: plots of many sizes, square and oblong, that take about half of the ground, each of one kind of crop, with a row of dirt or grass between them. Fences stand along sides of the plots and beside the roads, and small things of a farm lie about. Other landscapes and biomes keep their ground. Roads are there where the world map has them. Crops cost what grass costs.
- Forests are woods and clearings: many more trees, which stand close together in woods with open ground between them, and a clearing in the middle of the field where the armies start. Trees slow units down as before, so the woods are slow ground.
- Mountains and hills have more rocks.
- Swamps are islands in brackish green water: about half of the ground is raised, all that is low is under water. Roads run through the water as fords. Splashes in it are green. A swamp has twice the trees of grass land, on its islands. Units can wade through; a tile of a pool is rough ground and costs twice as much to cross.
- Weather, in test battles for now: light rain, heavy rain, light snow, heavy snow, and the shadows of clouds that drift over the ground and darken the units and the land under them. Rain and snow come down in gusts, as single pixels. Weather changes no rule.
- Rivers, in debug battles for now: shallow water 2 to 3 tiles wide that winds between the armies or beside them, over the ground of whatever the landscape is. Every unit can wade through; a tile of the river is rough ground and costs twice as much to cross, its banks cost what the ground costs. A battlefield with a river has no roads, a town has no river. A river that runs toward a coast runs into the sea, which comes in to meet it in a small bay. On tundra the river is frozen. Units that walk into the water throw up a small splash, a few drops for a single figure, a group not much more.
- Large pieces of ground over 2 by 2 tiles, such as a cluster of rocks or a patch of dirt, each once per battle at most. For the look only.
- Units the computer controls act at the same time, and decide as they would in turn.
- Space toggles auto combat; the AUTO button is lit while it is on.
- With the modern controls the buttons of a battle are AUTO, STAY and END in the place of WAIT, AUTO and DONE: STAY ends the turn of the selected unit, END the turn of all your units that still have theirs.
- What is marked on the ground (where a unit can go, the outlines of cells) shows over corpses. While Tab is held the corpses fade away.
- The area the selected unit can move to is shown. A click on a unit picks it; WAIT goes through the units by where they stand.
- A chevron marks the unit whose turn it is.
- Cursors sit on the tile they act on and are drawn on the pixels of the field. After a click that sends a unit to strike or to shoot the cursor fades away until the mouse moves.
- Figures of a unit move and strike each with timing of their own.
- A strike is a swing of steps: back, forward, the blow where the swing lands. Units that fight across a corner close in. With the setting "Single strikes" off, the simpler strike of before.
- Killed figures are thrown back and fall; corpses stay on the field, gray, and the oldest fade when a tile has more than 4.
- Figures cast shadows.
- The color of a wizard on figures and units keeps the shading of the art for every color; before, red, blue and yellow were nearly flat.
- Blood made of particles, dust for the undead and mist for spirits; it stains the ground and the corpses it lands on. Sparks for a blow that does no damage.
- Damage numbers in the original's style, as large as the field is drawn. Many at once spread out and keep apart while they rise. Bright red over the enemy's units, darker over yours.
- Spells play as in the original, and add to it: particles where they hit, marks on the ground, light that darkens the field around it and brightens the ground near it in its color, a lit rim on the figures that face it, corpses in the color of what killed them.
- A battle that is won or lost is shown for a moment before it ends.
- With the Enemy AI Chewbot, computer armies (and yours on auto) fight as in the original: they gather at a line and advance together, shooters stay back and shoot, heroes avoid fights they would lose, defenders keep to their walls and guard the gate, attackers go for the gate, and a hopeless army flees to save its heroes. Wizards and units cast as in the original too: the spell that suits how the battle stands, at the target the original would pick.

## Spellbook
- Descriptions and the info window of a spell in smaller, sharp text.
- The tip that says how many turns a spell takes to cast is solid and has a shadow of whole pixels, as all tips have now.
- The page turn is made by the game from the pages as they are, forward and back. The folded corners that turn the pages turn with their page, and are lighter under the mouse. A click on a tab while a page turns turns the next one at once.
- In battle the realms that filter the spells are in a row over the book.
- The letters of both spellbooks have the original's colors: lighter soft edges, and shades in the headers.
- The spell under the mouse pulses softly to the color of its realm (green for nature, blue for sorcery, red for chaos, white for life, purple for death), in the book for casting and when a spell to research is picked. The pulse starts when the mouse comes over the spell.
- Bookmarks at the right edge of the book for casting, one for every kind of spell with a symbol of its own: a click turns the book to that kind.
- The dark behind the spellbook fades in and out with it, and in a battle it covers the whole width of the window.

## For artists
- A replacement folder (`mod/`): a picture in it takes the place of the game's own. Figures, cursors, spells, the surroundings of battles, and any picture of the game's archives.
- The folder can ADD pictures: more ground tiles, trees, rocks, houses, large pieces, and props that lie or stand. Added ground tiles are sprinkled in among the game's own, which stay the most, and the same one does not show twice close together; of the others every picture is used once before any comes again.
- Biomes, the coast, the river and the farmland have folders of their own in the replacement folder. The beach and the water of a coast can be different for grass land, desert, mountains and tundra. The river takes glints: small animated pictures that are scattered over its water. A biome only needs the pictures that differ from its landscape.
- Figures can have more frames: a longer strike, dying and lying.
- The flag of the paths on the world map, the tabs of the spellbooks under the mouse and the bookmarks of the spellbook can be repainted.
- Values of spell effects and blood in a text file that is read again while the game runs.
- A tool that writes the game's pictures as png files, the surroundings of battles under names that say what each picture is for, and Aseprite files for figures and spells with an export back into the replacement folder.

## For testing (debug)
- A Debug list on the start screen: World Map, Random Battle, Random City Battle, Test Battle of a unit picked from a list (with its biome, coast, river, farmland, roads and weather, or a coast or river by chance; a button starts the last unit again on what is picked), Army Size. Random battles have a landscape, a biome, a coast, farmland and roads or a river by chance.
- World Map: a quick game with unlimited moves, no greetings of rival wizards, 3 more units around the city, and units that are never disbanded.
- In debug battles the player knows every spell and does not run out of mana.
- Escape in anything started from the Debug list goes back to the start screen.
- A Debug menu on the world map: Reveal All, Unlimited Moves, No Greetings.
- Under the Debug list: which build this is (lane, commit, time).
