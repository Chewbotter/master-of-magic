-- Takes the tags out of the Aseprite files of the figures, and leaves everything else as it is.
-- The user found tags in the way: with so few frames they make the timeline hard to scrub.
--
--   aseprite -b --script-param files=<folder of the .aseprite files> --script remove-tags.lua

local filesFolder = app.params["files"] or "D:/Work/MasterMagic_open/_sprites/aseprite"

local names = app.fs.listFiles(filesFolder)
table.sort(names)

for _, name in ipairs(names) do
  if name:lower():match("%.aseprite$") then
    local path = app.fs.joinPath(filesFolder, name)
    local sprite = Sprite{ fromFile = path }
    if sprite then
      local count = #sprite.tags
      while #sprite.tags > 0 do
        sprite:deleteTag(sprite.tags[1])
      end

      local layers = 0
      local cels = #sprite.cels
      for _, group in ipairs(sprite.layers) do
        if group.isGroup then
          layers = layers + #group.layers
        end
      end

      if count > 0 then
        sprite:saveAs(path)
      end
      print(string.format("%s: %d tags taken out, %d are left. %d units, %d layers, %d pictures, %d frames, %d slices", name, count, #sprite.tags, #sprite.layers, layers, cels, #sprite.frames, #sprite.slices))
      sprite:close()
    end
  end
end
