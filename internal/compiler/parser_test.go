package compiler

import "testing"

func TestParseKeyValuesRejectsDuplicateKeys(t *testing.T) {
	if _, err := parseKeyValues([]string{`id="one"`, `id="two"`}); err == nil {
		t.Fatal("duplicate declaration key was accepted")
	}
}
