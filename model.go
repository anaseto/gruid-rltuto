// This file defines the main model of the game: the Update function that
// updates the model state in response to user input, and the Draw function,
// which draws the grid.

package main

import (
	"sort"

	"github.com/anaseto/gruid"
	"github.com/anaseto/gruid/paths"
	"github.com/anaseto/gruid/ui"
)

// model represents our main application's state.
type model struct {
	grid   gruid.Grid // drawing grid
	game   *game      // game state
	action action     // UI action
	mode   mode       // UI mode
	log    *ui.Label  // label for log
	status *ui.Label  // label for status
}

// mode describes distinct kinds of modes for the UI
type mode int

const (
	modeNormal mode = iota
	modeEnd         // win or death (currently only death)
)

// Update implements gruid.Model.Update. It handles keyboard and mouse input
// messages and updates the model in response to them.
func (m *model) Update(msg gruid.Msg) gruid.Effect {
	m.action = action{} // reset last action information
	switch m.mode {
	case modeEnd:
		switch msg := msg.(type) {
		case gruid.MsgKeyDown:
			switch msg.Key {
			case "q", gruid.KeyEscape:
				// You died: quit on "q" or "escape"
				return gruid.End()
			}
		}
		return nil
	}
	switch msg := msg.(type) {
	case gruid.MsgInit:
		m.log = &ui.Label{}
		m.status = &ui.Label{}
		m.game = &game{}
		// Initialize map
		size := m.grid.Size()
		size.Y -= 3 // for log and status
		m.game.Map = NewMap(size)
		m.game.PR = paths.NewPathRange(gruid.NewRange(0, 0, size.X, size.Y))
		// Initialize entities
		m.game.ECS = NewECS()
		// Initialization: create a player entity centered on the map.
		m.game.ECS.PlayerID = m.game.ECS.AddEntity(NewPlayer(), m.game.Map.RandomFloor())
		m.game.ECS.Fighter[m.game.ECS.PlayerID] = &fighter{
			HP: 30, MaxHP: 30, Power: 5, Defense: 2,
		}
		m.game.ECS.Name[m.game.ECS.PlayerID] = "you"
		m.game.UpdateFOV()
		// Add some monsters
		m.game.SpawnMonsters()
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
		m.action = action{Type: ActionBump, Delta: pdelta.Shift(-1, 0)}
	case gruid.KeyArrowDown, "j":
		m.action = action{Type: ActionBump, Delta: pdelta.Shift(0, 1)}
	case gruid.KeyArrowUp, "k":
		m.action = action{Type: ActionBump, Delta: pdelta.Shift(0, -1)}
	case gruid.KeyArrowRight, "l":
		m.action = action{Type: ActionBump, Delta: pdelta.Shift(1, 0)}
	case gruid.KeyEnter, ".":
		m.action = action{Type: ActionWait}
	case gruid.KeyEscape, "q":
		m.action = action{Type: ActionQuit}
	}
}

// Color definitions. We start from 1, because 0 is gruid.ColorDefault, which
// we use for default foreground and background.
const (
	ColorFOV gruid.Color = iota + 1
	ColorPlayer
	ColorMonster
	ColorLogPlayerAttack
	ColorLogMonsterAttack
	ColorLogSpecial
	ColorStatusHealthy
	ColorStatusWounded
)

// Draw implements gruid.Model.Draw. It draws a simple map that spans the whole
// grid.
func (m *model) Draw() gruid.Grid {
	m.grid.Fill(gruid.Cell{Rune: ' '})
	mapgrid := m.grid.Slice(m.grid.Range().Shift(0, 2, 0, -1))
	g := m.game
	// We draw the map tiles.
	it := g.Map.Grid.Iterator()
	for it.Next() {
		if !g.Map.Explored[it.P()] {
			continue
		}
		c := gruid.Cell{Rune: g.Map.Rune(it.Cell())}
		if g.InFOV(it.P()) {
			c.Style.Bg = ColorFOV
		}
		mapgrid.Set(it.P(), c)
	}
	// We sort entity indexes using the render ordering.
	sortedEntities := make([]int, 0, len(g.ECS.Entities))
	for i := range g.ECS.Entities {
		sortedEntities = append(sortedEntities, i)
	}
	sort.Slice(sortedEntities, func(i, j int) bool {
		return g.ECS.RenderOrder(sortedEntities[i]) < g.ECS.RenderOrder(sortedEntities[j])
	})
	// We draw the sorted entities.
	for _, i := range sortedEntities {
		p := g.ECS.Positions[i]
		if !g.Map.Explored[p] || !g.InFOV(p) {
			continue
		}
		c := mapgrid.At(p)
		c.Rune, c.Style.Fg = g.ECS.Style(i)
		mapgrid.Set(p, c)
		// NOTE: We retrieved current cell at e.Pos() to preserve
		// background (in FOV or not).
	}
	m.DrawLog(m.grid.Slice(m.grid.Range().Lines(0, 2)))
	m.DrawStatus(m.grid.Slice(m.grid.Range().Line(m.grid.Size().Y - 1)))
	return m.grid
}

// DrawLog draws the last two lines of the log.
func (m *model) DrawLog(gd gruid.Grid) {
	j := 1
	for i := len(m.game.Log) - 1; i >= 0; i-- {
		if j < 0 {
			break
		}
		e := m.game.Log[i]
		st := gruid.Style{}
		st.Fg = e.Color
		m.log.Content = ui.NewStyledText(e.String(), st)
		m.log.Draw(gd.Slice(gd.Range().Line(j)))
		j--
	}
}

// DrawStatus draws the status line
func (m *model) DrawStatus(gd gruid.Grid) {
	st := gruid.Style{}
	st.Fg = ColorStatusHealthy
	g := m.game
	f := g.ECS.Fighter[g.ECS.PlayerID]
	if f.HP < f.MaxHP/2 {
		st.Fg = ColorStatusWounded
	}
	m.log.Content = ui.Textf("HP: %d/%d", f.HP, f.MaxHP).WithStyle(st)
	m.log.Draw(gd)
}
