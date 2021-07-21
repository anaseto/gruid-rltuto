// This file defines the main model of the game: the Update function that
// updates the model state in response to user input, and the Draw function,
// which draws the grid.

package main

import (
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/anaseto/gruid"
	"github.com/anaseto/gruid/paths"
	"github.com/anaseto/gruid/ui"
)

// model represents our main application's state.
type model struct {
	grid      gruid.Grid  // drawing grid
	game      *game       // game state
	action    action      // UI action
	mode      mode        // UI mode
	log       *ui.Label   // label for log
	status    *ui.Label   // label for status
	desc      *ui.Label   // label for position description
	inventory *ui.Menu    // inventory menu
	viewer    *ui.Pager   // message's history viewer
	mousePos  gruid.Point // mouse position
}

// mode describes distinct kinds of modes for the UI
type mode int

const (
	modeNormal mode = iota
	modeEnd         // win or death (currently only death)
	modeInventoryActivate
	modeInventoryDrop
	modeMessageViewer
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
	case modeMessageViewer:
		m.viewer.Update(msg)
		if m.viewer.Action() == ui.PagerQuit {
			m.mode = modeNormal
		}
		return nil
	case modeInventoryActivate, modeInventoryDrop:
		m.updateInventory(msg)
		return nil
	}
	switch msg := msg.(type) {
	case gruid.MsgInit:
		m.log = &ui.Label{}
		m.status = &ui.Label{}
		m.desc = &ui.Label{Box: &ui.Box{}}
		m.InitializeMessageViewer()
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
		m.game.ECS.DStyle[m.game.ECS.PlayerID] = EStyle{Rune: '@', Color: ColorPlayer}
		m.game.ECS.Name[m.game.ECS.PlayerID] = "player"
		m.game.ECS.Inventory[m.game.ECS.PlayerID] = &Inventory{}
		m.game.UpdateFOV()
		// Add some monsters
		m.game.SpawnMonsters()
		// Add items
		m.game.PlaceItems()
	case gruid.MsgKeyDown:
		// Update action information on key down.
		m.updateMsgKeyDown(msg)
	case gruid.MsgMouse:
		if msg.Action == gruid.MouseMove {
			m.mousePos = msg.P
		}
	}
	// Handle action (if any).
	return m.handleAction()
}

// updateInventory handles input messages when the inventory window is open.
func (m *model) updateInventory(msg gruid.Msg) {
	// We call the Update function of the menu widget, so that we can
	// inspect information about user activity on the menu.
	m.inventory.Update(msg)
	switch m.inventory.Action() {
	case ui.MenuQuit:
		// The user requested to quit the menu.
		m.mode = modeNormal
		return
	case ui.MenuInvoke:
		// The user invoked a particular entry of the menu (either by
		// using enter or clicking on it).
		n := m.inventory.Active()
		var err error
		switch m.mode {
		case modeInventoryDrop:
			err = m.game.InventoryRemove(m.game.ECS.PlayerID, n)
		case modeInventoryActivate:
			err = m.game.InventoryActivate(m.game.ECS.PlayerID, n)
		}
		if err != nil {
			m.game.Logf("%v", ColorLogSpecial, err)
		} else {
			m.game.EndTurn()
		}
		m.mode = modeNormal
	}
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
	case "m":
		m.action = action{Type: ActionViewMessages}
	case "i":
		m.action = action{Type: ActionInventory}
	case "d":
		m.action = action{Type: ActionDrop}
	case "g":
		m.action = action{Type: ActionPickup}
	}
}

// Color definitions. We start from 1, because 0 is gruid.ColorDefault, which
// we use for default foreground and background.
const (
	ColorFOV gruid.Color = iota + 1
	ColorPlayer
	ColorMonster
	ColorLogPlayerAttack
	ColorLogItemUse
	ColorLogMonsterAttack
	ColorLogSpecial
	ColorStatusHealthy
	ColorStatusWounded
	ColorConsumable
)

// Draw implements gruid.Model.Draw. It draws a simple map that spans the whole
// grid.
func (m *model) Draw() gruid.Grid {
	mapgrid := m.grid.Slice(m.grid.Range().Shift(0, 2, 0, -1))
	switch m.mode {
	case modeMessageViewer:
		m.grid.Copy(m.viewer.Draw())
		return m.grid
	case modeInventoryDrop, modeInventoryActivate:
		mapgrid.Copy(m.inventory.Draw())
		return m.grid
	}
	m.grid.Fill(gruid.Cell{Rune: ' '})
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
	m.DrawNames(mapgrid)
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

// DrawNames renders the names of the named entities at current mouse location
// if it is in the map.
func (m *model) DrawNames(gd gruid.Grid) {
	maprg := gruid.NewRange(0, 2, UIWidth, UIHeight-1)
	if !m.mousePos.In(maprg) {
		return
	}
	p := m.mousePos.Sub(gruid.Point{0, 2})
	// We get the names of the entities at p.
	names := []string{}
	for i, q := range m.game.ECS.Positions {
		if q != p || !m.game.InFOV(q) {
			continue
		}
		name := m.game.ECS.GetName(i)
		if name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return
	}
	// We sort the names. This could be improved to sort by entity type
	// too, as well as to remove duplicates (for example showing “corpse
	// (3x)” if there are three corpses).
	sort.Strings(names)

	text := strings.Join(names, ", ")
	width := utf8.RuneCountInString(text) + 2
	rg := gruid.NewRange(p.X+1, p.Y-1, p.X+1+width, p.Y+2)
	// we adjust a bit the box's placement in case it's on a edge.
	if p.X+1+width >= UIWidth {
		rg = rg.Shift(-1-width, 0, -1-width, 0)
	}
	if p.Y+2 > MapHeight {
		rg = rg.Shift(0, -1, 0, -1)
	}
	if p.Y-1 < 0 {
		rg = rg.Shift(0, 1, 0, 1)
	}
	slice := gd.Slice(rg)
	m.desc.Content = ui.Text(text)
	m.desc.Draw(slice)
}
