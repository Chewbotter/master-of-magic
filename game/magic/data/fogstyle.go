package data

// Fog of the world map as in the original: a tile is unexplored (black) or explored (shown as it is).
// Tiles that were explored and are not seen by a unit or city at the moment are NOT darkened, on the
// map and on the minimap. What is seen at the moment is still kept (FogTypeVisible): units of others
// are only shown on tiles that are seen.
// false: explored tiles that are not seen are darkened, and the start of a game reveals a larger area
var ClassicFog = true

// how far the tiles around the first city are revealed at the start of a game
const ClassicStartingSight = 2

// true for the 4 corner tiles of a square that is revealed, which stay as they are: the original shows
// only a quarter of them, so what is revealed is round (ReMoM Set_Map_Square_Explored_Flags_XYP,
// Init_Square_Explored). a square of 3 by 3 tiles keeps its corners
func ClassicFogCorner(dx int, dy int, squares int) bool {
    if !ClassicFog || squares < 2 {
        return false
    }

    return max(dx, -dx) == squares && max(dy, -dy) == squares
}
