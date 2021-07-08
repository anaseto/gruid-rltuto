# Gruid Go Roguelike Tutorial

This tutorial follows the overall structure of the [TCOD Python
Tutorial](http://rogueliketutorials.com/tutorials/tcod/v2), but makes use of
the [Go programming language](https://golang.org/) and the
[gruid](https://github.com/anaseto/gruid) roguelike game framework, instead of
TCOD.

[Table of Contents](https://github.com/anaseto/gruid-rltuto)

# Part 4

In this part, we implement a field of view for the player. We use the symmetric
shadow casting provided by gruid rl's package, inspired from [this
algorithm](https://www.albertford.com/shadowcasting/). Gruid also provides an
alternative and complementary FOV algorithm in case you need a non binary FOV
with semi-transparent obstacles. But we'll keep it simple here.

First, we update the `Player` type to hold the player's FOV, and add a
convenience `NewPlayer` method that returns a `Player` with an initialized FOV
structure.

``` diff
diff --git a/entity.go b/entity.go
index 60b29fe..f7f64b9 100644
--- a/entity.go
+++ b/entity.go
@@ -3,7 +3,10 @@
 
 package main
 
-import "github.com/anaseto/gruid"
+import (
+	"github.com/anaseto/gruid"
+	"github.com/anaseto/gruid/rl"
+)
 
 // ECS manages entities, as well as their positions. We don't go full “ECS”
 // (Entity-Component-System) in this tutorial, opting for a simpler hybrid
@@ -52,8 +55,20 @@ type Entity interface {
 }
 
 // Player contains information relevant to the player. It implements the Entity
-// interface. Empty for now, but in next parts it will information like HP.
-type Player struct{}
+// interface.
+type Player struct {
+	FOV *rl.FOV // player's field of view
+}
+
+// maxLOS is the maximum distance in player's field of view.
+const maxLOS = 10
+
+// NewPlayer returns a new Player entity at a given position.
+func NewPlayer() *Player {
+	player := &Player{}
+	player.FOV = rl.NewFOV(gruid.NewRange(-maxLOS, -maxLOS, maxLOS+1, maxLOS+1))
+	return player
+}
 
 func (p *Player) Rune() rune {
 	return '@'
```

Then, we add an `Explored` component to the map, to record explored cells.

``` diff
diff --git a/map.go b/map.go
index 4c24767..68539cf 100644
--- a/map.go
+++ b/map.go
@@ -19,15 +19,18 @@ const (
 
 // Map represents the rectangular map of the game's level.
 type Map struct {
-	Grid rl.Grid
-	Rand *rand.Rand // random number generator
+	Grid     rl.Grid
+	Rand     *rand.Rand           // random number generator
+	Explored map[gruid.Point]bool // explored cells
 }
 
 // NewMap returns a new map with given size.
 func NewMap(size gruid.Point) *Map {
-	m := &Map{}
-	m.Grid = rl.NewGrid(size.X, size.Y)
-	m.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
+	m := &Map{
+		Grid:     rl.NewGrid(size.X, size.Y),
+		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
+		Explored: make(map[gruid.Point]bool),
+	}
 	m.Generate()
 	return m
 }
```

The main changes are in file `actions.go`. We create a new `MovePlayer` method
for the model's `game`, that will handle both moving the player (if possible),
and then update the FOV and explored tiles. Further in the tutorial, when we
implement monster turns (they just keep still for now) and other actions, we'll
extend this to update FOV when needed.

``` diff
diff --git a/actions.go b/actions.go
index 5b9db72..ee49f6e 100644
--- a/actions.go
+++ b/actions.go
@@ -2,7 +2,10 @@
 
 package main
 
-import "github.com/anaseto/gruid"
+import (
+	"github.com/anaseto/gruid"
+	"github.com/anaseto/gruid/paths"
+)
 
 // action represents information relevant to the last UI action performed.
 type action struct {
@@ -23,11 +26,8 @@ const (
 func (m *model) handleAction() gruid.Effect {
 	switch m.action.Type {
 	case ActionMovement:
-		np := m.game.ECS.Positions[m.game.ECS.PlayerID]
-		np = np.Add(m.action.Delta)
-		if m.game.Map.Walkable(np) {
-			m.game.ECS.MovePlayer(np)
-		}
+		np := m.game.ECS.Positions[m.game.ECS.PlayerID].Add(m.action.Delta)
+		m.game.MovePlayer(np)
 	case ActionQuit:
 		// for now, just terminate with gruid End command: this will
 		// have to be updated later when implementing saving.
@@ -35,3 +35,48 @@ func (m *model) handleAction() gruid.Effect {
 	}
 	return nil
 }
+
+// MovePlayer moves the player to a given position and updates FOV information.
+func (g *game) MovePlayer(to gruid.Point) {
+	if !g.Map.Walkable(to) {
+		return
+	}
+	// We move the player to the new destination.
+	g.ECS.MovePlayer(to)
+	// Update FOV.
+	g.UpdateFOV()
+}
+
+// UpdateFOV updates the field of view.
+func (g *game) UpdateFOV() {
+	player := g.ECS.Player()
+	// player position
+	pp := g.ECS.Positions[g.ECS.PlayerID]
+	// We shift the FOV's Range so that it will be centered on the new
+	// player's position.
+	rg := gruid.NewRange(-maxLOS, -maxLOS, maxLOS+1, maxLOS+1)
+	player.FOV.SetRange(rg.Add(pp).Intersect(g.Map.Grid.Range()))
+	// We mark cells in field of view as explored. We use the symmetric
+	// shadow casting algorithm provided by the rl package.
+	passable := func(p gruid.Point) bool {
+		return g.Map.Grid.At(p) != Wall
+	}
+	for _, p := range player.FOV.SSCVisionMap(pp, maxLOS, passable, false) {
+		if paths.DistanceManhattan(p, pp) > maxLOS {
+			continue
+		}
+		if !g.Map.Explored[p] {
+			g.Map.Explored[p] = true
+		}
+	}
+}
+
+// InFOV returns true if p is in the player's field of view. We only keep cells
+// within maxLOS manhattan distance from the player, as natural given our
+// current 4-way movement. With 8-way movement, the natural distance choice
+// would be the Chebyshev one.
+func (g *game) InFOV(p gruid.Point) bool {
+	pp := g.ECS.Positions[g.ECS.PlayerID]
+	return g.ECS.Player().FOV.Visible(p) &&
+		paths.DistanceManhattan(pp, p) <= maxLOS
+}
```

We update then `model.go` and `tiles.go` to use the new FOV information, giving
different background colors depending on whether a tile is in FOV or not, and
not showing unexplored cells. Note that we'll use a pointer for the `game` type
in the model now, both for convenience and because we defined mutating methods
for it in `actions.go`.

``` diff
diff --git a/model.go b/model.go
index dd12d97..c39d2d1 100644
--- a/model.go
+++ b/model.go
@@ -11,7 +11,7 @@ import (
 // model represents our main application's state.
 type model struct {
 	grid   gruid.Grid // drawing grid
-	game   game       // game state
+	game   *game      // game state
 	action action     // UI action
 }
 
@@ -27,13 +27,15 @@ func (m *model) Update(msg gruid.Msg) gruid.Effect {
 	m.action = action{} // reset last action information
 	switch msg := msg.(type) {
 	case gruid.MsgInit:
+		m.game = &game{}
 		// Initialize map
 		size := m.grid.Size() // map size: for now the whole window
 		m.game.Map = NewMap(size)
 		// Initialize entities
 		m.game.ECS = NewECS()
 		// Initialization: create a player entity centered on the map.
-		m.game.ECS.PlayerID = m.game.ECS.AddEntity(&Player{}, m.game.Map.RandomFloor())
+		m.game.ECS.PlayerID = m.game.ECS.AddEntity(NewPlayer(), m.game.Map.RandomFloor())
+		m.game.UpdateFOV()
 	case gruid.MsgKeyDown:
 		// Update action information on key down.
 		m.updateMsgKeyDown(msg)
@@ -58,21 +60,42 @@ func (m *model) updateMsgKeyDown(msg gruid.MsgKeyDown) {
 	}
 }
 
+// Color definitions. For now, we use a special color for FOV. We start from 1,
+// because 0 is gruid.ColorDefault, which we use for default foreground and
+// background.
+const (
+	ColorFOV gruid.Color = iota + 1
+)
+
 // Draw implements gruid.Model.Draw. It draws a simple map that spans the whole
 // grid.
 func (m *model) Draw() gruid.Grid {
 	m.grid.Fill(gruid.Cell{Rune: ' '})
+	g := m.game
 	// We draw the map tiles.
-	it := m.game.Map.Grid.Iterator()
+	it := g.Map.Grid.Iterator()
 	for it.Next() {
-		m.grid.Set(it.P(), gruid.Cell{Rune: m.game.Map.Rune(it.Cell())})
+		if !g.Map.Explored[it.P()] {
+			continue
+		}
+		c := gruid.Cell{Rune: g.Map.Rune(it.Cell())}
+		if g.InFOV(it.P()) {
+			c.Style.Bg = ColorFOV
+		}
+		m.grid.Set(it.P(), c)
 	}
 	// We draw the entities.
-	for i, e := range m.game.ECS.Entities {
-		m.grid.Set(m.game.ECS.Positions[i], gruid.Cell{
-			Rune:  e.Rune(),
-			Style: gruid.Style{Fg: e.Color()},
-		})
+	for i, e := range g.ECS.Entities {
+		p := g.ECS.Positions[i]
+		if !g.Map.Explored[p] || !g.InFOV(p) {
+			continue
+		}
+		c := m.grid.At(p)
+		c.Rune = e.Rune()
+		c.Style.Fg = e.Color()
+		m.grid.Set(p, c)
+		// NOTE: We retrieved current cell at e.Pos() to preserve
+		// background (in FOV or not).
 	}
 	return m.grid
 }
diff --git a/tiles.go b/tiles.go
index cb5b226..35647fa 100644
--- a/tiles.go
+++ b/tiles.go
@@ -29,9 +29,11 @@ func (t *TileDrawer) GetImage(c gruid.Cell) image.Image {
 	// using the palette variant with dark backgound and light foreground.
 	fg := image.NewUniform(color.RGBA{0xad, 0xbc, 0xbc, 255})
 	bg := image.NewUniform(color.RGBA{0x10, 0x3c, 0x48, 255})
-	// NOTE: Here, we will add support for more colors further in the
-	// tutorial.
-
+	// We define non default-colors (for FOV, ...).
+	switch c.Style.Bg {
+	case ColorFOV:
+		bg = image.NewUniform(color.RGBA{0x18, 0x49, 0x56, 255})
+	}
 	// We return an image with the given rune drawn using the previously
 	// defined foreground and background colors.
 	return t.drawer.Draw(c.Rune, fg, bg)
```
