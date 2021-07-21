// This file describes item entities.

package main

import (
	"errors"
	"fmt"

	"github.com/anaseto/gruid"
)

// Consumable describes a consumable item, like a potion.
type Consumable interface {
	// Activate makes use of an item using a specific action. It returns
	// an error if the consumable could not be activated.
	Activate(g *game, a itemAction) error
}

// itemAction describes information relative to usage of an item: which
// actor does the action, and whether the action has a particular target
// position.
type itemAction struct {
	Actor  int          // entity doing the action
	Target *gruid.Point // optional target
}

// HealingPotion describes a potion that heals of a given amount.
type HealingPotion struct {
	Amount int
}

func (pt *HealingPotion) Activate(g *game, a itemAction) error {
	fi := g.ECS.Fighter[a.Actor]
	if fi == nil {
		// should not happen in practice
		return fmt.Errorf("%s cannot use healing potions.", g.ECS.Name[a.Actor])
	}
	hp := fi.Heal(pt.Amount)
	if hp <= 0 {
		return errors.New("Your health is already full.")
	}
	if a.Actor == g.ECS.PlayerID {
		g.Logf("You regained %d HP", ColorLogItemUse, hp)
	}
	return nil
}
