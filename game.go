// This file handles game related affairs that are not specific to entities or
// the map.

package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/anaseto/gruid"
	"github.com/anaseto/gruid/paths"
)

// game represents information relevant the current game's state.
type game struct {
	ECS *ECS             // entities present on the map
	Map *Map             // the game map, made of tiles
	PR  *paths.PathRange // path range for the map
	Log []LogEntry       // log entries
}

// SpawnMonsters adds some monsters in the current map.
func (g *game) SpawnMonsters() {
	const numberOfMonsters = 12
	for i := 0; i < numberOfMonsters; i++ {
		m := &Monster{}
		// We generate either an orc or a troll with 0.8 and 0.2
		// probabilities respectively.
		const (
			orc = iota
			troll
		)
		kind := orc
		switch {
		case g.Map.Rand.Intn(100) < 80:
		default:
			kind = troll
		}
		p := g.FreeFloorTile()
		i := g.ECS.AddEntity(m, p)
		switch kind {
		case orc:
			g.ECS.Fighter[i] = &fighter{
				HP: 10, MaxHP: 10, Defense: 0, Power: 3,
			}
			g.ECS.Name[i] = "orc"
			g.ECS.DStyle[i] = EStyle{Rune: 'o', Color: ColorMonster}
		case troll:
			g.ECS.Fighter[i] = &fighter{
				HP: 16, MaxHP: 16, Defense: 1, Power: 4,
			}
			g.ECS.Name[i] = "troll"
			g.ECS.DStyle[i] = EStyle{Rune: 'T', Color: ColorMonster}
		}
		g.ECS.AI[i] = &AI{}
	}
}

// FreeFloorTile returns a free floor tile in the map (it assumes it exists).
func (g *game) FreeFloorTile() gruid.Point {
	for {
		p := g.Map.RandomFloor()
		if g.ECS.NoBlockingEntityAt(p) {
			return p
		}
	}
}

// EndTurn is called when the player's turn ends. Currently, the player and
// monsters have all the same speed, so we make each monster act each time the
// player's does an action that ends a turn.
func (g *game) EndTurn() {
	g.UpdateFOV()
	for i, e := range g.ECS.Entities {
		if g.ECS.PlayerDied() {
			return
		}
		switch e.(type) {
		case *Monster:
			g.HandleMonsterTurn(i)
		}
	}
}

// UpdateFOV updates the field of view.
func (g *game) UpdateFOV() {
	player := g.ECS.Player()
	// player position
	pp := g.ECS.PP()
	// We shift the FOV's Range so that it will be centered on the new
	// player's position.
	rg := gruid.NewRange(-maxLOS, -maxLOS, maxLOS+1, maxLOS+1)
	player.FOV.SetRange(rg.Add(pp).Intersect(g.Map.Grid.Range()))
	// We mark cells in field of view as explored. We use the symmetric
	// shadow casting algorithm provided by the rl package.
	passable := func(p gruid.Point) bool {
		return g.Map.Grid.At(p) != Wall
	}
	for _, p := range player.FOV.SSCVisionMap(pp, maxLOS, passable, false) {
		if paths.DistanceManhattan(p, pp) > maxLOS {
			continue
		}
		if !g.Map.Explored[p] {
			g.Map.Explored[p] = true
		}
	}
}

// InFOV returns true if p is in the player's field of view. We only keep cells
// within maxLOS manhattan distance from the player, as natural given our
// current 4-way movement. With 8-way movement, the natural distance choice
// would be the Chebyshev one.
func (g *game) InFOV(p gruid.Point) bool {
	pp := g.ECS.PP()
	return g.ECS.Player().FOV.Visible(p) &&
		paths.DistanceManhattan(pp, p) <= maxLOS
}

// BumpAttack implements attack of a fighter entity on another.
func (g *game) BumpAttack(i, j int) {
	fi := g.ECS.Fighter[i]
	fj := g.ECS.Fighter[j]
	damage := fi.Power - fj.Defense
	attackDesc := fmt.Sprintf("%s attacks %s", strings.Title(g.ECS.Name[i]), g.ECS.Name[j])
	color := ColorLogMonsterAttack
	if i == g.ECS.PlayerID {
		color = ColorLogPlayerAttack
	}
	if damage > 0 {
		g.Logf("%s for %d damage", color, attackDesc, damage)
		fj.HP -= damage
	} else {
		g.Logf("%s but does no damage", color, attackDesc)
	}
}

// PlaceItems adds items in the current map.
func (g *game) PlaceItems() {
	const numberOfPotions = 5
	for i := 0; i < numberOfPotions; i++ {
		p := g.FreeFloorTile()
		id := g.ECS.AddEntity(&HealingPotion{Amount: 4}, p)
		g.ECS.Name[id] = "health potion"
		g.ECS.DStyle[id] = EStyle{Rune: '!', Color: ColorConsumable}
	}
}

const ErrNoShow = "ErrNoShow"

// IventoryAdd adds an item to the player's inventory, if there is room. It
// returns an error if the item could not be added.
func (g *game) InventoryAdd(actor, i int) error {
	const maxSize = 26
	switch g.ECS.Entities[i].(type) {
	case Consumable:
		inv := g.ECS.Inventory[actor]
		if len(inv.Items) >= maxSize {
			return errors.New("Inventory is full.")
		}
		inv.Items = append(inv.Items, i)
		delete(g.ECS.Positions, i)
		return nil
	}
	return errors.New(ErrNoShow)
}

// Drop an item from the inventory.
func (g *game) InventoryRemove(actor, n int) error {
	inv := g.ECS.Inventory[actor]
	if len(inv.Items) <= n {
		return errors.New("Empty slot.")
	}
	i := inv.Items[n]
	inv.Items[n] = inv.Items[len(inv.Items)-1]
	inv.Items = inv.Items[:len(inv.Items)-1]
	g.ECS.Positions[i] = g.ECS.PP()
	return nil
}

// InventoryActivate uses a given item from the inventory.
func (g *game) InventoryActivate(actor, n int) error {
	inv := g.ECS.Inventory[actor]
	if len(inv.Items) <= n {
		return errors.New("Empty slot.")
	}
	i := inv.Items[n]
	switch e := g.ECS.Entities[i].(type) {
	case Consumable:
		err := e.Activate(g, itemAction{Actor: actor})
		if err != nil {
			return err
		}
	}
	// Put the last item on the previous one: this could be improved,
	// sorting elements in a certain way, or moving elements as necessary
	// to preserve current order.
	inv.Items[n] = inv.Items[len(inv.Items)-1]
	inv.Items = inv.Items[:len(inv.Items)-1]
	return nil
}
