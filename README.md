# Gruid Go Roguelike Tutorial

This tutorial follows the overall structure of the [TCOD Python
Tutorial](http://rogueliketutorials.com/tutorials/tcod/v2), but makes use of
the [Go programming language](https://golang.org/) and the
[gruid](https://github.com/anaseto/gruid) roguelike game framework, instead of
TCOD.

[Table of Contents](https://github.com/anaseto/gruid-rltuto)

# Part 3 - Generating a Dungeon

In this part, we procedurally generate a map. For this tutorial in Go, we chose
a simple approach to get easily started: we rely on package `rl` cellular
automata generation to produce a natural cave-looking map.

Another (maybe even simpler) alternative would be to use the RandomWalk map
generation, or a simple room and tunnel generation like in the Python TCOD
tutorial. In a finished game, you probably want them all at some point, or even
combine them, usually with prefabricated rooms mixed in! But we'll keep it
simple for now.

Most of the changes occur in file `map.go`.

We add a new `Rand` field for the `Map` type, that will contain the random
number generator we'll use for maps, and also for finding a random floor cell
in which to place the player.

Also, because cellular automata generation does not guarantee a connected
result, we use the paths package to find a connected component using the `CCMap`
method of the `PathRange` type, and then `KeepCC` to keep only one connected
component. For this, we need to define a `path` type satisfying the `Pather`
interface which provides means to find neighbors of a map cell.

``` diff
diff --git a/map.go b/map.go
index 756fb1d..4c24767 100644
--- a/map.go
+++ b/map.go
@@ -3,7 +3,11 @@
 package main
 
 import (
+	"math/rand"
+	"time"
+
 	"github.com/anaseto/gruid"
+	"github.com/anaseto/gruid/paths"
 	"github.com/anaseto/gruid/rl"
 )
 
@@ -16,18 +20,15 @@ const (
 // Map represents the rectangular map of the game's level.
 type Map struct {
 	Grid rl.Grid
+	Rand *rand.Rand // random number generator
 }
 
 // NewMap returns a new map with given size.
 func NewMap(size gruid.Point) *Map {
 	m := &Map{}
 	m.Grid = rl.NewGrid(size.X, size.Y)
-	m.Grid.Fill(Floor)
-	for i := 0; i < 3; i++ {
-		// We add a few walls. We'll deal with map generation
-		// in the next part of the tutorial.
-		m.Grid.Set(gruid.Point{30 + i, 12}, Wall)
-	}
+	m.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
+	m.Generate()
 	return m
 }
 
@@ -46,3 +47,49 @@ func (m *Map) Rune(c rl.Cell) (r rune) {
 	}
 	return r
 }
+
+// Generate fills the Grid attribute of m with a procedurally generated map.
+func (m *Map) Generate() {
+	// map generator using the rl package from gruid
+	mgen := rl.MapGen{Rand: m.Rand, Grid: m.Grid}
+	// cellular automata map generation with rules that give a cave-like
+	// map.
+	rules := []rl.CellularAutomataRule{
+		{WCutoff1: 5, WCutoff2: 2, Reps: 4, WallsOutOfRange: true},
+		{WCutoff1: 5, WCutoff2: 25, Reps: 3, WallsOutOfRange: true},
+	}
+	mgen.CellularAutomataCave(Wall, Floor, 0.42, rules)
+	freep := m.RandomFloor()
+	// We put walls in floor cells non reachable from freep, to ensure that
+	// all the cells are connected (which is not guaranteed by cellular
+	// automata map generation).
+	pr := paths.NewPathRange(m.Grid.Range())
+	pr.CCMap(&path{m: m}, freep)
+	mgen.KeepCC(pr, freep, Wall)
+}
+
+// RandomFloor returns a random floor cell in the map. It assumes that such a
+// floor cell exists (otherwise the function does not end).
+func (m *Map) RandomFloor() gruid.Point {
+	size := m.Grid.Size()
+	for {
+		freep := gruid.Point{m.Rand.Intn(size.X), m.Rand.Intn(size.Y)}
+		if m.Grid.At(freep) == Floor {
+			return freep
+		}
+	}
+}
+
+// path implements the paths.Pather interface and is used to provide pathing
+// information in map generation.
+type path struct {
+	m  *Map
+	nb paths.Neighbors
+}
+
+// Neighbors returns the list of walkable neighbors of q in the map using 4-way
+// movement along cardinal directions.
+func (p *path) Neighbors(q gruid.Point) []gruid.Point {
+	return p.nb.Cardinal(q,
+		func(r gruid.Point) bool { return p.m.Walkable(r) })
+}
```

We also update in `model.go` the starting position for the player to be a
random floor tile.

``` diff
diff --git a/model.go b/model.go
index 84e6d5b..dd12d97 100644
--- a/model.go
+++ b/model.go
@@ -33,7 +33,7 @@ func (m *model) Update(msg gruid.Msg) gruid.Effect {
 		// Initialize entities
 		m.game.ECS = NewECS()
 		// Initialization: create a player entity centered on the map.
-		m.game.ECS.PlayerID = m.game.ECS.AddEntity(&Player{}, size.Div(2))
+		m.game.ECS.PlayerID = m.game.ECS.AddEntity(&Player{}, m.game.Map.RandomFloor())
 	case gruid.MsgKeyDown:
 		// Update action information on key down.
 		m.updateMsgKeyDown(msg)
```

* * *

[Next Part](https://github.com/anaseto/gruid-rltuto/tree/part-4)
