// This file manages actions resulting from user input.

package main

import (
	"log"

	"github.com/anaseto/gruid"
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
	ActionWait            // wait a turn
	ActionQuit            // quit the game
)

// handleAction updates the model in response to current recorded last action.
func (m *model) handleAction() gruid.Effect {
	if m.game.ECS.PlayerDied() {
		log.Print("You died")
		return gruid.End()
	}
	switch m.action.Type {
	case ActionBump:
		np := m.game.ECS.Positions[m.game.ECS.PlayerID].Add(m.action.Delta)
		m.game.Bump(np)
	case ActionWait:
		m.game.EndTurn()
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
	if i, _ := g.ECS.MonsterAt(to); g.ECS.Alive(i) {
		// We show a message to standard error. Later in the tutorial,
		// we'll put a message in the UI instead.
		g.BumpAttack(g.ECS.PlayerID, i)
		g.EndTurn()
		return
	}
	// We move the player to the new destination.
	g.ECS.MovePlayer(to)
	g.EndTurn()
}
