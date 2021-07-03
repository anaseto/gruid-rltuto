// This file handles game related affairs that are not specific to entities or
// the map.

package main

import "github.com/anaseto/gruid"

// game represents information relevant the current game's state.
type game struct {
	ECS *ECS // entities present on the map
	Map *Map // the game map, made of tiles
}

// SpawnMonsters adds some monsters in the current map.
func (g *game) SpawnMonsters() {
	const numberOfMonsters = 6
	for i := 0; i < numberOfMonsters; i++ {
		m := &Monster{}
		// We generate either an orc or a troll with 0.8 and 0.2
		// probabilities respectively.
		switch {
		case g.Map.Rand.Intn(100) < 80:
			m.Name = "orc"
			m.Char = 'o'
		default:
			m.Name = "troll"
			m.Char = 'T'
		}
		p := g.FreeFloorTile()
		g.ECS.AddEntity(m, p)
	}
}

// FreeFloorTile returns a free floor tile in the map (it assumes it exists).
func (g *game) FreeFloorTile() gruid.Point {
	for {
		p := g.Map.RandomFloor()
		if g.ECS.NoBlockingEntityAt(p) {
			return p
		}
	}
}
