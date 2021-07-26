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
	g.Logf("A lightning bolt strikes %v.", ColorLogItemUse, g.ECS.GetName(target))
	g.ECS.Fighter[target].HP -= sc.Damage
	return nil
}

// Targetter describes consumables (or other kind of activables) that need
// a target in order to be used.
type Targetter interface {
	// TODO: could be expanded to distinguish between different kind of
	// targetting requirements.
	NeedsTargeting()
}

// ConfusionScroll is an item that can be invoked to confuse an enemy.
type ConfusionScroll struct {
	Turns int
}

func (sc *ConfusionScroll) Activate(g *game, a itemAction) error {
	if a.Target == nil {
		return errors.New("You have to chose a target.")
	}
	p := *a.Target
	if !g.InFOV(p) {
		return errors.New("You cannot target what you cannot see.")
	}
	if p == g.ECS.PP() {
		return errors.New("You cannot confuse yourself.")
	}
	i := g.ECS.MonsterAt(p)
	if i <= 0 || !g.ECS.Alive(i) {
		return errors.New("You have to target a monster.")
	}
	g.Logf("%s looks confused (scroll).", ColorLogItemUse, g.ECS.GetName(i))
	g.ECS.PutStatus(i, StatusConfused, sc.Turns)
	return nil
}

func (sc *ConfusionScroll) NeedsTargeting() {}
