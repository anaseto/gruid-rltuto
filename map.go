// This file contains map-related code.

package main

import (
	"math/rand"
	"time"

	"github.com/anaseto/gruid"
	"github.com/anaseto/gruid/paths"
	"github.com/anaseto/gruid/rl"
)

// These constants represent the different kind of map tiles.
const (
	Wall rl.Cell = iota
	Floor
)

// Map represents the rectangular map of the game's level.
type Map struct {
	Grid     rl.Grid
	Rand     *rand.Rand           // random number generator
	Explored map[gruid.Point]bool // explored cells
}

// NewMap returns a new map with given size.
func NewMap(size gruid.Point) *Map {
	m := &Map{
		Grid:     rl.NewGrid(size.X, size.Y),
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
		Explored: make(map[gruid.Point]bool),
	}
	m.Generate()
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

// Generate fills the Grid attribute of m with a procedurally generated map.
func (m *Map) Generate() {
	// map generator using the rl package from gruid
	mgen := rl.MapGen{Rand: m.Rand, Grid: m.Grid}
	// cellular automata map generation with rules that give a cave-like
	// map.
	rules := []rl.CellularAutomataRule{
		{WCutoff1: 5, WCutoff2: 2, Reps: 4, WallsOutOfRange: true},
		{WCutoff1: 5, WCutoff2: 25, Reps: 3, WallsOutOfRange: true},
	}
	mgen.CellularAutomataCave(Wall, Floor, 0.42, rules)
	freep := m.RandomFloor()
	// We put walls in floor cells non reachable from freep, to ensure that
	// all the cells are connected (which is not guaranteed by cellular
	// automata map generation).
	pr := paths.NewPathRange(m.Grid.Range())
	pr.CCMap(&path{m: m}, freep)
	mgen.KeepCC(pr, freep, Wall)
}

// RandomFloor returns a random floor cell in the map. It assumes that such a
// floor cell exists (otherwise the function does not end).
func (m *Map) RandomFloor() gruid.Point {
	size := m.Grid.Size()
	for {
		freep := gruid.Point{m.Rand.Intn(size.X), m.Rand.Intn(size.Y)}
		if m.Grid.At(freep) == Floor {
			return freep
		}
	}
}

// path implements the paths.Pather interface and is used to provide pathing
// information in map generation.
type path struct {
	m  *Map
	nb paths.Neighbors
}

// Neighbors returns the list of walkable neighbors of q in the map using 4-way
// movement along cardinal directions.
func (p *path) Neighbors(q gruid.Point) []gruid.Point {
	return p.nb.Cardinal(q,
		func(r gruid.Point) bool { return p.m.Walkable(r) })
}
