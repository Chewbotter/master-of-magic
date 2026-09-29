package combat

// Biomes: kinds of a landscape with pictures of their own, see mod/biomes.go for the pictures. Here:
// which landscape a biome is a kind of, and what it changes of the ground.
//
// A biome keeps the rules of its landscape. Two were there before as the original has them and are
// biomes now: a forest has 31 to 60 trees, hills have 20 patches of raised ground.

import (
    "github.com/kazzmir/master-of-magic/game/magic/mod"
)

// the biome of the ground of a battle, and what goes with it
func (ground *ZoneGround) SetBiome(name string) {
    ground.Biome = name
    ground.Forest = name == mod.BiomeForest
    ground.Hills = name == mod.BiomeHills
}

// the landscape a biome is a kind of. the landscape itself without a biome
func BiomeLandscape(name string, landscape CombatLandscape) CombatLandscape {
    switch name {
        case mod.BiomeForest, mod.BiomeHills, mod.BiomeSwamp: return CombatLandscapeGrass
        case mod.BiomeVolcano, mod.BiomeSnowyMountain: return CombatLandscapeMountain
    }
    return landscape
}
