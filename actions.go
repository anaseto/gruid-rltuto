// This file manages actions resulting from user input.

package main

import (
	"log"

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
	ActionDrop                    // menu to drop an inventory item
	ActionInventory               // inventory menu to use an item
	ActionPickup                  // pickup an item on the ground
	ActionWait                    // wait a turn
	ActionQuit                    // quit the game (without saving)
	ActionSave                    // save the game
	ActionViewMessages            // view history messages
	ActionExamine                 // examine map
)

// handleAction updates the model in response to current recorded last action.
func (m *model) handleAction() gruid.Effect {
	switch m.action.Type {
	case ActionBump:
		np := m.game.ECS.PP().Add(m.action.Delta)
		m.game.Bump(np)
	case ActionDrop:
		m.OpenInventory("Drop item")
		m.mode = modeInventoryDrop
	case ActionInventory:
		m.OpenInventory("Use item")
		m.mode = modeInventoryActivate
	case ActionPickup:
		m.game.PickupItem()
	case ActionWait:
		m.game.EndTurn()
	case ActionSave:
		data, err := EncodeGame(m.game)
		if err == nil {
			err = SaveFile("save", data)
		}
		if err != nil {
			m.game.Logf("Could not save game.", ColorLogSpecial)
			log.Printf("could not save game: %v", err)
			break
		}
		return gruid.End()
	case ActionQuit:
		// Remove any previously saved files (if any).
		RemoveDataFile("save")
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
	case ActionExamine:
		m.mode = modeExamination
		m.targ.pos = m.game.ECS.PP().Shift(0, LogLines)
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
	if i := g.ECS.MonsterAt(to); g.ECS.Alive(i) {
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

// PickupItem takes an item on the floor.
func (g *game) PickupItem() {
	pp := g.ECS.PP()
	for i, p := range g.ECS.Positions {
		if p != pp {
			// Skip entities whose position is diffferent than the
			// player's.
			continue
		}
		err := g.InventoryAdd(g.ECS.PlayerID, i)
		if err != nil {
			if err.Error() == ErrNoShow {
				// Happens for example if the current entity is
				// not a consumable.
				continue
			}
			g.Logf("Could not pickup: %v", ColorLogSpecial, err)
			return
		}
		g.Logf("You pickup %v", ColorLogItemUse, g.ECS.Name[i])
		g.EndTurn()
		return
	}
}

// OpenInventory opens the inventory and allows the player to select an item.
func (m *model) OpenInventory(title string) {
	inv := m.game.ECS.Inventory[m.game.ECS.PlayerID]
	// We build a list of entries.
	entries := []ui.MenuEntry{}
	r := 'a'
	for _, it := range inv.Items {
		name := m.game.ECS.Name[it]
		entries = append(entries, ui.MenuEntry{
			Text: ui.Text(string(r) + " - " + name),
			// allow to use the character r to select the entry
			Keys: []gruid.Key{gruid.Key(r)},
		})
		r++
	}
	// We create a new menu widget for the inventory window.
	m.inventory = ui.NewMenu(ui.MenuConfig{
		Grid:    gruid.NewGrid(40, MapHeight),
		Box:     &ui.Box{Title: ui.Text(title)},
		Entries: entries,
	})
}
