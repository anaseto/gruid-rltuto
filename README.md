# Gruid Go Roguelike Tutorial

This tutorial follows the overall structure of the [TCOD Python
Tutorial](http://rogueliketutorials.com/tutorials/tcod/v2), but makes use of
the [Go programming language](https://golang.org/) and the
[gruid](https://github.com/anaseto/gruid) roguelike game framework, instead of
TCOD.

[Table of Contents](https://github.com/anaseto/gruid-rltuto)

# Part 9 - Ranged Scrolls and Targeting

In this part we add a few new items : scrolls of lightning (no targeting),
confusion (wit targeting), and fireball (targeting an area).

Most of the changess occur in `items.go`, where we define the new item
entities, their `Activate` method. We introduce a new interface that items
requiring targeting in order to be used have to satisfy. For now, there is only
one method: `TargetingRadius`. It gives the radius of the effect (zero means
just one cell).

``` diff
diff --git a/items.go b/items.go
index d41e9ea..d5b4137 100644
--- a/items.go
+++ b/items.go
@@ -7,6 +7,7 @@ import (
 	"fmt"
 
 	"github.com/anaseto/gruid"
+	"github.com/anaseto/gruid/paths"
 )
 
 // Consumable describes a consumable item, like a potion.
@@ -44,3 +45,107 @@ func (pt *HealingPotion) Activate(g *game, a itemAction) error {
 	}
 	return nil
 }
+
+// LightningScroll is an item that can be invoked to strike the closest enemy
+// within a particular range.
+type LightningScroll struct {
+	Range  int
+	Damage int
+}
+
+func (sc *LightningScroll) Activate(g *game, a itemAction) error {
+	target := -1
+	minDist := sc.Range + 1
+	for i := range g.ECS.Fighter {
+		p := g.ECS.Positions[i]
+		if i == a.Actor || g.ECS.Dead(i) || !g.InFOV(p) {
+			continue
+		}
+		dist := paths.DistanceManhattan(p, g.ECS.Positions[a.Actor])
+		if dist < minDist {
+			target = i
+			minDist = dist
+		}
+	}
+	if target < 0 {
+		return errors.New("No enemy within range.")
+	}
+	g.Logf("A lightning bolt strikes %v.", ColorLogItemUse, g.ECS.GetName(target))
+	g.ECS.Fighter[target].HP -= sc.Damage
+	return nil
+}
+
+// Targetter describes consumables (or other kind of activables) that need
+// a target in order to be used.
+type Targetter interface {
+	// TargetingRadius returns the radius of the affected area around the
+	// target.
+	TargetingRadius() int
+}
+
+// ConfusionScroll is an item that can be invoked to confuse an enemy.
+type ConfusionScroll struct {
+	Turns int
+}
+
+func (sc *ConfusionScroll) Activate(g *game, a itemAction) error {
+	if a.Target == nil {
+		return errors.New("You have to chose a target.")
+	}
+	p := *a.Target
+	if !g.InFOV(p) {
+		return errors.New("You cannot target what you cannot see.")
+	}
+	if p == g.ECS.PP() {
+		return errors.New("You cannot confuse yourself.")
+	}
+	i := g.ECS.MonsterAt(p)
+	if i <= 0 || !g.ECS.Alive(i) {
+		return errors.New("You have to target a monster.")
+	}
+	g.Logf("%s looks confused (scroll).", ColorLogPlayerAttack, g.ECS.GetName(i))
+	g.ECS.PutStatus(i, StatusConfused, sc.Turns)
+	return nil
+}
+
+func (sc *ConfusionScroll) TargetingRadius() int { return 0 }
+
+// FireballScroll is an item that can be invoked to produce a flame explosion
+// in an area around a target position.
+type FireballScroll struct {
+	Damage int
+	Radius int
+}
+
+func (sc *FireballScroll) Activate(g *game, a itemAction) error {
+	if a.Target == nil {
+		return errors.New("You have to chose a target.")
+	}
+	p := *a.Target
+	if !g.InFOV(p) {
+		return errors.New("You cannot target what you cannot see.")
+	}
+	hits := 0
+	// NOTE: this could be made more complicated by checking whether there
+	// are monsters in the way. For now, it's a fireball that goes up and
+	// then down and explodes on reaching the target!
+	for i, fi := range g.ECS.Fighter {
+		if g.ECS.Dead(i) {
+			continue
+		}
+		q := g.ECS.Positions[i]
+		dist := paths.DistanceManhattan(q, p)
+		if dist > sc.Radius {
+			continue
+		}
+		g.Logf("%v is engulfed in flames.", ColorLogPlayerAttack, g.ECS.GetName(i))
+		fi.HP -= sc.Damage
+		hits++
+	}
+	if hits <= 0 {
+		return errors.New("There are no targets in the radius.")
+	}
+	return nil
+}
+
+func (sc *FireballScroll) TargetingRadius() int { return sc.Radius }
```

We also introduce a new component for monster (or player) statutes. In this
case, a status for confusion.

``` diff
diff --git a/components.go b/components.go
index 7fa20f0..aa3b43b 100644
--- a/components.go
+++ b/components.go
@@ -40,3 +40,29 @@ type Style struct {
 type Inventory struct {
 	Items []int
 }
+
+// status describes different kind of statuses.
+type status int
+
+const (
+	StatusConfused status = iota
+)
+
+// Statuses maps ongoing statuses to their remaining turns.
+type Statuses map[status]int
+
+// NextTurn updates statuses for the next turn.
+func (sts Statuses) NextTurn() {
+	for st, turns := range sts {
+		if turns == 0 {
+			delete(sts, st)
+			continue
+		}
+		sts[st]--
+	}
+}
+
+// Put puts on a particular status for a given number of turns.
+func (sts Statuses) Put(st status, turns int) {
+	sts[st] = turns
+}
diff --git a/entity.go b/entity.go
```

The rest of the changes are related to UI : new mode for selecting a target, as
well as a new `targeting` struct for the model with targeting information.
Because it's similar, we add also a mode to examine the map using the keyboard.
We also update the `PlaceItems` method to generate the new kinds of items.

``` diff
diff --git a/actions.go b/actions.go
index 91b323b..4390653 100644
--- a/actions.go
+++ b/actions.go
@@ -25,6 +25,7 @@ const (
 	ActionWait                    // wait a turn
 	ActionQuit                    // quit the game
 	ActionViewMessages            // view history messages
+	ActionExamine                 // examine map
 )
 
 // handleAction updates the model in response to current recorded last action.
@@ -56,6 +57,9 @@ func (m *model) handleAction() gruid.Effect {
 			lines = append(lines, ui.NewStyledText(e.String(), st))
 		}
 		m.viewer.SetLines(lines)
+	case ActionExamine:
+		m.mode = modeExamination
+		m.targ.pos = m.game.ECS.PP().Shift(0, LogLines)
 	}
 	if m.game.ECS.PlayerDied() {
 		m.game.Logf("You died -- press “q” or escape to quit", ColorLogSpecial)
@@ -71,7 +75,7 @@ func (g *game) Bump(to gruid.Point) {
 	if !g.Map.Walkable(to) {
 		return
 	}
-	if i, _ := g.ECS.MonsterAt(to); g.ECS.Alive(i) {
+	if i := g.ECS.MonsterAt(to); g.ECS.Alive(i) {
 		// We show a message to standard error. Later in the tutorial,
 		// we'll put a message in the UI instead.
 		g.BumpAttack(g.ECS.PlayerID, i)
diff --git a/ai.go b/ai.go
index 46869b2..91f0f30 100644
--- a/ai.go
+++ b/ai.go
@@ -3,6 +3,8 @@
 package main
 
 import (
+	"math/rand"
+
 	"github.com/anaseto/gruid"
 	"github.com/anaseto/gruid/paths"
 )
@@ -15,6 +17,10 @@ func (g *game) HandleMonsterTurn(i int) {
 		// Do nothing if the entity corresponds to a dead monster.
 		return
 	}
+	if g.ECS.Status(i, StatusConfused) {
+		g.HandleConfusedMonster(i)
+		return
+	}
 	p := g.ECS.Positions[i]
 	ai := g.ECS.AI[i]
 	aip := &aiPath{g: g}
@@ -42,6 +48,24 @@ func (g *game) HandleMonsterTurn(i int) {
 	g.AIMove(i)
 }
 
+// HandleConfusedMonster handles the behavior of a confused monster. It simply
+// tries to bump into a random direction.
+func (g *game) HandleConfusedMonster(i int) {
+	p := g.ECS.Positions[i]
+	p.X += -1 + 2*rand.Intn(2)
+	p.Y += -1 + 2*rand.Intn(2)
+	if !p.In(g.Map.Grid.Range()) {
+		return
+	}
+	if p == g.ECS.PP() {
+		g.BumpAttack(i, g.ECS.PlayerID)
+		return
+	}
+	if g.Map.Walkable(p) && g.ECS.NoBlockingEntityAt(p) {
+		g.ECS.MoveEntity(i, p)
+	}
+}
+
 // AIMove moves a monster to the next position, if there is no blocking entity
 // at the destination. It assumes the destination is walkable.
 func (g *game) AIMove(i int) {
index 8bbe8a0..452a854 100644
--- a/entity.go
+++ b/entity.go
@@ -22,6 +22,7 @@ type ECS struct {
 	Name      map[int]string     // name component
 	Style     map[int]Style      // default style component
 	Inventory map[int]*Inventory // inventory component
+	Statuses  map[int]Statuses   // statuses (confused, etc.)
 }
 
 // NewECS returns an initialized ECS structure.
@@ -34,6 +35,7 @@ func NewECS() *ECS {
 		Name:      map[int]string{},
 		Style:     map[int]Style{},
 		Inventory: map[int]*Inventory{},
+		Statuses:  map[int]Statuses{},
 		NextID:    0,
 	}
 }
@@ -47,6 +49,14 @@ func (es *ECS) AddEntity(e Entity, p gruid.Point) int {
 	return id
 }
 
+// AddItem is a shorthand for adding item entities on the map.
+func (es *ECS) AddItem(e Entity, p gruid.Point, name string, r rune) int {
+	id := es.AddEntity(e, p)
+	es.Name[id] = name
+	es.Style[id] = Style{Rune: r, Color: ColorConsumable}
+	return id
+}
+
 // RemoveEntity removes an entity, given its identifier.
 func (es *ECS) RemoveEntity(i int) {
 	delete(es.Entities, i)
@@ -81,24 +91,24 @@ func (es *ECS) PP() gruid.Point {
 
 // MonsterAt returns the Monster at p along with its index, if any, or nil if
 // there is no monster at p.
-func (es *ECS) MonsterAt(p gruid.Point) (int, *Monster) {
+func (es *ECS) MonsterAt(p gruid.Point) int {
 	for i, q := range es.Positions {
 		if p != q || !es.Alive(i) {
 			continue
 		}
 		e := es.Entities[i]
-		switch e := e.(type) {
+		switch e.(type) {
 		case *Monster:
-			return i, e
+			return i
 		}
 	}
-	return -1, nil
+	return -1
 }
 
 // NoBlockingEntityAt returns true if there is no blocking entity at p (no
 // player nor monsters in this tutorial).
 func (es *ECS) NoBlockingEntityAt(p gruid.Point) bool {
-	i, _ := es.MonsterAt(p)
+	i := es.MonsterAt(p)
 	return es.PP() != p && !es.Alive(i)
 }
 
@@ -142,6 +152,29 @@ func (es *ECS) GetName(i int) (s string) {
 	return name
 }
 
+// StatusesNextTurn updates the remaining turns of entities' statuses.
+func (es *ECS) StatusesNextTurn() {
+	for _, sts := range es.Statuses {
+		sts.NextTurn()
+	}
+}
+
+// PutStatus puts on a particular status for a given entity for a certain
+// number of turns.
+func (es *ECS) PutStatus(i int, st status, turns int) {
+	if es.Statuses[i] == nil {
+		es.Statuses[i] = map[status]int{}
+	}
+	sts := es.Statuses[i]
+	sts.Put(st, turns)
+}
+
+// Status checks whether an entity has a particular status effect.
+func (es *ECS) Status(i int, st status) bool {
+	_, ok := es.Statuses[i][st]
+	return ok
+}
+
 // renderOrder is a type representing the priority of an entity rendering.
 type renderOrder int
 
diff --git a/game.go b/game.go
index 1486be1..3ccf60a 100644
--- a/game.go
+++ b/game.go
@@ -81,6 +81,7 @@ func (g *game) EndTurn() {
 			g.HandleMonsterTurn(i)
 		}
 	}
+	g.ECS.StatusesNextTurn()
 }
 
 // UpdateFOV updates the field of view.
@@ -137,12 +138,21 @@ func (g *game) BumpAttack(i, j int) {
 
 // PlaceItems adds items in the current map.
 func (g *game) PlaceItems() {
-	const numberOfPotions = 5
-	for i := 0; i < numberOfPotions; i++ {
+	const numberOfItems = 5
+	for i := 0; i < numberOfItems; i++ {
 		p := g.FreeFloorTile()
-		id := g.ECS.AddEntity(&HealingPotion{Amount: 4}, p)
-		g.ECS.Name[id] = "health potion"
-		g.ECS.Style[id] = Style{Rune: '!', Color: ColorConsumable}
+		r := g.Map.Rand.Float64()
+		switch {
+		case r < 0.7:
+			g.ECS.AddItem(&HealingPotion{Amount: 4}, p, "health potion", '!')
+		case r < 0.8:
+			g.ECS.AddItem(&ConfusionScroll{Turns: 10}, p, "confusion scroll", '?')
+		case r < 0.9:
+			g.ECS.AddItem(&FireballScroll{Damage: 12, Radius: 3}, p, "fireball scroll", '?')
+		default:
+			g.ECS.AddItem(&LightningScroll{Range: 5, Damage: 20},
+				p, "lightning scroll", '?')
+		}
 	}
 }
 
@@ -180,6 +190,12 @@ func (g *game) InventoryRemove(actor, n int) error {
 
 // InventoryActivate uses a given item from the inventory.
 func (g *game) InventoryActivate(actor, n int) error {
+	return g.InventoryActivateWithTarget(actor, n, nil)
+}
+
+// InventoryActivateWithTarget uses a given item from the inventory, with
+// an optional target.
+func (g *game) InventoryActivateWithTarget(actor, n int, targ *gruid.Point) error {
 	inv := g.ECS.Inventory[actor]
 	if len(inv.Items) <= n {
 		return errors.New("Empty slot.")
@@ -187,7 +203,7 @@ func (g *game) InventoryActivate(actor, n int) error {
 	i := inv.Items[n]
 	switch e := g.ECS.Entities[i].(type) {
 	case Consumable:
-		err := e.Activate(g, itemAction{Actor: actor})
+		err := e.Activate(g, itemAction{Actor: actor, Target: targ})
 		if err != nil {
 			return err
 		}
@@ -199,3 +215,18 @@ func (g *game) InventoryActivate(actor, n int) error {
 	inv.Items = inv.Items[:len(inv.Items)-1]
 	return nil
 }
+
+// NeedsTargeting checks whether using the n-th item requires targeting,
+// returning its radius (-1 if no targeting).
+func (g *game) TargetingRadius(n int) int {
+	inv := g.ECS.Inventory[g.ECS.PlayerID]
+	if len(inv.Items) <= n {
+		return -1
+	}
+	i := inv.Items[n]
+	switch e := g.ECS.Entities[i].(type) {
+	case Targetter:
+		return e.TargetingRadius()
+	}
+	return -1
+}
diff --git a/main.go b/main.go
index 7da0d56..08c41a6 100644
--- a/main.go
+++ b/main.go
@@ -12,8 +12,9 @@ import (
 const (
 	UIWidth   = 80
 	UIHeight  = 24
+	LogLines  = 2
 	MapWidth  = UIWidth
-	MapHeight = UIHeight - 3
+	MapHeight = UIHeight - 1 - LogLines
 )
 
 func main() {
diff --git a/model.go b/model.go
index ef65184..d06362e 100644
--- a/model.go
+++ b/model.go
@@ -16,19 +16,29 @@ import (
 
 // model represents our main application's state.
 type model struct {
-	grid      gruid.Grid  // drawing grid
-	game      *game       // game state
-	action    action      // UI action
-	mode      mode        // UI mode
-	log       *ui.Label   // label for log
-	status    *ui.Label   // label for status
-	desc      *ui.Label   // label for position description
-	inventory *ui.Menu    // inventory menu
-	viewer    *ui.Pager   // message's history viewer
-	mousePos  gruid.Point // mouse position
+	grid      gruid.Grid // drawing grid
+	game      *game      // game state
+	action    action     // UI action
+	mode      mode       // UI mode
+	log       *ui.Label  // label for log
+	status    *ui.Label  // label for status
+	desc      *ui.Label  // label for position description
+	inventory *ui.Menu   // inventory menu
+	viewer    *ui.Pager  // message's history viewer
+	targ      targeting  // targeting information
 }
 
-// mode describes distinct kinds of modes for the UI
+// targeting describes information related to examination or selection of
+// particular positions in the map.
+type targeting struct {
+	pos    gruid.Point
+	item   int // item to use after selecting target
+	radius int
+}
+
+// mode describes distinct kinds of modes for the UI. It is used to send user
+// input messages to different handlers (inventory window, map, message viewer,
+// etc.), depending on the current mode.
 type mode int
 
 const (
@@ -37,6 +47,8 @@ const (
 	modeInventoryActivate
 	modeInventoryDrop
 	modeMessageViewer
+	modeTargeting   // targeting mode (item use)
+	modeExamination // keyboad map examination mode
 )
 
 // Update implements gruid.Model.Update. It handles keyboard and mouse input
@@ -63,6 +75,9 @@ func (m *model) Update(msg gruid.Msg) gruid.Effect {
 	case modeInventoryActivate, modeInventoryDrop:
 		m.updateInventory(msg)
 		return nil
+	case modeTargeting, modeExamination:
+		m.updateTargeting(msg)
+		return nil
 	}
 	switch msg := msg.(type) {
 	case gruid.MsgInit:
@@ -96,13 +111,65 @@ func (m *model) Update(msg gruid.Msg) gruid.Effect {
 		m.updateMsgKeyDown(msg)
 	case gruid.MsgMouse:
 		if msg.Action == gruid.MouseMove {
-			m.mousePos = msg.P
+			m.targ.pos = msg.P
 		}
 	}
 	// Handle action (if any).
 	return m.handleAction()
 }
 
+// updateTargeting updates targeting information in response to user input
+// messages.
+func (m *model) updateTargeting(msg gruid.Msg) {
+	maprg := gruid.NewRange(0, LogLines, UIWidth, UIHeight-1)
+	if !m.targ.pos.In(maprg) {
+		m.targ.pos = m.game.ECS.PP().Add(maprg.Min)
+	}
+	p := m.targ.pos.Sub(maprg.Min)
+	switch msg := msg.(type) {
+	case gruid.MsgKeyDown:
+		switch msg.Key {
+		case gruid.KeyArrowLeft, "h":
+			p = p.Shift(-1, 0)
+		case gruid.KeyArrowDown, "j":
+			p = p.Shift(0, 1)
+		case gruid.KeyArrowUp, "k":
+			p = p.Shift(0, -1)
+		case gruid.KeyArrowRight, "l":
+			p = p.Shift(1, 0)
+		case gruid.KeyEnter, ".":
+			if m.mode == modeExamination {
+				break
+			}
+			m.activateTarget(p)
+			return
+		case gruid.KeyEscape, "q":
+			m.targ = targeting{}
+			m.mode = modeNormal
+			return
+		}
+		m.targ.pos = p.Add(maprg.Min)
+	case gruid.MsgMouse:
+		switch msg.Action {
+		case gruid.MouseMove:
+			m.targ.pos = msg.P
+		case gruid.MouseMain:
+			m.activateTarget(p)
+		}
+	}
+}
+
+func (m *model) activateTarget(p gruid.Point) {
+	err := m.game.InventoryActivateWithTarget(m.game.ECS.PlayerID, m.targ.item, &p)
+	if err != nil {
+		m.game.Logf("%v", ColorLogSpecial, err)
+	} else {
+		m.game.EndTurn()
+	}
+	m.mode = modeNormal
+	m.targ = targeting{}
+}
+
 // updateInventory handles input messages when the inventory window is open.
 func (m *model) updateInventory(msg gruid.Msg) {
 	// We call the Update function of the menu widget, so that we can
@@ -122,6 +189,15 @@ func (m *model) updateInventory(msg gruid.Msg) {
 		case modeInventoryDrop:
 			err = m.game.InventoryRemove(m.game.ECS.PlayerID, n)
 		case modeInventoryActivate:
+			if radius := m.game.TargetingRadius(n); radius >= 0 {
+				m.targ = targeting{
+					item:   n,
+					pos:    m.game.ECS.PP().Shift(0, LogLines),
+					radius: radius,
+				}
+				m.mode = modeTargeting
+				return
+			}
 			err = m.game.InventoryActivate(m.game.ECS.PlayerID, n)
 		}
 		if err != nil {
@@ -135,6 +211,7 @@ func (m *model) updateInventory(msg gruid.Msg) {
 
 func (m *model) updateMsgKeyDown(msg gruid.MsgKeyDown) {
 	pdelta := gruid.Point{}
+	m.targ.pos = gruid.Point{}
 	switch msg.Key {
 	case gruid.KeyArrowLeft, "h":
 		m.action = action{Type: ActionBump, Delta: pdelta.Shift(-1, 0)}
@@ -156,6 +233,8 @@ func (m *model) updateMsgKeyDown(msg gruid.MsgKeyDown) {
 		m.action = action{Type: ActionDrop}
 	case "g":
 		m.action = action{Type: ActionPickup}
+	case "x":
+		m.action = action{Type: ActionExamine}
 	}
 }
 
@@ -174,10 +253,14 @@ const (
 	ColorConsumable
 )
 
+const (
+	AttrReverse = 1 << iota
+)
+
 // Draw implements gruid.Model.Draw. It draws a simple map that spans the whole
 // grid.
 func (m *model) Draw() gruid.Grid {
-	mapgrid := m.grid.Slice(m.grid.Range().Shift(0, 2, 0, -1))
+	mapgrid := m.grid.Slice(m.grid.Range().Shift(0, LogLines, 0, -1))
 	switch m.mode {
 	case modeMessageViewer:
 		m.grid.Copy(m.viewer.Draw())
@@ -221,7 +304,7 @@ func (m *model) Draw() gruid.Grid {
 		// background (in FOV or not).
 	}
 	m.DrawNames(mapgrid)
-	m.DrawLog(m.grid.Slice(m.grid.Range().Lines(0, 2)))
+	m.DrawLog(m.grid.Slice(m.grid.Range().Lines(0, LogLines)))
 	m.DrawStatus(m.grid.Slice(m.grid.Range().Line(m.grid.Size().Y - 1)))
 	return m.grid
 }
@@ -258,11 +341,19 @@ func (m *model) DrawStatus(gd gruid.Grid) {
 // DrawNames renders the names of the named entities at current mouse location
 // if it is in the map.
 func (m *model) DrawNames(gd gruid.Grid) {
-	maprg := gruid.NewRange(0, 2, UIWidth, UIHeight-1)
-	if !m.mousePos.In(maprg) {
+	maprg := gruid.NewRange(0, LogLines, UIWidth, UIHeight-1)
+	if !m.targ.pos.In(maprg) {
 		return
 	}
-	p := m.mousePos.Sub(gruid.Point{0, 2})
+	p := m.targ.pos.Sub(maprg.Min)
+	rad := m.targ.radius
+	rg := gruid.Range{Min: p.Sub(gruid.Point{rad, rad}), Max: p.Add(gruid.Point{rad + 1, rad + 1})}
+	rg = rg.Intersect(maprg.Sub(maprg.Min))
+	rg.Iter(func(q gruid.Point) {
+		c := gd.At(q)
+		c.Style.Attrs |= AttrReverse
+		gd.Set(q, c)
+	})
 	// We get the names of the entities at p.
 	names := []string{}
 	for i, q := range m.game.ECS.Positions {
@@ -284,7 +375,7 @@ func (m *model) DrawNames(gd gruid.Grid) {
 
 	text := strings.Join(names, ", ")
 	width := utf8.RuneCountInString(text) + 2
-	rg := gruid.NewRange(p.X+1, p.Y-1, p.X+1+width, p.Y+2)
+	rg = gruid.NewRange(p.X+1, p.Y-1, p.X+1+width, p.Y+2)
 	// we adjust a bit the box's placement in case it's on a edge.
 	if p.X+1+width >= UIWidth {
 		rg = rg.Shift(-1-width, 0, -1-width, 0)
diff --git a/tiles.go b/tiles.go
index 482d13e..b7580dc 100644
--- a/tiles.go
+++ b/tiles.go
@@ -48,6 +48,9 @@ func (t *TileDrawer) GetImage(c gruid.Cell) image.Image {
 	case ColorConsumable:
 		fg = image.NewUniform(color.RGBA{0xdb, 0xb3, 0x2d, 255})
 	}
+	if c.Style.Attrs&AttrReverse != 0 {
+		fg, bg = bg, fg
+	}
 	// We return an image with the given rune drawn using the previously
 	// defined foreground and background colors.
 	return t.drawer.Draw(c.Rune, fg, bg)
```
