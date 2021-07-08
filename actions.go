// This file manages actions resulting from user input.

package main

import (
	"log"

	"github.com/anaseto/gruid"
	"github.com/anaseto/gruid/paths"
)

// action represents information relevant to the last UI action performed.
type action struct {
	Type  actionType  // kind of action (movement, quitting, ...)
	Delta gruid.Point // direction for ActionBump
}

type actionType int

// These constants represent the possible UI actions.
const (
	NoAction   actionType = iota
	ActionBump            // bump request (attack or movement)
	ActionQuit            // quit the game
)

// handleAction updates the model in response to current recorded last action.
func (m *model) handleAction() gruid.Effect {
	switch m.action.Type {
	case ActionBump:
		np := m.game.ECS.Positions[m.game.ECS.PlayerID].Add(m.action.Delta)
		m.game.Bump(np)
	case ActionQuit:
		// for now, just terminate with gruid End command: this will
		// have to be updated later when implementing saving.
		return gruid.End()
	}
	return nil
}

// Bump moves the player to a given position and updates FOV information,
// or attacks if there is a monster.
func (g *game) Bump(to gruid.Point) {
	if !g.Map.Walkable(to) {
		return
	}
	if m := g.ECS.MonsterAt(to); m != nil {
		// We show a message to standard error. Later in the tutorial,
		// we'll put a message in the UI instead.
		log.Printf("You kick the %s, much to its annoyance!\n", m.Name)
		return
	}
	// We move the player to the new destination.
	g.ECS.MovePlayer(to)
	// Update FOV.
	g.UpdateFOV()
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
