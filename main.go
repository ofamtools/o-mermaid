package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/unixsuperhero/mtm-mermaid/renderer"
	"github.com/unixsuperhero/oden"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	os.Exit(oden.Report(os.Stderr, "mtm-mermaid", commands(os.Args[1:]).Run(ctx)))
}

func commands(args []string) *oden.App {
	app := oden.New("mtm-mermaid", args, oden.WithBuiltins())
	app.Default = func(_ context.Context, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf("unknown command %q; run help", args[0])
		}
		app.Help(os.Stdout)
		return nil
	}
	app.Add(oden.Command{Name: "themes", Summary: "list theme names and colors (--json)", Run: listThemes})
	app.Add(oden.Command{Name: "convert", Summary: "convert Mermaid to SVG, PNG or HTML (--format)", Run: func(ctx context.Context, args []string) error { return convert(ctx, args, "") }})
	for _, format := range []string{"svg", "png", "html"} {
		app.Add(oden.Command{Name: format, Summary: "render Mermaid to " + strings.ToUpper(format), Run: func(ctx context.Context, args []string) error { return convert(ctx, args, format) }})
	}
	app.Add(oden.Command{Name: "wrap", Summary: "wrap an existing SVG in screenshot-ready HTML", Run: wrap})
	app.Add(oden.Command{Name: "pick", Summary: "choose a file or folder with ochazuke; print its path", Run: pick})
	return app
}

func parseFlags(f *flag.FlagSet, args []string) ([]string, error) {
	var words []string
	for len(args) > 0 {
		if args[0] == "--" {
			return append(words, args[1:]...), nil
		}
		if err := f.Parse(args); err != nil {
			return nil, err
		}
		args = f.Args()
		if len(args) > 0 {
			words = append(words, args[0])
			args = args[1:]
		}
	}
	return words, nil
}

func listThemes(_ context.Context, args []string) error {
	f := flag.NewFlagSet("themes", flag.ContinueOnError)
	asJSON := f.Bool("json", false, "print theme colors as JSON")
	words, err := parseFlags(f, args)
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(words) != 0 {
		return errors.New("usage: themes [--json]")
	}
	themes := renderer.Themes()
	if *asJSON {
		return json.NewEncoder(os.Stdout).Encode(themes)
	}
	names := make([]string, 0, len(themes))
	for name := range themes {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(os.Stdout, "%-22s %s  %s\n", name, themes[name]["bg"], themes[name]["fg"])
	}
	return nil
}

func convert(ctx context.Context, args []string, format string) error {
	f := flag.NewFlagSet("convert [INPUT.mmd] [options]", flag.ContinueOnError)
	var input, code, output, theme, chrome, background string
	var padding int
	var force bool
	f.StringVar(&input, "input", "", "Mermaid file; - reads stdin")
	f.StringVar(&input, "i", "", "alias for --input")
	f.StringVar(&code, "code", "", "inline Mermaid source")
	f.StringVar(&code, "c", "", "alias for --code")
	f.StringVar(&output, "output", "", "output path or existing folder; - writes stdout")
	f.StringVar(&output, "o", "", "alias for --output")
	f.StringVar(&theme, "theme", "default", "theme name; see themes")
	f.StringVar(&theme, "t", "default", "alias for --theme")
	f.StringVar(&chrome, "chrome", "", "Chrome/Chromium executable (auto-detected by default)")
	f.StringVar(&background, "background", "", "HTML/PNG background color (auto-detected)")
	f.IntVar(&padding, "padding", 40, "HTML/PNG padding in pixels")
	f.BoolVar(&force, "force", false, "replace an existing output file")
	timeout := f.Duration("timeout", time.Minute, "rendering timeout")
	if format == "" {
		f.StringVar(&format, "format", "svg", "svg, png or html")
	}
	words, err := parseFlags(f, args)
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(words) > 1 || len(words) == 1 && input != "" {
		return errors.New("provide one input file")
	}
	if len(words) == 1 {
		input = words[0]
	}
	if (input == "") == (code == "") {
		return errors.New("provide exactly one of --input (or positional file) and --code")
	}
	if format != "svg" && format != "png" && format != "html" {
		return fmt.Errorf("unsupported format %q: use svg, png or html", format)
	}
	if padding < 0 || *timeout <= 0 {
		return errors.New("padding must be nonnegative and timeout must be positive")
	}
	if _, ok := renderer.Themes()[theme]; !ok {
		return fmt.Errorf("unknown theme %q; run themes", theme)
	}
	if input != "" {
		var data []byte
		if input == "-" {
			data, err = io.ReadAll(os.Stdin)
		} else {
			data, err = os.ReadFile(input)
		}
		if err != nil {
			return err
		}
		code = string(data)
	}
	output, err = outputPath(input, output, format)
	if err != nil {
		return err
	}
	if err := checkOutput(input, output, force); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()
	data, err := renderer.Render(ctx, code, renderer.Options{Theme: theme, Format: format, Padding: padding, Background: background, Chrome: chrome})
	if err != nil {
		return err
	}
	return save(output, data, force)
}

