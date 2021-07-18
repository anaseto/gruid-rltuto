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
