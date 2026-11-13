// Command o-mermaid converts Mermaid diagrams to SVG, PNG and HTML with the
// embedded Beautiful Mermaid engine, using headless Chrome.
//
//	o-mermaid theme list|show NAME             the themes and their colors
//	o-mermaid diagram convert|svg|png|html [INPUT.mmd] [--code SRC] [--output PATH|-] [--theme NAME] [--format svg|png|html]
//	o-mermaid diagram wrap [INPUT.svg] [--output PATH] [--background COLOR] [--padding N]
//	o-mermaid pick [--kind file|dir|any] [--path PATH]   the file picker (a TUI)
//	o-mermaid catalog
//
// The nouns are an ochazuke runtime (app.go): every NOUN VERB prints JSON, and
// a verb-first word reaches one (o-mermaid themes, convert, svg, wrap). Only
// pick stays a command: it opens a terminal page and prints the choice.
package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"

	ocli "ofam.dev/ochazuke/cli"
	"ofam.dev/oden"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	os.Exit(oden.Report(os.Stderr, "o-mermaid", Main(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr)))
}

// Main runs one invocation.
func Main(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	a, err := New(stdin)
	if err != nil {
		return err
	}
	// The shell prints each result as JSON; a conversion that writes to
	// stdout (--output -) answers the bytes themselves, which replace it.
	var buf bytes.Buffer
	sh := a.Shell(&buf, stderr)
	var app *oden.App
	app = oden.New("o-mermaid", args, oden.WithBuiltins(), oden.WithStdio(stdin, stdout, stderr),
		oden.WithDefault(func(_ context.Context, args []string) error {
			if len(args) != 0 {
				return oden.ErrNotFound
			}
			app.Help(stdout)
			return nil
		}))
	app.Add(oden.Command{Name: "pick", Summary: "choose a file or folder with ochazuke; print its path", Run: func(ctx context.Context, args []string) error { return pick(ctx, args, stdout) }})
	// The tool's own command is added first: Mount reads a word that names
	// no command through the nouns' verbs.
	ocli.Mount(app, sh)
	err = app.Run(ctx)
	out := buf.Bytes()
	if a.raw != nil {
		out = a.raw
	}
	if _, werr := stdout.Write(out); err == nil && werr != nil {
		err = fmt.Errorf("write: %w", werr)
	}
	return err
}
