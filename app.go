package main

import (
	"context"
	"fmt"
	"io"
	"sort"
	"time"

	"ofam.dev/o-mermaid/renderer"
	"ofam.dev/ochazuke"
	ocli "ofam.dev/ochazuke/cli"
)

// The application: one ochazuke runtime with two nouns and no page.
//
//   - theme: the renderer's themes, a facts noun (list, show, get over the
//     embedded themes.json; read-only, so it has no archive or edit);
//   - diagram: a command noun with no store of its own: convert (and the
//     svg, png and html words, aliases that fix its format) renders Mermaid and
//     writes a file, wrap puts an existing SVG in a page. They create files,
//     not things the noun lists, so the scaffold's list, show, edit and
//     archive do not apply to it.
//
// pick (the file picker) is a command, not a verb: it opens a terminal page
// and prints the choice, so it has nothing to answer a caller that is not at
// a terminal.

// App is o-mermaid's state: the runtime and the streams a conversion reads
// (standard input for "-") and answers on.
type App struct {
	r     *ochazuke.Runtime
	Stdin io.Reader
	raw   []byte // bytes a conversion wrote to stdout (--output -), printed in place of the JSON
}

// New builds the runtime.
func New(stdin io.Reader) (*App, error) {
	a := &App{r: ochazuke.NewRuntime(), Stdin: stdin}
	a.r.Define("theme").Main().Props(themeProps()...).Store(themeStore()).Columns(themeColumns...).
		Verb("list", themeList, "the themes by name: rows {noun, id, label, data: every color, cells: name, bg, fg}").Rows("theme").Fetch()
	a.registerVerbs()
	return a, nil
}

// Shell is the NOUN VERB command line over the runtime.
func (a *App) Shell(stdout, stderr io.Writer) *ocli.Shell {
	sh := ocli.New(a.r, "o-mermaid", a.Stdin, stdout, stderr)
	sh.Usage = ocli.DefaultUsage("o-mermaid")
	return sh
}

var themeColumns = []ochazuke.Column{{Title: "name", Min: 12}, {Title: "bg", Priority: 1}, {Title: "fg", Priority: 2}}

func themeProps() []ochazuke.PropSpec {
	text := func(name, desc string) ochazuke.PropSpec {
		return ochazuke.PropSpec{Name: name, ReadOnly: true, Parameter: ochazuke.Parameter{Type: "string", Description: desc}}
	}
	return []ochazuke.PropSpec{
		{Name: "name", Key: true, Parameter: ochazuke.Parameter{Type: "string", Description: "The theme's name, as --theme takes it"}},
		text("bg", "The background color"),
		text("fg", "The foreground color"),
		{Name: "colors", ReadOnly: true, Parameter: ochazuke.Parameter{Type: ochazuke.PropObject, Description: "Every color the theme sets: bg, fg and, for the richer themes, line, accent and muted"}},
	}
}

func themeThing(name string, colors map[string]string) ochazuke.Thing {
	all := map[string]any{}
	for k, v := range colors {
		all[k] = v
	}
	return ochazuke.Thing{"name": name, "bg": colors["bg"], "fg": colors["fg"], "colors": all}
}

func themeStore() ochazuke.Store {
	return ochazuke.Store{Kind: ochazuke.KindCollection, Facts: true,
		Get: func(_ ochazuke.Ctx, name string) (ochazuke.Thing, error) {
			colors, ok := renderer.Themes()[name]
			if !ok {
				return nil, fmt.Errorf("unknown theme %q; run theme list", name)
			}
			return themeThing(name, colors), nil
		},
		List: func(ochazuke.Ctx, ochazuke.ListQuery) ([]ochazuke.Thing, error) {
			themes := renderer.Themes()
			names := make([]string, 0, len(themes))
			for name := range themes {
				names = append(names, name)
			}
			sort.Strings(names)
			out := make([]ochazuke.Thing, len(names))
			for i, name := range names {
				out[i] = themeThing(name, themes[name])
			}
			return out, nil
		},
	}
}

func themeList(ochazuke.Ctx) ochazuke.Result {
	themes := renderer.Themes()
	names := make([]string, 0, len(themes))
	for name := range themes {
		names = append(names, name)
	}
	sort.Strings(names)
	rows := make([]ocli.Row, len(names))
	for i, name := range names {
		c := themes[name]
		rows[i] = ocli.Row{Noun: "theme", ID: name, Label: name, Data: c, Cells: []string{name, c["bg"], c["fg"]}}
	}
	return ochazuke.Done(rows)
}

