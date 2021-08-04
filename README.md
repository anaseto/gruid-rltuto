# Gruid Go Roguelike Tutorial

This tutorial follows the overall structure of the [TCOD Python
Tutorial](http://rogueliketutorials.com/tutorials/tcod/v2), but makes use of
the [Go programming language](https://golang.org/) and the
[gruid](https://github.com/anaseto/gruid) roguelike game framework, instead of
TCOD.

[Table of Contents](https://github.com/anaseto/gruid-rltuto)

# Part 10 - Saving and Loading

In this part, we implement saving of games (by pressing `S`), and loading of
games via a new main game menu.

The saving and loading code is in `saving.go`. Because we use the `gob`
encoding package from the standard library, saving the current game is very
easy and handled almost automatically by the package. This is because we were
careful in our definition of the `game` type in the previous parts so that the
type supports well automatic serialization. The only exception was the random
number generator attached to the map, which we had to un-export in order to use
`gob`, so that it does not attempt at saving it (this could be improved in the
future by saving for example the seed, instead of the `Rand` structure which
cannot be encoded by `gob`).

``` go
// This file handles game saving.

package main

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
)

func init() {
	// We register Entity types so that gob can encode them.
	gob.Register(&Player{})
	gob.Register(&Monster{})
	gob.Register(&HealingPotion{})
	gob.Register(&LightningScroll{})
	gob.Register(&ConfusionScroll{})
	gob.Register(&FireballScroll{})
}

// EncodeGame uses the gob package of the standard library to encode the game
// so that it can be saved to a file.
func EncodeGame(g *game) ([]byte, error) {
	data := bytes.Buffer{}
	enc := gob.NewEncoder(&data)
	err := enc.Encode(g)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write(data.Bytes())
	w.Close()
	return buf.Bytes(), nil
}

// DecodeGame uses the gob package from the standard library to decode a saved
// game.
func DecodeGame(data []byte) (*game, error) {
	buf := bytes.NewReader(data)
	r, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}
	dec := gob.NewDecoder(r)
	g := &game{}
	err = dec.Decode(g)
	if err != nil {
		return nil, err
	}
	r.Close()
	return g, nil
}

// DataDir returns the directory for saving application's data, which depends
// on the platform. It builds the directory if it does not exist already.
func DataDir() (string, error) {
	var xdg string
	if runtime.GOOS == "windows" {
		// Windows
		xdg = os.Getenv("LOCALAPPDATA")
	} else {
		// Linux, BSD, etc.
		xdg = os.Getenv("XDG_DATA_HOME")
	}
	if xdg == "" {
		xdg = filepath.Join(os.Getenv("HOME"), ".local", "share")
	}
	dataDir := filepath.Join(xdg, "gruid-rltuto")
	_, err := os.Stat(dataDir)
	if err != nil {
		err = os.MkdirAll(dataDir, 0755)
		if err != nil {
			return dataDir, fmt.Errorf("building data directory: %v\n", err)
		}
	}
	return dataDir, nil
}

// SaveFile saves data to a file with a given filename. The data is first
// written to a temporary file and then renamed, to avoid corrupting any
// previous file with same filename in case of an error occurs while writing
// the file (for example due to an electric power outage).
func SaveFile(filename string, data []byte) error {
	dataDir, err := DataDir()
	if err != nil {
		return err
	}
	tempSaveFile := filepath.Join(dataDir, "temp-"+filename)
	f, err := os.OpenFile(tempSaveFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	saveFile := filepath.Join(dataDir, filename)
	if err := os.Rename(f.Name(), saveFile); err != nil {
		return err
	}
	return err
}

// LoadFile opens a file with given filename in the game's data directory, and
// returns its content or an error.
func LoadFile(filename string) ([]byte, error) {
	dataDir, err := DataDir()
	if err != nil {
		return nil, fmt.Errorf("could not read game's data directory: %s", dataDir)
	}
	fp := filepath.Join(dataDir, filename)
	_, err = os.Stat(fp)
	if err != nil {
		return nil, fmt.Errorf("no such file: %s", filename)
	}
	data, err := ioutil.ReadFile(fp)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// RemoveDataFile removes a file in the game's data directory.
func RemoveDataFile(filename string) error {
	dataDir, err := DataDir()
	if err != nil {
		return err
	}
	dataFile := filepath.Join(dataDir, filename)
	_, err = os.Stat(dataFile)
	if err == nil {
		err := os.Remove(dataFile)
		if err != nil {
			return err
		}
	}
	return nil
}
```

The rest of the changes are quite straightforward : we introduce a new model
mode for the main game menu, as well as a couple of new widgets (a widget for
the menu, and a label for displaying info) using package `ui` from gruid.

We moved new game creation to a separate function `NewGame`.

``` diff
diff --git a/actions.go b/actions.go
index 4390653..715f9c0 100644
--- a/actions.go
+++ b/actions.go
@@ -3,6 +3,8 @@
 package main
 
 import (
+	"log"
+
 	"github.com/anaseto/gruid"
 	"github.com/anaseto/gruid/ui"
 )
@@ -23,7 +25,8 @@ const (
 	ActionInventory               // inventory menu to use an item
 	ActionPickup                  // pickup an item on the ground
 	ActionWait                    // wait a turn
-	ActionQuit                    // quit the game
+	ActionQuit                    // quit the game (without saving)
+	ActionSave                    // save the game
 	ActionViewMessages            // view history messages
 	ActionExamine                 // examine map
 )
@@ -44,7 +47,20 @@ func (m *model) handleAction() gruid.Effect {
 		m.game.PickupItem()
 	case ActionWait:
 		m.game.EndTurn()
+	case ActionSave:
+		data, err := EncodeGame(m.game)
+		if err == nil {
+			err = SaveFile("save", data)
+		}
+		if err != nil {
+			m.game.Logf("Could not save game.", ColorLogSpecial)
+			log.Printf("could not save game: %v", err)
+			break
+		}
+		return gruid.End()
 	case ActionQuit:
+		// Remove any previously saved files (if any).
+		RemoveDataFile("save")
 		// for now, just terminate with gruid End command: this will
 		// have to be updated later when implementing saving.
 		return gruid.End()
diff --git a/game.go b/game.go
index 3ccf60a..919ce90 100644
--- a/game.go
+++ b/game.go
@@ -20,6 +20,31 @@ type game struct {
 	Log []LogEntry       // log entries
 }
 
+// NewGame initializes a new game.
+func NewGame() *game {
+	g := &game{}
+	size := gruid.Point{UIWidth, UIHeight}
+	size.Y -= 3 // for log and status
+	g.Map = NewMap(size)
+	g.PR = paths.NewPathRange(gruid.NewRange(0, 0, size.X, size.Y))
+	// Initialize entities
+	g.ECS = NewECS()
+	// Initialization: create a player entity centered on the map.
+	g.ECS.PlayerID = g.ECS.AddEntity(NewPlayer(), g.Map.RandomFloor())
+	g.ECS.Fighter[g.ECS.PlayerID] = &fighter{
+		HP: 30, MaxHP: 30, Power: 5, Defense: 2,
+	}
+	g.ECS.Style[g.ECS.PlayerID] = Style{Rune: '@', Color: ColorPlayer}
+	g.ECS.Name[g.ECS.PlayerID] = "player"
+	g.ECS.Inventory[g.ECS.PlayerID] = &Inventory{}
+	g.UpdateFOV()
+	// Add some monsters
+	g.SpawnMonsters()
+	// Add items
+	g.PlaceItems()
+	return g
+}
+
 // SpawnMonsters adds some monsters in the current map.
 func (g *game) SpawnMonsters() {
 	const numberOfMonsters = 12
@@ -33,7 +58,7 @@ func (g *game) SpawnMonsters() {
 		)
 		kind := orc
 		switch {
-		case g.Map.Rand.Intn(100) < 80:
+		case g.Map.rand.Intn(100) < 80:
 		default:
 			kind = troll
 		}
@@ -141,7 +166,7 @@ func (g *game) PlaceItems() {
 	const numberOfItems = 5
 	for i := 0; i < numberOfItems; i++ {
 		p := g.FreeFloorTile()
-		r := g.Map.Rand.Float64()
+		r := g.Map.rand.Float64()
 		switch {
 		case r < 0.7:
 			g.ECS.AddItem(&HealingPotion{Amount: 4}, p, "health potion", '!')
diff --git a/map.go b/map.go
index 6f021bc..0dd5225 100644
--- a/map.go
+++ b/map.go
@@ -20,7 +20,7 @@ const (
 // Map represents the rectangular map of the game's level.
 type Map struct {
 	Grid     rl.Grid
-	Rand     *rand.Rand           // random number generator
+	rand     *rand.Rand           // random number generator
 	Explored map[gruid.Point]bool // explored cells
 }
 
@@ -28,7 +28,7 @@ type Map struct {
 func NewMap(size gruid.Point) *Map {
 	m := &Map{
 		Grid:     rl.NewGrid(size.X, size.Y),
-		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
+		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
 		Explored: make(map[gruid.Point]bool),
 	}
 	m.Generate()
@@ -54,7 +54,7 @@ func (m *Map) Rune(c rl.Cell) (r rune) {
 // Generate fills the Grid attribute of m with a procedurally generated map.
 func (m *Map) Generate() {
 	// map generator using the rl package from gruid
-	mgen := rl.MapGen{Rand: m.Rand, Grid: m.Grid}
+	mgen := rl.MapGen{Rand: m.rand, Grid: m.Grid}
 	// cellular automata map generation with rules that give a cave-like
 	// map.
 	rules := []rl.CellularAutomataRule{
@@ -84,7 +84,7 @@ func (m *Map) Generate() {
 func (m *Map) RandomFloor() gruid.Point {
 	size := m.Grid.Size()
 	for {
-		freep := gruid.Point{m.Rand.Intn(size.X), m.Rand.Intn(size.Y)}
+		freep := gruid.Point{m.rand.Intn(size.X), m.rand.Intn(size.Y)}
 		if m.Grid.At(freep) == Floor {
 			return freep
 		}
diff --git a/model.go b/model.go
index d06362e..a20f480 100644
--- a/model.go
+++ b/model.go
@@ -5,12 +5,13 @@
 package main
 
 import (
+	"math/rand"
 	"sort"
 	"strings"
+	"time"
 	"unicode/utf8"
 
 	"github.com/anaseto/gruid"
-	"github.com/anaseto/gruid/paths"
 	"github.com/anaseto/gruid/ui"
 )
 
@@ -26,6 +27,8 @@ type model struct {
 	inventory *ui.Menu   // inventory menu
 	viewer    *ui.Pager  // message's history viewer
 	targ      targeting  // targeting information
+	gameMenu  *ui.Menu   // game's main menu
+	info      *ui.Label  // info label in main menu (for errors)
 }
 
 // targeting describes information related to examination or selection of
@@ -46,6 +49,7 @@ const (
 	modeEnd         // win or death (currently only death)
 	modeInventoryActivate
 	modeInventoryDrop
+	modeGameMenu
 	modeMessageViewer
 	modeTargeting   // targeting mode (item use)
 	modeExamination // keyboad map examination mode
@@ -54,8 +58,14 @@ const (
 // Update implements gruid.Model.Update. It handles keyboard and mouse input
 // messages and updates the model in response to them.
 func (m *model) Update(msg gruid.Msg) gruid.Effect {
+	switch msg.(type) {
+	case gruid.MsgInit:
+		return m.init()
+	}
 	m.action = action{} // reset last action information
 	switch m.mode {
+	case modeGameMenu:
+		return m.updateGameMenu(msg)
 	case modeEnd:
 		switch msg := msg.(type) {
 		case gruid.MsgKeyDown:
@@ -80,32 +90,6 @@ func (m *model) Update(msg gruid.Msg) gruid.Effect {
 		return nil
 	}
 	switch msg := msg.(type) {
-	case gruid.MsgInit:
-		m.log = &ui.Label{}
-		m.status = &ui.Label{}
-		m.desc = &ui.Label{Box: &ui.Box{}}
-		m.InitializeMessageViewer()
-		m.game = &game{}
-		// Initialize map
-		size := m.grid.Size()
-		size.Y -= 3 // for log and status
-		m.game.Map = NewMap(size)
-		m.game.PR = paths.NewPathRange(gruid.NewRange(0, 0, size.X, size.Y))
-		// Initialize entities
-		m.game.ECS = NewECS()
-		// Initialization: create a player entity centered on the map.
-		m.game.ECS.PlayerID = m.game.ECS.AddEntity(NewPlayer(), m.game.Map.RandomFloor())
-		m.game.ECS.Fighter[m.game.ECS.PlayerID] = &fighter{
-			HP: 30, MaxHP: 30, Power: 5, Defense: 2,
-		}
-		m.game.ECS.Style[m.game.ECS.PlayerID] = Style{Rune: '@', Color: ColorPlayer}
-		m.game.ECS.Name[m.game.ECS.PlayerID] = "player"
-		m.game.ECS.Inventory[m.game.ECS.PlayerID] = &Inventory{}
-		m.game.UpdateFOV()
-		// Add some monsters
-		m.game.SpawnMonsters()
-		// Add items
-		m.game.PlaceItems()
 	case gruid.MsgKeyDown:
 		// Update action information on key down.
 		m.updateMsgKeyDown(msg)
@@ -118,6 +102,72 @@ func (m *model) Update(msg gruid.Msg) gruid.Effect {
 	return m.handleAction()
 }
 
+const (
+	MenuNewGame = iota
+	MenuContinue
+	MenuQuit
+)
+
+// init initializes the model: widgets' initialization, and starting mode.
+func (m *model) init() gruid.Effect {
+	m.log = &ui.Label{}
+	m.status = &ui.Label{}
+	m.info = &ui.Label{}
+	m.desc = &ui.Label{Box: &ui.Box{}}
+	m.InitializeMessageViewer()
+	m.mode = modeGameMenu
+	entries := []ui.MenuEntry{
+		MenuNewGame:  {Text: ui.Text("(N)ew game"), Keys: []gruid.Key{"N", "n"}},
+		MenuContinue: {Text: ui.Text("(C)ontinue last game"), Keys: []gruid.Key{"C", "c"}},
+		MenuQuit:     {Text: ui.Text("(Q)uit")},
+	}
+	m.gameMenu = ui.NewMenu(ui.MenuConfig{
+		Grid:    gruid.NewGrid(UIWidth/2, len(entries)+2),
+		Box:     &ui.Box{Title: ui.Text("Gruid Roguelike Tutorial")},
+		Entries: entries,
+		Style:   ui.MenuStyle{Active: gruid.Style{}.WithFg(ColorMenuActive)},
+	})
+	return nil
+}
+
+// updateGameMenu updates the Game Menu and switchs mode to normal after
+// starting a new game or loading an old one.
+func (m *model) updateGameMenu(msg gruid.Msg) gruid.Effect {
+	rg := m.grid.Range().Intersect(m.grid.Range().Add(mainMenuAnchor))
+	m.gameMenu.Update(rg.RelMsg(msg))
+	switch m.gameMenu.Action() {
+	case ui.MenuMove:
+		m.info.SetText("")
+	case ui.MenuInvoke:
+		m.info.SetText("")
+		switch m.gameMenu.Active() {
+		case MenuNewGame:
+			m.game = NewGame()
+			m.mode = modeNormal
+		case MenuContinue:
+			data, err := LoadFile("save")
+			if err != nil {
+				m.info.SetText(err.Error())
+				break
+			}
+			g, err := DecodeGame(data)
+			if err != nil {
+				m.info.SetText(err.Error())
+				break
+			}
+			m.game = g
+			m.mode = modeNormal
+			// the random number generator is not saved
+			m.game.Map.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
+		case MenuQuit:
+			return gruid.End()
+		}
+	case ui.MenuQuit:
+		return gruid.End()
+	}
+	return nil
+}
+
 // updateTargeting updates targeting information in response to user input
 // messages.
 func (m *model) updateTargeting(msg gruid.Msg) {
@@ -223,8 +273,10 @@ func (m *model) updateMsgKeyDown(msg gruid.MsgKeyDown) {
 		m.action = action{Type: ActionBump, Delta: pdelta.Shift(1, 0)}
 	case gruid.KeyEnter, ".":
 		m.action = action{Type: ActionWait}
-	case gruid.KeyEscape, "q":
+	case "Q":
 		m.action = action{Type: ActionQuit}
+	case "S":
+		m.action = action{Type: ActionSave}
 	case "m":
 		m.action = action{Type: ActionViewMessages}
 	case "i":
@@ -251,6 +303,7 @@ const (
 	ColorStatusHealthy
 	ColorStatusWounded
 	ColorConsumable
+	ColorMenuActive
 )
 
 const (
@@ -262,6 +315,8 @@ const (
 func (m *model) Draw() gruid.Grid {
 	mapgrid := m.grid.Slice(m.grid.Range().Shift(0, LogLines, 0, -1))
 	switch m.mode {
+	case modeGameMenu:
+		return m.DrawGameMenu()
 	case modeMessageViewer:
 		m.grid.Copy(m.viewer.Draw())
 		return m.grid
@@ -309,6 +364,16 @@ func (m *model) Draw() gruid.Grid {
 	return m.grid
 }
 
+var mainMenuAnchor = gruid.Point{10, 6}
+
+// DrawGameMenu draws the game's main menu.
+func (m *model) DrawGameMenu() gruid.Grid {
+	m.grid.Fill(gruid.Cell{Rune: ' '})
+	m.grid.Slice(m.gameMenu.Bounds().Add(mainMenuAnchor)).Copy(m.gameMenu.Draw())
+	m.info.Draw(m.grid.Slice(m.grid.Range().Line(12).Shift(10, 0, 0, 0)))
+	return m.grid
+}
+
 // DrawLog draws the last two lines of the log.
 func (m *model) DrawLog(gd gruid.Grid) {
 	j := 1
diff --git a/tiles.go b/tiles.go
index b7580dc..e79a2bd 100644
--- a/tiles.go
+++ b/tiles.go
@@ -45,7 +45,7 @@ func (t *TileDrawer) GetImage(c gruid.Cell) image.Image {
 		fg = image.NewUniform(color.RGBA{0xed, 0x86, 0x49, 255})
 	case ColorLogSpecial:
 		fg = image.NewUniform(color.RGBA{0xf2, 0x75, 0xbe, 255})
-	case ColorConsumable:
+	case ColorConsumable, ColorMenuActive:
 		fg = image.NewUniform(color.RGBA{0xdb, 0xb3, 0x2d, 255})
 	}
 	if c.Style.Attrs&AttrReverse != 0 {
```
