---
name: beautiful-mermaid
description: Render Mermaid files as SVG, PNG, or HTML with o-mermaid; list themes or wrap existing SVG files.
---

# Beautiful Mermaid rendering

Use the Go `o-mermaid` CLI, also available as `mtm mermaid` through matomeru. See the repository [README](../README.md) for installation and dependencies. Diagram conversion requires Chrome/Chromium; theme listing and SVG wrapping do not.

## Render

1. Write Mermaid source with its header on its own line. Consult [syntax reference](references/mermaid-syntax.md) when authoring diagrams. The upstream parser rejects one-line `graph TD; A-->B` input.
2. Select a theme with `o-mermaid themes`; the default is `default`.
3. Render the requested formats:

```sh
o-mermaid svg diagram.mmd --theme tokyo-night
o-mermaid png diagram.mmd --theme tokyo-night
```

These commands create `diagram.svg` and `diagram.png` alongside the source. Run only the requested format when one output is wanted. PNG captures a full page at a 3840×2160 viewport, with a minimum 1200px diagram width. No intermediary files need cleanup.

Use `--output PATH` to choose another file or an existing output folder. Existing output requires `--force`; input files are protected from overwrite. For piping, use `--input -` and/or `--output -`. Diagnostics go to stderr.

## HTML

```sh
o-mermaid html diagram.mmd --padding 40
o-mermaid wrap diagram.svg --output diagram.html --background '#1a1b26'
```

The wrapper embeds the SVG as an image. Its scripts and external resources do not execute/load in that wrapper.

## Troubleshooting

- Unknown theme: inspect `o-mermaid themes --json`.
- Missing browser: install Chrome/Chromium or pass `--chrome PATH`.
- Invalid diagram: use a separate header line, quote special characters in labels, and prefer `A -->|label| B` for edge labels.
- Long-running diagram: adjust `--timeout`, whose default is one minute.
- Output exists: choose another path or explicitly pass `--force`.
