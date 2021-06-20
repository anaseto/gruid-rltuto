// This file manages actions resulting from user input.

package main

import "github.com/anaseto/gruid"

// action represents information relevant to the last UI action performed.
type action struct {
	Type  actionType  // kind of action
	Delta gruid.Point // for ActionMovement
}

type actionType int

// These constants represent the possible UI actions.
const (
	NoAction       actionType = iota
	ActionMovement            // movement request
	ActionQuit                // quit the game
)

func (m *model) handleAction() gruid.Effect {
	switch m.action.Type {
	case ActionMovement:
		m.game.PlayerPos = m.game.PlayerPos.Add(m.action.Delta)
	case ActionQuit:
		return gruid.End()
	}
	return nil
}
