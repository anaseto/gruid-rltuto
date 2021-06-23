// This is the main file of tutorial. It defines the main routine.
package main

import (
	"context"
	"log"

	"github.com/anaseto/gruid"
	sdl "github.com/anaseto/gruid-sdl"
)

func main() {
	// Create a new grid with standard 80x24 size.
	gd := gruid.NewGrid(80, 24)
	// Create the main application's model, using grid gd.
	m := &model{grid: gd}
	// Get a TileManager for drawing fonts on the screen.
	t, err := GetTileDrawer()
	if err != nil {
		log.Fatal(err)
	}
	// Use the SDL2 driver from gruid-sdl, using the previously defined
	// TileManager.
	dr := sdl.NewDriver(sdl.Config{
		TileManager: t,
	})

	// Define new application using the SDL2 gruid driver and our model.
	app := gruid.NewApp(gruid.AppConfig{
		Driver: dr,
		Model:  m,
	})

	// Start the application.
	if err := app.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
}
