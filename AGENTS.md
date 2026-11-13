@README.md

use go and look at these libs/packages:

- @~/proj/oden 
- @~/proj/ochazuke
- @~/proj/matomeru

i want to rewrite the scripts in the ./skill/ dir
using the listed libs.

it should have several subcommands for doing various things:

- list themes
- convert mmd files to:
  - svg
  - png
  - etc?

## Where does X live

| Concern | File |
|---|---|
| `main`, `Main`, the oden app and the `pick` command | `main.go` |
| The ochazuke runtime: the `theme` (facts) and `diagram` (command) nouns, their verbs, the bare words `themes`, `svg`, `png`, `html` | `app.go` |
| Converting and wrapping: options, output rules, the file writer | `convert.go` |
| The file picker (a TUI page, so a command, not a verb) | `picker.go` |
| The renderer: embedded engine, themes, headless Chrome | `renderer/` |

Every command-line word that is not `pick` is a verb of a noun; a new one is a verb (and an alias for a shortcut), not an oden command. `-` as `--output` writes the bytes through `App.raw`, which replaces the shell's JSON.

## Plans and todos

Keep planning work in the family planning workspace, separate from this repository. Do not add a TODO.md or docs/plans/ to this repo.
