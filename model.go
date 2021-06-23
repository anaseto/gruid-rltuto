// This file defines the main model of the game: the Update function that
// updates the model state in response to user input, and the Draw function,
// which draws the grid.

package main

import (
	"github.com/anaseto/gruid"
	"github.com/anaseto/gruid/rl"
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
	Map Map  // the game map, made of tiles
}

// Update implements gruid.Model.Update. It handles keyboard and mouse input
// messages and updates the model in response to them.
func (m *model) Update(msg gruid.Msg) gruid.Effect {
	m.action = action{} // reset last action information
	switch msg := msg.(type) {
	case gruid.MsgInit:
		// Initialize map
		size := m.grid.Size() // map size: for now the whole window
		m.game.Map.Grid = rl.NewGrid(size.X, size.Y)
		m.game.Map.Grid.Fill(Floor)
		for i := 0; i < 3; i++ {
			// We add a few walls. We'll deal with map generation
			// in the next part of the tutorial.
			m.game.Map.Grid.Set(gruid.Point{30 + i, 12}, Wall)
		}
		// Initialize entities
		m.game.ECS = &ECS{}
		// Initialization: create a player entity centered on the map.
		m.game.ECS.AddEntity(&Player{P: size.Div(2)})
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

// Draw implements gruid.Model.Draw. It draws a simple map that spans the whole
// grid.
func (m *model) Draw() gruid.Grid {
	m.grid.Fill(gruid.Cell{Rune: ' '})
	// We draw the map tiles.
	it := m.game.Map.Grid.Iterator()
	for it.Next() {
		m.grid.Set(it.P(), gruid.Cell{Rune: m.game.Map.Rune(it.Cell())})
	}
	// We draw the entities.
	for _, e := range m.game.ECS.Entities {
		m.grid.Set(e.Pos(), gruid.Cell{
			Rune:  e.Rune(),
			Style: gruid.Style{Fg: e.Color()},
		})
	}
	return m.grid
}
