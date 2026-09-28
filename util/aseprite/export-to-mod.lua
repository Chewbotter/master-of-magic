-- Writes the figures of an Aseprite file made by build-files.lua, and the spells of one made by
-- build-spell-files.lua, into the replacement folder of the game (game/magic/mod), one png per
-- frame, under the names the game looks for.
--
-- A folder (group of layers) of a file is a unit, a layer outside of a folder that has the name of
-- a spell of _sprites/spells is that spell.
--
-- Only what was changed is written: a frame that is the same as the game's own is left out, so the
-- replacement folder is the list of what was changed. A frame the game does not have (4 to 8 of a
-- figure, one after the last of a spell) is written when there is anything in it.
--
-- THE REPLACEMENT FOLDER MIRRORS THE FILES (user, 2026-09-27). For every unit or spell of a file
-- that is exported, its pictures in the replacement folder are the frames that are changed in the
-- file and no others. A picture of such a unit or spell that is not one of them, because its frame
-- was put back or emptied, or because it was put there by hand, is taken out. It is not deleted:
-- it is moved to _removed in the replacement folder, where the last one taken out of each name is
-- kept. Units and spells that are in no exported file, and everything under archives, are left
-- alone.
--
-- In Aseprite: File > Scripts, with the file of a race open.
-- Without a window, for every file of a folder:
--   aseprite -b --script-param files=<folder of the .aseprite files> --script export-to-mod.lua

local spritesFolder = app.params["sprites"] or "D:/Work/MasterMagic_open/_sprites"
local modFolder = app.params["mod"] or "D:/Work/MasterMagic_open/mod"
local filesFolder = app.params["files"]

local FACINGS = { up = true, upright = true, right = true, downright = true, down = true, downleft = true, left = true, upleft = true }
local WIDTH = 28
local HEIGHT = 30

local REMOVED = "_removed"
-- frames are looked for up to this number, as the game does
local MAX_FRAMES = 16

local written = 0
local same = 0
local removed = {}

-- moves a picture of the replacement folder that is no changed frame of the file out of the way.
-- kind is "units" or "spells"
local function takeOut(kind, owner, name)
  local from = app.fs.joinPath(modFolder, kind, owner, name)
  local folder = app.fs.joinPath(modFolder, REMOVED, kind, owner)
  local to = app.fs.joinPath(folder, name)

  app.fs.makeAllDirectories(folder)
  os.remove(to)
  local ok = os.rename(from, to)
  if ok then
    table.insert(removed, from)
  else
    table.insert(removed, from .. "  (COULD NOT BE MOVED)")
  end
end

-- the picture of a layer at a frame, width by height from the upper left corner of the file. nil
-- if there is nothing
local function framePicture(layer, frame, width, height)
  local cel = layer:cel(frame)
  if not cel then
    return nil
  end

  local picture = Image(width, height, ColorMode.INDEXED)
  picture:clear(0)

  -- pixel by pixel: the numbers of the colors have to stay what they are
  local from = cel.image
  local clear = from.spec.transparentColor
  for y = 0, from.height - 1 do
    for x = 0, from.width - 1 do
      local toX = x + cel.position.x
      local toY = y + cel.position.y
      local value = from:getPixel(x, y)
      if value ~= clear and toX >= 0 and toY >= 0 and toX < width and toY < height then
        picture:drawPixel(toX, toY, value)
      end
    end
  end

  if picture:isEmpty() then
    return nil
  end
  return picture
end

-- true if both pictures have the same color, by its number, at every place. see-through is the
-- same whatever the number of its color
local function samePixels(picture, own)
  if not own or own.width ~= picture.width or own.height ~= picture.height then
    return false
  end

  local clear = own.spec.transparentColor
  for y = 0, picture.height - 1 do
    for x = 0, picture.width - 1 do
      local mine = picture:getPixel(x, y)
      local theirs = own:getPixel(x, y)
      if own.colorMode ~= ColorMode.INDEXED then
        return false
      end
      if theirs == clear then
        theirs = 0
      end
      if mine ~= theirs then
        return false
      end
    end
  end

  return true
end

