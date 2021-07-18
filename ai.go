// This file handles the base AI for monsters.

package main

import (
	"github.com/anaseto/gruid"
	"github.com/anaseto/gruid/paths"
)

// HandleMonsterTurn handles a monster's turn. The function assumes the entity
// with the given index is indeed a monster initialized with fighter and AI
// components.
func (g *game) HandleMonsterTurn(i int) {
	if !g.ECS.Alive(i) {
		// Do nothing if the entity corresponds to a dead monster.
		return
	}
	p := g.ECS.Positions[i]
	ai := g.ECS.AI[i]
	aip := &aiPath{g: g}
	pp := g.ECS.Positions[g.ECS.PlayerID]
	if paths.DistanceManhattan(p, pp) == 1 {
		// If the monster is adjacent to the player, attack.
		g.BumpAttack(i, g.ECS.PlayerID)
		return
	}
	if !g.InFOV(p) {
		// The monster is not in player's FOV.
		if len(ai.Path) < 1 {
			// Pick new path to a random floor tile.
			ai.Path = g.PR.AstarPath(aip, p, g.Map.RandomFloor())
		}
		g.AIMove(i)
		// NOTE: this base AI can be improved for example to avoid
		// monster's getting stuck between them. It's enough to get
		// started, though.
		return
	}
	// The monster is in player's FOV, so we compute a suitable path to
	// reach the player.
	ai.Path = g.PR.AstarPath(aip, p, pp)
	g.AIMove(i)
}

// AIMove moves a monster to the next position, if there is no blocking entity
// at the destination. It assumes the destination is walkable.
func (g *game) AIMove(i int) {
	ai := g.ECS.AI[i]
	if len(ai.Path) > 0 && ai.Path[0] == g.ECS.Positions[i] {
		ai.Path = ai.Path[1:]
	}
	if len(ai.Path) > 0 && g.ECS.NoBlockingEntityAt(ai.Path[0]) {
		// Only move if there is no blocking entity.
		g.ECS.MoveEntity(i, ai.Path[0])
		ai.Path = ai.Path[1:]
	}
}

// aiPath implements the paths.Astar interface for use in AI pathfinding.
type aiPath struct {
	g  *game
	nb paths.Neighbors
}

// Neighbors returns the list of walkable neighbors of q in the map using 4-way
// movement along cardinal directions.
func (aip *aiPath) Neighbors(q gruid.Point) []gruid.Point {
	return aip.nb.Cardinal(q,
		func(r gruid.Point) bool {
			return aip.g.Map.Walkable(r)
		})
}

// Cost implements paths.Astar.Cost.
func (aip *aiPath) Cost(p, q gruid.Point) int {
	if !aip.g.ECS.NoBlockingEntityAt(q) {
		// Extra cost for blocked positions: this encourages the
		// pathfinding algorithm to take another path to reach the
		// player.
		return 8
	}
	return 1
}

// Estimation implements paths.Astar.Estimation. For 4-way movement, we use the
// Manhattan distance.
func (aip *aiPath) Estimation(p, q gruid.Point) int {
	return paths.DistanceManhattan(p, q)
}
