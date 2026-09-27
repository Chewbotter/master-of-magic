package combat

// Projectiles keep their positions in art pixels of the field as the original's screen shows it,
// not of the screen as the camera shows it. They are moved to where the camera shows them when they
// are drawn, so moving or zooming the camera while one flies keeps it on its way over the field.

import (
    "github.com/hajimehoshi/ebiten/v2"
)

// the size of the field projectiles are made for
const projectileScale = 1.0

// maps a tile to the middle of its diamond in the positions projectiles use
func projectileMatrix() ebiten.GeoM {
    return MakeBattlefieldMatrix()
}
