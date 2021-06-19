// This is the main file of tutorial part 1.
package main

import (
	"context"
	"log"

	"github.com/anaseto/gruid"
	sdl "github.com/anaseto/gruid-sdl"
)

func main() {
	// Create a new grid with standard 80x24 size.
	gd := gruid.NewGrid(80, 24)
	// Create the main application's model, using grid gd.
	m := &model{grid: gd}
	// Get a TileManager for drawing fonts on the screen.
	t, err := GetTileDrawer()
	if err != nil {
		log.Fatal(err)
	}
	// Use the SDL2 driver from gruid-sdl, using the previously defined
	// TileManager.
	dr := sdl.NewDriver(sdl.Config{
		TileManager: t,
	})

	// Define new application
	app := gruid.NewApp(gruid.AppConfig{
		Driver: dr,
		Model:  m,
	})

	// Start application
	if err := app.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
}

// models represents our main application state.
type model struct {
	grid   gruid.Grid // drawing grid
	game   game       // game state
	action action     // UI action
}

// game represents information relevant the current game's state.
type game struct {
	PlayerPos gruid.Point // tracks player position
}

// Update implements gruid.Model.Update. It handles keyboard and mouse input
// messages and updates the model in response to them.
func (m *model) Update(msg gruid.Msg) gruid.Effect {
	m.action = action{} // reset last action information
	switch msg := msg.(type) {
	case gruid.MsgKeyDown:
		// update action information on key down
		m.updateMsgKeyDown(msg)
	}
	// handle action (if any)
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
	case "Q", "q", gruid.KeyEscape:
		m.action = action{Type: ActionQuit}
	}
}

// Draw implements gruid.Model.Draw. It draws a simple map that spans the whole
// grid.
func (m *model) Draw() gruid.Grid {
	it := m.grid.Iterator()
	for it.Next() {
		switch {
		case it.P() == m.game.PlayerPos:
			it.SetCell(gruid.Cell{Rune: '@'})
		default:
			it.SetCell(gruid.Cell{Rune: ' '})
		}
	}
	return m.grid
}
