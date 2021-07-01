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
	return es.Entities[es.PlayerID].(*Player)
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
	return gruid.ColorDefault
}