func outputPath(input, output, format string) (string, error) {
	stem := "diagram"
	if input != "" && input != "-" {
		stem = strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
	}
	if output == "-" {
		return output, nil
	}
	if output == "" {
		if input != "" && input != "-" {
			return filepath.Join(filepath.Dir(input), stem+"."+format), nil
		}
		return "", errors.New("--output is required for inline code or stdin")
	}
	if info, err := os.Stat(output); err == nil && info.IsDir() {
		return filepath.Join(output, stem+"."+format), nil
	}
	if filepath.Ext(output) == "" {
		output += "." + format
	}
	return output, nil
}

func checkOutput(input, output string, force bool) error {
	if output == "-" {
		return nil
	}
	if input != "" && input != "-" {
		a, err := filepath.Abs(input)
		if err != nil {
			return err
		}
		b, err := filepath.Abs(output)
		if err != nil {
			return err
		}
		if a == b {
			return errors.New("output must not replace the input")
		}
		in, inErr := os.Stat(input)
		out, outErr := os.Stat(output)
		if inErr == nil && outErr == nil && os.SameFile(in, out) {
			return errors.New("output must not replace the input")
		}
	}
	if _, err := os.Lstat(output); err == nil && !force {
		return fmt.Errorf("output exists: %s (use --force)", output)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func save(path string, data []byte, force bool) error {
	if path == "-" {
		_, err := os.Stdout.Write(data)
		return err
	}
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if force {
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	file, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	fmt.Fprintln(os.Stderr, path)
	return nil
}

func wrap(_ context.Context, args []string) error {
	f := flag.NewFlagSet("wrap", flag.ContinueOnError)
	var input, output, background string
	var padding int
	f.StringVar(&input, "svg", "", "input SVG file")
	f.StringVar(&input, "s", "", "alias for --svg")
	f.StringVar(&output, "output", "", "output HTML path or folder")
	f.StringVar(&output, "o", "", "alias for --output")
	f.StringVar(&background, "background", "", "background color")
	f.StringVar(&background, "b", "", "alias for --background")
	f.IntVar(&padding, "padding", 40, "padding in pixels")
	f.IntVar(&padding, "p", 40, "alias for --padding")
	force := f.Bool("force", false, "replace existing output")
	words, err := parseFlags(f, args)
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(words) == 1 && input == "" {
		input = words[0]
	} else if len(words) > 0 {
		return errors.New("provide one SVG file")
	}
	if input == "" {
		return errors.New("--svg or positional SVG file is required")
	}
	data, err := os.ReadFile(input)
	if err != nil {
		return err
	}
	output, err = outputPath(input, output, "html")
	if err != nil {
		return err
	}
	if err := checkOutput(input, output, *force); err != nil {
		return err
	}
	document, err := renderer.HTML(string(data), strings.TrimSuffix(filepath.Base(input), filepath.Ext(input)), padding, background)
	if err != nil {
		return err
	}
	return save(output, document, *force)
}
