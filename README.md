# Gruid Go Roguelike Tutorial

This tutorial follows the overall structure of the [TCOD Python
Tutorial](http://rogueliketutorials.com/tutorials/tcod/v2), but makes use of
the [Go programming language](https://golang.org/) and the
[gruid](https://github.com/anaseto/gruid) roguelike game framework, instead of
TCOD.

[Table of Contents](https://github.com/anaseto/gruid-rltuto)

# Part 8 - Items and Inventory

In this part, we will implement inventory management and support for items
(just a health potion in this part).

For conveniency, a minor refactoring has been done with respect to styling
information: instead of storing this information in each `Entity`, a new
`Style` component has been added to `components.go`. Also, to allow easy
removal of entities (for example after using a potion it disappears), they are
now stored in a map instead of a slice. A shorthand method `PP` has been added
to access the player's position, and a `GetName` method (to give namings
different than the default for an Entity, which we use for “corpse” in the case
of dead monsters).

With respect to the inventory, a new `Inventory` component has been added.

The new file `items.go` describes the `Consumable` interface, which requires an
`Activate` method. This method receives arguments with information necesary for
the activation of an item, like who was the actor who used the item, and on
which target (if any) it was used. The file also describes the new
`HealingPotion`.

``` go
// This file describes item entities.

package main

import (
	"errors"
	"fmt"

	"github.com/anaseto/gruid"
)

// Consumable describes a consumable item, like a potion.
type Consumable interface {
	// Activate makes use of an item using a specific action. It returns
	// an error if the consumable could not be activated.
	Activate(g *game, a itemAction) error
}

// itemAction describes information relative to usage of an item: which
// actor does the action, and whether the action has a particular target
// position.
type itemAction struct {
	Actor  int          // entity doing the action
	Target *gruid.Point // optional target
}

// HealingPotion describes a potion that heals of a given amount.
type HealingPotion struct {
	Amount int
}

func (pt *HealingPotion) Activate(g *game, a itemAction) error {
	fi := g.ECS.Fighter[a.Actor]
	if fi == nil {
		// should not happen in practice
		return fmt.Errorf("%s cannot use healing potions.", g.ECS.Name[a.Actor])
	}
	hp := fi.Heal(pt.Amount)
	if hp <= 0 {
		return errors.New("Your health is already full.")
	}
	if a.Actor == g.ECS.PlayerID {
		g.Logf("You regained %d HP", ColorLogItemUse, hp)
	}
	return nil
}
```

The file `components` describes the new `Inventory` component, and adds support
for healing, as well as the new `Style` component, as per the refactoring
mentioned before.

``` diff
diff --git a/components.go b/components.go
index c4a3ade..7fa20f0 100644
--- a/components.go
+++ b/components.go
@@ -13,7 +13,30 @@ type fighter struct {
 	Defense int // defence
 }
 
+// Heal heals a fighter for a certain amount, if it does not exceed maximum HP.
+// The final amount of healing is returned.
+func (fi *fighter) Heal(n int) int {
+	fi.HP += n
+	if fi.HP > fi.MaxHP {
+		n -= fi.HP - fi.MaxHP
+		fi.HP = fi.MaxHP
+	}
+	return n
+}
+
 // AI holds simple AI data for monster's.
 type AI struct {
 	Path []gruid.Point // path to destination
 }
+
+// Style contains information relative to the default graphical representation
+// of an entity.
+type Style struct {
+	Rune  rune
+	Color gruid.Color
+}
+
+// Inventory holds items. For now, consumables.
+type Inventory struct {
+	Items []int
+}
```

Here are the updates that were done in `entity.go` to add support for Inventory
(in addition to small refactorings already mentioned).

``` diff
diff --git a/entity.go b/entity.go
index fac1c75..8bbe8a0 100644
--- a/entity.go
+++ b/entity.go
@@ -12,31 +12,50 @@ import (
 // (Entity-Component-System) in this tutorial, opting for a simpler hybrid
 // approach good enough for the tutorial purposes.
 type ECS struct {
-	Entities  []Entity            // list of entities
+	Entities  map[int]Entity      // set of entities
 	Positions map[int]gruid.Point // entity index: map position
 	PlayerID  int                 // index of Player's entity (for convenience)
+	NextID    int                 // next available id
 
-	Fighter map[int]*fighter // figthing component
-	AI      map[int]*AI      // AI component
-	Name    map[int]string   // name component
+	Fighter   map[int]*fighter   // figthing component
+	AI        map[int]*AI        // AI component
+	Name      map[int]string     // name component
+	Style     map[int]Style      // default style component
+	Inventory map[int]*Inventory // inventory component
 }
 
 // NewECS returns an initialized ECS structure.
 func NewECS() *ECS {
 	return &ECS{
+		Entities:  map[int]Entity{},
 		Positions: map[int]gruid.Point{},
 		Fighter:   map[int]*fighter{},
 		AI:        map[int]*AI{},
 		Name:      map[int]string{},
+		Style:     map[int]Style{},
+		Inventory: map[int]*Inventory{},
+		NextID:    0,
 	}
 }
 
 // Add adds a new entity at a given position and returns its index/id.
 func (es *ECS) AddEntity(e Entity, p gruid.Point) int {
-	i := len(es.Entities)
-	es.Entities = append(es.Entities, e)
-	es.Positions[i] = p
-	return i
+	id := es.NextID
+	es.Entities[id] = e
+	es.Positions[id] = p
+	es.NextID++
+	return id
+}
+
+// RemoveEntity removes an entity, given its identifier.
+func (es *ECS) RemoveEntity(i int) {
+	delete(es.Entities, i)
+	delete(es.Positions, i)
+	delete(es.Fighter, i)
+	delete(es.AI, i)
+	delete(es.Name, i)
+	delete(es.Style, i)
+	delete(es.Inventory, i)
 }
 
 // MoveEntity moves the i-th entity to p.
@@ -55,6 +74,11 @@ func (es *ECS) Player() *Player {
 	return es.Entities[es.PlayerID].(*Player)
 }
 
+// PP returns the Player's position. Just a convenience shorthand.
+func (es *ECS) PP() gruid.Point {
+	return es.Positions[es.PlayerID]
+}
+
 // MonsterAt returns the Monster at p along with its index, if any, or nil if
 // there is no monster at p.
 func (es *ECS) MonsterAt(p gruid.Point) (int, *Monster) {
@@ -75,7 +99,7 @@ func (es *ECS) MonsterAt(p gruid.Point) (int, *Monster) {
 // player nor monsters in this tutorial).
 func (es *ECS) NoBlockingEntityAt(p gruid.Point) bool {
 	i, _ := es.MonsterAt(p)
-	return es.Positions[es.PlayerID] != p && !es.Alive(i)
+	return es.PP() != p && !es.Alive(i)
 }
 
 // PlayerDied checks whether the player died.
@@ -95,11 +119,11 @@ func (es *ECS) Dead(i int) bool {
 	return fi != nil && fi.HP <= 0
 }
 
-// Style returns the graphical representation (rune and foreground color) of an
+// GetStyle returns the graphical representation (rune and foreground color) of an
 // entity.
-func (es *ECS) Style(i int) (r rune, c gruid.Color) {
-	r = es.Entities[i].Rune()
-	c = es.Entities[i].Color()
+func (es *ECS) GetStyle(i int) (r rune, c gruid.Color) {
+	r = es.Style[i].Rune
+	c = es.Style[i].Color
 	if es.Dead(i) {
 		// Alternate representation for corpses of dead monsters.
 		r = '%'
@@ -108,6 +132,16 @@ func (es *ECS) Style(i int) (r rune, c gruid.Color) {
 	return r, c
 }
 
+// GetName returns the name of an entity, which most often is name given by the
+// Name component, except for corpses.
+func (es *ECS) GetName(i int) (s string) {
+	name := es.Name[i]
+	if es.Dead(i) {
+		name = "corpse"
+	}
+	return name
+}
+
 // renderOrder is a type representing the priority of an entity rendering.
 type renderOrder int
 
@@ -132,18 +166,16 @@ func (es *ECS) RenderOrder(i int) (ro renderOrder) {
 		} else {
 			ro = ROActor
 		}
+	case *Consumable:
+		ro = ROItem
 	}
 	return ro
 }
 
 // Entity represents an object or creature on the map.
-type Entity interface {
-	Rune() rune         // the character representing the entity
-	Color() gruid.Color // the character's color
-}
+type Entity interface{}
 
-// Player contains information relevant to the player. It implements the Entity
-// interface.
+// Player contains information relevant to the player.
 type Player struct {
 	FOV *rl.FOV // player's field of view
 }
@@ -158,23 +190,5 @@ func NewPlayer() *Player {
 	return player
 }
 
-func (p *Player) Rune() rune {
-	return '@'
-}
-
-func (p *Player) Color() gruid.Color {
-	return ColorPlayer
-}
-
-// Monster represents a monster. It implements the Entity interface.
-type Monster struct {
-	Char rune // monster's graphical representation
-}
-
-func (m *Monster) Rune() rune {
-	return m.Char
-}
-
-func (m *Monster) Color() gruid.Color {
-	return ColorMonster
-}
+// Monster represents a monster.
+type Monster struct{}
```

We define inventory handling functions in `game.go`, as well as placement of
items.

``` diff
diff --git a/game.go b/game.go
index db69a5f..1486be1 100644
--- a/game.go
+++ b/game.go
@@ -4,6 +4,7 @@
 package main
 
 import (
+	"errors"
 	"fmt"
 	"strings"
 
@@ -21,30 +22,36 @@ type game struct {
 
 // SpawnMonsters adds some monsters in the current map.
 func (g *game) SpawnMonsters() {
-	const numberOfMonsters = 6
+	const numberOfMonsters = 12
 	for i := 0; i < numberOfMonsters; i++ {
 		m := &Monster{}
 		// We generate either an orc or a troll with 0.8 and 0.2
 		// probabilities respectively.
+		const (
+			orc = iota
+			troll
+		)
+		kind := orc
 		switch {
 		case g.Map.Rand.Intn(100) < 80:
-			m.Char = 'o'
 		default:
-			m.Char = 'T'
+			kind = troll
 		}
 		p := g.FreeFloorTile()
 		i := g.ECS.AddEntity(m, p)
-		switch m.Char {
-		case 'o':
+		switch kind {
+		case orc:
 			g.ECS.Fighter[i] = &fighter{
 				HP: 10, MaxHP: 10, Defense: 0, Power: 3,
 			}
 			g.ECS.Name[i] = "orc"
-		case 'T':
+			g.ECS.Style[i] = Style{Rune: 'o', Color: ColorMonster}
+		case troll:
 			g.ECS.Fighter[i] = &fighter{
 				HP: 16, MaxHP: 16, Defense: 1, Power: 4,
 			}
 			g.ECS.Name[i] = "troll"
+			g.ECS.Style[i] = Style{Rune: 'T', Color: ColorMonster}
 		}
 		g.ECS.AI[i] = &AI{}
 	}
@@ -80,7 +87,7 @@ func (g *game) EndTurn() {
 func (g *game) UpdateFOV() {
 	player := g.ECS.Player()
 	// player position
-	pp := g.ECS.Positions[g.ECS.PlayerID]
+	pp := g.ECS.PP()
 	// We shift the FOV's Range so that it will be centered on the new
 	// player's position.
 	rg := gruid.NewRange(-maxLOS, -maxLOS, maxLOS+1, maxLOS+1)
@@ -105,7 +112,7 @@ func (g *game) UpdateFOV() {
 // current 4-way movement. With 8-way movement, the natural distance choice
 // would be the Chebyshev one.
 func (g *game) InFOV(p gruid.Point) bool {
-	pp := g.ECS.Positions[g.ECS.PlayerID]
+	pp := g.ECS.PP()
 	return g.ECS.Player().FOV.Visible(p) &&
 		paths.DistanceManhattan(pp, p) <= maxLOS
 }
@@ -127,3 +134,68 @@ func (g *game) BumpAttack(i, j int) {
 		g.Logf("%s but does no damage", color, attackDesc)
 	}
 }
+
+// PlaceItems adds items in the current map.
+func (g *game) PlaceItems() {
+	const numberOfPotions = 5
+	for i := 0; i < numberOfPotions; i++ {
+		p := g.FreeFloorTile()
+		id := g.ECS.AddEntity(&HealingPotion{Amount: 4}, p)
+		g.ECS.Name[id] = "health potion"
+		g.ECS.Style[id] = Style{Rune: '!', Color: ColorConsumable}
+	}
+}
+
+const ErrNoShow = "ErrNoShow"
+
+// IventoryAdd adds an item to the player's inventory, if there is room. It
+// returns an error if the item could not be added.
+func (g *game) InventoryAdd(actor, i int) error {
+	const maxSize = 26
+	switch g.ECS.Entities[i].(type) {
+	case Consumable:
+		inv := g.ECS.Inventory[actor]
+		if len(inv.Items) >= maxSize {
+			return errors.New("Inventory is full.")
+		}
+		inv.Items = append(inv.Items, i)
+		delete(g.ECS.Positions, i)
+		return nil
+	}
+	return errors.New(ErrNoShow)
+}
+
+// Drop an item from the inventory.
+func (g *game) InventoryRemove(actor, n int) error {
+	inv := g.ECS.Inventory[actor]
+	if len(inv.Items) <= n {
+		return errors.New("Empty slot.")
+	}
+	i := inv.Items[n]
+	inv.Items[n] = inv.Items[len(inv.Items)-1]
+	inv.Items = inv.Items[:len(inv.Items)-1]
+	g.ECS.Positions[i] = g.ECS.PP()
+	return nil
+}
+
+// InventoryActivate uses a given item from the inventory.
+func (g *game) InventoryActivate(actor, n int) error {
+	inv := g.ECS.Inventory[actor]
+	if len(inv.Items) <= n {
+		return errors.New("Empty slot.")
+	}
+	i := inv.Items[n]
+	switch e := g.ECS.Entities[i].(type) {
+	case Consumable:
+		err := e.Activate(g, itemAction{Actor: actor})
+		if err != nil {
+			return err
+		}
+	}
+	// Put the last item on the previous one: this could be improved,
+	// sorting elements in a certain way, or moving elements as necessary
+	// to preserve current order.
+	inv.Items[n] = inv.Items[len(inv.Items)-1]
+	inv.Items = inv.Items[:len(inv.Items)-1]
+	return nil
+}
```

The rest of changes are quite straightforward : defining the UI elements and
actions that make use of all this preceding work. In particular, new actions
are defined, as well as new UI modes (when selecting items to use/drop).

``` diff
diff --git a/actions.go b/actions.go
index 723ed81..91b323b 100644
--- a/actions.go
+++ b/actions.go
@@ -19,6 +19,9 @@ type actionType int
 const (
 	NoAction           actionType = iota
 	ActionBump                    // bump request (attack or movement)
+	ActionDrop                    // menu to drop an inventory item
+	ActionInventory               // inventory menu to use an item
+	ActionPickup                  // pickup an item on the ground
 	ActionWait                    // wait a turn
 	ActionQuit                    // quit the game
 	ActionViewMessages            // view history messages
@@ -28,8 +31,16 @@ const (
 func (m *model) handleAction() gruid.Effect {
 	switch m.action.Type {
 	case ActionBump:
-		np := m.game.ECS.Positions[m.game.ECS.PlayerID].Add(m.action.Delta)
+		np := m.game.ECS.PP().Add(m.action.Delta)
 		m.game.Bump(np)
+	case ActionDrop:
+		m.OpenInventory("Drop item")
+		m.mode = modeInventoryDrop
+	case ActionInventory:
+		m.OpenInventory("Use item")
+		m.mode = modeInventoryActivate
+	case ActionPickup:
+		m.game.PickupItem()
 	case ActionWait:
 		m.game.EndTurn()
 	case ActionQuit:
@@ -71,3 +82,51 @@ func (g *game) Bump(to gruid.Point) {
 	g.ECS.MovePlayer(to)
 	g.EndTurn()
 }
+
+// PickupItem takes an item on the floor.
+func (g *game) PickupItem() {
+	pp := g.ECS.PP()
+	for i, p := range g.ECS.Positions {
+		if p != pp {
+			// Skip entities whose position is diffferent than the
+			// player's.
+			continue
+		}
+		err := g.InventoryAdd(g.ECS.PlayerID, i)
+		if err != nil {
+			if err.Error() == ErrNoShow {
+				// Happens for example if the current entity is
+				// not a consumable.
+				continue
+			}
+			g.Logf("Could not pickup: %v", ColorLogSpecial, err)
+			return
+		}
+		g.Logf("You pickup %v", ColorLogItemUse, g.ECS.Name[i])
+		g.EndTurn()
+		return
+	}
+}
+
+// OpenInventory opens the inventory and allows the player to select an item.
+func (m *model) OpenInventory(title string) {
+	inv := m.game.ECS.Inventory[m.game.ECS.PlayerID]
+	// We build a list of entries.
+	entries := []ui.MenuEntry{}
+	r := 'a'
+	for _, it := range inv.Items {
+		name := m.game.ECS.Name[it]
+		entries = append(entries, ui.MenuEntry{
+			Text: ui.Text(string(r) + " - " + name),
+			// allow to use the character r to select the entry
+			Keys: []gruid.Key{gruid.Key(r)},
+		})
+		r++
+	}
+	// We create a new menu widget for the inventory window.
+	m.inventory = ui.NewMenu(ui.MenuConfig{
+		Grid:    gruid.NewGrid(40, MapHeight),
+		Box:     &ui.Box{Title: ui.Text(title)},
+		Entries: entries,
+	})
+}
diff --git a/ai.go b/ai.go
index 0d0c2df..46869b2 100644
--- a/ai.go
+++ b/ai.go
@@ -18,7 +18,7 @@ func (g *game) HandleMonsterTurn(i int) {
 	p := g.ECS.Positions[i]
 	ai := g.ECS.AI[i]
 	aip := &aiPath{g: g}
-	pp := g.ECS.Positions[g.ECS.PlayerID]
+	pp := g.ECS.PP()
 	if paths.DistanceManhattan(p, pp) == 1 {
 		// If the monster is adjacent to the player, attack.
 		g.BumpAttack(i, g.ECS.PlayerID)
diff --git a/model.go b/model.go
index 55771f7..ef65184 100644
--- a/model.go
+++ b/model.go
@@ -16,15 +16,16 @@ import (
 
 // model represents our main application's state.
 type model struct {
-	grid     gruid.Grid  // drawing grid
-	game     *game       // game state
-	action   action      // UI action
-	mode     mode        // UI mode
-	log      *ui.Label   // label for log
-	status   *ui.Label   // label for status
-	desc     *ui.Label   // label for position description
-	viewer   *ui.Pager   // message's history viewer
-	mousePos gruid.Point // mouse position
+	grid      gruid.Grid  // drawing grid
+	game      *game       // game state
+	action    action      // UI action
+	mode      mode        // UI mode
+	log       *ui.Label   // label for log
+	status    *ui.Label   // label for status
+	desc      *ui.Label   // label for position description
+	inventory *ui.Menu    // inventory menu
+	viewer    *ui.Pager   // message's history viewer
+	mousePos  gruid.Point // mouse position
 }
 
 // mode describes distinct kinds of modes for the UI
@@ -33,6 +34,8 @@ type mode int
 const (
 	modeNormal mode = iota
 	modeEnd         // win or death (currently only death)
+	modeInventoryActivate
+	modeInventoryDrop
 	modeMessageViewer
 )
 
@@ -57,6 +60,9 @@ func (m *model) Update(msg gruid.Msg) gruid.Effect {
 			m.mode = modeNormal
 		}
 		return nil
+	case modeInventoryActivate, modeInventoryDrop:
+		m.updateInventory(msg)
+		return nil
 	}
 	switch msg := msg.(type) {
 	case gruid.MsgInit:
@@ -77,10 +83,14 @@ func (m *model) Update(msg gruid.Msg) gruid.Effect {
 		m.game.ECS.Fighter[m.game.ECS.PlayerID] = &fighter{
 			HP: 30, MaxHP: 30, Power: 5, Defense: 2,
 		}
+		m.game.ECS.Style[m.game.ECS.PlayerID] = Style{Rune: '@', Color: ColorPlayer}
 		m.game.ECS.Name[m.game.ECS.PlayerID] = "player"
+		m.game.ECS.Inventory[m.game.ECS.PlayerID] = &Inventory{}
 		m.game.UpdateFOV()
 		// Add some monsters
 		m.game.SpawnMonsters()
+		// Add items
+		m.game.PlaceItems()
 	case gruid.MsgKeyDown:
 		// Update action information on key down.
 		m.updateMsgKeyDown(msg)
@@ -93,6 +103,36 @@ func (m *model) Update(msg gruid.Msg) gruid.Effect {
 	return m.handleAction()
 }
 
+// updateInventory handles input messages when the inventory window is open.
+func (m *model) updateInventory(msg gruid.Msg) {
+	// We call the Update function of the menu widget, so that we can
+	// inspect information about user activity on the menu.
+	m.inventory.Update(msg)
+	switch m.inventory.Action() {
+	case ui.MenuQuit:
+		// The user requested to quit the menu.
+		m.mode = modeNormal
+		return
+	case ui.MenuInvoke:
+		// The user invoked a particular entry of the menu (either by
+		// using enter or clicking on it).
+		n := m.inventory.Active()
+		var err error
+		switch m.mode {
+		case modeInventoryDrop:
+			err = m.game.InventoryRemove(m.game.ECS.PlayerID, n)
+		case modeInventoryActivate:
+			err = m.game.InventoryActivate(m.game.ECS.PlayerID, n)
+		}
+		if err != nil {
+			m.game.Logf("%v", ColorLogSpecial, err)
+		} else {
+			m.game.EndTurn()
+		}
+		m.mode = modeNormal
+	}
+}
+
 func (m *model) updateMsgKeyDown(msg gruid.MsgKeyDown) {
 	pdelta := gruid.Point{}
 	switch msg.Key {
@@ -110,6 +150,12 @@ func (m *model) updateMsgKeyDown(msg gruid.MsgKeyDown) {
 		m.action = action{Type: ActionQuit}
 	case "m":
 		m.action = action{Type: ActionViewMessages}
+	case "i":
+		m.action = action{Type: ActionInventory}
+	case "d":
+		m.action = action{Type: ActionDrop}
+	case "g":
+		m.action = action{Type: ActionPickup}
 	}
 }
 
@@ -120,21 +166,27 @@ const (
 	ColorPlayer
 	ColorMonster
 	ColorLogPlayerAttack
+	ColorLogItemUse
 	ColorLogMonsterAttack
 	ColorLogSpecial
 	ColorStatusHealthy
 	ColorStatusWounded
+	ColorConsumable
 )
 
 // Draw implements gruid.Model.Draw. It draws a simple map that spans the whole
 // grid.
 func (m *model) Draw() gruid.Grid {
-	if m.mode == modeMessageViewer {
+	mapgrid := m.grid.Slice(m.grid.Range().Shift(0, 2, 0, -1))
+	switch m.mode {
+	case modeMessageViewer:
 		m.grid.Copy(m.viewer.Draw())
 		return m.grid
+	case modeInventoryDrop, modeInventoryActivate:
+		mapgrid.Copy(m.inventory.Draw())
+		return m.grid
 	}
 	m.grid.Fill(gruid.Cell{Rune: ' '})
-	mapgrid := m.grid.Slice(m.grid.Range().Shift(0, 2, 0, -1))
 	g := m.game
 	// We draw the map tiles.
 	it := g.Map.Grid.Iterator()
@@ -163,7 +215,7 @@ func (m *model) Draw() gruid.Grid {
 			continue
 		}
 		c := mapgrid.At(p)
-		c.Rune, c.Style.Fg = g.ECS.Style(i)
+		c.Rune, c.Style.Fg = g.ECS.GetStyle(i)
 		mapgrid.Set(p, c)
 		// NOTE: We retrieved current cell at e.Pos() to preserve
 		// background (in FOV or not).
@@ -217,13 +269,9 @@ func (m *model) DrawNames(gd gruid.Grid) {
 		if q != p || !m.game.InFOV(q) {
 			continue
 		}
-		name, ok := m.game.ECS.Name[i]
-		if ok {
-			if m.game.ECS.Alive(i) {
-				names = append(names, name)
-			} else {
-				names = append(names, "corpse")
-			}
+		name := m.game.ECS.GetName(i)
+		if name != "" {
+			names = append(names, name)
 		}
 	}
 	if len(names) == 0 {
diff --git a/tiles.go b/tiles.go
index 9a945c0..482d13e 100644
--- a/tiles.go
+++ b/tiles.go
@@ -35,7 +35,7 @@ func (t *TileDrawer) GetImage(c gruid.Cell) image.Image {
 		bg = image.NewUniform(color.RGBA{0x18, 0x49, 0x56, 255})
 	}
 	switch c.Style.Fg {
-	case ColorPlayer:
+	case ColorPlayer, ColorLogItemUse:
 		fg = image.NewUniform(color.RGBA{0x46, 0x95, 0xf7, 255})
 	case ColorMonster:
 		fg = image.NewUniform(color.RGBA{0xfa, 0x57, 0x50, 255})
@@ -45,6 +45,8 @@ func (t *TileDrawer) GetImage(c gruid.Cell) image.Image {
 		fg = image.NewUniform(color.RGBA{0xed, 0x86, 0x49, 255})
 	case ColorLogSpecial:
 		fg = image.NewUniform(color.RGBA{0xf2, 0x75, 0xbe, 255})
+	case ColorConsumable:
+		fg = image.NewUniform(color.RGBA{0xdb, 0xb3, 0x2d, 255})
 	}
 	// We return an image with the given rune drawn using the previously
 	// defined foreground and background colors.
```

* * *

[Next Part](https://github.com/anaseto/gruid-rltuto/tree/part-9)
