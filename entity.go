// This files handles a common representation for all kind of entities that can
// be placed on the map.

package main

import "github.com/anaseto/gruid"

// ECS manages access, additions and removals of entities.  For now, we use a
// simple list of entities as a representation. Later in the tutorial, we will
// show how to provide additional representations to, for example, have
// efficient access to the entities that exist at a given position.
type ECS struct {
	Entities []Entity
}

// Add adds a new entity.
func (es *ECS) AddEntity(e Entity) {
	es.Entities = append(es.Entities, e)
}

// Player returns the Player entity.
func (es *ECS) Player() *Player {
	for _, e := range es.Entities {
		e, ok := e.(*Player)
		if ok {
			return e
		}
	}
	return nil
}

// Entity represents an object or creature on the map.
type Entity interface {
	Pos() gruid.Point   // the position of the entity
	Rune() rune         // the character representing the entity
	Color() gruid.Color // the character's color
}

// Player contains information relevant to the player. It implements the Entity
// interface.
type Player struct {
	P gruid.Point // position on the map
}

func (p *Player) Pos() gruid.Point {
	return p.P
}

func (p *Player) Rune() rune {
	return '@'
}

func (p *Player) Color() gruid.Color {
	return gruid.ColorDefault
}
