# Gruid Go Roguelike Tutorial

This tutorial follows the overall structure of the [TCOD Python
Tutorial](http://rogueliketutorials.com/tutorials/tcod/v2), but makes use of
the [Go programming language](https://golang.org/) and the
[gruid](https://github.com/anaseto/gruid) roguelike game framework, instead of
TCOD.

[Table of Contents](https://github.com/anaseto/gruid-rltuto)

## Part 5 - Placing Enemies and kicking them (harmlessly)

In this part, we will place some monsters in the map, and allow the player to
bump (harmlessly) into them. We first create a new `Monster` type that
implements the `Entity` interface. We also define two utility functions that
return the monster at a given position in the map (if any), and whether there
is a blocking entity (player or monster) at a given position.

``` diff
diff --git a/entity.go b/entity.go
index f7f64b9..364a065 100644
--- a/entity.go
+++ b/entity.go
@@ -48,6 +48,28 @@ func (es *ECS) Player() *Player {
 	return es.Entities[es.PlayerID].(*Player)
 }
 
+// MonsterAt returns the Monster at p, if any, or nil if there is no monster at
+// p.
+func (es *ECS) MonsterAt(p gruid.Point) *Monster {
+	for i, q := range es.Positions {
+		if p != q {
+			continue
+		}
+		e := es.Entities[i]
+		switch e := e.(type) {
+		case *Monster:
+			return e
+		}
+	}
+	return nil
+}
+
+// NoBlockingEntityAt returns true if there is no blocking entity at p (no
+// player nor monsters in this tutorial).
+func (es *ECS) NoBlockingEntityAt(p gruid.Point) bool {
+	return es.Positions[es.PlayerID] != p && es.MonsterAt(p) == nil
+}
+
 // Entity represents an object or creature on the map.
 type Entity interface {
 	Rune() rune         // the character representing the entity
@@ -75,5 +97,20 @@ func (p *Player) Rune() rune {
 }
 
 func (p *Player) Color() gruid.Color {
-	return gruid.ColorDefault
+	return ColorPlayer
+}
+
+// Monster represents a monster. It implements the Entity interface. For now,
+// we simply give it a name and a rune for its graphical representation.
+type Monster struct {
+	Name string
+	Char rune
+}
+
+func (m *Monster) Rune() rune {
+	return m.Char
+}
+
+func (m *Monster) Color() gruid.Color {
+	return ColorMonster
 }
```

We create a new `game.go` file where we move the `game` type declaration and
write a `SpawnMonsters` function that will place monsters in the map.

``` go
// This file handles game related affairs that are not specific to entities or
// the map.

package main

import "github.com/anaseto/gruid"

// game represents information relevant the current game's state.
type game struct {
	ECS *ECS // entities present on the map
	Map *Map // the game map, made of tiles
}

// SpawnMonsters adds some monsters in the current map.
func (g *game) SpawnMonsters() {
	const numberOfMonsters = 6
	for i := 0; i < numberOfMonsters; i++ {
		m := &Monster{}
		// We generate either an orc or a troll with 0.8 and 0.2
		// probabilities respectively.
		switch {
		case g.Map.Rand.Intn(100) < 80:
			m.Name = "orc"
			m.Char = 'o'
		default:
			m.Name = "troll"
			m.Char = 'T'
		}
		p := g.FreeFloorTile()
		g.ECS.AddEntity(m, p)
	}
}

// FreeFloorTile returns a free floor tile in the map (it assumes it exists).
func (g *game) FreeFloorTile() gruid.Point {
	for {
		p := g.Map.RandomFloor()
		if g.ECS.NoBlockingEntityAt(p) {
			return p
		}
	}
}
```

We then update `actions.go` movement action. We rename the `ActionMovement`
action into `ActionBump`, and rename the `MovePlayer` method of `game` into
`Bump`. We use the previously defined `MonsterAt` to check whether there is a
monster or not at destination, and if there is a monster, we log a message. For
now, we write to standard error, so the message will appear on the console or
terminal. Later on in the tutorial, we will make space in the UI for text and
other information.

``` diff
diff --git a/actions.go b/actions.go
index ee49f6e..f945e7a 100644
--- a/actions.go
+++ b/actions.go
@@ -3,6 +3,8 @@
 package main
 
 import (
+	"log"
+
 	"github.com/anaseto/gruid"
 	"github.com/anaseto/gruid/paths"
 )
@@ -10,24 +12,24 @@ import (
 // action represents information relevant to the last UI action performed.
 type action struct {
 	Type  actionType  // kind of action (movement, quitting, ...)
-	Delta gruid.Point // direction for ActionMovement
+	Delta gruid.Point // direction for ActionBump
 }
 
 type actionType int
 
 // These constants represent the possible UI actions.
 const (
-	NoAction       actionType = iota
-	ActionMovement            // movement request
-	ActionQuit                // quit the game
+	NoAction   actionType = iota
+	ActionBump            // bump request (attack or movement)
+	ActionQuit            // quit the game
 )
 
 // handleAction updates the model in response to current recorded last action.
 func (m *model) handleAction() gruid.Effect {
 	switch m.action.Type {
-	case ActionMovement:
+	case ActionBump:
 		np := m.game.ECS.Positions[m.game.ECS.PlayerID].Add(m.action.Delta)
-		m.game.MovePlayer(np)
+		m.game.Bump(np)
 	case ActionQuit:
 		// for now, just terminate with gruid End command: this will
 		// have to be updated later when implementing saving.
@@ -36,11 +38,18 @@ func (m *model) handleAction() gruid.Effect {
 	return nil
 }
 
-// MovePlayer moves the player to a given position and updates FOV information.
-func (g *game) MovePlayer(to gruid.Point) {
+// Bump moves the player to a given position and updates FOV information,
+// or attacks if there is a monster.
+func (g *game) Bump(to gruid.Point) {
 	if !g.Map.Walkable(to) {
 		return
 	}
+	if m := g.ECS.MonsterAt(to); m != nil {
+		// We show a message to standard error. Later in the tutorial,
+		// we'll put a message in the UI instead.
+		log.Printf("You kick the %s, much to its annoyance!\n", m.Name)
+		return
+	}
 	// We move the player to the new destination.
 	g.ECS.MovePlayer(to)
 	// Update FOV.
```

Finally, we do some minor updates in other files. We first ensure that the
connected floor component during map generation is big enough, or we generate a
new map. This is to ensure that we have a wide enough map for it to be
reasonable to handle the new monsters. We add some colors for better
distinction of player, monsters and terrain, and update `tiles.go` with
appropiate values.

Note that no changes were necessary in the `Draw` method, because it already
handles any kind of entities.

``` diff
diff --git a/map.go b/map.go
index 68539cf..6f021bc 100644
--- a/map.go
+++ b/map.go
@@ -61,14 +61,22 @@ func (m *Map) Generate() {
 		{WCutoff1: 5, WCutoff2: 2, Reps: 4, WallsOutOfRange: true},
 		{WCutoff1: 5, WCutoff2: 25, Reps: 3, WallsOutOfRange: true},
 	}
-	mgen.CellularAutomataCave(Wall, Floor, 0.42, rules)
-	freep := m.RandomFloor()
-	// We put walls in floor cells non reachable from freep, to ensure that
-	// all the cells are connected (which is not guaranteed by cellular
-	// automata map generation).
-	pr := paths.NewPathRange(m.Grid.Range())
-	pr.CCMap(&path{m: m}, freep)
-	mgen.KeepCC(pr, freep, Wall)
+	for {
+		mgen.CellularAutomataCave(Wall, Floor, 0.42, rules)
+		freep := m.RandomFloor()
+		// We put walls in floor cells non reachable from freep, to ensure that
+		// all the cells are connected (which is not guaranteed by cellular
+		// automata map generation).
+		pr := paths.NewPathRange(m.Grid.Range())
+		pr.CCMap(&path{m: m}, freep)
+		ntiles := mgen.KeepCC(pr, freep, Wall)
+		const minCaveSize = 400
+		if ntiles > minCaveSize {
+			break
+		}
+		// If there were not enough free tiles, we run the map
+		// generation again.
+	}
 }
 
 // RandomFloor returns a random floor cell in the map. It assumes that such a
diff --git a/model.go b/model.go
index c39d2d1..1fd7adb 100644
--- a/model.go
+++ b/model.go
@@ -15,12 +15,6 @@ type model struct {
 	action action     // UI action
 }
 
-// game represents information relevant the current game's state.
-type game struct {
-	ECS *ECS // entities present on the map
-	Map *Map // the game map, made of tiles
-}
-
 // Update implements gruid.Model.Update. It handles keyboard and mouse input
 // messages and updates the model in response to them.
 func (m *model) Update(msg gruid.Msg) gruid.Effect {
@@ -36,6 +30,8 @@ func (m *model) Update(msg gruid.Msg) gruid.Effect {
 		// Initialization: create a player entity centered on the map.
 		m.game.ECS.PlayerID = m.game.ECS.AddEntity(NewPlayer(), m.game.Map.RandomFloor())
 		m.game.UpdateFOV()
+		// Add some monsters
+		m.game.SpawnMonsters()
 	case gruid.MsgKeyDown:
 		// Update action information on key down.
 		m.updateMsgKeyDown(msg)
@@ -48,23 +44,24 @@ func (m *model) updateMsgKeyDown(msg gruid.MsgKeyDown) {
 	pdelta := gruid.Point{}
 	switch msg.Key {
 	case gruid.KeyArrowLeft, "h":
-		m.action = action{Type: ActionMovement, Delta: pdelta.Shift(-1, 0)}
+		m.action = action{Type: ActionBump, Delta: pdelta.Shift(-1, 0)}
 	case gruid.KeyArrowDown, "j":
-		m.action = action{Type: ActionMovement, Delta: pdelta.Shift(0, 1)}
+		m.action = action{Type: ActionBump, Delta: pdelta.Shift(0, 1)}
 	case gruid.KeyArrowUp, "k":
-		m.action = action{Type: ActionMovement, Delta: pdelta.Shift(0, -1)}
+		m.action = action{Type: ActionBump, Delta: pdelta.Shift(0, -1)}
 	case gruid.KeyArrowRight, "l":
-		m.action = action{Type: ActionMovement, Delta: pdelta.Shift(1, 0)}
+		m.action = action{Type: ActionBump, Delta: pdelta.Shift(1, 0)}
 	case gruid.KeyEscape, "q":
 		m.action = action{Type: ActionQuit}
 	}
 }
 
-// Color definitions. For now, we use a special color for FOV. We start from 1,
-// because 0 is gruid.ColorDefault, which we use for default foreground and
-// background.
+// Color definitions. We start from 1, because 0 is gruid.ColorDefault, which
+// we use for default foreground and background.
 const (
 	ColorFOV gruid.Color = iota + 1
+	ColorPlayer
+	ColorMonster
 )
 
 // Draw implements gruid.Model.Draw. It draws a simple map that spans the whole
diff --git a/tiles.go b/tiles.go
index 35647fa..eeb7675 100644
--- a/tiles.go
+++ b/tiles.go
@@ -34,6 +34,12 @@ func (t *TileDrawer) GetImage(c gruid.Cell) image.Image {
 	case ColorFOV:
 		bg = image.NewUniform(color.RGBA{0x18, 0x49, 0x56, 255})
 	}
+	switch c.Style.Fg {
+	case ColorPlayer:
+		fg = image.NewUniform(color.RGBA{0x46, 0x95, 0xf7, 255})
+	case ColorMonster:
+		fg = image.NewUniform(color.RGBA{0xfa, 0x57, 0x50, 255})
+	}
 	// We return an image with the given rune drawn using the previously
 	// defined foreground and background colors.
 	return t.drawer.Draw(c.Rune, fg, bg)
```
