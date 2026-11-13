package main

import (
	"strings"
	"testing"

	"ofam.dev/ochazuke"
	"ofam.dev/ochazuke/scaffoldcheck"

	"ofam.dev/ochazuke/lexiconcheck"
)

// o-mermaid's registered operations are held to the lexicon's naming rules
// (lexicon plan): one word has one part of speech, nouns are singular, a
// namespace is not a noun or action word, a key has at most three parts.
// testdata/lexicon-words.tsv is a copy of ochazuke's word list.
func TestLexicon(t *testing.T) {
	m, _, err := pickerModel(t.TempDir(), "file")
	if err != nil {
		t.Fatal(err)
	}
	a, err := New(strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	lexiconcheck.Check(t, lexiconcheck.Config{
		Operations: &m.Runtime.Operations,
		MainNouns:  nil,
		Words:      "testdata/lexicon-words.tsv",
		Baseline:   "testdata/lexiconcheck.txt",
	})
	lexiconcheck.Check(t, lexiconcheck.Config{
		Operations: &a.r.Operations,
		MainNouns:  map[string]string{"mermaid": "theme"},
		Words:      "testdata/lexicon-words.tsv",
		Baseline:   "testdata/lexiconcheck.txt",
	})
}

// The theme noun is a facts noun and diagram a command noun: the scaffold
// has nothing to check on either (it holds record nouns), and the runtime
// agrees on their kinds.
func TestNounKinds(t *testing.T) {
	a, err := New(strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	if got := a.r.NounKind("theme"); got != ochazuke.NounKindFacts {
		t.Fatalf("theme kind %q", got)
	}
	if got := a.r.NounKind("diagram"); got != ochazuke.NounKindCommand {
		t.Fatalf("diagram kind %q", got)
	}
	scaffoldcheck.Check(t, scaffoldcheck.Config{Runtime: a.r, Baseline: "testdata/scaffoldcheck.txt"})
}
