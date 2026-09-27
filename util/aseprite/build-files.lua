-- Makes one Aseprite file per race out of the pictures util/export-sprites wrote.
--
-- In a file every unit is a folder (a group of layers) and every facing is a layer in it, so a row
-- of the timeline is a facing and a column is a frame. The frames 0 to 3 are the game's, 4 to 8 are
-- empty: the places of the longer strike and of the death frames (see game/magic/mod).
--
-- The feet of a figure are marked by a slice named "feet": View > Show > Slices shows and hides it.
-- It is not part of the pictures.
--
-- The files hold the game's own pictures. They are written OUTSIDE of the repository and are not
-- to be passed on.
--
--   aseprite -b --script-param sprites=<folder of the export> --script build-files.lua
--
-- The code is ours. It runs in stock Aseprite as well as in the fork.

local spritesFolder = app.params["sprites"] or "D:/Work/MasterMagic_open/_sprites"
local outFolder = app.params["out"] or app.fs.joinPath(spritesFolder, "aseprite")

local FACINGS = { "up", "upright", "right", "downright", "down", "downleft", "left", "upleft" }
-- the frames of the game, and the frames a figure can have with the new ones
local GAME_FRAMES = 4
local ALL_FRAMES = 9
-- the size of the picture of a figure, and where its feet are in it
local WIDTH = 28
local HEIGHT = 30
local FEET_X = 13
local FEET_Y = 23
-- seconds a frame shows: the strike of the game steps every second redraw of 18.2 a second
local FRAME_TIME = 0.11
local FEET_COLOR = Color{ r = 255, g = 0, b = 255, a = 255 }

-- names and frames of the tags, frames counted from 1 as Aseprite does
local TAGS = {
  { name = "walk", from = 1, to = 3 },
  { name = "strike", from = 4, to = 4 },
  { name = "long strike", from = 4, to = 7 },
  { name = "dying", from = 8, to = 8 },
  { name = "dead", from = 9, to = 9 },
}

local function readRace(unitFolder)
  local file = io.open(app.fs.joinPath(unitFolder, "_source.txt"), "r")
  if not file then
    return nil
  end
  local race = nil
  for line in file:lines() do
    local found = line:match("^race: (.+)$")
    if found then
      race = found
    end
  end
  file:close()
  return race
end

-- the folders of the units by race, in the order of their names
local function unitsByRace()
  local unitsFolder = app.fs.joinPath(spritesFolder, "units")
  local names = app.fs.listFiles(unitsFolder)
  table.sort(names)

  local races = {}
  local order = {}
  for _, name in ipairs(names) do
    local folder = app.fs.joinPath(unitsFolder, name)
    if app.fs.isDirectory(folder) then
      local race = readRace(folder)
      if race then
        if not races[race] then
          races[race] = {}
          table.insert(order, race)
        end
        table.insert(races[race], name)
      end
    end
  end

  table.sort(order)
  return races, order
end

local function makeFile(race, units, palette)
  local sprite = Sprite(WIDTH, HEIGHT, ColorMode.INDEXED)
  sprite:setPalette(palette)
  sprite.transparentColor = 0

  for _ = 2, ALL_FRAMES do
    sprite:newEmptyFrame()
  end
  for _, frame in ipairs(sprite.frames) do
    frame.duration = FRAME_TIME
  end

  for _, tag in ipairs(TAGS) do
    local made = sprite:newTag(tag.from, tag.to)
    made.name = tag.name
  end

  local firstLayer = sprite.layers[1]
  local pictures = 0

  -- a new layer goes on top: last unit first, so the first unit is the top row
  for index = #units, 1, -1 do
    local name = units[index]
    local unitFolder = app.fs.joinPath(spritesFolder, "units", name)

    local group = sprite:newGroup()
    group.name = name
    group.isCollapsed = true
    -- all units are in the same place: one shows at a time
    group.isVisible = (index == 1)

    for facing = #FACINGS, 1, -1 do
      local layer = sprite:newLayer()
      layer.name = FACINGS[facing]
      layer.parent = group

      for frame = 0, GAME_FRAMES - 1 do
        local path = app.fs.joinPath(unitFolder, FACINGS[facing] .. "_" .. frame .. ".png")
        if app.fs.isFile(path) then
          local picture = Image{ fromFile = path }
          if picture then
            sprite:newCel(layer, frame + 1, picture, Point(0, 0))
            pictures = pictures + 1
          end
        end
      end
    end
  end

  sprite:deleteLayer(firstLayer)

  -- the feet. a slice is drawn over the picture and is not part of it
  local feet = sprite:newSlice(Rectangle(FEET_X, FEET_Y, 1, 1))
  feet.name = "feet"
  feet.color = FEET_COLOR

  local path = app.fs.joinPath(outFolder, race .. ".aseprite")
  sprite:saveAs(path)
  sprite:close()

  print(string.format("%s: %d units, %d pictures", path, #units, pictures))
end

local palette = Palette{ fromFile = app.fs.joinPath(spritesFolder, "palette.gpl") }
if not palette then
  print("no palette at " .. app.fs.joinPath(spritesFolder, "palette.gpl"))
  return
end

app.fs.makeAllDirectories(outFolder)

local races, order = unitsByRace()
for _, race in ipairs(order) do
  makeFile(race, races[race], palette)
end
