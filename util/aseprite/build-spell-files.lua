-- Makes the Aseprite files of the spells out of the pictures util/export-sprites wrote into
-- spells/<name>/<frame>.png.
--
-- The spells come in a few sizes of picture, and a file is as large as its pictures: one file for
-- each size, "Spells <width>x<height>.aseprite". In a file every spell is a layer, so a row of the
-- timeline is a spell and a column is a frame. A file has as many frames as its longest spell;
-- the frames after the last of a shorter spell are empty.
--
-- A file that is already there is left as it is, so the changes in it are kept. To make it again
-- from the pictures of the game, pass overwrite=yes, which throws away what was changed in it.
--
-- The files hold the game's own pictures. They are written OUTSIDE of the repository and are not
-- to be passed on.
--
--   aseprite -b --script-param sprites=<folder of the export> --script build-spell-files.lua
--
-- The code is ours. It runs in stock Aseprite as well as in the fork.

local spritesFolder = app.params["sprites"] or "D:/Work/MasterMagic_open/_sprites"
local outFolder = app.params["out"] or app.fs.joinPath(spritesFolder, "aseprite")
local overwrite = app.params["overwrite"] == "yes"

-- seconds a frame shows: a spell steps every second redraw of 18.2 a second
local FRAME_TIME = 0.11

-- the name of the file of a frame, as the game reads it: 00.png, 01.png
local function frameFile(frame)
  return string.format("%02d.png", frame)
end

-- the spells by the size of their pictures, in the order of their names
local function spellsBySize()
  local spellsFolder = app.fs.joinPath(spritesFolder, "spells")
  local names = app.fs.listFiles(spellsFolder)
  table.sort(names)

  local sizes = {}
  local order = {}
  for _, name in ipairs(names) do
    local folder = app.fs.joinPath(spellsFolder, name)
    local first = app.fs.joinPath(folder, frameFile(0))
    if app.fs.isDirectory(folder) and app.fs.isFile(first) then
      local picture = Image{ fromFile = first }
      if picture then
        local frames = 0
        while app.fs.isFile(app.fs.joinPath(folder, frameFile(frames))) do
          frames = frames + 1
        end

        local size = picture.width .. "x" .. picture.height
        if not sizes[size] then
          sizes[size] = { width = picture.width, height = picture.height, spells = {}, frames = 0 }
          table.insert(order, size)
        end
        table.insert(sizes[size].spells, { name = name, frames = frames })
        sizes[size].frames = math.max(sizes[size].frames, frames)
      end
    end
  end

  table.sort(order)
  return sizes, order
end

local function makeFile(size, group, palette)
  local path = app.fs.joinPath(outFolder, "Spells " .. size .. ".aseprite")
  if app.fs.isFile(path) and not overwrite then
    print(path .. " is there already and is left as it is")
    return
  end

  local sprite = Sprite(group.width, group.height, ColorMode.INDEXED)
  sprite:setPalette(palette)
  sprite.transparentColor = 0

  for _ = 2, group.frames do
    sprite:newEmptyFrame()
  end
  for _, frame in ipairs(sprite.frames) do
    frame.duration = FRAME_TIME
  end

  local firstLayer = sprite.layers[1]
  local pictures = 0

  -- a new layer goes on top: last spell first, so the first spell is the top row
  for index = #group.spells, 1, -1 do
    local spell = group.spells[index]
    local folder = app.fs.joinPath(spritesFolder, "spells", spell.name)

    local layer = sprite:newLayer()
    layer.name = spell.name
    -- all spells are in the same place: one shows at a time
    layer.isVisible = (index == 1)

    for frame = 0, spell.frames - 1 do
      local picture = Image{ fromFile = app.fs.joinPath(folder, frameFile(frame)) }
      if picture then
        sprite:newCel(layer, frame + 1, picture, Point(0, 0))
        pictures = pictures + 1
      end
    end
  end

  sprite:deleteLayer(firstLayer)

  sprite:saveAs(path)
  sprite:close()

  print(string.format("%s: %d spells, %d pictures, %d frames", path, #group.spells, pictures, group.frames))
end

local palette = Palette{ fromFile = app.fs.joinPath(spritesFolder, "palette.gpl") }
if not palette then
  print("no palette at " .. app.fs.joinPath(spritesFolder, "palette.gpl"))
  return
end

app.fs.makeAllDirectories(outFolder)

local sizes, order = spellsBySize()
for _, size in ipairs(order) do
  makeFile(size, sizes[size], palette)
end
