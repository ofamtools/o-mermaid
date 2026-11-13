package main

import (
	"bytes"
	tea "charm.land/bubbletea/v2"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ofam.dev/ochazuke"
)

// run is one o-mermaid invocation with empty standard input; its output is
// discarded.
func run(args []string) error {
	return Main(context.Background(), args, strings.NewReader(""), io.Discard, io.Discard)
}

func TestWrapAndOverwriteProtection(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "a&b.svg")
	output := filepath.Join(dir, "a&b.html")
	svg := `<svg xmlns="http://www.w3.org/2000/svg" style="--bg:#282a36"><text>Hello</text></svg>`
	if err := os.WriteFile(input, []byte(svg), 0o600); err != nil {
		t.Fatal(err)
	}
	args := []string{"wrap", input, "--output", dir, "--padding", "0"}
	if err := run(args); err != nil {
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
	if err := run(args); err == nil {
		t.Fatal("overwrote output without --force")
	}
	if err := run(append(args, "--force")); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"wrap", input, "--output", input, "--force"}); err == nil {
		t.Fatal("allowed overwriting input")
	}
	alias := filepath.Join(dir, "alias.html")
	if err := os.Symlink(input, alias); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"wrap", input, "--output", alias, "--force"}); err == nil {
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

// themes is theme list through a bare word; theme show answers one theme's
// colors; svg is convert with the format set (it reaches convert's checks).
func TestThemeWordsAndFormatAliases(t *testing.T) {
	var out bytes.Buffer
	if err := Main(context.Background(), []string{"themes", "--json", "{}"}, strings.NewReader(""), &out, io.Discard); err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		ID    string   `json:"id"`
		Cells []string `json:"cells"`
	}
	if err := json.Unmarshal(out.Bytes(), &rows); err != nil || len(rows) < 2 || len(rows[0].Cells) != 3 {
		t.Fatalf("themes %q %v", out.String(), err)
	}
	out.Reset()
	if err := Main(context.Background(), []string{"theme", "show", "tokyo-night"}, strings.NewReader(""), &out, io.Discard); err != nil || !strings.Contains(out.String(), "#1a1b26") {
		t.Fatalf("theme show %q %v", out.String(), err)
	}
	err := run([]string{"svg", "--code", "graph TD", "--theme", "missing"})
	if err == nil || !strings.Contains(err.Error(), "unknown theme") {
		t.Fatalf("svg reached convert: %v", err)
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
		if err := run(append(args, "--output", output)); err == nil {
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

// ctrl+o in the path field runs picker.open: the path picker opens on its
// find source, and esc closes it again (no NewPicker shim is involved).
func TestPickerCtrlOOpensPathPicker(t *testing.T) {
	model, _, err := pickerModel(t.TempDir(), "file")
	if err != nil {
		t.Fatal(err)
	}
	model.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	model.Update(tea.KeyPressMsg{Code: 'o', Mod: tea.ModCtrl})
	if model.Runtime.Chooser(ochazuke.PathChooserKind) == nil {
		t.Fatal("ctrl+o did not open the path picker")
	}
	model.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	model.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if model.Runtime.Chooser(ochazuke.PathChooserKind) != nil {
		t.Fatal("esc, q did not close the path picker")
	}
}

func TestPickerDeclaresSingularNamespace(t *testing.T) {
	m, _, err := pickerModel(t.TempDir(), "file")
	if err != nil {
		t.Fatal(err)
	}
	if ochazuke.DefaultNamespace != "mermaid" {
		t.Fatalf("default namespace %q; want mermaid", ochazuke.DefaultNamespace)
	}
	op, ok := m.Runtime.Operations.Find("selection.accept")
	if !ok || op.Namespace() != "mermaid" || op.Name() != "accept" {
		t.Fatalf("selection.accept: %v %q %q", ok, op.Namespace(), op.Name())
	}
}
