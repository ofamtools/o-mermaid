# mtm-mermaid

Go rewrite of the [Beautiful Mermaid skill](https://github.com/intellectronica/agent-skills/tree/main/skills/beautiful-mermaid) scripts. Converts Mermaid files to SVG, PNG, and HTML.

- **oden** owns subcommands, help, JSON discovery, and executable extensions.
- **ochazuke** supplies the interactive file/folder picker.
- **matomeru** discovers the installed `mtm-mermaid` executable as `mtm mermaid`. No changes to matomeru are required.

The CLI, argument parsing, file handling, HTML wrapping, and browser lifecycle are Go. Diagram parsing/layout remains the upstream Beautiful Mermaid JavaScript engine, embedded in the binary and executed in headless Chrome. This is not a native-Go port of that engine.

## Build and install

Requires Go 1.26+ and Chrome or Chromium for diagram conversion. oden and ochazuke are required at their tags (`v0.1.0`); an untracked `go.work` (`go work init . && go work use ../oden ../ochazuke`) builds against the sibling checkouts, and `GOWORK=off` builds from the tags as a fresh clone does. Theme listing, help, the picker, and wrapping an existing SVG do not require Chrome.

```sh
go build -o mtm-mermaid .
go install .
```

Put Go's bin directory on `PATH`. With the oden-enabled matomeru installed:

```sh
mtm mermaid help
mtm mermaid themes
```

An older `mtm` without oden discovery will not recognize this command. Run `mtm-mermaid` directly or rebuild matomeru's `./cmd/mtm`.

The checked-in renderer bundle means ordinary Go builds and installed binaries do not need Node, Bun, npm, or runtime package downloads. Chrome is auto-detected; `--chrome /path/to/chromium` overrides its location.

## Commands

```sh
mtm-mermaid themes
mtm-mermaid themes --json
mtm-mermaid svg diagram.mmd
mtm-mermaid png diagram.mmd --theme tokyo-night
mtm-mermaid html diagram.mmd --padding 60
mtm-mermaid convert diagram.mmd --format png --output ./exports/
mtm-mermaid wrap diagram.svg --output diagram.html --background '#1a1b26'
```

`./exports/` must already exist. SVG, PNG, and HTML commands each produce only the requested format. Run both `svg` and `png` when both files are wanted.

Input can be a positional filename, `--input FILE` / `-i FILE`, inline `--code TEXT` / `-c TEXT`, or `--input -` for stdin. Provide exactly one source. Flags may appear before or after the filename.

```sh
printf 'graph TD\n  A[Input] --> B[Output]\n' | mtm-mermaid svg -i - -o diagram.svg
mtm-mermaid svg diagram.mmd --output -
```

Use a newline after the diagram header. The bundled upstream parser rejects one-line input such as `graph TD; A-->B`. Flowchart, sequence, state, class, and entity-relationship diagrams are supported; this is not the complete Mermaid.js grammar. See [syntax reference](skill/references/mermaid-syntax.md).

Output defaults to the input's directory and stem with the selected extension. `--output` / `-o` accepts an exact file path, a basename without extension, an existing directory, or `-` for stdout. Inline and stdin input require an explicit output. Progress goes to stderr so stdout can carry SVG or PNG bytes.

Existing files require `--force`. Even with `--force`, output cannot be the input file or a hard/symbolic link to it. Parent directories are not created implicitly. Rendering has a one-minute deadline, adjustable with `--timeout 30s`; Ctrl-C cancels it.

PNG uses a 3840×2160 browser viewport and full-page capture. Tall/wide diagrams can produce larger images. HTML and PNG use a minimum diagram width of 1200 pixels and 40 pixels of outer padding; `--padding 0` is supported. `--background` overrides the detected background for those formats. SVG keeps its native vector dimensions. No intermediate files are written.

The HTML wrapper embeds SVG as an image, so SVG scripts do not execute in the wrapper. It also prevents external SVG resources from loading there; fonts use available local fallbacks. Opening a raw SVG directly follows the browser's SVG rules instead.

## File and folder selection

```sh
mtm-mermaid pick --kind file
mtm-mermaid pick --kind dir --path ./
mtm-mermaid svg "$(mtm-mermaid pick --kind file)" \
  --output "$(mtm-mermaid pick --kind dir)"
```

The picker displays on stderr and prints the accepted absolute path on stdout. Ctrl-O opens the shared ochazuke browser. Ctrl-N/Ctrl-P move the highlight, Enter opens a directory or chooses a file, Ctrl-S chooses the highlighted file/directory, and Escape cancels browsing. Back in the path field, Enter accepts and exits; Ctrl-C cancels the command. For directory selection, the browser also offers the current directory as a row.

## Discovery and extensions

```sh
mtm-mermaid commands
mtm-mermaid which png
```

Install an executable named `mtm-mermaid-NAME` on `PATH` to add `mtm-mermaid NAME`. It appears in help and the JSON catalog. Built-in command names take precedence. The same hierarchy works through `mtm mermaid NAME`.

## Updating the embedded engine

Bun is needed only when updating the engine, theme catalog, or bundled license files:

```sh
bun install --frozen-lockfile
bun run bundle
go build -o mtm-mermaid .
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