local function exportUnit(sprite, group)
  -- the names of the pictures the unit has in the replacement folder after this
  local kept = {}

  for _, layer in ipairs(group.layers) do
    if layer.isImage and FACINGS[layer.name] then
      for frameNumber = 1, #sprite.frames do
        local name = layer.name .. "_" .. (frameNumber - 1) .. ".png"
        local original = app.fs.joinPath(spritesFolder, "units", group.name, name)
        local target = app.fs.joinPath(modFolder, "units", group.name, name)

        local picture = framePicture(layer, frameNumber, WIDTH, HEIGHT)
        local changed = false

        if picture then
          if app.fs.isFile(original) then
            local own = Image{ fromFile = original }
            changed = not samePixels(picture, own)
          else
            changed = true
          end
        end

        if changed then
          app.fs.makeAllDirectories(app.fs.joinPath(modFolder, "units", group.name))
          picture:saveAs{ filename = target, palette = sprite.palettes[1] }
          written = written + 1
          kept[name] = true
        elseif picture then
          same = same + 1
        end
      end
    end
  end

  -- the mirror: what the unit has in the replacement folder and is no changed frame of the file
  for facing, _ in pairs(FACINGS) do
    for frame = 0, MAX_FRAMES - 1 do
      local name = facing .. "_" .. frame .. ".png"
      if not kept[name] and app.fs.isFile(app.fs.joinPath(modFolder, "units", group.name, name)) then
        takeOut("units", group.name, name)
      end
    end
  end
end

-- the name of the file of a frame of a spell, as the game reads it: 00.png, 01.png
local function spellFrameFile(frame)
  return string.format("%02d.png", frame)
end

-- the folder of a spell in _sprites/spells, nil if the layer is no spell
local function spellFolder(layer)
  local folder = app.fs.joinPath(spritesFolder, "spells", layer.name)
  if app.fs.isFile(app.fs.joinPath(folder, spellFrameFile(0))) then
    return folder
  end
  return nil
end

local function exportSpell(sprite, layer, folder)
  -- a spell keeps the size of its picture, whatever the size of the file
  local first = Image{ fromFile = app.fs.joinPath(folder, spellFrameFile(0)) }
  local target = app.fs.joinPath(modFolder, "spells", layer.name)
  local kept = {}

  for frameNumber = 1, #sprite.frames do
    local name = spellFrameFile(frameNumber - 1)
    local original = app.fs.joinPath(folder, name)

    local picture = framePicture(layer, frameNumber, first.width, first.height)
    local changed = false

    if picture then
      if app.fs.isFile(original) then
        changed = not samePixels(picture, Image{ fromFile = original })
      else
        changed = true
      end
    end

    if changed then
      app.fs.makeAllDirectories(target)
      picture:saveAs{ filename = app.fs.joinPath(target, name), palette = sprite.palettes[1] }
      written = written + 1
      kept[name] = true
    elseif picture then
      same = same + 1
    end
  end

  -- the mirror: what the spell has in the replacement folder and is no changed frame of the file
  if app.fs.isDirectory(target) then
    for _, name in ipairs(app.fs.listFiles(target)) do
      if name:match("^%d%d%.png$") and not kept[name] then
        takeOut("spells", layer.name, name)
      end
    end
  end
end

local function exportSprite(sprite)
  if sprite.colorMode ~= ColorMode.INDEXED then
    print(sprite.filename .. " is not in indexed color, it is left out")
    return
  end

  for _, layer in ipairs(sprite.layers) do
    if layer.isGroup then
      exportUnit(sprite, layer)
    elseif layer.isImage then
      local folder = spellFolder(layer)
      if folder then
        exportSpell(sprite, layer, folder)
      end
    end
  end
end

if filesFolder then
  local names = app.fs.listFiles(filesFolder)
  table.sort(names)
  for _, name in ipairs(names) do
    if name:lower():match("%.aseprite$") then
      local sprite = Sprite{ fromFile = app.fs.joinPath(filesFolder, name) }
      if sprite then
        exportSprite(sprite)
        sprite:close()
      end
    end
  end
else
  local sprite = app.sprite
  if not sprite then
    print("no file is open")
    return
  end
  exportSprite(sprite)
end

local report = string.format("%d pictures written to %s, %d are the game's own and left out", written, modFolder, same)
if #removed > 0 then
  table.sort(removed)
  report = report .. string.format("\n%d pictures are no changed frame any more and were moved to %s:", #removed, app.fs.joinPath(modFolder, REMOVED))
  for _, path in ipairs(removed) do
    report = report .. "\n  " .. path
  end
end

print(report)
if app.isUIAvailable then
  app.alert{ title = "Export to the game", text = report }
end
