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
