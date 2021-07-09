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
