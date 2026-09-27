# MasterMagic (working title)

A heavily modified Master of Magic, forked from https://github.com/kazzmir/master-of-magic (BSD 3-Clause, keep the LICENSE and copyright notice). Go 1.27 + Ebiten. Not a Godot project: the Godot sections of the guides do not apply.

## Layout
- `D:/Work/MasterMagic_open/mastermagic_game/` : this repo. Remote `origin` is the user's public fork https://github.com/Chewbotter/master-of-magic (master tracks origin/master); `upstream` is kazzmir's original, fetch only. The user commits and pushes through GitHub Desktop.
- `D:/Work/MasterMagic_open/_tools/` : portable Go 1.27.1 and its module cache. Go is not installed system-wide and is not on PATH.
- `D:/Work/MasterMagic_open/_build/` : build output (`magic.exe`, logs). Never commit.
- Original game data (owned on Steam, never copy into the repo, never commit, never redistribute):
  `D:/SteamLibrary/steamapps/common/Master of Magic Classic/Master of Magic Official Release` (101 LBX files).
  Sibling folders hold the Community Patch and Caster of Magic data; the code targets the official release.

## Verify commands (Bash)
- Build: `bash D:/Work/MasterMagic_open/build.sh`
- Tests, once per change: `bash D:/Work/MasterMagic_open/test.sh` (expect every line `ok`, no `FAIL`)
- Menu capture (Claude's instrument): `go build -o ../_build/capture.exe ./test/main-screen-capture`, then run as in Current state.
- Boot probe: `cd D:/Work/MasterMagic_open/_build && timeout 12 ./magic.exe -corner -data "<data path>" -start -music=false > boot.log 2>&1`
  Pass banner: `Loaded data from` followed by `done create neutral player`. Exit code 124 is the timeout, not a failure.
- NEVER write, overwrite or delete `_build/display.json`: it is the user's saved settings. Development runs (`-corner` or `-capture`) read and write `_build/display-dev.json` instead; put probe settings there and remove only that file. Earlier probes deleted the user's file (2026-09-27), which is why his settings kept resetting.
- EVERY run Claude starts passes `-corner` (or `-capture`, which implies it): a 320x200 window in the lower right corner, so nothing pops up over the middle of the user's screen. `capture.exe` always does this. The user's `run.bat` never passes it and stays centered. User request, 2026-09-27.
- The user plays with `run.bat` or `bash run.sh [-start] [-load file] [-watch]`.

## Code map
- `game/magic/main.go` : entry point and command line flags.
- `game/magic/<system>/` : one package per system (city, combat, units, spellbook, ai, maplib, hero, artifact, diplomacy, ...).
- `lib/` : LBX loading, fonts, midi synth. `util/` : dev tools (lbxdump, sprite viewer, combat simulator). `test/` : small runnable probes.
- About 138k lines of Go, 79 test files. Tests run without the LBX data.

## Conventions
- Upstream style: 4 space indent, no gofmt, spelled-out variable names, return errors instead of panic, avoid new libraries.
- Keep changes mergeable with upstream where cheap: prefer new files and named constants over rewriting upstream files.
- No em dashes anywhere.

## Widescreen law (read before touching any screen)
- The picture is always 200 original pixels tall. With Widescreen on it is wider than 320, up to 480, following the window shape (`display.UpdateLayout`, called from `Layout` in `main.go`). Internal scale stays 3.
- A screen with no widescreen layout needs NO changes: `MagicGame.Draw` draws it on a 960x600 canvas in the middle with black bars, and `display.CursorPosition` removes the bar offset from the cursor. Never call `ebiten.CursorPosition` directly; use `display.CursorPosition` or `inputmanager.MousePosition`.
- A small popup that pushes its own drawer can stay over the wide map: call `game.markOverlayDrawer()` right after `PushDrawer` (defer its result), draw the popup ui with `game.drawOverlayUI`, and set the ui's `CursorTransform` to `game.overlayCursorTransform`. Done for the city name prompt (`doInput`). Every other pushed drawer still gets bars.
- A screen WITH a widescreen layout reports it through `display.WideContent`. Today that is only the plain world map: `Game.IsWideOverland` is true when exactly one drawer is on the stack. It then gets the real wide screen and raw cursor positions.
- World map layout (`game/magic/game/wide.go`): the hud frame is one piece of art, drawn against the right edge; the map keeps its origin at the left and shows more columns; `Camera.SizeX` follows the map width; the top bar is extended to the left edge by repeating an empty column of the bar. Base hud elements (layer 0) are drawn on a canvas shifted right by the extra width. Popups (layer above 0) are drawn on a second canvas in the middle. `hudCursorTransform` maps the cursor to match. A popup that covers the whole 320 width gets black bars (pixel probe in `drawPopupBars`).
- Staging agreed with the user: bars everywhere first, world map first, then one screen at a time. Next candidates are the map modes that push their own drawer (spell targeting, surveyor, road building) and combat.
- Dev instrument: `magic.exe -start -capture out.png -capture-frames 150 [-capture-popup]` writes one frame and exits. Capture mode skips the starting city naming prompt. Put a `display.json` beside the exe first to pick widescreen.

## Zoom law
- Only three zoom levels exist, the pixel perfect ones: camera zoom 12, 8, 4 (art at 3x, 2x, 1x). `camera.ZoomMin = 4`, `camera.ZoomStep = 4`, one wheel notch per level. The user rejected the in-between levels as blurred (2026-09-27). The canvas path below now only runs during the short zoom animation.
- World map art is only ever drawn at whole multiples of its size. `game/magic/game/zoom.go`: when the zoom scale (3 x camera zoom) is not whole, the map is drawn into a canvas at the next whole multiple up (by temporarily changing `scale.ScaleAmount`, which every map draw goes through via `scale.DrawScaled`) and the canvas is shrunk once with FilterLinear. Anything new drawn on the map must go through `scale.DrawScaled` or it will land at the wrong size in the canvas pass.
- Rejected: per-tile `ebiten.FilterPixelated`. It blends each tile's edge with transparency and leaves a grid of seams. Do not bring it back.
- `OverworldCleanZoom = false` restores the original look; dev flag `-capture-nearest` does the same.

## Dev capture flags (magic.exe)
`-capture out.png -capture-frames N` (implies `-corner`), `-capture-zoom 2..12`, `-capture-zoom-animation -1..1`, `-capture-window-width W` (corner window W x 200, tests other screen shapes), `-capture-popup`, `-capture-nearest`. The log line reports the frame rate at capture time. Frame rate readings in the corner window vary between about 60 and 115 run to run; compare several runs, never one.

## Original look law
- Text on the world map panel uses whole art pixel black borders, never the sub-pixel brown drop shadow upstream used. `lib/font/outline.go`: `PrintOutlined` with `OutlineFull` (Gold, Food, Mana income) or `OutlineBottomRight` (GP, MP reserves). Styles are named in `game/magic/game/panel.go`. Reserves keep the space, `2 GP`: the user asked for it after trying the original's `3166GP` (2026-09-27).
- Other screens may carry the same upstream shadow; check against the original before assuming.

## Debug menu
- `game/magic/game/debug.go`. The Debug button is a hud element in the upper left: in widescreen inside the empty stretch of the top bar (its rect is to the left of zero in the shifted hud's coordinates, and `drawHud` draws it on the screen), otherwise in the top left corner of the map. It opens the original selection menu (`uilib.MakeSelectionUI`, the Info menu's look). New options go in `makeDebugMenu`; state is package variables, session only.
- Reveal All swaps in a fully visible fog map for drawing only (`revealedFogFor`). The fog map must never be nil: unit drawing indexes it.

## Current state
- 2026-09-27 (debug, pan): Debug button and menu with Reveal All. Middle mouse button drag pans the world map (`game/magic/game/pan.go`, `PanButton`), starting only on the map, keeping the camera's vertical limits. The camera keeps a fractional position after a pan; the next jump to a tile drops the fraction.
- Capture options that must wait for the start of game events (pan, open menu) run through `capture.Later`, 20 frames before the capture. Flags: `-capture-reveal-all`, `-capture-debug-menu`, `-capture-pan-x`, `-capture-pan-y`.
- 2026-09-27 (defaults): new defaults are widescreen on, window scale 5 (1776 x 1000), music volume 0 (`music/defaults.go`, volume is still not saved between launches). City name prompt keeps the wide map beneath it. Dev flag `-capture-city-prompt` keeps the prompt in a capture.
- 2026-09-27 (zoom levels, panel text): zoom limited to the three pixel perfect levels; panel text restyled to the original. Dev flag `-capture-no-selection` shows the income pictures.
- 2026-09-27 (zoom): clean world map zoom (see Zoom law). FPS counter in the upper right of the real screen (`game/magic/fps.go`, `ShowFPS`). The user reported a gray right edge on the world map that flickers with zoom in widescreen; NOT reproduced: sweeps of 11 zoom levels, 13 mid-animation states and 7 screen widths from a fresh quick start showed no gray at the right edge. Waiting on a screenshot. Found instead: zoomed out near the top or bottom of the map, the area beyond the map edge shows the gray fill (upstream behavior, the camera is only clamped when it moves).
- Answered: porting to Godot 3D was advised against (rewrite of the whole codebase; 3D samples the same pixel art and faces the same filtering choice). The user has not decided on a 3D look as a design goal.
- 2026-09-27 (widescreen, stage one): Widescreen checkbox in Settings (off by default, persisted). Black bars for every screen, wide layout for the plain world map only. Windowed sizes in widescreen are 16:9 at the same heights as the 16:10 list (1422 x 800 and so on). Upstream files touched, all small: `main.go`, `game/game.go` (4 sites), `ui/ui.go` (cursor hook), `inputmanager/input.go`, `mouse/mouse.go`, `cartographer/cartographer.go`, `camera/camera.go` (tile culling width).
- Known gaps of stage one: minimap view brackets still show the 12 tile view; the full screen click catchers of upstream dialogs are 320 wide; touch input untested; anything that pushes a drawer over the map snaps to the narrow picture with bars.
- 2026-09-27 (settings): ONE settings screen for the whole game, on upstream's panel art. `MakeOptionsUI` in `game/magic/settings/options.go` calls upstream's untouched `MakeSettingsUI` (`settings.go`) and adds the display controls in the free right column: a Resolution box that opens a list, and a Fullscreen checkbox (unchecked, meaning windowed, by default). The earlier darkened text-row design was rejected by the user in favor of the existing Settings visuals; do not bring it back. The screen closes with the panel's Ok button; Escape also closes it from the main menu.
- Consumers of `MakeOptionsUI` (KEEP IN STEP): main menu Settings row (`mainview/options.go`), the load screen's settings button (`mainview/view.go` SettingsUI), the in-game menu's settings button (`game/game.go` SettingsUI).
- Main menu rows: Quick Start, Continue, Load, New Game, Settings, Credits, Quit. "Quit" keeps the original "Quit to Dos" help entry on right click.
- Window preferences live in `game/magic/display` and persist to `display.json` in the working directory (`_build/` when run through run.bat). "Resolution" means window size in whole multiples of 320x200; the game always renders at 960x600 internally and Ebiten stretches it. Fullscreen is borderless at desktop resolution. Capture probe flags: `-options`, `-rollout`, `-credits`.
- Glossary: the user says "Settings", not "Options". "Rollout" is a list that opens from a control.
- Trap: `sed -i` in Git Bash rewrites a CRLF file as LF. Use a Python script file for edits to upstream files.
- Persistence today: display choices in `display.json`; rule toggles and key bindings inside save files (upstream); music volume and music on/off are session only.
- 2026-09-27 (later): first quality of life pass on the main menu. Intro animation skipped at startup (`PlayIntroOnStartup` in `game/magic/main.go`). Main menu gained Options (no-op) and Credits rows; credits scroll only after Credits is pressed, one pass, press again to stop (tunables at the top of `game/magic/mainview/view.go`). New dev probe `test/main-screen-capture` renders the menu to a png: `_build/capture.exe -data <path> -out x.png -frames N [-credits]`.
- Trap: the working tree is CRLF (git autocrlf=true). Scripted patches must match CRLF anchors. Never build Python with sed or an unquoted heredoc.
- 2026-09-27: cloned upstream at e824e98, toolchain installed, build passes, test suite passes, boot probe loads the Steam data and starts a game. No modifications yet. The design of the modification has not been discussed.
