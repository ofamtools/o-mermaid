<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/logo-animated-dark.svg">
    <img alt="o-mermaid" src="assets/logo-animated-light.svg" width="360">
  </picture>
  <br>
  <a href="https://github.com/ofamtools/o-mermaid/tags"><img alt="Latest tag" src="https://img.shields.io/github/v/tag/ofamtools/o-mermaid"></a>
  <a href="https://pkg.go.dev/ofam.dev/o-mermaid"><img alt="Go Reference" src="https://pkg.go.dev/badge/ofam.dev/o-mermaid.svg"></a>
  <a href="go.mod"><img alt="Go version" src="https://img.shields.io/github/go-mod/go-version/ofamtools/o-mermaid"></a>
  <a href="LICENSE"><img alt="License" src="https://img.shields.io/github/license/ofamtools/o-mermaid"></a>
</p>

<p align="center">Converts Mermaid diagrams to SVG, PNG and HTML with the embedded Beautiful Mermaid engine, using headless Chrome; reached in mtm as <code>mtm mermaid</code>.</p>

Go rewrite of the [Beautiful Mermaid skill](https://github.com/intellectronica/agent-skills/tree/main/skills/beautiful-mermaid) scripts.

- **oden** owns subcommands, help, JSON discovery, and executable extensions.
- **ochazuke** supplies the interactive file/folder picker.
- **matomeru** reaches the installed `o-mermaid` executable as `mtm mermaid`. No changes to matomeru are required.

`o-mermaid` is an o-family extension: an `o-mermaid` executable on PATH is discoverable by the family's `o` and `mtm` commands, just like `o-cmds` or `o-git`. It was called `mtm-mermaid` (module `ofam.dev/mtm-mermaid`) before the rename; see the [CHANGELOG](CHANGELOG.md).

The CLI, argument parsing, file handling, HTML wrapping, and browser lifecycle are Go. Diagram parsing/layout remains the upstream Beautiful Mermaid JavaScript engine, embedded in the binary and executed in headless Chrome. This is not a native-Go port of that engine.

## Install

Requires Go 1.26+ and Chrome or Chromium for diagram conversion. Theme listing, help, the picker, and wrapping an existing SVG do not require Chrome.

```sh
go install ofam.dev/oden/cmd/o@latest   # oden's o: `o mermaid` runs o-mermaid, no dash needed
go install ofam.dev/o-mermaid@latest
```

From a checkout, `go build -o o-mermaid .` builds against the oden and ochazuke tags in [go.mod](go.mod); an untracked `go.work` (`go work init . && go work use ../oden ../ochazuke`) builds against sibling checkouts instead, and `GOWORK=off` ignores it.

Put Go's bin directory on `PATH`. With matomeru installed, `mtm mermaid` runs the `o-mermaid` on PATH:

```sh
mtm mermaid help
mtm mermaid themes
```

To mount its catalog in matomeru as the `mermaid` extension (listed on mtm's Extensions page), register it once; mtm finds the bin as `o-NAME` on PATH, so the name stays `mermaid`:

```sh
mtm extension register mermaid
mtm extension register mermaid --bin ~/bin/o-mermaid   # or name the bin
```

An older `mtm` that does not discover family tools will not recognize `mtm mermaid`. Run `o-mermaid` directly or rebuild matomeru's `./cmd/mtm`.

The checked-in renderer bundle means ordinary Go builds and installed binaries do not need Node, Bun, npm, or runtime package downloads. Chrome is auto-detected; `--chrome /path/to/chromium` overrides its location.

## Usage

```sh
o mermaid themes
o mermaid themes --json
o mermaid svg diagram.mmd
o mermaid png diagram.mmd --theme tokyo-night
o mermaid html diagram.mmd --padding 60
o mermaid convert diagram.mmd --format png --output ./exports/
o mermaid wrap diagram.svg --output diagram.html --background '#1a1b26'
```

`./exports/` must already exist. SVG, PNG, and HTML commands each produce only the requested format. Run both `svg` and `png` when both files are wanted.

Input can be a positional filename, `--input FILE` / `-i FILE`, inline `--code TEXT` / `-c TEXT`, or `--input -` for stdin. Provide exactly one source. Flags may appear before or after the filename.

```sh
printf 'graph TD\n  A[Input] --> B[Output]\n' | o mermaid svg -i - -o diagram.svg
o mermaid svg diagram.mmd --output -
```

Use a newline after the diagram header. The bundled upstream parser rejects one-line input such as `graph TD; A-->B`. Flowchart, sequence, state, class, and entity-relationship diagrams are supported; this is not the complete Mermaid.js grammar. See [syntax reference](skill/references/mermaid-syntax.md).

Output defaults to the input's directory and stem with the selected extension. `--output` / `-o` accepts an exact file path, a basename without extension, an existing directory, or `-` for stdout. Inline and stdin input require an explicit output. Progress goes to stderr so stdout can carry SVG or PNG bytes.

Existing files require `--force`. Even with `--force`, output cannot be the input file or a hard/symbolic link to it. Parent directories are not created implicitly. Rendering has a one-minute deadline, adjustable with `--timeout 30s`; Ctrl-C cancels it.

PNG uses a 3840×2160 browser viewport and full-page capture. Tall/wide diagrams can produce larger images. HTML and PNG use a minimum diagram width of 1200 pixels and 40 pixels of outer padding; `--padding 0` is supported. `--background` overrides the detected background for those formats. SVG keeps its native vector dimensions. No intermediate files are written.

The HTML wrapper embeds SVG as an image, so SVG scripts do not execute in the wrapper. It also prevents external SVG resources from loading there; fonts use available local fallbacks. Opening a raw SVG directly follows the browser's SVG rules instead.

## File and folder selection

```sh
o mermaid pick --kind file
o mermaid pick --kind dir --path ./
o mermaid svg "$(o mermaid pick --kind file)" \
  --output "$(o mermaid pick --kind dir)"
```

The picker displays on stderr and prints the accepted absolute path on stdout. Ctrl-O opens the shared ochazuke browser. Ctrl-N/Ctrl-P move the highlight, Enter opens a directory or chooses a file, Ctrl-S chooses the highlighted file/directory, and Escape cancels browsing. Back in the path field, Enter accepts and exits; Ctrl-C cancels the command. For directory selection, the browser also offers the current directory as a row.

## Discovery and extensions

```sh
o mermaid commands
o mermaid which png
```

Install an executable named `o-mermaid-NAME` on `PATH` to add `o mermaid NAME`. It appears in help and the JSON catalog. Built-in command names take precedence. The same hierarchy works through `mtm mermaid NAME`.

## Updating the embedded engine

Bun is needed only when updating the engine, theme catalog, or bundled license files:

```sh
bun install --frozen-lockfile
bun run bundle
go build -o o-mermaid .
```

`package.json` pins Beautiful Mermaid; `bun.lock` pins its transitive dependencies. `renderer/build.js` builds the browser bundle and theme catalog together, including the original skill's `default` and `solarized` colors. Commit the regenerated files with dependency updates.

Upstream sources: [Beautiful Mermaid](https://github.com/lukilabs/beautiful-mermaid), [ELK.js](https://github.com/kieler/elkjs), and [entities](https://github.com/fb55/entities). Their license notices are in `renderer/licenses/`; include those notices when distributing the binary.

## Verification

```sh
go test -race ./...
go vet ./...
```

The tests cover output preservation, same-file and symlink protection, invalid conversion arguments, HTML escaping/background/padding, and picker file/directory validation. Rendering also needs an end-to-end run with a real browser; unit tests do not substitute for that check.

The rewrite was smoke-tested locally against all five diagram families, stdin/stdout, all three output formats, themed PNG output, matomeru discovery, and actual file/folder picker interactions. The configured VPS hosts had neither Go nor Chrome available, so browser verification ran on macOS.
