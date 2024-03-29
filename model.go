// This file defines the main model of the game: the Update function that
// updates the model state in response to user input, and the Draw function,
// which draws the grid.

package main

import (
	"math/rand"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/anaseto/gruid"
	"github.com/anaseto/gruid/ui"
)

// model represents our main application's state.
type model struct {
	grid      gruid.Grid // drawing grid
	game      *game      // game state
	action    action     // UI action
	mode      mode       // UI mode
	log       *ui.Label  // label for log
	status    *ui.Label  // label for status
	desc      *ui.Label  // label for position description
	inventory *ui.Menu   // inventory menu
	viewer    *ui.Pager  // message's history viewer
	targ      targeting  // targeting information
	gameMenu  *ui.Menu   // game's main menu
	info      *ui.Label  // info label in main menu (for errors)
}

// targeting describes information related to examination or selection of
// particular positions in the map.
type targeting struct {
	pos    gruid.Point
	item   int // item to use after selecting target
	radius int
}

// mode describes distinct kinds of modes for the UI. It is used to send user
// input messages to different handlers (inventory window, map, message viewer,
// etc.), depending on the current mode.
type mode int

const (
	modeNormal mode = iota
	modeEnd         // win or death (currently only death)
	modeInventoryActivate
	modeInventoryDrop
	modeGameMenu
	modeMessageViewer
	modeTargeting   // targeting mode (item use)
	modeExamination // keyboad map examination mode
)

// Update implements gruid.Model.Update. It handles keyboard and mouse input
// messages and updates the model in response to them.
func (m *model) Update(msg gruid.Msg) gruid.Effect {
	switch msg.(type) {
	case gruid.MsgInit:
		return m.init()
	}
	m.action = action{} // reset last action information
	switch m.mode {
	case modeGameMenu:
		return m.updateGameMenu(msg)
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
	case modeTargeting, modeExamination:
		m.updateTargeting(msg)
		return nil
	}
	switch msg := msg.(type) {
	case gruid.MsgKeyDown:
		// Update action information on key down.
		m.updateMsgKeyDown(msg)
	case gruid.MsgMouse:
		if msg.Action == gruid.MouseMove {
			m.targ.pos = msg.P
		}
	}
	// Handle action (if any).
	return m.handleAction()
}

const (
	MenuNewGame = iota
	MenuContinue
	MenuQuit
)

// init initializes the model: widgets' initialization, and starting mode.
func (m *model) init() gruid.Effect {
	m.log = &ui.Label{}
	m.status = &ui.Label{}
	m.info = &ui.Label{}
	m.desc = &ui.Label{Box: &ui.Box{}}
	m.InitializeMessageViewer()
	m.mode = modeGameMenu
	entries := []ui.MenuEntry{
		MenuNewGame:  {Text: ui.Text("(N)ew game"), Keys: []gruid.Key{"N", "n"}},
		MenuContinue: {Text: ui.Text("(C)ontinue last game"), Keys: []gruid.Key{"C", "c"}},
		MenuQuit:     {Text: ui.Text("(Q)uit")},
	}
	m.gameMenu = ui.NewMenu(ui.MenuConfig{
		Grid:    gruid.NewGrid(UIWidth/2, len(entries)+2),
		Box:     &ui.Box{Title: ui.Text("Gruid Roguelike Tutorial")},
		Entries: entries,
		Style:   ui.MenuStyle{Active: gruid.Style{}.WithFg(ColorMenuActive)},
	})
	return nil
}

// updateGameMenu updates the Game Menu and switchs mode to normal after
// starting a new game or loading an old one.
func (m *model) updateGameMenu(msg gruid.Msg) gruid.Effect {
	rg := m.grid.Range().Intersect(m.grid.Range().Add(mainMenuAnchor))
	m.gameMenu.Update(rg.RelMsg(msg))
	switch m.gameMenu.Action() {
	case ui.MenuMove:
		m.info.SetText("")
	case ui.MenuInvoke:
		m.info.SetText("")
		switch m.gameMenu.Active() {
		case MenuNewGame:
			m.game = NewGame()
			m.mode = modeNormal
		case MenuContinue:
			data, err := LoadFile("save")
			if err != nil {
				m.info.SetText(err.Error())
				break
			}
			g, err := DecodeGame(data)
			if err != nil {
				m.info.SetText(err.Error())
				break
			}
			m.game = g
			m.mode = modeNormal
			// the random number generator is not saved
			m.game.Map.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
		case MenuQuit:
			return gruid.End()
		}
	case ui.MenuQuit:
		return gruid.End()
	}
	return nil
}

