package main

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unixsuperhero/ochazuke"
)

func TestWrapAndOverwriteProtection(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "a&b.svg")
	output := filepath.Join(dir, "a&b.html")
	svg := `<svg xmlns="http://www.w3.org/2000/svg" style="--bg:#282a36"><text>Hello</text></svg>`
	if err := os.WriteFile(input, []byte(svg), 0o600); err != nil {
		t.Fatal(err)
	}
	args := []string{"wrap", input, "--output", dir, "--padding", "0"}
	if err := commands(args).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	document, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<title>a&amp;b</title>", "padding:0px", "background:#282a36", base64.StdEncoding.EncodeToString([]byte(svg))} {
		if !strings.Contains(string(document), want) {
			t.Errorf("HTML missing %q", want)
		}
	}
	if err := commands(args).Run(context.Background()); err == nil {
		t.Fatal("overwrote output without --force")
	}
	if err := commands(append(args, "--force")).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := commands([]string{"wrap", input, "--output", input, "--force"}).Run(context.Background()); err == nil {
		t.Fatal("allowed overwriting input")
	}
	alias := filepath.Join(dir, "alias.html")
	if err := os.Symlink(input, alias); err != nil {
		t.Fatal(err)
	}
	if err := commands([]string{"wrap", input, "--output", alias, "--force"}).Run(context.Background()); err == nil {
		t.Fatal("allowed overwriting input through symlink")
	}
	after, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != svg {
		t.Fatalf("input changed: %s", after)
	}
}

func TestInvalidConversionDoesNotWrite(t *testing.T) {
	output := filepath.Join(t.TempDir(), "diagram.svg")
	for _, args := range [][]string{
		{"convert", "--code", "graph TD", "--format", "pdf"},
		{"svg", "--code", "graph TD", "--theme", "missing"},
		{"svg", "--code", "graph TD", "--input", "missing.mmd"},
		{"svg", "--code", "graph TD", "--timeout", "0s"},
		{"png", "--code", "graph TD", "--padding", "-1"},
	} {
		if err := commands(append(args, "--output", output)).Run(context.Background()); err == nil {
			t.Errorf("accepted invalid arguments: %v", args)
		}
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("invalid conversion created output: %v", err)
	}
	if err := save(output, []byte("original"), false); err != nil {
		t.Fatal(err)
	}
	if err := save(output, []byte("replacement"), false); err == nil {
		t.Fatal("save replaced existing file")
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original" {
		t.Fatalf("file changed: %s", data)
	}
}

func TestPickerRejectsWrongKindAndAcceptsDirectory(t *testing.T) {
	dir := t.TempDir()
	fileModel, fileSelection, err := pickerModel(dir, "file")
	if err != nil {
		t.Fatal(err)
	}
	request := ochazuke.Request{Noun: "selection", Verb: "accept"}
	if _, err := fileModel.Runtime.Execute(request); err == nil {
		t.Fatal("file picker accepted a directory")
	}
	if *fileSelection != "" || fileModel.Runtime.Quit {
		t.Fatal("invalid selection ended the picker")
	}
	dirModel, dirSelection, err := pickerModel(dir, "dir")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dirModel.Runtime.Execute(request); err != nil {
		t.Fatal(err)
	}
	if *dirSelection != dir || !dirModel.Runtime.Quit {
		t.Fatalf("selection = %q, quit = %v", *dirSelection, dirModel.Runtime.Quit)
	}
}
