# Gruid Go Roguelike Tutorial

This tutorial follows the overall structure of the [TCOD Python
Tutorial](http://rogueliketutorials.com/tutorials/tcod/v2), but makes use of
the [Go programming language](https://golang.org/) and the
[gruid](https://github.com/anaseto/gruid) roguelike game framework, instead of
TCOD.

For now, this tutorial also assumes a bit more familiarity with programming and
git: each part is a git branch and will come with a few explanations, but it's
expected that you read the code and comments and diffs between parts using git.

For example, you can [compare
changes](https://github.com/anaseto/gruid-rltuto/compare/part-1...part-2)
between two parts, or see the code of a [particular
part](https://github.com/anaseto/gruid-rltuto/tree/part-1).

# Part 2 - Generic entities, and the map

In this part, we introduce the `Entity` interface in a new file `entity.go`,
which will represent any kind of entities that can be placed on the map. A type
satisfying the `Entity` interface should have several methods that give
information on position and display. As a first example, we introduce a
`Player` type implementing the `Entity` interface.

We also introduce a `Map` type for representing the map in `map.go`. We define
`Wall` and `Floor` tiles, and give a graphical representation to them.

We then adjust the code of the `Draw` method in `model.go` to take into account
the new representation of entities and the map. We first draw the map, and then
we place entities.