// updateTargeting updates targeting information in response to user input
// messages.
func (m *model) updateTargeting(msg gruid.Msg) {
	maprg := gruid.NewRange(0, LogLines, UIWidth, UIHeight-1)
	if !m.targ.pos.In(maprg) {
		m.targ.pos = m.game.ECS.PP().Add(maprg.Min)
	}
	p := m.targ.pos.Sub(maprg.Min)
	switch msg := msg.(type) {
	case gruid.MsgKeyDown:
		switch msg.Key {
		case gruid.KeyArrowLeft, "h":
			p = p.Shift(-1, 0)
		case gruid.KeyArrowDown, "j":
			p = p.Shift(0, 1)
		case gruid.KeyArrowUp, "k":
			p = p.Shift(0, -1)
		case gruid.KeyArrowRight, "l":
			p = p.Shift(1, 0)
		case gruid.KeyEnter, ".":
			if m.mode == modeExamination {
				break
			}
			m.activateTarget(p)
			return
		case gruid.KeyEscape, "q":
			m.targ = targeting{}
			m.mode = modeNormal
			return
		}
		m.targ.pos = p.Add(maprg.Min)
	case gruid.MsgMouse:
		switch msg.Action {
		case gruid.MouseMove:
			m.targ.pos = msg.P
		case gruid.MouseMain:
			m.activateTarget(p)
		}
	}
}

func (m *model) activateTarget(p gruid.Point) {
	err := m.game.InventoryActivateWithTarget(m.game.ECS.PlayerID, m.targ.item, &p)
	if err != nil {
		m.game.Logf("%v", ColorLogSpecial, err)
	} else {
		m.game.EndTurn()
	}
	m.mode = modeNormal
	m.targ = targeting{}
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
			if radius := m.game.TargetingRadius(n); radius >= 0 {
				m.targ = targeting{
					item:   n,
					pos:    m.game.ECS.PP().Shift(0, LogLines),
					radius: radius,
				}
				m.mode = modeTargeting
				return
			}
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
	m.targ.pos = gruid.Point{}
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
	case "Q":
		m.action = action{Type: ActionQuit}
	case "S":
		m.action = action{Type: ActionSave}
	case "m":
		m.action = action{Type: ActionViewMessages}
	case "i":
		m.action = action{Type: ActionInventory}
	case "d":
		m.action = action{Type: ActionDrop}
	case "g":
		m.action = action{Type: ActionPickup}
	case "x":
		m.action = action{Type: ActionExamine}
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
	ColorMenuActive
)

const (
	AttrReverse = 1 << iota
)

// Draw implements gruid.Model.Draw. It draws a simple map that spans the whole
// grid.
func (m *model) Draw() gruid.Grid {
	mapgrid := m.grid.Slice(m.grid.Range().Shift(0, LogLines, 0, -1))
	switch m.mode {
	case modeGameMenu:
		return m.DrawGameMenu()
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
		c.Rune, c.Style.Fg = g.ECS.GetStyle(i)
		mapgrid.Set(p, c)
		// NOTE: We retrieved current cell at e.Pos() to preserve
		// background (in FOV or not).
	}
	m.DrawNames(mapgrid)
	m.DrawLog(m.grid.Slice(m.grid.Range().Lines(0, LogLines)))
	m.DrawStatus(m.grid.Slice(m.grid.Range().Line(m.grid.Size().Y - 1)))
	return m.grid
}

var mainMenuAnchor = gruid.Point{10, 6}

// DrawGameMenu draws the game's main menu.
func (m *model) DrawGameMenu() gruid.Grid {
	m.grid.Fill(gruid.Cell{Rune: ' '})
	m.grid.Slice(m.gameMenu.Bounds().Add(mainMenuAnchor)).Copy(m.gameMenu.Draw())
	m.info.Draw(m.grid.Slice(m.grid.Range().Line(12).Shift(10, 0, 0, 0)))
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
	maprg := gruid.NewRange(0, LogLines, UIWidth, UIHeight-1)
	if !m.targ.pos.In(maprg) {
		return
	}
	p := m.targ.pos.Sub(maprg.Min)
	rad := m.targ.radius
	rg := gruid.Range{Min: p.Sub(gruid.Point{rad, rad}), Max: p.Add(gruid.Point{rad + 1, rad + 1})}
	rg = rg.Intersect(maprg.Sub(maprg.Min))
	rg.Iter(func(q gruid.Point) {
		c := gd.At(q)
		c.Style.Attrs |= AttrReverse
		gd.Set(q, c)
	})
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
	rg = gruid.NewRange(p.X+1, p.Y-1, p.X+1+width, p.Y+2)
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
