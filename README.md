**Migrated to https://codeberg.org/anaseto/gruid-rltuto because of 2FA requirement**

# Gruid Go Roguelike Tutorial

This tutorial follows the overall structure of the [TCOD Python
Tutorial](http://rogueliketutorials.com/tutorials/tcod/v2), but makes use of
the [Go programming language](https://golang.org/) and the
[gruid](https://github.com/anaseto/gruid) roguelike game framework, instead of
TCOD.

## Table of Contents

* [Part 0 & 1](https://github.com/anaseto/gruid-rltuto/tree/part-1) - Setting Up & Drawing the “@” symbol and moving it around
* [Part 2](https://github.com/anaseto/gruid-rltuto/tree/part-2) - Generic entities and the map
* [Part 3](https://github.com/anaseto/gruid-rltuto/tree/part-3) - Generating a Dungeon
* [Part 4](https://github.com/anaseto/gruid-rltuto/tree/part-4) - Field of View
* [Part 5](https://github.com/anaseto/gruid-rltuto/tree/part-5) - Placing enemies and kicking them (harmlessly)
* [Part 6](https://github.com/anaseto/gruid-rltuto/tree/part-6) - Doing (and taking) some damage
* [Part 7](https://github.com/anaseto/gruid-rltuto/tree/part-7) - Creating the Interface
* [Part 8](https://github.com/anaseto/gruid-rltuto/tree/part-8) - Items and Inventory
* [Part 9](https://github.com/anaseto/gruid-rltuto/tree/part-9) - Ranged Scrolls and Targeting
* [Part 10](https://github.com/anaseto/gruid-rltuto/tree/part-10) - Saving and Loading

## Tips & Remarks

This tutorial assumes some familiarity with programming and git: each part is a
git branch and will come with a few explanations, but it's expected that you
read the code and comments and diffs between parts using git.

You can do some simple operations on the web, like [compare
changes](https://github.com/anaseto/gruid-rltuto/compare/part-1...part-2)
between two parts, or view the code of a [particular
part](https://github.com/anaseto/gruid-rltuto/tree/part-1).

Assuming you've followed the set up instructions of [Part
0](https://github.com/anaseto/gruid-rltuto/tree/part-1), you may want to clone
locally the tutorial's repository to explore:

``` sh
# Clone the repository in a new directory gruid-rltuto:
git clone https://github.com/anaseto/gruid-rltuto
cd gruid-rltuto
# You can then use git on the command line to switch between parts:
git checkout part-1
# View code changes between two parts:
git diff part-1..part-2 *.go
# View changes between two parts for a specific file:
git diff part-1..part-2 model.go
# Run the code of the current branch "part-1":
go run .
```

*Note:* The code in this repository can be used under the same permissive
license as gruid, or as a public domain work
[CC0](https://creativecommons.org/publicdomain/zero/1.0/), as you prefer. In
short, you can do whatever you want with it.
