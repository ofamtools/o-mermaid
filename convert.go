package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"ofam.dev/o-mermaid/renderer"
)

// Formats are what a diagram converts to.
var Formats = []string{"svg", "png", "html"}

// ConvertOptions are one conversion: the Mermaid source (a file, "-" for
// standard input, or inline code), where the result goes, and how it looks.
type ConvertOptions struct {
	Input, Code, Output, Theme, Format, Chrome, Background string
	Padding                                                int
	Force                                                  bool
	Timeout                                                time.Duration
}

// DefaultConvert is a conversion's defaults: the default theme, SVG, 40
// pixels of padding and a minute to render.
func DefaultConvert() ConvertOptions {
	return ConvertOptions{Theme: "default", Format: "svg", Padding: 40, Timeout: time.Minute}
}

// Converted is what a conversion made: where it went ("-" for standard
// output), its format and size.
type Converted struct {
	Path   string `json:"path"`
	Format string `json:"format"`
	Bytes  int    `json:"bytes"`
}

// Convert renders the source and writes the result: to the output path (a
// file, or a folder holding INPUT's name), beside the input by default, or
// to stdout for "-", whose bytes it returns. Nothing is written when an
// argument is invalid or the rendering fails.
func (a *App) Convert(ctx context.Context, o ConvertOptions) (Converted, []byte, error) {
	if (o.Input == "") == (o.Code == "") {
		return Converted{}, nil, errors.New("provide exactly one of --input (or positional file) and --code")
	}
	if !slices.Contains(Formats, o.Format) {
		return Converted{}, nil, fmt.Errorf("unsupported format %q: use svg, png or html", o.Format)
	}
	if o.Padding < 0 || o.Timeout <= 0 {
		return Converted{}, nil, errors.New("padding must be nonnegative and timeout must be positive")
	}
	if _, ok := renderer.Themes()[o.Theme]; !ok {
		return Converted{}, nil, fmt.Errorf("unknown theme %q; run themes", o.Theme)
	}
	code := o.Code
	if o.Input != "" {
		var data []byte
		var err error
		if o.Input == "-" {
			data, err = io.ReadAll(a.Stdin)
		} else {
			data, err = os.ReadFile(o.Input)
		}
		if err != nil {
			return Converted{}, nil, err
		}
		code = string(data)
	}
	output, err := outputPath(o.Input, o.Output, o.Format)
	if err != nil {
		return Converted{}, nil, err
	}
	if err := checkOutput(o.Input, output, o.Force); err != nil {
		return Converted{}, nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, o.Timeout)
	defer cancel()
	data, err := renderer.Render(ctx, code, renderer.Options{Theme: o.Theme, Format: o.Format, Padding: o.Padding, Background: o.Background, Chrome: o.Chrome})
	if err != nil {
		return Converted{}, nil, err
	}
	res := Converted{Path: output, Format: o.Format, Bytes: len(data)}
	if output == "-" {
		return res, data, nil
	}
	return res, nil, save(output, data, o.Force)
}

// WrapOptions are one wrapping of an SVG file in screenshot-ready HTML.
type WrapOptions struct {
	Input, Output, Background string
	Padding                   int
	Force                     bool
}

// Wrap writes the SVG file as an HTML page (beside the input by default).
func (a *App) Wrap(o WrapOptions) (Converted, error) {
	if o.Input == "" {
		return Converted{}, errors.New("--svg or positional SVG file is required")
	}
	data, err := os.ReadFile(o.Input)
	if err != nil {
		return Converted{}, err
	}
	output, err := outputPath(o.Input, o.Output, "html")
	if err != nil {
		return Converted{}, err
	}
	if err := checkOutput(o.Input, output, o.Force); err != nil {
		return Converted{}, err
	}
	document, err := renderer.HTML(string(data), strings.TrimSuffix(filepath.Base(o.Input), filepath.Ext(o.Input)), o.Padding, o.Background)
	if err != nil {
		return Converted{}, err
	}
	return Converted{Path: output, Format: "html", Bytes: len(document)}, save(output, document, o.Force)
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

// save writes data to path, refusing an existing file unless force.
func save(path string, data []byte, force bool) error {
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
	return closeErr
}
