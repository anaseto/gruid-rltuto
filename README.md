# Gruid Go Roguelike Tutorial

This tutorial follows the overall structure of the [TCOD Python
Tutorial](http://rogueliketutorials.com/tutorials/tcod/v2), but makes use of
the [Go programming language](https://golang.org/) and the
[gruid](https://github.com/anaseto/gruid) roguelike game framework, instead of
TCOD.

[Table of Contents](https://github.com/anaseto/gruid-rltuto)

# Part 2 - Generic entities, and the map

In this part, we introduce the `Entity` interface in a new file `entity.go`,
which will represent any kind of entities that can be placed on the map. A type
satisfying the `Entity` interface should have several methods that give
information on position and display. As a first example, we introduce a
`Player` type implementing the `Entity` interface.

``` go
// This files handles a common representation for all kind of entities that can
// be placed on the map.

package main

import "github.com/anaseto/gruid"

// ECS manages entities, as well as their positions. We don't go full “ECS”
// (Entity-Component-System) in this tutorial, opting for a simpler hybrid
// approach good enough for the tutorial purposes.
type ECS struct {
	Entities  []Entity            // list of entities
	Positions map[int]gruid.Point // entity index: map position
	PlayerID  int                 // index of Player's entity (for convenience)
}

// NewECS returns an initialized ECS structure.
func NewECS() *ECS {
	return &ECS{
		Positions: map[int]gruid.Point{},
	}
}

// Add adds a new entity at a given position and returns its index/id.
func (es *ECS) AddEntity(e Entity, p gruid.Point) int {
	i := len(es.Entities)
	es.Entities = append(es.Entities, e)
	es.Positions[i] = p
	return i
}

// MoveEntity moves the i-th entity to p.
func (es *ECS) MoveEntity(i int, p gruid.Point) {
	es.Positions[i] = p
}

// MovePlayer moves the player entity to p.
func (es *ECS) MovePlayer(p gruid.Point) {
	es.MoveEntity(es.PlayerID, p)
}

// Player returns the Player entity. Just a shorthand for easily accessing the
// Player entity.
func (es *ECS) Player() *Player {
	return es.Entities[es.PlayerID].(*Player) // index 0 for player entity (convention)
}

// Entity represents an object or creature on the map.
type Entity interface {
	Rune() rune         // the character representing the entity
	Color() gruid.Color // the character's color
}

// Player contains information relevant to the player. It implements the Entity
// interface. Empty for now, but in next parts it will information like HP.
type Player struct{}

func (p *Player) Rune() rune {
	return '@'
}

func (p *Player) Color() gruid.Color {
	return gruid.ColorDefault
}
```

We also introduce a `Map` type for representing the map in `map.go`. We define
`Wall` and `Floor` tiles, and give a graphical representation to them.

``` go
// This file contains map-related code.

package main

import (
	"github.com/anaseto/gruid"
	"github.com/anaseto/gruid/rl"
)

// These constants represent the different kind of map tiles.
const (
	Wall rl.Cell = iota
	Floor
)

// Map represents the rectangular map of the game's level.
type Map struct {
	Grid rl.Grid
}

// NewMap returns a new map with given size.
func NewMap(size gruid.Point) *Map {
	m := &Map{}
	m.Grid = rl.NewGrid(size.X, size.Y)
	m.Grid.Fill(Floor)
	for i := 0; i < 3; i++ {
		// We add a few walls. We'll deal with map generation
		// in the next part of the tutorial.
		m.Grid.Set(gruid.Point{30 + i, 12}, Wall)
	}
	return m
}

// Walkable returns true if at the given position there is a floor tile.
func (m *Map) Walkable(p gruid.Point) bool {
	return m.Grid.At(p) == Floor
}

// Rune returns the character rune representing a given terrain.
func (m *Map) Rune(c rl.Cell) (r rune) {
	switch c {
	case Wall:
		r = '#'
	case Floor:
		r = '.'
	}
	return r
}
```

We then adjust the code of the `Draw` method in `model.go` to take into account
the new representation of entities and the map. We first draw the map, and then
we place entities. We also make a few updates in `model.go` and `actions.go` to
adapt to the new code for the map and entities.

``` diff
diff --git a/actions.go b/actions.go
index 52e46a0..5b9db72 100644
--- a/actions.go
+++ b/actions.go
@@ -23,7 +23,11 @@ const (
 func (m *model) handleAction() gruid.Effect {
 	switch m.action.Type {
 	case ActionMovement:
-		m.game.PlayerPos = m.game.PlayerPos.Add(m.action.Delta)
+		np := m.game.ECS.Positions[m.game.ECS.PlayerID]
+		np = np.Add(m.action.Delta)
+		if m.game.Map.Walkable(np) {
+			m.game.ECS.MovePlayer(np)
+		}
 	case ActionQuit:
 		// for now, just terminate with gruid End command: this will
 		// have to be updated later when implementing saving.
diff --git a/model.go b/model.go
index 21ee8aa..476b60e 100644
--- a/model.go
+++ b/model.go
@@ -4,7 +4,9 @@
 
 package main
 
-import "github.com/anaseto/gruid"
+import (
+	"github.com/anaseto/gruid"
+)
 
 // models represents our main application state.
 type model struct {
@@ -15,7 +17,8 @@ type model struct {
 
 // game represents information relevant the current game's state.
 type game struct {
-	PlayerPos gruid.Point // tracks player position
+	ECS *ECS // entities present on the map
+	Map *Map // the game map, made of tiles
 }
 
 // Update implements gruid.Model.Update. It handles keyboard and mouse input
@@ -24,8 +27,13 @@ func (m *model) Update(msg gruid.Msg) gruid.Effect {
 	m.action = action{} // reset last action information
 	switch msg := msg.(type) {
 	case gruid.MsgInit:
-		// Initialization: set player position in the center.
-		m.game.PlayerPos = m.grid.Size().Div(2)
+		// Initialize map
+		size := m.grid.Size() // map size: for now the whole window
+		m.game.Map = NewMap(size)
+		// Initialize entities
+		m.game.ECS = NewECS()
+		// Initialization: create a player entity centered on the map.
+		m.game.ECS.PlayerID = m.game.ECS.AddEntity(&Player{}, size.Div(2))
 	case gruid.MsgKeyDown:
 		// Update action information on key down.
 		m.updateMsgKeyDown(msg)
@@ -53,14 +61,18 @@ func (m *model) updateMsgKeyDown(msg gruid.MsgKeyDown) {
 // Draw implements gruid.Model.Draw. It draws a simple map that spans the whole
 // grid.
 func (m *model) Draw() gruid.Grid {
-	it := m.grid.Iterator()
+	m.grid.Fill(gruid.Cell{Rune: ' '})
+	// We draw the map tiles.
+	it := m.game.Map.Grid.Iterator()
 	for it.Next() {
-		switch {
-		case it.P() == m.game.PlayerPos:
-			it.SetCell(gruid.Cell{Rune: '@'})
-		default:
-			it.SetCell(gruid.Cell{Rune: ' '})
-		}
+		m.grid.Set(it.P(), gruid.Cell{Rune: m.game.Map.Rune(it.Cell())})
+	}
+	// We draw the entities.
+	for i, e := range m.game.ECS.Entities {
+		m.grid.Set(m.game.ECS.Positions[i], gruid.Cell{
+			Rune:  e.Rune(),
+			Style: gruid.Style{Fg: e.Color()},
+		})
 	}
 	return m.grid
 }
```
