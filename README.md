# Gruid Go Roguelike Tutorial

This tutorial follows the overall structure of the [TCOD Python
Tutorial](http://rogueliketutorials.com/tutorials/tcod/v2), but makes use of
the [Go programming language](https://golang.org/) and the
[gruid](https://github.com/anaseto/gruid) roguelike game framework, instead of
TCOD.

[Table of Contents](https://github.com/anaseto/gruid-rltuto)

## Part 7 - Creating the Interface

In this part, we create the interface. There will be two lines at the top with
history messages, and a line at the bottom with status information showing the
player's HP. We will also be able to use the `m` key to view pasts messages
using a pager, and show monster's names by moving the mouse over them.

Most changes are quite easy in this part, relying mainly on using a few widgets
from gruid's `ui` package.

In `game.go`, we add a new `Log` field to the `game` type that will hold all
the message entries, and update logging calls to call functions that we will
define later in a new `log.go` file.

``` diff
diff --git a/game.go b/game.go
index caa36c3..db69a5f 100644
--- a/game.go
+++ b/game.go
@@ -5,7 +5,6 @@ package main
 
 import (
 	"fmt"
-	"log"
 	"strings"
 
 	"github.com/anaseto/gruid"
@@ -17,6 +16,7 @@ type game struct {
 	ECS *ECS             // entities present on the map
 	Map *Map             // the game map, made of tiles
 	PR  *paths.PathRange // path range for the map
+	Log []LogEntry       // log entries
 }
 
 // SpawnMonsters adds some monsters in the current map.
@@ -116,10 +116,14 @@ func (g *game) BumpAttack(i, j int) {
 	fj := g.ECS.Fighter[j]
 	damage := fi.Power - fj.Defense
 	attackDesc := fmt.Sprintf("%s attacks %s", strings.Title(g.ECS.Name[i]), g.ECS.Name[j])
+	color := ColorLogMonsterAttack
+	if i == g.ECS.PlayerID {
+		color = ColorLogPlayerAttack
+	}
 	if damage > 0 {
-		log.Printf("%s for %d damage", attackDesc, damage)
+		g.Logf("%s for %d damage", color, attackDesc, damage)
 		fj.HP -= damage
 	} else {
-		log.Printf("%s but does no damage", attackDesc)
+		g.Logf("%s but does no damage", color, attackDesc)
 	}
 }
```

We create a new file `log.go` that does logging of messages, handling
duplicates and colors. For example if two orcs attack you with same damage and
message, only one message will be shown with `(2x)` appended.

``` go
// This file handles the player's log.

package main

import (
	"fmt"

	"github.com/anaseto/gruid"
	"github.com/anaseto/gruid/ui"
)

// LogEntry contains information about a log entry.
type LogEntry struct {
	Text  string      // entry text
	Color gruid.Color // color
	Dups  int         // consecutive duplicates of same message
}

func (e LogEntry) String() string {
	if e.Dups == 0 {
		return e.Text
	}
	return fmt.Sprintf("%s (%d×)", e.Text, e.Dups)
}

// Log adds an entry to the player's log.
func (g *game) log(e LogEntry) {
	if len(g.Log) > 0 {
		if g.Log[len(g.Log)-1].Text == e.Text {
			g.Log[len(g.Log)-1].Dups++
			return
		}
	}
	g.Log = append(g.Log, e)
}

// Logf adds a formatted entry to the game log.
func (g *game) Logf(format string, color gruid.Color, a ...interface{}) {
	e := LogEntry{Text: fmt.Sprintf(format, a...), Color: color}
	g.log(e)
}

// InitializeHistoryViewer creates a new pager for viewing message's history.
func (m *model) InitializeMessageViewer() {
	m.viewer = ui.NewPager(ui.PagerConfig{
		Grid: gruid.NewGrid(UIWidth, UIHeight-1),
		Box:  &ui.Box{},
	})
}
```


We define a few constants for whole UI and map dimensions in `main.go`:

``` diff
diff --git a/main.go b/main.go
index f5d4705..7da0d56 100644
--- a/main.go
+++ b/main.go
@@ -9,9 +9,16 @@ import (
 	sdl "github.com/anaseto/gruid-sdl"
 )
 
+const (
+	UIWidth   = 80
+	UIHeight  = 24
+	MapWidth  = UIWidth
+	MapHeight = UIHeight - 3
+)
+
 func main() {
 	// Create a new grid with standard 80x24 size.
-	gd := gruid.NewGrid(80, 24)
+	gd := gruid.NewGrid(UIWidth, UIHeight)
 	// Create the main application's model, using grid gd.
 	m := &model{grid: gd}
 	// Get a TileManager for drawing fonts on the screen.
```

We write the drawing code for the new UI elements in `model.go`. The `model`
type has several new fields for text labels and the message history viewer, as
well as a new `mode` field and type. This mode keeps track of the current UI
state: whether we are currently in playing mode (move and attack), at the end
of the game (the player died), or using the pager to view past messages.
Depending on which mode is currently on, we send input messages to different
handlers. We also add a new `mousePos` field in which we record mouse position
changes, in order to check in `Draw` whether the mouse is over a monster or
not, to show its name in a box next to the monster.

``` diff
diff --git a/model.go b/model.go
index 04e5fd3..55771f7 100644
--- a/model.go
+++ b/model.go
@@ -6,27 +6,68 @@ package main
 
 import (
 	"sort"
+	"strings"
+	"unicode/utf8"
 
 	"github.com/anaseto/gruid"
 	"github.com/anaseto/gruid/paths"
+	"github.com/anaseto/gruid/ui"
 )
 
 // model represents our main application's state.
 type model struct {
-	grid   gruid.Grid // drawing grid
-	game   *game      // game state
-	action action     // UI action
+	grid     gruid.Grid  // drawing grid
+	game     *game       // game state
+	action   action      // UI action
+	mode     mode        // UI mode
+	log      *ui.Label   // label for log
+	status   *ui.Label   // label for status
+	desc     *ui.Label   // label for position description
+	viewer   *ui.Pager   // message's history viewer
+	mousePos gruid.Point // mouse position
 }
 
+// mode describes distinct kinds of modes for the UI
+type mode int
+
+const (
+	modeNormal mode = iota
+	modeEnd         // win or death (currently only death)
+	modeMessageViewer
+)
+
 // Update implements gruid.Model.Update. It handles keyboard and mouse input
 // messages and updates the model in response to them.
 func (m *model) Update(msg gruid.Msg) gruid.Effect {
 	m.action = action{} // reset last action information
+	switch m.mode {
+	case modeEnd:
+		switch msg := msg.(type) {
+		case gruid.MsgKeyDown:
+			switch msg.Key {
+			case "q", gruid.KeyEscape:
+				// You died: quit on "q" or "escape"
+				return gruid.End()
+			}
+		}
+		return nil
+	case modeMessageViewer:
+		m.viewer.Update(msg)
+		if m.viewer.Action() == ui.PagerQuit {
+			m.mode = modeNormal
+		}
+		return nil
+	}
 	switch msg := msg.(type) {
 	case gruid.MsgInit:
+		m.log = &ui.Label{}
+		m.status = &ui.Label{}
+		m.desc = &ui.Label{Box: &ui.Box{}}
+		m.InitializeMessageViewer()
 		m.game = &game{}
 		// Initialize map
-		size := m.grid.Size() // map size: for now the whole window
+		size := m.grid.Size()
+		size.Y -= 3 // for log and status
 		m.game.Map = NewMap(size)
 		m.game.PR = paths.NewPathRange(gruid.NewRange(0, 0, size.X, size.Y))
 		// Initialize entities
@@ -36,13 +77,17 @@ func (m *model) Update(msg gruid.Msg) gruid.Effect {
 		m.game.ECS.Fighter[m.game.ECS.PlayerID] = &fighter{
 			HP: 30, MaxHP: 30, Power: 5, Defense: 2,
 		}
-		m.game.ECS.Name[m.game.ECS.PlayerID] = "you"
+		m.game.ECS.Name[m.game.ECS.PlayerID] = "player"
 		m.game.UpdateFOV()
 		// Add some monsters
 		m.game.SpawnMonsters()
 	case gruid.MsgKeyDown:
 		// Update action information on key down.
 		m.updateMsgKeyDown(msg)
+	case gruid.MsgMouse:
+		if msg.Action == gruid.MouseMove {
+			m.mousePos = msg.P
+		}
 	}
 	// Handle action (if any).
 	return m.handleAction()
@@ -63,6 +108,8 @@ func (m *model) updateMsgKeyDown(msg gruid.MsgKeyDown) {
 		m.action = action{Type: ActionWait}
 	case gruid.KeyEscape, "q":
 		m.action = action{Type: ActionQuit}
+	case "m":
+		m.action = action{Type: ActionViewMessages}
 	}
 }
 
@@ -72,12 +119,22 @@ const (
 	ColorFOV gruid.Color = iota + 1
 	ColorPlayer
 	ColorMonster
+	ColorLogPlayerAttack
+	ColorLogMonsterAttack
+	ColorLogSpecial
+	ColorStatusHealthy
+	ColorStatusWounded
 )
 
 // Draw implements gruid.Model.Draw. It draws a simple map that spans the whole
 // grid.
 func (m *model) Draw() gruid.Grid {
+	if m.mode == modeMessageViewer {
+		m.grid.Copy(m.viewer.Draw())
+		return m.grid
+	}
 	m.grid.Fill(gruid.Cell{Rune: ' '})
+	mapgrid := m.grid.Slice(m.grid.Range().Shift(0, 2, 0, -1))
 	g := m.game
 	// We draw the map tiles.
 	it := g.Map.Grid.Iterator()
@@ -89,7 +146,7 @@ func (m *model) Draw() gruid.Grid {
 		if g.InFOV(it.P()) {
 			c.Style.Bg = ColorFOV
 		}
-		m.grid.Set(it.P(), c)
+		mapgrid.Set(it.P(), c)
 	}
 	// We sort entity indexes using the render ordering.
 	sortedEntities := make([]int, 0, len(g.ECS.Entities))
@@ -105,11 +162,92 @@ func (m *model) Draw() gruid.Grid {
 		if !g.Map.Explored[p] || !g.InFOV(p) {
 			continue
 		}
-		c := m.grid.At(p)
+		c := mapgrid.At(p)
 		c.Rune, c.Style.Fg = g.ECS.Style(i)
-		m.grid.Set(p, c)
+		mapgrid.Set(p, c)
 		// NOTE: We retrieved current cell at e.Pos() to preserve
 		// background (in FOV or not).
 	}
+	m.DrawNames(mapgrid)
+	m.DrawLog(m.grid.Slice(m.grid.Range().Lines(0, 2)))
+	m.DrawStatus(m.grid.Slice(m.grid.Range().Line(m.grid.Size().Y - 1)))
 	return m.grid
 }
+
+// DrawLog draws the last two lines of the log.
+func (m *model) DrawLog(gd gruid.Grid) {
+	j := 1
+	for i := len(m.game.Log) - 1; i >= 0; i-- {
+		if j < 0 {
+			break
+		}
+		e := m.game.Log[i]
+		st := gruid.Style{}
+		st.Fg = e.Color
+		m.log.Content = ui.NewStyledText(e.String(), st)
+		m.log.Draw(gd.Slice(gd.Range().Line(j)))
+		j--
+	}
+}
+
+// DrawStatus draws the status line
+func (m *model) DrawStatus(gd gruid.Grid) {
+	st := gruid.Style{}
+	st.Fg = ColorStatusHealthy
+	g := m.game
+	f := g.ECS.Fighter[g.ECS.PlayerID]
+	if f.HP < f.MaxHP/2 {
+		st.Fg = ColorStatusWounded
+	}
+	m.log.Content = ui.Textf("HP: %d/%d", f.HP, f.MaxHP).WithStyle(st)
+	m.log.Draw(gd)
+}
+
+// DrawNames renders the names of the named entities at current mouse location
+// if it is in the map.
+func (m *model) DrawNames(gd gruid.Grid) {
+	maprg := gruid.NewRange(0, 2, UIWidth, UIHeight-1)
+	if !m.mousePos.In(maprg) {
+		return
+	}
+	p := m.mousePos.Sub(gruid.Point{0, 2})
+	// We get the names of the entities at p.
+	names := []string{}
+	for i, q := range m.game.ECS.Positions {
+		if q != p || !m.game.InFOV(q) {
+			continue
+		}
+		name, ok := m.game.ECS.Name[i]
+		if ok {
+			if m.game.ECS.Alive(i) {
+				names = append(names, name)
+			} else {
+				names = append(names, "corpse")
+			}
+		}
+	}
+	if len(names) == 0 {
+		return
+	}
+	// We sort the names. This could be improved to sort by entity type
+	// too, as well as to remove duplicates (for example showing “corpse
+	// (3x)” if there are three corpses).
+	sort.Strings(names)
+
+	text := strings.Join(names, ", ")
+	width := utf8.RuneCountInString(text) + 2
+	rg := gruid.NewRange(p.X+1, p.Y-1, p.X+1+width, p.Y+2)
+	// we adjust a bit the box's placement in case it's on a edge.
+	if p.X+1+width >= UIWidth {
+		rg = rg.Shift(-1-width, 0, -1-width, 0)
+	}
+	if p.Y+2 > MapHeight {
+		rg = rg.Shift(0, -1, 0, -1)
+	}
+	if p.Y-1 < 0 {
+		rg = rg.Shift(0, 1, 0, 1)
+	}
+	slice := gd.Slice(rg)
+	m.desc.Content = ui.Text(text)
+	m.desc.Draw(slice)
+}
```

Also, we add a couple of new colors, and new RGB values in `tiles.go`.

``` diff
diff --git a/tiles.go b/tiles.go
index eeb7675..9a945c0 100644
--- a/tiles.go
+++ b/tiles.go
@@ -39,6 +39,12 @@ func (t *TileDrawer) GetImage(c gruid.Cell) image.Image {
 		fg = image.NewUniform(color.RGBA{0x46, 0x95, 0xf7, 255})
 	case ColorMonster:
 		fg = image.NewUniform(color.RGBA{0xfa, 0x57, 0x50, 255})
+	case ColorLogPlayerAttack, ColorStatusHealthy:
+		fg = image.NewUniform(color.RGBA{0x75, 0xb9, 0x38, 255})
+	case ColorLogMonsterAttack, ColorStatusWounded:
+		fg = image.NewUniform(color.RGBA{0xed, 0x86, 0x49, 255})
+	case ColorLogSpecial:
+		fg = image.NewUniform(color.RGBA{0xf2, 0x75, 0xbe, 255})
 	}
 	// We return an image with the given rune drawn using the previously
 	// defined foreground and background colors.
```

Finally, we add a new UI action in `actions.go` for opening the messages viewer
with past message lines.

``` diff
diff --git a/actions.go b/actions.go
index ccec92d..723ed81 100644
--- a/actions.go
+++ b/actions.go
@@ -3,9 +3,8 @@
 package main
 
 import (
-	"log"
-
 	"github.com/anaseto/gruid"
+	"github.com/anaseto/gruid/ui"
 )
 
 // action represents information relevant to the last UI action performed.
@@ -18,10 +17,11 @@ type actionType int
 
 // These constants represent the possible UI actions.
 const (
-	NoAction   actionType = iota
-	ActionBump            // bump request (attack or movement)
-	ActionWait            // wait a turn
-	ActionQuit            // quit the game
+	NoAction           actionType = iota
+	ActionBump                    // bump request (attack or movement)
+	ActionWait                    // wait a turn
+	ActionQuit                    // quit the game
+	ActionViewMessages            // view history messages
 )
 
 // handleAction updates the model in response to current recorded last action.
@@ -36,10 +36,20 @@ func (m *model) handleAction() gruid.Effect {
 		// for now, just terminate with gruid End command: this will
 		// have to be updated later when implementing saving.
 		return gruid.End()
+	case ActionViewMessages:
+		m.mode = modeMessageViewer
+		lines := []ui.StyledText{}
+		for _, e := range m.game.Log {
+			st := gruid.Style{}
+			st.Fg = e.Color
+			lines = append(lines, ui.NewStyledText(e.String(), st))
+		}
+		m.viewer.SetLines(lines)
 	}
 	if m.game.ECS.PlayerDied() {
-		log.Print("You died")
-		return gruid.End()
+		m.game.Logf("You died -- press “q” or escape to quit", ColorLogSpecial)
+		m.mode = modeEnd
+		return nil
 	}
 	return nil
 }
```
