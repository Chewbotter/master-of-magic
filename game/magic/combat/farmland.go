package combat

// Farmland (user, 2026-09-29): a battle one or two tiles of the world map from a town is fought
// on its fields. Not in the original, where only the battle of the town itself shows the town.
//
// In his words: "Both one and two tiles from the town center should be farmland. The only
// difference is that adjacent tiles should generate more buildings. Both should scatter individual
// buildings throughout the area. Roads should only appear if there are actual roads. Farmland
// doesn't have walls. The terrain should be a grassland base, with square plots of repeating crop
// land tiles. 1 row of grass tiles separates the plots so they look like farmed fields."
//
// WHERE. On land whose landscape is grass, with no town, lair, node or tower on the tile of the
// battle itself. The ground is plain grass land then, whatever biome the tile has: fields have
// taken the place of the forest. Desert, mountains and tundra beside a town stay what they are.
//
// THE FIELDS. The ground is laid out in squares of one size for the battle, farmPlotMin to
// farmPlotMax cells, with one row of grass between them. A square is a plot of crops or is left
// as grass, by chance. A plot shows one picture of crops in all of its cells, so it looks
// ploughed one way. No plot lies on a road, a coast or a river: a square they run through has a
// smaller plot beside them, or none. The ground is flat.
//
// BUILDINGS. Houses of the race of the town stand one by one on the grass between and beside the
// plots, not where the armies start. They are for the look, as the houses of a town are.
//
// RULES. None of its own: crops cost what grass costs.
//
// PICTURES, see mod/farmland.go: the folder Farmland of the replacement folder, crop 1 to 4 in
// place of the game's, crop 5 on added. The game makes its own from the grass of the landscape.

import (
    "image"
    "math/rand/v2"

    "github.com/kazzmir/master-of-magic/game/magic/mod"

    "github.com/hajimehoshi/ebiten/v2"
)

// turns it off
var Farmland = true

// how far from the town the battle is, in tiles of the world map
const (
    FarmlandNone = 0
    FarmlandNear = 1
    FarmlandFar = 2
)

// the cells of a side of a plot
const farmPlotMin = 3
const farmPlotMax = 5
// a plot beside a road, a coast or a river can be as small as this
const farmPlotSmall = 2
// of the squares this share is plots
const farmPlotShare = 0.6

// houses on the original's screen, beside the town and two tiles from it. beyond the screen as
// many for its size
const farmHousesNear = 7
const farmHousesFar = 3

// the pictures of crops as the ground has them, in place of the numbers of the archive of the
// landscape, and how many the game makes
const groundCropFirst = 80
const groundCropCount = 4

const farmSet = mod.FeatureFarmland
const farmCropName = "crop"

// the ground of a battle as it is fought: on farmland the biome of the tile gives way to the
// fields, and what is no farmland has none
func farmGround(landscape CombatLandscape, zone ZoneType) ZoneGround {
    ground := zone.Ground
    if ground.Farmland == FarmlandNone {
        return ground
    }

    if !Farmland || zone.City != nil || zone.Encounter != ZoneNone || BiomeLandscape(ground.Biome, landscape) != CombatLandscapeGrass {
        ground.Farmland = FarmlandNone
        return ground
    }

    ground.SetBiome("")
    return ground
}

// lays the fields over the ground, after the coast and the river
func (ground *BattleGround) makeFarmland(zone ZoneType) {
    if zone.Ground.Farmland == FarmlandNone {
        return
    }
    ground.Farmland = zone.Ground.Farmland

    cells := ground.Width * ground.Height
    ground.Crops = make([]int, cells)
    ground.Built = make([]bool, cells)

    // fields are flat
    for index := range ground.Heights {
        ground.Heights[index] = 0
    }

    free := func(cgx int, cgy int) bool {
        if !ground.contains(cgx, cgy) {
            return false
        }
        index := ground.index(cgx, cgy)
        group := ground.Group[index]
        return group != TerrainSand && group != TerrainWater && ground.Roads[index] == 0 && ground.coastAt(cgx, cgy) == coastLand && ground.riverAt(cgx, cgy) == riverLand
    }

    size := farmPlotMin + rand.N(farmPlotMax - farmPlotMin + 1)
    step := size + 1
    pictures := groundCropCount + ground.Extras[groundCropFirst]

    // the squares, from a corner by chance
    startX := ground.MinX - rand.N(step)
    startY := ground.MinY - rand.N(step)
    for cornerY := startY; cornerY < ground.MinY + ground.Height; cornerY += step {
        for cornerX := startX; cornerX < ground.MinX + ground.Width; cornerX += step {
            if rand.Float64() >= farmPlotShare {
                continue
            }

            // the largest plot the square has room for: a road, the coast or a river in it
            // leave a smaller one beside them
            plotX, plotY, plot := ground.largestPlot(cornerX, cornerY, size, free)
            if plot < farmPlotSmall {
                continue
            }

            picture := cropPicture(rand.N(pictures))
            for dy := range plot {
                for dx := range plot {
                    if !ground.contains(plotX + dx, plotY + dy) {
                        continue
                    }
                    index := ground.index(plotX + dx, plotY + dy)
                    ground.Group[index] = TerrainCrop
                    ground.Crops[index] = picture
                }
            }
        }
    }

    // the grass beside a plot is grass: dirt and mounds would end at the plot without an edge
    for cgy := ground.MinY; cgy < ground.MinY + ground.Height; cgy++ {
        for cgx := ground.MinX; cgx < ground.MinX + ground.Width; cgx++ {
            group := ground.GroupAt(cgx, cgy)
            if group != TerrainDirt && group != TerrainRough {
                continue
            }
            for dy := -1; dy <= 1; dy++ {
                for dx := -1; dx <= 1; dx++ {
                    if ground.contains(cgx + dx, cgy + dy) && ground.GroupAt(cgx + dx, cgy + dy) == TerrainCrop {
                        ground.setGroup(cgx, cgy, TerrainGrass)
                    }
                }
            }
        }
    }
}

