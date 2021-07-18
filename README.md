# Gruid Go Roguelike Tutorial

This tutorial follows the overall structure of the [TCOD Python
Tutorial](http://rogueliketutorials.com/tutorials/tcod/v2), but makes use of
the [Go programming language](https://golang.org/) and the
[gruid](https://github.com/anaseto/gruid) roguelike game framework, instead of
TCOD.

[Table of Contents](https://github.com/anaseto/gruid-rltuto)

## Part 6 - Doing (and taking) some damage

In this part we implement basic combat: HP, attack, defense, and, of course, a
basic AI for monsters.

We create a new file `components.go` which describes components (data) for
fighting and AI.

``` go
// This file describes entity components, for example for basic fighting or AI.

package main

import "github.com/anaseto/gruid"

// fighter holds data relevant to fighting. We'll use simple attack/defense
// stats.
type fighter struct {
	HP      int // Health Points
	MaxHP   int // Maximum Health Points
	Power   int // attack power
	Defense int // defence
}

// AI holds simple AI data for monster's.
type AI struct {
	Path []gruid.Point // path to destination
}
```

We then add those components in the `ECS` type. We also add a new `Name`
component, and remove the `Name` field in the `Monster` struct. We then add a
few methods, for example to determine whether an entity is alive or dead, and
depending on that, which rendering style should be used (default monster
rendering, or corpse rendering with a `%` character).

``` diff
diff --git a/entity.go b/entity.go
index 364a065..fac1c75 100644
--- a/entity.go
+++ b/entity.go
@@ -15,12 +15,19 @@ type ECS struct {
 	Entities  []Entity            // list of entities
 	Positions map[int]gruid.Point // entity index: map position
 	PlayerID  int                 // index of Player's entity (for convenience)
+
+	Fighter map[int]*fighter // figthing component
+	AI      map[int]*AI      // AI component
+	Name    map[int]string   // name component
 }
 
 // NewECS returns an initialized ECS structure.
 func NewECS() *ECS {
 	return &ECS{
 		Positions: map[int]gruid.Point{},
+		Fighter:   map[int]*fighter{},
+		AI:        map[int]*AI{},
+		Name:      map[int]string{},
 	}
 }
 
@@ -48,26 +55,85 @@ func (es *ECS) Player() *Player {
 	return es.Entities[es.PlayerID].(*Player)
 }
 
-// MonsterAt returns the Monster at p, if any, or nil if there is no monster at
-// p.
-func (es *ECS) MonsterAt(p gruid.Point) *Monster {
+// MonsterAt returns the Monster at p along with its index, if any, or nil if
+// there is no monster at p.
+func (es *ECS) MonsterAt(p gruid.Point) (int, *Monster) {
 	for i, q := range es.Positions {
-		if p != q {
+		if p != q || !es.Alive(i) {
 			continue
 		}
 		e := es.Entities[i]
 		switch e := e.(type) {
 		case *Monster:
-			return e
+			return i, e
 		}
 	}
-	return nil
+	return -1, nil
 }
 
 // NoBlockingEntityAt returns true if there is no blocking entity at p (no
 // player nor monsters in this tutorial).
 func (es *ECS) NoBlockingEntityAt(p gruid.Point) bool {
-	return es.Positions[es.PlayerID] != p && es.MonsterAt(p) == nil
+	i, _ := es.MonsterAt(p)
+	return es.Positions[es.PlayerID] != p && !es.Alive(i)
+}
+
+// PlayerDied checks whether the player died.
+func (es *ECS) PlayerDied() bool {
+	return es.Dead(es.PlayerID)
+}
+
+// Alive checks whether an entity is alive.
+func (es *ECS) Alive(i int) bool {
+	fi := es.Fighter[i]
+	return fi != nil && fi.HP > 0
+}
+
+// Dead checks whether an entity is dead (was alive).
+func (es *ECS) Dead(i int) bool {
+	fi := es.Fighter[i]
+	return fi != nil && fi.HP <= 0
+}
+
+// Style returns the graphical representation (rune and foreground color) of an
+// entity.
+func (es *ECS) Style(i int) (r rune, c gruid.Color) {
+	r = es.Entities[i].Rune()
+	c = es.Entities[i].Color()
+	if es.Dead(i) {
+		// Alternate representation for corpses of dead monsters.
+		r = '%'
+		c = gruid.ColorDefault
+	}
+	return r, c
+}
+
+// renderOrder is a type representing the priority of an entity rendering.
+type renderOrder int
+
+// Those constants represent distinct kinds of rendering priorities. In case
+// two entities are at a given position, only the one with the highest priority
+// gets displayed.
+const (
+	RONone renderOrder = iota
+	ROCorpse
+	ROItem
+	ROActor
+)
+
+// RenderOrder returns the rendering priority of an entity.
+func (es *ECS) RenderOrder(i int) (ro renderOrder) {
+	switch es.Entities[i].(type) {
+	case *Player:
+		ro = ROActor
+	case *Monster:
+		if es.Dead(i) {
+			ro = ROCorpse
+		} else {
+			ro = ROActor
+		}
+	}
+	return ro
 }
 
 // Entity represents an object or creature on the map.
@@ -100,11 +166,9 @@ func (p *Player) Color() gruid.Color {
 	return ColorPlayer
 }
 
-// Monster represents a monster. It implements the Entity interface. For now,
-// we simply give it a name and a rune for its graphical representation.
+// Monster represents a monster. It implements the Entity interface.
 type Monster struct {
-	Name string
-	Char rune
+	Char rune // monster's graphical representation
 }
 
 func (m *Monster) Rune() rune {
```

We create a new file `ai.go` that handles monster's AI and actions. The base AI
we implement is simple: if a monster is not in the player's FOV, it choses a
random target and we compute a path to it using `AstarPath` from gruid's
`paths` packages, then move along it. If it is in the player's FOV, we compute
a path to the player, and move along it. If the monster is adjacent to the
player, attack.

``` go
// This file handles the base AI for monsters.

package main

import (
	"github.com/anaseto/gruid"
	"github.com/anaseto/gruid/paths"
)

// HandleMonsterTurn handles a monster's turn. The function assumes the entity
// with the given index is indeed a monster initialized with fighter and AI
// components.
func (g *game) HandleMonsterTurn(i int) {
	if !g.ECS.Alive(i) {
		// Do nothing if the entity corresponds to a dead monster.
		return
	}
	p := g.ECS.Positions[i]
	ai := g.ECS.AI[i]
	aip := &aiPath{g: g}
	pp := g.ECS.Positions[g.ECS.PlayerID]
	if paths.DistanceManhattan(p, pp) == 1 {
		// If the monster is adjacent to the player, attack.
		g.BumpAttack(i, g.ECS.PlayerID)
		return
	}
	if !g.InFOV(p) {
		// The monster is not in player's FOV.
		if len(ai.Path) < 1 {
			// Pick new path to a random floor tile.
			ai.Path = g.PR.AstarPath(aip, p, g.Map.RandomFloor())
		}
		g.AIMove(i)
		// NOTE: this base AI can be improved for example to avoid
		// monster's getting stuck between them. It's enough to get
		// started, though.
		return
	}
	// The monster is in player's FOV, so we compute a suitable path to
	// reach the player.
	ai.Path = g.PR.AstarPath(aip, p, pp)
	g.AIMove(i)
}

// AIMove moves a monster to the next position, if there is no blocking entity
// at the destination. It assumes the destination is walkable.
func (g *game) AIMove(i int) {
	ai := g.ECS.AI[i]
	if len(ai.Path) > 0 && ai.Path[0] == g.ECS.Positions[i] {
		ai.Path = ai.Path[1:]
	}
	if len(ai.Path) > 0 && g.ECS.NoBlockingEntityAt(ai.Path[0]) {
		// Only move if there is no blocking entity.
		g.ECS.MoveEntity(i, ai.Path[0])
		ai.Path = ai.Path[1:]
	}
}

// aiPath implements the paths.Astar interface for use in AI pathfinding.
type aiPath struct {
	g  *game
	nb paths.Neighbors
}

// Neighbors returns the list of walkable neighbors of q in the map using 4-way
// movement along cardinal directions.
func (aip *aiPath) Neighbors(q gruid.Point) []gruid.Point {
	return aip.nb.Cardinal(q,
		func(r gruid.Point) bool {
			return aip.g.Map.Walkable(r)
		})
}

// Cost implements paths.Astar.Cost.
func (aip *aiPath) Cost(p, q gruid.Point) int {
	if !aip.g.ECS.NoBlockingEntityAt(q) {
		// Extra cost for blocked positions: this encourages the
		// pathfinding algorithm to take another path to reach the
		// player.
		return 8
	}
	return 1
}

// Estimation implements paths.Astar.Estimation. For 4-way movement, we use the
// Manhattan distance.
func (aip *aiPath) Estimation(p, q gruid.Point) int {
	return paths.DistanceManhattan(p, q)
}
```

We then write a couple of additional functions: one for handling turns
`EndTurn`, which simply iterates over entities, and handles monster turns, 
and `BumpAttack`, which does the damage and HP updates. In a more complete
game, in order for example to handle different kind of movement speeds, or to
schedule some events, more complex systems can be used (like
energy-based ones, or priority-queue based ones using for example the
`EventQueue` type provided by gruid's `rl` package).

Finally, we write also some glue code to hold things together, and to
initialize new data properly, like entity fighting statistics, and pathfinding
structures. We also moved `UpdateFOV` and `InFOV` methods in `game.go`.

``` diff
diff --git a/actions.go b/actions.go
index f945e7a..ccec92d 100644
--- a/actions.go
+++ b/actions.go
@@ -6,7 +6,6 @@ import (
 	"log"
 
 	"github.com/anaseto/gruid"
-	"github.com/anaseto/gruid/paths"
 )
 
 // action represents information relevant to the last UI action performed.
@@ -21,6 +20,7 @@ type actionType int
 const (
 	NoAction   actionType = iota
 	ActionBump            // bump request (attack or movement)
+	ActionWait            // wait a turn
 	ActionQuit            // quit the game
 )
 
@@ -30,11 +30,17 @@ func (m *model) handleAction() gruid.Effect {
 	case ActionBump:
 		np := m.game.ECS.Positions[m.game.ECS.PlayerID].Add(m.action.Delta)
 		m.game.Bump(np)
+	case ActionWait:
+		m.game.EndTurn()
 	case ActionQuit:
 		// for now, just terminate with gruid End command: this will
 		// have to be updated later when implementing saving.
 		return gruid.End()
 	}
+	if m.game.ECS.PlayerDied() {
+		log.Print("You died")
+		return gruid.End()
+	}
 	return nil
 }
 
@@ -44,48 +50,14 @@ func (g *game) Bump(to gruid.Point) {
 	if !g.Map.Walkable(to) {
 		return
 	}
-	if m := g.ECS.MonsterAt(to); m != nil {
+	if i, _ := g.ECS.MonsterAt(to); g.ECS.Alive(i) {
 		// We show a message to standard error. Later in the tutorial,
 		// we'll put a message in the UI instead.
-		log.Printf("You kick the %s, much to its annoyance!\n", m.Name)
+		g.BumpAttack(g.ECS.PlayerID, i)
+		g.EndTurn()
 		return
 	}
 	// We move the player to the new destination.
 	g.ECS.MovePlayer(to)
-	// Update FOV.
-	g.UpdateFOV()
-}
-
-// UpdateFOV updates the field of view.
-func (g *game) UpdateFOV() {
-	player := g.ECS.Player()
-	// player position
-	pp := g.ECS.Positions[g.ECS.PlayerID]
-	// We shift the FOV's Range so that it will be centered on the new
-	// player's position.
-	rg := gruid.NewRange(-maxLOS, -maxLOS, maxLOS+1, maxLOS+1)
-	player.FOV.SetRange(rg.Add(pp).Intersect(g.Map.Grid.Range()))
-	// We mark cells in field of view as explored. We use the symmetric
-	// shadow casting algorithm provided by the rl package.
-	passable := func(p gruid.Point) bool {
-		return g.Map.Grid.At(p) != Wall
-	}
-	for _, p := range player.FOV.SSCVisionMap(pp, maxLOS, passable, false) {
-		if paths.DistanceManhattan(p, pp) > maxLOS {
-			continue
-		}
-		if !g.Map.Explored[p] {
-			g.Map.Explored[p] = true
-		}
-	}
-}
-
-// InFOV returns true if p is in the player's field of view. We only keep cells
-// within maxLOS manhattan distance from the player, as natural given our
-// current 4-way movement. With 8-way movement, the natural distance choice
-// would be the Chebyshev one.
-func (g *game) InFOV(p gruid.Point) bool {
-	pp := g.ECS.Positions[g.ECS.PlayerID]
-	return g.ECS.Player().FOV.Visible(p) &&
-		paths.DistanceManhattan(pp, p) <= maxLOS
+	g.EndTurn()
 }
diff --git a/game.go b/game.go
index d07769e..caa36c3 100644
--- a/game.go
+++ b/game.go
@@ -3,12 +3,20 @@
 
 package main
 
-import "github.com/anaseto/gruid"
+import (
+	"fmt"
+	"log"
+	"strings"
+
+	"github.com/anaseto/gruid"
+	"github.com/anaseto/gruid/paths"
+)
 
 // game represents information relevant the current game's state.
 type game struct {
-	ECS *ECS // entities present on the map
-	Map *Map // the game map, made of tiles
+	ECS *ECS             // entities present on the map
+	Map *Map             // the game map, made of tiles
+	PR  *paths.PathRange // path range for the map
 }
 
 // SpawnMonsters adds some monsters in the current map.
@@ -20,14 +28,25 @@ func (g *game) SpawnMonsters() {
 		// probabilities respectively.
 		switch {
 		case g.Map.Rand.Intn(100) < 80:
-			m.Name = "orc"
 			m.Char = 'o'
 		default:
-			m.Name = "troll"
 			m.Char = 'T'
 		}
 		p := g.FreeFloorTile()
-		g.ECS.AddEntity(m, p)
+		i := g.ECS.AddEntity(m, p)
+		switch m.Char {
+		case 'o':
+			g.ECS.Fighter[i] = &fighter{
+				HP: 10, MaxHP: 10, Defense: 0, Power: 3,
+			}
+			g.ECS.Name[i] = "orc"
+		case 'T':
+			g.ECS.Fighter[i] = &fighter{
+				HP: 16, MaxHP: 16, Defense: 1, Power: 4,
+			}
+			g.ECS.Name[i] = "troll"
+		}
+		g.ECS.AI[i] = &AI{}
 	}
 }
 
@@ -40,3 +59,67 @@ func (g *game) FreeFloorTile() gruid.Point {
 		}
 	}
 }
+
+// EndTurn is called when the player's turn ends. Currently, the player and
+// monsters have all the same speed, so we make each monster act each time the
+// player's does an action that ends a turn.
+func (g *game) EndTurn() {
+	g.UpdateFOV()
+	for i, e := range g.ECS.Entities {
+		if g.ECS.PlayerDied() {
+			return
+		}
+		switch e.(type) {
+		case *Monster:
+			g.HandleMonsterTurn(i)
+		}
+	}
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
+
+// BumpAttack implements attack of a fighter entity on another.
+func (g *game) BumpAttack(i, j int) {
+	fi := g.ECS.Fighter[i]
+	fj := g.ECS.Fighter[j]
+	damage := fi.Power - fj.Defense
+	attackDesc := fmt.Sprintf("%s attacks %s", strings.Title(g.ECS.Name[i]), g.ECS.Name[j])
+	if damage > 0 {
+		log.Printf("%s for %d damage", attackDesc, damage)
+		fj.HP -= damage
+	} else {
+		log.Printf("%s but does no damage", attackDesc)
+	}
+}
diff --git a/model.go b/model.go
index 1fd7adb..04e5fd3 100644
--- a/model.go
+++ b/model.go
@@ -5,7 +5,10 @@
 package main
 
 import (
+	"sort"
+
 	"github.com/anaseto/gruid"
+	"github.com/anaseto/gruid/paths"
 )
 
 // model represents our main application's state.
@@ -25,10 +28,15 @@ func (m *model) Update(msg gruid.Msg) gruid.Effect {
 		// Initialize map
 		size := m.grid.Size() // map size: for now the whole window
 		m.game.Map = NewMap(size)
+		m.game.PR = paths.NewPathRange(gruid.NewRange(0, 0, size.X, size.Y))
 		// Initialize entities
 		m.game.ECS = NewECS()
 		// Initialization: create a player entity centered on the map.
 		m.game.ECS.PlayerID = m.game.ECS.AddEntity(NewPlayer(), m.game.Map.RandomFloor())
+		m.game.ECS.Fighter[m.game.ECS.PlayerID] = &fighter{
+			HP: 30, MaxHP: 30, Power: 5, Defense: 2,
+		}
+		m.game.ECS.Name[m.game.ECS.PlayerID] = "you"
 		m.game.UpdateFOV()
 		// Add some monsters
 		m.game.SpawnMonsters()
@@ -51,6 +59,8 @@ func (m *model) updateMsgKeyDown(msg gruid.MsgKeyDown) {
 		m.action = action{Type: ActionBump, Delta: pdelta.Shift(0, -1)}
 	case gruid.KeyArrowRight, "l":
 		m.action = action{Type: ActionBump, Delta: pdelta.Shift(1, 0)}
+	case gruid.KeyEnter, ".":
+		m.action = action{Type: ActionWait}
 	case gruid.KeyEscape, "q":
 		m.action = action{Type: ActionQuit}
 	}
@@ -81,15 +91,22 @@ func (m *model) Draw() gruid.Grid {
 		}
 		m.grid.Set(it.P(), c)
 	}
-	// We draw the entities.
-	for i, e := range g.ECS.Entities {
+	// We sort entity indexes using the render ordering.
+	sortedEntities := make([]int, 0, len(g.ECS.Entities))
+	for i := range g.ECS.Entities {
+		sortedEntities = append(sortedEntities, i)
+	}
+	sort.Slice(sortedEntities, func(i, j int) bool {
+		return g.ECS.RenderOrder(sortedEntities[i]) < g.ECS.RenderOrder(sortedEntities[j])
+	})
+	// We draw the sorted entities.
+	for _, i := range sortedEntities {
 		p := g.ECS.Positions[i]
 		if !g.Map.Explored[p] || !g.InFOV(p) {
 			continue
 		}
 		c := m.grid.At(p)
-		c.Rune = e.Rune()
-		c.Style.Fg = e.Color()
+		c.Rune, c.Style.Fg = g.ECS.Style(i)
 		m.grid.Set(p, c)
 		// NOTE: We retrieved current cell at e.Pos() to preserve
 		// background (in FOV or not).
```

* * *

[Next Part](https://github.com/anaseto/gruid-rltuto/tree/part-7)
