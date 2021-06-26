// This file defines the main model of the game: the Update function that
// updates the model state in response to user input, and the Draw function,
// which draws the grid.

package main

import (
	"github.com/anaseto/gruid"
)

// models represents our main application state.
type model struct {
	grid   gruid.Grid // drawing grid
	game   game       // game state
	action action     // UI action
}

// game represents information relevant the current game's state.
type game struct {
	ECS *ECS // entities present on the map
	Map *Map // the game map, made of tiles
}

// Update implements gruid.Model.Update. It handles keyboard and mouse input
// messages and updates the model in response to them.
func (m *model) Update(msg gruid.Msg) gruid.Effect {
	m.action = action{} // reset last action information
	switch msg := msg.(type) {
	case gruid.MsgInit:
		// Initialize map
		size := m.grid.Size() // map size: for now the whole window
		m.game.Map = NewMap(size)
		// Initialize entities
		m.game.ECS = &ECS{}
		// Position the player in a random floor cell.
		m.game.ECS.AddEntity(NewPlayer(m.game.Map.RandomFloor()))
		m.UpdateFOV()
	case gruid.MsgKeyDown:
		// Update action information on key down.
		m.updateMsgKeyDown(msg)
	}
	// Handle action (if any).
	return m.handleAction()
}

func (m *model) updateMsgKeyDown(msg gruid.MsgKeyDown) {
	pdelta := gruid.Point{}
	switch msg.Key {
	case gruid.KeyArrowLeft, "h":
		m.action = action{Type: ActionMovement, Delta: pdelta.Shift(-1, 0)}
	case gruid.KeyArrowDown, "j":
		m.action = action{Type: ActionMovement, Delta: pdelta.Shift(0, 1)}
	case gruid.KeyArrowUp, "k":
		m.action = action{Type: ActionMovement, Delta: pdelta.Shift(0, -1)}
	case gruid.KeyArrowRight, "l":
		m.action = action{Type: ActionMovement, Delta: pdelta.Shift(1, 0)}
	case gruid.KeyEscape, "q":
		m.action = action{Type: ActionQuit}
	}
}

// Color definitions. For now, we use a special color for FOV. We start from 1,
// because 0 is gruid.ColorDefault, which we use for default foreground and
// background.
const (
	ColorFOV gruid.Color = iota + 1
)

// Draw implements gruid.Model.Draw. It draws a simple map that spans the whole
// grid.
func (m *model) Draw() gruid.Grid {
	m.grid.Fill(gruid.Cell{Rune: ' '})
	// We draw the map tiles.
	it := m.game.Map.Grid.Iterator()
	for it.Next() {
		if !m.game.Map.Explored[it.P()] {
			continue
		}
		c := gruid.Cell{Rune: m.game.Map.Rune(it.Cell())}
		if m.game.ECS.Player().FOV.Visible(it.P()) {
			c.Style.Bg = ColorFOV
		}
		m.grid.Set(it.P(), c)
	}
	// We draw the entities.
	for _, e := range m.game.ECS.Entities {
		if !m.game.Map.Explored[e.Pos()] {
			continue
		}
		c := m.grid.At(e.Pos())
		c.Rune = e.Rune()
		c.Style.Fg = e.Color()
		m.grid.Set(e.Pos(), c)
		// NOTE: We retrieved current cell at e.Pos() to preserve
		// background (in FOV or not).
	}
	return m.grid
}
