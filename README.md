# Gruid Go Roguelike Tutorial

This tutorial follows the overall structure of the [TCOD Python
Tutorial](http://rogueliketutorials.com/tutorials/tcod/v2), but makes use of
the [Go programming language](https://golang.org/) and the
[gruid](https://github.com/anaseto gruid) roguelike game framework, instead of
TCOD.

For now, this tutorial also assumes a bit more familiarity with programming and
git: each part is a git branch and will come with a few explanations, but it's
expected that you read the code and comments and diffs between parts using git.

For example, you can [compare
changes](https://github.com/anaseto/gruid-rltuto/compare/part-1...part-2)
between two parts, or see the code of a [particular
part](https://github.com/anaseto/gruid-rltuto/tree/part-1).

# Part 0 - Setting Up (before any code)

You need to install [Go](https://golang.org/) and
[SDL2](https://libsdl.org/download-2.0.php SDL2) On Ubuntu or Debian:

```
sudo apt-get install libsdl2-dev
```

The SDL2 library is only necessary if you want to use a
graphical driver, and not a terminal one. The tutorial will
normally assume you want to use SDL2, because it is more
versatile, allowing both for tiles and ASCII. If you want to use the terminal
driver, you may want to have a look at
[gruid-examples](github.com/anaseto/gruid-examples), which has several examples
using different drivers (terminal, SDL and browser).

# Part 1 - Drawing the “@” symbol and moving it around

In this part, we will draw a `@` symbol on a window, and move it in response to
user input.

We will perform three steps:

- Use the module `tiles` from *gruid* to set up the drawing of fonts into
  tiles in the file `tiles.go`.
- We define a main *gruid* model for the application in `model.go`, that is a
  type that provides an `Update` method, that updates the game's state in
  response to user input messages, and a `Draw` method, that draws the current
  game's state into a grid. The `actions.go` file describes the behavior of the
  UI actions from `Update`.
- We define a `main` function in `main.go` that will run the defined *gruid*
  model using the SDL2 driver from the
  [gruid-sdl](https://github.com/anaseto/gruid-sdl) module.
