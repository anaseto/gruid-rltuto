// This file manages actions resulting from user input.

package main

import (
	"github.com/anaseto/gruid"
	"github.com/anaseto/gruid/paths"
)

// action represents information relevant to the last UI action performed.
type action struct {
	Type  actionType  // kind of action (movement, quitting, ...)
	Delta gruid.Point // direction for ActionMovement
}

type actionType int

// These constants represent the possible UI actions.
const (
	NoAction       actionType = iota
	ActionMovement            // movement request
	ActionQuit                // quit the game
)

// handleAction updates the model in response to current recorded last action.
func (m *model) handleAction() gruid.Effect {
	switch m.action.Type {
	case ActionMovement:
		np := m.game.ECS.Positions[m.game.ECS.PlayerID].Add(m.action.Delta)
		m.MovePlayer(np)
	case ActionQuit:
		// for now, just terminate with gruid End command: this will
		// have to be updated later when implementing saving.
		return gruid.End()
	}
	return nil
}

// MovePlayer moves the player to a given position.
func (m *model) MovePlayer(to gruid.Point) {
	if !m.game.Map.Walkable(to) {
		return
	}
	// We move the player to the new destination.
	m.game.ECS.MovePlayer(to)
	// Update FOV.
	m.UpdateFOV()
}

// UpdateFOV updates the field of view.
func (m *model) UpdateFOV() {
	// player position
	player := m.game.ECS.Player()
	pp := m.game.ECS.Positions[m.game.ECS.PlayerID]
	// We shift the FOV's Range so that it will be centered on the new
	// player's position.
	rg := gruid.NewRange(-maxLOS, -maxLOS, maxLOS+1, maxLOS+1)
	player.FOV.SetRange(rg.Add(pp).Intersect(m.game.Map.Grid.Range()))
	// We mark cells in field of view as explored. We use the symmetric
	// shadow casting algorithm provided by the rl package.
	passable := func(p gruid.Point) bool {
		return m.game.Map.Grid.At(p) != Wall
	}
	for _, p := range player.FOV.SSCVisionMap(pp, maxLOS, passable, false) {
		if paths.DistanceManhattan(p, pp) > maxLOS {
			continue
		}
		if !m.game.Map.Explored[p] {
			m.game.Map.Explored[p] = true
		}
	}
}
