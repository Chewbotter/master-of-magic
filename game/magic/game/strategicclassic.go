package game

// The game's side of the original's quick resolution (combat/strategicclassic.go): what each side's
// wizard brings and what the square of the battle is (ReMoM MoM/src/Combat.c: Combat,
// Combat_Structure; MoM/src/CMBMAGIC.c: Combat_Casting_Cost_Multiplier). The code is ours.

import (
    "github.com/kazzmir/master-of-magic/game/magic/combat"
    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/terrain"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

// a side: the monsters of a lair and the neutral player are no wizard
func (game *Game) strategicSide(player *playerlib.Player, monsters bool, x int, y int, plane data.Plane) combat.StrategicSide {
    side := combat.StrategicSide{
        Human: player.IsHuman(),
        Neutral: monsters || player.IsNeutral(),
    }
    side.Wizard = !side.Neutral
    if !side.Wizard {
        return side
    }
    side.Mana = player.Mana
    side.Skill = player.ComputeCastingSkill()
    side.Books = make(map[data.MagicType]int)
    for _, realm := range []data.MagicType{data.NatureMagic, data.SorceryMagic, data.ChaosMagic, data.LifeMagic, data.DeathMagic} {
        side.Books[realm] = player.Wizard.MagicLevel(realm)
    }
    // the distance to the fortress: the larger of across (around the world) and down; the other plane
    // (or no fortress) counts as half the width of the world
    width := game.GetMap(plane).Width()
    distance := width / 2
    if fortress := player.FindFortressCity(); fortress != nil && fortress.Plane == plane {
        across := max(fortress.X - x, x - fortress.X)
        if across > width / 2 {
            across = width - across
        }
        distance = max(across, max(fortress.Y - y, y - fortress.Y))
    }
    side.CastMultiplier = combat.StrategicCastMultiplier(distance, player.Wizard.RetortEnabled(data.RetortChanneler))
    return side
}

// Combat_Structure: a node on the square, else a city being attacked, else the sea
func (game *Game) strategicStructure(x int, y int, plane data.Plane, zone combat.ZoneType) int {
    structure := combat.StructureNone
    tile := game.GetMap(plane).GetTile(x, y).Tile
    if tile.IsWater() {
        structure = combat.StructureOcean
    }
    if zone.City != nil {
        structure = combat.StructureCity
    }
    switch tile.TerrainType() {
        case terrain.SorceryNode: structure = combat.StructureSorceryNode
        case terrain.NatureNode: structure = combat.StructureNatureNode
        case terrain.ChaosNode: structure = combat.StructureChaosNode
    }
    return structure
}
