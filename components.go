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

// Heal heals a fighter for a certain amount, if it does not exceed maximum HP.
// The final amount of healing is returned.
func (fi *fighter) Heal(n int) int {
	fi.HP += n
	if fi.HP > fi.MaxHP {
		n -= fi.HP - fi.MaxHP
		fi.HP = fi.MaxHP
	}
	return n
}

// AI holds simple AI data for monster's.
type AI struct {
	Path []gruid.Point // path to destination
}

// Style contains information relative to the default graphical representation
// of an entity.
type Style struct {
	Rune  rune
	Color gruid.Color
}

// Inventory holds items. For now, consumables.
type Inventory struct {
	Items []int
}
