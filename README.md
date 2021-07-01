# Gruid Go Roguelike Tutorial

This tutorial follows the overall structure of the [TCOD Python
Tutorial](http://rogueliketutorials.com/tutorials/tcod/v2), but makes use of
the [Go programming language](https://golang.org/) and the
[gruid](https://github.com/anaseto/gruid) roguelike game framework, instead of
TCOD.

[Table of Contents](https://github.com/anaseto/gruid-rltuto)

# Part 0 - Setting Up (before any code)

You need to install [Go](https://golang.org/) and
[SDL2](https://libsdl.org/download-2.0.php). For example, On Ubuntu or Debian,
it can be done with the following command:

```
sudo apt-get install libsdl2-dev go
```

The SDL2 library is only necessary if you want to use a
graphical driver, and not a terminal one. The tutorial will
normally assume you want to use SDL2, because it is more
versatile, allowing both for tiles and ASCII. If you want to use the terminal
driver, you may want to have a look at
[gruid-examples](https://github.com/anaseto/gruid-examples), which has several examples
using different drivers (terminal, SDL and browser).

If you intend to run the code of the tutorial, you should clone it with git.
You can do this from the web UI, or on the terminal, for example:

```
git clone https://github.com/anaseto/gruid-rltuto
```

clones the repository, and then you can check out a particular part:

```
git checkout part-1 
```

# Part 1 - Drawing the “@” symbol and moving it around

In this part, we will draw a `@` symbol on a window, and move it in response to
user input.

We perform three steps:

- We use the module `tiles` from *gruid* to set up the drawing of fonts into
  tiles in the file `tiles.go`.
- We define a main *gruid* Model for the application in `model.go`, that is a
  type that provides an `Update` method, which updates the game's state in
  response to user input messages, and a `Draw` method, which draws the current
  game's state into a grid. The `actions.go` file describes the behavior of the
  UI actions from `Update`.
- We define a `main` function in `main.go` that will run the defined *gruid*
  model using the SDL2 driver from the
  [gruid-sdl](https://github.com/anaseto/gruid-sdl) module.

You can then test the game for example by running `go run .` on the tutorial's
directory.  This may take a while the first time, as Go library dependencies
will be downloaded and installed.

* * *

[Next Part](https://github.com/anaseto/gruid-rltuto/tree/part-2)
