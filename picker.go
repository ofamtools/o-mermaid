package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"ofam.dev/ochazuke"
)

func pick(ctx context.Context, args []string) error {
	f := flag.NewFlagSet("pick", flag.ContinueOnError)
	kind := f.String("kind", "file", "file, dir or any")
	path := f.String("path", "", "initial path")
	words, err := parseFlags(f, args)
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(words) != 0 {
		return errors.New("usage: pick [--kind file|dir|any] [--path PATH]")
	}
	if *kind != "file" && *kind != "dir" && *kind != "any" {
		return errors.New("kind must be file, dir or any")
	}
	model, selected, err := pickerModel(*path, *kind)
	if err != nil {
		return err
	}
	if _, err := tea.NewProgram(model, tea.WithContext(ctx), tea.WithOutput(os.Stderr)).Run(); err != nil {
		return err
	}
	if *selected == "" {
		return errors.New("selection cancelled")
	}
	fmt.Fprintln(os.Stdout, *selected)
	return nil
}

func pickerModel(path, kind string) (*ochazuke.Model, *string, error) {
	r := ochazuke.NewRuntime()
	r.Instance("selection")
	input := ochazuke.NewInput("path", path, false)
	input.Kind, input.PathFilter = "path", kind
	if err := r.AddInput(input, ochazuke.Parent("selection"), ochazuke.Kind("prompt")); err != nil {
		return nil, nil, err
	}
	selected := new(string)
	r.Register(ochazuke.Operation{Verb: "accept", Noun: "selection", Description: "Accept the selected path and exit", Run: func(_ *ochazuke.Runtime, _ ochazuke.Context, _ ochazuke.Arguments) (any, error) {
		path, err := filepath.Abs(string(input.Text))
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if kind == "file" && !info.Mode().IsRegular() || kind == "dir" && !info.IsDir() {
			return nil, fmt.Errorf("choose a %s", kind)
		}
		*selected = path
		r.Quit = true
		return path, nil
	}})
	r.Define("app").Mode("").
		Bind("ctrl+o", ochazuke.Do("picker.open").Guard("no-modal").Captured()).
		Bind("enter", ochazuke.Do("selection.accept").Guard("no-modal").Captured())
	_, err := r.AddPage("selection", "Select a "+kind, func(_ *ochazuke.Runtime, width, height int) ochazuke.Block {
		return ochazuke.InputBlock(input, width, height)
	}, ochazuke.Focus("path"))
	if err != nil {
		return nil, nil, err
	}
	if err := r.OpenPage("selection"); err != nil {
		return nil, nil, err
	}
	model := ochazuke.NewModel(r)
	model.Title = "Mermaid: Ctrl-O browse · Enter accept · Ctrl-C cancel"
	return model, selected, nil
}
