// This file describes item entities.

package main

import (
	"errors"
	"fmt"

	"github.com/anaseto/gruid"
	"github.com/anaseto/gruid/paths"
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

// LightningScroll is an item that can be invoked to strike the closest enemy
// within a particular range.
type LightningScroll struct {
	Range  int
	Damage int
}

func (sc *LightningScroll) Activate(g *game, a itemAction) error {
	target := -1
	minDist := sc.Range + 1
	for i := range g.ECS.Fighter {
		p := g.ECS.Positions[i]
		if i == a.Actor || g.ECS.Dead(i) || !g.InFOV(p) {
			continue
		}
		dist := paths.DistanceManhattan(p, g.ECS.Positions[a.Actor])
		if dist < minDist {
			target = i
			minDist = dist
		}
	}
	if target < 0 {
		return errors.New("No enemy within range.")
	}
	g.Logf("A lightning bolt strikes %v.", ColorLogItemUse, g.ECS.Name[target])
	g.ECS.Fighter[target].HP -= sc.Damage
	return nil
}
