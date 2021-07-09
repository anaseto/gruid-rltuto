// This file handles game related affairs that are not specific to entities or
// the map.

package main

import (
	"github.com/anaseto/gruid"
	"github.com/anaseto/gruid/paths"
)

// game represents information relevant the current game's state.
type game struct {
	ECS *ECS             // entities present on the map
	Map *Map             // the game map, made of tiles
	PR  *paths.PathRange // path range for the map
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
			m.Char = 'o'
		default:
			m.Char = 'T'
		}
		p := g.FreeFloorTile()
		i := g.ECS.AddEntity(m, p)
		switch m.Char {
		case 'o':
			g.ECS.Fighter[i] = &fighter{
				HP: 10, MaxHP: 10, Defense: 0, Power: 3,
			}
			g.ECS.Name[i] = "orc"
		case 'T':
			g.ECS.Fighter[i] = &fighter{
				HP: 16, MaxHP: 16, Defense: 1, Power: 4,
			}
			g.ECS.Name[i] = "troll"
		}
		g.ECS.AI[i] = &AI{}
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

// EndTurn is called when the player's turn ends. Currently, the player and
// monsters have all the same speed, so we make each monster act each time the
// player's does an action that ends a turn.
func (g *game) EndTurn() {
	g.UpdateFOV()
	for i, e := range g.ECS.Entities {
		if g.ECS.PlayerDied() {
			return
		}
		switch e.(type) {
		case *Monster:
			g.HandleMonsterTurn(i)
		}
	}
}

// UpdateFOV updates the field of view.
func (g *game) UpdateFOV() {
	player := g.ECS.Player()
	// player position
	pp := g.ECS.Positions[g.ECS.PlayerID]
	// We shift the FOV's Range so that it will be centered on the new
	// player's position.
	rg := gruid.NewRange(-maxLOS, -maxLOS, maxLOS+1, maxLOS+1)
	player.FOV.SetRange(rg.Add(pp).Intersect(g.Map.Grid.Range()))
	// We mark cells in field of view as explored. We use the symmetric
	// shadow casting algorithm provided by the rl package.
	passable := func(p gruid.Point) bool {
		return g.Map.Grid.At(p) != Wall
	}
	for _, p := range player.FOV.SSCVisionMap(pp, maxLOS, passable, false) {
		if paths.DistanceManhattan(p, pp) > maxLOS {
			continue
		}
		if !g.Map.Explored[p] {
			g.Map.Explored[p] = true
		}
	}
}

// InFOV returns true if p is in the player's field of view. We only keep cells
// within maxLOS manhattan distance from the player, as natural given our
// current 4-way movement. With 8-way movement, the natural distance choice
// would be the Chebyshev one.
func (g *game) InFOV(p gruid.Point) bool {
	pp := g.ECS.Positions[g.ECS.PlayerID]
	return g.ECS.Player().FOV.Visible(p) &&
		paths.DistanceManhattan(pp, p) <= maxLOS
}
