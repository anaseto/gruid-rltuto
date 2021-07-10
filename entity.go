// This files handles a common representation for all kind of entities that can
// be placed on the map.

package main

import (
	"github.com/anaseto/gruid"
	"github.com/anaseto/gruid/rl"
)

// ECS manages entities, as well as their positions. We don't go full “ECS”
// (Entity-Component-System) in this tutorial, opting for a simpler hybrid
// approach good enough for the tutorial purposes.
type ECS struct {
	Entities  []Entity            // list of entities
	Positions map[int]gruid.Point // entity index: map position
	PlayerID  int                 // index of Player's entity (for convenience)

	Fighter map[int]*fighter // figthing component
	AI      map[int]*AI      // AI component
	Name    map[int]string   // name component
}

// NewECS returns an initialized ECS structure.
func NewECS() *ECS {
	return &ECS{
		Positions: map[int]gruid.Point{},
		Fighter:   map[int]*fighter{},
		AI:        map[int]*AI{},
		Name:      map[int]string{},
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
	return es.Entities[es.PlayerID].(*Player)
}

// MonsterAt returns the Monster at p along with its index, if any, or nil if
// there is no monster at p.
func (es *ECS) MonsterAt(p gruid.Point) (int, *Monster) {
	for i, q := range es.Positions {
		if p != q || !es.Alive(i) {
			continue
		}
		e := es.Entities[i]
		switch e := e.(type) {
		case *Monster:
			return i, e
		}
	}
	return -1, nil
}

// NoBlockingEntityAt returns true if there is no blocking entity at p (no
// player nor monsters in this tutorial).
func (es *ECS) NoBlockingEntityAt(p gruid.Point) bool {
	i, _ := es.MonsterAt(p)
	return es.Positions[es.PlayerID] != p && !es.Alive(i)
}

// PlayerDied checks whether the player died.
func (es *ECS) PlayerDied() bool {
	return es.Dead(es.PlayerID)
}

// Alive checks whether an entity is alive.
func (es *ECS) Alive(i int) bool {
	fi := es.Fighter[i]
	return fi != nil && fi.HP > 0
}

// Dead checks whether an entity is dead (was alive).
func (es *ECS) Dead(i int) bool {
	fi := es.Fighter[i]
	return fi != nil && fi.HP <= 0
}

// Style returns the graphical representation (rune and foreground color) of an
// entity.
func (es *ECS) Style(i int) (r rune, c gruid.Color) {
	r = es.Entities[i].Rune()
	c = es.Entities[i].Color()
	if es.Dead(i) {
		// Alternate representation for corpses of dead monsters.
		r = '%'
		c = gruid.ColorDefault
	}
	return r, c
}

// renderOrder is a type representing the priority of an entity rendering.
type renderOrder int

// Those constants represent distinct kinds of rendering priorities. In case
// two entities are at a given position, only the one with the highest priority
// gets displayed.
const (
	RONone renderOrder = iota
	ROCorpse
	ROItem
	ROActor
)

// RenderOrder returns the rendering priority of an entity.
func (es *ECS) RenderOrder(i int) (ro renderOrder) {
	switch es.Entities[i].(type) {
	case *Player:
		ro = ROActor
	case *Monster:
		if es.Dead(i) {
			ro = ROCorpse
		} else {
			ro = ROActor
		}
	}
	return ro
}

// Entity represents an object or creature on the map.
type Entity interface {
	Rune() rune         // the character representing the entity
	Color() gruid.Color // the character's color
}

// Player contains information relevant to the player. It implements the Entity
// interface.
type Player struct {
	FOV *rl.FOV // player's field of view
}

// maxLOS is the maximum distance in player's field of view.
const maxLOS = 10

// NewPlayer returns a new Player entity at a given position.
func NewPlayer() *Player {
	player := &Player{}
	player.FOV = rl.NewFOV(gruid.NewRange(-maxLOS, -maxLOS, maxLOS+1, maxLOS+1))
	return player
}

func (p *Player) Rune() rune {
	return '@'
}

func (p *Player) Color() gruid.Color {
	return ColorPlayer
}

// Monster represents a monster. It implements the Entity interface.
type Monster struct {
	Char rune // monster's graphical representation
}

func (m *Monster) Rune() rune {
	return m.Char
}

func (m *Monster) Color() gruid.Color {
	return ColorMonster
}
