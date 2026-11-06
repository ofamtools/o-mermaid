package main

import (
	"testing"

	"ofam.dev/ochazuke/namecheck"
)

// TestNoNewNameBranches holds the non-test Go to namecheck: no new
// comparison, case label or lookup table keyed by a noun, verb, mode,
// motion or parameter name the runtime declares, and no function grows
// or passes complexity 25. The accepted uses are testdata/namecheck.txt,
// which only shrinks.
func TestNoNewNameBranches(t *testing.T) {
	m, _, err := pickerModel(t.TempDir(), "file")
	if err != nil {
		t.Fatal(err)
	}
	namecheck.Check(t, namecheck.Config{
		Vocab:    m.Runtime.NameVocab(),
		Baseline: "testdata/namecheck.txt",
		Dirs:     []string{"."},
	})
}
