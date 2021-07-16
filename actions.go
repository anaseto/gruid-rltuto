// This file manages actions resulting from user input.

package main

import (
	"github.com/anaseto/gruid"
	"github.com/anaseto/gruid/ui"
)

// action represents information relevant to the last UI action performed.
type action struct {
	Type  actionType  // kind of action (movement, quitting, ...)
	Delta gruid.Point // direction for ActionBump
}

type actionType int

// These constants represent the possible UI actions.
const (
	NoAction           actionType = iota
	ActionBump                    // bump request (attack or movement)
	ActionWait                    // wait a turn
	ActionQuit                    // quit the game
	ActionViewMessages            // view history messages
)

// handleAction updates the model in response to current recorded last action.
func (m *model) handleAction() gruid.Effect {
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
	case ActionViewMessages:
		m.mode = modeMessageViewer
		lines := []ui.StyledText{}
		for _, e := range m.game.Log {
			st := gruid.Style{}
			st.Fg = e.Color
			lines = append(lines, ui.NewStyledText(e.String(), st))
		}
		m.viewer.SetLines(lines)
	}
	if m.game.ECS.PlayerDied() {
		m.game.Logf("You died -- press “q” or escape to quit", ColorLogSpecial)
		m.mode = modeEnd
		return nil
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
