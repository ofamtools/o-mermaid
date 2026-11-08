package main

import (
	"testing"

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
	lexiconcheck.Check(t, lexiconcheck.Config{
		Operations: &m.Runtime.Operations,
		MainNouns:  nil,
		Words:      "testdata/lexicon-words.tsv",
		Baseline:   "testdata/lexiconcheck.txt",
	})
}