// the largest square of free cells within a square of the ground, and its corner. the places
// it is looked for at are tried in an order by chance. a cell beyond the ground counts as free: a
// plot at the edge of the ground is cut off there
func (ground *BattleGround) largestPlot(cornerX int, cornerY int, size int, free func(int, int) bool) (int, int, int) {
    fits := func(x int, y int, plot int) bool {
        for dy := range plot {
            for dx := range plot {
                if ground.contains(x + dx, y + dy) && !free(x + dx, y + dy) {
                    return false
                }
            }
        }
        return true
    }

    for plot := size; plot >= farmPlotSmall; plot-- {
        room := size - plot + 1
        for _, place := range rand.Perm(room * room) {
            x := cornerX + place % room
            y := cornerY + place / room
            if fits(x, y, plot) {
                return x, y, plot
            }
        }
    }
    return 0, 0, 0
}

// the number the ground has for a picture of crops, by its number from 0 among the game's and the
// added ones
func cropPicture(number int) int {
    if number < groundCropCount {
        return groundCropFirst + number
    }
    return groundExtraFirst + groundCropFirst * groundExtraStep + number - groundCropCount
}

func (ground *BattleGround) cropAt(cgx int, cgy int) int {
    if ground.Crops == nil || !ground.contains(cgx, cgy) {
        return groundGrassFirst
    }
    return ground.Crops[ground.index(cgx, cgy)]
}

// true if a house of the farmland stands in a cell
func (ground *BattleGround) builtAt(cgx int, cgy int) bool {
    return ground != nil && ground.Built != nil && ground.contains(cgx, cgy) && ground.Built[ground.index(cgx, cgy)]
}

// the cells the armies start in, of the largest armies
func armyStarts(cgx int, cgy int) bool {
    firstX, lastX, firstY, lastY := armyPlaces(true)
    if cgy < firstY || cgy > lastY {
        return false
    }
    return cgx >= firstX && cgx <= deployDefenderRows[0] || cgx >= deployAttackerRows[0] && cgx <= lastX
}

// houses of one area, one to a cell, on the grass
func scatterFarmHouses(count int, pool sceneryPool, zone ZoneType, area sceneryArea, ground *BattleGround) []SceneryPiece {
    var out []SceneryPiece

    tries := 0
    for len(out) < count && tries < sceneryMaxTries {
        cgx, cgy := area.randomCell()
        screenX, screenY := cellScreen(cgx, cgy, 0, 0)

        if armyStarts(cgx, cgy) || ground.builtAt(cgx, cgy) || !ground.sceneryAllowed(cgx, cgy) || !area.Accept(screenX, screenY) {
            tries += 1
            continue
        }

        ground.Built[ground.index(cgx, cgy)] = true
        out = append(out, pool.piece(SceneryHouse, screenX, screenY + houseShiftY))
    }

    return out
}

// the houses of the farmland of a battlefield. before its trees and rocks, which keep off them
func makeFarmHouses(width int, height int, zone ZoneType, ground *BattleGround) []SceneryPiece {
    var out []SceneryPiece
    if ground == nil || ground.Farmland == FarmlandNone {
        return out
    }

    count := farmHousesFar
    if ground.Farmland == FarmlandNear {
        count = farmHousesNear
    }

    style := houseStyle(zone.Ground.FarmRace)
    pool := makeSceneryPool(houseSet, houseNames[style], houseLbx, houseIndex + style * housePictures, housePictures)

    out = append(out, scatterFarmHouses(count, pool, zone, originalArea(), ground)...)
    out = append(out, scatterFarmHouses(count * sceneryBeyondScreen(width, height), pool, zone, beyondArea(width, height), ground)...)
    return out
}

// the frames of a picture of crops, by its number from 0: of the folder of the farmland, or made
// of the grass of the landscape
func (combat *CombatScreen) cropPictures(lbx string, number int) []*ebiten.Image {
    pictures := combat.addedPictures(combat.Model.Ground.FarmSet, farmCropName, number + 1)
    if len(pictures) > 0 {
        return pictures
    }
    if number >= groundCropCount {
        number = 0
    }

    if made, ok := combat.cropsMade[number]; ok {
        return made
    }

    var made []*ebiten.Image
    grass, _ := combat.ImageCache.GetImages(lbx, groundGrassFirst + number % groundGrassCount)
    for _, frame := range grass {
        bounds := frame.Bounds()
        pixels := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
        frame.ReadPixels(pixels.Pix)
        made = append(made, ebiten.NewImageFromImage(mod.CropTile(pixels)))
    }

    if combat.cropsMade == nil {
        combat.cropsMade = make(map[int][]*ebiten.Image)
    }
    combat.cropsMade[number] = made
    return made
}
