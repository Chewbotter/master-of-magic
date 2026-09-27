-- Writes the figures of an Aseprite file made by build-files.lua into the replacement folder of
-- the game (game/magic/mod), one png per frame, under the names the game looks for.
--
-- Only what was changed is written: a frame that is the same as the game's own is left out, so the
-- replacement folder is the list of what was changed. A frame the game does not have (4 to 8) is
-- written when there is anything in it.
--
-- Nothing is ever deleted. A file of the replacement folder whose frame is the game's own again is
-- named at the end, to be taken out by hand.
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

local written = 0
local same = 0
local stale = {}

-- the picture of a layer at a frame, as large as the picture of a figure. nil if there is nothing
local function framePicture(sprite, layer, frame)
  local cel = layer:cel(frame)
  if not cel then
    return nil
  end

  local picture = Image(WIDTH, HEIGHT, ColorMode.INDEXED)
  picture:clear(0)

  -- pixel by pixel: the numbers of the colors have to stay what they are
  local from = cel.image
  local clear = from.spec.transparentColor
  for y = 0, from.height - 1 do
    for x = 0, from.width - 1 do
      local toX = x + cel.position.x
      local toY = y + cel.position.y
      local value = from:getPixel(x, y)
      if value ~= clear and toX >= 0 and toY >= 0 and toX < WIDTH and toY < HEIGHT then
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
  for _, layer in ipairs(group.layers) do
    if layer.isImage and FACINGS[layer.name] then
      for frameNumber = 1, #sprite.frames do
        local name = layer.name .. "_" .. (frameNumber - 1) .. ".png"
        local original = app.fs.joinPath(spritesFolder, "units", group.name, name)
        local target = app.fs.joinPath(modFolder, "units", group.name, name)

        local picture = framePicture(sprite, layer, frameNumber)
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
        else
          if picture then
            same = same + 1
          end
          if app.fs.isFile(target) then
            table.insert(stale, target)
          end
        end
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
if #stale > 0 then
  report = report .. string.format("\n%d files of the replacement folder are the game's own again. take them out by hand:", #stale)
  for _, path in ipairs(stale) do
    report = report .. "\n  " .. path
  end
end

print(report)
if app.isUIAvailable then
  app.alert{ title = "Export to the game", text = report }
end