func (a *App) registerVerbs() {
	r := a.r
	r.Define("diagram").Kind(ochazuke.NounKindCommand).
		Verb("convert", a.convert, "render Mermaid (a file, - for standard input, or --code) to svg, png or html (--format); writes a file beside the input, or --output PATH, a folder, or - for standard output: {path, format, bytes}").Positional("input").
		Verb("wrap", a.wrap, "wrap an existing SVG file in screenshot-ready HTML: {path, format, bytes}").Positional("input")
	text := func(d string) ochazuke.Parameter { return ochazuke.Parameter{Type: "string", Description: d} }
	path := func(d string) ochazuke.Parameter {
		return ochazuke.Parameter{Type: ochazuke.PropPath, Description: d, Free: true}
	}
	common := func() map[string]ochazuke.Parameter {
		return map[string]ochazuke.Parameter{
			"input":      path("the Mermaid file; - reads standard input"),
			"code":       text("inline Mermaid source, instead of a file"),
			"output":     path("the output path or an existing folder; - writes the bytes to standard output (default: beside the input)"),
			"theme":      {Type: "string", Noun: "theme", Description: "a theme name (theme list; default default)"},
			"chrome":     path("the Chrome or Chromium executable (default: found on this machine)"),
			"background": text("the HTML or PNG background color (default: the theme's)"),
			"padding":    {Type: ochazuke.SettingInteger, Description: "HTML and PNG padding in pixels (default 40)"},
			"force":      {Type: "boolean", Description: "replace an existing output file"},
			"timeout":    text("give up rendering after this long (default 1m)"),
		}
	}
	set := func(key string, params map[string]ochazuke.Parameter) {
		op := r.Operations.At(ochazuke.Key(r.CanonicalKey(key)))
		op.Parameters, op.Result = params, "{path, format, bytes}"
		r.Operations.Set(op)
	}
	withFormat := common()
	withFormat["format"] = ochazuke.Parameter{Type: "string", Enum: Formats, Description: "svg (default), png or html"}
	set("diagram.convert", withFormat)
	set("diagram.wrap", map[string]ochazuke.Parameter{
		"input":      path("the SVG file"),
		"output":     path("the output HTML path or an existing folder (default: beside the input)"),
		"background": text("the page's background color (default: the SVG's)"),
		"padding":    {Type: ochazuke.SettingInteger, Description: "padding in pixels (default 40)"},
		"force":      {Type: "boolean", Description: "replace an existing output file"},
	})
	// The command-line words from before they were verbs: svg, png and html
	// are convert with the format set; themes is theme list.
	words := []ochazuke.ActionAlias{{Alias: "themes", Action: "theme.list", Bare: true, Shadows: true}}
	for _, format := range Formats {
		words = append(words, ochazuke.ActionAlias{Alias: format, Action: "diagram.convert", Args: ochazuke.Arguments{"format": format}, Bare: true, Shadows: true,
			Note: "convert with --format " + format})
	}
	for _, w := range words {
		if err := r.AddActionAlias(w); err != nil {
			panic(err)
		}
	}
	// Both verbs write a file; the history records them as changes.
	for _, key := range []string{"diagram.convert", "diagram.wrap"} {
		op, ok := r.Operations.Find(ochazuke.Key(r.CanonicalKey(key)))
		if !ok {
			panic("no operation " + key)
		}
		op.HistoryKind = ochazuke.HistoryChange
		r.Operations.Set(op)
	}
}

// convert is the handler of convert: --format, default svg.
func (a *App) convert(c ochazuke.Ctx) ochazuke.Result {
	o := DefaultConvert()
	o.Input, o.Code, o.Output = ochazuke.StringArg(c.Args, "input", ""), ochazuke.StringArg(c.Args, "code", ""), ochazuke.StringArg(c.Args, "output", "")
	o.Theme, o.Chrome, o.Background = ochazuke.StringArg(c.Args, "theme", o.Theme), ochazuke.StringArg(c.Args, "chrome", ""), ochazuke.StringArg(c.Args, "background", "")
	o.Format = ochazuke.StringArg(c.Args, "format", o.Format)
	o.Padding = ochazuke.IntArg(c.Args, "padding", o.Padding)
	o.Force, _ = c.Args["force"].(bool)
	if v := ochazuke.StringArg(c.Args, "timeout", ""); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return ochazuke.Failed(err)
		}
		o.Timeout = d
	}
	res, data, err := a.Convert(context.Background(), o)
	if err != nil {
		return ochazuke.Failed(err)
	}
	if data != nil {
		a.raw = data
	}
	return ochazuke.Done(res)
}

func (a *App) wrap(c ochazuke.Ctx) ochazuke.Result {
	o := WrapOptions{Input: ochazuke.StringArg(c.Args, "input", ""), Output: ochazuke.StringArg(c.Args, "output", ""), Background: ochazuke.StringArg(c.Args, "background", ""), Padding: ochazuke.IntArg(c.Args, "padding", 40)}
	o.Force, _ = c.Args["force"].(bool)
	res, err := a.Wrap(o)
	if err != nil {
		return ochazuke.Failed(err)
	}
	return ochazuke.Done(res)
}
